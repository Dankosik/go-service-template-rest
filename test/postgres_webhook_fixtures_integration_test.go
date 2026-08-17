//go:build integration

package integration_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgres/pgtest"
	"github.com/example/go-service-template-rest/internal/infra/postgreswebhook"
)

func newPostgresWebhookFixture(t *testing.T) (context.Context, *postgres.Pool, *postgreswebhook.Store, *postgreswebhook.SecretManifest) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	t.Cleanup(cancel)
	dsn := pgtest.Migrated(t, os.DirFS(".."), "migrations")
	pool, err := postgres.New(ctx, postgres.Options{DSN: dsn, ConnectTimeout: 3 * time.Second, HealthcheckTimeout: 3 * time.Second, MaxOpenConns: 12, AcquireTimeout: time.Second, ConnMaxLifetime: time.Hour, StatementTimeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("postgres.New(): %v", err)
	}
	t.Cleanup(pool.Close)
	manifest := webhookManifest(t, 1, "owner-a", "dest-a", "key-a")
	store, err := postgreswebhook.NewStore(pool, postgreswebhook.StoreOptions{
		OperationTimeout: 3 * time.Second, CapacityRevision: 1, GlobalConcurrency: 2, ManifestRevision: manifest.Revision(),
		AttemptTimeout: 5 * time.Second, ResponseHeaderTimeout: 2 * time.Second, ResponseHeaderBytes: 4096,
		ResponseBodyBytes: 4096, DrainTimeout: 10 * time.Second,
	})
	if err != nil {
		t.Fatalf("postgreswebhook.NewStore(): %v", err)
	}
	if err := store.InitializeOrTransitionCapacity(ctx); err != nil {
		t.Fatalf("InitializeOrTransitionCapacity(): %v", err)
	}
	return ctx, pool, store, manifest
}

func webhookManifest(t *testing.T, revision int64, owner, destination, reference string) *postgreswebhook.SecretManifest {
	t.Helper()
	secret := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	raw := fmt.Sprintf(`{"revision":%d,"entries":[{"owner_scope":%q,"destination_id":%q,"key_reference":%q,"secret":"whsec_%s"}]}`, revision, owner, destination, reference, secret)
	manifest, err := postgreswebhook.ParseSecretManifest(raw)
	if err != nil {
		t.Fatalf("ParseSecretManifest(): %v", err)
	}
	return manifest
}

func webhookRotationManifest(t *testing.T) *postgreswebhook.SecretManifest {
	t.Helper()
	oldSecret := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	newSecret := base64.StdEncoding.EncodeToString([]byte("abcdef0123456789abcdef0123456789"))
	raw := fmt.Sprintf(`{"revision":2,"entries":[{"owner_scope":"owner-a","destination_id":"dest-a","key_reference":"key-new","secret":"whsec_%s"},{"owner_scope":"owner-a","destination_id":"dest-a","key_reference":"key-a","secret":"whsec_%s"}]}`, newSecret, oldSecret)
	manifest, err := postgreswebhook.ParseSecretManifest(raw)
	if err != nil {
		t.Fatalf("ParseSecretManifest(rotation): %v", err)
	}
	return manifest
}

func webhookPrepared(t *testing.T, suffix string) postgreswebhook.PreparedAcceptance {
	t.Helper()
	policy := postgreswebhook.DeliveryPolicy{
		MaximumPayloadBytes: 262144, AcceptedContentTypes: []string{"application/json"}, AcceptedBusinessSchemas: []string{"1"},
		MaximumAttempts: 3, MaximumDeliveryAge: time.Hour, BackoffBase: time.Second, BackoffCap: time.Minute,
		RetryAfterCap: time.Minute, AttemptTimeout: 5 * time.Second, ResponseHeaderTimeout: 2 * time.Second,
		ResponseHeaderBytes: 4096, ResponseBodyBytes: 4096, DestinationConcurrency: 1, GlobalConcurrency: 2,
		DrainTimeout: 10 * time.Second, RedriveAttempts: 2, RedriveAge: time.Hour,
		Horizons: postgreswebhook.RetentionHorizons{Payload: time.Hour, Active: time.Hour, TerminalSummary: 2 * time.Hour, Attempt: 2 * time.Hour, Action: 3 * time.Hour, DestinationGeneration: 3 * time.Hour, RedriveEligibility: time.Hour, ReceiverDedup: time.Hour},
	}
	prepared, err := postgreswebhook.PrepareAcceptance(postgreswebhook.Acceptance{
		OwnerScope: "owner-a", AcceptanceID: "accept-" + suffix, BusinessEventID: "event-" + suffix,
		FanoutSnapshotID: "fanout-" + suffix, EventType: "order.created", BusinessSchemaVersion: "1",
		ContentType: "application/json", Body: []byte(`{"id":"` + suffix + `"}`), DeliveryEnvelopeVersion: "1",
		SubscriberPolicyRevision: "policy-1", Destinations: []postgreswebhook.DestinationSnapshot{{
			DestinationID: "dest-a", Generation: 1, OwnershipVerificationReceipt: "verify-1",
			URL: "https://hooks.example.com/orders", SelectionRevision: "selection-1",
			PayloadVersionPreference: "1", SignatureProfile: "v1", SigningAuthorityBinding: "key-a", Policy: policy,
		}},
	})
	if err != nil {
		t.Fatalf("PrepareAcceptance(): %v", err)
	}
	return prepared
}
