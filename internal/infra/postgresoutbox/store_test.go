package postgresoutbox

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5"
)

func TestStoreRejectsInvalidUseBeforeDatabaseAccess(t *testing.T) {
	t.Parallel()

	if _, err := NewStore(nil, nil); !errors.Is(err, ErrConfig) {
		t.Fatalf("NewStore(nil) error = %v", err)
	}
	// Constructed enough to pass the entry-point guards, so each assertion below
	// reaches the argument validation it is actually about. A zero-value Store
	// stops earlier, which TestZeroValueStoreRejectsEveryExportedMethod covers.
	store := &Store{pool: &postgres.Pool{}, queries: sqlcgen.New(nil)}
	if err := store.Append(t.Context(), nil, outboxEventForUnit()); !errors.Is(err, ErrConfig) {
		t.Fatalf("Append(nil tx) error = %v", err)
	}
	if _, err := store.Claim(t.Context(), 0, 1, 1); !errors.Is(err, ErrConfig) {
		t.Fatalf("Claim(0 lease) error = %v", err)
	}
	if _, err := store.Claim(t.Context(), time.Second, 0, 1); !errors.Is(err, ErrConfig) {
		t.Fatalf("Claim(0 batch) error = %v", err)
	}
	if err := store.MarkUnorderedPublished(t.Context(), "lease", ""); !errors.Is(err, ErrConfig) {
		t.Fatalf("MarkUnorderedPublished(invalid id) error = %v", err)
	}
	if err := store.MarkOrderedPublished(t.Context(), "lease", OrderedDirective{
		ID: "event", OrderingKey: "key", OrderingSequence: 0,
	}); !errors.Is(err, ErrConfig) {
		t.Fatalf("MarkOrderedPublished(invalid sequence) error = %v", err)
	}
	if _, err := store.MarkUnorderedPublishedBatch(t.Context(), "lease", nil); !errors.Is(err, ErrConfig) {
		t.Fatalf("MarkUnorderedPublishedBatch(empty) error = %v", err)
	}
	if _, err := store.MarkOrderedPublishedBatch(t.Context(), "lease", nil); !errors.Is(err, ErrConfig) {
		t.Fatalf("MarkOrderedPublishedBatch(empty) error = %v", err)
	}
	if _, err := store.MarkOrderedPublishedBatch(t.Context(), "lease", []OrderedDirective{
		{ID: "event", OrderingKey: "key", OrderingSequence: 0},
	}); !errors.Is(err, ErrConfig) {
		t.Fatalf("MarkOrderedPublishedBatch(invalid sequence) error = %v", err)
	}
	if err := store.ScheduleRetryBatch(t.Context(), "", []RetryDirective{
		{ID: "event", ErrorClass: "temporary", Delay: time.Second},
	}); !errors.Is(err, ErrConfig) {
		t.Fatalf("ScheduleRetryBatch(invalid token) error = %v", err)
	}
	if err := store.ScheduleRetryBatch(t.Context(), "lease", []RetryDirective{
		{ID: "event", ErrorClass: "temporary", Delay: -1},
	}); !errors.Is(err, ErrConfig) {
		t.Fatalf("ScheduleRetryBatch(negative delay) error = %v", err)
	}
	if err := store.MarkPoisonedBatch(t.Context(), "lease", []PoisonDirective{
		{ID: "event", ErrorClass: ""},
	}); !errors.Is(err, ErrConfig) {
		t.Fatalf("MarkPoisonedBatch(invalid class) error = %v", err)
	}
	if _, err := store.Get(t.Context(), ""); !errors.Is(err, ErrConfig) {
		t.Fatalf("Get(invalid id) error = %v", err)
	}
	if err := store.Redrive(t.Context(), "", "audit"); !errors.Is(err, ErrConfig) {
		t.Fatalf("Redrive(invalid id) error = %v", err)
	}
	if err := store.Redrive(t.Context(), "event", "audit"); !errors.Is(err, postgres.ErrConfig) {
		t.Fatalf("Redrive(unusable pool) error = %v", err)
	}
	if _, err := store.CleanupPublished(t.Context(), 0, 0); !errors.Is(err, ErrConfig) {
		t.Fatalf("CleanupPublished(invalid) error = %v", err)
	}
}

