package bootstrap

import (
	"fmt"
	"strings"

	"github.com/example/go-service-template-rest/cmd/internal/runtimeopts"
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/natsjs"
)

func validateRuntimeConfig(cfg config.Config) error {
	if err := runtimeopts.ValidateGracePeriod(
		cfg.HTTP.GracePeriod,
		"http.shutdown_timeout",
		cfg.HTTP.ShutdownTimeout,
		workerTailBudget,
	); err != nil {
		return err
	}
	if strings.TrimSpace(cfg.Messaging.URLs) == "" {
		return fmt.Errorf("%w: messaging must be enabled for worker", config.ErrValidate)
	}
	return runtimeopts.RequireDiagnosticsAddr(cfg.Observability.Metrics.Addr, "worker")
}

func messagingWorkerConfig(cfg config.MessagingConfig) (natsjs.WorkerConfig, error) {
	result := natsjs.DefaultWorkerConfig(
		strings.TrimSpace(cfg.Worker.Consumer),
		strings.TrimSpace(cfg.Worker.FilterSubject),
		strings.TrimSpace(cfg.Worker.DeadLetterSubject),
		cfg.Worker.MaxConcurrency,
		cfg.MaxPayloadBytes,
	)
	if err := natsjs.ValidateWorkerConfig(result, cfg.MaxPayloadBytes); err != nil {
		return natsjs.WorkerConfig{}, fmt.Errorf("validate messaging worker config: %w", err)
	}
	return result, nil
}
