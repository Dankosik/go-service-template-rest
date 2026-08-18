//go:build integration

package integration_test

import (
	"errors"
	"net/netip"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgreswebhook"
	"github.com/jackc/pgx/v5"
)

func TestPostgresWebhookOperatorInspection(t *testing.T) {
	ctx, pool, store, manifest := newPostgresWebhookFixture(t)
	prepared := webhookPrepared(t, "inspection")
	if err := postgres.InTx(ctx, pool, pgx.TxOptions{}, func(tx pgx.Tx) error { _, err := store.Accept(ctx, tx, prepared); return err }); err != nil {
		t.Fatal(err)
	}
	for i, enabled := range []bool{true, false} {
		action := postgreswebhook.ActionRequest{OwnerScope: "owner-a", Actor: "operator-a", ActionID: []string{"hold-inspection", "release-inspection"}[i], Kind: postgreswebhook.ActionRetentionHold, TargetKind: "delivery", TargetID: prepared.Destinations[0].DeliveryID, Reason: "inspection-proof", Payload: &postgreswebhook.RetentionHoldAction{Enabled: enabled}}
		if receipt, err := store.ApplyAction(ctx, action, manifest); err != nil || receipt.Result != "applied" {
			t.Fatalf("ApplyAction(%d) = %+v, %v", i, receipt, err)
		}
	}
	claim, err := store.Claim(ctx, "worker-inspection", 1, 30*time.Second, manifest)
	if err != nil || claim.Attempt == nil {
		t.Fatalf("Claim() = %+v, %v", claim, err)
	}
	if err := store.AuthorizeAttempt(ctx, *claim.Attempt, manifest, postgreswebhook.AuthorizationEvidence{KeyReference: "key-a", KeyReferences: []string{"key-a"}, SelectedAddress: netip.MustParseAddr("8.8.8.8")}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.FinalizeAttempt(ctx, *claim.Attempt, postgreswebhook.Finalization{Evidence: postgreswebhook.TransportEvidence{StatusCode: 204, MayHaveSent: true}}); err != nil {
		t.Fatal(err)
	}
	request := postgreswebhook.InspectionRequest{OwnerScope: "owner-a", DeliveryID: prepared.Destinations[0].DeliveryID, PageSize: 1}
	first, err := store.InspectDelivery(ctx, request)
	if err != nil || first.AcceptanceID != prepared.Acceptance.AcceptanceID || first.BusinessEventID != prepared.Acceptance.BusinessEventID || first.FanoutSnapshotID != prepared.Acceptance.FanoutSnapshotID || len(first.Cycles) != 1 || len(first.Attempts) != 1 || len(first.Actions) != 1 || !first.MoreActions || first.CumulativeSummary != postgreswebhook.OutcomeHTTPAccepted {
		t.Fatalf("InspectDelivery(first) = %+v, %v", first, err)
	}
	request.Cursor = first.Next
	second, err := store.InspectDelivery(ctx, request)
	if err != nil || len(second.Cycles) != 0 || len(second.Attempts) != 0 || len(second.Actions) != 1 || second.MoreActions || second.Actions[0].ActionID == first.Actions[0].ActionID {
		t.Fatalf("InspectDelivery(second) = %+v, %v", second, err)
	}
	if _, err := store.InspectDelivery(ctx, postgreswebhook.InspectionRequest{OwnerScope: "owner-a", DeliveryID: "missing-delivery", PageSize: 1}); !errors.Is(err, postgreswebhook.ErrNotFound) {
		t.Fatalf("InspectDelivery(missing) error = %v", err)
	}
}

func TestPostgresWebhookNotFoundActionIdentityIsReplayable(t *testing.T) {
	ctx, pool, store, manifest := newPostgresWebhookFixture(t)
	action := postgreswebhook.ActionRequest{OwnerScope: "owner-a", Actor: "operator-a", ActionID: "redrive-missing", Kind: postgreswebhook.ActionRedrive, TargetKind: "delivery", TargetID: "missing-delivery", Reason: "operator-check", Payload: &postgreswebhook.RedriveAction{MaximumAttempts: 1, MaximumAge: time.Minute, AcknowledgeDuplicateRisk: true}}
	first, err := store.ApplyAction(ctx, action, manifest)
	if err != nil || first.Result != "not_found" || first.Replay {
		t.Fatalf("ApplyAction(first) = %+v, %v", first, err)
	}
	replay, err := store.ApplyAction(ctx, action, manifest)
	if err != nil || replay.Result != "not_found" || !replay.Replay {
		t.Fatalf("ApplyAction(replay) = %+v, %v", replay, err)
	}
	var count int
	var finite bool
	if err := pool.QueryRow(ctx, `SELECT count(*), bool_and(isfinite(retain_until)) FROM webhook_operator_actions WHERE owner_scope = 'owner-a' AND action_id = $1`, action.ActionID).Scan(&count, &finite); err != nil || count != 1 || !finite {
		t.Fatalf("not-found action ledger = count:%d finite:%t err:%v", count, finite, err)
	}
	missingDestination := postgreswebhook.ActionRequest{OwnerScope: "owner-a", Actor: "operator-a", ActionID: "disable-missing", Kind: postgreswebhook.ActionDestinationState, TargetKind: "destination", TargetID: "missing-destination", TargetGeneration: 1, ExpectedRevision: 1, Reason: "operator-check", Payload: &postgreswebhook.DestinationStateAction{Disposition: "disabled"}}
	first, err = store.ApplyAction(ctx, missingDestination, manifest)
	if err != nil || first.Result != "state_conflict" || first.Replay {
		t.Fatalf("ApplyAction(missing destination) = %+v, %v", first, err)
	}
	replay, err = store.ApplyAction(ctx, missingDestination, manifest)
	if err != nil || replay.Result != "state_conflict" || !replay.Replay {
		t.Fatalf("ApplyAction(missing destination replay) = %+v, %v", replay, err)
	}
}
