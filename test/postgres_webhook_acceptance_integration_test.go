//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgres/pgtest"
	"github.com/example/go-service-template-rest/internal/infra/postgreswebhook"
	"github.com/jackc/pgx/v5"
)

func TestPostgresWebhookAcceptanceUsesJobsAuthority(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	dsn := pgtest.Migrated(t, os.DirFS(".."), "migrations")
	pool, err := postgres.Open(ctx, postgres.Options{DSN: dsn, MaxOpenConns: 4})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	var activeRelations, deprecatedRelations int
	if err := pool.QueryRow(ctx, `
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
	futureOccurredAt := time.Now().UTC().Add(30 * 24 * time.Hour).Truncate(time.Second)
	event := postgreswebhook.Event{
		OwnerScope: "orders", ID: "evt-1", Type: "order.created",
		OccurredAt: futureOccurredAt,
		Data:       json.RawMessage(`{"order_id":"ord-1"}`),
	}
	prepared, err := dispatcher.Prepare(event, []postgreswebhook.ReceiverID{"alpha", "beta"})
	if err != nil {
		t.Fatal(err)
	}
	stage := func(prepared postgreswebhook.Prepared) (bool, error) {
		var inserted bool
		err := postgres.InTx(ctx, pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
			var err error
			inserted, err = prepared.Stage(ctx, tx)
			return err
		})
		return inserted, err
	}
	if inserted, err := stage(prepared); err != nil || !inserted {
		t.Fatalf("Stage(new) = %t, %v", inserted, err)
	}
	if inserted, err := stage(prepared); err != nil || inserted {
		t.Fatalf("Stage(existing) = %t, %v", inserted, err)
	}

	var count, available int
	if err := pool.QueryRow(ctx, `
		SELECT count(*), count(*) FILTER (WHERE state = 'available')
		FROM river_job
		WHERE kind = 'outbound_webhook'`).Scan(&count, &available); err != nil {
		t.Fatal(err)
	}
	if count != 2 || available != 2 {
		t.Fatalf("webhook jobs = %d, available = %d, want 2 immediately eligible", count, available)
	}

	narrowed, err := dispatcher.Prepare(event, []postgreswebhook.ReceiverID{"alpha"})
	if err != nil {
		t.Fatal(err)
	}
	if inserted, err := stage(narrowed); !errors.Is(err, postgreswebhook.ErrConflict) || inserted {
		t.Fatalf("Stage(narrowed fanout) = %t, %v", inserted, err)
	}
	replaced, err := dispatcher.Prepare(event, []postgreswebhook.ReceiverID{"gamma"})
	if err != nil {
		t.Fatal(err)
	}
	if inserted, err := stage(replaced); !errors.Is(err, postgreswebhook.ErrConflict) || inserted {
		t.Fatalf("Stage(replaced fanout) = %t, %v", inserted, err)
	}
	if accepted, err := prepared.ResolveCurrent(ctx, pool); err != nil || !accepted {
		t.Fatalf("ResolveCurrent() = %t, %v", accepted, err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM river_job WHERE kind = 'outbound_webhook'`); err != nil {
		t.Fatal(err)
	}
	if accepted, err := prepared.ResolveCurrent(ctx, pool); err != nil || accepted {
		t.Fatalf("ResolveCurrent(after retention) = %t, %v", accepted, err)
	}
}
