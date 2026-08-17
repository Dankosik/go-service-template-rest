package postgresidempotency

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/httpidempotency"
	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestClassifyRowPreservesIdempotencyOutcomes(t *testing.T) {
	t.Parallel()

	contract, attempt, resolver := testIdempotencyInputs(t)
	store := Store{}
	fingerprint := attempt.Fingerprint

	for _, tc := range []struct {
		name        string
		row         storedRow
		wantOutcome httpidempotency.Outcome
		wantReserve httpidempotency.ReservationRecovery
	}{
		{
			name:        "recovery due reserved row executes once",
			row:         storedRow{phase: phaseReserved, generation: 3, provisionalVersion: fingerprint.Version, provisionalFingerprint: fingerprint.Digest[:], recoveryDue: true},
			wantOutcome: httpidempotency.OutcomeExecute, wantReserve: httpidempotency.ReservationRecoveryDue,
		},
		{
			name:        "matching reserved row remains in progress",
			row:         storedRow{phase: phaseReserved, generation: 3, provisionalVersion: fingerprint.Version, provisionalFingerprint: fingerprint.Digest[:]},
			wantOutcome: httpidempotency.OutcomeInProgress,
		},
		{
			name:        "different reserved request mismatches",
			row:         storedRow{phase: phaseReserved, generation: 3, provisionalVersion: fingerprint.Version, provisionalFingerprint: append([]byte(nil), fingerprint.Digest[:]...)},
			wantOutcome: httpidempotency.OutcomeMismatch,
		},
		{
			name:        "completed result expires only after fingerprint matches",
			row:         storedRow{phase: phaseCompleted, fingerprintVersion: fingerprint.Version, fingerprint: fingerprint.Digest[:], committedAt: testCommittedAt()},
			wantOutcome: httpidempotency.OutcomeExpired,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.name == "different reserved request mismatches" {
				tc.row.provisionalFingerprint[0] ^= 1
			}
			reservation, decision, err := store.classifyRow(t.Context(), contract, attempt, resolver, tc.row)
			if err != nil {
				t.Fatalf("classifyRow() error = %v", err)
			}
			if decision.Outcome != tc.wantOutcome {
				t.Fatalf("classifyRow() outcome = %q, want %q", decision.Outcome, tc.wantOutcome)
			}
			if reservation.Recovery != tc.wantReserve {
				t.Fatalf("classifyRow() reservation recovery = %q, want %q", reservation.Recovery, tc.wantReserve)
			}
		})
	}

	result, err := httpidempotency.EncodeResult(contract, httpidempotency.Result{
		Status: http.StatusCreated, MediaType: "application/json", Codec: "create/v1", Payload: []byte(`{"id":"widget-1"}`),
	})
	if err != nil {
		t.Fatalf("EncodeResult() error = %v", err)
	}
	_, decision, err := store.classifyRow(t.Context(), contract, attempt, resolver, storedRow{
		phase: phaseCompleted, fingerprintVersion: fingerprint.Version, fingerprint: fingerprint.Digest[:], result: result, committedAt: testCommittedAt(),
	})
	if err != nil {
		t.Fatalf("classifyRow() replay error = %v", err)
	}
	if decision.Outcome != httpidempotency.OutcomeReplay || decision.Result == nil || string(decision.Result.Payload) != `{"id":"widget-1"}` {
		t.Fatalf("classifyRow() replay = %#v, want decoded retained result", decision)
	}
}

