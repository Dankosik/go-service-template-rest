package httpclient

import (
	"net/http"
	"strings"

	"github.com/example/go-service-template-rest/internal/observability/correlationpolicy"
	"github.com/example/go-service-template-rest/internal/reqctx"
)

const requestIDHeader = reqctx.RequestIDHeader

// reservedPropagationHeaders is built once because RoundTrip walks it per header
// of every request. correlationpolicy owns the list; this file owns removing it
// from an http.Header.
var reservedPropagationHeaders = correlationpolicy.ReservedFields(requestIDHeader)

// PropagationPolicy selects which correlation fields one fixed target may
// receive. The zero value is deliberately fail-closed.
//
// It is an alias rather than a second enum so the grpcclient half cannot answer
// the same question differently; correlationpolicy owns the rule.
type PropagationPolicy = correlationpolicy.Policy

const (
	// PropagationNone keeps local client telemetry but emits no remote
	// correlation fields.
	PropagationNone = correlationpolicy.None
	// PropagationTraceContext emits only W3C Trace Context.
	PropagationTraceContext = correlationpolicy.TraceContext
	// PropagationTrustedService emits W3C Trace Context and the valid request ID
	// already present in the call context.
	PropagationTrustedService = correlationpolicy.TrustedService
)

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
