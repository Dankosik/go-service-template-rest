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

func TestPostgresWebhookExpiredSlotRequiresReconciliationBeforeReuse(t *testing.T) {
	ctx, pool, store, manifest := newPostgresWebhookFixtureWithConcurrency(t, 1)
	for _, suffix := range []string{"slot-a", "slot-b"} {
		prepared := webhookPrepared(t, suffix)
		if err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error { _, err := store.Accept(ctx, tx, prepared); return err }); err != nil {
			t.Fatal(err)
		}
	}
	first, err := store.Claim(ctx, "worker-a", 2, time.Second, manifest)
	if err != nil || first.Attempt == nil {
		t.Fatalf("Claim(first) = %+v, %v", first, err)
	}
	if _, err := pool.PGX().Exec(ctx, `UPDATE webhook_attempts SET lease_expires_at = clock_timestamp() - interval '1 second' WHERE attempt_id = $1`, first.Attempt.Identity.AttemptID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.PGX().Exec(ctx, `UPDATE webhook_capacity_slots SET lease_expires_at = clock_timestamp() - interval '1 second' WHERE attempt_id = $1`, first.Attempt.Identity.AttemptID); err != nil {
		t.Fatal(err)
	}
	blocked, err := store.Claim(ctx, "worker-b", 2, time.Second, manifest)
	if err != nil || blocked.Attempt != nil {
		t.Fatalf("Claim(before reconciliation) = %+v, %v", blocked, err)
	}
	if reconciled, err := store.ReconcileExpired(ctx, 10); err != nil || reconciled != 1 {
		t.Fatalf("ReconcileExpired() = %d, %v", reconciled, err)
	}
	second, err := store.Claim(ctx, "worker-b", 2, time.Second, manifest)
	if err != nil || second.Attempt == nil || second.Attempt.Identity.AttemptID == first.Attempt.Identity.AttemptID {
		t.Fatalf("Claim(after reconciliation) = %+v, %v", second, err)
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
	if _, err := pool.PGX().Exec(ctx, `UPDATE webhook_operator_actions SET retained_until = clock_timestamp() - interval '1 second' WHERE action_id = $1`, action.ActionID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CleanupRetention(ctx, 10); err != nil {
		t.Fatal(err)
	}
	var actionCount int
	if err := pool.PGX().QueryRow(ctx, `SELECT count(*) FROM webhook_operator_actions WHERE action_id = $1`, action.ActionID).Scan(&actionCount); err != nil || actionCount != 1 {
		t.Fatalf("active redrive action count = %d, %v", actionCount, err)
	}
	next, err := store.Claim(ctx, "worker-b", 1, time.Second, manifest)
	if err != nil || next.Attempt == nil || next.Attempt.Identity.Cycle != 1 || next.Attempt.Identity.AttemptID == claim.Attempt.Identity.AttemptID {
		t.Fatalf("Claim(redrive) = %+v, %v", next, err)
	}
}

func TestPostgresWebhookRetryAfterStoresOnlyNormalizedEvidence(t *testing.T) {
	ctx, pool, store, manifest := newPostgresWebhookFixture(t)
	prepared := webhookPrepared(t, "retry-after-evidence")
	if err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error { _, err := store.Accept(ctx, tx, prepared); return err }); err != nil {
		t.Fatal(err)
	}
	claim, err := store.Claim(ctx, "worker-a", 1, 30*time.Second, manifest)
	if err != nil || claim.Attempt == nil {
		t.Fatalf("Claim() = %+v, %v", claim, err)
	}
	if _, err := store.FinalizeAttempt(ctx, *claim.Attempt, postgreswebhook.Finalization{Evidence: postgreswebhook.TransportEvidence{StatusCode: 429}, RetryAfter: "120", LocalRetryDelay: time.Second}); err != nil {
		t.Fatal(err)
	}
	var delay int64
	var source string
	if err := pool.PGX().QueryRow(ctx, `SELECT retry_after_delay_ms, retry_after_source FROM webhook_attempts WHERE attempt_id = $1`, claim.Attempt.Identity.AttemptID).Scan(&delay, &source); err != nil || delay != 60000 || source != "delay_seconds" {
		t.Fatalf("normalized Retry-After = %d/%q, %v", delay, source, err)
	}
}

func TestPostgresWebhookRedriveRevalidatesDestination(t *testing.T) {
	ctx, pool, store, manifest := newPostgresWebhookFixture(t)
	prepared := webhookPrepared(t, "redrive-disabled")
	if err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error { _, err := store.Accept(ctx, tx, prepared); return err }); err != nil {
		t.Fatal(err)
	}
	claim, err := store.Claim(ctx, "worker-a", 1, time.Second, manifest)
	if err != nil || claim.Attempt == nil {
		t.Fatalf("Claim() = %+v, %v", claim, err)
	}
	if _, err := pool.PGX().Exec(ctx, `UPDATE webhook_attempts SET lease_expires_at = clock_timestamp() - interval '1 second', may_have_sent = true, send_authorized = true WHERE attempt_id = $1`, claim.Attempt.Identity.AttemptID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ReconcileExpired(ctx, 10); err != nil {
		t.Fatal(err)
	}
	disable := postgreswebhook.ActionRequest{OwnerScope: "owner-a", Actor: "actor-a", ActionID: "disable-redrive", Kind: postgreswebhook.ActionDestinationState, TargetKind: "destination", TargetID: "dest-a", TargetGeneration: 1, Expected: "1", Reason: "admin_disable", Values: []string{"disabled", ""}}
	if receipt, err := store.ApplyAction(ctx, disable); err != nil || receipt.Result != "applied" {
		t.Fatalf("ApplyAction(disable) = %+v, %v", receipt, err)
	}
	redrive := postgreswebhook.ActionRequest{OwnerScope: "owner-a", Actor: "actor-a", ActionID: "redrive-disabled", Kind: postgreswebhook.ActionRedrive, TargetKind: "delivery", TargetID: claim.Attempt.Identity.DeliveryID, Expected: "0", Reason: "remediated", DuplicateRisk: true, Values: []string{"2", "3600000000000"}}
	if receipt, err := store.ApplyAction(ctx, redrive); err != nil || receipt.Result != "state_conflict" {
		t.Fatalf("ApplyAction(redrive disabled) = %+v, %v", receipt, err)
	}
}
