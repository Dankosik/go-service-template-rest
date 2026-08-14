package postgresjobs

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/example/go-service-template-rest/internal/jobs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type TransitionStatus string

const (
	TransitionApplied  TransitionStatus = "applied"
	TransitionRepeated TransitionStatus = "repeated"
	TransitionStale    TransitionStatus = "stale"
	TransitionNotFound TransitionStatus = "not_found"
)

type FinalizeInput struct {
	Attempt     AttemptIdentity
	Transition  jobs.Transition
	FailureCode string
}

type PersistedTransition struct {
	Status      TransitionStatus
	Transition  jobs.Transition
	FailureCode string
	RetryAt     time.Time
	FinalizedAt time.Time
}

//nolint:dupl // Separate sqlc result types keep finalize and rescue queries type-safe.
func (s *Session) Finalize(ctx context.Context, input FinalizeInput) (result PersistedTransition, err error) {
	params, err := transitionParams(input.Attempt, input.Transition, input.FailureCode)
	if err != nil {
		return PersistedTransition{}, err
	}
	err = s.withOperation(ctx, pgx.ReadWrite, func(operationCtx context.Context, queries *sqlcgen.Queries) error {
		row, queryErr := queries.FinalizePostgresJobAttempt(operationCtx, sqlcgen.FinalizePostgresJobAttemptParams(params))
		if queryErr != nil {
			return fmt.Errorf("finalize postgres job attempt: %w", queryErr)
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

type transitionQueryParams struct {
	LogicalJobID            string
	AttemptGeneration       int64
	RecoveryGeneration      int64
	WorkerID                string
	FinalState              string
	DelayMicroseconds       int64
	AttemptsUsed            int32
	Outcome                 string
	EffectStatus            string
	FailureCode             string
	ElapsedUsedMilliseconds int64
}

type transitionResultRow struct {
	Status                  string
	HasResult               bool
	FinalState              string
	Outcome                 string
	EffectStatus            string
	FailureCode             string
	RetryAt                 pgtype.Timestamptz
	AttemptsUsed            int32
	ElapsedUsedMilliseconds int64
	FinalizedAt             pgtype.Timestamptz
}

//nolint:cyclop // Keeping the transition invariants together makes the persistence boundary auditable.
func transitionParams(
	attempt AttemptIdentity,
	transition jobs.Transition,
	failureCode string,
) (transitionQueryParams, error) {
	if err := validateAttemptIdentity(attempt); err != nil {
		return transitionQueryParams{}, err
	}
	if _, err := stateFromDatabase(string(transition.State)); err != nil {
		return transitionQueryParams{}, err
	}
	if transition.State != jobs.StateRetryWait && transition.State != jobs.StateSucceeded &&
		transition.State != jobs.StateCancelled && transition.State != jobs.StateExhausted &&
		transition.State != jobs.StatePermanent && transition.State != jobs.StatePoison &&
		transition.State != jobs.StateOutcomeUnknown {
		return transitionQueryParams{}, fmt.Errorf("%w: final state %q", jobs.ErrInvalidTransition, transition.State)
	}
	if _, err := outcomeFromDatabase(string(transition.Outcome)); err != nil {
		return transitionQueryParams{}, err
	}
	if _, err := effectFromDatabase(string(transition.Effect)); err != nil {
		return transitionQueryParams{}, err
	}
	if transition.AttemptsUsed == 0 || transition.AttemptsUsed > math.MaxInt32 || transition.ElapsedUsed < 0 || transition.Delay < 0 {
		return transitionQueryParams{}, fmt.Errorf("%w: invalid persisted budget", jobs.ErrInvalidTransition)
	}
	if transition.State != jobs.StateRetryWait && transition.Delay != 0 {
		return transitionQueryParams{}, fmt.Errorf("%w: terminal transition has a retry delay", jobs.ErrInvalidTransition)
	}
	if failureCode != "" {
		if err := validateStoreToken("failure_code", failureCode); err != nil {
			return transitionQueryParams{}, err
		}
	}
	delayMicroseconds, err := nonNegativeDurationMicroseconds(transition.Delay)
	if err != nil {
		return transitionQueryParams{}, err
	}
	elapsedMilliseconds := int64(transition.ElapsedUsed / time.Millisecond)
	if transition.ElapsedUsed%time.Millisecond != 0 {
		elapsedMilliseconds++
	}
	return transitionQueryParams{
		LogicalJobID: string(attempt.LogicalJobID), AttemptGeneration: int64(attempt.AttemptGeneration), // #nosec G115 -- validateAttemptIdentity rejects generations above math.MaxInt64.
		RecoveryGeneration: int64(attempt.RecoveryGeneration), WorkerID: attempt.WorkerID, // #nosec G115 -- validateAttemptIdentity rejects generations above math.MaxInt64.
		FinalState: string(transition.State), DelayMicroseconds: delayMicroseconds,
		AttemptsUsed: int32(transition.AttemptsUsed), Outcome: string(transition.Outcome),
		EffectStatus: string(transition.Effect), FailureCode: failureCode,
		ElapsedUsedMilliseconds: elapsedMilliseconds,
	}, nil
}

func persistedTransition(row transitionResultRow) (PersistedTransition, error) {
	status := TransitionStatus(row.Status)
	switch status {
	case TransitionApplied, TransitionRepeated, TransitionStale, TransitionNotFound:
	default:
		return PersistedTransition{}, fmt.Errorf("%w: transition status %q", ErrUnknownVocabulary, row.Status)
	}
	result := PersistedTransition{Status: status}
	if !row.HasResult {
		return result, nil
	}
	state, err := stateFromDatabase(row.FinalState)
	if err != nil {
		return PersistedTransition{}, err
	}
	outcome, err := outcomeFromDatabase(row.Outcome)
	if err != nil {
		return PersistedTransition{}, err
	}
	effect, err := effectFromDatabase(row.EffectStatus)
	if err != nil {
		return PersistedTransition{}, err
	}
	if row.AttemptsUsed < 1 || row.ElapsedUsedMilliseconds < 0 {
		return PersistedTransition{}, fmt.Errorf("%w: invalid persisted transition budget", ErrUnknownVocabulary)
	}
	finalizedAt, err := requiredTime("transition finalized_at", row.FinalizedAt)
	if err != nil {
		return PersistedTransition{}, err
	}
	result.Transition = jobs.Transition{
		State: state, AttemptsUsed: uint32(row.AttemptsUsed),
		ElapsedUsed: time.Duration(row.ElapsedUsedMilliseconds) * time.Millisecond,
		Outcome:     outcome, Effect: effect,
	}
	result.FailureCode = row.FailureCode
	result.FinalizedAt = finalizedAt
	if state == jobs.StateRetryWait {
		retryAt, retryErr := requiredTime("transition retry_at", row.RetryAt)
		if retryErr != nil {
			return PersistedTransition{}, retryErr
		}
		result.RetryAt = retryAt
	}
	return result, nil
}

func nonNegativeDurationMicroseconds(duration time.Duration) (int64, error) {
	if duration < 0 {
		return 0, fmt.Errorf("%w: duration must not be negative", jobs.ErrInvalidTransition)
	}
	microseconds := int64(duration / time.Microsecond)
	if duration%time.Microsecond != 0 {
		microseconds++
	}
	return microseconds, nil
}
