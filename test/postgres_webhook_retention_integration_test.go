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

func TestPostgresWebhookRetention(t *testing.T) {
	ctx, pool, store, manifest := newPostgresWebhookFixture(t)
	prepared := webhookPrepared(t, "retention")
	if err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error { _, err := store.Accept(ctx, tx, prepared); return err }); err != nil {
		t.Fatal(err)
	}
	claim, err := store.Claim(ctx, "worker-a", 1, 30*time.Second, manifest)
	if err != nil || claim.Attempt == nil {
		t.Fatalf("Claim() = %+v, %v", claim, err)
	}
	if err := store.AuthorizeAttempt(ctx, *claim.Attempt, manifest, postgreswebhook.AuthorizationEvidence{KeyReference: "key-a", KeyReferences: []string{"key-a"}, SelectedAddress: netip.MustParseAddr("8.8.8.8")}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.FinalizeAttempt(ctx, *claim.Attempt, postgreswebhook.Finalization{Evidence: postgreswebhook.TransportEvidence{StatusCode: 400, MayHaveSent: true}}); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.PGX().Exec(ctx, `UPDATE webhook_deliveries SET state = 'terminal', cumulative_summary = 'http_rejected', terminal_at = clock_timestamp(),
		redrive_eligible_until = clock_timestamp() - interval '1 second', payload_retained_until = clock_timestamp() - interval '1 second',
		active_retained_until = clock_timestamp() - interval '1 second', terminal_summary_retained_until = clock_timestamp() + interval '1 hour',
		attempt_retained_until = clock_timestamp() - interval '1 second', action_retained_until = clock_timestamp() + interval '1 hour',
		destination_generation_retained_until = clock_timestamp() + interval '1 hour', receiver_dedup_retained_until = clock_timestamp() + interval '1 hour'
		WHERE owner_scope = 'owner-a' AND delivery_id = $1`, prepared.Destinations[0].DeliveryID); err != nil {
		t.Fatal(err)
	}
	hold := postgreswebhook.ActionRequest{OwnerScope: "owner-a", Actor: "legal-a", ActionID: "hold-on", Kind: postgreswebhook.ActionRetentionHold, TargetKind: "delivery", TargetID: prepared.Destinations[0].DeliveryID, Reason: "legal_hold", Payload: &postgreswebhook.RetentionHoldAction{Enabled: true}}
	if receipt, err := store.ApplyAction(ctx, hold, manifest); err != nil || receipt.Result != "applied" {
		t.Fatalf("ApplyAction(hold) = %+v, %v", receipt, err)
	}
	if deleted, err := store.CleanupRetention(ctx, 10); err != nil || deleted != 0 {
		t.Fatalf("CleanupRetention(held) = %d, %v", deleted, err)
	}
	hold.ActionID = "hold-off"
	hold.Payload = &postgreswebhook.RetentionHoldAction{}
	if receipt, err := store.ApplyAction(ctx, hold, manifest); err != nil || receipt.Result != "applied" {
		t.Fatalf("ApplyAction(release hold) = %+v, %v", receipt, err)
	}
	deleted, err := store.CleanupRetention(ctx, 10)
	if err != nil || deleted != 3 {
		t.Fatalf("CleanupRetention(payload, attempt, and cycle) = %d, %v", deleted, err)
	}
	var events, attempts, cycles int
	var body []byte
	if err := pool.PGX().QueryRow(ctx, `SELECT (SELECT count(*) FROM webhook_events WHERE business_event_id = $1), (SELECT count(*) FROM webhook_attempts WHERE delivery_id = $2), (SELECT count(*) FROM webhook_cycles WHERE delivery_id = $2), body FROM webhook_events WHERE business_event_id = $1`, prepared.Acceptance.BusinessEventID, prepared.Destinations[0].DeliveryID).Scan(&events, &attempts, &cycles, &body); err != nil {
		t.Fatal(err)
	}
	if events != 1 || attempts != 0 || cycles != 0 || body != nil {
		t.Fatalf("separate retention = events:%d attempts:%d cycles:%d body:%v", events, attempts, cycles, body)
	}
	if _, err := pool.PGX().Exec(ctx, `UPDATE webhook_deliveries SET terminal_summary_retained_until = clock_timestamp() - interval '1 second', action_retained_until = clock_timestamp() - interval '1 second', destination_generation_retained_until = clock_timestamp() - interval '1 second', receiver_dedup_retained_until = clock_timestamp() - interval '1 second' WHERE delivery_id = $1`, prepared.Destinations[0].DeliveryID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.PGX().Exec(ctx, `UPDATE webhook_operator_actions SET retain_until = clock_timestamp() - interval '1 second' WHERE owner_scope = 'owner-a'`); err != nil {
		t.Fatal(err)
	}
	if deleted, err := store.CleanupRetention(ctx, 10); err != nil || deleted < 1 {
		t.Fatalf("CleanupRetention(final evidence) = %d, %v", deleted, err)
	}
	retire := postgreswebhook.ActionRequest{OwnerScope: "owner-a", Actor: "operator-a", ActionID: "retire-a", Kind: postgreswebhook.ActionDestinationState, TargetKind: "destination", TargetID: "dest-a", TargetGeneration: 1, ExpectedRevision: 1, Reason: "retire", Payload: &postgreswebhook.DestinationStateAction{Disposition: "retired"}}
	if receipt, err := store.ApplyAction(ctx, retire, manifest); err != nil || receipt.Result != "applied" {
		t.Fatalf("ApplyAction(retire) = %+v, %v", receipt, err)
	}
	if _, err := pool.PGX().Exec(ctx, `UPDATE webhook_operator_actions SET retain_until = clock_timestamp() - interval '1 second' WHERE action_id = $1`, retire.ActionID); err != nil {
		t.Fatal(err)
	}
	if deleted, err := store.CleanupRetention(ctx, 10); err != nil || deleted != 2 {
		t.Fatalf("CleanupRetention(retired destination) = %d, %v", deleted, err)
	}
	var destinations, tombstones int
	if err := pool.PGX().QueryRow(ctx, `SELECT (SELECT count(*) FROM webhook_destinations WHERE destination_id = 'dest-a'), (SELECT count(*) FROM webhook_destination_tombstones WHERE destination_id = 'dest-a')`).Scan(&destinations, &tombstones); err != nil {
		t.Fatal(err)
	}
	if destinations != 0 || tombstones != 1 {
		t.Fatalf("retired destination authority = destinations:%d tombstones:%d", destinations, tombstones)
	}
	resurrection := webhookPrepared(t, "retention-resurrection")
	err = pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error { _, err := store.Accept(ctx, tx, resurrection); return err })
	if !errors.Is(err, postgreswebhook.ErrConflict) {
		t.Fatalf("Accept(retired generation) error = %v", err)
	}
}

