//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/example/go-service-template-rest/internal/infra/postgreswebhook"
	"github.com/jackc/pgx/v5"
)

func TestPostgresWebhookAcceptance(t *testing.T) {
	t.Parallel()
	ctx, pool, store, _ := newPostgresWebhookFixture(t)
	if _, err := pool.PGX().Exec(ctx, `CREATE TABLE webhook_business_fixture (id text PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}
	prepared := webhookPrepared(t, "atomic")
	receipt, err := store.AcceptAtomic(ctx, prepared, func(ctx context.Context, tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `INSERT INTO webhook_business_fixture VALUES ('atomic')`); err != nil {
			return err
		}
		return nil
	})
	if err != nil || receipt.Disposition != postgreswebhook.AcceptanceAccepted || len(receipt.DeliveryIDs) != 1 {
		t.Fatalf("AcceptAtomic() = %+v, %v", receipt, err)
	}
	reconstructedInput := prepared.Acceptance
	reconstructedInput.Destinations = []postgreswebhook.DestinationSnapshot{prepared.Destinations[0].DestinationSnapshot}
	reconstructed, err := postgreswebhook.PrepareAcceptance(reconstructedInput)
	if err != nil || reconstructed.Destinations[0].DeliveryID != prepared.Destinations[0].DeliveryID {
		t.Fatalf("PrepareAcceptance(reconstructed) = %+v, %v", reconstructed, err)
	}
	readback, err := store.ResolveAcceptance(ctx, reconstructed)
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
	_, err = store.AcceptAtomic(ctx, rolledBack, func(ctx context.Context, tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `INSERT INTO webhook_business_fixture VALUES ('rollback')`); err != nil {
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
	if err := pool.PGX().QueryRow(ctx, `SELECT count(*) FROM webhook_business_fixture WHERE id = 'rollback'`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("rolled-back feature count = %d, %v", count, err)
	}
}

func TestPostgresWebhookAcceptanceReadback(t *testing.T) {
	t.Parallel()
	ctx, _, store, _ := newPostgresWebhookFixture(t)
	prepared := webhookPrepared(t, "absent")
	receipt, err := store.ResolveAcceptance(ctx, prepared)
	if err != nil || receipt.Disposition != postgreswebhook.AcceptanceRejected {
		t.Fatalf("ResolveAcceptance(absent) = %+v, %v", receipt, err)
	}
}

func TestPostgresWebhookAcceptanceCollisionAndDestinationState(t *testing.T) {
	t.Parallel()
	ctx, pool, store, manifest := newPostgresWebhookFixture(t)
	accepted := webhookPrepared(t, "collision-a")
	if err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error { _, err := store.Accept(ctx, tx, accepted); return err }); err != nil {
		t.Fatal(err)
	}

	conflictingInput := accepted.Acceptance
	conflictingInput.AcceptanceID = "accept-collision-b"
	conflictingInput.FanoutSnapshotID = "fanout-collision-b"
	conflictingInput.Destinations = []postgreswebhook.DestinationSnapshot{accepted.Destinations[0].DestinationSnapshot}
	conflicting, err := postgreswebhook.PrepareAcceptance(conflictingInput)
	if err != nil {
		t.Fatal(err)
	}
	err = pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		receipt, err := store.Accept(ctx, tx, conflicting)
		if !errors.Is(err, postgreswebhook.ErrConflict) || receipt.Disposition != postgreswebhook.AcceptanceConflict {
			return fmt.Errorf("Accept(conflict) = %+v, %w", receipt, err)
		}
		return err
	})
	if !errors.Is(err, postgreswebhook.ErrConflict) {
		t.Fatalf("collision error = %v", err)
	}

	disable := postgreswebhook.ActionRequest{OwnerScope: "owner-a", Actor: "operator-a", ActionID: "disable-a", Kind: postgreswebhook.ActionDestinationState, TargetKind: "destination", TargetID: "dest-a", TargetGeneration: 1, ExpectedRevision: 1, Reason: "admin_disable", Payload: &postgreswebhook.DestinationStateAction{Disposition: "disabled"}}
	if receipt, err := store.ApplyAction(ctx, disable, manifest); err != nil || receipt.Result != "applied" {
		t.Fatalf("ApplyAction(disable) = %+v, %v", receipt, err)
	}
	if err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		receipt, err := store.Accept(ctx, tx, accepted)
		if err != nil || receipt.Disposition != postgreswebhook.AcceptanceAccepted {
			return fmt.Errorf("Accept(replay after disable) = %+v, %w", receipt, err)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	newInput := conflictingInput
	newInput.BusinessEventID = "event-collision-b"
	newAcceptance, err := postgreswebhook.PrepareAcceptance(newInput)
	if err != nil {
		t.Fatal(err)
	}
	err = pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error { _, err := store.Accept(ctx, tx, newAcceptance); return err })
	if !errors.Is(err, postgreswebhook.ErrConflict) {
		t.Fatalf("Accept(disabled generation) error = %v", err)
	}
	var events, fanouts, deliveries, cycles int
	if err := pool.PGX().QueryRow(ctx, `SELECT (SELECT count(*) FROM webhook_events), (SELECT count(*) FROM webhook_fanouts), (SELECT count(*) FROM webhook_deliveries), (SELECT count(*) FROM webhook_cycles)`).Scan(&events, &fanouts, &deliveries, &cycles); err != nil {
		t.Fatal(err)
	}
	if events != 1 || fanouts != 1 || deliveries != 1 || cycles != 1 {
		t.Fatalf("partial authority remained: events=%d fanouts=%d deliveries=%d cycles=%d", events, fanouts, deliveries, cycles)
	}
}
