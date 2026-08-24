// profile:inbound-webhooks-standard:start
package postgresinboundwebhook

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/inboundwebhook"
	standardwebhooks "github.com/standard-webhooks/standard-webhooks/libraries/go"
)

const (
	reviewedVectorID        = "msg_123"
	reviewedVectorTimestamp = "1700000000"
	reviewedVectorBody      = `{"hello":"world"}`
	reviewedVectorSignature = "v1,jUcl6cc4RhnPU/D4RhXcoyQYBvOxqIsONY9102iBndo="
)

func reviewedVectorKey() []byte {
	return []byte("0123456789abcdef0123456789abcdef")
}

func testTrust(t *testing.T, endpoint string, active, predecessor []byte) *TrustManifest {
	t.Helper()
	endpoints := `{"endpoints":[{"endpoint_id":"` + endpoint + `","active_key_reference":"active"`
	if len(predecessor) > 0 {
		endpoints += `,"predecessor_key_reference":"pred"`
	}
	endpoints += `}]}`
	secrets := `{"entries":[{"endpoint_id":"` + endpoint + `","key_reference":"active","secret":"whsec_` + base64.StdEncoding.EncodeToString(active) + `"}`
	if len(predecessor) > 0 {
		secrets += `,{"endpoint_id":"` + endpoint + `","key_reference":"pred","secret":"whsec_` + base64.StdEncoding.EncodeToString(predecessor) + `"}`
	}
	secrets += `]}`
	parsedEndpoints, err := ParseEndpointManifest(endpoints)
	if err != nil {
		t.Fatal(err)
	}
	parsedSecrets, err := ParseSecretManifest(secrets)
	if err != nil {
		t.Fatal(err)
	}
	trust, err := BindSecrets(parsedEndpoints, parsedSecrets)
	if err != nil {
		t.Fatal(err)
	}
	return trust
}

type spyStore struct {
	accepts int
}

func (s *spyStore) Accept(context.Context, receiptRecord) (inboundwebhook.Outcome, error) {
	s.accepts++
	return inboundwebhook.OutcomeAccepted, nil
}

func (s *spyStore) loadByID(context.Context, string) (storedReceipt, error) {
	return storedReceipt{}, nil
}
func (s *spyStore) MarkHandled(context.Context, string) (bool, error) { return false, nil }
func (s *spyStore) MarkQuarantined(context.Context, string, string) (bool, error) {
	return false, nil
}
func (s *spyStore) MarkFailed(context.Context, string) (bool, error) { return false, nil }

func TestStandardWebhooksVerificationBoundary(t *testing.T) {
	now := time.Unix(1700000000, 0).UTC()
	store := &spyStore{}
	receiver, err := NewReceiver(nil, testTrust(t, "orders", reviewedVectorKey(), nil),
		withStore(store),
		WithClock(func() time.Time { return now }),
	)
	if err != nil {
		t.Fatal(err)
	}

	result, err := receiver.Receive(context.Background(), inboundwebhook.Delivery{
		EndpointID: "orders",
		DeliveryID: reviewedVectorID,
		Timestamp:  reviewedVectorTimestamp,
		Signature:  reviewedVectorSignature,
		Body:       []byte(reviewedVectorBody),
	})
	if err != nil || result.Outcome != inboundwebhook.OutcomeAccepted {
		t.Fatalf("vector result = %+v, err = %v", result, err)
	}
	if store.accepts != 1 {
		t.Fatalf("accepts = %d", store.accepts)
	}

	for _, tc := range []struct {
		name       string
		now        time.Time
		deliveryID string
		timestamp  string
		body       string
		signature  string
		endpoint   string
	}{
		{name: "stale", now: now.Add(301 * time.Second), body: reviewedVectorBody, signature: reviewedVectorSignature, endpoint: "orders"},
		{name: "future", now: now.Add(-301 * time.Second), body: reviewedVectorBody, signature: reviewedVectorSignature, endpoint: "orders"},
		{name: "body", now: now, body: `{"hello":"world!"}`, signature: reviewedVectorSignature, endpoint: "orders"},
		{name: "signature", now: now, body: reviewedVectorBody, signature: "v1,AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=", endpoint: "orders"},
		{name: "delivery id binding", now: now, deliveryID: "msg_124", body: reviewedVectorBody, signature: reviewedVectorSignature, endpoint: "orders"},
		{name: "timestamp binding", now: now, timestamp: "1700000001", body: reviewedVectorBody, signature: reviewedVectorSignature, endpoint: "orders"},
		{name: "other endpoint", now: now, body: reviewedVectorBody, signature: reviewedVectorSignature, endpoint: "payments"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			beforeAccepts := store.accepts
			receiver.now = func() time.Time { return tc.now }
			deliveryID := tc.deliveryID
			if deliveryID == "" {
				deliveryID = reviewedVectorID
			}
			timestamp := tc.timestamp
			if timestamp == "" {
				timestamp = reviewedVectorTimestamp
			}
			result, err := receiver.Receive(context.Background(), inboundwebhook.Delivery{
				EndpointID: tc.endpoint,
				DeliveryID: deliveryID,
				Timestamp:  timestamp,
				Signature:  tc.signature,
				Body:       []byte(tc.body),
			})
			if err != nil {
				t.Fatal(err)
			}
			if result.Outcome != inboundwebhook.OutcomeRejected && result.Outcome != inboundwebhook.OutcomeUnknownEndpoint {
				t.Fatalf("outcome = %s", result.Outcome)
			}
			if store.accepts != beforeAccepts {
				t.Fatal("store called on rejection")
			}
		})
	}

	t.Run("predecessor", func(t *testing.T) {
		pred := []byte("abcdef0123456789abcdef0123456789")
		webhook, err := standardwebhooks.NewWebhookRaw(pred)
		if err != nil {
			t.Fatal(err)
		}
		signature, err := webhook.Sign(reviewedVectorID, now, []byte(reviewedVectorBody))
		if err != nil {
			t.Fatal(err)
		}
		rotated, err := NewReceiver(nil, testTrust(t, "orders", reviewedVectorKey(), pred),
			withStore(&spyStore{}),
			WithClock(func() time.Time { return now }),
		)
		if err != nil {
			t.Fatal(err)
		}
		result, err := rotated.Receive(context.Background(), inboundwebhook.Delivery{
			EndpointID: "orders",
			DeliveryID: reviewedVectorID,
			Timestamp:  reviewedVectorTimestamp,
			Signature:  signature,
			Body:       []byte(reviewedVectorBody),
		})
		if err != nil || result.Outcome != inboundwebhook.OutcomeAccepted {
			t.Fatalf("predecessor result = %+v err = %v", result, err)
		}
	})

	t.Run("tolerance edges", func(t *testing.T) {
		for _, delta := range []time.Duration{-300 * time.Second, 300 * time.Second} {
			receiver.now = func() time.Time { return now.Add(delta) }
			result, err := receiver.Receive(context.Background(), inboundwebhook.Delivery{
				EndpointID: "orders",
				DeliveryID: reviewedVectorID,
				Timestamp:  reviewedVectorTimestamp,
				Signature:  reviewedVectorSignature,
				Body:       []byte(reviewedVectorBody),
			})
			if err != nil || result.Outcome != inboundwebhook.OutcomeAccepted {
				t.Fatalf("edge %s result = %+v err = %v", delta, result, err)
			}
		}
	})
}

// profile:inbound-webhooks-standard:end
