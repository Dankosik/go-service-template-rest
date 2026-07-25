package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/example/go-service-template-rest/examples/reference-service/internal/article"
	"github.com/example/go-service-template-rest/examples/reference-service/internal/article/memory"
	"github.com/example/go-service-template-rest/examples/reference-service/internal/openapi"
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
	router, err := NewRouter(service, testWriteToken)
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}

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
	router, err := NewRouter(service, testWriteToken)
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}

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
	router, err := NewRouter(service, testWriteToken)
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}

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
	router, err := NewRouter(service, testWriteToken)
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}
	return router
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
			router, err := NewRouter(service, testWriteToken)
			if err != nil {
				t.Fatalf("NewRouter() error = %v", err)
			}

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
	if _, err := NewRouter(service, "   "); err == nil {
		t.Fatal("NewRouter() error = nil, want rejection of an empty write token")
	}
}
