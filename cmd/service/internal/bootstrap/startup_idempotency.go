package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/health"
	httpx "github.com/example/go-service-template-rest/internal/infra/http"
	"github.com/example/go-service-template-rest/internal/infra/postgresidempotency"
	"github.com/jackc/pgx/v5/pgxpool"
)

type httpIdempotencyRuntime struct {
	store          *postgresidempotency.Store
	maintain       func(context.Context) error
	terminalErrors <-chan error
	interval       time.Duration
}

func initHTTPIdempotencyRuntime(
	ctx context.Context,
	cfg config.Config,
	pool *pgxpool.Pool,
	operations []httpx.IdempotencyOperation,
) (httpIdempotencyRuntime, error) {
	if len(operations) == 0 {
		return httpIdempotencyRuntime{}, nil
	}
	if err := config.ValidateHTTPIdempotencyActive(cfg.HTTPIdempotency, cfg.Postgres); err != nil {
		return httpIdempotencyRuntime{}, fmt.Errorf("validate HTTP idempotency: %w", err)
	}
	store, err := postgresidempotency.NewStore(pool, idempotencyStoreOptions(cfg.HTTPIdempotency))
	if err != nil {
		return httpIdempotencyRuntime{}, fmt.Errorf("initialize HTTP idempotency: %w", err)
	}
	if err := store.Maintain(ctx); err != nil {
		return httpIdempotencyRuntime{}, fmt.Errorf("initialize HTTP idempotency maintenance: %w", err)
	}
	return httpIdempotencyRuntime{
		store:          store,
		maintain:       store.Maintain,
		terminalErrors: store.TerminalErrors(),
		interval:       cfg.HTTPIdempotency.MaintenanceInterval,
	}, nil
}

func idempotencyStoreOptions(cfg config.HTTPIdempotencyConfig) postgresidempotency.StoreOptions {
	return postgresidempotency.StoreOptions{
		OwnerRecoveryDelay:     cfg.OwnerRecoveryDelay,
		CleanupBatchSize:       cfg.CleanupBatchSize,
		MaxMaintenanceLag:      cfg.MaxMaintenanceLag,
		MaxRelationBytes:       cfg.MaxRelationBytes,
		AdmissionHeadroomBytes: cfg.AdmissionHeadroomBytes,
	}
}

func (r httpIdempotencyRuntime) ReadinessProbes() []health.Probe {
	if r.store == nil {
		return nil
	}
	return []health.Probe{r.store}
}

func (r httpIdempotencyRuntime) Run(ctx context.Context) error {
	if r.maintain == nil {
		return nil
	}
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("stop HTTP idempotency maintenance: %w", ctx.Err())
		case err := <-r.terminalErrors:
			return fmt.Errorf("stop HTTP idempotency maintenance: %w", err)
		case <-ticker.C:
			err := r.maintain(ctx)
			if err == nil || errors.Is(err, postgresidempotency.ErrUnavailable) {
				continue
			}
			return fmt.Errorf("maintain HTTP idempotency: %w", err)
		}
	}
}
