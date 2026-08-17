//go:build integration

package integration_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgres/pgtest"
	"github.com/example/go-service-template-rest/internal/infra/postgresinbox"
	"github.com/example/go-service-template-rest/internal/waittest"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestPostgresInboxClaimAndEffectAtomicity(t *testing.T) {
	t.Parallel()
	ctx, pool := newInboxFixture(t)
	createInboxEffects(t, ctx, pool)

	for _, consumer := range []string{"stream/consumer-a", "stream/consumer-b"} {
		applied, err := applyInbox(ctx, pool, consumer, "message-1", recordInboxEffect(ctx, consumer, "message-1"))
		if err != nil || !applied {
			t.Fatalf("first delivery for %s = applied %t, error %v", consumer, applied, err)
		}
		called := false
		applied, err = applyInbox(ctx, pool, consumer, "message-1", func(pgx.Tx) error {
			called = true
			return nil
		})
		if err != nil || applied || called {
			t.Fatalf("duplicate for %s = applied %t called %t error %v", consumer, applied, called, err)
		}
	}
	assertInboxCounts(t, ctx, pool, 2, 2)

	rollbackErr := errors.New("effect rejected")
	_, err := applyInbox(ctx, pool, "stream/consumer-a", "rolled-back", func(tx pgx.Tx) error {
		if err := recordInboxEffect(ctx, "stream/consumer-a", "rolled-back")(tx); err != nil {
			return err
		}
		return rollbackErr
	})
	if !errors.Is(err, rollbackErr) {
		t.Fatalf("rolled-back delivery error = %v, want effect error", err)
	}
	assertInboxIdentityCount(t, ctx, pool, "stream/consumer-a", "rolled-back", 0)
}

func TestPostgresInboxRedeliveryResolvesTransactionOutcome(t *testing.T) {
	t.Parallel()
	ctx, pool := newInboxFixture(t)
	createInboxEffects(t, ctx, pool)

	t.Run("failed effect rolls back claim", func(t *testing.T) {
		t.Parallel()
		effectErr := errors.New("effect failed")
		_, err := applyInbox(ctx, pool, "stream/retry", "failed-effect", func(pgx.Tx) error { return effectErr })
		if !errors.Is(err, effectErr) {
			t.Fatalf("failed effect error = %v", err)
		}
		applied, err := applyInbox(ctx, pool, "stream/retry", "failed-effect", recordInboxEffect(ctx, "stream/retry", "failed-effect"))
		if err != nil || !applied {
			t.Fatalf("redelivery = applied %t, error %v", applied, err)
		}
	})

	t.Run("lost commit response skips on redelivery", func(t *testing.T) {
		t.Parallel()
		err := commitInboxThenReport(ctx, pool, "stream/retry", "commit-unknown", postgres.ErrCommitUnknown)
		if !errors.Is(err, postgres.ErrCommitUnknown) {
			t.Fatalf("commit observation error = %v, want ErrCommitUnknown", err)
		}
		called := false
		applied, retryErr := applyInbox(ctx, pool, "stream/retry", "commit-unknown", func(pgx.Tx) error {
			called = true
			return nil
		})
		if retryErr != nil || applied || called {
			t.Fatalf("redelivery after unknown = applied %t called %t error %v", applied, called, retryErr)
		}
	})

	t.Run("confirmed rollback applies on redelivery", func(t *testing.T) {
		t.Parallel()
		err := rollbackInboxThenReport(ctx, pool, "stream/retry", "confirmed-rollback", errors.New("handler failed"))
		if err == nil {
			t.Fatal("confirmed rollback reported success")
		}
		applied, retryErr := applyInbox(ctx, pool, "stream/retry", "confirmed-rollback", recordInboxEffect(ctx, "stream/retry", "confirmed-rollback"))
		if retryErr != nil || !applied {
			t.Fatalf("redelivery after rollback = applied %t, error %v", applied, retryErr)
		}
	})

	t.Run("canceled handler rolls back before redelivery", func(t *testing.T) {
		t.Parallel()
		handlerCtx, cancel := context.WithCancel(ctx)
		defer cancel()
		claimed := make(chan error, 1)
		stopped := make(chan error, 1)
		go func() {
			tx, err := pool.PGX().Begin(handlerCtx)
			if err != nil {
				claimed <- err
				return
			}
			didClaim, err := postgresinbox.Claim(handlerCtx, tx, "stream/retry", "canceled")
			if err != nil {
				_ = tx.Rollback(context.WithoutCancel(ctx))
				claimed <- fmt.Errorf("claim before shutdown: %w", err)
				return
			}
			if !didClaim {
				_ = tx.Rollback(context.WithoutCancel(ctx))
				claimed <- errors.New("claim before shutdown: duplicate in fresh transaction")
				return
			}
			claimed <- nil
			<-handlerCtx.Done()
			stopped <- tx.Rollback(context.WithoutCancel(ctx))
		}()
		if err := <-claimed; err != nil {
			t.Fatalf("start cancelable handler: %v", err)
		}
		cancel()
		if err := <-stopped; err != nil {
			t.Fatalf("roll back canceled handler: %v", err)
		}
		applied, retryErr := applyInbox(ctx, pool, "stream/retry", "canceled", recordInboxEffect(ctx, "stream/retry", "canceled"))
		if retryErr != nil || !applied {
			t.Fatalf("redelivery after cancellation = applied %t, error %v", applied, retryErr)
		}
	})

	assertInboxCounts(t, ctx, pool, 4, 4)
}

