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
	ctx, pool, store, manifest := newPostgresWebhookFixture(t)
	prepared := webhookPrepared(t, "privacy")
	if err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error { _, err := store.Accept(ctx, tx, prepared); return err }); err != nil {
		t.Fatal(err)
	}
	hold := postgreswebhook.ActionRequest{OwnerScope: "owner-a", Actor: "legal-a", ActionID: "hold-a", Kind: postgreswebhook.ActionRetentionHold, TargetKind: "delivery", TargetID: prepared.Destinations[0].DeliveryID, ExpectedRevision: 0, Reason: "legal_hold", Payload: &postgreswebhook.RetentionHoldAction{Enabled: true}}
	if receipt, err := store.ApplyAction(ctx, hold, manifest); err != nil || receipt.Result != "applied" {
		t.Fatalf("ApplyAction(retention hold) = %+v, %v", receipt, err)
	}
	action := postgreswebhook.ActionRequest{OwnerScope: "owner-a", Actor: "privacy-a", ActionID: "privacy-a", Kind: postgreswebhook.ActionPrivacyDelete, TargetKind: "event", TargetID: prepared.Acceptance.BusinessEventID, Reason: "privacy_request", Payload: &postgreswebhook.PrivacyDeletionAction{TargetKind: "event", TargetID: prepared.Acceptance.BusinessEventID, Mode: "minimal_tombstone", DeletionAuthority: "privacy-ticket-a"}}
	receipt, err := store.RequestEventPrivacyDeletion(ctx, action)
	if err != nil || receipt.Result != "applied" {
		t.Fatalf("RequestEventPrivacyDeletion() = %+v, %v", receipt, err)
	}
	readback, err := store.ResolveAcceptance(ctx, prepared)
	if !errors.Is(err, postgreswebhook.ErrPrivacyDeleted) || readback.Disposition != postgreswebhook.AcceptancePrivacyDeleted {
		t.Fatalf("ResolveAcceptance(deleted) = %+v, %v", readback, err)
	}
	var retainedActions int
	if err := pool.PGX().QueryRow(ctx, `SELECT count(*) FROM webhook_operator_actions WHERE owner_scope = 'owner-a' AND action_id = $1`, hold.ActionID).Scan(&retainedActions); err != nil || retainedActions != 0 {
		t.Fatalf("event-owned operator actions after privacy deletion = %d, %v", retainedActions, err)
	}
	replay, err := store.RequestEventPrivacyDeletion(ctx, action)
	if err != nil || !replay.Replay || replay.Result != "applied" {
		t.Fatalf("RequestEventPrivacyDeletion(replay) = %+v, %v", replay, err)
	}
}

func TestPostgresWebhookTombstoneProtectsEveryDeliveryIdentity(t *testing.T) {
	ctx, pool, store, _ := newPostgresWebhookFixture(t)
	for _, test := range []struct {
		name       string
		acceptance *string
		fanout     *string
		deliveries []string
	}{
		{name: "acceptance"},
		{name: "fanout"},
		{name: "delivery"},
	} {
		prepared := webhookPrepared(t, "tombstone-"+test.name)
		switch test.name {
		case "acceptance":
			test.acceptance = &prepared.Acceptance.AcceptanceID
		case "fanout":
			test.fanout = &prepared.Acceptance.FanoutSnapshotID
		case "delivery":
			test.deliveries = []string{prepared.Destinations[0].DeliveryID}
		}
		if _, err := pool.PGX().Exec(ctx, `INSERT INTO webhook_tombstones (
			owner_scope, target_kind, target_id, acceptance_id, fanout_snapshot_id,
			delivery_identities, last_semantic_class, action_id, action_encoding_version,
			request_fingerprint, first_disposition, deletion_authority, created_at
		) VALUES ('owner-a', 'event', $1, $2, $3,
			(SELECT COALESCE(jsonb_agg(jsonb_build_array(value)), '[]'::jsonb) FROM unnest($4::text[]) value),
			'privacy_deleted', $5, 'webhook-operator-action-v1', decode(repeat('00', 32), 'hex'),
			'applied', 'privacy-test', clock_timestamp())`, "deleted-"+test.name, test.acceptance, test.fanout, test.deliveries, "delete-"+test.name); err != nil {
			t.Fatal(err)
		}
		err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
			receipt, err := store.Accept(ctx, tx, prepared)
			if receipt.Disposition != postgreswebhook.AcceptancePrivacyDeleted {
				t.Fatalf("Accept(%s) = %+v, %v", test.name, receipt, err)
			}
			return err
		})
		if !errors.Is(err, postgreswebhook.ErrPrivacyDeleted) {
			t.Fatalf("Accept(%s) error = %v", test.name, err)
		}
	}
}

