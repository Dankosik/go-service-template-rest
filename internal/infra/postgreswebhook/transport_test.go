package postgreswebhook

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"net/netip"
	"net/url"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestWebhookPrepareSendUsesCallerAttemptDeadline(t *testing.T) {
	secret := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	manifest, err := ParseSecretManifest(`{"revision":1,"entries":[{"owner_scope":"owner-a","destination_id":"dest-a","key_reference":"key-a","secret":"whsec_` + secret + `"}]}`)
	if err != nil {
		t.Fatal(err)
	}
	resolver := &net.Resolver{PreferGo: true, Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}}
	attemptedAt := time.Now()
	attempt := ClaimedAttempt{
		Identity: AttemptIdentity{OwnerScope: "owner-a", DeliveryID: "delivery-a"}, DestinationID: "dest-a",
		URL: "https://hooks.example.test/orders", Body: []byte(`{}`), KeyReference: "key-a", ManifestRevision: 1,
		AttemptedAt: attemptedAt, Deadline: attemptedAt.Add(50 * time.Millisecond),
	}
	ctx, cancel := context.WithDeadline(t.Context(), attempt.Deadline)
	defer cancel()
	started := time.Now()
	if _, err := PrepareSend(ctx, resolver, attempt, manifest); err == nil {
		t.Fatal("PrepareSend() error = nil")
	}
	if elapsed := time.Since(started); elapsed > 500*time.Millisecond {
		t.Fatalf("DNS exceeded total attempt deadline: %s", elapsed)
	}
}

func TestWebhookDNSEvidenceVector(t *testing.T) {
	digest, err := DNSSetEvidence([]netip.Addr{netip.MustParseAddr("2001:db8::1"), netip.MustParseAddr("192.0.2.1")})
	if err != nil {
		t.Fatal(err)
	}
	if got := hex.EncodeToString(digest[:]); got != "b8885b9ec04d4deff5ba050bc10c60c280fdae844902444fe82e754cab46aaa4" {
		t.Fatalf("digest = %s", got)
	}
}

func TestWebhookAddressFallbackAuthorizesEveryCandidate(t *testing.T) {
	first := netip.MustParseAddr("8.8.8.8")
	second := netip.MustParseAddr("1.1.1.1")
	prepared := PreparedSend{Addresses: []netip.Addr{first, second}, SelectedAddress: first}
	var authorized, sent []netip.Addr
	result, err := tryPreparedAddresses(t.Context(), prepared, func(candidate PreparedSend) error {
		authorized = append(authorized, candidate.SelectedAddress)
		return nil
	}, func(_ context.Context, candidate PreparedSend) (SendResult, error) {
		sent = append(sent, candidate.SelectedAddress)
		if candidate.SelectedAddress == first {
			return SendResult{Evidence: TransportEvidence{DefinitelyNotSent: true}}, io.EOF
		}
		return SendResult{Evidence: TransportEvidence{StatusCode: http.StatusAccepted, MayHaveSent: true}}, nil
	})
	if err != nil || result.Evidence.StatusCode != http.StatusAccepted || !slices.Equal(authorized, []netip.Addr{first, second}) || !slices.Equal(sent, authorized) {
		t.Fatalf("fallback = %+v, %v; authorized=%v sent=%v", result, err, authorized, sent)
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

func TestWebhookReceiverAmbiguityAndIdentity(t *testing.T) {
	type received struct {
		body      string
		delivery  string
		timestamp string
		signature string
	}
	receivedRequests := make(chan received, 2)
	var requests atomic.Int64
	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Errorf("read receiver body: %v", err)
		}
		if request.Method != http.MethodPost || request.Header.Get("Authorization") != "" || request.Header.Get("Cookie") != "" || request.Header.Get("Traceparent") != "" {
			t.Errorf("request = %s %#v", request.Method, request.Header)
		}
		receivedRequests <- received{body: string(body), delivery: request.Header.Get("Webhook-Id"), timestamp: request.Header.Get("Webhook-Timestamp"), signature: request.Header.Get("Webhook-Signature")}
		if requests.Add(1) == 1 {
			hijacker, ok := response.(http.Hijacker)
			if !ok {
				t.Error("receiver does not support connection hijacking")
				return
			}
			connection, _, err := hijacker.Hijack()
			if err != nil {
				t.Errorf("hijack receiver connection: %v", err)
				return
			}
			_ = connection.Close()
			return
		}
		response.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()
	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	serverTransport, ok := server.Client().Transport.(*http.Transport)
	if !ok {
		t.Fatalf("server transport = %T, want *http.Transport", server.Client().Transport)
	}
	key := []byte("0123456789abcdef0123456789abcdef")
	body := []byte(`{"event":"order.created","id":"evt-01"}`)
	send := func(at time.Time) (SendResult, error) {
		t.Helper()
		signature, _, err := SignV1("delivery-01", at, body, []SigningKey{{Reference: "key-a", Bytes: key}})
		if err != nil {
			t.Fatal(err)
		}
		attempt := ClaimedAttempt{
			Identity: AttemptIdentity{DeliveryID: "delivery-01"}, Body: body, ContentType: "application/json", AttemptedAt: at,
			Deadline: time.Now().Add(time.Second), Policy: DeliveryPolicy{AttemptTimeout: time.Second, ResponseHeaderTimeout: time.Second, ResponseHeaderBytes: 4096, ResponseBodyBytes: 32},
		}
		prepared := PreparedSend{Attempt: attempt, URL: &url.URL{Scheme: "https", Host: "hooks.example.test"}, SelectedAddress: netip.MustParseAddr("8.8.8.8"), Signature: signature}
		transport := newAttemptTransport("127.0.0.1", prepared.SelectedAddress, serverTransport.TLSClientConfig.RootCAs, time.Second, 4096)
		transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, network, serverURL.Host)
		}
		defer transport.CloseIdleConnections()
		ctx, cancel := context.WithDeadline(t.Context(), attempt.Deadline)
		defer cancel()
		wrote := false
		ctx = httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{WroteRequest: func(httptrace.WroteRequestInfo) { wrote = true }})
		return sendWithTransport(ctx, prepared, transport, &wrote)
	}

	firstAt := time.Unix(1700000000, 0)
	first, firstErr := send(firstAt)
	second, secondErr := send(firstAt.Add(time.Second))
	if firstErr == nil || !first.Evidence.MayHaveSent || first.Evidence.DefinitelyNotSent {
		t.Fatalf("lost response = %+v, %v", first, firstErr)
	}
	if secondErr != nil || second.Evidence.StatusCode != http.StatusAccepted || !second.Evidence.MayHaveSent {
		t.Fatalf("retry response = %+v, %v", second, secondErr)
	}
	firstRequest, secondRequest := <-receivedRequests, <-receivedRequests
	if requests.Load() != 2 || firstRequest.delivery != "delivery-01" || secondRequest.delivery != firstRequest.delivery || firstRequest.body != string(body) || secondRequest.body != firstRequest.body || firstRequest.timestamp == secondRequest.timestamp || firstRequest.signature == secondRequest.signature {
		t.Fatalf("receiver requests = %#v, %#v", firstRequest, secondRequest)
	}
	if !VerifyV1(firstRequest.delivery, firstRequest.timestamp, []byte(firstRequest.body), firstRequest.signature, [][]byte{key}) || !VerifyV1(secondRequest.delivery, secondRequest.timestamp, []byte(secondRequest.body), secondRequest.signature, [][]byte{key}) {
		t.Fatal("receiver could not verify retry signatures")
	}
}

