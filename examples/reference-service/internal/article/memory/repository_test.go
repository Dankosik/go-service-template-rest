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
