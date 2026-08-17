package oidcjwt

// Proof for grpc.go at the interceptor level: identity parity with the HTTP
// boundary, credential removal from the handler-visible context, and the exact
// health allowlist. The same boundary over a real TLS connection is in
// grpc_tls_contract_test.go.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/reqctx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestGRPCIdentityParityAndMetadataRemoval(t *testing.T) {
	t.Parallel()
	now := testNow
	key := loadTestRSAKey(t, testSigningKey)
	verifier := newTestVerifier(t, key)
	token := signToken(t, key, "key-1", "at+jwt", validClaims(now))
	ctx := metadata.NewIncomingContext(t.Context(), metadata.Pairs("authorization", "Bearer "+token))

	handlerCalled := false
	_, err := verifier.UnaryInterceptor()(
		ctx,
		"request",
		&grpc.UnaryServerInfo{FullMethod: "/example.Service/Get"},
		func(ctx context.Context, _ any) (any, error) {
			handlerCalled = true
			principal, ok := reqctx.PrincipalFromContext(ctx)
			if !ok ||
				principal.Issuer != testIssuer ||
				principal.Subject != "opaque-subject" ||
				principal.ClientID != "client-1" {
				t.Fatalf("principal = (%+v, %v), want verified issuer, subject, and client ID", principal, ok)
			}
			incoming, _ := metadata.FromIncomingContext(ctx)
			if len(incoming.Get("authorization")) != 0 {
				t.Fatal("handler-visible authorization metadata was retained")
			}
			return "response", nil
		},
	)
	if err != nil || !handlerCalled {
		t.Fatalf("unary authentication error = %v, handlerCalled = %v", err, handlerCalled)
	}

	_, err = verifier.UnaryInterceptor()(
		t.Context(),
		"request",
		&grpc.UnaryServerInfo{FullMethod: "/example.Service/Get"},
		func(context.Context, any) (any, error) {
			t.Fatal("handler ran without a credential")
			return "unexpected", nil
		},
	)
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("missing credential status = %v, want Unauthenticated", status.Code(err))
	}
}

// TestGRPCLifetimeIsIndistinguishableFromAnInvalidCredential holds the half of
// [KindLifetime] that must not leak.
//
// The category exists so an operator can tell a provider configured for
// long-lived tokens from a signature that did not verify. A caller gets no such
// distinction: it learns that its credential was not accepted and nothing about
// which check said so, exactly as it would for KindInvalid. The HTTP side of the
// same property is a case in TestHTTPAuthnBoundary.
func TestGRPCLifetimeIsIndistinguishableFromAnInvalidCredential(t *testing.T) {
	t.Parallel()
	lifetime := status.Convert(grpcAuthenticationError(failure(KindLifetime)))
	invalid := status.Convert(grpcAuthenticationError(failure(KindInvalid)))
	if lifetime.Code() != invalid.Code() || lifetime.Message() != invalid.Message() {
		t.Fatalf(
			"lifetime status = (%v, %q), want the invalid-credential status (%v, %q)",
			lifetime.Code(),
			lifetime.Message(),
			invalid.Code(),
			invalid.Message(),
		)
	}
	if lifetime.Code() != codes.Unauthenticated {
		t.Fatalf("lifetime status code = %v, want %v", lifetime.Code(), codes.Unauthenticated)
	}
}

func TestGRPCHealthCheckIsPublicAndWatchIsProtected(t *testing.T) {
	t.Parallel()
	key := loadTestRSAKey(t, testSigningKey)
	verifier := newTestVerifier(t, key)
	ctx := metadata.NewIncomingContext(t.Context(), metadata.Pairs("authorization", "not-a-token"))

	called := false
	_, err := verifier.UnaryInterceptor()(
		ctx,
		nil,
		&grpc.UnaryServerInfo{FullMethod: healthpb.Health_Check_FullMethodName},
		func(ctx context.Context, _ any) (any, error) {
			called = true
			incoming, _ := metadata.FromIncomingContext(ctx)
			if len(incoming.Get("authorization")) != 0 {
				t.Fatal("health handler-visible authorization metadata was retained")
			}
			return "health", nil
		},
	)
	if err != nil || !called {
		t.Fatalf("health interception = (%v, %v), want public handler", called, err)
	}

	streamCalled := false
	err = verifier.StreamInterceptor()(
		nil,
		serverStreamWithContext{ctx: ctx},
		&grpc.StreamServerInfo{FullMethod: healthpb.Health_Watch_FullMethodName},
		func(_ any, _ grpc.ServerStream) error {
			streamCalled = true
			return nil
		},
	)
	if status.Code(err) != codes.Unauthenticated || streamCalled {
		t.Fatalf("health Watch interception = (called %v, status %v), want protected stream", streamCalled, status.Code(err))
	}
}

func TestGRPCAuthnBoundaryExactHealthAllowlist(t *testing.T) {
	t.Parallel()
	key := loadTestRSAKey(t, testSigningKey)
	verifier := newTestVerifier(t, key)
	const futureHealthMethod = "/grpc.health.v1.Health/Future"

	var unaryCalls int
	_, err := verifier.UnaryInterceptor()(
		t.Context(),
		nil,
		&grpc.UnaryServerInfo{FullMethod: futureHealthMethod},
		func(context.Context, any) (any, error) {
			unaryCalls++
			return nil, errors.New("unexpected future health handler call")
		},
	)
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("future health unary status = %v, want Unauthenticated", status.Code(err))
	}
	if unaryCalls != 0 {
		t.Fatalf("future health unary handler calls = %d, want zero", unaryCalls)
	}

	var streamCalls int
	err = verifier.StreamInterceptor()(
		nil,
		serverStreamWithContext{ctx: t.Context()},
		&grpc.StreamServerInfo{FullMethod: futureHealthMethod},
		func(any, grpc.ServerStream) error {
			streamCalls++
			return nil
		},
	)
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("future health stream status = %v, want Unauthenticated", status.Code(err))
	}
	if streamCalls != 0 {
		t.Fatalf("future health stream handler calls = %d, want zero", streamCalls)
	}
}

