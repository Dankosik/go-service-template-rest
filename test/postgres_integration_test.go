//go:build integration

package integration_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

const postgresTestImage = "postgres:17@sha256:2cd82735a36356842d5eb1ef80db3ae8f1154172f0f653db48fde079b2a0b7f7"

func TestPostgresPool(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()

	dsn := integrationPostgresDSN(t)
	spanRecorder := telemetrytest.InstallSpanRecorder(t)
	metricReader := telemetrytest.InstallManualReader(t)

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

func TestMain(m *testing.M) {
	os.Exit(runWithSharedPostgres(m))
}

// runWithSharedPostgres starts one PostgreSQL container for the whole
// integration package. Each test derives its own database from it through
// integrationPostgresDSN, so tests stay isolated without paying container
// startup per test.
func runWithSharedPostgres(m *testing.M) int {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if !requireDockerForIntegration() && !dockerProviderIsHealthy(ctx) {
		// Docker-backed tests skip through integrationPostgresDSN.
		return m.Run()
	}

	container, err := tcpostgres.Run(
		ctx,
		postgresTestImage,
		tcpostgres.WithDatabase("app"),
		tcpostgres.WithUsername("app"),
		tcpostgres.WithPassword("app"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		terminateSharedPostgres(container)
		log.Printf("start shared postgres container: %v", err)
		return 1
	}
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		terminateSharedPostgres(container)
		log.Printf("resolve shared postgres dsn: %v", err)
		return 1
	}

	sharedPostgresAdminDSN = dsn
	code := m.Run()
	terminateSharedPostgres(container)
	return code
}

var (
	sharedPostgresAdminDSN string
	integrationDatabaseSeq atomic.Int64
)

func dockerProviderIsHealthy(ctx context.Context) bool {
	provider, err := testcontainers.NewDockerProvider()
	if err != nil {
		return false
	}
	defer provider.Close()
	return provider.Health(ctx) == nil
}

func terminateSharedPostgres(container *tcpostgres.PostgresContainer) {
	if container == nil {
		return
	}
	if err := testcontainers.TerminateContainer(container); err != nil {
		log.Printf("terminate shared postgres container: %v", err)
	}
}

// integrationPostgresDSN returns a DSN for a database owned by this test,
// created on the shared container and dropped on cleanup. It skips the test
// when Docker is unavailable and REQUIRE_DOCKER does not require it.
func integrationPostgresDSN(t *testing.T) string {
	t.Helper()

	if sharedPostgresAdminDSN == "" {
		t.Skip("docker provider is not healthy; set REQUIRE_DOCKER=1 to require it")
	}

	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()

	databaseName := fmt.Sprintf("test_db_%d", integrationDatabaseSeq.Add(1))
	admin, err := pgxpool.New(ctx, sharedPostgresAdminDSN)
	if err != nil {
		t.Fatalf("connect shared postgres admin database: %v", err)
	}
	defer admin.Close()
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+databaseName); err != nil {
		t.Fatalf("create test database %s: %v", databaseName, err)
	}
	t.Cleanup(func() {
		dropCtx, dropCancel := context.WithTimeout(context.Background(), time.Minute)
		defer dropCancel()
		drop, err := pgxpool.New(dropCtx, sharedPostgresAdminDSN)
		if err != nil {
			t.Errorf("connect to drop test database %s: %v", databaseName, err)
			return
		}
		defer drop.Close()
		if _, err := drop.Exec(dropCtx, "DROP DATABASE "+databaseName+" WITH (FORCE)"); err != nil {
			t.Errorf("drop test database %s: %v", databaseName, err)
		}
	})

	adminURL, err := url.Parse(sharedPostgresAdminDSN)
	if err != nil {
		t.Fatalf("parse shared postgres dsn: %v", err)
	}
	adminURL.Path = "/" + databaseName
	return adminURL.String()
}

func requireDockerForIntegration() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("REQUIRE_DOCKER")))
	return v == "1" || v == "true" || v == "yes"
}
