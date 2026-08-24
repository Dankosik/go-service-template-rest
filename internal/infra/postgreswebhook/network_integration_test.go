//go:build integration

package postgreswebhook

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	standardwebhooks "github.com/standard-webhooks/standard-webhooks/libraries/go"
	"golang.org/x/net/dns/dnsmessage"
)

func TestWebhookNetworkSecurity(t *testing.T) {
	manifest := webhookSecretManifest(t)
	tests := []struct {
		name      string
		addresses []netip.Addr
		wantDeny  bool
	}{
		{name: "public", addresses: []netip.Addr{netip.MustParseAddr("8.8.8.8")}},
		{name: "private", addresses: []netip.Addr{netip.MustParseAddr("127.0.0.1")}, wantDeny: true},
		{name: "mixed", addresses: []netip.Addr{netip.MustParseAddr("8.8.8.8"), netip.MustParseAddr("10.0.0.1")}, wantDeny: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resolver := webhookResolver(t, "hooks.test", test.addresses)
			attempt := webhookNetworkAttempt()
			ctx, cancel := context.WithDeadline(t.Context(), attempt.Deadline)
			defer cancel()
			prepared, err := prepareSend(ctx, resolver, attempt, manifest)
			if test.wantDeny {
				if !errors.Is(err, errDestinationDenied) {
					t.Fatalf("prepareSend() error = %v, want destination denial", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if prepared.SelectedAddress != test.addresses[0] {
				t.Fatalf("prepared address = %+v", prepared)
			}
		})
	}
}

func TestWebhookBoundedAttempt(t *testing.T) {
	attempt := webhookNetworkAttempt()
	prepared := preparedSend{
		Attempt: attempt, URL: mustWebhookURL(t, attempt.URL),
		SelectedAddress: netip.MustParseAddr("127.0.0.1"), Signature: "v1,test",
	}
	ctx, cancel := context.WithDeadline(t.Context(), attempt.Deadline)
	defer cancel()
	started := time.Now()
	result, err := send(ctx, prepared)
	if !errors.Is(err, errDestinationDenied) {
		t.Fatalf("send() error = %v, want destination denial", err)
	}
	if !result.Evidence.DefinitelyNotSent || result.Evidence.MayHaveSent || time.Since(started) > time.Second {
		t.Fatalf("bounded denial result = %+v", result)
	}
}

func TestWebhookComposedTransportContract(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	body := []byte(`{"type":"order.created","data":{"id":"ord-1"}}`)
	attemptedAt := time.Unix(1_700_000_000, 0).UTC()
	verifier, err := standardwebhooks.NewWebhookRaw(key)
	if err != nil {
		t.Fatal(err)
	}
	var delivered atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/deliver", func(response http.ResponseWriter, request *http.Request) {
		got, readErr := io.ReadAll(request.Body)
		if readErr != nil || string(got) != string(body) {
			t.Errorf("receiver body = %q, %v", got, readErr)
			response.WriteHeader(http.StatusBadRequest)
			return
		}
		if request.Header.Get("Content-Type") != "application/json" || verifier.VerifyIgnoringTimestamp(got, request.Header) != nil {
			t.Errorf("receiver headers = %v", request.Header)
			response.WriteHeader(http.StatusBadRequest)
			return
		}
		delivered.Add(1)
		response.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/redirect", func(response http.ResponseWriter, request *http.Request) {
		http.Redirect(response, request, "/deliver", http.StatusTemporaryRedirect)
	})
	server := httptest.NewTLSServer(mux)
	t.Cleanup(server.Close)
	transport := server.Client().Transport.(*http.Transport).Clone()
	t.Cleanup(transport.CloseIdleConnections)
	signature, err := signV1("whd_test", attemptedAt, body, [][]byte{key})
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(server.URL + "/deliver")
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	prepared := preparedSend{
		Attempt: deliveryAttempt{ID: "whd_test", Body: body, AttemptedAt: attemptedAt, Deadline: deadline},
		URL:     parsed, SelectedAddress: netip.MustParseAddr("8.8.8.8"), Signature: signature,
	}
	ctx, cancel := context.WithDeadline(t.Context(), deadline)
	defer cancel()
	result, err := sendWithTransport(ctx, prepared, transport, new(bool))
	if err != nil || result.Evidence.StatusCode != http.StatusNoContent || delivered.Load() != 1 {
		t.Fatalf("delivery result = %+v, %v, count=%d", result, err, delivered.Load())
	}
	prepared.URL.Path = "/redirect"
	result, err = sendWithTransport(ctx, prepared, transport, new(bool))
	if err != nil || result.Evidence.StatusCode != http.StatusTemporaryRedirect || delivered.Load() != 1 {
		t.Fatalf("redirect result = %+v, %v, count=%d", result, err, delivered.Load())
	}
}

