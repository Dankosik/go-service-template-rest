package postgreswebhook

import (
	"encoding/hex"
	"encoding/json"
	"errors"
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
			Policy: DeliveryPolicy{MaximumPayloadBytes: 262144, AcceptedContentTypes: []string{"application/json"}, AcceptedBusinessSchemas: []string{"1"}, MaximumAttempts: 8, MaximumDeliveryAge: day, BackoffBase: time.Second, BackoffCap: 5 * time.Minute, RetryAfterCap: time.Hour, AttemptTimeout: 10 * time.Second, ResponseHeaderTimeout: 3 * time.Second, ResponseHeaderBytes: 16384, ResponseBodyBytes: 65536, DestinationConcurrency: 2, GlobalConcurrency: 32, DrainTimeout: 20 * time.Second, RedriveAttempts: 3, RedriveAge: time.Hour, Horizons: RetentionHorizons{Payload: 7 * day, Active: 7 * day, TerminalSummary: 30 * day, Attempt: 30 * day, Action: 90 * day, DestinationGeneration: 90 * day, RedriveEligibility: 7 * day, ReceiverDedup: 7 * day}},
		}},
	}
}

func TestWebhookAcceptanceEnforcesDestinationAdmission(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*Acceptance)
	}{
		{name: "payload", mutate: func(input *Acceptance) { input.Destinations[0].Policy.MaximumPayloadBytes = len(input.Body) - 1 }},
		{name: "content type", mutate: func(input *Acceptance) { input.ContentType = "application/xml" }},
		{name: "business schema", mutate: func(input *Acceptance) { input.BusinessSchemaVersion = "2" }},
		{name: "automatic pause", mutate: func(input *Acceptance) {
			input.Destinations[0].Policy.AutomaticPause = true
			input.Destinations[0].Policy.AutomaticPauseClasses = []string{"http_rejected"}
			input.Destinations[0].Policy.PauseWindow = time.Minute
			input.Destinations[0].Policy.PauseThreshold = 1
			input.Destinations[0].Policy.PauseMinimumTraffic = 1
			input.Destinations[0].Policy.PauseAlertPolicy = "alert-1"
		}},
		{name: "attempt ceiling", mutate: func(input *Acceptance) {
			input.Destinations[0].Policy.AttemptTimeout = MaxAttemptTime + time.Nanosecond
			input.Destinations[0].Policy.DrainTimeout = input.Destinations[0].Policy.AttemptTimeout + time.Second
		}},
		{name: "drain boundary", mutate: func(input *Acceptance) {
			input.Destinations[0].Policy.DrainTimeout = input.Destinations[0].Policy.AttemptTimeout
		}},
		{name: "response ceiling", mutate: func(input *Acceptance) { input.Destinations[0].Policy.ResponseBodyBytes = MaxResponseBytes + 1 }},
		{name: "content type list ceiling", mutate: func(input *Acceptance) {
			input.Destinations[0].Policy.AcceptedContentTypes = make([]string, MaxPolicyListItems+1)
		}},
		{name: "tls policy", mutate: func(input *Acceptance) { input.Destinations[0].Policy.MinimumTLSVersion = "1.1" }},
		{name: "retention dependency", mutate: func(input *Acceptance) { input.Destinations[0].Policy.Horizons.Payload = time.Hour }},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := goldenAcceptance()
			test.mutate(&input)
			if _, err := PrepareAcceptance(input); !errors.Is(err, ErrConfig) {
				t.Fatalf("PrepareAcceptance() error = %v, want ErrConfig", err)
			}
		})
	}
}

func TestWebhookRetentionHorizonsRequireExactShape(t *testing.T) {
	var horizons RetentionHorizons
	if err := json.Unmarshal([]byte(`[1,2,3,4,5,6,7,8,9]`), &horizons); err == nil {
		t.Fatal("UnmarshalJSON() accepted a ninth retention horizon")
	}
}

func TestWebhookAcceptanceCanonicalVector(t *testing.T) {
	prepared, err := PrepareAcceptance(goldenAcceptance())
	if err != nil {
		t.Fatal(err)
	}
	if got := len(prepared.CanonicalBytes); got != 743 {
		t.Fatalf("canonical length = %d, want 743; digest=%s", got, hex.EncodeToString(prepared.Fingerprint[:]))
	}
	const expected = "24dd3797de2eda77cd0646ef823255f4d60aa6ebfbb5a372723473c1c0d9c784"
	if got := hex.EncodeToString(prepared.Fingerprint[:]); got != expected {
		t.Fatalf("fingerprint = %s, want %s", got, expected)
	}
	if prepared.Destinations[0].DeliveryID == "" {
		t.Fatal("delivery ID is empty")
	}
	reconstructed, err := PrepareAcceptance(goldenAcceptance())
	if err != nil {
		t.Fatal(err)
	}
	if reconstructed.Destinations[0].DeliveryID != prepared.Destinations[0].DeliveryID {
		t.Fatal("delivery ID changed when the acceptance intent was reconstructed")
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
	if other.Destinations[0].DeliveryID == prepared.Destinations[0].DeliveryID {
		t.Fatal("changed intent retained delivery ID")
	}
}
