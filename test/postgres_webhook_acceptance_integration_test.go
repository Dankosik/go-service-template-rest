//go:build integration

package integration_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgreswebhook"
	"github.com/jackc/pgx/v5"
)

func TestPostgresWebhookAcceptanceUsesJobsAuthority(t *testing.T) {
	ctx, pool, store := newPostgresJobsFixture(t)
	var activeRelations, deprecatedRelations int
	if err := pool.PGX().QueryRow(ctx, `
SELECT count(*) FILTER (WHERE tablename LIKE 'webhook_%'),
       count(*) FILTER (WHERE tablename LIKE 'deprecated_webhook_%')
FROM pg_tables
WHERE schemaname = current_schema()`).Scan(&activeRelations, &deprecatedRelations); err != nil {
		t.Fatal(err)
	}
	if activeRelations != 0 || deprecatedRelations != 11 {
		t.Fatalf("webhook relations = active %d, deprecated %d", activeRelations, deprecatedRelations)
	}
	endpoints, err := postgreswebhook.ParseEndpointManifest(`{"endpoints":[
		{"owner_scope":"orders","receiver_id":"alpha","generation":1,"url":"https://alpha.example/hooks","active_key_reference":"alpha-v1"},
		{"owner_scope":"orders","receiver_id":"beta","generation":1,"url":"https://beta.example/hooks","active_key_reference":"beta-v1"},
		{"owner_scope":"orders","receiver_id":"gamma","generation":1,"url":"https://gamma.example/hooks","active_key_reference":"gamma-v1"}
	]}`)
	if err != nil {
		t.Fatal(err)
	}
	dispatcher, err := postgreswebhook.NewDispatcher(endpoints)
	if err != nil {
		t.Fatal(err)
	}
	event := postgreswebhook.Event{
		OwnerScope: "orders", ID: "evt-1", Type: "order.created",
		OccurredAt: time.Date(2026, 8, 18, 10, 0, 0, 0, time.UTC),
		Data:       json.RawMessage(`{"order_id":"ord-1"}`),
	}
	prepared, err := dispatcher.Prepare(event, []postgreswebhook.ReceiverID{"alpha", "beta"})
	if err != nil {
		t.Fatal(err)
	}
	stage := func(prepared postgreswebhook.Prepared) (postgreswebhook.AcceptanceStatus, error) {
		var status postgreswebhook.AcceptanceStatus
		err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
			var err error
			status, err = prepared.Stage(ctx, store, tx)
			return err
		})
		return status, err
	}
	if status, err := stage(prepared); err != nil || status != postgreswebhook.AcceptanceNew {
		t.Fatalf("Stage(new) = %s, %v", status, err)
	}
	if status, err := stage(prepared); err != nil || status != postgreswebhook.AcceptanceExisting {
		t.Fatalf("Stage(existing) = %s, %v", status, err)
	}

	var count int
	if err := pool.PGX().QueryRow(ctx, `SELECT count(*) FROM postgres_jobs WHERE kind = 'outbound_webhook'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("webhook jobs = %d, want 2", count)
	}

	narrowed, err := dispatcher.Prepare(event, []postgreswebhook.ReceiverID{"alpha"})
	if err != nil {
		t.Fatal(err)
	}
	if status, err := stage(narrowed); !errors.Is(err, postgreswebhook.ErrConflict) || status != postgreswebhook.AcceptanceConflict {
		t.Fatalf("Stage(narrowed fanout) = %s, %v", status, err)
	}
	replaced, err := dispatcher.Prepare(event, []postgreswebhook.ReceiverID{"gamma"})
	if err != nil {
		t.Fatal(err)
	}
	if status, err := stage(replaced); !errors.Is(err, postgreswebhook.ErrConflict) || status != postgreswebhook.AcceptanceConflict {
		t.Fatalf("Stage(replaced fanout) = %s, %v", status, err)
	}
	if status, err := prepared.Resolve(ctx, store); err != nil || status != postgreswebhook.AcceptanceAccepted {
		t.Fatalf("Resolve() = %s, %v", status, err)
	}
}