func TestPostgresWebhookMixedVersionRetentionBackfill(t *testing.T) {
	ctx, pool, store, manifest := newPostgresWebhookFixture(t)
	prepared := webhookPrepared(t, "retention-backfill")
	if err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error { _, err := store.Accept(ctx, tx, prepared); return err }); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.PGX().Exec(ctx, `UPDATE webhook_deliveries SET
		payload_retained_until = DEFAULT, active_retained_until = DEFAULT,
		terminal_summary_retained_until = DEFAULT, attempt_retained_until = DEFAULT,
		action_retained_until = DEFAULT, destination_generation_retained_until = DEFAULT,
		receiver_dedup_retained_until = DEFAULT WHERE delivery_id = $1`, prepared.Destinations[0].DeliveryID); err != nil {
		t.Fatal(err)
	}
	observation, err := store.ObserveReadiness(ctx, manifest)
	if !errors.Is(err, postgreswebhook.ErrConfig) || observation.RetentionBackfill != 1 {
		t.Fatalf("ObserveReadiness(backfill) = %+v, %v", observation, err)
	}
	if updated, err := store.CleanupRetention(ctx, 10); err != nil || updated != 1 {
		t.Fatalf("CleanupRetention(backfill) = %d, %v", updated, err)
	}
	if observation, err := store.ObserveReadiness(ctx, manifest); err != nil || observation.RetentionBackfill != 0 {
		t.Fatalf("ObserveReadiness(backfilled) = %+v, %v", observation, err)
	}
	var finite bool
	if err := pool.PGX().QueryRow(ctx, `SELECT isfinite(payload_retained_until) AND isfinite(active_retained_until)
		AND isfinite(terminal_summary_retained_until) AND isfinite(attempt_retained_until)
		AND isfinite(action_retained_until) AND isfinite(destination_generation_retained_until)
		AND isfinite(receiver_dedup_retained_until) FROM webhook_deliveries WHERE delivery_id = $1`, prepared.Destinations[0].DeliveryID).Scan(&finite); err != nil || !finite {
		t.Fatalf("finite retention horizons = %t, %v", finite, err)
	}
}
