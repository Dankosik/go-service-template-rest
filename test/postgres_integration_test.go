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
)

const postgresTestImage = "postgres:17@sha256:2cd82735a36356842d5eb1ef80db3ae8f1154172f0f653db48fde079b2a0b7f7"

func TestPostgresPool(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()

	dsn := postgresTestDSN(t, ctx)

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
		txCtx, txCancel := context.WithTimeout(t.Context(), 3*time.Second)
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
