package postgreswebhook

import (
	"bytes"
	"net/http"
	"testing"
	"time"

	standardwebhooks "github.com/standard-webhooks/standard-webhooks/libraries/go"
)

func TestWebhookSigningUsesStandardWebhooks(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	attemptedAt := time.Unix(1_700_000_000, 0).UTC()
	body := []byte(`{"type":"order.created","data":{"id":"ord-1"}}`)
	header, err := signV1("whd_test", attemptedAt, body, [][]byte{key})
	if err != nil {
		t.Fatal(err)
	}
	webhook, err := standardwebhooks.NewWebhookRaw(key)
	if err != nil {
		t.Fatal(err)
	}
	headers := http.Header{
		"Webhook-Id":        []string{"whd_test"},
		"Webhook-Timestamp": []string{"1700000000"},
		"Webhook-Signature": []string{header},
	}
	if err := webhook.VerifyIgnoringTimestamp(body, headers); err != nil {
		t.Fatalf("VerifyIgnoringTimestamp() error = %v", err)
	}

	nextAt := attemptedAt.Add(time.Second)
	next, err := signV1("whd_test", nextAt, body, [][]byte{key})
	if err != nil {
		t.Fatal(err)
	}
	headers.Set("Webhook-Timestamp", "1700000001")
	headers.Set("Webhook-Signature", next)
	if header == next || webhook.VerifyIgnoringTimestamp(body, headers) != nil {
		t.Fatal("retry changed identity or body instead of only timestamp and signature")
	}
}

func TestWebhookSigningRejectsKeysOutsideSharedBounds(t *testing.T) {
	t.Parallel()

	for _, size := range []int{31, 65} {
		_, err := signV1("whd_test", time.Unix(1_700_000_000, 0), nil, [][]byte{bytes.Repeat([]byte{'k'}, size)})
		if err == nil {
			t.Fatalf("signV1(%d byte key) error = %v", size, err)
		}
	}
}