func TestPostgresWebhookAbsentPrivacyDeletionIsPermanentAndReplayable(t *testing.T) {
	ctx, pool, store, _ := newPostgresWebhookFixture(t)
	action := postgreswebhook.ActionRequest{OwnerScope: "owner-a", Actor: "privacy-a", ActionID: "privacy-absent", Kind: postgreswebhook.ActionPrivacyDelete, TargetKind: "event", TargetID: "event-absent", Reason: "privacy_request", Payload: &postgreswebhook.PrivacyDeletionAction{TargetKind: "event", TargetID: "event-absent", Mode: "minimal_tombstone", DeletionAuthority: "privacy-ticket-absent"}}
	first, err := store.RequestEventPrivacyDeletion(ctx, action)
	if err != nil || first.Result != "not_found" || first.Replay {
		t.Fatalf("RequestEventPrivacyDeletion(first) = %+v, %v", first, err)
	}
	replay, err := store.RequestEventPrivacyDeletion(ctx, action)
	if err != nil || replay.Result != "not_found" || !replay.Replay {
		t.Fatalf("RequestEventPrivacyDeletion(replay) = %+v, %v", replay, err)
	}
	base := webhookPrepared(t, "privacy-future")
	input := base.Acceptance
	input.BusinessEventID = action.TargetID
	input.Destinations = []postgreswebhook.DestinationSnapshot{base.Destinations[0].DestinationSnapshot}
	prepared, err := postgreswebhook.PrepareAcceptance(input)
	if err != nil {
		t.Fatal(err)
	}
	err = pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error { _, err := store.Accept(ctx, tx, prepared); return err })
	if !errors.Is(err, postgreswebhook.ErrPrivacyDeleted) {
		t.Fatalf("Accept(after absent deletion) error = %v", err)
	}
}

