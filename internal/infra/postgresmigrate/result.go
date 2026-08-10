package postgresmigrate

import (
	"errors"
	"fmt"
	"path/filepath"
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

type MigrationResult struct {
	Version   int64
	Filename  string
	Direction string
	Duration  time.Duration
	Empty     bool
}

type RunResult struct {
	Before     int64
	Target     int64
	After      int64
	Migrations []MigrationResult
	Failed     *MigrationResult
	Duration   time.Duration
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

func migrationResultFromGoose(result *goose.MigrationResult) MigrationResult {
	if result == nil || result.Source == nil {
		return MigrationResult{}
	}
	return MigrationResult{
		Version:   result.Source.Version,
		Filename:  filepath.Base(result.Source.Path),
		Direction: result.Direction,
		Duration:  result.Duration,
		Empty:     result.Empty,
	}
}

func migrationResultsFromGoose(results []*goose.MigrationResult) []MigrationResult {
	mapped := make([]MigrationResult, 0, len(results))
	for _, result := range results {
		mapped = append(mapped, migrationResultFromGoose(result))
	}
	return mapped
}

func setPartialAfter(
	result *RunResult,
	direction migrationDirection,
	source []sourceMigration,
) {
	if len(result.Migrations) == 0 {
		return
	}
	switch direction {
	case directionUp:
		result.After = result.Migrations[len(result.Migrations)-1].Version
	case directionDown:
		beforeIndex := -1
		for i, migration := range source {
			if migration.Version == result.Before {
				beforeIndex = i
				break
			}
		}
		remainingIndex := beforeIndex - len(result.Migrations)
		if remainingIndex < 0 {
			result.After = 0
			return
		}
		result.After = source[remainingIndex].Version
	}
}
