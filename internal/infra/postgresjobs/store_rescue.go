package postgresjobs

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/example/go-service-template-rest/internal/jobs"
	"github.com/jackc/pgx/v5"
)

type RescueCandidate struct {
	Attempt         AttemptIdentity
	Revision        jobs.Revision
	State           jobs.State
	AttemptNumber   uint32
	BudgetStartedAt time.Time
	StartedAt       time.Time
	LeaseExpiresAt  time.Time
	ObservedAt      time.Time
	Elapsed         time.Duration
}

type RescueInput struct {
	Attempt     AttemptIdentity
	Transition  jobs.Transition
	FailureCode string
}

type RescueLimit struct {
	Revision        jobs.Revision
	MaxRecoveryWave uint32
}

type RescueCandidateOptions struct {
	Limits []RescueLimit
	Limit  int
}

func (s *Session) RescueCandidates(ctx context.Context, options RescueCandidateOptions) (candidates []RescueCandidate, err error) {
	if options.Limit < 1 || options.Limit > math.MaxInt32 {
		return nil, fmt.Errorf("%w: rescue limit must be between 1 and %d", ErrConfig, math.MaxInt32)
	}
	kinds, argsVersions, policyVersions, waves, err := rescueLimitColumns(options.Limits)
	if err != nil {
		return nil, err
	}
	err = s.withOperation(ctx, "rescue_candidates", pgx.ReadOnly, func(operationCtx context.Context, queries *sqlcgen.Queries) error {
		rows, queryErr := queries.ListExpiredPostgresJobAttempts(operationCtx, sqlcgen.ListExpiredPostgresJobAttemptsParams{
			Kinds: kinds, ArgsVersions: argsVersions, PolicyVersions: policyVersions, MaxRecoveryWaves: waves,
			CandidateLimit: int32(options.Limit), // #nosec G115 -- rescue limit is validated in [1, math.MaxInt32].
		})
		if queryErr != nil {
			return fmt.Errorf("list expired postgres job attempts: %w", queryErr)
		}
		var mapErr error
		candidates, mapErr = rescueCandidatesFromRows(rows)
		return mapErr
	})
	return candidates, err
}

func rescueLimitColumns(limits []RescueLimit) ([]string, []string, []string, []int32, error) {
	if len(limits) == 0 {
		return nil, nil, nil, nil, fmt.Errorf("%w: at least one rescue revision limit is required", ErrConfig)
	}
	keys := make([]jobs.Revision, len(limits))
	waves := make([]int32, len(limits))
	for index, limit := range limits {
		if limit.MaxRecoveryWave == 0 || limit.MaxRecoveryWave > math.MaxInt32 {
			return nil, nil, nil, nil, fmt.Errorf("%w: invalid recovery wave", ErrConfig)
		}
		keys[index] = limit.Revision
		waves[index] = int32(limit.MaxRecoveryWave) // #nosec G115 -- recovery wave is validated against math.MaxInt32.
	}
	kinds, argsVersions, policyVersions, err := registryColumns(keys)
	return kinds, argsVersions, policyVersions, waves, err
}

func rescueCandidatesFromRows(rows []sqlcgen.ListExpiredPostgresJobAttemptsRow) ([]RescueCandidate, error) {
	candidates := make([]RescueCandidate, 0, len(rows))
	for _, row := range rows {
		if row.CurrentWorkerID == nil || row.AttemptsUsed < 1 || row.ElapsedMilliseconds < 0 {
			return candidates, fmt.Errorf("%w: invalid rescue candidate", ErrUnknownVocabulary)
		}
		attempt, err := attemptIdentity(row.LogicalJobID, row.AttemptGeneration, row.RecoveryGeneration, *row.CurrentWorkerID)
		if err != nil {
			return candidates, err
		}
		revision := jobs.Revision{Kind: row.Kind, ArgsVersion: row.ArgsVersion, PolicyVersion: row.PolicyVersion}
		if err := revision.Validate(); err != nil {
			return candidates, fmt.Errorf("validate rescue revision: %w", err)
		}
		state, err := stateFromDatabase(row.State)
		if err != nil {
			return candidates, err
		}
		budgetStartedAt, err := requiredTime("rescue budget_started_at", row.BudgetStartedAt)
		if err != nil {
			return candidates, err
		}
		startedAt, err := requiredTime("rescue started_at", row.StartedAt)
		if err != nil {
			return candidates, err
		}
		leaseExpiresAt, err := requiredTime("rescue lease_expires_at", row.LeaseExpiresAt)
		if err != nil {
			return candidates, err
		}
		observedAt, err := requiredTime("rescue observed_at", row.ObservedAt)
		if err != nil {
			return candidates, err
		}
		candidates = append(candidates, RescueCandidate{
			Attempt: attempt, Revision: revision, State: state, AttemptNumber: uint32(row.AttemptsUsed),
			BudgetStartedAt: budgetStartedAt, StartedAt: startedAt, LeaseExpiresAt: leaseExpiresAt,
			ObservedAt: observedAt, Elapsed: time.Duration(row.ElapsedMilliseconds) * time.Millisecond,
		})
	}
	return candidates, nil
}

//nolint:dupl // Separate sqlc result types keep finalize and rescue queries type-safe.
func (s *Session) Rescue(ctx context.Context, input RescueInput) (result PersistedTransition, err error) {
	params, err := transitionParams(input.Attempt, input.Transition, input.FailureCode)
	if err != nil {
		return PersistedTransition{}, err
	}
	err = s.withOperation(ctx, "rescue", pgx.ReadWrite, func(operationCtx context.Context, queries *sqlcgen.Queries) error {
		row, queryErr := queries.RescuePostgresJobAttempt(operationCtx, sqlcgen.RescuePostgresJobAttemptParams(params))
		if queryErr != nil {
			return fmt.Errorf("rescue postgres job attempt: %w", queryErr)
		}
		result, queryErr = persistedTransition(transitionResultRow{
			Status: row.Status, HasResult: row.HasResult, FinalState: row.FinalState,
			Outcome: row.Outcome, EffectStatus: row.EffectStatus, FailureCode: row.FailureCode,
			RetryAt: row.RetryAt, AttemptsUsed: row.AttemptsUsed,
			ElapsedUsedMilliseconds: row.ElapsedUsedMilliseconds, FinalizedAt: row.FinalizedAt,
		})
		return queryErr
	})
	return result, err
}
