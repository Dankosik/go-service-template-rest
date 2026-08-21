package bootstrap

import (
	"log/slog"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/postgresoutbox"
	"github.com/riverqueue/river"
)

func TestRiverClientConfigUsesFixedBoundedCapacity(t *testing.T) {
	t.Parallel()

	cfg := config.Config{}
	cfg.HTTP.ShutdownTimeout = 3 * time.Second
	got := riverClientConfig(cfg, river.NewWorkers(), slog.Default())
	if got.Queues[postgresoutbox.Queue].MaxWorkers != defaultOutboxWorkers {
		t.Fatalf("MaxWorkers = %d, want %d", got.Queues[postgresoutbox.Queue].MaxWorkers, defaultOutboxWorkers)
	}
	if got.CancelledJobRetentionPeriod != -1 || got.DiscardedJobRetentionPeriod != -1 {
		t.Fatal("unfinished outbox jobs can be deleted")
	}
	if len(got.Plugins) != 1 {
		t.Fatalf("Plugins = %d, want 1", len(got.Plugins))
	}
	if !got.PollOnly {
		t.Fatal("River LISTEN must stay disabled on the statement-bounded pool")
	}
}

func TestValidateRuntimeConfig(t *testing.T) {
	t.Parallel()

	valid := config.Config{}
	valid.Postgres.Enabled = true
	valid.Messaging.URLs = "tls://nats.example:4222"
	valid.Observability.Metrics.Addr = "127.0.0.1:9090"
	if err := validateRuntimeConfig(valid); err != nil {
		t.Fatalf("validateRuntimeConfig() error = %v", err)
	}
	for name, mutate := range map[string]func(*config.Config){
		"postgres":    func(cfg *config.Config) { cfg.Postgres.Enabled = false },
		"messaging":   func(cfg *config.Config) { cfg.Messaging.URLs = "" },
		"diagnostics": func(cfg *config.Config) { cfg.Observability.Metrics.Addr = "" },
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			invalid := valid
			mutate(&invalid)
			if err := validateRuntimeConfig(invalid); err == nil {
				t.Fatal("validateRuntimeConfig() error = nil")
			}
		})
	}
}
