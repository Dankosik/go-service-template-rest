package postgreswebhook

import (
	"net/http"
	"testing"
	"time"

	standardwebhooks "github.com/standard-webhooks/standard-webhooks/libraries/go"
)

func TestWebhookSigningUsesStandardWebhooks(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	attemptedAt := time.Unix(1_700_000_000, 0).UTC()
	body := []byte(`{"type":"order.created","data":{"id":"ord-1"}}`)
	header, err := SignV1("whd_test", attemptedAt, body, []SigningKey{{Reference: "v1", Bytes: key}})
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
}
