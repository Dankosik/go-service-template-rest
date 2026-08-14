//go:build integration

package integration_test

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

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