// ErrInvalidEvent means one Event failed Validate and nothing else, so no
// identity check may report it. A store method that borrowed that sentinel for
// a bad id, lease token, or error class would tell a caller to inspect an
// envelope that is not at fault — and would do it through the same
// validateText rules Event.Validate uses, which is exactly how the two drifted
// together before.
func TestStoreIdentityFailuresAreNotEventFailures(t *testing.T) {
	t.Parallel()

	store := &Store{pool: &postgres.Pool{}, queries: sqlcgen.New(nil)}
	for name, call := range map[string]func() error{
		"MarkUnorderedPublished empty id": func() error {
			return store.MarkUnorderedPublished(t.Context(), "lease", "")
		},
		"MarkOrderedPublished empty id": func() error {
			return store.MarkOrderedPublished(t.Context(), "lease", OrderedDirective{
				OrderingKey: "key", OrderingSequence: 1,
			})
		},
		"MarkUnorderedPublishedBatch empty token": func() error {
			_, err := store.MarkUnorderedPublishedBatch(t.Context(), "", []string{"event"})
			return err
		},
		"MarkOrderedPublishedBatch empty key": func() error {
			_, err := store.MarkOrderedPublishedBatch(t.Context(), "lease", []OrderedDirective{
				{ID: "event", OrderingKey: "", OrderingSequence: 1},
			})
			return err
		},
		"ScheduleRetryBatch oversized class": func() error {
			return store.ScheduleRetryBatch(t.Context(), "lease", []RetryDirective{
				{ID: "event", ErrorClass: strings.Repeat("c", maxErrorClassBytes+1)},
			})
		},
		"MarkPoisonedBatch empty class": func() error {
			return store.MarkPoisonedBatch(t.Context(), "lease", []PoisonDirective{{ID: "event"}})
		},
		"Get empty id":     func() error { _, err := store.Get(t.Context(), ""); return err },
		"Redrive empty id": func() error { return store.Redrive(t.Context(), "", "audit") },
	} {
		err := call()
		if !errors.Is(err, ErrConfig) {
			t.Errorf("%s error = %v, want ErrConfig", name, err)
		}
		if errors.Is(err, ErrInvalidEvent) {
			t.Errorf("%s reported ErrInvalidEvent; that sentinel belongs to Event.Validate", name)
		}
	}

	// The other direction: a rejected envelope still reports ErrInvalidEvent and
	// never ErrConfig, so the split is readable from both sides.
	invalid := outboxEventForUnit()
	invalid.ID = ""
	err := store.Append(t.Context(), &transactionStub{}, invalid)
	if !errors.Is(err, ErrInvalidEvent) || errors.Is(err, ErrConfig) {
		t.Errorf("Append(invalid event) error = %v, want ErrInvalidEvent alone", err)
	}
}

