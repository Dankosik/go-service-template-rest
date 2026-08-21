package bootstrap

import (
	"fmt"
	"strings"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/natsjs"
)

func parseLoadOptions(args []string) (config.LoadOptions, error) {
	return config.ParseLoadOptions("worker", args, nil)
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
