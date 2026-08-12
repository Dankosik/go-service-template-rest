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

//nolint:gocognit // A generated row is validated completely before it becomes domain data.
func (s *Session) RescueCandidates(ctx context.Context, limit int) (candidates []RescueCandidate, err error) {
	if limit < 1 || limit > math.MaxInt32 {
		return nil, fmt.Errorf("%w: rescue limit must be between 1 and %d", ErrConfig, math.MaxInt32)
	}
	err = s.withOperation(ctx, pgx.ReadOnly, func(operationCtx context.Context, queries *sqlcgen.Queries) error {
		rows, queryErr := queries.ListExpiredPostgresJobAttempts(operationCtx, int32(limit))
		if queryErr != nil {
			return fmt.Errorf("list expired postgres job attempts: %w", queryErr)
		}
		candidates = make([]RescueCandidate, 0, len(rows))
		for _, row := range rows {
			if row.CurrentWorkerID == nil || row.AttemptsUsed < 1 || row.ElapsedMilliseconds < 0 {
				return fmt.Errorf("%w: invalid rescue candidate", ErrUnknownVocabulary)
			}
			attempt, mapErr := attemptIdentity(row.LogicalJobID, row.AttemptGeneration, row.RecoveryGeneration, *row.CurrentWorkerID)
			if mapErr != nil {
				return mapErr
			}
			revision := jobs.Revision{Kind: row.Kind, ArgsVersion: row.ArgsVersion, PolicyVersion: row.PolicyVersion}
			if mapErr := revision.Validate(); mapErr != nil {
				return fmt.Errorf("validate rescue revision: %w", mapErr)
			}
			state, mapErr := stateFromDatabase(row.State)
			if mapErr != nil {
				return mapErr
			}
			budgetStartedAt, mapErr := requiredTime("rescue budget_started_at", row.BudgetStartedAt)
			if mapErr != nil {
				return mapErr
			}
			startedAt, mapErr := requiredTime("rescue started_at", row.StartedAt)
			if mapErr != nil {
				return mapErr
			}
			leaseExpiresAt, mapErr := requiredTime("rescue lease_expires_at", row.LeaseExpiresAt)
			if mapErr != nil {
				return mapErr
			}
			observedAt, mapErr := requiredTime("rescue observed_at", row.ObservedAt)
			if mapErr != nil {
				return mapErr
			}
			candidates = append(candidates, RescueCandidate{
				Attempt: attempt, Revision: revision, State: state, AttemptNumber: uint32(row.AttemptsUsed),
				BudgetStartedAt: budgetStartedAt, StartedAt: startedAt, LeaseExpiresAt: leaseExpiresAt,
				ObservedAt: observedAt, Elapsed: time.Duration(row.ElapsedMilliseconds) * time.Millisecond,
			})
		}
		return nil
	})
	return candidates, err
}

//nolint:dupl // Separate sqlc result types keep finalize and rescue queries type-safe.
func (s *Session) Rescue(ctx context.Context, input RescueInput) (result PersistedTransition, err error) {
	params, err := transitionParams(input.Attempt, input.Transition, input.FailureCode)
	if err != nil {
		return PersistedTransition{}, err
	}
	err = s.withOperation(ctx, pgx.ReadWrite, func(operationCtx context.Context, queries *sqlcgen.Queries) error {
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
