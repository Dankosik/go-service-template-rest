// Package memory adapts immutable in-memory data to the article repository port.
package memory

import (
	"context"
	"fmt"
	"sync"

	"github.com/example/go-service-template-rest/examples/reference-service/internal/article"
)

type Repository struct {
	// mu guards bySlug: writes arrive concurrently from the HTTP server, and
	// the uniqueness check plus the insert must be one atomic step.
	mu     sync.RWMutex
	bySlug map[string]article.Article
}

func New(articles []article.Article) (*Repository, error) {
	bySlug := make(map[string]article.Article, len(articles))
	for _, item := range articles {
		if item.Slug == "" {
			return nil, fmt.Errorf("memory article repository: slug is required")
		}
		if _, exists := bySlug[item.Slug]; exists {
			return nil, fmt.Errorf("memory article repository: duplicate slug %q", item.Slug)
		}
		bySlug[item.Slug] = item
	}
	return &Repository{bySlug: bySlug}, nil
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
