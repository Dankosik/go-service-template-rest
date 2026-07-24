package article

import (
	"context"
	"errors"
	"testing"
)

type repositoryStub struct {
	find func(context.Context, string) (Article, error)
}

func (r repositoryStub) FindBySlug(ctx context.Context, slug string) (Article, error) {
	return r.find(ctx, slug)
}

func TestServiceGetReturnsRepositoryArticle(t *testing.T) {
	t.Parallel()

	want := Article{Slug: "clear-owners", Title: "Clear owners", Summary: "Keep behavior with its owner.", Published: true}
	service, err := NewService(repositoryStub{find: func(_ context.Context, slug string) (Article, error) {
		if slug != want.Slug {
			t.Fatalf("FindBySlug() slug = %q, want %q", slug, want.Slug)
		}
		return want, nil
	}})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	got, err := service.Get(context.Background(), want.Slug)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got != want {
		t.Fatalf("Get() = %+v, want %+v", got, want)
	}
}

func TestServiceGetPreservesNotFoundIdentity(t *testing.T) {
	t.Parallel()

	service, err := NewService(repositoryStub{find: func(context.Context, string) (Article, error) {
		return Article{}, ErrNotFound
	}})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	_, err = service.Get(context.Background(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get() error = %v, want wrapped %v", err, ErrNotFound)
	}
}

func TestServiceGetHidesUnpublishedArticle(t *testing.T) {
	t.Parallel()

	service, err := NewService(repositoryStub{find: func(context.Context, string) (Article, error) {
		return Article{Slug: "draft"}, nil
	}})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	if _, err := service.Get(context.Background(), "draft"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get() error = %v, want %v", err, ErrNotFound)
	}
}

func TestNewServiceRejectsMissingRepository(t *testing.T) {
	t.Parallel()

	if _, err := NewService(nil); err == nil {
		t.Fatal("NewService(nil) error = nil, want non-nil")
	}
}
