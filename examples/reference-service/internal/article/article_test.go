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
	append func(context.Context, Event) error
	// commits counts how many units of work were committed, so a test can tell a
	// write that was rolled back from one that never ran.
	commits int
}

func (r *repositoryStub) FindBySlug(ctx context.Context, slug string) (Article, error) {
	return r.find(ctx, slug)
}

func (r *repositoryStub) Create(ctx context.Context, created Article) error {
	if r.create == nil {
		return nil
	}
	return r.create(ctx, created)
}

func (r *repositoryStub) AppendEvent(ctx context.Context, event Event) error {
	if r.append == nil {
		return nil
	}
	return r.append(ctx, event)
}

// Do stands in for a real transaction: it commits only when fn succeeds, which
// is the property the use case depends on and must not silently lose.
func (r *repositoryStub) Do(_ context.Context, fn func(Repository) error) error {
	if err := fn(r); err != nil {
		return err
	}
	r.commits++
	return nil
}

// newTestService builds the service over one stub that is both the repository
// and the unit of work, which is the shape a real adapter has.
func newTestService(tb testing.TB, stub *repositoryStub) *Service {
	tb.Helper()

	service, err := NewService(stub, stub)
	if err != nil {
		tb.Fatalf("NewService() error = %v", err)
	}
	return service
}

func TestServiceGetReturnsRepositoryArticle(t *testing.T) {
	t.Parallel()

	want := Article{Slug: "clear-owners", Title: "Clear owners", Summary: "Keep behavior with its owner.", Published: true}
	service := newTestService(t, &repositoryStub{find: func(_ context.Context, slug string) (Article, error) {
		if slug != want.Slug {
			t.Fatalf("FindBySlug() slug = %q, want %q", slug, want.Slug)
		}
		return want, nil
	}})

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

	service := newTestService(t, &repositoryStub{find: func(context.Context, string) (Article, error) {
		return Article{}, ErrNotFound
	}})

	_, err := service.Get(context.Background(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get() error = %v, want wrapped %v", err, ErrNotFound)
	}
}

func TestServiceGetHidesUnpublishedArticle(t *testing.T) {
	t.Parallel()

	service := newTestService(t, &repositoryStub{find: func(context.Context, string) (Article, error) {
		return Article{Slug: "draft"}, nil
	}})

	if _, err := service.Get(context.Background(), "draft"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get() error = %v, want %v", err, ErrNotFound)
	}
}

func TestNewServiceRejectsMissingRepository(t *testing.T) {
	t.Parallel()

	stub := &repositoryStub{}
	if _, err := NewService(nil, stub); err == nil {
		t.Fatal("NewService(nil repository) error = nil, want non-nil")
	}
	if _, err := NewService(stub, nil); err == nil {
		t.Fatal("NewService(nil unit of work) error = nil, want non-nil")
	}
}

func TestServiceCreatePublishesTrimmedArticle(t *testing.T) {
	t.Parallel()

	var stored Article
	var events []Event
	stub := &repositoryStub{
		find: func(context.Context, string) (Article, error) { return Article{}, ErrNotFound },
		create: func(_ context.Context, created Article) error {
			stored = created
			return nil
		},
		append: func(_ context.Context, event Event) error {
			events = append(events, event)
			return nil
		},
	}
	service := newTestService(t, stub)

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
	// The event is written inside the same unit of work as the article, which is
	// the whole point of routing Create through Atomically.
	if len(events) != 1 || events[0] != (Event{Slug: want.Slug, Kind: EventArticleCreated}) {
		t.Fatalf("events = %+v, want one %s for %q", events, EventArticleCreated, want.Slug)
	}
	if stub.commits != 1 {
		t.Fatalf("commits = %d, want 1", stub.commits)
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
			service := newTestService(t, &repositoryStub{
				find:   func(context.Context, string) (Article, error) { return Article{}, ErrNotFound },
				create: func(context.Context, Article) error { created = true; return nil },
			})

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

	service := newTestService(t, &repositoryStub{
		find:   func(context.Context, string) (Article, error) { return Article{}, ErrNotFound },
		create: func(context.Context, Article) error { return ErrAlreadyExists },
	})

	if _, err := service.Create(context.Background(), Draft{Slug: "clear", Title: "t", Summary: "s"}); !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("Create() error = %v, want %v", err, ErrAlreadyExists)
	}
}

// TestServiceCreateRollsBackWhenTheEventFails is the property the Atomically port
// exists for. Without it the two writes are independent, and a failure between
// them leaves an article nobody was told about — the exact state a compensating
// job gets written for, permanently.
func TestServiceCreateRollsBackWhenTheEventFails(t *testing.T) {
	t.Parallel()

	appendErr := errors.New("event log is unavailable")
	created := false
	stub := &repositoryStub{
		find:   func(context.Context, string) (Article, error) { return Article{}, ErrNotFound },
		create: func(context.Context, Article) error { created = true; return nil },
		append: func(context.Context, Event) error { return appendErr },
	}
	service := newTestService(t, stub)

	_, err := service.Create(context.Background(), Draft{Slug: "clear", Title: "t", Summary: "s"})
	if !errors.Is(err, appendErr) {
		t.Fatalf("Create() error = %v, want wrapped %v", err, appendErr)
	}
	if !created {
		t.Fatal("the article write never ran, so this proves nothing about rollback")
	}
	if stub.commits != 0 {
		t.Fatalf("commits = %d, want 0 so the article write is rolled back with the event", stub.commits)
	}
}
