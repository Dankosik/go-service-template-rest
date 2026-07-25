package article

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type repositoryStub struct {
	find   func(context.Context, string) (Article, error)
	create func(context.Context, Article) error
}

func (r repositoryStub) FindBySlug(ctx context.Context, slug string) (Article, error) {
	return r.find(ctx, slug)
}

func (r repositoryStub) Create(ctx context.Context, created Article) error {
	if r.create == nil {
		return nil
	}
	return r.create(ctx, created)
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

func TestServiceCreatePublishesTrimmedArticle(t *testing.T) {
	t.Parallel()

	var stored Article
	service, err := NewService(repositoryStub{
		find: func(context.Context, string) (Article, error) { return Article{}, ErrNotFound },
		create: func(_ context.Context, created Article) error {
			stored = created
			return nil
		},
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	got, err := service.Create(context.Background(), Draft{
		Slug: "  clear-owners  ", Title: " Clear owners ", Summary: " Keep behavior with its owner. ",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	want := Article{Slug: "clear-owners", Title: "Clear owners", Summary: "Keep behavior with its owner.", Published: true}
	if got != want {
		t.Fatalf("Create() = %+v, want %+v", got, want)
	}
	if stored != want {
		t.Fatalf("stored = %+v, want %+v", stored, want)
	}
}

func TestServiceCreateRejectsInvalidDraft(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name  string
		draft Draft
	}{
		{name: "empty slug", draft: Draft{Title: "t", Summary: "s"}},
		{name: "uppercase slug", draft: Draft{Slug: "Clear", Title: "t", Summary: "s"}},
		{name: "slug starts with digit", draft: Draft{Slug: "1clear", Title: "t", Summary: "s"}},
		{name: "empty title", draft: Draft{Slug: "clear", Summary: "s"}},
		{name: "empty summary", draft: Draft{Slug: "clear", Title: "t"}},
		{name: "title too long", draft: Draft{Slug: "clear", Title: strings.Repeat("t", 201), Summary: "s"}},
		{name: "summary too long", draft: Draft{Slug: "clear", Title: "t", Summary: strings.Repeat("s", 501)}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			created := false
			service, err := NewService(repositoryStub{
				find:   func(context.Context, string) (Article, error) { return Article{}, ErrNotFound },
				create: func(context.Context, Article) error { created = true; return nil },
			})
			if err != nil {
				t.Fatalf("NewService() error = %v", err)
			}

			if _, err := service.Create(context.Background(), tt.draft); !errors.Is(err, ErrInvalid) {
				t.Fatalf("Create() error = %v, want %v", err, ErrInvalid)
			}
			// An invalid draft must never reach storage.
			if created {
				t.Fatal("repository Create was called for an invalid draft")
			}
		})
	}
}

func TestServiceCreatePreservesAlreadyExistsIdentity(t *testing.T) {
	t.Parallel()

	service, err := NewService(repositoryStub{
		find:   func(context.Context, string) (Article, error) { return Article{}, ErrNotFound },
		create: func(context.Context, Article) error { return ErrAlreadyExists },
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	if _, err := service.Create(context.Background(), Draft{Slug: "clear", Title: "t", Summary: "s"}); !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("Create() error = %v, want %v", err, ErrAlreadyExists)
	}
}
