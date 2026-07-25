// Package article owns the reference feature's business model and use case.
package article

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var slugPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)

var (
	ErrNotFound      = errors.New("article not found")
	ErrAlreadyExists = errors.New("article already exists")
	ErrInvalid       = errors.New("article is invalid")
)

const (
	maxTitleLength   = 200
	maxSummaryLength = 500
)

type Article struct {
	Slug      string
	Title     string
	Summary   string
	Published bool
}

// Draft is the caller-supplied shape of a new article. It is a feature-owned
// type: the transport maps generated OpenAPI types onto it so business rules
// never depend on the wire contract.
type Draft struct {
	Slug    string
	Title   string
	Summary string
}

// Repository is owned by the use case that consumes article storage.
type Repository interface {
	FindBySlug(ctx context.Context, slug string) (Article, error)
	// Create stores a new article and returns ErrAlreadyExists when the slug
	// is taken. Uniqueness is enforced by the adapter, which is the only layer
	// that can make the check and the write atomic.
	Create(ctx context.Context, created Article) error
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

// Create validates a draft and stores it as a published article.
//
// Validation is deliberately duplicated with the OpenAPI schema: the contract
// rejects malformed requests at the edge, and this check keeps the invariant
// true for any other caller of the use case.
func (s *Service) Create(ctx context.Context, draft Draft) (Article, error) {
	created := Article{
		Slug:      strings.TrimSpace(draft.Slug),
		Title:     strings.TrimSpace(draft.Title),
		Summary:   strings.TrimSpace(draft.Summary),
		Published: true,
	}
	if err := validateDraft(created); err != nil {
		return Article{}, err
	}

	// Wrapping preserves the sentinel identity for errors.Is while still adding
	// the operation that failed.
	if err := s.repository.Create(ctx, created); err != nil {
		return Article{}, fmt.Errorf("create article: %w", err)
	}
	return created, nil
}

func validateDraft(candidate Article) error {
	if !slugPattern.MatchString(candidate.Slug) {
		return fmt.Errorf("%w: slug must match %s", ErrInvalid, slugPattern)
	}
	if candidate.Title == "" || len(candidate.Title) > maxTitleLength {
		return fmt.Errorf("%w: title must be 1..%d characters", ErrInvalid, maxTitleLength)
	}
	if candidate.Summary == "" || len(candidate.Summary) > maxSummaryLength {
		return fmt.Errorf("%w: summary must be 1..%d characters", ErrInvalid, maxSummaryLength)
	}
	return nil
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
