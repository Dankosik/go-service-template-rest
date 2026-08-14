//go:build integration

package integration_test

import (
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestPostgresWebhookRetention(t *testing.T) {
	ctx, pool, store, _ := newPostgresWebhookFixture(t)
	prepared := webhookPrepared(t, "retention")
	if err := pool.InTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error { _, err := store.Accept(ctx, tx, prepared); return err }); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.PGX().Exec(ctx, `UPDATE webhook_cycles SET disposition = 'http_rejected', finalized_at = clock_timestamp() WHERE owner_scope = 'owner-a' AND delivery_id = $1`, prepared.Destinations[0].DeliveryID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.PGX().Exec(ctx, `UPDATE webhook_deliveries SET state = 'terminal', cumulative_summary = 'http_rejected', terminal_at = clock_timestamp(), redrive_eligible_until = clock_timestamp() - interval '1 second' WHERE owner_scope = 'owner-a' AND delivery_id = $1`, prepared.Destinations[0].DeliveryID); err != nil {
		t.Fatal(err)
	}
	deleted, err := store.CleanupRetention(ctx, 10)
	if err != nil || deleted != 1 {
		t.Fatalf("CleanupRetention() = %d, %v", deleted, err)
	}
}
