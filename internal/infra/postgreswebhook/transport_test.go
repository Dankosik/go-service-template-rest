package postgreswebhook

import (
	"context"
	"encoding/hex"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"net/netip"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestWebhookDNSEvidenceVector(t *testing.T) {
	digest, err := DNSSetEvidence([]netip.Addr{netip.MustParseAddr("2001:db8::1"), netip.MustParseAddr("192.0.2.1")})
	if err != nil {
		t.Fatal(err)
	}
	if got := hex.EncodeToString(digest[:]); got != "b8885b9ec04d4deff5ba050bc10c60c280fdae844902444fe82e754cab46aaa4" {
		t.Fatalf("digest = %s", got)
	}
}

func TestWebhookBoundedOneSendTransport(t *testing.T) {
	var requests atomic.Int64
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		if request.Method != http.MethodPost || request.Header.Get("Webhook-Id") != "delivery-1" || request.Header.Get("Authorization") != "" || request.Header.Get("Cookie") != "" || request.Header.Get("Traceparent") != "" {
			t.Errorf("request = %s %#v", request.Method, request.Header)
		}
		response.Header().Set("Location", "/must-not-follow")
		response.WriteHeader(http.StatusTemporaryRedirect)
	}))
	defer server.Close()
	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	preparedURL, _ := url.Parse("https://127.0.0.1:443/hook")
	prepared := PreparedSend{
		URL: preparedURL, SelectedAddress: netip.MustParseAddr("8.8.8.8"), Signature: "v1,test",
		Attempt: ClaimedAttempt{Identity: AttemptIdentity{DeliveryID: "delivery-1"}, Body: []byte("{}"), ContentType: "application/json", AttemptedAt: time.Unix(1700000000, 0), Policy: DeliveryPolicy{AttemptTimeout: time.Second, ResponseHeaderTimeout: time.Second, ResponseHeaderBytes: 4096, ResponseBodyBytes: 32}},
	}
	serverTransport, ok := server.Client().Transport.(*http.Transport)
	if !ok {
		t.Fatalf("test server transport = %T", server.Client().Transport)
	}
	transport := newAttemptTransport("127.0.0.1", prepared.SelectedAddress, serverTransport.TLSClientConfig.RootCAs, time.Second, 4096)
	transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, serverURL.Host)
	}
	defer transport.CloseIdleConnections()
	wrote := false
	ctx := httptrace.WithClientTrace(t.Context(), &httptrace.ClientTrace{WroteRequest: func(httptrace.WroteRequestInfo) { wrote = true }})
	result, err := sendWithTransport(ctx, prepared, transport, &wrote)
	if err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 1 || result.Evidence.StatusCode != http.StatusTemporaryRedirect || !result.Evidence.MayHaveSent {
		t.Fatalf("result = %+v, requests = %d", result, requests.Load())
	}
}

func TestWebhookRequestContract(t *testing.T) {
	prepared := PreparedSend{Attempt: ClaimedAttempt{Identity: AttemptIdentity{DeliveryID: "delivery-1"}, Body: []byte("{}"), ContentType: "application/json", AttemptedAt: time.Unix(1700000000, 0)}, Signature: "v1,test"}
	prepared.URL, _ = parseWebhookURL("https://hooks.example.test/orders")
	request, err := webhookRequest(t.Context(), prepared)
	if err != nil {
		t.Fatal(err)
	}
	if request.Method != http.MethodPost || request.GetBody != nil || request.ContentLength != 2 || request.Header.Get("Webhook-Id") != "delivery-1" || request.Header.Get("User-Agent") != webhookUserAgent || request.Header.Get("Authorization") != "" || request.Header.Get("Idempotency-Key") != "" {
		t.Fatalf("request contract = %#v", request)
	}
}

func TestWebhookSendBoundsAndURLValidation(t *testing.T) {
	for _, raw := range []string{
		"http://hooks.example.test/orders",
		"https://user@hooks.example.test/orders",
		"https://hooks.example.test/orders?debug=1",
		"https://hooks.example.test:8443/orders",
	} {
		if _, err := parseWebhookURL(raw); err == nil {
			t.Fatalf("parseWebhookURL(%q) error = nil", raw)
		}
	}
	parsed, err := parseWebhookURL("https://hooks.example.test/orders")
	if err != nil || parsed.Port() != "443" {
		t.Fatalf("parseWebhookURL() = %v, %v", parsed, err)
	}

	invalid, err := Send(t.Context(), PreparedSend{})
	if err == nil || !invalid.Evidence.LocalDenial || !invalid.Evidence.DefinitelyNotSent {
		t.Fatalf("Send(invalid) = %+v, %v", invalid, err)
	}

	for _, test := range []struct {
		name    string
		header  string
		body    string
		wantErr string
	}{
		{name: "header limit", header: strings.Repeat("x", 128), wantErr: "header limit"},
		{name: "body limit", body: string(make([]byte, 33)), wantErr: "body limit"},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if test.header != "" {
					w.Header().Set("X-Long", test.header)
				}
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(test.body))
			}))
			defer server.Close()
			serverURL, err := url.Parse(server.URL)
			if err != nil {
				t.Fatal(err)
			}
			prepared := PreparedSend{
				URL: &url.URL{Scheme: "https", Host: "hooks.example.test"}, SelectedAddress: netip.MustParseAddr("8.8.8.8"), Signature: "v1,test",
				Attempt: ClaimedAttempt{Identity: AttemptIdentity{DeliveryID: "delivery-1"}, Body: []byte("{}"), ContentType: "application/json", AttemptedAt: time.Unix(1700000000, 0), Policy: DeliveryPolicy{ResponseHeaderBytes: 128, ResponseBodyBytes: 32}},
			}
			serverTransport, ok := server.Client().Transport.(*http.Transport)
			if !ok {
				t.Fatalf("test server transport = %T", server.Client().Transport)
			}
			transport := newAttemptTransport("127.0.0.1", prepared.SelectedAddress, serverTransport.TLSClientConfig.RootCAs, time.Second, 4096)
			transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, network, serverURL.Host)
			}
			defer transport.CloseIdleConnections()
			result, err := sendWithTransport(t.Context(), prepared, transport, new(bool))
			if err == nil || !strings.Contains(err.Error(), test.wantErr) || !result.Evidence.MayHaveSent {
				t.Fatalf("sendWithTransport() = %+v, %v", result, err)
			}
		})
	}
}