// Store is exported, so a zero value can reach a caller that never used
// NewStore. Every exported method opens with Store.valid, so each one rejects
// such a store instead of dereferencing a nil dependency — including the two
// half-built shapes that would otherwise pass whichever single field that
// method happens to read.
//
// The list below is every exported method, not a chosen subset: one with
// arguments valid enough to reach past its own checks otherwise panics on a nil
// *Queries, and the count assertion at the end is what fails when a new method is
// added and left out of both the guard and this table.
func TestZeroValueStoreRejectsEveryExportedMethod(t *testing.T) {
	t.Parallel()

	calls := map[string]func(*Store) error{
		"Append": func(s *Store) error {
			return s.Append(t.Context(), &transactionStub{}, outboxEventForUnit())
		},
		"Claim":           func(s *Store) error { _, err := s.Claim(t.Context(), time.Second, 1, 1); return err },
		"Get":             func(s *Store) error { _, err := s.Get(t.Context(), "event"); return err },
		"Observe":         func(s *Store) error { _, err := s.Observe(t.Context()); return err },
		"Redrive":         func(s *Store) error { return s.Redrive(t.Context(), "event", "audit") },
		"RedriveUnknown":  func(s *Store) error { return s.RedriveUnknown(t.Context(), "event", "audit") },
		"ConfirmAccepted": func(s *Store) error { return s.ConfirmAccepted(t.Context(), "event", "audit") },
		"ClassifyLegacyUncertainty": func(s *Store) error {
			_, err := s.ClassifyLegacyUncertainty(t.Context(), 3, 10)
			return err
		},
		"ReconcileCommit": func(s *Store) error {
			_, err := s.ReconcileCommit(t.Context(), outboxEventForUnit())
			return err
		},
		"MarkUnorderedPublished": func(s *Store) error {
			return s.MarkUnorderedPublished(t.Context(), "lease", "event")
		},
		"MarkOrderedPublished": func(s *Store) error {
			return s.MarkOrderedPublished(t.Context(), "lease", unitOrdered()[0])
		},
		"MarkUnorderedPublishedBatch": func(s *Store) error {
			_, err := s.MarkUnorderedPublishedBatch(t.Context(), "lease", []string{"event"})
			return err
		},
		"MarkOrderedPublishedBatch": func(s *Store) error {
			_, err := s.MarkOrderedPublishedBatch(t.Context(), "lease", unitOrdered())
			return err
		},
		"ScheduleRetryBatch": func(s *Store) error {
			return s.ScheduleRetryBatch(t.Context(), "lease", unitRetries())
		},
		"MarkPoisonedBatch": func(s *Store) error {
			return s.MarkPoisonedBatch(t.Context(), "lease", unitPoisons())
		},
		"CleanupPublished": func(s *Store) error {
			_, err := s.CleanupPublished(t.Context(), time.Hour, 10)
			return err
		},
		"RetireOrderingKeys": func(s *Store) error {
			return s.RetireOrderingKeys(t.Context(), &transactionStub{}, "key")
		},
	}
	if got, want := len(calls), reflect.TypeFor[*Store]().NumMethod(); got != want {
		t.Fatalf("this table drives %d exported methods, Store has %d; add the new one here and give it a valid() guard", got, want)
	}
	for _, store := range map[string]*Store{
		"zero":         {},
		"pool only":    {pool: &postgres.Pool{}},
		"queries only": {queries: sqlcgen.New(databaseStub{})},
		"nil":          nil,
	} {
		for name, call := range calls {
			if err := call(store); !errors.Is(err, ErrConfig) {
				t.Errorf("%s(half-built store) error = %v, want ErrConfig", name, err)
			}
		}
	}
}

func TestStoreProgressAndTelemetryHelpers(t *testing.T) {
	t.Parallel()

	if err := validateProgressIdentity("event", "lease"); err != nil {
		t.Fatalf("validateProgressIdentity() error = %v", err)
	}
	if err := validateErrorClass("temporary"); err != nil {
		t.Fatalf("validateErrorClass() error = %v", err)
	}
	databaseErr := errors.New("database")
	if err := leaseProgressError("mark", 1, 1, nil); err != nil {
		t.Fatalf("leaseProgressError(success) = %v", err)
	}
	if err := leaseProgressError("mark", 0, 1, nil); !errors.Is(err, ErrLeaseLost) {
		t.Fatalf("leaseProgressError(lost) = %v", err)
	}
	if err := leaseProgressError("mark", 0, 1, databaseErr); !errors.Is(err, databaseErr) {
		t.Fatalf("leaseProgressError(database) = %v", err)
	}
	if err := leaseProgressError("mark", 3, 3, nil); err != nil {
		t.Fatalf("leaseProgressError(success) = %v", err)
	}
	if err := leaseProgressError("mark", 2, 3, nil); !errors.Is(err, ErrLeaseLost) {
		t.Fatalf("leaseProgressError(short) = %v", err)
	}
	if err := leaseProgressError("mark", 0, 3, databaseErr); !errors.Is(err, databaseErr) {
		t.Fatalf("leaseProgressError(database) = %v", err)
	}

	telemetry := &Telemetry{}
	store := &Store{}
	if got := (*Store)(nil).withTelemetry(telemetry); got != nil {
		t.Errorf("(*Store)(nil).withTelemetry() = %p, want nil", got)
	}
	if got := store.withTelemetry(nil); got != store {
		t.Errorf("withTelemetry(nil) = %p, want the receiver %p unchanged", got, store)
	}
	if clone := store.withTelemetry(telemetry); clone == store || clone.telemetry != telemetry {
		t.Fatal("withTelemetry did not create instrumented view")
	}
	store.telemetry = telemetry
	if store.withTelemetry(telemetry) != store {
		t.Fatal("withTelemetry duplicated the same telemetry")
	}
	// Both recorders run from deferred calls that a zero-value Store can reach
	// before its entry-point guard returns, so the nil receiver is the claim
	// here. What each one records is asserted elsewhere: storeOutcome's
	// classification by TestErrorClassVocabularyIsBounded, and the empty-claim
	// outcome by TestTelemetryBoundedContract.
	(*Store)(nil).recordOperation(context.Background(), "claim", time.Now(), nil)
	(*Store)(nil).recordClaim(context.Background(), ClaimedBatch{}, time.Now(), nil)
}

