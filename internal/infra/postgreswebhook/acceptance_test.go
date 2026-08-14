package postgreswebhook

import (
	"encoding/hex"
	"testing"
	"time"
)

func goldenAcceptance() Acceptance {
	day := 24 * time.Hour
	return Acceptance{
		OwnerScope: "owner-a", AcceptanceID: "accept-01", BusinessEventID: "evt-01", FanoutSnapshotID: "fanout-01",
		EventType: "order.created", BusinessSchemaVersion: "1", ContentType: "application/json", Body: []byte(`{"id":"evt-01"}`), DeliveryEnvelopeVersion: "1", SubscriberPolicyRevision: "subrev-7",
		Destinations: []DestinationSnapshot{{
			DestinationID: "dest-01", Generation: 3, OwnershipVerificationReceipt: "verify-9", URL: "https://hooks.example.test/orders", SelectionRevision: "sel-4", PayloadVersionPreference: "1", SignatureProfile: "v1", SigningAuthorityBinding: "keys-01",
			Policy: DeliveryPolicy{MaximumPayloadBytes: 262144, AcceptedContentTypes: []string{"application/json"}, AcceptedBusinessSchemas: []string{"1"}, MaximumAttempts: 8, MaximumDeliveryAge: day, BackoffBase: time.Second, BackoffCap: 5 * time.Minute, RetryAfterCap: time.Hour, AttemptTimeout: 10 * time.Second, ResponseHeaderTimeout: 3 * time.Second, ResponseHeaderBytes: 16384, ResponseBodyBytes: 65536, DestinationConcurrency: 2, GlobalConcurrency: 32, DrainTimeout: 20 * time.Second, RedriveAttempts: 3, RedriveAge: time.Hour, Horizons: [8]time.Duration{7 * day, 7 * day, 30 * day, 30 * day, 90 * day, 90 * day, 7 * day, 7 * day}},
		}},
	}
}

func TestWebhookAcceptanceCanonicalVector(t *testing.T) {
	prepared, err := PrepareAcceptance(goldenAcceptance())
	if err != nil {
		t.Fatal(err)
	}
	if got := len(prepared.CanonicalBytes); got != 736 {
		t.Fatalf("canonical length = %d, want 736; digest=%s", got, hex.EncodeToString(prepared.Fingerprint[:]))
	}
	const expected = "40d72664c74d6e84ce96f82dec63b8471c15d9c3586e59438d586c2dd0d232a2"
	if got := hex.EncodeToString(prepared.Fingerprint[:]); got != expected {
		t.Fatalf("fingerprint = %s, want %s", got, expected)
	}
	if prepared.Destinations[0].DeliveryID == "" {
		t.Fatal("delivery ID is empty")
	}

	mutated := goldenAcceptance()
	mutated.Body = []byte(`{"id":"evt-02"}`)
	other, err := PrepareAcceptance(mutated)
	if err != nil {
		t.Fatal(err)
	}
	if other.Fingerprint == prepared.Fingerprint {
		t.Fatal("changed intent retained fingerprint")
	}
}
