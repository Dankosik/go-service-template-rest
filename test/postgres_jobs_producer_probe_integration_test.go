//go:build integration

package integration_test

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestPostgresJobsProducerProbe(t *testing.T) {
	t.Parallel()
	ctx, pool, store := newPostgresJobsFixture(t)
	if err := store.CheckProducerPath(ctx); err != nil {
		t.Fatalf("CheckProducerPath() error = %v", err)
	}
	dsn := postgresJobsDSN(pool)

	t.Run("caller deadline closes readiness", func(t *testing.T) {
		t.Parallel()
		cancelled, cancel := context.WithDeadline(ctx, time.Now().Add(-time.Second))
		cancel()
		if err := store.CheckProducerPath(cancelled); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("CheckProducerPath(expired) error = %v, want context.DeadlineExceeded", err)
		}
	})

	t.Run("read-only endpoint closes readiness", func(t *testing.T) {
		t.Parallel()
		readOnlyPool, readOnlyStore := newPostgresJobsStore(
			ctx,
			t,
			postgresJobsDSNParam(t, dsn, "default_transaction_read_only", "on"),
			2,
			100*time.Millisecond,
		)
		defer readOnlyPool.Close()
		if err := readOnlyStore.CheckProducerPath(ctx); err == nil {
			t.Fatal("CheckProducerPath(read-only) error = nil")
		}
	})

	t.Run("producer privilege loss closes readiness", func(t *testing.T) {
		t.Parallel()
		var database string
		if err := pool.PGX().QueryRow(ctx, "SELECT current_database()").Scan(&database); err != nil {
			t.Fatalf("read current database: %v", err)
		}
		role := "jobs_probe_" + database
		roleIdentifier := pgx.Identifier{role}.Sanitize()
		if _, err := pool.PGX().Exec(ctx, fmt.Sprintf("CREATE ROLE %s LOGIN PASSWORD 'jobs-probe'", roleIdentifier)); err != nil {
			t.Fatalf("create producer role: %v", err)
		}
		t.Cleanup(func() {
			cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
			defer cancel()
			if _, err := pool.PGX().Exec(cleanupCtx, "DROP OWNED BY "+roleIdentifier); err != nil {
				t.Errorf("drop producer role grants: %v", err)
			}
			if _, err := pool.PGX().Exec(cleanupCtx, "DROP ROLE "+roleIdentifier); err != nil {
				t.Errorf("drop producer role: %v", err)
			}
		})
		statements := []string{
			"GRANT CONNECT ON DATABASE " + pgx.Identifier{database}.Sanitize() + " TO " + roleIdentifier,
			"GRANT USAGE ON SCHEMA public TO " + roleIdentifier,
			"GRANT SELECT ON postgres_job_attempts, postgres_job_claim_scopes, postgres_jobs TO " + roleIdentifier,
			"GRANT INSERT ON postgres_jobs TO " + roleIdentifier,
		}
		for _, statement := range statements {
			if _, err := pool.PGX().Exec(ctx, statement); err != nil {
				t.Fatalf("grant producer authority with %q: %v", statement, err)
			}
		}

		roleDSN := postgresJobsProbeDSNUser(t, dsn, role, "jobs-probe")
		rolePool, roleStore := newPostgresJobsStore(ctx, t, roleDSN, 2, 100*time.Millisecond)
		defer rolePool.Close()
		if err := roleStore.CheckProducerPath(ctx); err != nil {
			t.Fatalf("CheckProducerPath(granted role) error = %v", err)
		}
		if _, err := pool.PGX().Exec(ctx, "REVOKE INSERT ON postgres_jobs FROM "+roleIdentifier); err != nil {
			t.Fatalf("revoke producer insert: %v", err)
		}
		if err := roleStore.CheckProducerPath(ctx); err == nil {
			t.Fatal("CheckProducerPath(without INSERT) error = nil")
		}
	})

	t.Run("pool saturation is capacity-only", func(t *testing.T) {
		t.Parallel()
		saturatedPool, saturatedStore := newPostgresJobsStore(ctx, t, dsn, 1, 20*time.Millisecond)
		defer saturatedPool.Close()
		conn, err := saturatedPool.Acquire(ctx)
		if err != nil {
			t.Fatalf("hold saturated pool connection: %v", err)
		}
		defer conn.Release()
		if err := saturatedStore.CheckProducerPath(ctx); err != nil {
			t.Fatalf("CheckProducerPath(saturated) error = %v, want capacity-only readiness", err)
		}
	})

	t.Run("schema authority loss closes readiness", func(t *testing.T) {
		t.Parallel()
		if _, err := pool.PGX().Exec(ctx, "ALTER TABLE postgres_jobs DROP CONSTRAINT postgres_jobs_producer_key"); err != nil {
			t.Fatalf("remove producer schema authority: %v", err)
		}
		if err := store.CheckProducerPath(ctx); err == nil {
			t.Fatal("CheckProducerPath(incompatible schema) error = nil")
		}
	})
}

func postgresJobsProbeDSNUser(t *testing.T, dsn, username, password string) string {
	t.Helper()
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("parse PostgreSQL DSN: %v", err)
	}
	parsed.User = url.UserPassword(username, password)
	return parsed.String()
}
