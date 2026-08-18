//go:build integration

package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestInTxCommitOutcomes(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()

	container, err := tcpostgres.Run(
		ctx,
		"postgres:17@sha256:a426e44bac0b759c95894d68e1a0ac03ecc20b619f498a91aae373bf06d8508d",
		tcpostgres.WithDatabase("app"),
		tcpostgres.WithUsername("app"),
		tcpostgres.WithPassword("app"),
		tcpostgres.BasicWaitStrategies(),
		testcontainers.WithEnv(map[string]string{"POSTGRES_INITDB_ARGS": "--no-sync"}),
	)
	if err != nil {
		t.Fatalf("start PostgreSQL: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), time.Minute)
		defer cleanupCancel()
		if err := container.Terminate(cleanupCtx); err != nil {
			t.Errorf("terminate PostgreSQL: %v", err)
		}
	})

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("resolve PostgreSQL DSN: %v", err)
	}
	pool, err := Open(ctx, Options{
		DSN: dsn,

		MaxOpenConns: 4,
	})
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(pool.Close)

	if _, err := pool.Exec(ctx, `
		CREATE TABLE intx_commit_probe (
			id integer PRIMARY KEY,
			value integer NOT NULL,
			CONSTRAINT intx_commit_probe_value_key UNIQUE (value)
				DEFERRABLE INITIALLY DEFERRED
		)`); err != nil {
		t.Fatalf("create probe table: %v", err)
	}
	if _, err := pool.Exec(ctx, "INSERT INTO intx_commit_probe (id, value) VALUES (1, 1)"); err != nil {
		t.Fatalf("seed probe table: %v", err)
	}

	t.Run("server confirmed failure", func(t *testing.T) {
		err := InTx(ctx, pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, "INSERT INTO intx_commit_probe (id, value) VALUES (2, 1)")
			return err
		})
		if !errors.Is(err, ErrTransaction) {
			t.Fatalf("InTx() error = %v, want ErrTransaction", err)
		}
		if errors.Is(err, ErrCommitUnknown) {
			t.Fatalf("InTx() error = %v, did not want ErrCommitUnknown", err)
		}

		var count int
		if err := pool.QueryRow(ctx, "SELECT count(*) FROM intx_commit_probe WHERE id = 2").Scan(&count); err != nil {
			t.Fatalf("count rejected rows: %v", err)
		}
		if count != 0 {
			t.Fatalf("rows after rejected commit = %d, want 0", count)
		}
	})

	t.Run("result lost after commit", func(t *testing.T) {
		tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
		if err != nil {
			t.Fatalf("BeginTx() error = %v", err)
		}
		err = runInTx(ctx, tx, func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, "INSERT INTO intx_commit_probe (id, value) VALUES (3, 3)")
			return err
		}, func(ctx context.Context, tx pgx.Tx) error {
			if err := commitTx(ctx, tx); err != nil {
				return err
			}
			return errors.New("commit result lost")
		}, nil)
		if !errors.Is(err, ErrCommitUnknown) {
			t.Fatalf("runInTx() error = %v, want ErrCommitUnknown", err)
		}

		verifier, err := pgx.Connect(ctx, dsn)
		if err != nil {
			t.Fatalf("connect independent verifier: %v", err)
		}
		defer verifier.Close(context.WithoutCancel(ctx))

		var count int
		if err := verifier.QueryRow(ctx, "SELECT count(*) FROM intx_commit_probe WHERE id = 3").Scan(&count); err != nil {
			t.Fatalf("count committed rows: %v", err)
		}
		if count != 1 {
			t.Fatalf("rows after unknown commit result = %d, want 1", count)
		}
	})
}