func webhookNetworkAttempt() deliveryAttempt {
	attemptedAt := time.Now()
	return deliveryAttempt{
		ID: "whd_delivery-01", OwnerScope: "owner-a", ReceiverID: "receiver-a",
		URL: "https://hooks.test/deliver", Body: []byte(`{"id":"evt-01"}`),
		AttemptedAt: attemptedAt, Deadline: attemptedAt.Add(time.Second), KeyReference: "key-a",
	}
}

func webhookSecretManifest(t *testing.T) *SecretManifest {
	t.Helper()
	secret := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	manifest, err := ParseSecretManifest(`{"entries":[{"owner_scope":"owner-a","receiver_id":"receiver-a","key_reference":"key-a","secret":"whsec_` + secret + `"}]}`)
	if err != nil {
		t.Fatal(err)
	}
	return manifest
}

func webhookResolver(t *testing.T, hostname string, addresses []netip.Addr) *net.Resolver {
	t.Helper()
	packetConn, err := (&net.ListenConfig{}).ListenPacket(t.Context(), "udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		buffer := make([]byte, 1232)
		for {
			size, peer, readErr := packetConn.ReadFrom(buffer)
			if readErr != nil {
				done <- readErr
				return
			}
			response, responseErr := webhookDNSResponse(buffer[:size], hostname, addresses)
			if responseErr != nil {
				done <- responseErr
				return
			}
			if _, writeErr := packetConn.WriteTo(response, peer); writeErr != nil {
				done <- writeErr
				return
			}
		}
	}()
	t.Cleanup(func() {
		_ = packetConn.Close()
		if err := <-done; err != nil && !errors.Is(err, net.ErrClosed) {
			t.Errorf("DNS server: %v", err)
		}
	})
	return &net.Resolver{PreferGo: true, Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "udp", packetConn.LocalAddr().String())
	}}
}

func webhookDNSResponse(query []byte, hostname string, addresses []netip.Addr) ([]byte, error) {
	var parser dnsmessage.Parser
	header, err := parser.Start(query)
	if err != nil {
		return nil, fmt.Errorf("parse DNS header: %w", err)
	}
	question, err := parser.Question()
	if err != nil {
		return nil, fmt.Errorf("parse DNS question: %w", err)
	}
	builder := dnsmessage.NewBuilder(nil, dnsmessage.Header{ID: header.ID, Response: true, RecursionDesired: header.RecursionDesired, RecursionAvailable: true})
	builder.EnableCompression()
	if err := builder.StartQuestions(); err != nil {
		return nil, fmt.Errorf("start DNS questions: %w", err)
	}
	if err := builder.Question(question); err != nil {
		return nil, fmt.Errorf("write DNS question: %w", err)
	}
	if err := builder.StartAnswers(); err != nil {
		return nil, fmt.Errorf("start DNS answers: %w", err)
	}
	if question.Name.String() == hostname+"." {
		for _, address := range addresses {
			switch {
			case address.Is4() && question.Type == dnsmessage.TypeA:
				if err := builder.AResource(dnsmessage.ResourceHeader{Name: question.Name, Type: dnsmessage.TypeA, Class: dnsmessage.ClassINET, TTL: 1}, dnsmessage.AResource{A: address.As4()}); err != nil {
					return nil, fmt.Errorf("write DNS A answer: %w", err)
				}
			case address.Is6() && question.Type == dnsmessage.TypeAAAA:
				if err := builder.AAAAResource(dnsmessage.ResourceHeader{Name: question.Name, Type: dnsmessage.TypeAAAA, Class: dnsmessage.ClassINET, TTL: 1}, dnsmessage.AAAAResource{AAAA: address.As16()}); err != nil {
					return nil, fmt.Errorf("write DNS AAAA answer: %w", err)
				}
			}
		}
	}
	response, err := builder.Finish()
	if err != nil {
		return nil, fmt.Errorf("finish DNS response: %w", err)
	}
	return response, nil
}
