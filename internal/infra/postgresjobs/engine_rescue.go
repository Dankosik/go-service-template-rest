package postgresjobs

import (
	"context"
	"fmt"
	"time"

	"github.com/example/go-service-template-rest/internal/jobs"
)

func (e *Engine) rescue(ctx context.Context) error {
	limits := make([]RescueLimit, 0, len(e.registry.Keys()))
	for _, revision := range e.registry.Keys() {
		registered, err := e.registry.Lookup(revision)
		if err != nil {
			return fmt.Errorf("lookup rescue revision limit: %w", err)
		}
		limits = append(limits, RescueLimit{Revision: revision, MaxRecoveryWave: registered.MaxRecoveryWave()})
	}
	listedAt := time.Now()
	candidates, err := e.store.RescueCandidates(ctx, RescueCandidateOptions{Limits: limits, Limit: e.config.MaxConcurrency})
	if err != nil {
		return fmt.Errorf("list expired job attempts: %w", err)
	}
	for _, candidate := range candidates {
		if err := e.renew(ctx); err != nil {
			return err
		}
		registered, lookupErr := e.registry.Lookup(candidate.Revision)
		if lookupErr != nil {
			e.mu.Lock()
			e.compatible = false
			e.closeAdmissionLocked()
			e.mu.Unlock()
			return fmt.Errorf("lookup rescued revision: %w", lookupErr)
		}
		transition, evaluateErr := registered.Evaluate(jobs.AttemptFacts{
			LogicalJobID: candidate.Attempt.LogicalJobID, AttemptGeneration: candidate.Attempt.AttemptGeneration,
			RecoveryGeneration: candidate.Attempt.RecoveryGeneration, AttemptNumber: candidate.AttemptNumber,
			Elapsed: candidate.Elapsed + time.Since(listedAt), Outcome: jobs.OutcomeLost, Effect: jobs.EffectUnknown,
		})
		if evaluateErr != nil {
			return fmt.Errorf("evaluate rescued attempt: %w", evaluateErr)
		}
		result, rescueErr := e.store.Rescue(ctx, RescueInput{Attempt: candidate.Attempt, Transition: transition, FailureCode: "lease_lost"})
		if rescueErr != nil {
			return fmt.Errorf("rescue job attempt: %w", rescueErr)
		}
		if result.Status == TransitionApplied {
			e.telemetry.RecordRescue(ctx, transition.Outcome)
			if transition.State == jobs.StateRetryWait {
				e.telemetry.RecordRetry(ctx, transition.Outcome)
			}
		}
	}
	return nil
}
