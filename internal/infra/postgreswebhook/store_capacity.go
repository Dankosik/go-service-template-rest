package postgreswebhook

import (
	"context"
	"fmt"

	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5"
)

func (s *Store) InitializeOrTransitionCapacity(ctx context.Context) error {
	if !s.valid() {
		return fmt.Errorf("%w: store is required", ErrConfig)
	}
	slots, err := int32Value(s.options.GlobalConcurrency)
	if err != nil {
		return err
	}
	return s.transaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
		queries := sqlcgen.New(tx)
		if err := queries.LockWebhookCapacityTable(ctx); err != nil {
			return fmt.Errorf("lock webhook capacity: %w", err)
		}
		current, err := queries.ReadWebhookCapacity(ctx)
		if err != nil {
			return fmt.Errorf("read webhook capacity: %w", err)
		}
		if current.SlotCount == 0 {
			return queries.InsertWebhookCapacity(ctx, sqlcgen.InsertWebhookCapacityParams{SlotCount: slots, CapacityRevision: s.options.CapacityRevision})
		}
		if current.RevisionCount != 1 || current.CapacityRevision > s.options.CapacityRevision ||
			current.CapacityRevision == s.options.CapacityRevision && int(current.SlotCount) != s.options.GlobalConcurrency {
			return fmt.Errorf("%w: capacity revision/count conflict", ErrConfig)
		}
		if current.CapacityRevision == s.options.CapacityRevision {
			return nil
		}
		if current.LeasedCount != 0 {
			return fmt.Errorf("%w: capacity transition requires zero leases", ErrConfig)
		}
		maxDeclared, err := queries.ReadWebhookMinimumDeclaredConcurrency(ctx)
		if err != nil {
			return fmt.Errorf("read webhook capacity ceiling: %w", err)
		}
		if maxDeclared != 0 && s.options.GlobalConcurrency > int(maxDeclared) {
			return fmt.Errorf("%w: capacity exceeds a retained destination policy", ErrConfig)
		}
		if err := queries.ClearWebhookCapacity(ctx); err != nil {
			return fmt.Errorf("clear webhook capacity: %w", err)
		}
		if err := queries.InsertWebhookCapacity(ctx, sqlcgen.InsertWebhookCapacityParams{SlotCount: slots, CapacityRevision: s.options.CapacityRevision}); err != nil {
			return fmt.Errorf("insert webhook capacity: %w", err)
		}
		return nil
	})
}

func (s *Store) EnsureCapacity(ctx context.Context) error {
	return s.InitializeOrTransitionCapacity(ctx)
}
