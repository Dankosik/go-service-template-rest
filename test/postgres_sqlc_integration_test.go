//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/testcontainers/testcontainers-go"
)

const migrationPathInitUp = "../env/migrations/000001_init.up.sql"

func TestPingHistoryRepositorySQLCReadWrite(t *testing.T) {
	pool := setupPostgresPoolWithMigrations(t)

	repo := postgres.NewPingHistoryRepository(pool.DB())
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	first, err := repo.Create(ctx, "first")
	if err != nil {
		t.Fatalf("Create(first) error: %v", err)
	}
	second, err := repo.Create(ctx, "second")
	if err != nil {
		t.Fatalf("Create(second) error: %v", err)
	}

	if second.ID <= first.ID {
		t.Fatalf("expected monotonic ids: first=%d second=%d", first.ID, second.ID)
	}

	recent, err := repo.ListRecent(ctx, 2)
	if err != nil {
		t.Fatalf("ListRecent(2) error: %v", err)
	}
	if len(recent) != 2 {
		t.Fatalf("ListRecent(2) len = %d, want 2", len(recent))
	}
	if recent[0].ID != second.ID || recent[1].ID != first.ID {
		t.Fatalf("ListRecent order mismatch: got [%d %d], want [%d %d]", recent[0].ID, recent[1].ID, second.ID, first.ID)
	}
}

func TestPingHistoryRepositorySQLCTxPath(t *testing.T) {
	pool := setupPostgresPoolWithMigrations(t)

	repo := postgres.NewPingHistoryRepository(pool.DB())
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	created, recent, err := repo.CreateAndListRecentInTx(ctx, "tx", 1)
	if err != nil {
		t.Fatalf("CreateAndListRecentInTx error: %v", err)
	}
	if len(recent) != 1 {
		t.Fatalf("CreateAndListRecentInTx recent len = %d, want 1", len(recent))
	}
	if recent[0].ID != created.ID {
		t.Fatalf("CreateAndListRecentInTx mismatch: created=%d recent[0]=%d", created.ID, recent[0].ID)
	}
	if recent[0].Payload != "tx" {
		t.Fatalf("CreateAndListRecentInTx payload = %q, want %q", recent[0].Payload, "tx")
	}
}

func setupPostgresPoolWithMigrations(t *testing.T) *postgres.Pool {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	container, err := runPostgresContainer(ctx)
	if err != nil {
		if isDockerUnavailable(err) {
			if requireDockerForIntegration() {
				t.Fatalf("docker is required for integration tests: %v", err)
			}
			t.Skipf("docker is unavailable: %v", err)
		}
		t.Fatalf("start postgres container: %v", err)
	}

	t.Cleanup(func() {
		if termErr := testcontainers.TerminateContainer(container); termErr != nil {
			t.Errorf("terminate postgres container: %v", termErr)
		}
	})

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("build postgres dsn: %v", err)
	}

	pool, err := postgres.New(ctx, postgres.Options{
		DSN:                dsn,
		ConnectTimeout:     3 * time.Second,
		HealthcheckTimeout: 3 * time.Second,
		MaxOpenConns:       10,
		MaxIdleConns:       5,
		ConnMaxLifetime:    time.Hour,
	})
	if err != nil {
		t.Fatalf("create postgres pool: %v", err)
	}

	t.Cleanup(pool.Close)

	if err := applyMigrationFile(ctx, pool.DB(), migrationPathInitUp); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	return pool
}

type pgExec interface {
	Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error)
}

func applyMigrationFile(ctx context.Context, db pgExec, migrationPath string) error {
	contents, err := os.ReadFile(filepath.Clean(migrationPath))
	if err != nil {
		return fmt.Errorf("read migration file %q: %w", migrationPath, err)
	}

	sql := strings.TrimSpace(string(contents))
	if sql == "" {
		return fmt.Errorf("migration file %q is empty", migrationPath)
	}

	if _, err := db.Exec(ctx, sql); err != nil {
		return fmt.Errorf("execute migration %q: %w", migrationPath, err)
	}

	return nil
}
