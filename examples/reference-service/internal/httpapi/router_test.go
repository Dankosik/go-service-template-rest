package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/example/go-service-template-rest/examples/reference-service/internal/article"
	"github.com/example/go-service-template-rest/examples/reference-service/internal/article/memory"
	"github.com/example/go-service-template-rest/examples/reference-service/internal/openapi"
	"github.com/example/go-service-template-rest/internal/problem"
	"github.com/example/go-service-template-rest/internal/reqctx"
	"github.com/getkin/kin-openapi/openapi3filter"
)

const testWriteToken = "reference-write-token"

func TestOpenAPIOperationsDeclareHardenedMiddlewareFailures(t *testing.T) {
	t.Parallel()

	spec, err := openapi.GetSpec()
	if err != nil {
		t.Fatalf("openapi.GetSpec() error = %v", err)
	}
	for path, item := range spec.Paths.Map() {
		if item == nil || strings.HasPrefix(path, "/health/") {
			continue
		}
		for method, operation := range item.Operations() {
			for _, status := range []string{"413", "503", "504"} {
				responseRef := operation.Responses.Value(status)
				if responseRef == nil || responseRef.Value == nil {
					t.Errorf("%s %s lacks %s response from the hardened middleware chain", method, path, status)
					continue
				}
				if responseRef.Value.Content.Get("application/problem+json") == nil {
					t.Errorf("%s %s response %s is not application/problem+json", method, path, status)
				}
				if status == "503" && responseRef.Value.Headers["Retry-After"] == nil {
					t.Errorf("%s %s response %s lacks Retry-After", method, path, status)
				}
			}
		}
	}
}

func TestRouterRejectsInvalidSlugBeforeUseCase(t *testing.T) {
	t.Parallel()

	repository := &countingRepository{}
	service, err := article.NewService(repository)
	if err != nil {
		t.Fatalf("article.NewService() error = %v", err)
	}
	router := mustNewRouter(t, service)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/articles/INVALID", nil))

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusBadRequest, response.Body.String())
	}
	if repository.calls != 0 {
		t.Fatalf("repository calls = %d, want 0", repository.calls)
	}
}

type countingRepository struct {
	calls int
}

func (r *countingRepository) FindBySlug(context.Context, string) (article.Article, error) {
	r.calls++
	return article.Article{}, article.ErrNotFound
}

func (r *countingRepository) Create(context.Context, article.Article) error {
	r.calls++
	return nil
}

func (r *countingRepository) AppendEvent(context.Context, article.Event) error {
	r.calls++
	return nil
}

func (r *countingRepository) Do(_ context.Context, fn func(article.Writer) error) error {
	return fn(r)
}

// newTestRouter builds the router over an empty in-memory repository.
func newTestRouter(t *testing.T) http.Handler {
	t.Helper()

	repository := memory.New()
	service, err := article.NewService(repository)
	if err != nil {
		t.Fatalf("article.NewService() error = %v", err)
	}
	return mustNewRouter(t, service)
}

// mustNewRouter builds the API handler this package owns.
//
// The reject mappers are local test doubles rather than the shared ones from the
// transport adapter: a feature package must not import that adapter, so the real
// mapping is proved at the composition root instead. See
// TestReferenceRouterInheritsHardenedChain in the reference binary's tests.
func mustNewRouter(tb testing.TB, service *article.Service) http.Handler {
	tb.Helper()

	handler, err := NewAPIHandler(service, Options{
		Authenticate:   testAuthenticate(ArticleWriteScope),
		RejectRequest:  testRejectRequest,
		RejectResponse: testRejectResponse,
	})
	if err != nil {
		tb.Fatalf("NewAPIHandler() error = %v", err)
	}
	return handler
}

// testAuthenticate accepts testWriteToken and grants it scopes. The binary uses
// httpx.Authenticated for this; a feature package must not import that adapter,
// so this test double publishes the same transport-neutral context value itself.
func testAuthenticate(scopes ...string) openapi3filter.AuthenticationFunc {
	return func(_ context.Context, input *openapi3filter.AuthenticationInput) error {
		request := input.RequestValidationInput.Request
		presented, ok := strings.CutPrefix(request.Header.Get("Authorization"), "Bearer ")
		if !ok || strings.TrimSpace(presented) != testWriteToken {
			return errors.New("bearer credential is invalid")
		}
		principal := reqctx.Principal{
			Subject: "test-writer",
			Scopes:  scopes,
		}
		//nolint:contextcheck // ContextWithPrincipal derives from the request's current context.
		*request = *request.WithContext(reqctx.ContextWithPrincipal(request.Context(), principal))
		return nil
	}
}

func testRejectRequest(w http.ResponseWriter, r *http.Request, err error) {
	if _, ok := errors.AsType[*openapi3filter.SecurityRequirementsError](err); ok {
		w.Header().Set("WWW-Authenticate", "Bearer")
		writeTestProblem(w, newProblem(r.Context(), problem.CodeUnauthorized, "credentials are missing or invalid"))
		return
	}
	writeTestProblem(w, newProblem(r.Context(), problem.CodeBadRequest, "request is malformed or invalid"))
}

func testRejectResponse(w http.ResponseWriter, r *http.Request, _ error) {
	writeTestProblem(w, newProblem(r.Context(), problem.CodeInternalError, "request failed"))
}