func TestPostgresWebhookPrivacySendBarrier(t *testing.T) {
	ctx, pool, store, manifest := newPostgresWebhookFixture(t)
	prepared := webhookPrepared(t, "privacy-barrier")
	if err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error { _, err := store.Accept(ctx, tx, prepared); return err }); err != nil {
		t.Fatal(err)
	}
	claim, err := store.Claim(ctx, "privacy-worker", 1, 30*time.Second, manifest)
	if err != nil || claim.Attempt == nil {
		t.Fatalf("Claim() = %+v, %v", claim, err)
	}

	blocker, err := pool.PGX().Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = blocker.Rollback(ctx) }()
	if _, err := blocker.Exec(ctx, `SELECT 1 FROM webhook_attempts WHERE attempt_id = $1 FOR UPDATE`, claim.Attempt.Identity.AttemptID); err != nil {
		t.Fatal(err)
	}
	action := postgreswebhook.ActionRequest{OwnerScope: "owner-a", Actor: "privacy-a", ActionID: "privacy-barrier", Kind: postgreswebhook.ActionPrivacyDelete, TargetKind: "event", TargetID: prepared.Acceptance.BusinessEventID, Reason: "privacy_request", Payload: &postgreswebhook.PrivacyDeletionAction{TargetKind: "event", TargetID: prepared.Acceptance.BusinessEventID, Mode: "minimal_tombstone", DeletionAuthority: "privacy-ticket-barrier"}}
	type result struct {
		receipt postgreswebhook.ActionReceipt
		err     error
	}
	resultC := make(chan result, 1)
	go func() {
		receipt, err := store.RequestEventPrivacyDeletion(ctx, action)
		resultC <- result{receipt: receipt, err: err}
	}()
	deadline := time.Now().Add(3 * time.Second)
	blocked := false
	for !blocked && time.Now().Before(deadline) {
		if err := pool.PGX().QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE $1 = ANY(pg_blocking_pids(pid)))`, blocker.Conn().PgConn().PID()).Scan(&blocked); err != nil {
			t.Fatal(err)
		}
		if !blocked {
			time.Sleep(10 * time.Millisecond)
		}
	}
	if !blocked {
		t.Fatal("privacy deletion did not wait on the send barrier")
	}
	if _, err := blocker.Exec(ctx, `UPDATE webhook_attempts SET send_authorized = true, may_have_sent = true,
		key_reference = 'key-a', key_references = ARRAY['key-a'],
		signature_header_digest = decode(repeat('00', 32), 'hex'),
		dns_set_digest = decode(repeat('00', 32), 'hex'), selected_address = decode('08080808', 'hex')
		WHERE attempt_id = $1`, claim.Attempt.Identity.AttemptID); err != nil {
		t.Fatal(err)
	}
	if err := blocker.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	privacy := <-resultC
	if privacy.err != nil || privacy.receipt.Result != "applied" {
		t.Fatalf("RequestEventPrivacyDeletion() = %+v, %v", privacy.receipt, privacy.err)
	}
	var semantic string
	if err := pool.PGX().QueryRow(ctx, `SELECT last_semantic_class FROM webhook_tombstones WHERE owner_scope = 'owner-a' AND target_kind = 'event' AND target_id = $1`, prepared.Acceptance.BusinessEventID).Scan(&semantic); err != nil || semantic != "outcome_unknown" {
		t.Fatalf("privacy tombstone semantic = %q, %v", semantic, err)
	}
	evidence := postgreswebhook.AuthorizationEvidence{KeyReference: "key-a", KeyReferences: []string{"key-a"}, SelectedAddress: netip.MustParseAddr("8.8.8.8")}
	if err := store.AuthorizeAttempt(ctx, *claim.Attempt, manifest, evidence); !errors.Is(err, postgreswebhook.ErrStaleAttempt) {
		t.Fatalf("AuthorizeAttempt(after privacy) error = %v", err)
	}
	var leased int
	if err := pool.PGX().QueryRow(ctx, `SELECT count(*) FROM webhook_capacity_slots WHERE attempt_id = $1`, claim.Attempt.Identity.AttemptID).Scan(&leased); err != nil || leased != 1 {
		t.Fatalf("privacy capacity fence = %d, %v", leased, err)
	}
	if _, err := pool.PGX().Exec(ctx, `UPDATE webhook_capacity_slots SET lease_expires_at = clock_timestamp() - interval '1 second' WHERE attempt_id = $1`, claim.Attempt.Identity.AttemptID); err != nil {
		t.Fatal(err)
	}
	if reconciled, err := store.ReconcileExpired(ctx, 10); err != nil || reconciled != 1 {
		t.Fatalf("ReconcileExpired(orphan capacity) = %d, %v", reconciled, err)
	}
}

func TestPostgresWebhookNamespaceRetirement(t *testing.T) {
	ctx, pool, store, manifest := newPostgresWebhookFixture(t)
	prepared := webhookPrepared(t, "namespace")
	if err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error { _, err := store.Accept(ctx, tx, prepared); return err }); err != nil {
		t.Fatal(err)
	}
	second := webhookPrepared(t, "namespace-second")
	if err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error { _, err := store.Accept(ctx, tx, second); return err }); err != nil {
		t.Fatal(err)
	}
	claim, err := store.Claim(ctx, "namespace-worker", 1, 30*time.Second, manifest)
	if err != nil || claim.Attempt == nil {
		t.Fatalf("Claim() = %+v, %v", claim, err)
	}
	action := postgreswebhook.ActionRequest{OwnerScope: "owner-a", Actor: "privacy-a", ActionID: "namespace-a", Kind: postgreswebhook.ActionNamespaceRetire, TargetKind: "namespace", TargetID: "owner-a", Reason: "privacy_request", Payload: &postgreswebhook.NamespaceRetirementAction{Mode: "full_erasure", DeletionAuthority: "privacy-ticket-a"}}
	receipt, err := store.RequestNamespaceRetirement(ctx, action, 1)
	if err != nil || receipt.Result != "pending" {
		t.Fatalf("RequestNamespaceRetirement(first batch) = %+v, %v", receipt, err)
	}
	for attempts := 0; attempts < 20; attempts++ {
		progressed, resumeErr := store.ResumeNamespaceRetirements(ctx, 1)
		err = resumeErr
		if err != nil {
			t.Fatalf("ResumeNamespaceRetirements() error = %v", err)
		}
		if progressed == 0 {
			break
		}
	}
	receipt, err = store.RequestNamespaceRetirement(ctx, action, 1)
	if err != nil || receipt.Result != "pending" {
		t.Fatalf("RequestNamespaceRetirement(replay) = %+v, %v", receipt, err)
	}
	if _, err := pool.PGX().Exec(ctx, `UPDATE webhook_capacity_slots SET lease_expires_at = clock_timestamp() - interval '1 second' WHERE owner_scope = 'owner-a'`); err != nil {
		t.Fatal(err)
	}
	if reconciled, err := store.ReconcileExpired(ctx, 10); err != nil || reconciled != 1 {
		t.Fatalf("ReconcileExpired(namespace capacity) = %d, %v", reconciled, err)
	}
	for attempts := 0; attempts < 20; attempts++ {
		progressed, err := store.ResumeNamespaceRetirements(ctx, 1)
		if err != nil {
			t.Fatalf("ResumeNamespaceRetirements(after fence) error = %v", err)
		}
		if progressed == 0 {
			break
		}
	}
	receipt, err = store.RequestNamespaceRetirement(ctx, action, 1)
	if err != nil {
		t.Fatalf("RequestNamespaceRetirement(completed replay) = %+v, %v", receipt, err)
	}
	if receipt.Result != "completed" {
		t.Fatalf("RequestNamespaceRetirement() = %+v, want completed", receipt)
	}
	err = pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error { _, err := store.Accept(ctx, tx, prepared); return err })
	if !errors.Is(err, postgreswebhook.ErrPrivacyDeleted) {
		t.Fatalf("Accept(retired namespace) error = %v", err)
	}
}
