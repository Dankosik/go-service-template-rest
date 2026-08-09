package memory

import (
	"context"
	"errors"
	"testing"

	"github.com/example/go-service-template-rest/examples/reference-service/internal/article"
)

func TestRepositoryFindBySlug(t *testing.T) {
	t.Parallel()

	want := article.Article{Slug: "small-slices", Title: "Small slices", Summary: "Prove one path at a time.", Published: true}
	repository, err := New([]article.Article{want})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	got, err := repository.FindBySlug(context.Background(), want.Slug)
	if err != nil {
		t.Fatalf("FindBySlug() error = %v", err)
	}
	if got != want {
		t.Fatalf("FindBySlug() = %+v, want %+v", got, want)
	}

	if _, err := repository.FindBySlug(context.Background(), "missing"); !errors.Is(err, article.ErrNotFound) {
		t.Fatalf("FindBySlug(missing) error = %v, want %v", err, article.ErrNotFound)
	}
}

func TestRepositoryRejectsDuplicateSlug(t *testing.T) {
	t.Parallel()

	_, err := New([]article.Article{{Slug: "same"}, {Slug: "same"}})
	if err == nil {
		t.Fatal("New() error = nil, want duplicate rejection")
	}
}

// TestRepositoryCreateAnswersTheSameSlugRuleAsStaged covers the port methods
// production traffic does not reach. The use case writes through Do, so the
// staged view is what serves a real request; *Repository implements the same port
// directly and answered the slug rule from its own copy of it. Both now call one,
// and this is what would notice if only one of them stopped.
func TestRepositoryCreateAnswersTheSameSlugRuleAsStaged(t *testing.T) {
	t.Parallel()

	repository, err := New(nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	created := article.Article{Slug: "direct-write", Title: "Direct write", Published: true}

	if err := repository.Create(context.Background(), created); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	got, err := repository.FindBySlug(context.Background(), created.Slug)
	if err != nil {
		t.Fatalf("FindBySlug() after Create() error = %v", err)
	}
	if got != created {
		t.Fatalf("FindBySlug() = %+v, want %+v", got, created)
	}
	if err := repository.Create(context.Background(), created); !errors.Is(err, article.ErrAlreadyExists) {
		t.Fatalf("Create(duplicate) error = %v, want %v", err, article.ErrAlreadyExists)
	}
	if _, err := repository.FindBySlug(context.Background(), "absent"); !errors.Is(err, article.ErrNotFound) {
		t.Fatalf("FindBySlug(absent) error = %v, want %v", err, article.ErrNotFound)
	}
}
