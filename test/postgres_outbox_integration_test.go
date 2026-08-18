//go:build integration

package integration_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/domainevent"
	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgresoutbox"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
)

func TestPostgresOutboxAtomicAppendAndIdentity(t *testing.T) {
	ctx, pool, appender := newOutboxFixture(t)
	assertRiverMigrationBaseline(t, ctx, pool)
	if _, err := pool.Exec(ctx, "CREATE TABLE outbox_domain_probe (id text PRIMARY KEY)"); err != nil {
		t.Fatalf("create domain probe: %v", err)
	}
	event, err := domainevent.New(
		"event-1",
		"example.changed",
		1,
		time.Unix(1, 0).UTC(),
		struct {
			ID string `json:"id"`
		}{ID: "domain-1"},
	)
	if err != nil {
		t.Fatalf("domainevent.New(): %v", err)
	}
	if err := postgres.InTx(ctx, pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "INSERT INTO outbox_domain_probe (id) VALUES ($1)", "domain-1"); err != nil {
			return err
		}
		return appender.Append(ctx, tx, event)
	}); err != nil {
		t.Fatalf("commit domain mutation and event: %v", err)
	}
	assertOutboxAtomicCounts(t, pool, "domain-1", event.ID, 1, 1)

	rollback := errors.New("rollback")
	rolledBack := event
	rolledBack.ID = "event-rollback"
	if err := postgres.InTx(ctx, pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "INSERT INTO outbox_domain_probe (id) VALUES ($1)", "domain-rollback"); err != nil {
			return err
		}
		if err := appender.Append(ctx, tx, rolledBack); err != nil {
			return err
		}
		return rollback
	}); !errors.Is(err, rollback) {
		t.Fatalf("rollback transaction error = %v", err)
	}
	assertOutboxAtomicCounts(t, pool, "domain-rollback", rolledBack.ID, 0, 0)

	if err := postgres.InTx(ctx, pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		return appender.Append(ctx, tx, event)
	}); err != nil {
		t.Fatalf("repeat identical event: %v", err)
	}
	assertOutboxAtomicCounts(t, pool, "domain-1", event.ID, 1, 1)

	conflict, err := domainevent.New(
		event.ID,
		event.Type,
		event.Version,
		event.OccurredAt,
		map[string]string{"id": "different"},
	)
	if err != nil {
		t.Fatalf("domainevent.New(conflict): %v", err)
	}
	err = postgres.InTx(ctx, pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		return appender.Append(ctx, tx, conflict)
	})
	if !errors.Is(err, postgresoutbox.ErrEventIDConflict) {
		t.Fatalf("conflicting event error = %v", err)
	}
}

func assertRiverMigrationBaseline(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	migrator, err := rivermigrate.New(riverpgxv5.New(pool), nil)
	if err != nil {
		t.Fatalf("rivermigrate.New(): %v", err)
	}
	versions, err := migrator.ExistingVersions(ctx)
	if err != nil {
		t.Fatalf("rivermigrate.ExistingVersions(): %v", err)
	}
	if len(versions) != 7 {
		t.Fatalf("River migration versions = %v, want 1 through 7", versions)
	}
	for index, migration := range versions {
		if migration.Version != index+1 {
			t.Fatalf("River migration version[%d] = %d, want %d", index, migration.Version, index+1)
		}
	}
	result, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, nil)
	if err != nil {
		t.Fatalf("rivermigrate.Migrate(up): %v", err)
	}
	if len(result.Versions) != 0 {
		t.Fatalf("River pending migrations = %v, want none", result.Versions)
	}
}

type outboxCountQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func assertOutboxAtomicCounts(
	t *testing.T,
	db outboxCountQuerier,
	domainID string,
	eventID string,
	wantDomain int,
	wantJobs int,
) {
	t.Helper()
	var domainCount, jobCount int
	if err := db.QueryRow(t.Context(), "SELECT count(*) FROM outbox_domain_probe WHERE id = $1", domainID).Scan(&domainCount); err != nil {
		t.Fatalf("count domain rows: %v", err)
	}
	if err := db.QueryRow(t.Context(), "SELECT count(*) FROM river_job WHERE args->>'id' = $1", eventID).Scan(&jobCount); err != nil {
		t.Fatalf("count River jobs: %v", err)
	}
	if domainCount != wantDomain || jobCount != wantJobs {
		t.Fatalf("counts = domain %d jobs %d, want %d/%d", domainCount, jobCount, wantDomain, wantJobs)
	}
}
