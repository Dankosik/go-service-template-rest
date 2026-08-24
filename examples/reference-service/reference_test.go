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
	"github.com/example/go-service-template-rest/internal/problem"
)

const testWriteToken = "reference-write-token"

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

	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, server.URL+"/api/v1/articles", strings.NewReader(
		`{"slug":"clear-owners","title":"Clear owners","summary":"Keep behavior with its owner."}`,
	))
	if err != nil {
		t.Fatalf("build request error = %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+testWriteToken)
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatalf("POST article error = %v", err)
	}
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusCreated)
	}
	if got := response.Header.Get("X-Request-ID"); got == "" {
		t.Fatal("X-Request-ID is empty, want correlation applied by the shared chain")
	}
	if err := response.Body.Close(); err != nil {
		t.Fatalf("close POST response: %v", err)
	}

	request, err = http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL+"/api/v1/articles/clear-owners", http.NoBody)
	if err != nil {
		t.Fatalf("build GET request error = %v", err)
	}
	response, err = server.Client().Do(request)
	if err != nil {
		t.Fatalf("GET article error = %v", err)
	}
	t.Cleanup(func() { _ = response.Body.Close() })
	if response.StatusCode != http.StatusOK {
		t.Fatalf("GET status = %d, want %d", response.StatusCode, http.StatusOK)
	}
}

func TestReferenceServiceMapsAlreadyExistsOverHTTP(t *testing.T) {
	t.Parallel()

	handler := mustNewHandler(t)
	first := postReferenceArticle(t, handler,
		`{"slug":"clear-owners","title":"Original","summary":"Created first."}`)
	if first.Code != http.StatusCreated {
		t.Fatalf("first create status = %d, want %d; body = %s", first.Code, http.StatusCreated, first.Body.String())
	}
	response := postReferenceArticle(t, handler,
		`{"slug":"clear-owners","title":"Duplicate","summary":"Already present."}`)

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

	handler := mustBuildReferenceRouter(t, 16, memory.New())

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

	for _, testCase := range []struct {
		name  string
		token string
	}{
		{name: "missing"},
		{name: "wrong", token: "not-the-token"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/articles", strings.NewReader(`{"slug":"s","title":"t","summary":"x"}`))
			request.Header.Set("Content-Type", "application/json")
			if testCase.token != "" {
				request.Header.Set("Authorization", "Bearer "+testCase.token)
			}
			response := httptest.NewRecorder()
			mustNewHandler(t).ServeHTTP(response, request)

			if response.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusUnauthorized, response.Body.String())
			}
			if got := response.Header().Get("WWW-Authenticate"); got != authenticateChallenge {
				t.Fatalf("WWW-Authenticate = %q, want %q", got, authenticateChallenge)
			}
		})
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

// TestReferenceServiceNamesTheInvalidFieldOverHTTP drives the real router so the
// contract, the generated validator, and the problem writer all have to be
// present for it to pass.
//
// It pins both halves of what a validation rejection may say. The caller learns
// which member failed and which constraint it broke, because a 400 that says
// only "invalid" makes them diff their payload against the spec by hand. And the
// value they sent appears nowhere: kin-openapi keeps it in SchemaError.Value,
// and the canary below is what fails if a future edit reaches for that field or
// for the error's own text.
func TestReferenceServiceNamesTheInvalidFieldOverHTTP(t *testing.T) {
	t.Parallel()

	// Violates the contract's `^[a-z][a-z0-9-]*$`, and stands in for a payload
	// value that must not be echoed.
	const rejectedSlug = "NOT_A_SLUG_secret_value"

	handler := mustNewHandler(t)
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/articles", strings.NewReader(
		`{"slug":"`+rejectedSlug+`","title":"Title","summary":"Summary."}`,
	))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+testWriteToken)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusBadRequest, response.Body.String())
	}
	if strings.Contains(response.Body.String(), rejectedSlug) {
		t.Fatalf("problem body echoes the submitted value: %s", response.Body.String())
	}

	var body struct {
		Code          string `json:"code"`
		InvalidParams []struct {
			Name   string `json:"name"`
			Reason string `json:"reason"`
		} `json:"invalid_params"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode problem: %v; body = %s", err, response.Body.String())
	}
	if body.Code != string(problem.CodeBadRequest) {
		t.Fatalf("code = %q, want %q", body.Code, problem.CodeBadRequest)
	}
	if len(body.InvalidParams) != 1 {
		t.Fatalf("invalid_params = %+v, want exactly the one member that failed", body.InvalidParams)
	}
	if body.InvalidParams[0].Name != "/slug" {
		t.Errorf("invalid_params[0].name = %q, want the RFC 6901 pointer %q", body.InvalidParams[0].Name, "/slug")
	}
	if body.InvalidParams[0].Reason == "" {
		t.Error("invalid_params[0].reason is empty, want the constraint that failed")
	}
}

// TestReferenceServiceOmitsInvalidParamsForANonValidationProblem keeps the
// extension member from becoming noise on every problem: a domain 404 has no
// field to point at, and an empty array would read as one that was checked.
func TestReferenceServiceOmitsInvalidParamsForANonValidationProblem(t *testing.T) {
	t.Parallel()

	handler := mustNewHandler(t)
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/articles/no-such-article", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusNotFound, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "invalid_params") {
		t.Fatalf("problem body carries invalid_params for a domain failure: %s", response.Body.String())
	}
}

func mustNewHandler(tb testing.TB) http.Handler {
	tb.Helper()

	handler, err := NewHandler(slog.New(slog.DiscardHandler), testWriteToken)
	if err != nil {
		tb.Fatalf("NewHandler() error = %v", err)
	}
	return handler
}

// mustBuildReferenceRouter composes what NewHandler composes, with the transport
// bounds a single test needs to vary.
func mustBuildReferenceRouter(tb testing.TB, bodyLimit int64, repository article.Store) http.Handler {
	tb.Helper()

	service, err := article.NewService(repository)
	if err != nil {
		tb.Fatalf("article.NewService() error = %v", err)
	}
	log := slog.New(slog.DiscardHandler)

	apiHandler, err := httpapi.NewAPIHandler(service, httpapi.Options{
		Authenticate:   httpx.Authenticated(resolveWriter(testWriteToken)),
		RejectRequest:  httpx.RejectRequest(log, authenticateChallenge),
		RejectResponse: httpx.RejectResponse(log, article.ClassifyError),
	})
	if err != nil {
		tb.Fatalf("NewAPIHandler() error = %v", err)
	}

	handler, err := httpx.Harden(log, telemetry.New(), httpx.HardenConfig{
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

func (p panickingRepository) Do(_ context.Context, fn func(article.Writer) error) error {
	return fn(p)
}

func postReferenceArticle(tb testing.TB, handler http.Handler, body string) *httptest.ResponseRecorder {
	tb.Helper()

	request := httptest.NewRequestWithContext(tb.Context(), http.MethodPost, "/api/v1/articles", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+testWriteToken)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
