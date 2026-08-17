//go:build integration

package integration_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgreswebhook"
	"golang.org/x/net/dns/dnsmessage"
)

func TestWebhookNetworkSecurity(t *testing.T) {
	manifest := webhookManifest(t, 1, "owner-a", "dest-a", "key-a")
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
			prepared, err := postgreswebhook.PrepareSend(ctx, resolver, attempt, manifest)
			if test.wantDeny {
				if !errors.Is(err, postgreswebhook.ErrDestinationDenied) {
					t.Fatalf("PrepareSend() error = %v, want ErrDestinationDenied", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if prepared.SelectedAddress != test.addresses[0] || prepared.DNSSetDigest == ([32]byte{}) {
				t.Fatalf("prepared DNS evidence = %+v", prepared)
			}
		})
	}
}

func TestWebhookBoundedAttempt(t *testing.T) {
	attempt := webhookNetworkAttempt()
	prepared := postgreswebhook.PreparedSend{
		Attempt: attempt, URL: mustWebhookURL(t, attempt.URL),
		SelectedAddress: netip.MustParseAddr("127.0.0.1"), Signature: "v1,test",
	}
	ctx, cancel := context.WithDeadline(t.Context(), attempt.Deadline)
	defer cancel()
	started := time.Now()
	result, err := postgreswebhook.Send(ctx, prepared)
	if !errors.Is(err, postgreswebhook.ErrDestinationDenied) {
		t.Fatalf("Send() error = %v, want ErrDestinationDenied", err)
	}
	if !result.Evidence.DefinitelyNotSent || result.Evidence.MayHaveSent || time.Since(started) > attempt.Policy.AttemptTimeout {
		t.Fatalf("bounded denial result = %+v", result)
	}
}

func TestWebhookRetrySignatureIdentity(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	body := []byte(`{"event":"order.created","id":"evt-01"}`)
	firstAt := time.Unix(1700000000, 0)
	secondAt := firstAt.Add(time.Second)
	first, _, err := postgreswebhook.SignV1("delivery-01", firstAt, body, []postgreswebhook.SigningKey{{Reference: "key-a", Bytes: key}})
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := postgreswebhook.SignV1("delivery-01", secondAt, body, []postgreswebhook.SigningKey{{Reference: "key-a", Bytes: key}})
	if err != nil {
		t.Fatal(err)
	}
	if first == second || !postgreswebhook.VerifyV1("delivery-01", "1700000000", body, first, [][]byte{key}) || !postgreswebhook.VerifyV1("delivery-01", "1700000001", body, second, [][]byte{key}) {
		t.Fatal("retry identity did not retain body/delivery while refreshing attempt time and signature")
	}
}

func webhookNetworkAttempt() postgreswebhook.ClaimedAttempt {
	attemptedAt := time.Now()
	return postgreswebhook.ClaimedAttempt{
		Identity:      postgreswebhook.AttemptIdentity{OwnerScope: "owner-a", DeliveryID: "delivery-01", AttemptID: "attempt-01", Fence: 1},
		DestinationID: "dest-a", URL: "https://hooks.test/deliver", Body: []byte(`{"id":"evt-01"}`), ContentType: "application/json",
		AttemptedAt: attemptedAt, Deadline: attemptedAt.Add(time.Second), KeyReference: "key-a", ManifestRevision: 1,
		Policy: postgreswebhook.DeliveryPolicy{AttemptTimeout: time.Second, ResponseHeaderTimeout: 500 * time.Millisecond, ResponseHeaderBytes: 4096, ResponseBodyBytes: 4096},
	}
}

func mustWebhookURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
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