func TestPostgresInboxConcurrentClaimAndEffect(t *testing.T) {
	t.Parallel()
	ctx, pool := newInboxFixture(t)
	createInboxEffects(t, ctx, pool)

	for _, test := range []struct {
		name    string
		resolve string
	}{
		{name: "winner commits", resolve: "commit"},
		{name: "winner rolls back", resolve: "rollback"},
		{name: "waiter is canceled", resolve: "cancel"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			messageID := "concurrent-" + test.resolve
			consumer := "stream/concurrent"
			winnerConn, err := pool.PGX().Acquire(ctx)
			if err != nil {
				t.Fatalf("acquire winner connection: %v", err)
			}
			defer winnerConn.Release()
			winner, err := winnerConn.Begin(ctx)
			if err != nil {
				t.Fatalf("begin winner: %v", err)
			}
			claimed, err := postgresinbox.Claim(ctx, winner, consumer, messageID)
			if err != nil || !claimed {
				t.Fatalf("winner claim = %t, %v", claimed, err)
			}
			if test.resolve != "rollback" {
				if err := recordInboxEffect(ctx, consumer, messageID)(winner); err != nil {
					t.Fatalf("winner effect: %v", err)
				}
			}
			var winnerPID int
			if err := winner.QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&winnerPID); err != nil {
				t.Fatalf("winner backend pid: %v", err)
			}

			waitCtx, cancelWait := context.WithCancel(ctx)
			result := make(chan inboxClaimResult, 1)
			ready := make(chan inboxWaiterReady, 1)
			go func() {
				applied := false
				readySent := false
				publishReady := func(result inboxWaiterReady) {
					if !readySent {
						ready <- result
						readySent = true
					}
				}
				err := pool.InTx(waitCtx, pgx.TxOptions{}, func(tx pgx.Tx) error {
					if _, err := tx.Exec(waitCtx, "SELECT set_config('application_name', 'inbox-waiter', false)"); err != nil {
						publishReady(inboxWaiterReady{err: err})
						return err
					}
					var pid int
					if err := tx.QueryRow(waitCtx, "SELECT pg_backend_pid()").Scan(&pid); err != nil {
						publishReady(inboxWaiterReady{err: err})
						return err
					}
					publishReady(inboxWaiterReady{pid: pid})

					claimed, err := postgresinbox.Claim(waitCtx, tx, consumer, messageID)
					if err != nil || !claimed {
						return err
					}
					applied = true
					if test.resolve == "rollback" {
						return recordInboxEffect(waitCtx, consumer, messageID)(tx)
					}
					return nil
				})
				publishReady(inboxWaiterReady{err: err})
				result <- inboxClaimResult{claimed: applied, err: err}
			}()
			waiter := <-ready
			if waiter.err != nil {
				t.Fatalf("start waiter: %v", waiter.err)
			}
			waitForInboxBlock(t, ctx, pool, winnerPID, waiter.pid)

			switch test.resolve {
			case "commit":
				if err := winner.Commit(ctx); err != nil {
					t.Fatalf("commit winner: %v", err)
				}
				got := <-result
				if got.err != nil || got.claimed {
					t.Fatalf("waiter after commit = %+v, want skipped", got)
				}
			case "rollback":
				if err := winner.Rollback(ctx); err != nil {
					t.Fatalf("roll back winner: %v", err)
				}
				got := <-result
				if got.err != nil || !got.claimed {
					t.Fatalf("waiter after rollback = %+v, want claimed", got)
				}
			case "cancel":
				cancelWait()
				got := <-result
				if !errors.Is(got.err, context.Canceled) || got.claimed {
					t.Fatalf("canceled waiter = %+v", got)
				}
				var pgErr *pgconn.PgError
				if !errors.As(got.err, &pgErr) || pgErr.Code != pgerrcode.QueryCanceled {
					t.Fatalf("canceled waiter error = %v, want retained SQLSTATE %s", got.err, pgerrcode.QueryCanceled)
				}
				if err := winner.Commit(ctx); err != nil {
					t.Fatalf("commit winner: %v", err)
				}
				assertInboxWaiterReleased(t, ctx, pool, waiter.pid)
			default:
				t.Fatalf("unknown resolution %q", test.resolve)
			}
			cancelWait()
			assertInboxIdentityCount(t, ctx, pool, consumer, messageID, 1)
		})
	}
}

