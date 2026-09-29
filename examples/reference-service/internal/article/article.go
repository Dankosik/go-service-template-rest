// Package article owns the reference feature's business model and use case.
package article

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"unicode/utf16"

	"github.com/example/go-service-template-rest/internal/failure"
)

var slugPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)

const (
	maxTitleUTF16CodeUnits   = 200
	maxSummaryUTF16CodeUnits = 500
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

// Writer is the transactional view Create needs.
type Writer interface {
	// Create stores a new article and returns ErrAlreadyExists when the slug
	// is taken. Uniqueness is enforced by the adapter, which is the only layer
	// that can make the check and the write atomic.
	Create(ctx context.Context, created Article) error
	// AppendEvent records that something happened. It must be written in the
	// same transaction as the change it describes; Store.Do owns that unit of
	// work.
	AppendEvent(ctx context.Context, event Event) error
}

// Store owns reads and the unit of work that commits article writes together.
//
// This is the port that keeps a transaction from leaking into the domain. The
// repository-level answer this template ships is postgres.InTx, which yields
// a pgx.Tx — and the repository contract forbids a feature package from importing
// a concrete infra adapter, so a use case cannot call it and must not want to. The
// shape that resolves it is this one: the feature declares what it needs, the
// adapter binds it to whatever transaction its datastore has, and fn receives a
// Writer rather than a driver handle.
//
// Getting this wrong has one shape, and it is always the same one: the second
// write is moved outside the transaction "for now". Then a crash between the two
// leaves an article nobody was told about, or an event for an article that was
// rolled back — and the compensating job written to clean that up becomes
// permanent.
//
// fn returning an error rolls everything back. A partial success is never
// observable.
type Store interface {
	// FindBySlug returns ErrNotFound when no article has slug; the adapter
	// translates its own "no row" result into that identity.
	FindBySlug(ctx context.Context, slug string) (Article, error)
	// Do calls fn with a transactional Writer. All Writer operations finish
	// before fn returns; fn must not retain the Writer after that call.
	Do(ctx context.Context, fn func(Writer) error) error
}

type Service struct {
	store Store
}

func NewService(store Store) (*Service, error) {
	if store == nil {
		return nil, errors.New("article service: store is required")
	}
	return &Service{store: store}, nil
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
	if err := validateDraft(draft); err != nil {
		return Article{}, err
	}
	created := Article{
		Slug:      draft.Slug,
		Title:     draft.Title,
		Summary:   draft.Summary,
		Published: true,
	}

	// failure.Op rather than fmt.Errorf, because this unit of work has two writes
	// that fail the same way. Both preserve the sentinel identity errors.Is matches
	// on; only Op survives into the record a transport writes for an unclassified
	// failure, and without it an operator sees one chain of *fmt.wrapError and
	// cannot tell which of the two broke.
	err := s.store.Do(ctx, func(writer Writer) error {
		if createErr := writer.Create(ctx, created); createErr != nil {
			return failure.Op("store article", createErr)
		}
		if appendErr := writer.AppendEvent(ctx, Event{Slug: created.Slug, Kind: EventArticleCreated}); appendErr != nil {
			return failure.Op("append event", appendErr)
		}
		return nil
	})
	if err != nil {
		return Article{}, failure.Op("create article", err)
	}
	return created, nil
}

func validateDraft(draft Draft) error {
	if !slugPattern.MatchString(draft.Slug) {
		return fmt.Errorf("%w: slug must match %s", ErrInvalid, slugPattern)
	}
	if draft.Title == "" || utf16CodeUnitCount(draft.Title) > maxTitleUTF16CodeUnits {
		return fmt.Errorf("%w: title must be 1..%d characters", ErrInvalid, maxTitleUTF16CodeUnits)
	}
	if draft.Summary == "" || utf16CodeUnitCount(draft.Summary) > maxSummaryUTF16CodeUnits {
		return fmt.Errorf("%w: summary must be 1..%d characters", ErrInvalid, maxSummaryUTF16CodeUnits)
	}
	return nil
}

func utf16CodeUnitCount(value string) int {
	length := 0
	for _, r := range value {
		length += utf16.RuneLen(r)
	}
	return length
}

func (s *Service) Get(ctx context.Context, slug string) (Article, error) {
	found, err := s.store.FindBySlug(ctx, slug)
	if err != nil {
		return Article{}, failure.Op("get article", err)
	}
	if !found.Published {
		return Article{}, ErrNotFound
	}
	return found, nil
}