func TestClassifyRowRejectsInvalidPersistedState(t *testing.T) {
	t.Parallel()

	contract, attempt, resolver := testIdempotencyInputs(t)
	store := Store{}
	fingerprint := attempt.Fingerprint

	for _, tc := range []struct {
		name string
		row  storedRow
	}{
		{name: "unknown phase", row: storedRow{phase: "other"}},
		{name: "reserved without generation", row: storedRow{phase: phaseReserved, provisionalVersion: fingerprint.Version, provisionalFingerprint: fingerprint.Digest[:]}},
		{name: "completed without fingerprint", row: storedRow{phase: phaseCompleted, committedAt: testCommittedAt()}},
		{name: "completed with invalid retained result", row: storedRow{phase: phaseCompleted, fingerprintVersion: fingerprint.Version, fingerprint: fingerprint.Digest[:], result: []byte("not a result"), committedAt: testCommittedAt()}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, _, err := store.classifyRow(t.Context(), contract, attempt, resolver, tc.row)
			if !errors.Is(err, ErrIntegrityConflict) {
				t.Fatalf("classifyRow() error = %v, want ErrIntegrityConflict", err)
			}
		})
	}
}

func TestLockedReservationClassificationPreservesClosedDecisions(t *testing.T) {
	t.Parallel()

	contract, attempt, resolver := testIdempotencyInputs(t)
	fingerprint := attempt.Fingerprint
	result, err := httpidempotency.EncodeResult(contract, httpidempotency.Result{
		Status: http.StatusCreated, MediaType: "application/json", Codec: "create/v1", Payload: []byte(`{"id":"widget-1"}`),
	})
	if err != nil {
		t.Fatalf("EncodeResult() error = %v", err)
	}

	store := Store{}
	for _, tc := range []struct {
		name string
		row  storedRow
		want httpidempotency.Outcome
	}{
		{name: "epoch unavailable", row: storedRow{phase: phaseCompleted}, want: httpidempotency.OutcomeUnavailable},
		{name: "malformed completed row", row: storedRow{phase: phaseCompleted, committedAt: testCommittedAt()}, want: httpidempotency.OutcomeIntegrityConflict},
		{name: "resolver failure is unavailable", row: storedRow{phase: phaseCompleted, committedAt: testCommittedAt(), fingerprintVersion: "gone", fingerprint: fingerprint.Digest[:], result: result}, want: httpidempotency.OutcomeUnavailable},
		{name: "different completed request mismatches", row: storedRow{phase: phaseCompleted, committedAt: testCommittedAt(), fingerprintVersion: fingerprint.Version, fingerprint: append([]byte(nil), fingerprint.Digest[:]...), result: result}, want: httpidempotency.OutcomeMismatch},
		{name: "valid completed row replays", row: storedRow{phase: phaseCompleted, committedAt: testCommittedAt(), fingerprintVersion: fingerprint.Version, fingerprint: fingerprint.Digest[:], result: result}, want: httpidempotency.OutcomeReplay},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.name == "different completed request mismatches" {
				tc.row.fingerprint[0] ^= 1
			}
			_, decision, err := store.classifyLockedCompleted(contract, resolver, tc.row)
			if err != nil {
				t.Fatalf("classifyLockedCompleted() error = %v", err)
			}
			if decision.Outcome != tc.want {
				t.Fatalf("classifyLockedCompleted() outcome = %q, want %q", decision.Outcome, tc.want)
			}
		})
	}

	if got := store.classifyStaleReservation(resolver, storedRow{provisionalVersion: fingerprint.Version, provisionalFingerprint: fingerprint.Digest[:]}); got.Outcome != httpidempotency.OutcomeInProgress {
		t.Fatalf("classifyStaleReservation() outcome = %q, want in progress", got.Outcome)
	}
	if got := store.classifyStaleReservation(resolver, storedRow{}); got.Outcome != httpidempotency.OutcomeIntegrityConflict {
		t.Fatalf("classifyStaleReservation() outcome = %q, want integrity conflict", got.Outcome)
	}
	if !isLockUnavailable(&pgconn.PgError{Code: pgerrcode.LockNotAvailable}) || isLockUnavailable(errors.New("other")) {
		t.Fatal("isLockUnavailable() did not classify PostgreSQL lock availability exactly")
	}
}

