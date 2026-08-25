package bearerauthn

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/example/go-service-template-rest/internal/reqctx"
	"go.opentelemetry.io/otel/metric"
)

// Fixed safe defaults shared by every concrete trust engine. Issuer, audience,
// and provider bounds remain engine-specific.
const (
	MaxTokenBytes = 32 << 10
	ClockSkew     = 30 * time.Second
)

// Verifier is the consumer-owned trust-engine contract. It verifies one already
// parsed bearer value and returns a principal plus expiry, or one sanitized
// invalid, unavailable, or caller-context failure.
type Verifier interface {
	Verify(ctx context.Context, token string) (Result, error)
	Close()
}

// Result is the admitted identity and expiry a successful verification publishes.
type Result struct {
	Principal reqctx.Principal
	ExpiresAt time.Time
}

// Runtime is the shared HTTP and gRPC authentication surface bootstrap consumes.
type Runtime struct {
	verifier Verifier
	metrics  authnMetrics
}

// New builds a runtime around one concrete verifier. Ownership of verifier
// transfers only after construction succeeds.
func New(verifier Verifier, meterProvider metric.MeterProvider) (*Runtime, error) {
	if verifier == nil {
		return nil, errors.New("bearer authentication verifier is required")
	}
	return &Runtime{verifier: verifier, metrics: newAuthnMetrics(meterProvider)}, nil
}

// Close delegates idempotent concrete-engine cleanup.
func (r *Runtime) Close() {
	if r == nil || r.verifier == nil {
		return
	}
	r.verifier.Close()
}

type transport string

const (
	transportHTTP transport = "http"
	transportGRPC transport = "grpc"
)

func (r *Runtime) verifyCredential(ctx context.Context, values []string, carrier transport) (Result, error) {
	token, err := bearerToken(values)
	if err != nil {
		return Result{}, r.recordRejection(ctx, carrier, err)
	}
	verified, err := r.verifier.Verify(ctx, token)
	if err == nil && !validResult(verified) {
		verified = Result{}
		err = failure(KindUnavailable)
	}
	return verified, r.recordRejection(ctx, carrier, sanitizeVerifierError(err))
}

func validResult(result Result) bool {
	return strings.TrimSpace(result.Principal.Issuer) != "" &&
		!result.ExpiresAt.IsZero() &&
		(strings.TrimSpace(result.Principal.Subject) != "" || strings.TrimSpace(result.Principal.ClientID) != "")
}

func (r *Runtime) recordRejection(ctx context.Context, carrier transport, err error) error {
	r.metrics.recordVerification(ctx, carrier, err)
	return err
}

func sanitizeVerifierError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if _, ok := KindOf(err); ok {
		return err
	}
	return fmt.Errorf("verify access token: %w", failure(KindInvalid))
}
