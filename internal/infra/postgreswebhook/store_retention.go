package postgreswebhook

import (
	"context"
	"fmt"

	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5"
)

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
		deleted, err = queries.CleanupRetainedWebhookEvents(ctx, sqlcgen.CleanupRetainedWebhookEventsParams{SampledAt: pgtime(now), BatchSize: batch})
		if err != nil {
			return fmt.Errorf("cleanup retained webhook events: %w", err)
		}
		return nil
	})
	return deleted, err
}
