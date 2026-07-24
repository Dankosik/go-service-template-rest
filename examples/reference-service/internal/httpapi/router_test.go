package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/go-service-template-rest/examples/reference-service/internal/article"
	"github.com/example/go-service-template-rest/examples/reference-service/internal/article/memory"
	"github.com/example/go-service-template-rest/examples/reference-service/internal/openapi"
)

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
	router, err := NewRouter(service)
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
	router, err := NewRouter(service)
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
	router, err := NewRouter(service)
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
