package postgresjobs

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/jobs"
	"github.com/samber/lo"
)

func (e *Engine) claim(ctx, attemptParent context.Context) error {
	e.mu.Lock()
	if !e.admission {
		e.mu.Unlock()
		return nil
	}
	if err := ctx.Err(); err != nil {
		e.mu.Unlock()
		return fmt.Errorf("claim context: %w", err)
	}
	limit := e.freeCapacityLocked()
	e.mu.Unlock()
	if limit == 0 {
		return nil
	}

	claimStartedAt := time.Now()
	claims, err := e.store.Claim(ctx, ClaimOptions{
		RegistryKeys: e.registry.Keys(), WorkerID: e.config.WorkerID,
		Limit: limit, LeaseDuration: e.config.LeaseDuration,
	})
	if err != nil && !errors.Is(err, postgres.ErrCommitUnknown) {
		if errors.Is(err, jobs.ErrUnsupportedRevision) {
			e.mu.Lock()
			e.compatible = false
			e.closeAdmissionLocked()
			e.mu.Unlock()
		}
		return fmt.Errorf("claim jobs: %w", err)
	}

	committed := claims.Attempts
	if errors.Is(err, postgres.ErrCommitUnknown) {
		if errors.Is(err, ErrSessionTerminal) {
			return fmt.Errorf("claim jobs with lost control session: %w", err)
		}
		if len(committed) == 0 {
			return nil
		}
		resolved, resolveErr := e.store.ResolveClaims(ctx, claimIdentities(claims.Attempts))
		if resolveErr != nil {
			return fmt.Errorf("resolve unknown job claim: %w", resolveErr)
		}
		committed = knownClaims(claims.Attempts, resolved)
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, claim := range committed {
		if err := e.registerClaimLocked(attemptParent, claim, claimStartedAt); err != nil {
			return err
		}
	}
	return nil
}

func claimIdentities(claims []ClaimedAttempt) []AttemptIdentity {
	return lo.Map(claims, func(claim ClaimedAttempt, _ int) AttemptIdentity { return claim.Attempt })
}

func knownClaims(claims []ClaimedAttempt, resolutions []ClaimResolution) []ClaimedAttempt {
	known := lo.Associate(resolutions, func(resolution ClaimResolution) (AttemptIdentity, bool) {
		return resolution.Attempt, resolution.Committed
	})
	return lo.Filter(claims, func(claim ClaimedAttempt, _ int) bool { return known[claim.Attempt] })
}

func (e *Engine) registerClaimLocked(ctx context.Context, claim ClaimedAttempt, claimStartedAt time.Time) error {
	registered, err := e.registry.Lookup(claim.Revision)
	if err != nil {
		e.compatible = false
		e.closeAdmissionLocked()
		return fmt.Errorf("lookup claimed revision: %w", err)
	}

	if e.freeCapacityLocked() == 0 {
		return nil
	}
	attemptCtx, cancel := context.WithTimeout(ctx, registered.MaxAttemptDuration())
	// This map insertion is the registration barrier: a drain that takes this
	// mutex can acknowledge quiescence only after the handler is join-visible.
	e.inflight[claim.Attempt] = inflightAttempt{cancel: cancel, renewAt: claimStartedAt.Add(e.config.LeaseDuration / 3)}
	e.telemetry.RecordClaim(ctx, jobs.OutcomeSuccess, claim.QueueDelay)
	attemptStartedAt := time.Now()
	e.attempts.Go(func() {
		e.runAttempt(attemptCtx, cancel, claim, registered, claimStartedAt, attemptStartedAt)
	})
	return nil
}
