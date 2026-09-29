package postgresmigrate

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pressly/goose/v3"
)

type FailureStage string

const (
	FailureSource  FailureStage = "source"
	FailureConfig  FailureStage = "config"
	FailureConnect FailureStage = "connect"
	FailureLock    FailureStage = "lock"
	FailureState   FailureStage = "state"
	FailureExecute FailureStage = "execute"
	FailureCleanup FailureStage = "cleanup"
)

// RunResult describes migrations completed before MigrateUp or MigrateDown
// returned. It remains useful with an error: AppliedCount reports completed
// migrations, and After can be derived from them rather than read back from the
// database. Early failures leave versions zero, and cleanup can fail after work
// completed; an errored result is not authoritative database state.
type RunResult struct {
	Before       int64
	BeforeKnown  bool
	Target       int64
	TargetKnown  bool
	After        int64
	AfterKnown   bool
	AppliedCount int
	Duration     time.Duration
}

// recordVersions records the versions read before any migration runs; After
// starts at before until applied migrations move it.
func (r *RunResult) recordVersions(before, target int64) {
	r.Before, r.After, r.Target = before, before, target
	r.BeforeKnown, r.AfterKnown, r.TargetKnown = true, true, true
}

type RunError struct {
	Stage FailureStage
	Err   error
}

func (e *RunError) Error() string {
	return fmt.Sprintf("postgres migration %s: %v", e.Stage, e.Err)
}

func (e *RunError) Unwrap() error {
	return e.Err
}

func FailureStageOf(err error) FailureStage {
	if runErr, ok := errors.AsType[*RunError](err); ok {
		return runErr.Stage
	}
	return FailureExecute
}

func SQLStateOf(err error) string {
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
		return pgErr.Code
	}
	return ""
}

func stageError(stage FailureStage, err error) error {
	if err == nil {
		return nil
	}
	return &RunError{Stage: stage, Err: err}
}

func migrationVersion(result *goose.MigrationResult) int64 {
	if result == nil || result.Source == nil {
		return 0
	}
	return result.Source.Version
}

func setAfterFromApplied(
	result *RunResult,
	direction migrationDirection,
	sources []*goose.Source,
	applied []*goose.MigrationResult,
) {
	result.AppliedCount = len(applied)
	if len(applied) == 0 {
		return
	}
	switch direction {
	case directionUp:
		result.After = migrationVersion(applied[len(applied)-1])
	case directionDown:
		beforeIndex := slices.IndexFunc(sources, func(source *goose.Source) bool {
			return source != nil && source.Version == result.Before
		})
		remainingIndex := beforeIndex - len(applied)
		if remainingIndex < 0 {
			result.After = 0
			return
		}
		result.After = sources[remainingIndex].Version
	}
}
