package httpclient

import (
	"context"
	"net/http"
	"strings"

	"github.com/example/go-service-template-rest/internal/reqctx"
	"go.opentelemetry.io/otel/propagation"
)

const requestIDHeader = reqctx.RequestIDHeader

var reservedPropagationHeaders = [...]string{
	"traceparent",
	"tracestate",
	"baggage",
	requestIDHeader,
}

// PropagationPolicy selects which correlation fields one fixed target may
// receive. The zero value is deliberately fail-closed.
type PropagationPolicy uint8

const (
	// PropagationNone keeps local client telemetry but emits no remote
	// correlation fields.
	PropagationNone PropagationPolicy = iota
	// PropagationTraceContext emits only W3C Trace Context.
	PropagationTraceContext
	// PropagationTrustedService emits W3C Trace Context and the valid request ID
	// already present in the call context.
	PropagationTrustedService
)

func (p PropagationPolicy) valid() bool {
	return p <= PropagationTrustedService
}

type propagationSanitizer struct {
	base http.RoundTripper
}

// RoundTrip removes every caller-supplied correlation field from a request
// copy. The inner OpenTelemetry transport then injects only the selected
// policy for the concrete attempt span.
func (t propagationSanitizer) RoundTrip(request *http.Request) (*http.Response, error) {
	attempt := request.Clone(request.Context())
	if attempt.Header == nil {
		attempt.Header = make(http.Header)
	}
	removeReservedHeaders(attempt.Header)
	removeReservedHeaders(attempt.Trailer)

	//nolint:wrapcheck // The owned outer client adds operation context.
	return t.base.RoundTrip(attempt)
}

type policyPropagator struct {
	policy PropagationPolicy
}

func (p policyPropagator) Inject(ctx context.Context, carrier propagation.TextMapCarrier) {
	if p.policy == PropagationNone {
		return
	}

	propagation.TraceContext{}.Inject(ctx, carrier)
	if p.policy != PropagationTrustedService {
		return
	}
	requestID := reqctx.RequestID(ctx)
	if reqctx.ValidRequestID(requestID) {
		carrier.Set(requestIDHeader, requestID)
	}
}

func (policyPropagator) Extract(ctx context.Context, _ propagation.TextMapCarrier) context.Context {
	return ctx
}

func (p policyPropagator) Fields() []string {
	switch p.policy {
	case PropagationNone:
		return nil
	case PropagationTraceContext:
		return propagation.TraceContext{}.Fields()
	case PropagationTrustedService:
		fields := propagation.TraceContext{}.Fields()
		return append(fields, requestIDHeader)
	default:
		return nil
	}
}

func removeReservedHeaders(header http.Header) {
	for name := range header {
		for _, reserved := range reservedPropagationHeaders {
			if strings.EqualFold(name, reserved) {
				delete(header, name)
				break
			}
		}
	}
}
