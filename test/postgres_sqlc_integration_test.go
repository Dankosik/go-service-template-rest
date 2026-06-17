//go:build integration

package integration_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
)

func TestPingHistorySQLCReadWrite(t *testing.T) {
	pool := setupMigratedPGXPool(t)
	queries := sqlcgen.New(pool)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	first, err := queries.CreatePingHistory(ctx, "first")
	if err != nil {
		t.Fatalf("CreatePingHistory(first) error: %v", err)
	}
	second, err := queries.CreatePingHistory(ctx, "second")
	if err != nil {
		t.Fatalf("CreatePingHistory(second) error: %v", err)
	}

	if second.ID <= first.ID {
		t.Fatalf("expected monotonic ids: first=%d second=%d", first.ID, second.ID)
	}
	if first.Payload != "first" || second.Payload != "second" {
		t.Fatalf("created payloads = [%q %q], want [first second]", first.Payload, second.Payload)
	}
	if !first.CreatedAt.Valid || !second.CreatedAt.Valid || first.CreatedAt.Time.IsZero() || second.CreatedAt.Time.IsZero() {
		t.Fatalf("created timestamps must be non-zero: first=%v second=%v", first.CreatedAt, second.CreatedAt)
	}

	recent, err := queries.ListRecentPingHistory(ctx, 2)
	if err != nil {
		t.Fatalf("ListRecentPingHistory(2) error: %v", err)
	}
	if len(recent) != 2 {
		t.Fatalf("ListRecentPingHistory(2) len = %d, want 2", len(recent))
	}
	if recent[0].ID != second.ID || recent[1].ID != first.ID {
		t.Fatalf("ListRecentPingHistory order mismatch: got [%d %d], want [%d %d]", recent[0].ID, recent[1].ID, second.ID, first.ID)
	}
	if recent[0].Payload != second.Payload || recent[1].Payload != first.Payload {
		t.Fatalf("ListRecentPingHistory payloads = [%q %q], want [%q %q]", recent[0].Payload, recent[1].Payload, second.Payload, first.Payload)
	}
	if !recent[0].CreatedAt.Time.Equal(second.CreatedAt.Time) || !recent[1].CreatedAt.Time.Equal(first.CreatedAt.Time) {
		t.Fatalf("ListRecentPingHistory timestamps = [%v %v], want [%v %v]", recent[0].CreatedAt, recent[1].CreatedAt, second.CreatedAt, first.CreatedAt)
	}
}

func setupMigratedPGXPool(t *testing.T) *pgxpool.Pool {
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

	if _, err := postgres.MigrateUp(ctx, postgres.MigrationOptions{
		DSN:        dsn,
		SourceFS:   os.DirFS(".."),
		SourcePath: "env/migrations",
	}); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("create pgx pool: %v", err)
	}
	t.Cleanup(pool.Close)

	return pool
}
