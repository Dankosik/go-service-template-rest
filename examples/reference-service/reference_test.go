package referenceservice

import (
	"context"
	"encoding/json"
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

var seedArticles = []article.Article{{
	Slug:      "clear-owners",
	Title:     "Keep responsibilities with their owner",
	Summary:   "A transport maps HTTP, a use case owns behavior, and an adapter owns storage details.",
	Published: true,
}}

// These tests live at the composition root because that is where the transport
// adapter is wired. The feature package must not import it, so the assertions
// that the example actually inherits the shared chain and the shared rejection
// mapping belong here.

// TestReferenceServiceServesOverHTTP exercises the composition end to end through
// a real server rather than a recorder, which is what the deleted main used to
// half-prove with a hand-rolled listener and a lifecycle nobody should copy.
func TestReferenceServiceServesOverHTTP(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(mustNewHandler(t))
	t.Cleanup(server.Close)

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL+"/api/v1/articles/clear-owners", nil)
	if err != nil {
		t.Fatalf("build request error = %v", err)
	}
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatalf("GET article error = %v", err)
	}
	t.Cleanup(func() { _ = response.Body.Close() })

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	if got := response.Header.Get("X-Request-ID"); got == "" {
		t.Fatal("X-Request-ID is empty, want correlation applied by the shared chain")
	}
}

func TestReferenceServiceMapsAlreadyExistsOverHTTP(t *testing.T) {
	t.Parallel()

	handler := mustNewHandler(t)
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/articles", strings.NewReader(
		`{"slug":"clear-owners","title":"Duplicate","summary":"Already present."}`,
	))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+testWriteToken)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusConflict || response.Header().Get("Retry-After") != "" {
		t.Fatalf("status/Retry-After = %d/%q, want 409/empty; body = %s", response.Code, response.Header().Get("Retry-After"), response.Body.String())
	}
	var body struct {
		Code   string  `json:"code"`
		Detail *string `json:"detail"`
		Status int     `json:"status"`
		Title  string  `json:"title"`
		Type   string  `json:"type"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode problem: %v", err)
	}
	if body.Code != "already_exists" || body.Status != http.StatusConflict || body.Title != "conflict" ||
		body.Type != "https://www.rfc-editor.org/rfc/rfc9110#section-15.5.10" || body.Detail == nil ||
		*body.Detail != "an article with this slug already exists" || strings.Contains(response.Body.String(), "create article") {
		t.Fatalf("problem = %+v, want safe already_exists envelope", body)
	}
}

// TestReferenceRouterInheritsHardenedChain is the point of routing this example
// through httpx.Harden. Before it did, a reader copying the example got a router
// with no correlation, no security headers, no body limit, no request budget, and
// no panic recovery — while believing the template supplied all of them.
func TestReferenceRouterInheritsHardenedChain(t *testing.T) {
	t.Parallel()

	handler := mustNewHandler(t)

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/articles/clear-owners", nil))

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
	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/articles", strings.NewReader(body))
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

	handler := mustNewHandler(t)

	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/articles", strings.NewReader(`{"slug":"s","title":"t","summary":"x"}`))
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
	handler.ServeHTTP(response, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/articles/anything", nil))

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	if got := response.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/problem+json") {
		t.Fatalf("Content-Type = %q, want a problem envelope", got)
	}
}

// TestReferenceRouterLimitsOneCallerWithoutAffectingAnother is the property the
// rate limiter seam exists for: global shedding cannot tell two callers apart, so
// one client's burst costs everyone else their requests.
func TestReferenceRouterLimitsOneCallerWithoutAffectingAnother(t *testing.T) {
	t.Parallel()

	handler := mustNewHandler(t)

	limited := false
	for range rateLimitBurst * 2 {
		if readArticleStatus(handler, "Bearer "+testWriteToken) == http.StatusTooManyRequests {
			limited = true
			break
		}
	}
	if !limited {
		t.Fatalf("a caller never hit the %d/s limit after %d requests", rateLimitPerSecond, rateLimitBurst*2)
	}

	if got := readArticleStatus(handler, "Bearer some-other-caller"); got == http.StatusTooManyRequests {
		t.Fatal("a second caller was limited by the first caller's burst")
	}
}

func readArticleStatus(handler http.Handler, authorization string) int {
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/articles/clear-owners", nil)
	request.Header.Set("Authorization", authorization)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response.Code
}

func mustNewHandler(tb testing.TB) http.Handler {
	tb.Helper()

	handler, err := NewHandler(slog.New(slog.DiscardHandler), Options{
		WriteToken: testWriteToken,
		Seed:       seedArticles,
	})
	if err != nil {
		tb.Fatalf("NewHandler() error = %v", err)
	}
	return handler
}

// mustBuildReferenceRouter composes what NewHandler composes, with the transport
// bounds a single test needs to vary.
func mustBuildReferenceRouter(tb testing.TB, bodyLimit int64, repository articleStore) http.Handler {
	tb.Helper()

	service, err := article.NewService(repository, repository)
	if err != nil {
		tb.Fatalf("article.NewService() error = %v", err)
	}
	log := slog.New(slog.DiscardHandler)

	apiHandler, err := httpapi.NewAPIHandler(service, httpapi.Options{
		Authenticate:   httpx.Authenticated(resolveWriter(testWriteToken)),
		RejectRequest:  httpx.RejectRequest(log, authenticateChallenge),
		RejectResponse: httpx.RejectResponse(article.ClassifyError),
	})
	if err != nil {
		tb.Fatalf("NewAPIHandler() error = %v", err)
	}

	handler, err := httpx.Harden(log, telemetry.New(), httpx.RouterConfig{
		MaxBodyBytes:   bodyLimit,
		RequestTimeout: RequestTimeout,
		MaxInFlight:    maxInFlight,
		OTelServerName: "reference-service-test",
	}, apiHandler)
	if err != nil {
		tb.Fatalf("httpx.Harden() error = %v", err)
	}
	return handler
}

// articleStore is both halves of what the use case needs, which is what a real
// adapter supplies from one type.
type articleStore interface {
	article.Repository
	article.Atomically
}

func memoryRepository(tb testing.TB) *memory.Repository {
	tb.Helper()

	repository, err := memory.New(seedArticles)
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

func (panickingRepository) AppendEvent(context.Context, article.Event) error {
	panic("feature code bug")
}

func (p panickingRepository) Do(_ context.Context, fn func(article.Repository) error) error {
	return fn(p)
}
