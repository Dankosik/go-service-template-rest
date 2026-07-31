package oidcjwt

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/reqctx"
	"google.golang.org/grpc"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestGRPCIdentityParityAndMetadataRemoval(t *testing.T) {
	now := time.Unix(1_900_000_000, 0)
	key := loadTestRSAKey(t, "test-key-1.pem")
	verifier := newTestVerifier(t, now, key)
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
			if !ok || principal.Subject != "opaque-subject" {
				t.Fatalf("principal = (%+v, %v), want opaque subject", principal, ok)
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
	if status.Code(err).String() != "Unauthenticated" {
		t.Fatalf("missing credential status = %v, want Unauthenticated", status.Code(err))
	}
}

func TestGRPCHealthIsPublicAndCredentialIsRemoved(t *testing.T) {
	now := time.Unix(1_900_000_000, 0)
	key := loadTestRSAKey(t, "test-key-1.pem")
	verifier := newTestVerifier(t, now, key)
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
		testServerStream{ctx: ctx},
		&grpc.StreamServerInfo{FullMethod: healthpb.Health_Watch_FullMethodName},
		func(_ any, stream grpc.ServerStream) error {
			streamCalled = true
			incoming, _ := metadata.FromIncomingContext(stream.Context())
			if len(incoming.Get("authorization")) != 0 {
				t.Fatal("health stream handler-visible authorization metadata was retained")
			}
			return nil
		},
	)
	if err != nil || !streamCalled {
		t.Fatalf("health stream interception = (%v, %v), want public handler", streamCalled, err)
	}
}

func TestGRPCAuthnBoundaryExactHealthAllowlist(t *testing.T) {
	now := time.Unix(1_900_000_000, 0)
	key := loadTestRSAKey(t, "test-key-1.pem")
	verifier := newTestVerifier(t, now, key)
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
	if status.Code(err).String() != "Unauthenticated" {
		t.Fatalf("future health unary status = %v, want Unauthenticated", status.Code(err))
	}
	if unaryCalls != 0 {
		t.Fatalf("future health unary handler calls = %d, want zero", unaryCalls)
	}

	var streamCalls int
	err = verifier.StreamInterceptor()(
		nil,
		testServerStream{ctx: t.Context()},
		&grpc.StreamServerInfo{FullMethod: futureHealthMethod},
		func(any, grpc.ServerStream) error {
			streamCalls++
			return nil
		},
	)
	if status.Code(err).String() != "Unauthenticated" {
		t.Fatalf("future health stream status = %v, want Unauthenticated", status.Code(err))
	}
	if streamCalls != 0 {
		t.Fatalf("future health stream handler calls = %d, want zero", streamCalls)
	}
}

func TestGRPCStreamIdentityParityAndMetadataRemoval(t *testing.T) {
	now := time.Unix(1_900_000_000, 0)
	key := loadTestRSAKey(t, "test-key-1.pem")
	verifier := newTestVerifier(t, now, key)
	token := signToken(t, key, "key-1", "at+jwt", validClaims(now))
	source := metadata.NewIncomingContext(t.Context(), metadata.Pairs("authorization", "Bearer "+token))

	called := false
	err := verifier.StreamInterceptor()(
		nil,
		testServerStream{ctx: source},
		&grpc.StreamServerInfo{FullMethod: "/example.Service/Watch"},
		func(_ any, stream grpc.ServerStream) error {
			called = true
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
		testServerStream{ctx: t.Context()},
		&grpc.StreamServerInfo{FullMethod: "/example.Service/Watch"},
		func(any, grpc.ServerStream) error {
			t.Fatal("stream handler ran without a credential")
			return nil
		},
	)
	if status.Code(err).String() != "Unauthenticated" {
		t.Fatalf("missing credential status = %v, want Unauthenticated", status.Code(err))
	}
}

type testServerStream struct {
	grpc.ServerStream

	ctx context.Context //nolint:containedctx // grpc.ServerStream requires Context to expose the fixed test RPC context.
}

func (s testServerStream) Context() context.Context {
	return s.ctx
}
