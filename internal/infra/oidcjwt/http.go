package oidcjwt

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"strings"

	"github.com/example/go-service-template-rest/internal/reqctx"
	"github.com/getkin/kin-openapi/openapi3filter"
)

// errUnsupportedSecurityScheme reports a security requirement this boundary does
// not implement. It carries no [Kind] and is not counted: no credential was
// read, and the requirement was never this Verifier's to answer, so it is not a
// verification outcome. The validator treats it as an unmet requirement, which
// is the right answer under either OpenAPI reading — it fails an AND and moves
// on from an OR.
var errUnsupportedSecurityScheme = errors.New("authentication security scheme is not supported")

// ResolveHTTP is the concrete PrincipalResolver used by httpx.Authenticated.
func (v *Verifier) ResolveHTTP(
	ctx context.Context,
	input *openapi3filter.AuthenticationInput,
) (reqctx.Principal, error) {
	// The scheme is checked first because this function consumes the credential,
	// and the validator calls it once per scheme in every security requirement
	// until one is met. Answering a scheme this Verifier does not implement would
	// therefore do two things at once: accept a bearer access token as proof of
	// some other scheme's credential, and strip the header before the requirement
	// that actually wanted it is asked.
	if !bearerSecurityScheme(input) {
		return reqctx.Principal{}, errUnsupportedSecurityScheme
	}
	request := authenticatedRequest(input)
	if request == nil {
		return reqctx.Principal{}, v.recordRejection(ctx, transportHTTP, failure(KindMalformed))
	}
	if !v.trustedHTTPRequest(request) {
		return reqctx.Principal{}, v.recordRejection(ctx, transportHTTP, failure(KindUntrustedTransport))
	}

	// The credential is taken off the request as soon as this boundary owns it,
	// so no handler, logger, or downstream client can reach it. The untrusted
	// transport check above returns before this point on purpose: that request
	// never became ours to authenticate, and rewriting a rejected caller's
	// headers would hide from them what was actually sent.
	values := request.Header.Values("Authorization")
	request.Header.Del("Authorization")
	verified, err := v.verifyCredential(ctx, values, transportHTTP)
	return verified.principal, err
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
// the fix is the CIDR list, not a laxer reading here. bearerToken owns the same
// single-reading rule for the credential header, and the RFC 9110 argument for
// it; trimming is free here only because this compares a fixed token rather than
// carrying opaque bytes onward.
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

// bearerSecurityScheme reports whether the requirement being validated is the
// HTTP Bearer scheme this Verifier implements.
//
// It asks what the scheme is rather than what the contract named it, because the
// name is the service's to choose and carries no meaning here. The declaration
// is the validator's own view of the contract, so a request cannot influence it.
func bearerSecurityScheme(input *openapi3filter.AuthenticationInput) bool {
	if input == nil || input.SecurityScheme == nil {
		return false
	}
	return strings.EqualFold(input.SecurityScheme.Type, "http") &&
		strings.EqualFold(input.SecurityScheme.Scheme, "bearer")
}

func authenticatedRequest(input *openapi3filter.AuthenticationInput) *http.Request {
	if input == nil || input.RequestValidationInput == nil {
		return nil
	}
	return input.RequestValidationInput.Request
}
