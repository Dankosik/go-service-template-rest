package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/example/go-service-template-rest/internal/background"
	"github.com/example/go-service-template-rest/internal/config"
	httpx "github.com/example/go-service-template-rest/internal/infra/http"
	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgresidempotency"
)

const httpIdempotencyMaintenanceInterval = time.Minute

type httpIdempotencyRuntime struct {
	store    *postgresidempotency.Store
	cleanup  func(context.Context) (int64, error)
	interval time.Duration
	log      *slog.Logger
}

func (r httpIdempotencyRuntime) Supervise(supervisor *background.Supervisor) {
	if r.cleanup != nil {
		supervisor.Go(background.Task{Name: "http_idempotency_maintenance", Run: r.Run})
	}
}

func initDeclaredHTTPIdempotency(
	ctx context.Context,
	cfg config.Config,
	pool *postgres.Pool,
	log *slog.Logger,
) (httpIdempotencyRuntime, error) {
	enabled, err := httpx.HasIdempotentOperations()
	if err != nil {
		return httpIdempotencyRuntime{}, fmt.Errorf("inspect HTTP idempotency declarations: %w", err)
	}
	return initHTTPIdempotencyRuntime(ctx, cfg, pool, log, enabled)
}

func initHTTPIdempotencyRuntime(
	ctx context.Context,
	cfg config.Config,
	pool *postgres.Pool,
	log *slog.Logger,
	enabled bool,
) (httpIdempotencyRuntime, error) {
	if !enabled {
		return httpIdempotencyRuntime{}, nil
	}
	if err := config.ValidateHTTPIdempotencyActive(cfg.HTTPIdempotency, cfg.Postgres); err != nil {
		return httpIdempotencyRuntime{}, fmt.Errorf("validate HTTP idempotency: %w", err)
	}
	store, err := postgresidempotency.NewStore(pool, cfg.HTTPIdempotency.Retention)
	if err != nil {
		return httpIdempotencyRuntime{}, fmt.Errorf("initialize HTTP idempotency: %w", err)
	}
	if _, err := store.Cleanup(ctx); err != nil {
		return httpIdempotencyRuntime{}, fmt.Errorf("initialize HTTP idempotency cleanup: %w", err)
	}
	return httpIdempotencyRuntime{
		store: store, cleanup: store.Cleanup,
		interval: httpIdempotencyMaintenanceInterval, log: log,
	}, nil
}

func (r httpIdempotencyRuntime) Run(ctx context.Context) error {
	if r.cleanup == nil {
		return nil
	}
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("stop HTTP idempotency cleanup: %w", ctx.Err())
		case <-ticker.C:
			if _, err := r.cleanup(ctx); err != nil && r.log != nil {
				r.log.WarnContext(ctx, "http idempotency cleanup failed", "error", err)
			}
		}
	}
}
