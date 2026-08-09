package httpx

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/example/go-service-template-rest/internal/problem"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers/gorillamux"
)

// securedSpec is a minimal contract that declares a security requirement. The
// shipped contract deliberately declares none, so the authentication seam has
// nothing to exercise against it; this supplies the one thing it is missing.
const securedSpec = `
openapi: 3.0.3
info:
  title: secured
  version: 0.0.1
paths:
  /secret:
    get:
      operationId: getSecret
      security:
        - bearerAuth: []
      responses:
        "204":
          description: no content
components:
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
`

// TestUnwiredAuthenticationFailsClosedAs401 is the defect this seam fixes.
//
// openapi3filter returns ErrAuthenticationServiceMissing when no
// AuthenticationFunc is set, so an unwired contract already failed closed — but
// it failed as a 400 "bad request", which tells a client its payload was wrong
// and which no client library retries with credentials.
func TestUnwiredAuthenticationFailsClosedAs401(t *testing.T) {
	t.Parallel()

	handler := securedHandler(t, nil, defaultAuthenticateChallenge)

	resp := doRequest(handler, http.MethodGet, "/secret")

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusUnauthorized)
	}
	if got := resp.Header().Get("WWW-Authenticate"); got != "Bearer" {
		t.Fatalf("WWW-Authenticate = %q, want %q", got, "Bearer")
	}
	assertProblemContentType(t, resp.Header())
	assertProblemCode(t, resp, problem.CodeUnauthorized)
}

func TestRejectedCredentialIs401(t *testing.T) {
	t.Parallel()

	authenticate := func(context.Context, *openapi3filter.AuthenticationInput) error {
		return errors.New("bearer credential is invalid")
	}
	handler := securedHandler(t, authenticate, defaultAuthenticateChallenge)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/secret", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusUnauthorized)
	}
	assertProblemCode(t, resp, problem.CodeUnauthorized)
}

func TestAcceptedCredentialReachesOperation(t *testing.T) {
	t.Parallel()

	var seen string
	authenticate := func(_ context.Context, input *openapi3filter.AuthenticationInput) error {
		seen = input.SecuritySchemeName
		return nil
	}
	handler := securedHandler(t, authenticate, defaultAuthenticateChallenge)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/secret", nil)
	req.Header.Set("Authorization", "Bearer right")
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusNoContent)
	}
	if seen != "bearerAuth" {
		t.Fatalf("security scheme name = %q, want %q", seen, "bearerAuth")
	}
}

// TestAuthenticateChallengeIsConfigurable keeps the WWW-Authenticate value in the
// service's hands. The contract's securityScheme keys are not HTTP auth schemes,
// so deriving the challenge from them would emit an illegal challenge.
func TestAuthenticateChallengeIsConfigurable(t *testing.T) {
	t.Parallel()

	handler := securedHandler(t, nil, `Basic realm="ops"`)

	resp := doRequest(handler, http.MethodGet, "/secret")

	if got := resp.Header().Get("WWW-Authenticate"); got != `Basic realm="ops"` {
		t.Fatalf("WWW-Authenticate = %q, want %q", got, `Basic realm="ops"`)
	}
}

// TestMalformedRequestStaysBadRequest keeps the 401 mapping from swallowing the
// ordinary validation failures it sits next to.
func TestMalformedRequestStaysBadRequest(t *testing.T) {
	t.Parallel()

	var logged bytes.Buffer
	reject := handleGeneratedRequestError(newTestServiceLogger(&logged), defaultAuthenticateChallenge)

	resp := httptest.NewRecorder()
	reject(resp, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/x", nil), errors.New("parameter is required"))

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusBadRequest)
	}
	assertProblemCode(t, resp, problem.CodeBadRequest)
	if got := resp.Header().Get("WWW-Authenticate"); got != "" {
		t.Fatalf("WWW-Authenticate = %q, want empty for a malformed request", got)
	}
}

// TestOversizedBodyStaysRequestEntityTooLarge keeps the ordering of the three
// classifications stable.
func TestOversizedBodyStaysRequestEntityTooLarge(t *testing.T) {
	t.Parallel()

	reject := handleGeneratedRequestError(slog.New(slog.DiscardHandler), defaultAuthenticateChallenge)

	resp := httptest.NewRecorder()
	reject(resp, httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/x", nil), &http.MaxBytesError{Limit: 1})

	if resp.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusRequestEntityTooLarge)
	}
	assertProblemCode(t, resp, problem.CodeRequestEntityTooLarge)
}

// TestSecurityRejectionDetailsAreSanitized keeps a 401 from echoing whatever the
// authenticator said about the credential.
func TestSecurityRejectionDetailsAreSanitized(t *testing.T) {
	t.Parallel()

	const leak = "token sk-live-abcdef is not in the allow list"
	var logged bytes.Buffer
	authenticate := func(context.Context, *openapi3filter.AuthenticationInput) error {
		return errors.New(leak)
	}
	handler := securedHandlerWithLog(t, authenticate, defaultAuthenticateChallenge, newTestServiceLogger(&logged))

	resp := doRequest(handler, http.MethodGet, "/secret")

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusUnauthorized)
	}
	if strings.Contains(resp.Body.String(), leak) {
		t.Fatalf("problem body leaks authenticator detail: %q", resp.Body.String())
	}
	if strings.Contains(logged.String(), leak) {
		t.Fatalf("log line leaks authenticator detail: %q", logged.String())
	}
}

func securedHandler(tb testing.TB, authenticate openapi3filter.AuthenticationFunc, challenge string) http.Handler {
	tb.Helper()
	return securedHandlerWithLog(tb, authenticate, challenge, slog.New(slog.DiscardHandler))
}

func securedHandlerWithLog(
	tb testing.TB,
	authenticate openapi3filter.AuthenticationFunc,
	challenge string,
	log *slog.Logger,
) http.Handler {
	tb.Helper()
	return securedHandlerWithTerminal(tb, authenticate, challenge, log, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
}

// securedHandlerWithTerminal runs securedSpec through the same requestValidator
// openAPIRequestValidator installs, so the mapping under test cannot drift from
// the one the router serves. The terminal handler is a parameter because the
// identity seam can only be proved by a handler that inspects the request it was
// given.
func securedHandlerWithTerminal(
	tb testing.TB,
	authenticate openapi3filter.AuthenticationFunc,
	challenge string,
	log *slog.Logger,
	terminal http.Handler,
) http.Handler {
	tb.Helper()

	loader := openapi3.NewLoader()
	spec, err := loader.LoadFromData([]byte(securedSpec))
	if err != nil {
		tb.Fatalf("load secured spec: %v", err)
	}
	if err := spec.Validate(loader.Context); err != nil {
		tb.Fatalf("validate secured spec: %v", err)
	}
	if _, err := gorillamux.NewRouter(spec); err != nil {
		tb.Fatalf("build secured router: %v", err)
	}

	validator := requestValidator(spec, authenticate, handleGeneratedRequestError(log, challenge))

	return RequestCorrelation(validator(terminal))
}
