package httpx

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/go-service-template-rest/internal/problem"
	"github.com/example/go-service-template-rest/internal/reqctx"
	"github.com/getkin/kin-openapi/openapi3filter"
)

// TestAuthenticatedPrincipalReachesOperation is the whole point of the seam, and
// it is also the canary for the assumption it rests on: the validator hands the
// next handler the same *http.Request the AuthenticationFunc was given. If a
// future version of oapi-codegen/nethttp-middleware copies the request instead,
// this test fails rather than authorization silently going missing.
func TestAuthenticatedPrincipalReachesOperation(t *testing.T) {
	t.Parallel()

	var (
		observed  reqctx.Principal
		present   bool
		requestID string
	)
	authenticate := Authenticated(func(_ context.Context, input *openapi3filter.AuthenticationInput) (reqctx.Principal, error) {
		if input.SecuritySchemeName != "bearerAuth" {
			return reqctx.Principal{}, errors.New("unexpected security scheme")
		}
		return reqctx.Principal{Subject: "svc-checkout", Scopes: []string{"articles:write"}}, nil
	})
	terminal := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		observed, present = reqctx.PrincipalFromContext(r.Context())
		requestID = reqctx.RequestID(r.Context())
		w.WriteHeader(http.StatusNoContent)
	})
	handler := securedHandlerWithTerminal(t, authenticate, defaultAuthenticateChallenge, slog.New(slog.DiscardHandler), terminal)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/secret", nil)
	req.Header.Set("Authorization", "Bearer right")
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusNoContent)
	}
	if !present {
		t.Fatal("handler observed no principal; the resolved identity did not survive the validator")
	}
	if observed.Subject != "svc-checkout" {
		t.Fatalf("subject = %q, want %q", observed.Subject, "svc-checkout")
	}
	if !observed.HasScope("articles:write") {
		t.Fatalf("scopes = %v, want articles:write", observed.Scopes)
	}
	// The correlation identifier travels the same context, and a handler needs it
	// for records that must sit beside the access log line for this request.
	if requestID == "" {
		t.Fatal("handler observed no request ID")
	}
}

// TestAuthenticatedRejectionKeepsUnauthorizedMapping keeps the wrapper from
// changing what a failed credential looks like to a client.
func TestAuthenticatedRejectionKeepsUnauthorizedMapping(t *testing.T) {
	t.Parallel()

	reached := false
	authenticate := Authenticated(func(context.Context, *openapi3filter.AuthenticationInput) (reqctx.Principal, error) {
		return reqctx.Principal{}, errors.New("bearer credential is invalid")
	})
	terminal := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reached = true
		w.WriteHeader(http.StatusNoContent)
	})
	handler := securedHandlerWithTerminal(t, authenticate, defaultAuthenticateChallenge, slog.New(slog.DiscardHandler), terminal)

	resp := doRequest(handler, http.MethodGet, "/secret")

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusUnauthorized)
	}
	if reached {
		t.Fatal("operation ran for a rejected credential")
	}
	assertProblemCode(t, resp, problem.CodeUnauthorized)
}

func TestAuthenticatedRejectsPrincipalWithoutSubject(t *testing.T) {
	t.Parallel()

	reached := false
	authenticate := Authenticated(func(context.Context, *openapi3filter.AuthenticationInput) (reqctx.Principal, error) {
		return reqctx.Principal{Subject: " \t", Scopes: []string{"articles:write"}}, nil
	})
	terminal := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reached = true
		w.WriteHeader(http.StatusNoContent)
	})
	handler := securedHandlerWithTerminal(t, authenticate, defaultAuthenticateChallenge, slog.New(slog.DiscardHandler), terminal)

	resp := doRequest(handler, http.MethodGet, "/secret")

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusUnauthorized)
	}
	if reached {
		t.Fatal("operation ran for an authenticated principal without a subject")
	}
	assertProblemCode(t, resp, problem.CodeUnauthorized)
}

// TestAuthenticatedNilResolverStaysFailClosed keeps the unwired default the same
// as leaving RouterConfig.Authenticate nil: 401, not an admitted request with no
// principal.
func TestAuthenticatedNilResolverStaysFailClosed(t *testing.T) {
	t.Parallel()

	if got := Authenticated(nil); got != nil {
		t.Fatal("Authenticated(nil) returned a function; an unwired seam must stay nil so secured operations answer 401")
	}

	handler := securedHandlerWithLog(t, Authenticated(nil), defaultAuthenticateChallenge, slog.New(slog.DiscardHandler))

	resp := doRequest(handler, http.MethodGet, "/secret")

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusUnauthorized)
	}
}

func TestAuthenticatedWithoutRequestFailsClosed(t *testing.T) {
	t.Parallel()

	authenticate := Authenticated(func(context.Context, *openapi3filter.AuthenticationInput) (reqctx.Principal, error) {
		return reqctx.Principal{Subject: "svc", Scopes: nil}, nil
	})

	err := authenticate(context.Background(), &openapi3filter.AuthenticationInput{})

	if !errors.Is(err, errUnresolvableAuthenticatedRequest) {
		t.Fatalf("err = %v, want %v", err, errUnresolvableAuthenticatedRequest)
	}
}
