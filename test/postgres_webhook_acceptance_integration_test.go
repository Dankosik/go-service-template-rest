//go:build integration

package integration_test

import (
	"bytes"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgreswebhook"
	"github.com/jackc/pgx/v5"
)

func TestPostgresWebhookAcceptance(t *testing.T) {
	ctx, pool, store, _ := newPostgresWebhookFixture(t)
	if _, err := pool.PGX().Exec(ctx, `CREATE TABLE webhook_business_fixture (id text PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}
	prepared := webhookPrepared(t, "atomic")
	err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `INSERT INTO webhook_business_fixture VALUES ('atomic')`); err != nil {
			return err
		}
		receipt, err := store.Accept(ctx, tx, prepared)
		if err != nil || receipt.Disposition != postgreswebhook.AcceptanceAccepted || len(receipt.DeliveryIDs) != 1 {
			t.Fatalf("Accept() = %+v, %v", receipt, err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("commit acceptance: %v", err)
	}
	readback, err := store.ResolveAcceptance(ctx, prepared)
	if err != nil || readback.Disposition != postgreswebhook.AcceptanceAccepted || !bytes.Equal([]byte(readback.DeliveryIDs[0]), []byte(prepared.Destinations[0].DeliveryID)) {
		t.Fatalf("ResolveAcceptance() = %+v, %v", readback, err)
	}
	if err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		receipt, err := store.Accept(ctx, tx, prepared)
		if err != nil || receipt.Disposition != postgreswebhook.AcceptanceAccepted {
			return fmt.Errorf("replay acceptance: %+v: %w", receipt, err)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	rolledBack := webhookPrepared(t, "rollback")
	rollback := errors.New("rollback fixture")
	err = pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		if _, err := store.Accept(ctx, tx, rolledBack); err != nil {
			return err
		}
		return rollback
	})
	if !errors.Is(err, rollback) {
		t.Fatalf("rollback error = %v", err)
	}
	var count int
	if err := pool.PGX().QueryRow(ctx, `SELECT count(*) FROM webhook_events WHERE acceptance_id = $1`, rolledBack.Acceptance.AcceptanceID).Scan(&count); err != nil || count != 0 {
		t.Fatalf("rolled-back event count = %d, %v", count, err)
	}
}

func TestPostgresWebhookAcceptanceReadback(t *testing.T) {
	ctx, _, store, _ := newPostgresWebhookFixture(t)
	prepared := webhookPrepared(t, "absent")
	receipt, err := store.ResolveAcceptance(ctx, prepared)
	if err != nil || receipt.Disposition != postgreswebhook.AcceptanceRejected {
		t.Fatalf("ResolveAcceptance(absent) = %+v, %v", receipt, err)
	}
}

func TestPostgresWebhookDestinationReuseAfterKeyRotation(t *testing.T) {
	ctx, pool, store, _ := newPostgresWebhookFixture(t)
	first := webhookPrepared(t, "destination-reuse-first")
	if err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error { _, err := store.Accept(ctx, tx, first); return err }); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.PGX().Exec(ctx, `UPDATE webhook_destinations SET active_key_reference = 'key-b', destination_retained_until = clock_timestamp() + interval '1 minute', key_references_retained_until = clock_timestamp() + interval '1 minute' WHERE owner_scope = 'owner-a' AND destination_id = 'dest-a' AND generation = 1`); err != nil {
		t.Fatal(err)
	}
	second := webhookPrepared(t, "destination-reuse-second")
	var receipt postgreswebhook.AcceptanceReceipt
	if err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		var err error
		receipt, err = store.Accept(ctx, tx, second)
		return err
	}); err != nil || receipt.Disposition != postgreswebhook.AcceptanceAccepted {
		t.Fatalf("Accept(after rotation) = %+v, %v", receipt, err)
	}
	var activeKey string
	var destinationUntil, keyUntil time.Time
	if err := pool.PGX().QueryRow(ctx, `SELECT active_key_reference, destination_retained_until, key_references_retained_until FROM webhook_destinations WHERE owner_scope = 'owner-a' AND destination_id = 'dest-a' AND generation = 1`).Scan(&activeKey, &destinationUntil, &keyUntil); err != nil {
		t.Fatal(err)
	}
	if activeKey != "key-b" || destinationUntil.Before(receipt.AcceptedAt.Add(3*time.Hour)) || keyUntil.Before(receipt.AcceptedAt.Add(time.Hour)) {
		t.Fatalf("reused destination = key %q, destination %v, key %v, accepted %v", activeKey, destinationUntil, keyUntil, receipt.AcceptedAt)
	}
}
