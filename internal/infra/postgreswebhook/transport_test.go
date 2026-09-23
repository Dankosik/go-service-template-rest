package postgreswebhook

import (
	"context"
	"errors"
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
	prepared := preparedSend{
		Attempt: deliveryAttempt{DeliveryID: "whd_test", Body: []byte(`{"ok":true}`), AttemptedAt: time.Unix(1_700_000_000, 0), Deadline: deadline},
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
	result, err := tryPreparedAddresses(ctx, prepared, func(_ context.Context, candidate preparedSend) (sendResult, error) {
		visited = append(visited, candidate.SelectedAddress)
		if len(visited) == 1 {
			return sendResult{Evidence: transportEvidence{Certainty: sendCertaintyDefinitelyNotSent}}, context.DeadlineExceeded
		}
		return sendResult{Evidence: transportEvidence{StatusCode: http.StatusNoContent, Certainty: sendCertaintyMayHaveSent}}, nil
	})
	if err != nil || result.Evidence.StatusCode != http.StatusNoContent || len(visited) != 2 {
		t.Fatalf("fallback = %+v, %v, visited=%v", result, err, visited)
	}

	visited = visited[:0]
	result, err = tryPreparedAddresses(ctx, prepared, func(_ context.Context, candidate preparedSend) (sendResult, error) {
		visited = append(visited, candidate.SelectedAddress)
		return sendResult{}, context.DeadlineExceeded
	})
	if !errors.Is(err, context.DeadlineExceeded) || result.Evidence.Certainty != sendCertaintyUnspecified || len(visited) != 1 {
		t.Fatalf("unspecified fallback = %+v, %v, visited=%v", result, err, visited)
	}
}

func TestWebhookURLAndDialPolicy(t *testing.T) {
	for _, raw := range []string{"http://example.com", "https://example.com:8443", "https://user@example.com", "https://example.com/?", "https://example.com/?query=1"} {
		if _, err := parseWebhookURL(raw); err == nil {
			t.Fatalf("parseWebhookURL(%q) succeeded", raw)
		}
	}
	transport := newAttemptTransport("localhost", netip.MustParseAddr("127.0.0.1"))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := transport.DialContext(ctx, "tcp", "ignored:443"); err == nil || !strings.Contains(err.Error(), errDestinationDenied.Error()) {
		t.Fatalf("private dial error = %v", err)
	}
}

func TestAdmitDestinationAddresses(t *testing.T) {
	addresses := []netip.Addr{netip.MustParseAddr("::ffff:1.1.1.1"), netip.MustParseAddr("8.8.8.8")}
	admitted, err := admitDestinationAddresses(addresses)
	if err != nil {
		t.Fatal(err)
	}
	if addresses[0] != netip.MustParseAddr("1.1.1.1") || len(admitted) != 2 || admitted[0] != netip.MustParseAddr("1.1.1.1") {
		t.Fatalf("admitted addresses = %v; input = %v", admitted, addresses)
	}
	admitted[0] = netip.Addr{}
	if addresses[0] != netip.MustParseAddr("1.1.1.1") {
		t.Fatalf("admitted addresses alias input: %v", addresses)
	}

	tooMany := make([]netip.Addr, maxDNSAddresses+1)
	for i := range tooMany {
		tooMany[i] = netip.MustParseAddr("8.8.8.8")
	}
	if _, err := admitDestinationAddresses(tooMany); !errors.Is(err, errDestinationDenied) {
		t.Fatalf("raw oversized answer error = %v, want destination denial", err)
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
