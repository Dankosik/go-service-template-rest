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

func TestPostgresWebhookSchema(t *testing.T) {
	ctx, _, store, _ := newPostgresWebhookFixture(t)
	if err := store.CheckSchema(ctx); err != nil {
		t.Fatalf("CheckSchema(): %v", err)
	}
}

func TestPostgresWebhookClockHighWater(t *testing.T) {
	ctx, pool, store, manifest := newPostgresWebhookFixture(t)
	if _, err := pool.PGX().Exec(ctx, `UPDATE webhook_clock SET high_water = clock_timestamp() + interval '1 hour', regression = false`); err != nil {
		t.Fatal(err)
	}
	observation, err := store.ObserveReadiness(ctx, manifest)
	if !errors.Is(err, postgreswebhook.ErrClockRegression) || !observation.ClockRegression {
		t.Fatalf("ObserveReadiness(regression) = %+v, %v", observation, err)
	}
	var regression bool
	if err := pool.PGX().QueryRow(ctx, `SELECT regression FROM webhook_clock WHERE singleton`).Scan(&regression); err != nil || !regression {
		t.Fatalf("durable regression = %v, %v", regression, err)
	}
	if _, err := pool.PGX().Exec(ctx, `UPDATE webhook_clock SET high_water = clock_timestamp(), regression = false`); err != nil {
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
		if err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
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
	if err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error { _, err := store.Accept(ctx, tx, prepared); return err }); err != nil {
		t.Fatal(err)
	}
	claim, err := store.Claim(ctx, "worker-a", 2, 30*time.Second, manifest)
	if err != nil || claim.Attempt == nil {
		t.Fatalf("Claim() = %+v, %v", claim, err)
	}
	higher, err := postgreswebhook.NewStore(pool, postgreswebhook.StoreOptions{OperationTimeout: 3 * time.Second, CapacityRevision: 2, GlobalConcurrency: 1, ManifestRevision: 1})
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
	if err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error { _, err := store.Accept(ctx, tx, prepared); return err }); err != nil {
		t.Fatal(err)
	}
	claim, err := store.Claim(ctx, "worker-a", 1, 30*time.Second, manifest)
	if err != nil || claim.Attempt == nil {
		t.Fatalf("Claim() = %+v, %v", claim, err)
	}
	action := postgreswebhook.ActionRequest{OwnerScope: "owner-a", Actor: "actor-a", ActionID: "disable-a", Kind: postgreswebhook.ActionDestinationState, TargetKind: "destination", TargetID: "dest-a", TargetGeneration: 1, Expected: "1", Reason: "admin_disable", Values: []string{"disabled", ""}}
	if receipt, err := store.ApplyAction(ctx, action); err != nil || receipt.Result != "applied" {
		t.Fatalf("ApplyAction() = %+v, %v", receipt, err)
	}
	evidence := postgreswebhook.AuthorizationEvidence{KeyReference: "key-a", SelectedAddress: netip.MustParseAddr("8.8.8.8")}
	if err := store.AuthorizeAttempt(ctx, *claim.Attempt, manifest, evidence); !errors.Is(err, postgreswebhook.ErrStaleAttempt) {
		t.Fatalf("AuthorizeAttempt(disabled) error = %v", err)
	}
}

func TestPostgresWebhookOwnerIsolation(t *testing.T) {
	ctx, pool, _, _ := newPostgresWebhookFixture(t)
	var count int
	if err := pool.PGX().QueryRow(ctx, `SELECT count(*) FROM webhook_destinations WHERE owner_scope = $1`, "owner-b").Scan(&count); err != nil || count != 0 {
		t.Fatalf("cross-owner count = %d, %v", count, err)
	}
}
