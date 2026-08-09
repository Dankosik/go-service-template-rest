package bootstrap

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/example/go-service-template-rest/cmd/internal/runtimeopts"
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgresoutbox"
)

func validateClassificationConfig(cfg config.Config) error {
	if !cfg.Outbox.Enabled {
		return fmt.Errorf("%w: outbox must be enabled for legacy classification", postgresoutbox.ErrConfig)
	}
	if !cfg.Postgres.Enabled {
		return fmt.Errorf("%w: postgres must be enabled for legacy classification", postgresoutbox.ErrConfig)
	}
	return nil
}

func runLegacyClassification(
	ctx context.Context,
	startupCtx context.Context,
	cfg config.Config,
	log *slog.Logger,
) error {
	pool, err := postgres.New(startupCtx, runtimeopts.Postgres(cfg.Postgres))
	if err != nil {
		return fmt.Errorf("initialize outbox postgres for legacy classification: %w", err)
	}
	defer pool.Close()
	store, err := postgresoutbox.NewStore(pool, nil)
	if err != nil {
		return fmt.Errorf("initialize outbox store for legacy classification: %w", err)
	}
	return classifyLegacyUntilZero(
		ctx, cfg.Outbox.MaxAttempts, cfg.Outbox.CleanupBatchSize, log,
		store.ClassifyLegacyUncertainty,
	)
}

func classifyLegacyUntilZero(
	ctx context.Context,
	maxAttempts, batchSize int,
	log *slog.Logger,
	classify func(context.Context, int, int) (int, error),
) error {
	for {
		classified, err := classify(ctx, maxAttempts, batchSize)
		if err != nil {
			return fmt.Errorf("classify legacy outbox uncertainty: %w", err)
		}
		if classified == 0 {
			log.InfoContext(ctx, "outbox_legacy_uncertainty_classification_complete")
			return nil
		}
		log.InfoContext(ctx, "outbox_legacy_uncertainty_classified", "count", classified)
	}
}
