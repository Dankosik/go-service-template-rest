package bootstrap

import (
	"flag"
	"io"
	"log/slog"

	"github.com/example/go-service-template-rest/cmd/internal/runtimeopts"
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/postgresoutbox"
	"github.com/example/go-service-template-rest/internal/observability/logctx"
)

func parseLoadOptions(args []string) (config.LoadOptions, bool, error) {
	var classifyLegacy bool
	loadOptions, err := config.ParseLoadOptions("outbox-relay", args, func(flags *flag.FlagSet) {
		flags.BoolVar(&classifyLegacy, "classify-legacy-uncertainty", false,
			"classify legacy outbox uncertainty and exit")
	})
	if err != nil {
		return config.LoadOptions{}, false, err
	}
	return loadOptions, classifyLegacy, nil
}

func newLogger(out io.Writer, cfg config.Config) *slog.Logger {
	return logctx.NewProcessLogger(out, cfg.Log.Level).With(
		append(runtimeopts.LoggerFields(cfg), "component", "outbox_relay")...,
	)
}

func relayConfig(cfg config.OutboxConfig) postgresoutbox.RelayConfig {
	return postgresoutbox.RelayConfig{
		PollInterval: cfg.PollInterval, BatchSize: cfg.BatchSize, PublishConcurrency: cfg.PublishConcurrency,
		PublishTimeout: cfg.PublishTimeout, LeaseDuration: cfg.LeaseDuration,
		MaxAttempts: cfg.MaxAttempts, RetryBase: cfg.RetryBase, RetryMax: cfg.RetryMax,
		ObservationInterval: cfg.ObservationInterval, CleanupInterval: cfg.CleanupInterval,
		PublishedRetention: cfg.PublishedRetention, CleanupBatchSize: cfg.CleanupBatchSize,
	}
}
