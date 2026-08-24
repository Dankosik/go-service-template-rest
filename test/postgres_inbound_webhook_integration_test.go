//go:build integration

package integration_test

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/inboundwebhook"
	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgres/pgtest"
	"github.com/example/go-service-template-rest/internal/infra/postgresinboundwebhook"
	standardwebhooks "github.com/standard-webhooks/standard-webhooks/libraries/go"
)

const (
	inboundVectorID        = "msg_123"
	inboundVectorTimestamp = "1700000000"
	inboundVectorBody      = `{"hello":"world"}`
	inboundVectorSignature = "v1,jUcl6cc4RhnPU/D4RhXcoyQYBvOxqIsONY9102iBndo="
)

func inboundKey() []byte {
	return []byte("0123456789abcdef0123456789abcdef")
}

func inboundTrust(t *testing.T, endpoint string) *postgresinboundwebhook.TrustManifest {
	t.Helper()
	endpoints := `{"endpoints":[{"endpoint_id":"` + endpoint + `","active_key_reference":"active"},{"endpoint_id":"other","active_key_reference":"other"}]}`
	secret := base64.StdEncoding.EncodeToString(inboundKey())
	other := base64.StdEncoding.EncodeToString([]byte("fedcba9876543210fedcba9876543210"))
	secrets := `{"entries":[{"endpoint_id":"` + endpoint + `","key_reference":"active","secret":"whsec_` + secret + `"},{"endpoint_id":"other","key_reference":"other","secret":"whsec_` + other + `"}]}`
	parsedEndpoints, err := postgresinboundwebhook.ParseEndpointManifest(endpoints)
	if err != nil {
		t.Fatal(err)
	}
	parsedSecrets, err := postgresinboundwebhook.ParseSecretManifest(secrets)
	if err != nil {
		t.Fatal(err)
	}
	trust, err := postgresinboundwebhook.BindSecrets(parsedEndpoints, parsedSecrets)
	if err != nil {
		t.Fatal(err)
	}
	return trust
}

func inboundReceiver(t *testing.T, dsn string) *postgresinboundwebhook.Receiver {
	t.Helper()
	ctx := t.Context()
	pool, err := postgres.Open(ctx, postgres.Options{DSN: dsn, MaxOpenConns: 8})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	receiver, err := postgresinboundwebhook.NewReceiver(
		pool,
		inboundTrust(t, "orders"),
		postgresinboundwebhook.WithClock(func() time.Time { return time.Unix(1700000000, 0).UTC() }),
	)
	if err != nil {
		t.Fatal(err)
	}
	return receiver
}

func inboundDelivery(endpoint, id, body, signature string) inboundwebhook.Delivery {
	return inboundwebhook.Delivery{
		EndpointID: endpoint,
		DeliveryID: id,
		Timestamp:  inboundVectorTimestamp,
		Signature:  signature,
		Body:       []byte(body),
	}
}