func TestWebhookRequestContract(t *testing.T) {
	prepared := PreparedSend{Attempt: ClaimedAttempt{Identity: AttemptIdentity{DeliveryID: "delivery-1"}, Body: []byte("{}"), ContentType: "application/json", AttemptedAt: time.Unix(1700000000, 0)}, Signature: "v1,test"}
	prepared.URL, _ = parseWebhookURL("https://hooks.example.test/orders")
	request, err := webhookRequest(t.Context(), prepared)
	if err != nil {
		t.Fatal(err)
	}
	if request.Method != http.MethodPost || request.Host != "hooks.example.test:443" || request.GetBody != nil || request.ContentLength != 2 || request.Header.Get("Webhook-Id") != "delivery-1" || request.Header.Get("User-Agent") != webhookUserAgent || request.Header.Get("Authorization") != "" || request.Header.Get("Idempotency-Key") != "" {
		t.Fatalf("request contract = %#v", request)
	}
	transport := newAttemptTransport("hooks.example.test", netip.MustParseAddr("8.8.8.8"), nil, time.Second, 4096)
	defer transport.CloseIdleConnections()
	if transport.Proxy != nil || transport.TLSClientConfig.ServerName != "hooks.example.test" || transport.TLSClientConfig.MinVersion != tls.VersionTLS13 || minimumTLSVersion("1.2") != tls.VersionTLS12 {
		t.Fatalf("transport authority = proxy:%t server:%q min:%d", transport.Proxy != nil, transport.TLSClientConfig.ServerName, transport.TLSClientConfig.MinVersion)
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
		name   string
		header string
		body   string
	}{
		{name: "header limit", header: strings.Repeat("x", 128)},
		{name: "body limit", body: string(make([]byte, 33))},
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
			if !errors.Is(err, ErrResponseLimit) || !result.Evidence.MayHaveSent {
				t.Fatalf("sendWithTransport() = %+v, %v", result, err)
			}
		})
	}
}

func TestWebhookCertificateFailureIsLocalDenial(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer server.Close()
	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	prepared := PreparedSend{
		URL: &url.URL{Scheme: "https", Host: "hooks.example.test:443"}, SelectedAddress: netip.MustParseAddr("8.8.8.8"), Signature: "v1,test",
		Attempt: ClaimedAttempt{Identity: AttemptIdentity{DeliveryID: "delivery-1"}, Body: []byte("{}"), ContentType: "application/json", AttemptedAt: time.Unix(1700000000, 0), Policy: DeliveryPolicy{ResponseHeaderBytes: 4096, ResponseBodyBytes: 32}},
	}
	transport := newAttemptTransport("hooks.example.test", prepared.SelectedAddress, nil, time.Second, 4096)
	transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, serverURL.Host)
	}
	defer transport.CloseIdleConnections()
	result, err := sendWithTransport(t.Context(), prepared, transport, new(bool))
	if err == nil || !result.Evidence.DefinitelyNotSent || !result.Evidence.LocalDenial || result.Evidence.MayHaveSent {
		t.Fatalf("certificate failure = %+v, %v", result, err)
	}
}
