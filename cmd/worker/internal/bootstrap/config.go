package bootstrap

import (
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/example/go-service-template-rest/cmd/internal/runtimeopts"
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/natsjs"
)

func parseLoadOptions(args []string) (config.LoadOptions, error) {
	return config.ParseLoadOptions("worker", args, nil)
}

func newWorkerLogger(out io.Writer, cfg config.Config) *slog.Logger {
	return runtimeopts.Logger(out, cfg)
}

func messagingWorkerConfig(cfg config.MessagingConfig) (natsjs.WorkerConfig, error) {
	delays := make([]time.Duration, 0)
	for value := range strings.SplitSeq(cfg.Worker.RetryDelays, ",") {
		delay, err := time.ParseDuration(strings.TrimSpace(value))
		if err != nil {
			return natsjs.WorkerConfig{}, fmt.Errorf("%w: invalid messaging worker retry delay", natsjs.ErrRejected)
		}
		delays = append(delays, delay)
	}
	result := natsjs.WorkerConfig{
		Consumer:             strings.TrimSpace(cfg.Worker.Consumer),
		FilterSubject:        strings.TrimSpace(cfg.Worker.FilterSubject),
		DeadLetterSubject:    strings.TrimSpace(cfg.Worker.DeadLetterSubject),
		MaxConcurrency:       cfg.Worker.MaxConcurrency,
		MaxDeliveryBytes:     cfg.Worker.MaxDeliveryBytes,
		HandlerTimeout:       cfg.Worker.HandlerTimeout,
		RetryDelays:          delays,
		DeadLetterRetryDelay: cfg.Worker.DeadLetterRetryDelay,
	}
	if cfg.Worker.DrainTimeout <= 0 {
		return natsjs.WorkerConfig{}, fmt.Errorf("%w: messaging worker drain timeout must be positive", natsjs.ErrRejected)
	}
	if err := natsjs.ValidateWorkerConfig(result, cfg.MaxPayloadBytes); err != nil {
		return natsjs.WorkerConfig{}, fmt.Errorf("validate messaging worker config: %w", err)
	}
	return result, nil
}
