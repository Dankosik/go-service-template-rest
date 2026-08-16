package postgreswebhook

import (
	"context"
	"fmt"

	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5"
)

func (s *Store) FinalizeExpiredCycles(ctx context.Context, batchSize int) (int64, error) {
	if !s.valid() || batchSize < 1 || batchSize > 1000 {
		return 0, fmt.Errorf("%w: deadline batch is invalid", ErrConfig)
	}
	batch, err := int32Value(batchSize)
	if err != nil {
		return 0, err
	}
	var finalized int64
	err = s.transaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
		queries := sqlcgen.New(tx)
		now, err := advanceClock(ctx, queries)
		if err != nil {
			return err
		}
		finalized, err = queries.FinalizeExpiredWebhookCycles(ctx, sqlcgen.FinalizeExpiredWebhookCyclesParams{SampledAt: pgtime(now), BatchSize: batch})
		if err != nil {
			return fmt.Errorf("finalize expired webhook cycles: %w", err)
		}
		return nil
	})
	return finalized, err
}

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
		stages := []struct {
			name    string
			cleanup func() (int64, error)
		}{
			{"payloads", func() (int64, error) {
				return queries.EraseRetainedWebhookPayloads(ctx, sqlcgen.EraseRetainedWebhookPayloadsParams{SampledAt: pgtime(now), BatchSize: batch})
			}},
			{"attempts", func() (int64, error) {
				return queries.CleanupRetainedWebhookAttempts(ctx, sqlcgen.CleanupRetainedWebhookAttemptsParams{SampledAt: pgtime(now), BatchSize: batch})
			}},
			{"actions", func() (int64, error) {
				return queries.CleanupRetainedWebhookActions(ctx, sqlcgen.CleanupRetainedWebhookActionsParams{SampledAt: pgtime(now), BatchSize: batch})
			}},
			{"deliveries", func() (int64, error) {
				return queries.CleanupRetainedWebhookDeliveries(ctx, sqlcgen.CleanupRetainedWebhookDeliveriesParams{SampledAt: pgtime(now), BatchSize: batch})
			}},
			{"events", func() (int64, error) { return queries.CleanupRetainedWebhookEvents(ctx, batch) }},
			{"key references", func() (int64, error) {
				return queries.EraseRetainedWebhookKeyReferences(ctx, sqlcgen.EraseRetainedWebhookKeyReferencesParams{SampledAt: pgtime(now), BatchSize: batch})
			}},
			{"destinations", func() (int64, error) {
				return queries.CleanupRetainedWebhookDestinations(ctx, sqlcgen.CleanupRetainedWebhookDestinationsParams{SampledAt: pgtime(now), BatchSize: batch})
			}},
		}
		for _, stage := range stages {
			count, err := stage.cleanup()
			if err != nil {
				return fmt.Errorf("cleanup retained webhook %s: %w", stage.name, err)
			}
			deleted += count
		}
		return nil
	})
	return deleted, err
}