func writeTestProblem(w http.ResponseWriter, body openapi.Problem) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(int(body.Status))
	encoded, err := json.Marshal(body)
	if err != nil {
		return
	}
	_, _ = w.Write(encoded)
}

func postArticle(t *testing.T, router http.Handler, body string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/articles", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+testWriteToken)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func TestRouterCreateArticleReturnsCreatedWithLocation(t *testing.T) {
	t.Parallel()

	router := newTestRouter(t)
	response := postArticle(t, router,
		`{"slug":"clear-owners","title":"Clear owners","summary":"Keep behavior with its owner."}`)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusCreated, response.Body.String())
	}
	if got, want := response.Header().Get("Location"), "/api/v1/articles/clear-owners"; got != want {
		t.Fatalf("Location = %q, want %q", got, want)
	}

	var body openapi.Article
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Slug != "clear-owners" || body.Title != "Clear owners" {
		t.Fatalf("body = %+v, want the created article", body)
	}

	// A created article must be readable through the public operation.
	read := httptest.NewRecorder()
	router.ServeHTTP(read, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/articles/clear-owners", nil))
	if read.Code != http.StatusOK {
		t.Fatalf("read-back status = %d, want %d; body = %s", read.Code, http.StatusOK, read.Body.String())
	}
}

func TestRouterCreateArticleAcceptsUnicodeAtContractLimit(t *testing.T) {
	t.Parallel()

	title := strings.Repeat("я", 200)
	response := postArticle(t, newTestRouter(t),
		`{"slug":"unicode","title":"`+title+`","summary":"Accepted by both boundaries."}`)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusCreated, response.Body.String())
	}
}

func TestRouterRejectsMalformedCreateBodyBeforeUseCase(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name string
		body string
	}{
		{name: "not json", body: `{`},
		{name: "missing required field", body: `{"slug":"clear-owners","title":"Clear owners"}`},
		{name: "unknown field", body: `{"slug":"a","title":"t","summary":"s","extra":true}`},
		{name: "slug violates pattern", body: `{"slug":"Clear_Owners","title":"t","summary":"s"}`},
		{name: "empty title", body: `{"slug":"clear-owners","title":"","summary":"s"}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repository := &countingRepository{}
			service, err := article.NewService(repository)
			if err != nil {
				t.Fatalf("article.NewService() error = %v", err)
			}
			router := mustNewRouter(t, service)

			response := postArticle(t, router, tt.body)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusBadRequest, response.Body.String())
			}
			// The contract rejects the request before any business code runs.
			if repository.calls != 0 {
				t.Fatalf("repository calls = %d, want 0", repository.calls)
			}
		})
	}
}

func TestNewAPIHandlerRejectsMissingCollaborators(t *testing.T) {
	t.Parallel()

	service := mustArticleService(t)

	if _, err := NewAPIHandler(nil, Options{
		Authenticate:   testAuthenticate(ArticleWriteScope),
		RejectRequest:  testRejectRequest,
		RejectResponse: testRejectResponse,
	}); err == nil {
		t.Fatal("NewAPIHandler() error = nil, want rejection of a missing article service")
	}
	if _, err := NewAPIHandler(service, Options{Authenticate: testAuthenticate(ArticleWriteScope)}); err == nil {
		t.Fatal("NewAPIHandler() error = nil, want rejection of missing reject mappers")
	}
}

// TestCredentialWithoutWriteScopeIsForbidden is the half of the identity seam a
// 401 cannot express. The credential is valid — the validator admitted it — and
// the operation is still refused, which is only decidable because the resolved
// principal reached the handler.
func TestCredentialWithoutWriteScopeIsForbidden(t *testing.T) {
	t.Parallel()

	handler, err := NewAPIHandler(mustArticleService(t), Options{
		Authenticate:   testAuthenticate("articles:read"),
		RejectRequest:  testRejectRequest,
		RejectResponse: testRejectResponse,
	})
	if err != nil {
		t.Fatalf("NewAPIHandler() error = %v", err)
	}

	response := postArticle(t, handler,
		`{"slug":"scoped","title":"Scoped","summary":"Requires the write scope."}`)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
}

// TestMissingAuthenticatedPrincipalIsInternalError pins the wiring-defect path.
// An operation that declares `security:` fails closed before a handler runs, so a
// handler that sees no principal is looking at a broken seam, not a caller whose
// valid credentials lack permission.
func TestMissingAuthenticatedPrincipalIsInternalError(t *testing.T) {
	t.Parallel()

	admitEveryone := func(context.Context, *openapi3filter.AuthenticationInput) error { return nil }
	handler, err := NewAPIHandler(mustArticleService(t), Options{
		Authenticate:   admitEveryone,
		RejectRequest:  testRejectRequest,
		RejectResponse: testRejectResponse,
	})
	if err != nil {
		t.Fatalf("NewAPIHandler() error = %v", err)
	}

	response := postArticle(t, handler,
		`{"slug":"unattributed","title":"Unattributed","summary":"No principal was published."}`)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
}

func mustArticleService(tb testing.TB) *article.Service {
	tb.Helper()

	repository := memory.New()
	service, err := article.NewService(repository)
	if err != nil {
		tb.Fatalf("article.NewService() error = %v", err)
	}
	return service
}