func TestPostgresInboxClaimSurvivesRestart(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	dsn := pgtest.Migrated(t, os.DirFS(".."), "migrations")
	pool := openInboxPool(t, ctx, dsn)
	createInboxEffects(t, ctx, pool)
	applied, err := applyInbox(ctx, pool, "stream/restart", "old-message", recordInboxEffect(ctx, "stream/restart", "old-message"))
	if err != nil || !applied {
		t.Fatalf("initial apply = %t, %v", applied, err)
	}
	pool.Close()

	restarted := openInboxPool(t, ctx, dsn)
	called := false
	applied, err = applyInbox(ctx, restarted, "stream/restart", "old-message", func(pgx.Tx) error {
		called = true
		return nil
	})
	if err != nil || applied || called {
		t.Fatalf("post-restart duplicate = applied %t called %t error %v", applied, called, err)
	}
	assertInboxCounts(t, ctx, restarted, 1, 1)
}

type inboxClaimResult struct {
	claimed bool
	err     error
}

type inboxWaiterReady struct {
	pid int
	err error
}

func newInboxFixture(t *testing.T) (context.Context, *postgres.Pool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	t.Cleanup(cancel)
	dsn := pgtest.Migrated(t, os.DirFS(".."), "migrations")
	return ctx, openInboxPool(t, ctx, dsn)
}

func openInboxPool(t *testing.T, ctx context.Context, dsn string) *postgres.Pool {
	t.Helper()
	pool, err := postgres.New(ctx, postgres.Options{
		DSN:                dsn,
		ConnectTimeout:     3 * time.Second,
		HealthcheckTimeout: 3 * time.Second,
		MaxOpenConns:       16,
		AcquireTimeout:     time.Second,
		ConnMaxLifetime:    time.Hour,
		StatementTimeout:   10 * time.Second,
	})
	if err != nil {
		t.Fatalf("postgres.New(): %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func createInboxEffects(t *testing.T, ctx context.Context, pool *postgres.Pool) {
	t.Helper()
	if _, err := pool.PGX().Exec(ctx, `
		CREATE TABLE inbox_test_effects (
			consumer_identity text NOT NULL,
			message_id text NOT NULL,
			PRIMARY KEY (consumer_identity, message_id)
		)`); err != nil {
		t.Fatalf("create inbox effect table: %v", err)
	}
}

func applyInbox(
	ctx context.Context,
	pool *postgres.Pool,
	consumer, messageID string,
	effect func(pgx.Tx) error,
) (bool, error) {
	applied := false
	err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		claimed, err := postgresinbox.Claim(ctx, tx, consumer, messageID)
		if err != nil || !claimed {
			return err
		}
		applied = true
		return effect(tx)
	})
	return applied, err
}

