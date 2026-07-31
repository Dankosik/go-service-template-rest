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
		err := failure(KindMalformed)
		v.metrics.recordVerification(ctx, TransportHTTP, err)
		return reqctx.Principal{}, err
	}
	if !v.trustedHTTPRequest(request) {
		err := failure(KindUntrustedTransport)
		v.metrics.recordVerification(ctx, TransportHTTP, err)
		return reqctx.Principal{}, err
	}

	values := request.Header.Values("Authorization")
	request.Header.Del("Authorization")
	token, err := bearerToken(values)
	if err != nil {
		v.metrics.recordVerification(ctx, TransportHTTP, err)
		return reqctx.Principal{}, err
	}
	return v.Verify(ctx, token, TransportHTTP)
}

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

func bearerToken(values []string) (string, error) {
	if len(values) == 0 {
		return "", failure(KindMissing)
	}
	if len(values) != 1 {
		return "", failure(KindMalformed)
	}
	value := values[0]
	if strings.TrimSpace(value) != value || strings.Contains(value, ",") {
		return "", failure(KindMalformed)
	}
	scheme, token, found := strings.Cut(value, " ")
	if !found ||
		!strings.EqualFold(scheme, "Bearer") ||
		token == "" ||
		strings.ContainsAny(token, " \t\r\n") {
		return "", failure(KindMalformed)
	}
	if len(token) > MaxTokenBytes {
		return "", failure(KindOversize)
	}
	return token, nil
}

func authenticatedRequest(input *openapi3filter.AuthenticationInput) *http.Request {
	if input == nil || input.RequestValidationInput == nil {
		return nil
	}
	return input.RequestValidationInput.Request
}
