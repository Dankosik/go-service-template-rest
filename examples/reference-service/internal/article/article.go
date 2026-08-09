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

// EventKind names something that happened to an article.
type EventKind string

const EventArticleCreated EventKind = "article.created"

// Event is the record a downstream consumer eventually reads. It stands in for
// the outbox row a real service publishes from, which is the ordinary reason a
// use case needs two writes to succeed or fail together.
type Event struct {
	Slug string
	Kind EventKind
}

// Repository is owned by the use case that consumes article storage.
type Repository interface {
	FindBySlug(ctx context.Context, slug string) (Article, error)
	// Create stores a new article and returns ErrAlreadyExists when the slug
	// is taken. Uniqueness is enforced by the adapter, which is the only layer
	// that can make the check and the write atomic.
	Create(ctx context.Context, created Article) error
	// AppendEvent records that something happened. It must be written in the
	// same transaction as the change it describes; see Atomically.
	AppendEvent(ctx context.Context, event Event) error
}

// Atomically runs several repository calls as one unit of work.
//
// This is the port that keeps a transaction from leaking into the domain. The
// repository-level answer this template ships is postgres.Pool.InTx, which yields
// a pgx.Tx — and the repository contract forbids a feature package from importing
// a concrete infra adapter, so a use case cannot call it and must not want to. The
// shape that resolves it is this one: the feature declares what it needs, the
// adapter binds it to whatever transaction its datastore has, and fn receives a
// Repository rather than a driver handle.
//
// Getting this wrong has one shape, and it is always the same one: the second
// write is moved outside the transaction "for now". Then a crash between the two
// leaves an article nobody was told about, or an event for an article that was
// rolled back — and the compensating job written to clean that up becomes
// permanent.
//
// fn returning an error rolls everything back. A partial success is never
// observable.
type Atomically interface {
	Do(ctx context.Context, fn func(Repository) error) error
}

type Service struct {
	repository Repository
	atomically Atomically
}

func NewService(repository Repository, atomically Atomically) (*Service, error) {
	if repository == nil {
		return nil, errors.New("article service: repository is required")
	}
	if atomically == nil {
		return nil, errors.New("article service: unit of work is required")
	}
	return &Service{repository: repository, atomically: atomically}, nil
}

// Create validates a draft and stores it as a published article, together with
// the event that announces it.
//
// Validation is deliberately duplicated with the OpenAPI schema: the contract
// rejects malformed requests at the edge, and this check keeps the invariant
// true for any other caller of the use case.
//
// The two writes are one unit of work. Nothing here knows what kind of
// transaction that is.
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
	err := s.atomically.Do(ctx, func(repository Repository) error {
		if createErr := repository.Create(ctx, created); createErr != nil {
			return fmt.Errorf("store article: %w", createErr)
		}
		if appendErr := repository.AppendEvent(ctx, Event{Slug: created.Slug, Kind: EventArticleCreated}); appendErr != nil {
			return fmt.Errorf("append %s event: %w", EventArticleCreated, appendErr)
		}
		return nil
	})
	if err != nil {
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