func TestStoredRowsCopyDatabaseValues(t *testing.T) {
	t.Parallel()

	version := "v1"
	committedAt := time.Date(2026, time.August, 13, 12, 0, 0, 0, time.UTC)
	read := storedRowFromRead(sqlcgen.ReadHTTPIdempotencyRow{
		RowExists: true, WriterPrimary: true, Generation: new(int64(3)), Phase: new(phaseCompleted),
		FingerprintVersion: &version, Fingerprint: make([]byte, 32), Result: []byte("result"), CommittedAt: pgtype.Timestamptz{Time: committedAt, Valid: true},
	})
	if !read.exists || !read.writer || read.generation != 3 || read.phase != phaseCompleted || read.committedAt == nil || !read.committedAt.Equal(committedAt) {
		t.Fatalf("storedRowFromRead() = %#v, want complete database row", read)
	}
	locked := storedRowFromLocked(sqlcgen.LockHTTPIdempotencyReservationRow{Generation: 3, Phase: phaseReserved, ProvisionalFingerprintVersion: &version, ProvisionalFingerprint: make([]byte, 32)})
	if !locked.exists || !locked.writer || locked.phase != phaseReserved || locked.provisionalVersion != version {
		t.Fatalf("storedRowFromLocked() = %#v, want locked reservation row", locked)
	}
}

func TestStoreInputValidationAndClassificationBudget(t *testing.T) {
	t.Parallel()

	contract, attempt, resolver := testIdempotencyInputs(t)
	if err := validateAttempt(contract, attempt, resolver); err != nil {
		t.Fatalf("validateAttempt() error = %v", err)
	}
	if err := validateAttempt(contract, attempt, nil); !errors.Is(err, ErrConfig) {
		t.Fatalf("validateAttempt(nil resolver) error = %v, want ErrConfig", err)
	}
	invalidScope := attempt
	invalidScope.Scope.Authority = ""
	if err := validateAttempt(contract, invalidScope, resolver); !errors.Is(err, ErrConfig) {
		t.Fatalf("validateAttempt(invalid scope) error = %v, want ErrConfig", err)
	}

	fingerprint := attempt.Fingerprint
	if !sameFingerprint(fingerprint.Version, fingerprint.Digest[:], fingerprint) || sameFingerprint("v2", fingerprint.Digest[:], fingerprint) || sameFingerprint(fingerprint.Version, fingerprint.Digest[:31], fingerprint) {
		t.Fatal("sameFingerprint() did not enforce exact retained fingerprint identity")
	}
	if _, err := resolveFingerprint(resolver, fingerprint.Version); err != nil {
		t.Fatalf("resolveFingerprint() error = %v", err)
	}
	if _, err := resolveFingerprint(resolver, "v2"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("resolveFingerprint(unknown) error = %v, want ErrUnavailable", err)
	}

	for _, tc := range []struct {
		name  string
		value httpidempotency.Reservation
		valid bool
	}{
		{name: "ordinary", value: httpidempotency.Reservation{Attempt: attempt, Generation: 1, Recovery: httpidempotency.ReservationRecoveryNone}, valid: true},
		{name: "recovery due", value: httpidempotency.Reservation{Attempt: attempt, Generation: 1, Recovery: httpidempotency.ReservationRecoveryDue}, valid: true},
		{name: "reconciled", value: httpidempotency.Reservation{Attempt: attempt, Generation: 1, Recovery: httpidempotency.ReservationRecoveryReconciled}, valid: true},
		{name: "missing generation", value: httpidempotency.Reservation{Attempt: attempt, Recovery: httpidempotency.ReservationRecoveryNone}},
		{name: "unknown recovery", value: httpidempotency.Reservation{Attempt: attempt, Generation: 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := validReservation(tc.value); got != tc.valid {
				t.Fatalf("validReservation() = %t, want %t", got, tc.valid)
			}
		})
	}

	parent, cancelParent := context.WithCancel(t.Context())
	t.Cleanup(cancelParent)
	budget, cancelBudget, ownBudget := classificationContext(parent, time.Millisecond)
	t.Cleanup(cancelBudget)
	if !ownBudget {
		t.Fatal("classificationContext() did not create the configured classification budget")
	}
	cancelBudget()
	if err := classificationError(parent, budget, ownBudget, errors.New("database wait")); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("classificationError() = %v, want ErrUnavailable after private budget exhaustion", err)
	}
	if decision, handled := decisionForClassificationError(ErrEpochLost); !handled || decision.Outcome != httpidempotency.OutcomeUnavailable {
		t.Fatalf("decisionForClassificationError(ErrEpochLost) = (%#v, %t), want unavailable", decision, handled)
	}
	if _, handled := decisionForClassificationError(errors.New("other")); handled {
		t.Fatal("decisionForClassificationError() handled an unclassified error")
	}
}

