package postgreswebhook

import (
	"context"
	"fmt"

	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5"
)

//nolint:gocognit,cyclop // One global batch must preserve payload, attempt, cycle, summary, event, action, and destination cleanup order.
func (s *Store) CleanupRetention(ctx context.Context, batchSize int) (int64, error) {
	if !s.valid() || batchSize < 1 || batchSize > 1000 {
		return 0, fmt.Errorf("%w: retention batch is invalid", ErrConfig)
	}
	batch, err := int32Value(batchSize)
	if err != nil {
		return 0, err
	}
	var deleted int64
	err = s.transaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
		queries := sqlcgen.New(tx)
		now, err := advanceClock(ctx, queries)
		if err != nil {
			return err
		}
		deleted, err = queries.BackfillWebhookDeliveryRetention(ctx, batch)
		if err != nil {
			return fmt.Errorf("backfill webhook delivery retention: %w", err)
		}
		remaining, err := remainingBatch(batch, deleted)
		if err != nil {
			return err
		}
		if remaining > 0 {
			actions, err := queries.BackfillWebhookActionRetention(ctx, remaining)
			if err != nil {
				return fmt.Errorf("backfill webhook action retention: %w", err)
			}
			deleted += actions
			remaining, err = remainingBatch(remaining, actions)
			if err != nil {
				return err
			}
		}
		if remaining == 0 {
			return nil
		}
		payloads, err := queries.CleanupRetainedWebhookPayloads(ctx, sqlcgen.CleanupRetainedWebhookPayloadsParams{SampledAt: pgtime(now), BatchSize: remaining})
		if err != nil {
			return fmt.Errorf("cleanup retained webhook payloads: %w", err)
		}
		deleted += payloads
		remaining, err = remainingBatch(remaining, payloads)
		if err != nil {
			return err
		}
		if remaining > 0 {
			attempts, err := queries.CleanupRetainedWebhookAttempts(ctx, sqlcgen.CleanupRetainedWebhookAttemptsParams{SampledAt: pgtime(now), BatchSize: remaining})
			if err != nil {
				return fmt.Errorf("cleanup retained webhook attempts: %w", err)
			}
			deleted += attempts
			remaining, err = remainingBatch(remaining, attempts)
			if err != nil {
				return err
			}
		}
		if remaining > 0 {
			cycles, err := queries.CleanupRetainedWebhookCycles(ctx, sqlcgen.CleanupRetainedWebhookCyclesParams{SampledAt: pgtime(now), BatchSize: remaining})
			if err != nil {
				return fmt.Errorf("cleanup retained webhook cycles: %w", err)
			}
			deleted += cycles
			remaining, err = remainingBatch(remaining, cycles)
			if err != nil {
				return err
			}
		}
		if remaining > 0 {
			summaries, err := queries.CleanupRetainedWebhookSummaries(ctx, sqlcgen.CleanupRetainedWebhookSummariesParams{SampledAt: pgtime(now), BatchSize: remaining})
			if err != nil {
				return fmt.Errorf("cleanup retained webhook summaries: %w", err)
			}
			deleted += summaries
			remaining, err = remainingBatch(remaining, summaries)
			if err != nil {
				return err
			}
		}
		if remaining > 0 {
			events, err := queries.CleanupRetainedWebhookEvents(ctx, sqlcgen.CleanupRetainedWebhookEventsParams{SampledAt: pgtime(now), BatchSize: remaining})
			if err != nil {
				return fmt.Errorf("cleanup retained webhook events: %w", err)
			}
			deleted += events
			remaining, err = remainingBatch(remaining, events)
			if err != nil {
				return err
			}
		}
		if remaining > 0 {
			actions, err := queries.CleanupRetainedWebhookActions(ctx, sqlcgen.CleanupRetainedWebhookActionsParams{SampledAt: pgtime(now), BatchSize: remaining})
			if err != nil {
				return fmt.Errorf("cleanup retained webhook actions: %w", err)
			}
			deleted += actions
			remaining, err = remainingBatch(remaining, actions)
			if err != nil {
				return err
			}
		}
		if remaining > 0 {
			destinations, err := queries.CleanupRetiredWebhookDestinations(ctx, remaining)
			if err != nil {
				return fmt.Errorf("cleanup retired webhook destinations: %w", err)
			}
			deleted += destinations
		}
		return nil
	})
	return deleted, err
}
