//go:build integration

package integration_test

import (
	"errors"
	"net/netip"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgreswebhook"
	"github.com/jackc/pgx/v5"
)

func TestPostgresWebhookAttemptIdentity(t *testing.T) {
	ctx, pool, store, manifest := newPostgresWebhookFixture(t)
	prepared := webhookPrepared(t, "identity")
	if err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error { _, err := store.Accept(ctx, tx, prepared); return err }); err != nil {
		t.Fatal(err)
	}
	claim, err := store.Claim(ctx, "worker-a", 1, time.Second, manifest)
	if err != nil || claim.Attempt == nil || claim.Attempt.Identity.AttemptID == "" || claim.Attempt.Identity.Fence != 1 {
		t.Fatalf("Claim() = %+v, %v", claim, err)
	}
}

func TestPostgresWebhookFencedRecovery(t *testing.T) {
	ctx, pool, store, manifest := newPostgresWebhookFixture(t)
	prepared := webhookPrepared(t, "recovery")
	if err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error { _, err := store.Accept(ctx, tx, prepared); return err }); err != nil {
		t.Fatal(err)
	}
	claim, err := store.Claim(ctx, "worker-a", 1, time.Second, manifest)
	if err != nil || claim.Attempt == nil {
		t.Fatalf("Claim() = %+v, %v", claim, err)
	}
	authorization := postgreswebhook.AuthorizationEvidence{KeyReference: "key-a", SelectedAddress: netip.MustParseAddr("8.8.8.8")}
	if err := store.AuthorizeAttempt(ctx, *claim.Attempt, manifest, authorization); err != nil {
		t.Fatalf("AuthorizeAttempt(): %v", err)
	}
	if _, err := pool.PGX().Exec(ctx, `UPDATE webhook_attempts SET lease_expires_at = clock_timestamp() - interval '1 second'; UPDATE webhook_capacity_slots SET lease_expires_at = clock_timestamp() - interval '1 second' WHERE attempt_id IS NOT NULL`); err != nil {
		t.Fatal(err)
	}
	reconciled, err := store.ReconcileExpired(ctx, 10)
	if err != nil || reconciled != 1 {
		t.Fatalf("ReconcileExpired() = %d, %v", reconciled, err)
	}
	var state, summary string
	if err := pool.PGX().QueryRow(ctx, `SELECT state, cumulative_summary FROM webhook_deliveries WHERE delivery_id = $1`, claim.Attempt.Identity.DeliveryID).Scan(&state, &summary); err != nil || state != "suspended" || summary != "outcome_unknown" {
		t.Fatalf("recovered delivery = %s/%s, %v", state, summary, err)
	}
	if _, err := store.FinalizeAttempt(ctx, *claim.Attempt, postgreswebhook.Finalization{Evidence: postgreswebhook.TransportEvidence{MayHaveSent: true}}); !errors.Is(err, postgreswebhook.ErrStaleAttempt) {
		t.Fatalf("stale FinalizeAttempt() error = %v", err)
	}
}

func TestPostgresWebhookRedrive(t *testing.T) {
	ctx, pool, store, manifest := newPostgresWebhookFixture(t)
	prepared := webhookPrepared(t, "redrive")
	if err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error { _, err := store.Accept(ctx, tx, prepared); return err }); err != nil {
		t.Fatal(err)
	}
	claim, _ := store.Claim(ctx, "worker-a", 1, time.Second, manifest)
	if _, err := pool.PGX().Exec(ctx, `UPDATE webhook_attempts SET lease_expires_at = clock_timestamp() - interval '1 second'; UPDATE webhook_capacity_slots SET lease_expires_at = clock_timestamp() - interval '1 second' WHERE attempt_id IS NOT NULL; UPDATE webhook_attempts SET may_have_sent = true, send_authorized = true`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ReconcileExpired(ctx, 10); err != nil {
		t.Fatal(err)
	}
	action := postgreswebhook.ActionRequest{OwnerScope: "owner-a", Actor: "actor-a", ActionID: "redrive-a", Kind: postgreswebhook.ActionRedrive, TargetKind: "delivery", TargetID: claim.Attempt.Identity.DeliveryID, Expected: "0", Reason: "remediated", DuplicateRisk: true, Values: []string{"2", "3600000000000"}}
	receipt, err := store.ApplyAction(ctx, action)
	if err != nil || receipt.Result != "applied" || receipt.Cycle != 1 {
		t.Fatalf("ApplyAction(redrive) = %+v, %v", receipt, err)
	}
	replay, err := store.ApplyAction(ctx, action)
	if err != nil || !replay.Replay || replay.Result != "applied" {
		t.Fatalf("ApplyAction(replay) = %+v, %v", replay, err)
	}
	next, err := store.Claim(ctx, "worker-b", 1, time.Second, manifest)
	if err != nil || next.Attempt == nil || next.Attempt.Identity.Cycle != 1 || next.Attempt.Identity.AttemptID == claim.Attempt.Identity.AttemptID {
		t.Fatalf("Claim(redrive) = %+v, %v", next, err)
	}
}
