package postgreswebhook

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5"
)

type StoreOptions struct {
	OperationTimeout      time.Duration
	CapacityRevision      int64
	GlobalConcurrency     int
	ManifestRevision      int64
	AttemptTimeout        time.Duration
	ResponseHeaderTimeout time.Duration
	ResponseHeaderBytes   int
	ResponseBodyBytes     int
	DrainTimeout          time.Duration
}

type Store struct {
	pool    *postgres.Pool
	options StoreOptions
}

func (s *Store) CheckSchema(ctx context.Context) error {
	if !s.valid() {
		return fmt.Errorf("%w: store is required", ErrConfig)
	}
	queries := sqlcgen.New(s.pool.PGX())
	relations, err := queries.ListPostgresWebhookRelations(ctx)
	if err != nil {
		return fmt.Errorf("inspect webhook schema: %w", err)
	}
	want := []string{"webhook_attempts", "webhook_capacity_slots", "webhook_clock", "webhook_cycles", "webhook_deliveries", "webhook_destinations", "webhook_events", "webhook_fanouts", "webhook_operator_actions", "webhook_tombstones"}
	if !slices.Equal(relations, want) {
		return fmt.Errorf("%w: webhook schema relations = %v", ErrConfig, relations)
	}
	writable, err := queries.CheckPostgresWebhookWriter(ctx)
	if err != nil {
		return fmt.Errorf("inspect webhook writer privilege: %w", err)
	}
	if writable == nil || !*writable {
		return fmt.Errorf("%w: webhook writer privileges are incomplete", ErrConfig)
	}
	return nil
}

func NewStore(pool *postgres.Pool, options StoreOptions) (*Store, error) {
	if pool == nil || pool.PGX() == nil {
		return nil, fmt.Errorf("%w: postgres pool is required", ErrConfig)
	}
	if options.OperationTimeout <= 0 || options.OperationTimeout > MaxStoreOperationTime ||
		options.CapacityRevision <= 0 || options.GlobalConcurrency < 1 ||
		options.GlobalConcurrency > MaxConcurrency || options.ManifestRevision <= 0 ||
		options.ResponseHeaderTimeout <= 0 || options.ResponseHeaderTimeout > options.AttemptTimeout ||
		options.AttemptTimeout <= 0 || options.AttemptTimeout > options.DrainTimeout || options.DrainTimeout <= 0 ||
		options.ResponseHeaderBytes < 1 || options.ResponseHeaderBytes > MaxResponseBytes ||
		options.ResponseBodyBytes < 1 || options.ResponseBodyBytes > MaxResponseBytes {
		return nil, fmt.Errorf("%w: store bounds are invalid", ErrConfig)
	}
	return &Store{pool: pool, options: options}, nil
}

func (s *Store) accepts(policy DeliveryPolicy) bool {
	return policy.AttemptTimeout <= s.options.AttemptTimeout &&
		policy.ResponseHeaderTimeout <= s.options.ResponseHeaderTimeout &&
		policy.ResponseHeaderBytes <= s.options.ResponseHeaderBytes &&
		policy.ResponseBodyBytes <= s.options.ResponseBodyBytes &&
		policy.DrainTimeout <= s.options.DrainTimeout
}

func (s *Store) valid() bool {
	return s != nil && s.pool != nil && s.pool.PGX() != nil && s.options.OperationTimeout > 0
}

func (s *Store) transaction(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
	if !s.valid() {
		return fmt.Errorf("%w: store is required", ErrConfig)
	}
	opCtx, cancel := context.WithTimeout(ctx, s.options.OperationTimeout)
	defer cancel()
	tx, err := s.pool.PGX().BeginTx(opCtx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin webhook transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(opCtx)) }()
	if err := fn(opCtx, tx); err != nil {
		return err
	}
	if err := postgres.CommitTx(opCtx, tx); err != nil {
		return fmt.Errorf("commit webhook transaction: %w", err)
	}
	return nil
}
