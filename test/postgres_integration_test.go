//go:build integration

package integration_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgres/pgtest"
	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

func TestPostgresPool(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()

	dsn := pgtest.DSN(t)
	spanRecorder := telemetrytest.InstallSpanRecorder(t)
	metricReader := telemetrytest.InstallManualReader(t)

	pool, err := postgres.New(ctx, postgres.Options{
		DSN:                dsn,
		ConnectTimeout:     3 * time.Second,
		HealthcheckTimeout: 3 * time.Second,
		MaxOpenConns:       10,
		AcquireTimeout:     time.Second,
		ConnMaxLifetime:    time.Hour,
		StatementTimeout:   time.Second,
	})
	if err != nil {
		t.Fatalf("create postgres pool: %v", err)
	}
	t.Cleanup(pool.Close)
	if pool.PGX() == nil {
		t.Fatal("postgres PGX() = nil, want initialized pool")
	}

	t.Run("readiness probe", func(t *testing.T) {
		traceCtx, traceSpan := otel.Tracer("postgres-integration-test").Start(t.Context(), "postgres readiness")
		defer traceSpan.End()

		checkCtx, checkCancel := context.WithTimeout(traceCtx, 3*time.Second)
		defer checkCancel()

		if err := pool.Check(checkCtx); err != nil {
			t.Fatalf("readiness check failed: %v", err)
		}
	})

	t.Run("telemetry is useful and does not expose database details", func(t *testing.T) {
		assertPostgresTracePrivacy(t, spanRecorder)
		assertPostgresPoolMetrics(t, ctx, metricReader, pool.Name())
	})
}

func assertPostgresTracePrivacy(t *testing.T, recorder *tracetest.SpanRecorder) {
	t.Helper()

	sensitiveKeys := []attribute.Key{
		semconv.DBQueryTextKey,
		semconv.ServerAddressKey,
		semconv.ServerPortKey,
		semconv.UserNameKey,
		semconv.DBNamespaceKey,
		attribute.Key("pgx.query.parameters"),
	}
	sawDatabaseSpan := false
	observedNames := make([]string, 0, len(recorder.Ended()))
	for _, span := range recorder.Ended() {
		name := span.Name()
		observedNames = append(observedNames, name)
		if name != "pool.acquire" {
			continue
		}
		sawDatabaseSpan = true

		// The readiness path stays bounded to a fixed operation label and never
		// carries raw database or connection details.
		attrs := attribute.NewSet(span.Attributes()...)
		for _, key := range sensitiveKeys {
			if _, present := attrs.Value(key); present {
				t.Fatalf("database span %q contains sensitive attribute %q", name, key)
			}
		}
	}
	if !sawDatabaseSpan {
		t.Fatalf("postgres spans contain no bounded database spans; observed spans: %q", observedNames)
	}
}

func assertPostgresPoolMetrics(
	t *testing.T,
	ctx context.Context,
	reader *sdkmetric.ManualReader,
	wantPoolName string,
) {
	t.Helper()

	var resourceMetrics metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &resourceMetrics); err != nil {
		t.Fatalf("collect postgres metrics: %v", err)
	}
	for _, scopeMetrics := range resourceMetrics.ScopeMetrics {
		for _, metric := range scopeMetrics.Metrics {
			if metric.Name != "pgxpool.max_connections" {
				continue
			}
			gauge, ok := metric.Data.(metricdata.Gauge[int64])
			if !ok || len(gauge.DataPoints) == 0 {
				t.Fatalf("%s has no gauge data points", metric.Name)
			}
			value, present := gauge.DataPoints[0].Attributes.Value(semconv.DBClientConnectionPoolNameKey)
			if !present || value.AsString() != wantPoolName {
				t.Fatalf(
					"%s pool name = %q, present = %t; want %q",
					metric.Name,
					value.AsString(),
					present,
					wantPoolName,
				)
			}
			return
		}
	}
	t.Fatal("postgres metrics do not contain pgxpool.max_connections")
}

func TestMain(m *testing.M) {
	os.Exit(pgtest.Main(m, ""))
}