func recordInboxEffect(ctx context.Context, consumer, messageID string) func(pgx.Tx) error {
	return func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO inbox_test_effects (consumer_identity, message_id) VALUES ($1, $2)
		`, consumer, messageID)
		return err
	}
}

func commitInboxThenReport(
	ctx context.Context,
	pool *postgres.Pool,
	consumer, messageID string,
	reported error,
) error {
	tx, err := pool.PGX().Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin inbox transaction: %w", err)
	}
	claimed, err := postgresinbox.Claim(ctx, tx, consumer, messageID)
	if err != nil {
		_ = tx.Rollback(ctx)
		return fmt.Errorf("claim inbox message: %w", err)
	}
	if !claimed {
		_ = tx.Rollback(ctx)
		return errors.New("claim inbox message: duplicate in fresh transaction")
	}
	if err := recordInboxEffect(ctx, consumer, messageID)(tx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return reported
}

func rollbackInboxThenReport(
	ctx context.Context,
	pool *postgres.Pool,
	consumer, messageID string,
	reported error,
) error {
	tx, err := pool.PGX().Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin inbox transaction: %w", err)
	}
	if _, err := postgresinbox.Claim(ctx, tx, consumer, messageID); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	if err := tx.Rollback(ctx); err != nil {
		return err
	}
	return reported
}

// waitForInboxBlock waits until the duplicate claim is parked behind the winner.
//
// The statement is the outbox suite's outboxBlockedBy verbatim, and stays a copy
// on purpose: OUTBOX=none removes that file while INBOX=postgres keeps this one,
// so neither suite can declare the shared helper the other would need. Only the
// wait itself is shared, through internal/waittest.
func waitForInboxBlock(t *testing.T, ctx context.Context, pool *postgres.Pool, blockerPID, waiterPID int) {
	t.Helper()
	waittest.Until(t, 10*time.Second, func() bool {
		var blocked bool
		if err := pool.PGX().QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM pg_stat_activity AS activity
				WHERE activity.pid = $2 AND $1 = ANY(pg_blocking_pids(activity.pid))
			)`, blockerPID, waiterPID).Scan(&blocked); err != nil {
			t.Fatalf("observe inbox claim lock: %v", err)
		}
		return blocked
	}, "the duplicate inbox claim to wait on the winner transaction")
}

func assertInboxWaiterReleased(t *testing.T, ctx context.Context, pool *postgres.Pool, waiterPID int) {
	t.Helper()
	verifier, err := pgx.Connect(ctx, pool.PGX().Config().ConnString())
	if err != nil {
		t.Fatalf("connect waiter verifier: %v", err)
	}
	t.Cleanup(func() { _ = verifier.Close(context.WithoutCancel(ctx)) })

	var state string
	if err := verifier.QueryRow(ctx, "SELECT state FROM pg_stat_activity WHERE pid = $1", waiterPID).Scan(&state); err != nil {
		t.Fatalf("read waiter backend state: %v", err)
	}
	if state != "idle" {
		t.Fatalf("waiter backend state = %q, want idle", state)
	}
	var waiting bool
	if err := verifier.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_locks WHERE pid = $1 AND NOT granted)", waiterPID).Scan(&waiting); err != nil {
		t.Fatalf("read waiter locks: %v", err)
	}
	if waiting {
		t.Fatal("waiter backend retains a lock wait")
	}
	if err := pool.PGX().QueryRow(ctx, "SELECT 1").Scan(new(int)); err != nil {
		t.Fatalf("run query after waiter cancellation: %v", err)
	}
}

func assertInboxCounts(t *testing.T, ctx context.Context, pool *postgres.Pool, claims, effects int) {
	t.Helper()
	var gotClaims, gotEffects int
	if err := pool.PGX().QueryRow(ctx, "SELECT count(*) FROM postgres_inbox_claims").Scan(&gotClaims); err != nil {
		t.Fatalf("count inbox claims: %v", err)
	}
	if err := pool.PGX().QueryRow(ctx, "SELECT count(*) FROM inbox_test_effects").Scan(&gotEffects); err != nil {
		t.Fatalf("count inbox effects: %v", err)
	}
	if gotClaims != claims || gotEffects != effects {
		t.Fatalf("inbox rows = claims %d effects %d, want %d/%d", gotClaims, gotEffects, claims, effects)
	}
}

func assertInboxIdentityCount(
	t *testing.T,
	ctx context.Context,
	pool *postgres.Pool,
	consumer, messageID string,
	want int,
) {
	t.Helper()
	var claims, effects int
	if err := pool.PGX().QueryRow(ctx, `
		SELECT count(*) FROM postgres_inbox_claims
		WHERE consumer_identity = $1 AND message_id = $2
	`, consumer, messageID).Scan(&claims); err != nil {
		t.Fatalf("count inbox identity claims: %v", err)
	}
	if err := pool.PGX().QueryRow(ctx, `
		SELECT count(*) FROM inbox_test_effects
		WHERE consumer_identity = $1 AND message_id = $2
	`, consumer, messageID).Scan(&effects); err != nil {
		t.Fatalf("count inbox identity effects: %v", err)
	}
	if claims != want || effects != want {
		t.Fatalf("inbox identity rows = claims %d effects %d, want %d/%d", claims, effects, want, want)
	}
}
