package oidcjwt

import (
	"context"
	"errors"
	"net/http"
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

	// The credential is taken off the request as soon as this boundary owns it,
	// so no handler, logger, or downstream client can reach it. The untrusted
	// transport is owned by the listener/ingress deployment rather than inferred
	// from caller-controlled forwarding headers.
	values := request.Header.Values("Authorization")
	request.Header.Del("Authorization")
	verified, err := v.verifyCredential(ctx, values, transportHTTP)
	return verified.principal, err
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
