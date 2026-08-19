//go:build integration

package integration_test

import (
	"encoding/json"
	"errors"
	"net/netip"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgreswebhook"
	"github.com/jackc/pgx/v5"
)

func TestPostgresWebhookSchema(t *testing.T) {
	ctx, pool, store, _ := newPostgresWebhookFixture(t)
	if err := store.CheckSchema(ctx); err != nil {
		t.Fatalf("CheckSchema(): %v", err)
	}
	policy, err := json.Marshal(webhookPrepared(t, "schema").Destinations[0].Policy)
	if err != nil {
		t.Fatal(err)
	}
	var valid bool
	if err := pool.QueryRow(ctx, `SELECT webhook_delivery_policy_valid($1::jsonb)`, policy).Scan(&valid); err != nil || !valid {
		t.Fatalf("webhook_delivery_policy_valid(%s) = %t, %v", policy, valid, err)
	}
	if err := pool.QueryRow(ctx, `SELECT webhook_delivery_policy_valid(jsonb_set($1::jsonb, '{automatic_pause}', 'true'))`, policy).Scan(&valid); err != nil || valid {
		t.Fatalf("webhook_delivery_policy_valid(automatic pause) = %t, %v", valid, err)
	}
	for name, query := range map[string]string{
		"unknown key":       `SELECT webhook_delivery_policy_valid($1::jsonb || '{"future_policy":1}'::jsonb)`,
		"duplicate content": `SELECT webhook_delivery_policy_valid(jsonb_set($1::jsonb, '{accepted_content_types}', '["application/json","APPLICATION/JSON"]'::jsonb))`,
		"invalid schema":    `SELECT webhook_delivery_policy_valid(jsonb_set($1::jsonb, '{accepted_business_schemas}', '[1]'::jsonb))`,
		"short retention":   `SELECT webhook_delivery_policy_valid(jsonb_set($1::jsonb, '{horizons,0}', '1'::jsonb))`,
		"equal drain":       `SELECT webhook_delivery_policy_valid(jsonb_set($1::jsonb, '{drain_timeout}', $1::jsonb->'attempt_timeout'))`,
	} {
		if err := pool.QueryRow(ctx, query, policy).Scan(&valid); err != nil || valid {
			t.Fatalf("webhook_delivery_policy_valid(%s) = %t, %v", name, valid, err)
		}
	}
	for _, index := range []string{"webhook_deliveries_event_idx", "webhook_operator_actions_target_idx"} {
		var exists bool
		if err := pool.QueryRow(ctx, `SELECT to_regclass($1) IS NOT NULL`, index).Scan(&exists); err != nil || !exists {
			t.Fatalf("index %s exists = %t, %v", index, exists, err)
		}
	}
}

