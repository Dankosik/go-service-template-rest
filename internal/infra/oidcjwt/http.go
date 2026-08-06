package oidcjwt

import (
	"context"
	"net"
	"net/http"
	"net/netip"
	"strings"

	"github.com/example/go-service-template-rest/internal/reqctx"
	"github.com/getkin/kin-openapi/openapi3filter"
)

// ResolveHTTP is the concrete PrincipalResolver used by httpx.Authenticated.
func (v *Verifier) ResolveHTTP(
	ctx context.Context,
	input *openapi3filter.AuthenticationInput,
) (reqctx.Principal, error) {
	request := authenticatedRequest(input)
	if request == nil {
		return reqctx.Principal{}, v.recordRejection(ctx, TransportHTTP, failure(KindMalformed))
	}
	if !v.trustedHTTPRequest(request) {
		return reqctx.Principal{}, v.recordRejection(ctx, TransportHTTP, failure(KindUntrustedTransport))
	}

	// The credential is taken off the request as soon as this boundary owns it,
	// so no handler, logger, or downstream client can reach it. The untrusted
	// transport check above returns before this point on purpose: that request
	// never became ours to authenticate, and rewriting a rejected caller's
	// headers would hide from them what was actually sent.
	values := request.Header.Values("Authorization")
	request.Header.Del("Authorization")
	return v.verifyCredential(ctx, values, TransportHTTP)
}

// trustedHTTPRequest reports whether this request reached the service the way
// the deployment says it must: through a proxy named in trusted_proxy_cidrs, and
// over TLS as that proxy reports it.
//
// Both terms are needed and neither substitutes for the other. The peer check is
// what makes the forwarded header worth reading at all — anyone can send
// X-Forwarded-Proto, so only a peer the operator listed gets to state it. The
// header check is what stops a trusted proxy's plaintext port from carrying
// credentials.
//
// Exactly one value, with no comma inside it, is the strict reading on purpose. A
// repeated or comma-joined header is a chain's accumulated claim, not the
// immediate peer's, and picking one entry out of it would be this boundary
// guessing which hop to believe. A deployment that terminates TLS further out
// than its trusted peer is one where the value is no longer that peer's to make;
// the fix is the CIDR list, not a laxer reading here.
//
// bearerToken applies the same single-reading rule to the credential header and
// owns the RFC 9110 argument for it. The two differ on surrounding whitespace,
// and a third forwarded header should copy this side: this check compares a fixed
// token case-insensitively, so trimming is free, while bearerToken carries opaque
// bytes onward and refuses a value whose framing was altered at all.
func (v *Verifier) trustedHTTPRequest(request *http.Request) bool {
	host, _, err := net.SplitHostPort(request.RemoteAddr)
	if err != nil {
		return false
	}
	address, err := netip.ParseAddr(host)
	if err != nil || !v.policy.trustedProxy(address) {
		return false
	}
	forwardedProto := request.Header.Values("X-Forwarded-Proto")
	return len(forwardedProto) == 1 &&
		!strings.Contains(forwardedProto[0], ",") &&
		strings.EqualFold(strings.TrimSpace(forwardedProto[0]), "https")
}

func authenticatedRequest(input *openapi3filter.AuthenticationInput) *http.Request {
	if input == nil || input.RequestValidationInput == nil {
		return nil
	}
	return input.RequestValidationInput.Request
}
