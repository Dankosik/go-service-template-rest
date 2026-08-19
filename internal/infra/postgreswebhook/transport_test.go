package postgreswebhook

import (
	"context"
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestWebhookRequestContractAndAddressFallback(t *testing.T) {
	deadline := time.Now().Add(time.Minute)
	prepared := PreparedSend{
		Attempt: DeliveryAttempt{ID: "whd_test", Body: []byte(`{"ok":true}`), AttemptedAt: time.Unix(1_700_000_000, 0), Deadline: deadline},
		URL:     mustWebhookURL(t, "https://example.com/hooks"), Signature: "v1,signature",
		Addresses: []netip.Addr{netip.MustParseAddr("8.8.8.8"), netip.MustParseAddr("1.1.1.1")},
	}
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()
	request, err := webhookRequest(ctx, prepared)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(request.Body)
	if string(body) != `{"ok":true}` || request.Header.Get("Webhook-Id") != "whd_test" || request.Header.Get("Webhook-Signature") != "v1,signature" {
		t.Fatalf("request = headers=%v body=%s", request.Header, body)
	}

	visited := make([]netip.Addr, 0, 2)
	result, err := tryPreparedAddresses(ctx, prepared, func(_ context.Context, candidate PreparedSend) (SendResult, error) {
		visited = append(visited, candidate.SelectedAddress)
		if len(visited) == 1 {
			return SendResult{Evidence: TransportEvidence{DefinitelyNotSent: true}}, context.DeadlineExceeded
		}
		return SendResult{Evidence: TransportEvidence{StatusCode: http.StatusNoContent, MayHaveSent: true}}, nil
	})
	if err != nil || result.Evidence.StatusCode != http.StatusNoContent || len(visited) != 2 {
		t.Fatalf("fallback = %+v, %v, visited=%v", result, err, visited)
	}
}

func TestWebhookURLAndDialPolicy(t *testing.T) {
	for _, raw := range []string{"http://example.com", "https://example.com:8443", "https://user@example.com", "https://example.com/?query=1"} {
		if _, err := parseWebhookURL(raw); err == nil {
			t.Fatalf("parseWebhookURL(%q) succeeded", raw)
		}
	}
	transport := newAttemptTransport("localhost", netip.MustParseAddr("127.0.0.1"))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := transport.DialContext(ctx, "tcp", "ignored:443"); err == nil || !strings.Contains(err.Error(), ErrDestinationDenied.Error()) {
		t.Fatalf("private dial error = %v", err)
	}
}

func mustWebhookURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	parsed, err := parseWebhookURL(raw)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}
