// Package memory adapts immutable in-memory data to the article repository port.
package memory

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sync"

	"github.com/example/go-service-template-rest/examples/reference-service/internal/article"
)

var (
	_ article.Repository = (*Repository)(nil)
	_ article.Atomically = (*Repository)(nil)
	_ article.Repository = (*staged)(nil)
)

type Repository struct {
	// mu guards the state below: writes arrive concurrently from the HTTP server,
	// and the uniqueness check plus the insert must be one atomic step.
	mu     sync.RWMutex
	bySlug map[string]article.Article
	events []article.Event
}

func New(articles []article.Article) (*Repository, error) {
	bySlug := make(map[string]article.Article, len(articles))
	for _, item := range articles {
		if item.Slug == "" {
			return nil, errors.New("memory article repository: slug is required")
		}
		if _, exists := bySlug[item.Slug]; exists {
			return nil, fmt.Errorf("memory article repository: duplicate slug %q", item.Slug)
		}
		bySlug[item.Slug] = item
	}
	return &Repository{bySlug: bySlug}, nil
}

// Do runs fn against a staged copy of the whole store and keeps the result only
// when fn succeeds, which is this adapter's version of a transaction.
//
// A datastore adapter binds the same port to its own transaction — for
// PostgreSQL, postgres.Pool.InTx, with fn handed a repository built over the
// pgx.Tx rather than over the pool. What must not change is the signature: fn
// receives an article.Repository, so the use case never sees a driver handle.
func (r *Repository) Do(ctx context.Context, fn func(article.Repository) error) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("article unit of work: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	staged := &staged{
		bySlug: maps.Clone(r.bySlug),
		events: slices.Clone(r.events),
	}
	if err := fn(staged); err != nil {
		// Discarded rather than undone. The staged copy is the rollback.
		return err
	}
	r.bySlug = staged.bySlug
	r.events = staged.events
	return nil
}

// Events returns what has been recorded, for tests and for the demonstration
// that a rolled-back Create leaves none behind.
func (r *Repository) Events() []article.Event {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return slices.Clone(r.events)
}

// staged is the uncommitted view fn writes into. It holds no lock of its own: it
// is reachable only from inside Do, which already holds the write lock.
type staged struct {
	bySlug map[string]article.Article
	events []article.Event
}

func (s *staged) FindBySlug(ctx context.Context, slug string) (article.Article, error) {
	if err := ctx.Err(); err != nil {
		return article.Article{}, fmt.Errorf("find article: %w", err)
	}
	found, ok := s.bySlug[slug]
	if !ok {
		return article.Article{}, article.ErrNotFound
	}
	return found, nil
}

func (s *staged) Create(ctx context.Context, created article.Article) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("create article: %w", err)
	}
	if _, exists := s.bySlug[created.Slug]; exists {
		return article.ErrAlreadyExists
	}
	s.bySlug[created.Slug] = created
	return nil
}

func (s *staged) AppendEvent(ctx context.Context, event article.Event) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("append article event: %w", err)
	}
	s.events = append(s.events, event)
	return nil
}

func (r *Repository) FindBySlug(ctx context.Context, slug string) (article.Article, error) {
	if err := ctx.Err(); err != nil {
		return article.Article{}, fmt.Errorf("find article: %w", err)
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	found, ok := r.bySlug[slug]
	if !ok {
		return article.Article{}, article.ErrNotFound
	}
	return found, nil
}

// Create inserts the article if its slug is free. A real datastore adapter
// enforces this with a unique constraint and maps the driver's violation to
// article.ErrAlreadyExists rather than reading before writing.
func (r *Repository) Create(ctx context.Context, created article.Article) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("create article: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.bySlug[created.Slug]; exists {
		return article.ErrAlreadyExists
	}
	r.bySlug[created.Slug] = created
	return nil
}

// AppendEvent records an event outside any unit of work. The use case writes
// events through Do instead; this exists so *Repository satisfies the same port
// its staged view does.
func (r *Repository) AppendEvent(ctx context.Context, event article.Event) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("append article event: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, event)
	return nil
}