func TestPostgresWebhookClockHighWater(t *testing.T) {
	ctx, pool, store, manifest := newPostgresWebhookFixture(t)
	if _, err := pool.Exec(ctx, `UPDATE webhook_clock SET high_water = clock_timestamp() + interval '1 hour', regression = false`); err != nil {
		t.Fatal(err)
	}
	observation, err := store.ObserveReadiness(ctx, manifest)
	if !errors.Is(err, postgreswebhook.ErrClockRegression) || !observation.ClockRegression {
		t.Fatalf("ObserveReadiness(regression) = %+v, %v", observation, err)
	}
	var regression bool
	if err := pool.QueryRow(ctx, `SELECT regression FROM webhook_clock WHERE singleton`).Scan(&regression); err != nil || !regression {
		t.Fatalf("durable regression = %v, %v", regression, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE webhook_clock SET high_water = clock_timestamp(), regression = false`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ObserveReadiness(ctx, manifest); err != nil {
		t.Fatalf("ObserveReadiness(recovered): %v", err)
	}
}

func TestPostgresWebhookBoundedFairness(t *testing.T) {
	ctx, pool, store, manifest := newPostgresWebhookFixture(t)
	for _, suffix := range []string{"fair-a", "fair-b"} {
		prepared := webhookPrepared(t, suffix)
		if err := postgres.InTx(ctx, pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
			_, err := store.Accept(ctx, tx, prepared)
			return err
		}); err != nil {
			t.Fatal(err)
		}
	}
	first, err := store.Claim(ctx, "worker-a", 1, 30*time.Second, manifest)
	if err != nil || first.Attempt == nil {
		t.Fatalf("Claim(first) = %+v, %v", first, err)
	}
	blocked, err := store.Claim(ctx, "worker-b", 1, 30*time.Second, manifest)
	if err != nil || blocked.Attempt != nil || !blocked.Progress {
		t.Fatalf("Claim(saturated) = %+v, %v", blocked, err)
	}
	if _, err := store.FinalizeAttempt(ctx, *first.Attempt, postgreswebhook.Finalization{Evidence: postgreswebhook.TransportEvidence{DefinitelyNotSent: true}, LocalRetryDelay: time.Second}); err != nil {
		t.Fatalf("FinalizeAttempt(): %v", err)
	}
	second, err := store.Claim(ctx, "worker-b", 1, 30*time.Second, manifest)
	if err != nil || second.Attempt == nil || second.Attempt.Identity.DeliveryID == first.Attempt.Identity.DeliveryID {
		t.Fatalf("Claim(second destination work) = %+v, %v", second, err)
	}
}

func TestPostgresWebhookReplicaCapacityAndSecretRevision(t *testing.T) {
	ctx, pool, store, manifest := newPostgresWebhookFixture(t)
	prepared := webhookPrepared(t, "capacity")
	if err := postgres.InTx(ctx, pool, pgx.TxOptions{}, func(tx pgx.Tx) error { _, err := store.Accept(ctx, tx, prepared); return err }); err != nil {
		t.Fatal(err)
	}
	incompatible, err := postgreswebhook.NewStore(pool, postgreswebhook.StoreOptions{
		OperationTimeout: 3 * time.Second, CapacityRevision: 1, GlobalConcurrency: 2, ManifestRevision: 1,
		AttemptTimeout: time.Second, ResponseHeaderTimeout: time.Second, ResponseHeaderBytes: 4096,
		ResponseBodyBytes: 4096, DrainTimeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := incompatible.ObserveReadiness(ctx, manifest); err == nil {
		t.Fatal("incompatible retained policy opened readiness")
	}
	claim, err := store.Claim(ctx, "worker-a", 2, 30*time.Second, manifest)
	if err != nil || claim.Attempt == nil {
		t.Fatalf("Claim() = %+v, %v", claim, err)
	}
	higher, err := postgreswebhook.NewStore(pool, postgreswebhook.StoreOptions{
		OperationTimeout: 3 * time.Second, CapacityRevision: 2, GlobalConcurrency: 1, ManifestRevision: 1,
		AttemptTimeout: 5 * time.Second, ResponseHeaderTimeout: 2 * time.Second, ResponseHeaderBytes: 4096,
		ResponseBodyBytes: 4096, DrainTimeout: 10 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := higher.InitializeOrTransitionCapacity(ctx); err == nil {
		t.Fatal("live-slot capacity transition succeeded")
	}
	if _, err := store.FinalizeAttempt(ctx, *claim.Attempt, postgreswebhook.Finalization{Evidence: postgreswebhook.TransportEvidence{DefinitelyNotSent: true}, LocalRetryDelay: time.Second}); err != nil {
		t.Fatal(err)
	}
	if err := higher.InitializeOrTransitionCapacity(ctx); err != nil {
		t.Fatalf("drained capacity transition: %v", err)
	}
	if _, err := store.Claim(ctx, "stale-worker", 1, 30*time.Second, manifest); err == nil {
		t.Fatal("stale capacity revision claimed work")
	}
}

func TestPostgresWebhookDestinationControlBarrier(t *testing.T) {
	ctx, pool, store, manifest := newPostgresWebhookFixture(t)
	prepared := webhookPrepared(t, "barrier")
	if err := postgres.InTx(ctx, pool, pgx.TxOptions{}, func(tx pgx.Tx) error { _, err := store.Accept(ctx, tx, prepared); return err }); err != nil {
		t.Fatal(err)
	}
	claim, err := store.Claim(ctx, "worker-a", 1, 30*time.Second, manifest)
	if err != nil || claim.Attempt == nil {
		t.Fatalf("Claim() = %+v, %v", claim, err)
	}
	action := postgreswebhook.ActionRequest{OwnerScope: "owner-a", Actor: "actor-a", ActionID: "disable-a", Kind: postgreswebhook.ActionDestinationState, TargetKind: "destination", TargetID: "dest-a", TargetGeneration: 1, ExpectedRevision: 1, Reason: "admin_disable", Payload: &postgreswebhook.DestinationStateAction{Disposition: "disabled"}}
	if receipt, err := store.ApplyAction(ctx, action, manifest); err != nil || receipt.Result != "applied" {
		t.Fatalf("ApplyAction() = %+v, %v", receipt, err)
	}
	evidence := postgreswebhook.AuthorizationEvidence{KeyReference: "key-a", KeyReferences: []string{"key-a"}, SelectedAddress: netip.MustParseAddr("8.8.8.8")}
	if err := store.AuthorizeAttempt(ctx, *claim.Attempt, manifest, evidence); !errors.Is(err, postgreswebhook.ErrStaleAttempt) {
		t.Fatalf("AuthorizeAttempt(disabled) error = %v", err)
	}
}

func TestPostgresWebhookAttemptLeaseAndCapacityFences(t *testing.T) {
	ctx, pool, store, manifest := newPostgresWebhookFixture(t)
	prepared := webhookPrepared(t, "attempt-fences")
	if err := postgres.InTx(ctx, pool, pgx.TxOptions{}, func(tx pgx.Tx) error { _, err := store.Accept(ctx, tx, prepared); return err }); err != nil {
		t.Fatal(err)
	}
	claim, err := store.Claim(ctx, "worker-fences", 1, 30*time.Second, manifest)
	if err != nil || claim.Attempt == nil {
		t.Fatalf("Claim() = %+v, %v", claim, err)
	}
	evidence := postgreswebhook.AuthorizationEvidence{KeyReference: "key-a", KeyReferences: []string{"key-a"}, SelectedAddress: netip.MustParseAddr("8.8.8.8")}
	wrongRevision := *claim.Attempt
	wrongRevision.CapacityRevision++
	if err := store.AuthorizeAttempt(ctx, wrongRevision, manifest, evidence); !errors.Is(err, postgreswebhook.ErrConfig) {
		t.Fatalf("AuthorizeAttempt(wrong capacity) error = %v", err)
	}
	if _, err := store.FinalizeAttempt(ctx, wrongRevision, postgreswebhook.Finalization{Evidence: postgreswebhook.TransportEvidence{DefinitelyNotSent: true}}); !errors.Is(err, postgreswebhook.ErrConfig) {
		t.Fatalf("FinalizeAttempt(wrong capacity) error = %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE webhook_capacity_slots SET attempt_id = 'replacement-attempt' WHERE attempt_id = $1`, claim.Attempt.Identity.AttemptID); err != nil {
		t.Fatal(err)
	}
	if err := store.AuthorizeAttempt(ctx, *claim.Attempt, manifest, evidence); !errors.Is(err, postgreswebhook.ErrStaleAttempt) {
		t.Fatalf("AuthorizeAttempt(reused slot) error = %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE webhook_capacity_slots SET attempt_id = $1 WHERE attempt_id = 'replacement-attempt'`, claim.Attempt.Identity.AttemptID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE webhook_attempts SET lease_expires_at = clock_timestamp() - interval '1 second' WHERE attempt_id = $1`, claim.Attempt.Identity.AttemptID); err != nil {
		t.Fatal(err)
	}
	if err := store.AuthorizeAttempt(ctx, *claim.Attempt, manifest, evidence); !errors.Is(err, postgreswebhook.ErrStaleAttempt) {
		t.Fatalf("AuthorizeAttempt(expired lease) error = %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE webhook_attempts SET lease_expires_at = clock_timestamp() + interval '30 seconds' WHERE attempt_id = $1`, claim.Attempt.Identity.AttemptID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE webhook_cycles SET accepted_at = clock_timestamp() - interval '2 seconds', deadline_at = clock_timestamp() - interval '1 second' WHERE delivery_id = $1`, claim.Attempt.Identity.DeliveryID); err != nil {
		t.Fatal(err)
	}
	if err := store.AuthorizeAttempt(ctx, *claim.Attempt, manifest, evidence); !errors.Is(err, postgreswebhook.ErrStaleAttempt) {
		t.Fatalf("AuthorizeAttempt(expired cycle) error = %v", err)
	}
}

func TestPostgresWebhookDatabaseRejectsInvalidPolicyAndAuthorizationEvidence(t *testing.T) {
	ctx, pool, store, manifest := newPostgresWebhookFixture(t)
	prepared := webhookPrepared(t, "database-constraints")
	if err := postgres.InTx(ctx, pool, pgx.TxOptions{}, func(tx pgx.Tx) error { _, err := store.Accept(ctx, tx, prepared); return err }); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE webhook_destinations SET policy = '{}'::jsonb WHERE destination_id = 'dest-a'`); err == nil {
		t.Fatal("database accepted an invalid destination policy")
	}
	claim, err := store.Claim(ctx, "worker-constraints", 1, 30*time.Second, manifest)
	if err != nil || claim.Attempt == nil {
		t.Fatalf("Claim() = %+v, %v", claim, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE webhook_attempts SET send_authorized = true, may_have_sent = true WHERE attempt_id = $1`, claim.Attempt.Identity.AttemptID); err == nil {
		t.Fatal("database accepted send authorization without evidence")
	}
}

func TestPostgresWebhookOwnerIsolation(t *testing.T) {
	ctx, pool, _, _ := newPostgresWebhookFixture(t)
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM webhook_destinations WHERE owner_scope = $1`, "owner-b").Scan(&count); err != nil || count != 0 {
		t.Fatalf("cross-owner count = %d, %v", count, err)
	}
}

func TestPostgresWebhookRotationAttemptEvidence(t *testing.T) {
	ctx, pool, store, _ := newPostgresWebhookFixture(t)
	prepared := webhookPrepared(t, "rotation-evidence")
	if err := postgres.InTx(ctx, pool, pgx.TxOptions{}, func(tx pgx.Tx) error { _, err := store.Accept(ctx, tx, prepared); return err }); err != nil {
		t.Fatal(err)
	}
	current, err := postgreswebhook.NewStore(pool, postgreswebhook.StoreOptions{
		OperationTimeout: 3 * time.Second, CapacityRevision: 1, GlobalConcurrency: 2, ManifestRevision: 2,
		AttemptTimeout: 5 * time.Second, ResponseHeaderTimeout: 2 * time.Second, ResponseHeaderBytes: 4096,
		ResponseBodyBytes: 4096, DrainTimeout: 10 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	rotation := postgreswebhook.ActionRequest{
		OwnerScope: "owner-a", Actor: "security-a", ActionID: "rotation-evidence", Kind: postgreswebhook.ActionKeyRotation,
		TargetKind: "destination", TargetID: "dest-a", TargetGeneration: 1, ExpectedRevision: 1, Reason: "rotate",
		Payload: &postgreswebhook.KeyRotationAction{SecretRevision: 2, KeyRevision: 2, ActiveKeyReference: "key-new", PredecessorReference: "key-a", OverlapStartsAt: now.Add(-time.Second), PredecessorValidUntil: now.Add(time.Hour), AuthorityReceipt: "stage-receipt-2"},
	}
	manifest := webhookRotationManifest(t)
	if receipt, err := current.ApplyAction(ctx, rotation, manifest); err != nil || receipt.Result != "applied" {
		t.Fatalf("ApplyAction(rotation) = %+v, %v", receipt, err)
	}
	claim, err := current.Claim(ctx, "worker-rotation", 1, 30*time.Second, manifest)
	if err != nil || claim.Attempt == nil || claim.Attempt.KeyReference != "key-new" || claim.Attempt.PredecessorReference != "key-a" {
		t.Fatalf("Claim(rotation) = %+v, %v", claim, err)
	}
	evidence := postgreswebhook.AuthorizationEvidence{KeyReference: "key-new", KeyReferences: []string{"key-new", "key-a"}, SelectedAddress: netip.MustParseAddr("8.8.8.8")}
	if err := current.AuthorizeAttempt(ctx, *claim.Attempt, manifest, evidence); err != nil {
		t.Fatal(err)
	}
	var references []string
	if err := pool.QueryRow(ctx, `SELECT key_references FROM webhook_attempts WHERE attempt_id = $1`, claim.Attempt.Identity.AttemptID).Scan(&references); err != nil {
		t.Fatal(err)
	}
	if len(references) != 2 || references[0] != "key-new" || references[1] != "key-a" {
		t.Fatalf("attempt key references = %v", references)
	}
}
