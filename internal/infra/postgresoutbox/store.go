package postgresoutbox

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5"
)

// Store owns every outbox statement, for three audiences. Feature code calls
// only Append, inside the transaction that owns its domain mutation. The relay
// owns Claim, the Mark and ScheduleRetry family, CleanupPublished, and Observe,
// plus Get, which it uses to resolve a finalization the batch statement did not
// report. Redrive, and Get again, are operator tooling.
//
// Those audiences are the file layout, one file per step of the cycle doc.go
// describes: store_append.go, store_claim.go, store_finalize.go,
// store_maintenance.go, and store_operator.go, over the row mapping and
// identity checks in store_rows.go. This file owns only what all of them share.
// The store prefix is load-bearing — see doc.go.
//
// Get therefore has two callers with different needs: Relay.reconcilePublished
// branches on both its error and the returned LeaseToken, so its error semantics
// are relay behavior as well as operator ergonomics.
//
// The Mark family is split by disposition rather than by arity: Relay.classify
// sorts a batch into ordered and unordered claims once, so no method here
// re-derives that split from the event it was handed.
//
// NewRelay does not mutate the Store it is given; when their telemetry differs it
// works through a copy carrying its own recorder.
//
// NewStore is the only constructor and it rejects a nil pool. The remaining
// guards exist for the zero value an exported type cannot prevent: every exported
// method opens with valid(), so a method added later cannot admit a half-built
// Store.
type Store struct {
	pool      *postgres.Pool
	queries   *sqlcgen.Queries
	telemetry *Telemetry
}

func NewStore(pool *postgres.Pool, telemetry *Telemetry) (*Store, error) {
	if pool == nil || pool.PGX() == nil {
		return nil, fmt.Errorf("%w: postgres pool is required", ErrConfig)
	}
	return &Store{pool: pool, queries: sqlcgen.New(pool.PGX()), telemetry: telemetry}, nil
}

// valid reports whether this Store came from NewStore. It is the one predicate
// every exported method uses, so a method added later cannot accidentally admit
// a half-built Store by testing only the field it happens to read.
func (s *Store) valid() bool {
	return s != nil && s.pool != nil && s.queries != nil
}

// errStoreRequired is what every exported method returns for a Store that never
// came from NewStore. One message, so the guard reads the same everywhere and a
// new method copies a line rather than inventing wording.
func errStoreRequired() error {
	return fmt.Errorf("%w: store is required", ErrConfig)
}

// listenerConfig is the connection the append listener opens for itself. It
// inherits the pool's validated DSN, TLS, and timeouts without ever competing
// for a pooled connection.
func (s *Store) listenerConfig() *pgx.ConnConfig {
	return s.pool.PGX().Config().ConnConfig
}

func (s *Store) withTelemetry(telemetry *Telemetry) *Store {
	if s == nil || telemetry == nil || s.telemetry == telemetry {
		return s
	}
	return &Store{pool: s.pool, queries: s.queries, telemetry: telemetry}
}

// The nil check here and in recordOperation is for the receiver, not for the
// recorder: a nil *Telemetry is already a working no-op, but these run from
// deferred calls that a zero-value Store can reach before its entry-point guard
// returns.
func (s *Store) recordClaim(ctx context.Context, batch ClaimedBatch, started time.Time, err error) {
	if s == nil {
		return
	}
	if err == nil && len(batch.Events) == 0 {
		s.telemetry.RecordOperation(ctx, "claim", "empty", classNone, time.Since(started))
		return
	}
	s.recordOperation(ctx, "claim", started, err)
}

func (s *Store) recordOperation(ctx context.Context, operation string, started time.Time, err error) {
	if s == nil {
		return
	}
	outcome, errorClass := storeOutcome(err)
	s.telemetry.RecordOperation(ctx, operation, outcome, errorClass, time.Since(started))
}

// storeOutcome classes one store failure into the outcome and error class its
// metric sample carries. It is the only producer of the lost-lease, validation,
// and database classes, so TestErrorClassVocabularyIsBounded drives it rather
// than restating those literals.
//
// Every statement in this package reaches PostgreSQL, so anything this switch
// does not recognize is a database error. A new sentinel the store can
// distinguish belongs here and in [boundedErrorType].
func storeOutcome(err error) (outcome, errorClass string) {
	switch {
	case err == nil:
		return outcomeSuccess, classNone
	case errors.Is(err, ErrLeaseLost):
		return outcomeError, classLostLease
	case errors.Is(err, ErrInvalidEvent), errors.Is(err, ErrConfig), errors.Is(err, ErrOrderingSequence),
		errors.Is(err, ErrRedriveRejected), errors.Is(err, ErrRedriveConflict):
		return outcomeRejected, classValidation
	default:
		return outcomeError, classDatabase
	}
}