func TestStoreRejectsUnboundDatabaseOperationsBeforeAnyQuery(t *testing.T) {
	contract, attempt, resolver := testIdempotencyInputs(t)
	reservation := httpidempotency.Reservation{Attempt: attempt, Generation: 1, Recovery: httpidempotency.ReservationRecoveryNone}
	result := httpidempotency.Result{Status: http.StatusCreated, MediaType: "application/json", Codec: "create/v1", Payload: []byte(`{"id":"widget-1"}`)}
	var store Store
	for _, test := range []struct {
		name string
		call func() error
	}{
		{name: "reserve", call: func() error { _, _, err := store.Reserve(t.Context(), contract, attempt, resolver); return err }},
		{name: "reconcile", call: func() error { _, _, err := store.Reconcile(t.Context(), contract, attempt, resolver); return err }},
		{name: "acquire", call: func() error {
			_, _, err := store.Acquire(t.Context(), nil, contract, reservation, resolver)
			return err
		}},
		{name: "complete", call: func() error { return store.Complete(t.Context(), nil, contract, reservation, result) }},
		{name: "release", call: func() error { return store.Release(t.Context(), reservation) }},
		{name: "materialize epoch", call: func() error { _, err := store.MaterializeEpoch(t.Context(), attempt); return err }},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := test.call(); !errors.Is(err, ErrConfig) {
				t.Fatalf("unbound Store error = %v, want ErrConfig", err)
			}
		})
	}
	if store.Name() != maintenanceProbeName {
		t.Fatalf("Name() = %q, want %q", store.Name(), maintenanceProbeName)
	}
}

func testIdempotencyInputs(t *testing.T) (httpidempotency.Contract, httpidempotency.Attempt, FingerprintResolver) {
	t.Helper()
	contract := httpidempotency.Contract{
		OperationID: "CreateWidget", APIVersion: "v1", KeyMaxBytes: 64,
		FingerprintVersions: []string{"v1"}, ResultCodecs: []string{"create/v1"}, ReplayStatuses: []int{http.StatusCreated},
		ResultMaxBytes: 1024, ReplayTTL: time.Hour, DuplicateRisk: httpidempotency.DuplicateRiskPolicy{Duration: 2 * time.Hour},
		InProgressWait: time.Second, RetryAfter: time.Second, ExternalEffect: httpidempotency.ExternalEffectNone,
	}
	fingerprint, err := httpidempotency.NewFingerprint("v1", []byte(`{"name":"widget"}`))
	if err != nil {
		t.Fatalf("NewFingerprint() error = %v", err)
	}
	attempt, err := httpidempotency.NewAttempt(httpidempotency.Scope{Authority: "tenant-1", OperationID: contract.OperationID, APIVersion: contract.APIVersion}, "key-1", fingerprint)
	if err != nil {
		t.Fatalf("NewAttempt() error = %v", err)
	}
	return contract, attempt, func(version string) (httpidempotency.Fingerprint, error) {
		if version != fingerprint.Version {
			return httpidempotency.Fingerprint{}, errors.New("unknown fingerprint version")
		}
		return fingerprint, nil
	}
}

func testCommittedAt() *time.Time {
	value := time.Date(2026, time.August, 13, 12, 0, 0, 0, time.UTC)
	return &value
}
