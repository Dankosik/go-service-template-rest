// profile:inbound-webhooks-standard:start
package inboundwebhook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestInboundWebhookTypedBindings(t *testing.T) {
	t.Parallel()

	type payload struct {
		Hello string `json:"hello"`
	}
	raw := json.RawMessage(`{ "hello" : "world" }`)
	reg := NewRegistry()
	var seen []byte
	var handled payload
	if err := Bind(reg, "orders", func(body json.RawMessage) (payload, error) {
		seen = append([]byte(nil), body...)
		if !json.Valid(body) {
			return payload{}, ErrDecodeRejected
		}
		var value payload
		if err := json.Unmarshal(body, &value); err != nil {
			return payload{}, ErrDecodeRejected
		}
		return value, nil
	}, func(_ context.Context, delivery VerifiedDelivery, value payload) error {
		if !bytes.Equal(delivery.Body, raw) {
			t.Fatalf("handler body = %q, want %q", delivery.Body, raw)
		}
		handled = value
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	if err := Bind[payload](reg, "orders", nil, nil); !errors.Is(err, ErrInvalidBinding) {
		t.Fatalf("nil binding error = %v", err)
	}
	if err := Bind(reg, "orders", func(json.RawMessage) (payload, error) { return payload{}, nil }, func(context.Context, VerifiedDelivery, payload) error { return nil }); !errors.Is(err, ErrDuplicateBinding) {
		t.Fatalf("duplicate binding error = %v", err)
	}

	other := NewRegistry()
	if err := Bind(other, "payments", func(json.RawMessage) (payload, error) {
		t.Fatal("other endpoint decoder invoked")
		return payload{}, nil
	}, func(context.Context, VerifiedDelivery, payload) error {
		t.Fatal("other endpoint handler invoked")
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	delivery := VerifiedDelivery{
		EndpointID: "orders",
		DeliveryID: "msg_123",
		SignedAt:   time.Unix(1700000000, 0).UTC(),
		Body:       raw,
		ReceivedAt: time.Unix(1700000001, 0).UTC(),
	}
	if err := reg.Dispatch(context.Background(), delivery); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(seen, raw) {
		t.Fatalf("decoder input = %q, want exact bytes %q", seen, raw)
	}
	if handled.Hello != "world" {
		t.Fatalf("handled = %+v", handled)
	}
	if err := other.Dispatch(context.Background(), delivery); !errors.Is(err, ErrUnknownBinding) {
		t.Fatalf("cross-endpoint dispatch error = %v", err)
	}
	if err := reg.Dispatch(context.Background(), VerifiedDelivery{EndpointID: "orders", Body: json.RawMessage(`not-json`)}); !errors.Is(err, ErrDecodeRejected) {
		t.Fatalf("invalid JSON error = %v", err)
	}
}

// profile:inbound-webhooks-standard:end
