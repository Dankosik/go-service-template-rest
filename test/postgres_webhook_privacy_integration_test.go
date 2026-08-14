//go:build integration

package integration_test

import (
	"errors"
	"testing"

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

func TestPostgresWebhookNamespaceRetirement(t *testing.T) {
	ctx, pool, store, _ := newPostgresWebhookFixture(t)
	prepared := webhookPrepared(t, "namespace")
	if err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error { _, err := store.Accept(ctx, tx, prepared); return err }); err != nil {
		t.Fatal(err)
	}
	action := postgreswebhook.ActionRequest{OwnerScope: "owner-a", Actor: "privacy-a", ActionID: "namespace-a", Kind: postgreswebhook.ActionNamespaceRetire, TargetKind: "namespace", TargetID: "owner-a", Expected: "", Reason: "privacy_request", Values: []string{"full_erasure", "privacy-ticket-a"}}
	receipt, err := store.RequestNamespaceRetirement(ctx, action, 10)
	if err != nil || receipt.Result != "completed" {
		t.Fatalf("RequestNamespaceRetirement() = %+v, %v", receipt, err)
	}
	err = pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error { _, err := store.Accept(ctx, tx, prepared); return err })
	if !errors.Is(err, postgreswebhook.ErrPrivacyDeleted) {
		t.Fatalf("Accept(retired namespace) error = %v", err)
	}
}