func TestGRPCStreamIdentityParityAndMetadataRemoval(t *testing.T) {
	t.Parallel()
	now := testNow
	key := loadTestRSAKey(t, testSigningKey)
	verifier := newTestVerifier(t, key)
	claims := validClaims(now)
	token := signToken(t, key, "key-1", "at+jwt", claims)
	source := metadata.NewIncomingContext(t.Context(), metadata.Pairs("authorization", "Bearer "+token))

	called := false
	err := verifier.StreamInterceptor()(
		nil,
		serverStreamWithContext{ctx: source},
		&grpc.StreamServerInfo{FullMethod: "/example.Service/Watch"},
		func(_ any, stream grpc.ServerStream) error {
			called = true
			deadline, ok := stream.Context().Deadline()
			wantDeadline := time.Unix(claims.Expires, 0).Add(ClockSkew)
			if !ok || !deadline.Equal(wantDeadline) {
				t.Fatalf("stream deadline = (%v, %v), want token expiry %v", deadline, ok, wantDeadline)
			}
			principal, ok := reqctx.PrincipalFromContext(stream.Context())
			if !ok || principal.Subject != "opaque-subject" {
				t.Fatalf("principal = (%+v, %v), want opaque subject", principal, ok)
			}
			incoming, _ := metadata.FromIncomingContext(stream.Context())
			if len(incoming.Get("authorization")) != 0 {
				t.Fatal("handler-visible authorization metadata was retained")
			}
			return nil
		},
	)
	if err != nil || !called {
		t.Fatalf("stream authentication error = %v, handlerCalled = %v", err, called)
	}

	err = verifier.StreamInterceptor()(
		nil,
		serverStreamWithContext{ctx: t.Context()},
		&grpc.StreamServerInfo{FullMethod: "/example.Service/Watch"},
		func(any, grpc.ServerStream) error {
			t.Fatal("stream handler ran without a credential")
			return nil
		},
	)
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("missing credential status = %v, want Unauthenticated", status.Code(err))
	}
}

func TestGRPCStreamMessageOperationsHonorAuthenticatedContext(t *testing.T) {
	t.Parallel()
	expired, cancelExpired := context.WithDeadline(t.Context(), time.Now().Add(-time.Second))
	defer cancelExpired()
	var recvCalls, sendCalls int
	underlying := &scriptedServerStream{
		ctx: t.Context(),
		recv: func(any) error {
			recvCalls++
			return nil
		},
		send: func(any) error {
			sendCalls++
			return nil
		},
	}
	stream := serverStreamWithContext{ServerStream: underlying, ctx: expired}
	if err := stream.RecvMsg(new(any)); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("RecvMsg() error = %v, want authenticated-context deadline", err)
	}
	if err := stream.SendMsg("message"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("SendMsg() error = %v, want authenticated-context deadline", err)
	}
	if recvCalls != 0 || sendCalls != 0 {
		t.Fatalf("underlying message calls = (recv %d, send %d), want zero after expiry", recvCalls, sendCalls)
	}

	active, cancelActive := context.WithTimeout(t.Context(), 10*time.Millisecond)
	defer cancelActive()
	started := make(chan struct{})
	release := make(chan struct{})
	underlying.recv = func(any) error {
		recvCalls++
		close(started)
		<-release
		return nil
	}
	stream.ctx = active
	result := make(chan error, 1)
	go func() {
		result <- stream.RecvMsg(new(any))
	}()
	<-started
	<-active.Done()
	close(release)
	if err := <-result; !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("RecvMsg() crossing deadline = %v, want authenticated-context deadline", err)
	}
}

func TestGRPCProviderTimeoutIsUnavailable(t *testing.T) {
	t.Parallel()
	now := testNow
	first := loadTestRSAKey(t, testSigningKey)
	second := loadTestRSAKey(t, testRotatedKey)
	client := &scriptedClient{responses: append(initialResponses(t, first), scriptedResponse{
		err: context.DeadlineExceeded,
	})}
	verifier := requireTestVerifier(t, testVerifierOptions{now: newTestClock(now).now, client: client})
	token := signToken(t, second, "key-2", "at+jwt", validClaims(now))
	ctx := metadata.NewIncomingContext(t.Context(), metadata.Pairs("authorization", "Bearer "+token))

	_, err := verifier.UnaryInterceptor()(
		ctx,
		nil,
		&grpc.UnaryServerInfo{FullMethod: "/example.Service/Get"},
		func(context.Context, any) (any, error) {
			t.Fatal("handler ran without current authentication trust")
			return nil, errors.New("handler should not run")
		},
	)
	if code := status.Code(err); code != codes.Unavailable {
		t.Fatalf("provider timeout status = %v, want Unavailable; error = %v", code, err)
	}
}

type scriptedServerStream struct {
	grpc.ServerStream

	ctx  context.Context //nolint:containedctx // grpc.ServerStream's test double must return the source stream context.
	recv func(any) error
	send func(any) error
}

func (s *scriptedServerStream) Context() context.Context {
	return s.ctx
}

func (s *scriptedServerStream) RecvMsg(message any) error {
	return s.recv(message)
}

func (s *scriptedServerStream) SendMsg(message any) error {
	return s.send(message)
}
