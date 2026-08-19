//go:build integration

package integration_test

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgreswebhook"
	"github.com/jackc/pgx/v5"
)

func TestPostgresWebhookAttemptIdentity(t *testing.T) {
	ctx, pool, store, manifest := newPostgresWebhookFixture(t)
	prepared := webhookPrepared(t, "identity")
	if err := postgres.InTx(ctx, pool, pgx.TxOptions{}, func(tx pgx.Tx) error { _, err := store.Accept(ctx, tx, prepared); return err }); err != nil {
		t.Fatal(err)
	}
	claim, err := store.Claim(ctx, "worker-a", 1, 30*time.Second, manifest)
	if err != nil || claim.Attempt == nil || claim.Attempt.Identity.AttemptID == "" || claim.Attempt.Identity.Fence != 1 {
		t.Fatalf("Claim() = %+v, %v", claim, err)
	}
}

func TestPostgresWebhookRetryEvidenceAndExhaustion(t *testing.T) {
	ctx, pool, store, manifest := newPostgresWebhookFixture(t)
	prepared := webhookPrepared(t, "retry-evidence")
	if err := postgres.InTx(ctx, pool, pgx.TxOptions{}, func(tx pgx.Tx) error { _, err := store.Accept(ctx, tx, prepared); return err }); err != nil {
		t.Fatal(err)
	}
	claim, err := store.Claim(ctx, "worker-a", 1, 30*time.Second, manifest)
	if err != nil || claim.Attempt == nil {
		t.Fatalf("Claim() = %+v, %v", claim, err)
	}
	if _, err := store.FinalizeAttempt(ctx, *claim.Attempt, postgreswebhook.Finalization{
		Evidence: postgreswebhook.TransportEvidence{DefinitelyNotSent: true}, RetryAfter: strings.Repeat("9", 512),
		LocalRetryDelay: 2 * time.Second,
	}); err != nil {
		t.Fatal(err)
	}
	var raw *string
	var retryAfterNS, retryDelayNS *int64
	if err := pool.QueryRow(ctx, `SELECT retry_after, retry_after_delay_ns, retry_delay_ns FROM webhook_attempts WHERE attempt_id = $1`, claim.Attempt.Identity.AttemptID).Scan(&raw, &retryAfterNS, &retryDelayNS); err != nil {
		t.Fatal(err)
	}
	if raw != nil || retryAfterNS == nil || *retryAfterNS != int64(time.Minute) || retryDelayNS == nil || *retryDelayNS != int64(2*time.Second) {
		t.Fatalf("normalized retry evidence = raw:%v retry-after:%v retry:%v", raw, retryAfterNS, retryDelayNS)
	}
	if _, err := pool.Exec(ctx, `UPDATE webhook_deliveries SET next_due_at = clock_timestamp() - interval '1 second' WHERE delivery_id = $1`, claim.Attempt.Identity.DeliveryID); err != nil {
		t.Fatal(err)
	}
	next, err := store.Claim(ctx, "worker-b", 1, 30*time.Second, manifest)
	if err != nil || next.Attempt == nil || next.Attempt.PreviousRetryDelay != 2*time.Second {
		t.Fatalf("Claim(progressive retry) = %+v, %v", next, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE webhook_attempts SET lease_expires_at = clock_timestamp() - interval '1 second' WHERE attempt_id = $1`, next.Attempt.Identity.AttemptID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE webhook_capacity_slots SET lease_expires_at = clock_timestamp() - interval '1 second' WHERE attempt_id = $1`, next.Attempt.Identity.AttemptID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ReconcileExpired(ctx, 10); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE webhook_deliveries SET next_due_at = clock_timestamp() - interval '1 second' WHERE delivery_id = $1`, next.Attempt.Identity.DeliveryID); err != nil {
		t.Fatal(err)
	}
	last, err := store.Claim(ctx, "worker-c", 1, 30*time.Second, manifest)
	if err != nil || last.Attempt == nil {
		t.Fatalf("Claim(last attempt) = %+v, %v", last, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE webhook_attempts SET lease_expires_at = clock_timestamp() - interval '1 second' WHERE attempt_id = $1`, last.Attempt.Identity.AttemptID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE webhook_capacity_slots SET lease_expires_at = clock_timestamp() - interval '1 second' WHERE attempt_id = $1`, last.Attempt.Identity.AttemptID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ReconcileExpired(ctx, 10); err != nil {
		t.Fatal(err)
	}
	var state, summary, disposition string
	var attempts int
	if err := pool.QueryRow(ctx, `SELECT d.state, d.cumulative_summary, c.disposition, c.attempts_used FROM webhook_deliveries d JOIN webhook_cycles c ON c.owner_scope = d.owner_scope AND c.delivery_id = d.delivery_id AND c.cycle_number = d.current_cycle WHERE d.delivery_id = $1`, last.Attempt.Identity.DeliveryID).Scan(&state, &summary, &disposition, &attempts); err != nil {
		t.Fatal(err)
	}
	if state != "terminal" || summary != "attempts_exhausted" || disposition != "attempts_exhausted" || attempts != 3 {
		t.Fatalf("exhausted delivery = %s/%s/%s attempts=%d", state, summary, disposition, attempts)
	}
	if claim, err := store.Claim(ctx, "worker-d", 1, 30*time.Second, manifest); err != nil || claim.Attempt != nil {
		t.Fatalf("Claim(exhausted) = %+v, %v", claim, err)
	}
}

func TestPostgresWebhookZeroAttemptDeadlineExhaustion(t *testing.T) {
	ctx, pool, store, _ := newPostgresWebhookFixture(t)
	prepared := webhookPrepared(t, "zero-attempt-expiry")
	if err := postgres.InTx(ctx, pool, pgx.TxOptions{}, func(tx pgx.Tx) error { _, err := store.Accept(ctx, tx, prepared); return err }); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE webhook_cycles SET accepted_at = clock_timestamp() - interval '2 seconds', deadline_at = clock_timestamp() - interval '1 second' WHERE delivery_id = $1`, prepared.Destinations[0].DeliveryID); err != nil {
		t.Fatal(err)
	}
	if reconciled, err := store.ReconcileExpired(ctx, 10); err != nil || reconciled != 1 {
		t.Fatalf("ReconcileExpired(zero-attempt) = %d, %v", reconciled, err)
	}
	var state, summary string
	if err := pool.QueryRow(ctx, `SELECT state, cumulative_summary FROM webhook_deliveries WHERE delivery_id = $1`, prepared.Destinations[0].DeliveryID).Scan(&state, &summary); err != nil {
		t.Fatal(err)
	}
	if state != "terminal" || summary != "attempts_exhausted" {
		t.Fatalf("zero-attempt expiry = %s/%s", state, summary)
	}
}

func TestPostgresWebhookQuarantinesLedgerDivergence(t *testing.T) {
	ctx, pool, store, manifest := newPostgresWebhookFixture(t)
	prepared := webhookPrepared(t, "ledger-divergence")
	if err := postgres.InTx(ctx, pool, pgx.TxOptions{}, func(tx pgx.Tx) error { _, err := store.Accept(ctx, tx, prepared); return err }); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE webhook_cycles SET disposition = 'http_rejected', finalized_at = clock_timestamp() WHERE delivery_id = $1`, prepared.Destinations[0].DeliveryID); err != nil {
		t.Fatal(err)
	}
	if reconciled, err := store.ReconcileExpired(ctx, 10); err != nil || reconciled != 1 {
		t.Fatalf("ReconcileExpired(divergence) = %d, %v", reconciled, err)
	}
	var state string
	var sendable bool
	if err := pool.QueryRow(ctx, `SELECT state, sendable FROM webhook_deliveries WHERE delivery_id = $1`, prepared.Destinations[0].DeliveryID).Scan(&state, &sendable); err != nil {
		t.Fatal(err)
	}
	if state != "quarantined" || sendable {
		t.Fatalf("divergent delivery = %s sendable=%t", state, sendable)
	}
	if claim, err := store.Claim(ctx, "worker-a", 1, 30*time.Second, manifest); err != nil || claim.Attempt != nil {
		t.Fatalf("Claim(quarantined) = %+v, %v", claim, err)
	}
}

func TestPostgresWebhookFencedRecovery(t *testing.T) {
	ctx, pool, store, manifest := newPostgresWebhookFixture(t)
	prepared := webhookPrepared(t, "recovery")
	if err := postgres.InTx(ctx, pool, pgx.TxOptions{}, func(tx pgx.Tx) error { _, err := store.Accept(ctx, tx, prepared); return err }); err != nil {
		t.Fatal(err)
	}
	claim, err := store.Claim(ctx, "worker-a", 1, 30*time.Second, manifest)
	if err != nil || claim.Attempt == nil {
		t.Fatalf("Claim() = %+v, %v", claim, err)
	}
	authorization := postgreswebhook.AuthorizationEvidence{KeyReference: "key-a", KeyReferences: []string{"key-a"}, SelectedAddress: netip.MustParseAddr("8.8.8.8")}
	if err := store.AuthorizeAttempt(ctx, *claim.Attempt, manifest, authorization); err != nil {
		t.Fatalf("AuthorizeAttempt(): %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE webhook_attempts SET lease_expires_at = clock_timestamp() - interval '1 second'; UPDATE webhook_capacity_slots SET lease_expires_at = clock_timestamp() - interval '1 second' WHERE attempt_id IS NOT NULL`); err != nil {
		t.Fatal(err)
	}
	reconciled, err := store.ReconcileExpired(ctx, 10)
	if err != nil || reconciled != 1 {
		t.Fatalf("ReconcileExpired() = %d, %v", reconciled, err)
	}
	var state, summary string
	var nextDue time.Time
	var retryDelay int64
	if err := pool.QueryRow(ctx, `SELECT d.state, d.cumulative_summary, d.next_due_at, a.retry_delay_ns FROM webhook_deliveries d JOIN webhook_attempts a ON a.owner_scope = d.owner_scope AND a.delivery_id = d.delivery_id WHERE a.attempt_id = $1`, claim.Attempt.Identity.AttemptID).Scan(&state, &summary, &nextDue, &retryDelay); err != nil || state != "scheduled" || summary != "outcome_unknown" || !nextDue.After(time.Now()) || retryDelay <= 0 {
		t.Fatalf("recovered delivery = %s/%s due:%s retry:%d, %v", state, summary, nextDue, retryDelay, err)
	}
	if _, err := store.FinalizeAttempt(ctx, *claim.Attempt, postgreswebhook.Finalization{Evidence: postgreswebhook.TransportEvidence{MayHaveSent: true}}); !errors.Is(err, postgreswebhook.ErrStaleAttempt) {
		t.Fatalf("stale FinalizeAttempt() error = %v", err)
	}
}

