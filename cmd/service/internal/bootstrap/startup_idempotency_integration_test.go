//go:build integration

package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/health"
	"github.com/example/go-service-template-rest/internal/httpidempotency"
	httpx "github.com/example/go-service-template-rest/internal/infra/http"
	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgres/pgtest"
	"github.com/example/go-service-template-rest/internal/infra/postgresmigrate"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestPostgresHTTPIdempotencyActiveBootstrap(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	container, err := tcpostgres.Run(
		ctx,
		pgtest.DefaultImage,
		tcpostgres.WithDatabase("app"),
		tcpostgres.WithUsername("app"),
		tcpostgres.WithPassword("app"),
		tcpostgres.BasicWaitStrategies(),
		testcontainers.WithCmd("postgres", "-c", "track_commit_timestamp=on"),
		testcontainers.WithEnv(map[string]string{"POSTGRES_INITDB_ARGS": "--no-sync"}),
	)
	if err != nil {
		t.Fatalf("start PostgreSQL: %v", err)
	}
	t.Cleanup(func() { _ = testcontainers.TerminateContainer(container) })
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("resolve PostgreSQL DSN: %v", err)
	}
	if _, err := postgresmigrate.MigrateUp(ctx, postgresmigrate.MigrationOptions{
		DSN:              dsn,
		SourceFS:         os.DirFS("../../../.."),
		SourcePath:       "migrations",
		ConnectTimeout:   5 * time.Second,
		StatementTimeout: time.Minute,
		LockTimeout:      15 * time.Second,
		CleanupTimeout:   15 * time.Second,
	}); err != nil {
		t.Fatalf("migrate PostgreSQL: %v", err)
	}
	pool, err := postgres.New(ctx, postgres.Options{
		DSN:                dsn,
		ConnectTimeout:     5 * time.Second,
		HealthcheckTimeout: 5 * time.Second,
		MaxOpenConns:       4,
		AcquireTimeout:     2 * time.Second,
		ConnMaxLifetime:    time.Hour,
		StatementTimeout:   10 * time.Second,
	})
	if err != nil {
		t.Fatalf("connect PostgreSQL: %v", err)
	}
	t.Cleanup(pool.Close)

	contract := activeBootstrapContract()
	operation := httpx.IdempotencyOperation{
		Contract: contract,
		Authorize: func(context.Context, *http.Request) (httpidempotency.Scope, bool) {
			return httpidempotency.Scope{}, true
		},
		Admit: func(context.Context, httpidempotency.Scope) httpidempotency.Decision {
			return httpidempotency.Decision{Outcome: httpidempotency.OutcomeExecute}
		},
	}
	type bracket struct {
		start, end time.Time
	}
	brackets := make([]bracket, 0, 2)
	for _, delay := range []time.Duration{30 * time.Second, 50 * time.Second} {
		cfg := config.Config{
			Postgres: config.PostgresConfig{Enabled: true},
			HTTPIdempotency: config.HTTPIdempotencyConfig{
				OwnerRecoveryDelay:     delay,
				MaintenanceInterval:    time.Second,
				CleanupBatchSize:       10,
				MaxMaintenanceLag:      time.Minute,
				MaxRelationBytes:       1 << 40,
				AdmissionHeadroomBytes: 1 << 20,
			},
		}
		runtime, err := initHTTPIdempotencyRuntime(ctx, cfg, pool, []httpx.IdempotencyOperation{operation})
		if err != nil {
			t.Fatalf("initHTTPIdempotencyRuntime(%s): %v", delay, err)
		}
		probes := runtime.ReadinessProbes()
		if len(probes) != 1 || probes[0].Name() != "postgres_http_idempotency" {
			t.Fatalf("active probes for %s = %v", delay, probes)
		}
		if err := probes[0].Check(ctx); err != nil {
			t.Fatalf("initial maintained readiness for %s = %v", delay, err)
		}

		attempt := activeBootstrapAttempt(t, delay.String())
		resolver := func(version string) (httpidempotency.Fingerprint, error) {
			if version != attempt.Fingerprint.Version {
				return httpidempotency.Fingerprint{}, errors.New("fingerprint version unavailable")
			}
			return attempt.Fingerprint, nil
		}
		var before, after time.Time
		if err := pool.PGX().QueryRow(ctx, "SELECT clock_timestamp()").Scan(&before); err != nil {
			t.Fatalf("writer time before %s reservation: %v", delay, err)
		}
		_, decision, err := runtime.store.Reserve(ctx, contract, attempt, resolver)
		if err != nil || decision.Outcome != httpidempotency.OutcomeExecute {
			t.Fatalf("Reserve(%s) = (%v, %v), want execute", delay, decision.Outcome, err)
		}
		if err := pool.PGX().QueryRow(ctx, "SELECT clock_timestamp()").Scan(&after); err != nil {
			t.Fatalf("writer time after %s reservation: %v", delay, err)
		}
		if width := after.Sub(before); width <= 0 || width >= 5*time.Second {
			t.Fatalf("writer bracket for %s = %s, want (0,5s)", delay, width)
		}
		var recoverAfter time.Time
		if err := pool.PGX().QueryRow(ctx, `
			SELECT recover_after
			FROM postgres_http_idempotency
			WHERE identity_token = $1`, attempt.Identity[:]).Scan(&recoverAfter); err != nil {
			t.Fatalf("read recover_after for %s: %v", delay, err)
		}
		shifted := bracket{start: before.Add(delay), end: after.Add(delay)}
		if recoverAfter.Before(shifted.start) || recoverAfter.After(shifted.end) {
			t.Fatalf("recover_after %s outside writer bracket [%s,%s]", recoverAfter, shifted.start, shifted.end)
		}
		brackets = append(brackets, shifted)

		_, err = newHTTPHandler(
			cfg,
			slog.New(slog.DiscardHandler),
			telemetry.New(),
			nil,
			httpRuntimeBindings{
				Handlers: httpx.Handlers{
					Health:        health.New(probes...),
					ReadinessGate: func(context.Context) error { return nil },
				},
				IdempotencyOperations: []httpx.IdempotencyOperation{operation},
			},
		)
		if err == nil || !strings.Contains(err.Error(), "has no OpenAPI declaration") {
			t.Fatalf("health-only HTTP construction error = %v, want active registration boundary", err)
		}
	}
	if !brackets[0].end.Before(brackets[1].start) {
		t.Fatalf("30s and 50s shifted writer brackets overlap: %+v", brackets)
	}
}

