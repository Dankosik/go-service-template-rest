package postgresjobs

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/jobs"
)

func (e *Engine) runAttempt(
	ctx context.Context,
	cancel context.CancelFunc,
	claim ClaimedAttempt,
	registered jobs.Registered,
	budgetLocalStartedAt time.Time,
	attemptStartedAt time.Time,
) {
	defer func() {
		cancel()
		e.mu.Lock()
		delete(e.inflight, claim.Attempt)
		e.mu.Unlock()
	}()

	result, outcome, effect := dispatchAttempt(ctx, claim, registered)
	duration := time.Since(attemptStartedAt)
	// PostgreSQL supplies elapsed budget through the claim snapshot; the local
	// monotonic clock owns only the time after that snapshot returns.
	elapsed := claim.BudgetElapsed + time.Since(budgetLocalStartedAt)
	decision, err := registered.Evaluate(jobs.AttemptFacts{
		LogicalJobID: claim.Identity.LogicalJobID, AttemptGeneration: claim.Attempt.AttemptGeneration,
		RecoveryGeneration: claim.Attempt.RecoveryGeneration, AttemptNumber: claim.AttemptNumber,
		Elapsed: elapsed, Outcome: outcome, Effect: effect, RetryHint: result.RetryHint,
	})
	if err != nil {
		_ = e.fail(ctx, fmt.Errorf("evaluate job attempt: %w", err))
		return
	}
	e.finalizeAttempt(context.WithoutCancel(ctx), FinalizeInput{Attempt: claim.Attempt, Transition: decision}, duration)
}

func (e *Engine) finalizeAttempt(ctx context.Context, input FinalizeInput, duration time.Duration) {
	if !e.lockCycle(ctx) {
		_ = e.fail(ctx, fmt.Errorf("finalize job attempt: %w", ctx.Err()))
		return
	}
	defer e.unlockCycle()
	if err := e.renew(ctx); err != nil {
		_ = e.fail(ctx, err)
		return
	}
	transition, err := e.store.Finalize(ctx, input)
	commitUnknown := errors.Is(err, postgres.ErrCommitUnknown)
	if commitUnknown && !errors.Is(err, ErrSessionTerminal) {
		transition, err = e.store.Finalize(ctx, input)
	}
	if err != nil {
		_ = e.fail(ctx, fmt.Errorf("finalize job attempt: %w", err))
		return
	}
	recorded := transition.Status == TransitionApplied || commitUnknown && transition.Status == TransitionRepeated
	if recorded {
		persisted := input.Transition
		if transition.Status == TransitionRepeated {
			persisted = transition.Transition
		}
		e.telemetry.RecordAttempt(ctx, persisted.Outcome, duration)
		if persisted.State == jobs.StateRetryWait {
			e.telemetry.RecordRetry(ctx, persisted.Outcome)
		}
	}
}

func dispatchAttempt(ctx context.Context, claim ClaimedAttempt, registered jobs.Registered) (result jobs.HandlerResult, outcome jobs.OutcomeClass, effect jobs.EffectStatus) {
	outcome, effect = jobs.OutcomeUnknown, jobs.EffectUnknown
	defer func() {
		if recover() != nil {
			outcome, effect = jobs.OutcomePanic, jobs.EffectUnknown
		}
	}()
	result, err := registered.Dispatch(ctx, jobs.DispatchInput{
		Payload: claim.Payload, Identity: claim.Identity, AttemptGeneration: claim.Attempt.AttemptGeneration,
		RecoveryGeneration: claim.Attempt.RecoveryGeneration,
	})
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return jobs.HandlerResult{}, jobs.OutcomeTimeout, jobs.EffectUnknown
	}
	if errors.Is(ctx.Err(), context.Canceled) {
		return jobs.HandlerResult{}, jobs.OutcomeCancelled, jobs.EffectUnknown
	}
	if err == nil {
		return result, result.Outcome, result.Effect
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return jobs.HandlerResult{}, jobs.OutcomeTimeout, jobs.EffectUnknown
	}
	if errors.Is(err, context.Canceled) {
		return jobs.HandlerResult{}, jobs.OutcomeCancelled, jobs.EffectUnknown
	}
	if errors.Is(err, jobs.ErrPoisonPayload) {
		return jobs.HandlerResult{}, jobs.OutcomePoison, jobs.EffectNone
	}
	return jobs.HandlerResult{}, jobs.OutcomeUnknown, jobs.EffectUnknown
}