// A missing row is an absent event rather than a database failure, so only Get
// collapses it into ErrNotFound.
func TestStoreMapsMissingRowToNotFound(t *testing.T) {
	t.Parallel()

	store := stubbedStore(databaseStub{rowErr: pgx.ErrNoRows})
	if _, err := store.Get(t.Context(), "event"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get(no rows) error = %v, want ErrNotFound", err)
	}
}

// Every statement reachable through a stubbed driver wraps its failure rather
// than reporting it as lost lease, absent event, or silent success. Append
// takes its transaction from the caller and Redrive owns one, so those two are
// proven against their own doubles and against real PostgreSQL instead.
func TestStorePropagatesDatabaseFailures(t *testing.T) {
	t.Parallel()

	databaseErr := errors.New("database")
	reads := stubbedStore(databaseStub{rowErr: databaseErr, queryErr: databaseErr})
	if _, err := reads.Claim(t.Context(), time.Second, 10, 5); !errors.Is(err, databaseErr) {
		t.Errorf("Claim(database) error = %v, want the driver failure", err)
	}
	if _, err := reads.Get(t.Context(), "event"); !errors.Is(err, databaseErr) {
		t.Errorf("Get(database) error = %v, want the driver failure", err)
	}
	if _, err := reads.Observe(t.Context()); !errors.Is(err, databaseErr) {
		t.Errorf("Observe(database) error = %v, want the driver failure", err)
	}

	writes := stubbedStore(databaseStub{execErr: databaseErr, queryErr: databaseErr})
	if err := writes.MarkPoisonedBatch(t.Context(), "lease", unitPoisons()); !errors.Is(err, databaseErr) {
		t.Errorf("MarkPoisonedBatch(database) error = %v, want the driver failure", err)
	}
	if _, err := writes.MarkUnorderedPublishedBatch(t.Context(), "lease", []string{"event"}); !errors.Is(err, databaseErr) {
		t.Errorf("MarkUnorderedPublishedBatch(database) error = %v, want the driver failure", err)
	}
	if _, err := writes.MarkOrderedPublishedBatch(t.Context(), "lease", unitOrdered()); !errors.Is(err, databaseErr) {
		t.Errorf("MarkOrderedPublishedBatch(database) error = %v, want the driver failure", err)
	}
	if _, err := writes.CleanupPublished(t.Context(), time.Hour, 10); !errors.Is(err, databaseErr) {
		t.Errorf("CleanupPublished(database) error = %v, want the driver failure", err)
	}
	if err := writes.ScheduleRetryBatch(t.Context(), "lease", unitRetries()); !errors.Is(err, databaseErr) {
		t.Errorf("ScheduleRetryBatch(database) error = %v, want the driver failure", err)
	}
	// MarkUnorderedPublished takes the single-event path, which reaches a
	// different statement from MarkUnorderedPublishedBatch and wraps through
	// leaseProgressError.
	if err := writes.MarkUnorderedPublished(t.Context(), "lease", "event"); !errors.Is(err, databaseErr) {
		t.Errorf("MarkUnorderedPublished(database) error = %v, want the driver failure", err)
	}
	if err := writes.MarkOrderedPublished(t.Context(), "lease", unitOrdered()[0]); !errors.Is(err, databaseErr) {
		t.Errorf("MarkOrderedPublished(database) error = %v, want the driver failure", err)
	}
}
