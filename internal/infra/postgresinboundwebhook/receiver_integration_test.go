//go:build integration

// profile:inbound-webhooks-standard:start
package postgresinboundwebhook

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/inboundwebhook"
	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgres/pgtest"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMain(m *testing.M) { os.Exit(pgtest.Main(m, "")) }

func TestPostgresInboundWebhookCommitUnknownRetry(t *testing.T) {
	dsn := pgtest.Migrated(t, os.DirFS("../../.."), "migrations")
	ctx := t.Context()
	pool, err := postgres.Open(ctx, postgres.Options{DSN: dsn, MaxOpenConns: 4})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	store, err := newPostgresStore(pool)
	if err != nil {
		t.Fatal(err)
	}
	store.inTx = func(ctx context.Context, pool *pgxpool.Pool, opts pgx.TxOptions, fn func(pgx.Tx) error) error {
		if err := postgres.InTx(ctx, pool, opts, fn); err != nil {
			return err
		}
		return postgres.ErrCommitUnknown
	}
	receiver, err := NewReceiver(nil, testTrust(t, "orders", reviewedVectorKey(), nil),
		withStore(store),
		WithClock(func() time.Time { return time.Unix(1700000000, 0).UTC() }),
	)
	if err != nil {
		t.Fatal(err)
	}
	delivery := inboundwebhook.Delivery{
		EndpointID: "orders",
		DeliveryID: reviewedVectorID,
		Timestamp:  reviewedVectorTimestamp,
		Signature:  reviewedVectorSignature,
		Body:       []byte(reviewedVectorBody),
	}
	result, err := receiver.Receive(ctx, delivery)
	if result != inboundwebhook.OutcomeUnavailable || !errors.Is(err, inboundwebhook.ErrUnavailable) {
		t.Fatalf("commit-unknown result=%+v err=%v", result, err)
	}

	plain, err := NewReceiver(pool, testTrust(t, "orders", reviewedVectorKey(), nil),
		WithClock(func() time.Time { return time.Unix(1700000000, 0).UTC() }),
	)
	if err != nil {
		t.Fatal(err)
	}
	retry, err := plain.Receive(ctx, delivery)
	if err != nil || retry != inboundwebhook.OutcomeDuplicate {
		t.Fatalf("retry=%+v err=%v", retry, err)
	}
}

// profile:inbound-webhooks-standard:end
