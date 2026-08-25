//go:build inbound_webhook_test_worker

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync/atomic"

	"github.com/example/go-service-template-rest/cmd/jobs-worker/internal/bootstrap"
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/inboundwebhook"
	"github.com/example/go-service-template-rest/internal/infra/postgresinboundwebhook"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"go.opentelemetry.io/otel/metric"
)

var inboundTestCalls atomic.Int64

func init() {
	buildWorkers = func(_ context.Context, cfg config.Config, log *slog.Logger) (bootstrap.WorkersRuntime, error) {
		workers := river.NewWorkers()
		return bootstrap.WorkersRuntime{
			Workers: workers,
			Bind: func(_ context.Context, pool *pgxpool.Pool, meter metric.MeterProvider) error {
				endpoints, err := postgresinboundwebhook.ParseEndpointManifest(cfg.InboundWebhooks.Endpoints)
				if err != nil {
					return err
				}
				reg := inboundwebhook.NewRegistry()
				if err := reg.Bind("orders", func(raw json.RawMessage) (json.RawMessage, error) {
					return raw, nil
				}, func(_ context.Context, delivery inboundwebhook.VerifiedDelivery, _ json.RawMessage) error {
					inboundTestCalls.Add(1)
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
				}); err != nil {
					return err
				}
				if err := reg.RequireExact(endpoints.IDs()); err != nil {
					return err
				}
				return postgresinboundwebhook.AddWorker(workers, pool, reg, meter, log)
			},
		}, nil
	}
}
