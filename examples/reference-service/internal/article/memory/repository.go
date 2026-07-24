// Package memory adapts immutable in-memory data to the article repository port.
package memory

import (
	"context"
	"fmt"

	"github.com/example/go-service-template-rest/examples/reference-service/internal/article"
)

type Repository struct {
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
	found, ok := r.bySlug[slug]
	if !ok {
		return article.Article{}, article.ErrNotFound
	}
	return found, nil
}
