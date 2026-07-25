package main

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/example/go-service-template-rest/examples/reference-service/internal/article"
	"github.com/example/go-service-template-rest/examples/reference-service/internal/article/memory"
	"github.com/example/go-service-template-rest/examples/reference-service/internal/httpapi"
	httpx "github.com/example/go-service-template-rest/internal/infra/http"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
)

const testWriteToken = "reference-write-token"

// These tests live at the composition root because that is where the transport
// adapter is wired. The feature package must not import it, so the assertions
// that the example actually inherits the shared chain and the shared rejection
// mapping belong here.

// TestReferenceRouterInheritsHardenedChain is the point of routing this example
// through httpx.Harden. Before it did, a reader copying the example got a router
// with no correlation, no security headers, no body limit, no request budget, and
// no panic recovery — while believing the template supplied all of them.
func TestReferenceRouterInheritsHardenedChain(t *testing.T) {
	t.Parallel()

	handler := mustBuildReferenceRouter(t, maxBodyBytes, memoryRepository(t))

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/articles/clear-owners", nil))

	if got := response.Header().Get("X-Request-ID"); got == "" {
		t.Fatal("X-Request-ID is empty, want correlation applied by the shared chain")
	}
	if got := response.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q, want %q", got, "nosniff")
	}
}

// TestReferenceRouterRejectsOversizedBody proves the shared body limit is in
// force and that the rejection uses a problem envelope rather than a bare
// net/http error page.
func TestReferenceRouterRejectsOversizedBody(t *testing.T) {
	t.Parallel()

	handler := mustBuildReferenceRouter(t, 16, memoryRepository(t))

	body := `{"slug":"a-very-long-slug-value","title":"t","summary":"s"}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/articles", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+testWriteToken)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusRequestEntityTooLarge, response.Body.String())
	}
	if got := response.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/problem+json") {
		t.Fatalf("Content-Type = %q, want a problem envelope", got)
	}
}

// TestReferenceRouterMapsMissingCredentialTo401 pins the shared mapping. A
// missing credential must not be reported as a malformed request, or no client
// library will retry with credentials.
func TestReferenceRouterMapsMissingCredentialTo401(t *testing.T) {
	t.Parallel()

	handler := mustBuildReferenceRouter(t, maxBodyBytes, memoryRepository(t))

	request := httptest.NewRequest(http.MethodPost, "/api/v1/articles", strings.NewReader(`{"slug":"s","title":"t","summary":"x"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusUnauthorized, response.Body.String())
	}
	if got := response.Header().Get("WWW-Authenticate"); got != authenticateChallenge {
		t.Fatalf("WWW-Authenticate = %q, want %q", got, authenticateChallenge)
	}
}

// TestReferenceRouterRecoversPanicFromFeatureCode proves panic recovery reaches
// this example's handlers. Without the shared chain, net/http's per-connection
// recovery dropped the connection and printed plain text to stderr.
func TestReferenceRouterRecoversPanicFromFeatureCode(t *testing.T) {
	t.Parallel()

	handler := mustBuildReferenceRouter(t, maxBodyBytes, panickingRepository{})

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/articles/anything", nil))

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	if got := response.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/problem+json") {
		t.Fatalf("Content-Type = %q, want a problem envelope", got)
	}
}

// mustBuildReferenceRouter composes exactly what run() composes.
func mustBuildReferenceRouter(tb testing.TB, bodyLimit int64, repository article.Repository) http.Handler {
	tb.Helper()

	service, err := article.NewService(repository)
	if err != nil {
		tb.Fatalf("article.NewService() error = %v", err)
	}
	log := slog.New(slog.DiscardHandler)

	apiHandler, err := httpapi.NewAPIHandler(service, httpapi.Options{
		WriteToken:     testWriteToken,
		RejectRequest:  httpx.RejectRequest(log, authenticateChallenge),
		RejectResponse: httpx.RejectResponse(),
	})
	if err != nil {
		tb.Fatalf("NewAPIHandler() error = %v", err)
	}

	handler, err := httpx.Harden(log, telemetry.New(), httpx.RouterConfig{
		MaxBodyBytes:   bodyLimit,
		RequestTimeout: requestTimeout,
		MaxInFlight:    maxInFlight,
		OTelServerName: "reference-service-test",
	}, apiHandler)
	if err != nil {
		tb.Fatalf("httpx.Harden() error = %v", err)
	}
	return handler
}

func memoryRepository(tb testing.TB) article.Repository {
	tb.Helper()

	repository, err := memory.New([]article.Article{{
		Slug:      "clear-owners",
		Title:     "Keep responsibilities with their owner",
		Summary:   "A transport maps HTTP, a use case owns behavior, and an adapter owns storage details.",
		Published: true,
	}})
	if err != nil {
		tb.Fatalf("memory.New() error = %v", err)
	}
	return repository
}

type panickingRepository struct{}

func (panickingRepository) FindBySlug(context.Context, string) (article.Article, error) {
	panic("feature code bug")
}

func (panickingRepository) Create(context.Context, article.Article) error {
	panic("feature code bug")
}
