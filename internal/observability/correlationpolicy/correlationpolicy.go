// Package correlationpolicy owns the reserved outbound correlation fields and
// the explicit gRPC policy for one fixed target. Fixed HTTP targets emit none.
//
// It is a leaf for the same reason internal/authntrust is: internal/infra/httpclient
// and internal/infra/grpcclient share the reserved-field rule at the same trust
// boundary and neither may import the other. The two adapters differ only in
// how their transport spells the request identifier, which is why that spelling
// is the one thing this package takes as an argument rather than owns —
// internal/reqctx owns both spellings.
//
// Stripping what a caller supplied is deliberately not here. A carrier is
// http.Header or metadata.MD, and each adapter must remove the fields from its
// own type at its own seam; [ReservedFields] and [Reserved] are what keep the two
// seams removing one list under one matching rule.
package correlationpolicy

import (
	"context"
	"slices"
	"strings"

	"github.com/example/go-service-template-rest/internal/reqctx"
	"go.opentelemetry.io/otel/propagation"
)

// Policy selects which correlation fields one fixed target may receive. The zero
// value is deliberately fail-closed.
type Policy uint8

const (
	// None keeps local client telemetry but emits no remote correlation.
	None Policy = iota
	// TraceContext emits only W3C Trace Context.
	TraceContext
	// TrustedService emits W3C Trace Context and the valid request identifier
	// already present in the call context.
	TrustedService
)

// Valid reports whether p is one of the declared policies. A caller validates
// rather than defaulting, because the zero value is the fail-closed policy and
// an out-of-range value silently reading as "emit nothing" would hide a
// configuration mistake behind the safest answer.
func (p Policy) Valid() bool {
	return p <= TrustedService
}

// ReservedFields are the correlation field names only the propagator may set,
// with requestIDKey spelled the way the caller's transport requires.
//
// Every one of them is removed from what a caller supplied before the propagator
// injects, so what crosses the trust boundary is exactly what Policy selected.
// baggage is on the list and is never emitted: it is caller-controlled key-value
// data with no bound, and forwarding it would leak whatever an upstream put there
// to a target this service chose the correlation for.
func ReservedFields(requestIDKey string) []string {
	return []string{"traceparent", "tracestate", "baggage", requestIDKey}
}

// Reserved reports whether key names one of [ReservedFields]. Each adapter holds
// one of these and asks it per field of every outbound call.
//
// The match is case-insensitive because the two carriers spell one field
// differently: http.Header canonicalizes "traceparent" to "Traceparent", and
// metadata.MD is lowercase by convention rather than by construction. Each seam
// wrote that comparison out for itself before, which left the list shared and
// the rule for reading it duplicated — so an entry whose matching is not a plain
// comparison would have had to be added to the list here and to two loops
// elsewhere, and the loop that was missed would have gone on stripping nothing.
func Reserved(requestIDKey string) func(key string) bool {
	fields := ReservedFields(requestIDKey)
	return func(key string) bool {
		return slices.ContainsFunc(fields, func(reserved string) bool {
			return strings.EqualFold(key, reserved)
		})
	}
}

// Propagator injects exactly what its policy selects. It is a
// propagation.TextMapPropagator, and it is a concrete type so a caller can hold
// one without an interface value.
type Propagator struct {
	policy       Policy
	requestIDKey string
}

// NewPropagator returns the propagator for policy, writing the request
// identifier under requestIDKey when the policy emits one.
func NewPropagator(policy Policy, requestIDKey string) Propagator {
	return Propagator{policy: policy, requestIDKey: requestIDKey}
}

func (p Propagator) Inject(ctx context.Context, carrier propagation.TextMapCarrier) {
	if p.policy == None {
		return
	}

	propagation.TraceContext{}.Inject(ctx, carrier)
	if p.policy != TrustedService {
		return
	}
	requestID := reqctx.RequestID(ctx)
	if reqctx.ValidRequestID(requestID) {
		carrier.Set(p.requestIDKey, requestID)
	}
}

// Extract returns ctx unchanged. This propagator is outbound only: what a
// response carries is the target's statement about itself, and adopting it would
// let a peer rewrite this process's own correlation.
func (Propagator) Extract(ctx context.Context, _ propagation.TextMapCarrier) context.Context {
	return ctx
}

func (p Propagator) Fields() []string {
	switch p.policy {
	case None:
		return nil
	case TraceContext:
		return propagation.TraceContext{}.Fields()
	case TrustedService:
		return append(propagation.TraceContext{}.Fields(), p.requestIDKey)
	default:
		return nil
	}
}
