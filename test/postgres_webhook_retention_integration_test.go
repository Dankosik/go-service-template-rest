//go:build integration

package integration_test

import (
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgreswebhook"
	"github.com/jackc/pgx/v5"
)

func TestPostgresWebhookRetention(t *testing.T) {
	ctx, pool, store, _ := newPostgresWebhookFixture(t)
	prepared := webhookPrepared(t, "retention")
	if err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error { _, err := store.Accept(ctx, tx, prepared); return err }); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.PGX().Exec(ctx, `UPDATE webhook_events SET payload_retained_until = clock_timestamp() - interval '1 second' WHERE owner_scope = 'owner-a' AND business_event_id = $1`, prepared.Acceptance.BusinessEventID); err != nil {
		t.Fatal(err)
	}
	if changed, err := store.CleanupRetention(ctx, 10); err != nil || changed != 0 {
		t.Fatalf("CleanupRetention(active payload) = %d, %v", changed, err)
	}
	if _, err := pool.PGX().Exec(ctx, `UPDATE webhook_cycles SET disposition = 'http_rejected', finalized_at = clock_timestamp() WHERE owner_scope = 'owner-a' AND delivery_id = $1`, prepared.Destinations[0].DeliveryID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.PGX().Exec(ctx, `UPDATE webhook_deliveries SET state = 'terminal', cumulative_summary = 'http_rejected', terminal_at = clock_timestamp(), redrive_eligible_until = clock_timestamp() - interval '1 second', terminal_retained_until = clock_timestamp() - interval '1 second' WHERE owner_scope = 'owner-a' AND delivery_id = $1`, prepared.Destinations[0].DeliveryID); err != nil {
		t.Fatal(err)
	}
	deleted, err := store.CleanupRetention(ctx, 10)
	if err != nil || deleted != 3 {
		t.Fatalf("CleanupRetention() = %d, %v", deleted, err)
	}
}

func TestPostgresWebhookKeyAndDestinationRetentionAreIndependent(t *testing.T) {
	ctx, pool, store, _ := newPostgresWebhookFixture(t)
	prepared := webhookPrepared(t, "destination-retention")
	if err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error { _, err := store.Accept(ctx, tx, prepared); return err }); err != nil {
		t.Fatal(err)
	}
	privacy := postgreswebhook.ActionRequest{OwnerScope: "owner-a", Actor: "privacy-a", ActionID: "erase-retention-event", Kind: postgreswebhook.ActionPrivacyDelete, TargetKind: "event", TargetID: prepared.Acceptance.BusinessEventID, Expected: "1", Reason: "privacy_request", Values: []string{"event", prepared.Acceptance.BusinessEventID, "minimal_tombstone", "privacy-ticket-a"}}
	if receipt, err := store.RequestEventPrivacyDeletion(ctx, privacy); err != nil || receipt.Result != "applied" {
		t.Fatalf("RequestEventPrivacyDeletion() = %+v, %v", receipt, err)
	}
	retire := postgreswebhook.ActionRequest{OwnerScope: "owner-a", Actor: "actor-a", ActionID: "retire-destination", Kind: postgreswebhook.ActionDestinationState, TargetKind: "destination", TargetID: "dest-a", TargetGeneration: 1, Expected: "1", Reason: "retired", Values: []string{"retired", ""}}
	if receipt, err := store.ApplyAction(ctx, retire); err != nil || receipt.Result != "applied" {
		t.Fatalf("ApplyAction(retire) = %+v, %v", receipt, err)
	}
	if _, err := pool.PGX().Exec(ctx, `UPDATE webhook_destinations SET key_references_retained_until = clock_timestamp() - interval '1 second', destination_retained_until = clock_timestamp() + interval '1 hour' WHERE owner_scope = 'owner-a' AND destination_id = 'dest-a'`); err != nil {
		t.Fatal(err)
	}
	if changed, err := store.CleanupRetention(ctx, 10); err != nil || changed != 1 {
		t.Fatalf("CleanupRetention(key references) = %d, %v", changed, err)
	}
	var activeKey *string
	var erasedAt *time.Time
	if err := pool.PGX().QueryRow(ctx, `SELECT active_key_reference, key_references_erased_at FROM webhook_destinations WHERE owner_scope = 'owner-a' AND destination_id = 'dest-a'`).Scan(&activeKey, &erasedAt); err != nil || activeKey != nil || erasedAt == nil {
		t.Fatalf("erased key references = %v/%v, %v", activeKey, erasedAt, err)
	}
	if _, err := pool.PGX().Exec(ctx, `UPDATE webhook_destinations SET destination_retained_until = clock_timestamp() - interval '1 second' WHERE owner_scope = 'owner-a' AND destination_id = 'dest-a'`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.PGX().Exec(ctx, `UPDATE webhook_operator_actions SET retained_until = clock_timestamp() - interval '1 second' WHERE owner_scope = 'owner-a' AND action_id = 'retire-destination'`); err != nil {
		t.Fatal(err)
	}
	if changed, err := store.CleanupRetention(ctx, 10); err != nil || changed != 2 {
		t.Fatalf("CleanupRetention(destination) = %d, %v", changed, err)
	}
}

func TestPostgresWebhookZeroAttemptDeadlineExpiry(t *testing.T) {
	ctx, pool, store, _ := newPostgresWebhookFixture(t)
	prepared := webhookPrepared(t, "zero-attempt-expiry")
	if err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error { _, err := store.Accept(ctx, tx, prepared); return err }); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.PGX().Exec(ctx, `UPDATE webhook_cycles SET accepted_at = clock_timestamp() - interval '2 hours', deadline_at = clock_timestamp() - interval '1 hour' WHERE owner_scope = 'owner-a' AND delivery_id = $1`, prepared.Destinations[0].DeliveryID); err != nil {
		t.Fatal(err)
	}
	if finalized, err := store.FinalizeExpiredCycles(ctx, 10); err != nil || finalized != 1 {
		t.Fatalf("FinalizeExpiredCycles() = %d, %v", finalized, err)
	}
	var state, summary string
	var attempts int
	if err := pool.PGX().QueryRow(ctx, `SELECT d.state, d.cumulative_summary, c.attempts_used FROM webhook_deliveries d JOIN webhook_cycles c USING (owner_scope, delivery_id) WHERE d.owner_scope = 'owner-a' AND d.delivery_id = $1`, prepared.Destinations[0].DeliveryID).Scan(&state, &summary, &attempts); err != nil || state != "terminal" || summary != "attempts_exhausted" || attempts != 0 {
		t.Fatalf("expired zero-attempt delivery = %s/%s/%d, %v", state, summary, attempts, err)
	}
}
