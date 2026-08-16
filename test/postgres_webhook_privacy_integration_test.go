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

func TestPostgresWebhookEventPrivacyDeletion(t *testing.T) {
	ctx, pool, store, _ := newPostgresWebhookFixture(t)
	prepared := webhookPrepared(t, "privacy")
	if err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error { _, err := store.Accept(ctx, tx, prepared); return err }); err != nil {
		t.Fatal(err)
	}
	action := postgreswebhook.ActionRequest{OwnerScope: "owner-a", Actor: "privacy-a", ActionID: "privacy-a", Kind: postgreswebhook.ActionPrivacyDelete, TargetKind: "event", TargetID: prepared.Acceptance.BusinessEventID, Expected: "1", Reason: "privacy_request", Values: []string{"event", prepared.Acceptance.BusinessEventID, "minimal_tombstone", "privacy-ticket-a"}}
	receipt, err := store.RequestEventPrivacyDeletion(ctx, action)
	if err != nil || receipt.Result != "applied" {
		t.Fatalf("RequestEventPrivacyDeletion() = %+v, %v", receipt, err)
	}
	readback, err := store.ResolveAcceptance(ctx, prepared)
	if !errors.Is(err, postgreswebhook.ErrPrivacyDeleted) || readback.Disposition != postgreswebhook.AcceptancePrivacyDeleted {
		t.Fatalf("ResolveAcceptance(deleted) = %+v, %v", readback, err)
	}
	replay, err := store.RequestEventPrivacyDeletion(ctx, action)
	if err != nil || !replay.Replay || replay.Result != "applied" {
		t.Fatalf("RequestEventPrivacyDeletion(replay) = %+v, %v", replay, err)
	}
}

func TestPostgresWebhookPrivacyDeletionPreservesAuthorizedSendAmbiguity(t *testing.T) {
	ctx, pool, store, manifest := newPostgresWebhookFixture(t)
	prepared := webhookPrepared(t, "privacy-authorized")
	if err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error { _, err := store.Accept(ctx, tx, prepared); return err }); err != nil {
		t.Fatal(err)
	}
	claim, err := store.Claim(ctx, "worker-a", 1, 30*time.Second, manifest)
	if err != nil || claim.Attempt == nil {
		t.Fatalf("Claim() = %+v, %v", claim, err)
	}
	if err := store.AuthorizeAttempt(ctx, *claim.Attempt, manifest, postgreswebhook.AuthorizationEvidence{KeyReference: "key-a", SelectedAddress: netip.MustParseAddr("8.8.8.8")}); err != nil {
		t.Fatal(err)
	}
	action := postgreswebhook.ActionRequest{OwnerScope: "owner-a", Actor: "privacy-a", ActionID: "privacy-authorized", Kind: postgreswebhook.ActionPrivacyDelete, TargetKind: "event", TargetID: prepared.Acceptance.BusinessEventID, Expected: "1", Reason: "privacy_request", Values: []string{"event", prepared.Acceptance.BusinessEventID, "minimal_tombstone", "privacy-ticket-a"}}
	if receipt, err := store.RequestEventPrivacyDeletion(ctx, action); err != nil || receipt.Result != "applied" {
		t.Fatalf("RequestEventPrivacyDeletion() = %+v, %v", receipt, err)
	}
	var summary string
	if err := pool.PGX().QueryRow(ctx, `SELECT last_semantic_class FROM webhook_tombstones WHERE owner_scope = 'owner-a' AND target_kind = 'event' AND target_id = $1`, prepared.Acceptance.BusinessEventID).Scan(&summary); err != nil || summary != "outcome_unknown" {
		t.Fatalf("tombstone summary = %q, %v", summary, err)
	}
	if _, err := store.FinalizeAttempt(ctx, *claim.Attempt, postgreswebhook.Finalization{Evidence: postgreswebhook.TransportEvidence{MayHaveSent: true}}); !errors.Is(err, postgreswebhook.ErrStaleAttempt) {
		t.Fatalf("FinalizeAttempt() error = %v", err)
	}
}

func TestPostgresWebhookNamespaceRetirement(t *testing.T) {
	ctx, pool, store, manifest := newPostgresWebhookFixture(t)
	prepared := webhookPrepared(t, "namespace")
	if err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error { _, err := store.Accept(ctx, tx, prepared); return err }); err != nil {
		t.Fatal(err)
	}
	claim, err := store.Claim(ctx, "worker-a", 1, 30*time.Second, manifest)
	if err != nil || claim.Attempt == nil {
		t.Fatalf("Claim() = %+v, %v", claim, err)
	}
	if err := store.AuthorizeAttempt(ctx, *claim.Attempt, manifest, postgreswebhook.AuthorizationEvidence{KeyReference: "key-a", SelectedAddress: netip.MustParseAddr("8.8.8.8")}); err != nil {
		t.Fatal(err)
	}
	action := postgreswebhook.ActionRequest{OwnerScope: "owner-a", Actor: "privacy-a", ActionID: "namespace-a", Kind: postgreswebhook.ActionNamespaceRetire, TargetKind: "namespace", TargetID: "owner-a", Expected: "", Reason: "privacy_request", Values: []string{"full_erasure", "privacy-ticket-a"}}
	receipt, err := store.RequestNamespaceRetirement(ctx, action, 10)
	if err != nil || receipt.Result != "completed" {
		t.Fatalf("RequestNamespaceRetirement() = %+v, %v", receipt, err)
	}
	var summary string
	if err := pool.PGX().QueryRow(ctx, `SELECT last_semantic_class FROM webhook_tombstones WHERE owner_scope = 'owner-a' AND target_kind = 'namespace'`).Scan(&summary); err != nil || summary != "outcome_unknown" {
		t.Fatalf("namespace tombstone summary = %q, %v", summary, err)
	}
	err = pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error { _, err := store.Accept(ctx, tx, prepared); return err })
	if !errors.Is(err, postgreswebhook.ErrPrivacyDeleted) {
		t.Fatalf("Accept(retired namespace) error = %v", err)
	}
}
