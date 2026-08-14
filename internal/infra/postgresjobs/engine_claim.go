package postgresjobs

import (
	"context"
	"errors"
	"fmt"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/jobs"
)

func (e *Engine) claim(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.admission {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("claim context: %w", err)
	}
	limit := e.freeCapacityLocked()
	if limit == 0 {
		return nil
	}

	claims, err := e.store.Claim(ctx, ClaimOptions{
		RegistryKeys: e.registry.Keys(), WorkerID: e.config.WorkerID,
		Limit: limit, LeaseDuration: e.config.LeaseDuration,
	})
	if err != nil && !errors.Is(err, postgres.ErrCommitUnknown) {
		if errors.Is(err, jobs.ErrUnsupportedRevision) {
			e.compatible = false
			e.closeAdmissionLocked()
		}
		return fmt.Errorf("claim jobs: %w", err)
	}

	committed := claims.Attempts
	if errors.Is(err, postgres.ErrCommitUnknown) {
		if len(committed) == 0 {
			return nil
		}
		resolved, resolveErr := e.store.ResolveClaims(ctx, claimIdentities(claims.Attempts))
		if resolveErr != nil {
			return fmt.Errorf("resolve unknown job claim: %w", resolveErr)
		}
		committed = knownClaims(claims.Attempts, resolved)
	}
	for _, claim := range committed {
		if err := e.registerClaimLocked(ctx, claim); err != nil {
			return err
		}
		claimedAt := claim.LeaseExpiresAt.Add(-e.config.LeaseDuration)
		if claimedAt.After(e.lastLease) {
			e.lastLease = claimedAt
		}
	}
	return nil
}

func claimIdentities(claims []ClaimedAttempt) []AttemptIdentity {
	identities := make([]AttemptIdentity, len(claims))
	for index, claim := range claims {
		identities[index] = claim.Attempt
	}
	return identities
}

func knownClaims(claims []ClaimedAttempt, resolutions []ClaimResolution) []ClaimedAttempt {
	known := make(map[AttemptIdentity]bool, len(resolutions))
	for _, resolution := range resolutions {
		known[resolution.Attempt] = resolution.Committed
	}
	committed := make([]ClaimedAttempt, 0, len(claims))
	for _, claim := range claims {
		if known[claim.Attempt] {
			committed = append(committed, claim)
		}
	}
	return committed
}

func (e *Engine) registerClaimLocked(ctx context.Context, claim ClaimedAttempt) error {
	registered, err := e.registry.Lookup(claim.Revision)
	if err != nil {
		e.compatible = false
		e.closeAdmissionLocked()
		return fmt.Errorf("lookup claimed revision: %w", err)
	}

	if !e.admission {
		return nil
	}
	if e.freeCapacityLocked() == 0 {
		return nil
	}
	attemptCtx, cancel := context.WithTimeout(ctx, registered.MaxAttemptDuration())
	// This map insertion is the registration barrier: a drain that takes this
	// mutex can acknowledge quiescence only after the handler is join-visible.
	e.inflight[claim.Attempt] = cancel
	e.telemetry.RecordClaim(ctx, jobs.OutcomeSuccess)
	e.attempts.Go(func() {
		e.runAttempt(attemptCtx, cancel, claim, registered)
	})
	return nil
}