func activeBootstrapContract() httpidempotency.Contract {
	return httpidempotency.Contract{
		OperationID:         "test.active",
		APIVersion:          "v1",
		KeyMaxBytes:         64,
		FingerprintVersions: []string{"v1"},
		ResultCodecs:        []string{"test/v1"},
		ReplayStatuses:      []int{http.StatusCreated},
		ResultMaxBytes:      1024,
		ReplayTTL:           time.Hour,
		DuplicateRisk:       httpidempotency.DuplicateRiskPolicy{Duration: 2 * time.Hour},
		InProgressWait:      time.Second,
		RetryAfter:          time.Second,
		ExternalEffect:      httpidempotency.ExternalEffectNone,
	}
}

func activeBootstrapAttempt(t *testing.T, key string) httpidempotency.Attempt {
	t.Helper()
	fingerprint, err := httpidempotency.NewFingerprint("v1", []byte(key))
	if err != nil {
		t.Fatalf("NewFingerprint(): %v", err)
	}
	attempt, err := httpidempotency.NewAttempt(httpidempotency.Scope{
		Authority:   "test",
		OperationID: "test.active",
		APIVersion:  "v1",
		Resource:    "resource",
		Environment: "test",
		Region:      "local",
	}, key, fingerprint)
	if err != nil {
		t.Fatalf("NewAttempt(): %v", err)
	}
	return attempt
}
