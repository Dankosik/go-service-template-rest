//go:build integration

package integration_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

const postgresTestImage = "postgres:17@sha256:2cd82735a36356842d5eb1ef80db3ae8f1154172f0f653db48fde079b2a0b7f7"

func TestPostgresPool(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()

	dsn := postgresTestDSN(t, ctx)
	spanRecorder, metricReader := installPostgresTelemetry(t)

	pool, err := postgres.New(ctx, postgres.Options{
		DSN:                dsn,
		ConnectTimeout:     3 * time.Second,
		HealthcheckTimeout: 3 * time.Second,
		MaxOpenConns:       10,
		ConnMaxLifetime:    time.Hour,
	})
	if err != nil {
		t.Fatalf("create postgres pool: %v", err)
	}
	t.Cleanup(pool.Close)

	t.Run("readiness probe", func(t *testing.T) {
		checkCtx, checkCancel := context.WithTimeout(t.Context(), 3*time.Second)
		defer checkCancel()

		if err := pool.Check(checkCtx); err != nil {
			t.Fatalf("readiness check failed: %v", err)
		}
	})

	// TEMPLATE EXAMPLE: delete this subtest with template_example.sql if unused,
	// or replace both with transaction behavior owned by a real feature.
	t.Run("sqlc queries share one pgx transaction", func(t *testing.T) {
		traceCtx, traceSpan := otel.Tracer("postgres-integration-test").Start(t.Context(), "postgres transaction")
		defer traceSpan.End()

		txCtx, txCancel := context.WithTimeout(traceCtx, 3*time.Second)
		defer txCancel()

		var firstID string
		var secondID string
		err := pool.InTx(txCtx, func(queries *sqlcgen.Queries) error {
			var queryErr error
			firstID, queryErr = queries.TemplateExampleTransactionID(txCtx)
			if queryErr != nil {
				return queryErr
			}
			secondID, queryErr = queries.TemplateExampleTransactionID(txCtx)
			return queryErr
		})
		if err != nil {
			t.Fatalf("InTx() error = %v, want nil", err)
		}
		if firstID == "" || firstID != secondID {
			t.Fatalf("transaction IDs = %q, %q; want one non-empty ID", firstID, secondID)
		}

		sentinel := errors.New("template callback failure")
		err = pool.InTx(txCtx, func(*sqlcgen.Queries) error {
			return sentinel
		})
		if !errors.Is(err, postgres.ErrTransaction) || !errors.Is(err, sentinel) {
			t.Fatalf("InTx() error = %v, want ErrTransaction and callback failure", err)
		}
	})

	t.Run("telemetry is useful and does not expose query details", func(t *testing.T) {
		assertPostgresTracePrivacy(t, spanRecorder)
		assertPostgresPoolMetrics(t, ctx, metricReader, pool.Name())
	})
}

func installPostgresTelemetry(t *testing.T) (*tracetest.SpanRecorder, *sdkmetric.ManualReader) {
	t.Helper()

	previousTracerProvider := otel.GetTracerProvider()
	previousMeterProvider := otel.GetMeterProvider()

	spanRecorder := tracetest.NewSpanRecorder()
	tracerProvider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	metricReader := sdkmetric.NewManualReader()
	meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(metricReader))
	otel.SetTracerProvider(tracerProvider)
	otel.SetMeterProvider(meterProvider)

	t.Cleanup(func() {
		otel.SetTracerProvider(previousTracerProvider)
		otel.SetMeterProvider(previousMeterProvider)
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := errors.Join(
			tracerProvider.Shutdown(shutdownCtx),
			meterProvider.Shutdown(shutdownCtx),
		); err != nil {
			t.Errorf("shutdown postgres telemetry: %v", err)
		}
	})

	return spanRecorder, metricReader
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
	for _, span := range recorder.Ended() {
		if span.Name() != "query SELECT" {
			continue
		}

		attrs := attribute.NewSet(span.Attributes()...)
		for _, key := range sensitiveKeys {
			if _, present := attrs.Value(key); present {
				t.Fatalf("query span contains sensitive attribute %q", key)
			}
		}
		return
	}
	t.Fatal(`postgres spans do not contain the bounded name "query SELECT"`)
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

func postgresTestDSN(t *testing.T, ctx context.Context) string {
	t.Helper()

	if !requireDockerForIntegration() {
		testcontainers.SkipIfProviderIsNotHealthy(t)
	}

	container, err := tcpostgres.Run(
		ctx,
		postgresTestImage,
		tcpostgres.WithDatabase("app"),
		tcpostgres.WithUsername("app"),
		tcpostgres.WithPassword("app"),
		tcpostgres.BasicWaitStrategies(),
	)
	testcontainers.CleanupContainer(t, container)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("build postgres dsn: %v", err)
	}
	return dsn
}

func requireDockerForIntegration() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("REQUIRE_DOCKER")))
	return v == "1" || v == "true" || v == "yes"
}
