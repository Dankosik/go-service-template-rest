// Package article owns the reference feature's business model and use case.
package article

import (
	"context"
	"errors"
	"fmt"
)

var ErrNotFound = errors.New("article not found")

type Article struct {
	Slug      string
	Title     string
	Summary   string
	Published bool
}

// Repository is owned by the use case that consumes article storage.
type Repository interface {
	FindBySlug(ctx context.Context, slug string) (Article, error)
}

type Service struct {
	repository Repository
}

func NewService(repository Repository) (*Service, error) {
	if repository == nil {
		return nil, errors.New("article service: repository is required")
	}
	return &Service{repository: repository}, nil
}

func (s *Service) Get(ctx context.Context, slug string) (Article, error) {
	found, err := s.repository.FindBySlug(ctx, slug)
	if err != nil {
		return Article{}, fmt.Errorf("get article: %w", err)
	}
	if !found.Published {
		return Article{}, ErrNotFound
	}
	return found, nil
}