func TestPostgresWebhookCapacitySlotRequiresReconciliation(t *testing.T) {
	ctx, pool, store, _ := newPostgresWebhookFixture(t)
	secretA := base64.StdEncoding.EncodeToString([]byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
	secretB := base64.StdEncoding.EncodeToString([]byte("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"))
	secretC := base64.StdEncoding.EncodeToString([]byte("cccccccccccccccccccccccccccccccc"))
	manifest, err := postgreswebhook.ParseSecretManifest(fmt.Sprintf(`{"revision":1,"entries":[{"owner_scope":"owner-a","destination_id":"dest-a","key_reference":"key-a","secret":"whsec_%s"},{"owner_scope":"owner-a","destination_id":"dest-b","key_reference":"key-a","secret":"whsec_%s"},{"owner_scope":"owner-a","destination_id":"dest-c","key_reference":"key-a","secret":"whsec_%s"}]}`, secretA, secretB, secretC))
	if err != nil {
		t.Fatal(err)
	}
	for i, destinationID := range []string{"dest-a", "dest-b", "dest-c"} {
		prepared := webhookPrepared(t, fmt.Sprintf("slot-%d", i))
		input := prepared.Acceptance
		destination := prepared.Destinations[0].DestinationSnapshot
		destination.DestinationID = destinationID
		input.Destinations = []postgreswebhook.DestinationSnapshot{destination}
		prepared, err = postgreswebhook.PrepareAcceptance(input)
		if err != nil {
			t.Fatal(err)
		}
		if err := postgres.InTx(ctx, pool, pgx.TxOptions{}, func(tx pgx.Tx) error { _, err := store.Accept(ctx, tx, prepared); return err }); err != nil {
			t.Fatal(err)
		}
	}
	first, err := store.Claim(ctx, "worker-a", 3, 30*time.Second, manifest)
	if err != nil || first.Attempt == nil {
		t.Fatalf("Claim(first) = %+v, %v", first, err)
	}
	second, err := store.Claim(ctx, "worker-b", 3, 30*time.Second, manifest)
	if err != nil || second.Attempt == nil {
		t.Fatalf("Claim(second) = %+v, %v", second, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE webhook_attempts SET may_have_sent = true, lease_expires_at = clock_timestamp() - interval '1 second' WHERE attempt_id = $1`, first.Attempt.Identity.AttemptID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE webhook_capacity_slots SET lease_expires_at = clock_timestamp() - interval '1 second' WHERE attempt_id = $1`, first.Attempt.Identity.AttemptID); err != nil {
		t.Fatal(err)
	}
	if blocked, err := store.Claim(ctx, "worker-c", 3, 30*time.Second, manifest); err != nil || blocked.Attempt != nil {
		t.Fatalf("Claim(before reconciliation) = %+v, %v", blocked, err)
	}
	if reconciled, err := store.ReconcileExpired(ctx, 10); err != nil || reconciled != 1 {
		t.Fatalf("ReconcileExpired() = %d, %v", reconciled, err)
	}
	third, err := store.Claim(ctx, "worker-c", 3, 30*time.Second, manifest)
	if err != nil || third.Attempt == nil {
		t.Fatalf("Claim(after reconciliation) = %+v, %v", third, err)
	}
	if _, err := store.FinalizeAttempt(ctx, *first.Attempt, postgreswebhook.Finalization{Evidence: postgreswebhook.TransportEvidence{MayHaveSent: true}}); !errors.Is(err, postgreswebhook.ErrStaleAttempt) {
		t.Fatalf("FinalizeAttempt(stale) error = %v", err)
	}
	var leased, replacement int
	if err := pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE attempt_id IS NOT NULL), count(*) FILTER (WHERE attempt_id = $1) FROM webhook_capacity_slots`, third.Attempt.Identity.AttemptID).Scan(&leased, &replacement); err != nil || leased != 2 || replacement != 1 {
		t.Fatalf("capacity after stale finalize = leased:%d replacement:%d err:%v", leased, replacement, err)
	}
}

func TestPostgresWebhookRedrive(t *testing.T) {
	ctx, pool, store, manifest := newPostgresWebhookFixture(t)
	prepared := webhookPrepared(t, "redrive")
	if err := postgres.InTx(ctx, pool, pgx.TxOptions{}, func(tx pgx.Tx) error { _, err := store.Accept(ctx, tx, prepared); return err }); err != nil {
		t.Fatal(err)
	}
	claim, _ := store.Claim(ctx, "worker-a", 1, 30*time.Second, manifest)
	if claim.Attempt == nil {
		t.Fatal("Claim() returned no attempt")
	}
	authorization := postgreswebhook.AuthorizationEvidence{KeyReference: "key-a", KeyReferences: []string{"key-a"}, SelectedAddress: netip.MustParseAddr("8.8.8.8")}
	if err := store.AuthorizeAttempt(ctx, *claim.Attempt, manifest, authorization); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE webhook_attempts SET lease_expires_at = clock_timestamp() - interval '1 second'; UPDATE webhook_capacity_slots SET lease_expires_at = clock_timestamp() - interval '1 second' WHERE attempt_id IS NOT NULL`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE webhook_cycles SET attempts_used = maximum_attempts WHERE delivery_id = $1`, claim.Attempt.Identity.DeliveryID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ReconcileExpired(ctx, 10); err != nil {
		t.Fatal(err)
	}
	action := postgreswebhook.ActionRequest{OwnerScope: "owner-a", Actor: "actor-a", ActionID: "redrive-a", Kind: postgreswebhook.ActionRedrive, TargetKind: "delivery", TargetID: claim.Attempt.Identity.DeliveryID, Reason: "remediated", Payload: &postgreswebhook.RedriveAction{MaximumAttempts: 2, MaximumAge: time.Hour, AcknowledgeDuplicateRisk: true}}
	missingKey := action
	missingKey.ActionID = "redrive-missing-key"
	if receipt, err := store.ApplyAction(ctx, missingKey, webhookManifest(t, 1, "owner-a", "dest-a", "other-key")); err != nil || receipt.Result != "state_conflict" {
		t.Fatalf("ApplyAction(missing key) = %+v, %v", receipt, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE webhook_events SET body = NULL WHERE business_event_id = $1`, prepared.Acceptance.BusinessEventID); err != nil {
		t.Fatal(err)
	}
	missingPayload := action
	missingPayload.ActionID = "redrive-missing-payload"
	if receipt, err := store.ApplyAction(ctx, missingPayload, manifest); err != nil || receipt.Result != "state_conflict" {
		t.Fatalf("ApplyAction(missing payload) = %+v, %v", receipt, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE webhook_events SET body = $1 WHERE business_event_id = $2`, prepared.Acceptance.Body, prepared.Acceptance.BusinessEventID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE webhook_deliveries SET receiver_dedup_retained_until = clock_timestamp() - interval '1 second' WHERE delivery_id = $1`, claim.Attempt.Identity.DeliveryID); err != nil {
		t.Fatal(err)
	}
	expiredRetention := action
	expiredRetention.ActionID = "redrive-expired-retention"
	if receipt, err := store.ApplyAction(ctx, expiredRetention, manifest); err != nil || receipt.Result != "state_conflict" {
		t.Fatalf("ApplyAction(expired retention) = %+v, %v", receipt, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE webhook_deliveries SET receiver_dedup_retained_until = clock_timestamp() + interval '1 hour' WHERE delivery_id = $1`, claim.Attempt.Identity.DeliveryID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE webhook_destinations SET disposition = 'administratively_disabled' WHERE destination_id = 'dest-a'`); err != nil {
		t.Fatal(err)
	}
	inactive := action
	inactive.ActionID = "redrive-inactive-destination"
	if receipt, err := store.ApplyAction(ctx, inactive, manifest); err != nil || receipt.Result != "state_conflict" {
		t.Fatalf("ApplyAction(inactive destination) = %+v, %v", receipt, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE webhook_destinations SET disposition = 'active' WHERE destination_id = 'dest-a'`); err != nil {
		t.Fatal(err)
	}
	incompatible, err := postgreswebhook.NewStore(pool, postgreswebhook.StoreOptions{OperationTimeout: 3 * time.Second, CapacityRevision: 1, GlobalConcurrency: 2, ManifestRevision: 1, AttemptTimeout: 5 * time.Second, ResponseHeaderTimeout: 2 * time.Second, ResponseHeaderBytes: 1024, ResponseBodyBytes: 1024, DrainTimeout: 10 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	incompatiblePolicy := action
	incompatiblePolicy.ActionID = "redrive-incompatible-policy"
	if _, err := incompatible.ApplyAction(ctx, incompatiblePolicy, manifest); !errors.Is(err, postgreswebhook.ErrConflict) {
		t.Fatalf("ApplyAction(incompatible worker policy) error = %v", err)
	}
	receipt, err := store.ApplyAction(ctx, action, manifest)
	if err != nil || receipt.Result != "applied" || receipt.Cycle != 1 {
		t.Fatalf("ApplyAction(redrive) = %+v, %v", receipt, err)
	}
	replay, err := store.ApplyAction(ctx, action, manifest)
	if err != nil || !replay.Replay || replay.Result != "applied" || replay.Cycle != 1 {
		t.Fatalf("ApplyAction(replay) = %+v, %v", replay, err)
	}
	var payload []byte
	var resultCycle int64
	if err := pool.QueryRow(ctx, `SELECT request_payload, result_cycle FROM webhook_operator_actions WHERE action_id = $1`, action.ActionID).Scan(&payload, &resultCycle); err != nil || len(payload) == 0 || resultCycle != 1 {
		t.Fatalf("redrive action evidence = payload:%s cycle:%d err:%v", payload, resultCycle, err)
	}
	next, err := store.Claim(ctx, "worker-b", 1, 30*time.Second, manifest)
	if err != nil || next.Attempt == nil || next.Attempt.Identity.Cycle != 1 || next.Attempt.Identity.AttemptID == claim.Attempt.Identity.AttemptID {
		t.Fatalf("Claim(redrive) = %+v, %v", next, err)
	}
}
