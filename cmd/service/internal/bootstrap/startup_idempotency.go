package bootstrap

import (
	"context"
	"fmt"

	"github.com/example/go-service-template-rest/internal/config"
	httpx "github.com/example/go-service-template-rest/internal/infra/http"
	"github.com/example/go-service-template-rest/internal/infra/postgresidempotency"
	"github.com/jackc/pgx/v5/pgxpool"
)

func initDeclaredHTTPIdempotency(
	ctx context.Context,
	cfg config.Config,
	pool *pgxpool.Pool,
) (*postgresidempotency.Store, error) {
	enabled, err := httpx.HasIdempotentOperations()
	if err != nil {
		return nil, fmt.Errorf("inspect HTTP idempotency declarations: %w", err)
	}
	return initHTTPIdempotencyRuntime(ctx, cfg, pool, enabled)
}

func initHTTPIdempotencyRuntime(
	ctx context.Context,
	cfg config.Config,
	pool *pgxpool.Pool,
	enabled bool,
) (*postgresidempotency.Store, error) {
	if !enabled {
		return nil, nil //nolint:nilnil // Disabled optional profile has no store and no error.
	}
	if err := config.ValidateHTTPIdempotencyActive(cfg.HTTPIdempotency, cfg.Postgres); err != nil {
		return nil, fmt.Errorf("validate HTTP idempotency: %w", err)
	}
	store, err := postgresidempotency.NewStore(pool, cfg.HTTPIdempotency.Retention)
	if err != nil {
		return nil, fmt.Errorf("initialize HTTP idempotency: %w", err)
	}
	if _, err := store.Cleanup(ctx); err != nil {
		return nil, fmt.Errorf("initialize HTTP idempotency cleanup: %w", err)
	}
	return store, nil
}
