package postgreswebhook

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

//nolint:gocognit,cyclop // Recovery keeps each persisted ambiguity and exhaustion transition in order.
func (s *Store) ReconcileExpired(ctx context.Context, batchSize int) (int, error) {
	if !s.valid() || batchSize < 1 || batchSize > 1000 {
		return 0, fmt.Errorf("%w: recovery batch is invalid", ErrConfig)
	}
	batch, err := int32Value(batchSize)
	if err != nil {
		return 0, err
	}
	reconciled := 0
	err = s.transaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
		queries := sqlcgen.New(tx)
		sampledAt, err := advanceClock(ctx, queries)
		if err != nil {
			return err
		}
		quarantined, err := queries.QuarantineInconsistentWebhookDeliveries(ctx, sqlcgen.QuarantineInconsistentWebhookDeliveriesParams{SampledAt: pgtime(sampledAt), BatchSize: batch})
		if err != nil {
			return fmt.Errorf("quarantine inconsistent webhook deliveries: %w", err)
		}
		reconciled = int(quarantined)
		remaining, err := remainingBatch(batch, quarantined)
		if err != nil || remaining == 0 {
			return err
		}
		attempts, err := queries.ListExpiredWebhookAttempts(ctx, sqlcgen.ListExpiredWebhookAttemptsParams{SampledAt: pgtime(sampledAt), BatchSize: remaining})
		if err != nil {
			return fmt.Errorf("list expired webhook attempts: %w", err)
		}
		for _, attempt := range attempts {
			var policy DeliveryPolicy
			if err := json.Unmarshal(attempt.PolicySnapshot, &policy); err != nil || policy.validate() != nil || !s.admits(policy) || attempt.PreviousRetryDelayNs < 0 {
				return fmt.Errorf("%w: retained webhook recovery policy is incompatible", ErrConflict)
			}
			delay := recoveryRetryDelay(policy, time.Duration(attempt.PreviousRetryDelayNs), attempt.AttemptID)
			outcome := OutcomeDefinitelyNotSentRetry
			if attempt.MayHaveSent || attempt.SendAuthorized {
				outcome = OutcomeTransportAmbiguous
			}
			summary := CumulativeSummary(OutcomeClass(attempt.CumulativeSummary), outcome)
			if outcome == OutcomeDefinitelyNotSentRetry && attempt.CumulativeSummary == "none" {
				summary = OutcomeClass("none")
			}
			state, cycle := string(DeliveryScheduled), activeDisposition
			nextDueAt, dueErr := RetryDue(sampledAt, attempt.DeadlineAt.Time, delay, 0)
			var terminalAt pgtype.Timestamptz
			if attempt.AttemptsUsed >= attempt.MaximumAttempts || dueErr != nil {
				nextDueAt = sampledAt
				terminalAt = pgtime(sampledAt)
				if summary == OutcomeUnknown {
					state, cycle = string(DeliverySuspended), string(OutcomeUnknown)
				} else {
					state, summary, cycle = string(DeliveryTerminal), OutcomeAttemptsExhausted, string(OutcomeAttemptsExhausted)
				}
			}
			outcomeText := string(outcome)
			retryDelayNS := delay.Nanoseconds()
			rows, err := queries.FinalizeWebhookAttempt(ctx, sqlcgen.FinalizeWebhookAttemptParams{
				OutcomeClass: &outcomeText, FinalizedAt: pgtime(sampledAt), OwnerScope: attempt.OwnerScope,
				DeliveryID: attempt.DeliveryID, CycleNumber: attempt.CycleNumber, AttemptID: attempt.AttemptID,
				Fence: attempt.Fence, DeliveryState: state, NextDueAt: pgtime(nextDueAt), RetryDelayNs: &retryDelayNS,
				CumulativeSummary: string(summary), TerminalAt: terminalAt, CycleDisposition: cycle,
				CapacityRevision: s.options.CapacityRevision,
			})
			if err != nil {
				return fmt.Errorf("reconcile webhook attempt: %w", err)
			}
			if rows == 1 {
				reconciled++
			}
		}
		remaining, err = remainingBatch(batch, int64(reconciled))
		if err != nil {
			return err
		}
		if remaining > 0 {
			finalized, err := queries.FinalizeExpiredWebhookCycles(ctx, sqlcgen.FinalizeExpiredWebhookCyclesParams{SampledAt: pgtime(sampledAt), BatchSize: remaining})
			if err != nil {
				return fmt.Errorf("finalize expired webhook cycles: %w", err)
			}
			reconciled += int(finalized)
			remaining, err = remainingBatch(remaining, finalized)
			if err != nil {
				return err
			}
		}
		if remaining > 0 {
			released, err := queries.ReleaseExpiredWebhookOrphanCapacity(ctx, sqlcgen.ReleaseExpiredWebhookOrphanCapacityParams{CapacityRevision: s.options.CapacityRevision, SampledAt: pgtime(sampledAt), BatchSize: remaining})
			if err != nil {
				return fmt.Errorf("release expired webhook orphan capacity: %w", err)
			}
			reconciled += int(released)
		}
		return nil
	})
	return reconciled, err
}

func recoveryRetryDelay(policy DeliveryPolicy, previous time.Duration, attemptID string) time.Duration {
	seed := sha256.Sum256([]byte(attemptID))
	return DecorrelatedJitter(previous, policy.BackoffBase, policy.BackoffCap, binary.BigEndian.Uint64(seed[:8]))
}
