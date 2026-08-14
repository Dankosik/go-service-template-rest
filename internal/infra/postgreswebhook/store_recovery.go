package postgreswebhook

import (
	"context"
	"fmt"

	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

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
		attempts, err := queries.ListExpiredWebhookAttempts(ctx, sqlcgen.ListExpiredWebhookAttemptsParams{SampledAt: pgtime(sampledAt), BatchSize: batch})
		if err != nil {
			return fmt.Errorf("list expired webhook attempts: %w", err)
		}
		for _, attempt := range attempts {
			outcome, summary, state, cycle := OutcomeDefinitelyNotSentRetry, OutcomeClass("none"), string(DeliveryScheduled), activeDisposition
			var terminalAt pgtype.Timestamptz
			if attempt.MayHaveSent || attempt.SendAuthorized {
				outcome, summary, state, cycle = OutcomeUnknown, OutcomeUnknown, string(DeliverySuspended), string(OutcomeUnknown)
				terminalAt = pgtime(sampledAt)
			}
			outcomeText := string(outcome)
			rows, err := queries.FinalizeWebhookAttempt(ctx, sqlcgen.FinalizeWebhookAttemptParams{
				OutcomeClass: &outcomeText, FinalizedAt: pgtime(sampledAt), OwnerScope: attempt.OwnerScope,
				DeliveryID: attempt.DeliveryID, CycleNumber: attempt.CycleNumber, AttemptID: attempt.AttemptID,
				Fence: attempt.Fence, DeliveryState: state, NextDueAt: pgtime(sampledAt),
				CumulativeSummary: string(summary), TerminalAt: terminalAt, CycleDisposition: cycle,
			})
			if err != nil {
				return fmt.Errorf("reconcile webhook attempt: %w", err)
			}
			if rows == 1 {
				reconciled++
			}
		}
		return nil
	})
	return reconciled, err
}
