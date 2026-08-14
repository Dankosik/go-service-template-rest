package postgresjobs

import (
	"context"
	"errors"
	"time"

	"github.com/example/go-service-template-rest/internal/jobs"
)

func (e *Engine) runAttempt(ctx context.Context, cancel context.CancelFunc, claim ClaimedAttempt, registered jobs.Registered) {
	defer func() {
		cancel()
		e.mu.Lock()
		delete(e.inflight, claim.Attempt)
		e.mu.Unlock()
	}()

	result, outcome, effect := dispatchAttempt(ctx, claim, registered)
	decision, err := registered.Evaluate(jobs.AttemptFacts{
		LogicalJobID: claim.Identity.LogicalJobID, AttemptGeneration: claim.Attempt.AttemptGeneration,
		RecoveryGeneration: claim.Attempt.RecoveryGeneration, AttemptNumber: claim.AttemptNumber,
		Elapsed: time.Since(claim.BudgetStartedAt), Outcome: outcome, Effect: effect, RetryHint: result.RetryHint,
	})
	if err != nil {
		return
	}
	e.finalizeAttempt(context.WithoutCancel(ctx), FinalizeInput{Attempt: claim.Attempt, Transition: decision})
}

func (e *Engine) finalizeAttempt(ctx context.Context, input FinalizeInput) {
	e.cycleMu.Lock()
	defer e.cycleMu.Unlock()
	if err := e.renew(ctx); err != nil {
		_ = e.fail(err)
		return
	}
	if transition, err := e.store.Finalize(ctx, input); err == nil && transition.Status == TransitionApplied {
		e.telemetry.RecordAttempt(ctx, input.Transition.Outcome)
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
	return jobs.HandlerResult{}, jobs.OutcomeUnknown, jobs.EffectUnknown
}
