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
	"github.com/getkin/kin-openapi/openapi3filter"
)

const testWriteToken = "reference-write-token"

func TestRouterGetArticle(t *testing.T) {
	t.Parallel()

	want := article.Article{Slug: "clear-owners", Title: "Clear owners", Summary: "Keep behavior with its owner.", Published: true}
	repository, err := memory.New([]article.Article{want})
	if err != nil {
		t.Fatalf("memory.New() error = %v", err)
	}
	service, err := article.NewService(repository)
	if err != nil {
		t.Fatalf("article.NewService() error = %v", err)
	}
	router := mustNewRouter(t, service)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/articles/"+want.Slug, nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusOK, response.Body.String())
	}
	var body openapi.Article
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Slug != want.Slug || body.Title != want.Title || body.Summary != want.Summary {
		t.Fatalf("body = %+v, want %+v", body, want)
	}
}

func TestRouterMapsMissingArticleToProblem(t *testing.T) {
	t.Parallel()

	repository, err := memory.New(nil)
	if err != nil {
		t.Fatalf("memory.New() error = %v", err)
	}
	service, err := article.NewService(repository)
	if err != nil {
		t.Fatalf("article.NewService() error = %v", err)
	}
	router := mustNewRouter(t, service)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/articles/missing", nil))

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusNotFound, response.Body.String())
	}
	var body openapi.Problem
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Code != "not_found" || body.Status != http.StatusNotFound {
		t.Fatalf("body = %+v, want not_found Problem", body)
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
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/articles/INVALID", nil))

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

// newTestRouter builds the router over an empty in-memory repository.
func newTestRouter(t *testing.T, seed ...article.Article) http.Handler {
	t.Helper()

	repository, err := memory.New(seed)
	if err != nil {
		t.Fatalf("memory.New() error = %v", err)
	}
	service, err := article.NewService(repository)
	if err != nil {
		t.Fatalf("article.NewService() error = %v", err)
	}
	return mustNewRouter(t, service)
}

// mustNewRouter builds the example router with the same hardened chain the binary
// serves, so these tests exercise the middleware a reader inherits by copying it.
// mustNewRouter builds the API handler this package owns.
//
// The reject mappers are local test doubles rather than the shared ones from the
// transport adapter: a feature package must not import that adapter, so the real
// mapping is proved at the composition root instead. See
// TestReferenceRouterInheritsHardenedChain in the reference binary's tests.
func mustNewRouter(tb testing.TB, service *article.Service) http.Handler {
	tb.Helper()

	handler, err := NewAPIHandler(service, Options{
		WriteToken:     testWriteToken,
		RejectRequest:  testRejectRequest,
		RejectResponse: testRejectResponse,
	})
	if err != nil {
		tb.Fatalf("NewAPIHandler() error = %v", err)
	}
	return handler
}

func testRejectRequest(w http.ResponseWriter, _ *http.Request, err error) {
	var securityErr *openapi3filter.SecurityRequirementsError
	if errors.As(err, &securityErr) {
		w.Header().Set("WWW-Authenticate", "Bearer")
		writeTestProblem(w, problem("unauthorized", "unauthorized", http.StatusUnauthorized, "credentials are missing or invalid"))
		return
	}
	writeTestProblem(w, problem("bad_request", "bad request", http.StatusBadRequest, "request is malformed or invalid"))
}

func testRejectResponse(w http.ResponseWriter, _ *http.Request, _ error) {
	writeTestProblem(w, problem("internal_error", "internal server error", http.StatusInternalServerError, "request failed"))
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

func postArticle(t *testing.T, router http.Handler, token, body string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(http.MethodPost, "/api/v1/articles", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func TestRouterCreateArticleReturnsCreatedWithLocation(t *testing.T) {
	t.Parallel()

	router := newTestRouter(t)
	response := postArticle(t, router, testWriteToken,
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
	router.ServeHTTP(read, httptest.NewRequest(http.MethodGet, "/api/v1/articles/clear-owners", nil))
	if read.Code != http.StatusOK {
		t.Fatalf("read-back status = %d, want %d; body = %s", read.Code, http.StatusOK, read.Body.String())
	}
}

func TestRouterCreateArticleRequiresCredentials(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name  string
		token string
	}{
		{name: "missing credential", token: ""},
		{name: "wrong credential", token: "not-the-token"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			response := postArticle(t, newTestRouter(t), tt.token,
				`{"slug":"clear-owners","title":"Clear owners","summary":"Keep behavior with its owner."}`)

			if response.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusUnauthorized, response.Body.String())
			}
			if got := response.Header().Get("WWW-Authenticate"); got != "Bearer" {
				t.Fatalf("WWW-Authenticate = %q, want %q", got, "Bearer")
			}
			var body openapi.Problem
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if body.Code != "unauthorized" || body.Status != http.StatusUnauthorized {
				t.Fatalf("body = %+v, want unauthorized Problem", body)
			}
		})
	}
}

func TestRouterCreateArticleRejectsDuplicateSlugWithConflict(t *testing.T) {
	t.Parallel()

	router := newTestRouter(t, article.Article{
		Slug: "clear-owners", Title: "Clear owners", Summary: "Existing.", Published: true,
	})
	response := postArticle(t, router, testWriteToken,
		`{"slug":"clear-owners","title":"Another title","summary":"Another summary."}`)

	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusConflict, response.Body.String())
	}
	var body openapi.Problem
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Code != "conflict" || body.Status != http.StatusConflict {
		t.Fatalf("body = %+v, want conflict Problem", body)
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

			response := postArticle(t, router, testWriteToken, tt.body)
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

func TestNewRouterRejectsEmptyWriteToken(t *testing.T) {
	t.Parallel()

	repository, err := memory.New(nil)
	if err != nil {
		t.Fatalf("memory.New() error = %v", err)
	}
	service, err := article.NewService(repository)
	if err != nil {
		t.Fatalf("article.NewService() error = %v", err)
	}
	_, err = NewAPIHandler(service, Options{
		WriteToken:     "   ",
		RejectRequest:  testRejectRequest,
		RejectResponse: testRejectResponse,
	})
	if err == nil {
		t.Fatal("NewAPIHandler() error = nil, want rejection of an empty write token")
	}
	if _, err := NewAPIHandler(service, Options{WriteToken: testWriteToken}); err == nil {
		t.Fatal("NewAPIHandler() error = nil, want rejection of missing reject mappers")
	}
}
