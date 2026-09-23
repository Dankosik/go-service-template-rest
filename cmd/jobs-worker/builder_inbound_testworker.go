//go:build inbound_webhook_test_worker

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/example/go-service-template-rest/cmd/jobs-worker/internal/bootstrap"
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/inboundwebhook"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"go.opentelemetry.io/otel/metric"
)

func init() {
	buildWorkers = func(_ context.Context, cfg config.Config, log *slog.Logger) (bootstrap.WorkerRegistration, error) {
		workers := river.NewWorkers()
		return bootstrap.WorkerRegistration{
			Workers: workers,
			Bind: func(_ context.Context, pool *pgxpool.Pool, meter metric.MeterProvider) error {
				return bindInboundWebhookWorkers(cfg, workers, pool, meter, log, bindTestOrdersHandler)
			},
		}, nil
	}
}

func bindTestOrdersHandler(registry *inboundwebhook.Registry) error {
	return inboundwebhook.Bind(registry, "orders", func(raw json.RawMessage) (json.RawMessage, error) {
		return raw, nil
	}, func(_ context.Context, delivery inboundwebhook.VerifiedDelivery, _ json.RawMessage) error {
		if marker := os.Getenv("INBOUND_WEBHOOK_TEST_MARKER"); marker != "" {
			file, err := os.OpenFile(marker, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
			if err != nil {
				return err
			}
			if _, err := fmt.Fprintln(file, delivery.DeliveryID); err != nil {
				_ = file.Close()
				return err
			}
			return file.Close()
		}
		return nil
	})
}
