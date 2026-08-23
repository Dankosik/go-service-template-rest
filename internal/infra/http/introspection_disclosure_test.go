package httpx

// profile:authn-bearer:start

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/example/go-service-template-rest/internal/infra/bearerauthn"
	"github.com/example/go-service-template-rest/internal/reqctx"
	"github.com/getkin/kin-openapi/openapi3filter"
)

func TestIntrospectionDisclosureBoundary(t *testing.T) {
	t.Parallel()
	canary := "http-disclosure-canary"
	handler := securedHandlerWithTerminal(t, Authenticated(func(context.Context, *openapi3filter.AuthenticationInput) (reqctx.Principal, error) {
		return reqctx.Principal{}, fmt.Errorf("%s: %w", canary, bearerauthn.NewError(bearerauthn.KindUnavailable))
	}), defaultAuthenticateChallenge, slog.New(slog.DiscardHandler), http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler ran")
	}))
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "https://api.example.com/secret", http.NoBody)
	request.Header.Set("Authorization", "Bearer "+canary)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d", response.Code)
	}
	if strings.Contains(response.Body.String(), canary) {
		t.Fatalf("response disclosed canary: %s", response.Body.String())
	}
}

// profile:authn-bearer:end
