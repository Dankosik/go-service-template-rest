//go:build integration

package integration_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgreswebhook"
	standardwebhooks "github.com/standard-webhooks/standard-webhooks/libraries/go"
)

func TestWebhookRetrySignatureIdentity(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	body := []byte(`{"event":"order.created","id":"evt-01"}`)
	firstAt := time.Unix(1700000000, 0)
	secondAt := firstAt.Add(time.Second)
	first, err := postgreswebhook.SignV1("whd_delivery-01", firstAt, body, []postgreswebhook.SigningKey{{Reference: "key-a", Bytes: key}})
	if err != nil {
		t.Fatal(err)
	}
	second, err := postgreswebhook.SignV1("whd_delivery-01", secondAt, body, []postgreswebhook.SigningKey{{Reference: "key-a", Bytes: key}})
	if err != nil {
		t.Fatal(err)
	}
	webhook, err := standardwebhooks.NewWebhookRaw(key)
	if err != nil {
		t.Fatal(err)
	}
	firstHeaders := http.Header{"Webhook-Id": {"whd_delivery-01"}, "Webhook-Timestamp": {"1700000000"}, "Webhook-Signature": {first}}
	secondHeaders := http.Header{"Webhook-Id": {"whd_delivery-01"}, "Webhook-Timestamp": {"1700000001"}, "Webhook-Signature": {second}}
	if first == second || webhook.VerifyIgnoringTimestamp(body, firstHeaders) != nil || webhook.VerifyIgnoringTimestamp(body, secondHeaders) != nil {
		t.Fatal("retry identity did not retain body/delivery while refreshing attempt time and signature")
	}
}
