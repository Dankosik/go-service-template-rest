package bearerauthn

import (
	"context"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/reqctx"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
)

type fakeVerifier struct {
	result   Result
	err      error
	calls    atomic.Int64
	inFlight atomic.Int64
	peak     atomic.Int64
	entered  chan struct{}
	block    <-chan struct{}
}

func (f *fakeVerifier) Verify(ctx context.Context, _ string) (Result, error) {
	f.calls.Add(1)
	current := f.inFlight.Add(1)
	defer f.inFlight.Add(-1)
	for {
		previous := f.peak.Load()
		if current <= previous || f.peak.CompareAndSwap(previous, current) {
			break
		}
	}
	if f.entered != nil {
		select {
		case f.entered <- struct{}{}:
		default:
		}
	}
	if f.block != nil {
		select {
		case <-f.block:
		case <-ctx.Done():
			return Result{}, fmt.Errorf("wait for test barrier: %w", ctx.Err())
		}
	}
	if f.err != nil {
		return Result{}, f.err
	}
	if f.result.Principal.Subject == "" && f.result.ExpiresAt.IsZero() {
		return Result{
			Principal: reqctx.Principal{Issuer: "https://issuer.example.com", Subject: "subject-1"},
			ExpiresAt: time.Unix(1_900_003_600, 0),
		}, nil
	}
	return f.result, nil
}

func (f *fakeVerifier) Close() {}

func newTestRuntime(t *testing.T, verifier *fakeVerifier) *Runtime {
	t.Helper()
	runtime, err := New(verifier, nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return runtime
}

func bearerAuthInput(request *http.Request) *openapi3filter.AuthenticationInput {
	return &openapi3filter.AuthenticationInput{
		SecuritySchemeName: "bearerAuth",
		SecurityScheme:     &openapi3.SecurityScheme{Type: "http", Scheme: "bearer"},
		RequestValidationInput: &openapi3filter.RequestValidationInput{
			Request: request,
		},
	}
}

func requireKind(t *testing.T, err error, want Kind) {
	t.Helper()
	got, ok := KindOf(err)
	if !ok || got != want {
		t.Fatalf("KindOf(%v) = %v, %v; want %v, true", err, got, ok, want)
	}
}
