package memory

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/example/go-service-template-rest/examples/reference-service/internal/article"
)

func TestRepositoryFindBySlug(t *testing.T) {
	t.Parallel()

	want := article.Article{Slug: "small-slices", Title: "Small slices", Summary: "Prove one path at a time.", Published: true}
	repository := New()
	if err := repository.Do(t.Context(), func(writer article.Writer) error {
		if err := writer.Create(t.Context(), want); err != nil {
			return fmt.Errorf("create fixture article: %w", err)
		}
		return nil
	}); err != nil {
		t.Fatalf("seed article: %v", err)
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

func TestRepositoryRollsBackDuplicateSlug(t *testing.T) {
	t.Parallel()

	repository := New()
	created := article.Article{Slug: "direct-write", Title: "Direct write", Published: true}

	err := repository.Do(t.Context(), func(writer article.Writer) error {
		if err := writer.Create(t.Context(), created); err != nil {
			return fmt.Errorf("create first article: %w", err)
		}
		return writer.Create(t.Context(), created)
	})
	if !errors.Is(err, article.ErrAlreadyExists) {
		t.Fatalf("Do(duplicate) error = %v, want %v", err, article.ErrAlreadyExists)
	}
	if _, err := repository.FindBySlug(context.Background(), created.Slug); !errors.Is(err, article.ErrNotFound) {
		t.Fatalf("FindBySlug() error = %v, want rollback with %v", err, article.ErrNotFound)
	}
}