func TestPostgresInboundWebhookAtomicAcceptance(t *testing.T) {
	dsn := pgtest.Migrated(t, os.DirFS(".."), "migrations")
	receiver := inboundReceiver(t, dsn)
	ctx := t.Context()
	result, err := receiver.Receive(ctx, inboundDelivery("orders", inboundVectorID, inboundVectorBody, inboundVectorSignature))
	if err != nil || result.Outcome != inboundwebhook.OutcomeAccepted {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	pool, err := postgres.Open(ctx, postgres.Options{DSN: dsn, MaxOpenConns: 2})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	var receipts, jobs int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM inbound_webhook_receipts`).Scan(&receipts); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM river_job WHERE kind = 'inbound_webhook_receipt'`).Scan(&jobs); err != nil {
		t.Fatal(err)
	}
	if receipts != 1 || jobs != 1 {
		t.Fatalf("receipts=%d jobs=%d", receipts, jobs)
	}
	var args []byte
	if err := pool.QueryRow(ctx, `SELECT args FROM river_job WHERE kind = 'inbound_webhook_receipt'`).Scan(&args); err != nil {
		t.Fatal(err)
	}
	var decoded map[string]string
	if err := json.Unmarshal(args, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["receipt_id"] == "" || decoded["endpoint_id"] != "" || decoded["body"] != "" {
		t.Fatalf("job args = %s", args)
	}
}

func TestPostgresInboundWebhookAtomicAcceptanceRollsBackOnJobFailure(t *testing.T) {
	dsn := pgtest.Migrated(t, os.DirFS(".."), "migrations")
	ctx := t.Context()
	pool, err := postgres.Open(ctx, postgres.Options{DSN: dsn, MaxOpenConns: 2})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.Exec(ctx, `
		CREATE FUNCTION reject_inbound_webhook_job() RETURNS trigger
		LANGUAGE plpgsql AS $$
		BEGIN
			IF NEW.kind = 'inbound_webhook_receipt' THEN
				RAISE EXCEPTION 'reject inbound webhook job';
			END IF;
			RETURN NEW;
		END
		$$`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		CREATE TRIGGER reject_inbound_webhook_job
		BEFORE INSERT ON river_job
		FOR EACH ROW EXECUTE FUNCTION reject_inbound_webhook_job()`); err != nil {
		t.Fatal(err)
	}

	receiver := inboundReceiver(t, dsn)
	result, err := receiver.Receive(ctx, inboundDelivery("orders", inboundVectorID, inboundVectorBody, inboundVectorSignature))
	if result.Outcome != inboundwebhook.OutcomeUnavailable || !errors.Is(err, inboundwebhook.ErrUnavailable) {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	var receipts, jobs int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM inbound_webhook_receipts`).Scan(&receipts); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM river_job WHERE kind = 'inbound_webhook_receipt'`).Scan(&jobs); err != nil {
		t.Fatal(err)
	}
	if receipts != 0 || jobs != 0 {
		t.Fatalf("receipts=%d jobs=%d", receipts, jobs)
	}
}

func TestPostgresInboundWebhookIdentityArbitration(t *testing.T) {
	dsn := pgtest.Migrated(t, os.DirFS(".."), "migrations")
	receiver := inboundReceiver(t, dsn)
	ctx := t.Context()
	start := make(chan struct{})
	var wg sync.WaitGroup
	outcomes := make(chan inboundwebhook.Outcome, 32)
	for range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			result, err := receiver.Receive(ctx, inboundDelivery("orders", inboundVectorID, inboundVectorBody, inboundVectorSignature))
			if err != nil {
				t.Errorf("receive: %v", err)
				return
			}
			outcomes <- result.Outcome
		}()
	}
	close(start)
	wg.Wait()
	close(outcomes)
	var accepted, duplicate int
	for outcome := range outcomes {
		switch outcome {
		case inboundwebhook.OutcomeAccepted:
			accepted++
		case inboundwebhook.OutcomeDuplicate:
			duplicate++
		default:
			t.Fatalf("unexpected outcome %s", outcome)
		}
	}
	if accepted != 1 || duplicate != 31 {
		t.Fatalf("accepted=%d duplicate=%d", accepted, duplicate)
	}
	pool, err := postgres.Open(ctx, postgres.Options{DSN: dsn, MaxOpenConns: 2})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	var receipts, jobs int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM inbound_webhook_receipts`).Scan(&receipts); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM river_job WHERE kind = 'inbound_webhook_receipt'`).Scan(&jobs); err != nil {
		t.Fatal(err)
	}
	if receipts != 1 || jobs != 1 {
		t.Fatalf("receipts=%d jobs=%d", receipts, jobs)
	}

	otherBody := `{"hello":"other"}`
	webhook, err := standardwebhooks.NewWebhookRaw(inboundKey())
	if err != nil {
		t.Fatal(err)
	}
	sig, err := webhook.Sign(inboundVectorID, time.Unix(1700000000, 0).UTC(), []byte(otherBody))
	if err != nil {
		t.Fatal(err)
	}
	conflict, err := receiver.Receive(ctx, inboundDelivery("orders", inboundVectorID, otherBody, sig))
	if err != nil || conflict.Outcome != inboundwebhook.OutcomeConflict {
		t.Fatalf("conflict=%+v err=%v", conflict, err)
	}

	otherKey := []byte("fedcba9876543210fedcba9876543210")
	otherHook, err := standardwebhooks.NewWebhookRaw(otherKey)
	if err != nil {
		t.Fatal(err)
	}
	otherSig, err := otherHook.Sign(inboundVectorID, time.Unix(1700000000, 0).UTC(), []byte(inboundVectorBody))
	if err != nil {
		t.Fatal(err)
	}
	other, err := receiver.Receive(ctx, inboundDelivery("other", inboundVectorID, inboundVectorBody, otherSig))
	if err != nil || other.Outcome != inboundwebhook.OutcomeAccepted {
		t.Fatalf("other endpoint=%+v err=%v", other, err)
	}
}

func TestPostgresInboundWebhookSchemaLifecycle(t *testing.T) {
	dsn := pgtest.Migrated(t, os.DirFS(".."), "migrations")
	ctx := t.Context()
	pool, err := postgres.Open(ctx, postgres.Options{DSN: dsn, MaxOpenConns: 2})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	_, err = pool.Exec(ctx, `INSERT INTO inbound_webhook_receipts (receipt_id, endpoint_id, delivery_id, body_sha256, signed_at, payload, outcome)
		VALUES ('r1','e1','d1', repeat('\x00', 32)::bytea, now(), 'x', 'handled')`)
	if err == nil {
		t.Fatal("handled with payload accepted")
	}
	var writer bool
	var pending int
	if err := pool.QueryRow(ctx, `
		SELECT
			NOT pg_is_in_recovery()
				AND current_setting('transaction_read_only') = 'off' AS writer_primary,
			count(*) FILTER (WHERE outcome = 'pending') AS pending_receipts
		FROM inbound_webhook_receipts`).Scan(&writer, &pending); err != nil {
		t.Fatal(err)
	}
	if !writer || pending != 0 {
		t.Fatalf("writer=%t pending=%d", writer, pending)
	}
}
