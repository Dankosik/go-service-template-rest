package postgresidempotency

import (
	"context"
	"errors"
	"fmt"

	"github.com/example/go-service-template-rest/internal/httpidempotency"
	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Acquire locks and verifies the exact reservation inside the endpoint's
// transaction. It never begins, commits, or retries that feature transaction.
//
//nolint:cyclop // The lock/error classifications share one caller-owned transaction boundary.
func (s *Store) Acquire(
	ctx context.Context,
	tx pgx.Tx,
	contract httpidempotency.Contract,
	reservation httpidempotency.Reservation,
	resolve FingerprintResolver,
) (httpidempotency.Reservation, httpidempotency.Decision, error) {
	if !s.valid() || tx == nil || !validReservation(reservation) {
		return httpidempotency.Reservation{}, httpidempotency.Decision{}, fmt.Errorf("%w: store, transaction, and reservation are required", ErrConfig)
	}
	if err := validateAttempt(contract, reservation.Attempt, resolve); err != nil {
		return httpidempotency.Reservation{}, httpidempotency.Decision{}, err
	}
	if err := setTransactionTimeouts(ctx, tx); err != nil {
		return httpidempotency.Reservation{}, httpidempotency.Decision{}, err
	}

	queries := sqlcgen.New(tx)
	writer, err := queries.CheckHTTPIdempotencyWriter(ctx)
	if err != nil {
		return httpidempotency.Reservation{}, httpidempotency.Decision{}, unavailable(ctx, "check transaction writer")
	}
	if !writer {
		return httpidempotency.Reservation{}, httpidempotency.Decision{Outcome: httpidempotency.OutcomeUnavailable}, nil
	}

	// NOWAIT raises 55P03, which otherwise aborts the caller's transaction.
	savepoint, err := tx.Begin(ctx)
	if err != nil {
		return httpidempotency.Reservation{}, httpidempotency.Decision{}, unavailable(ctx, "begin reservation lock savepoint")
	}
	locked, err := sqlcgen.New(savepoint).LockHTTPIdempotencyReservation(ctx, identityBytes(reservation.Attempt))
	if err != nil {
		if rollbackErr := savepoint.Rollback(ctx); rollbackErr != nil {
			return httpidempotency.Reservation{}, httpidempotency.Decision{}, unavailable(ctx, "rollback reservation lock savepoint")
		}
		if isLockUnavailable(err) {
			return httpidempotency.Reservation{}, httpidempotency.Decision{Outcome: httpidempotency.OutcomeInProgress}, nil
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return httpidempotency.Reservation{}, httpidempotency.Decision{Outcome: httpidempotency.OutcomeIntegrityConflict}, nil
		}
		return httpidempotency.Reservation{}, httpidempotency.Decision{}, unavailable(ctx, "lock reservation")
	}
	if err := savepoint.Commit(ctx); err != nil {
		return httpidempotency.Reservation{}, httpidempotency.Decision{}, unavailable(ctx, "release reservation lock savepoint")
	}
	row := storedRowFromLocked(locked)
	if row.phase == phaseCompleted {
		return s.classifyLockedCompleted(contract, resolve, row)
	}
	if row.phase != phaseReserved || row.generation <= 0 {
		return httpidempotency.Reservation{}, httpidempotency.Decision{Outcome: httpidempotency.OutcomeIntegrityConflict}, nil
	}
	if row.generation != reservation.Generation {
		return httpidempotency.Reservation{}, s.classifyStaleReservation(resolve, row), nil
	}
	if reservation.Recovery == httpidempotency.ReservationRecoveryNone {
		if !sameFingerprint(row.provisionalVersion, row.provisionalFingerprint, reservation.Attempt.Fingerprint) {
			return httpidempotency.Reservation{}, httpidempotency.Decision{Outcome: httpidempotency.OutcomeMismatch}, nil
		}
		return reservation, httpidempotency.Decision{Outcome: httpidempotency.OutcomeExecute}, nil
	}
	if reservation.Recovery == httpidempotency.ReservationRecoveryDue && !row.recoveryDue {
		return httpidempotency.Reservation{}, httpidempotency.Decision{Outcome: httpidempotency.OutcomeInProgress}, nil
	}

	generation, err := queries.AdvanceHTTPIdempotencyReservation(ctx, sqlcgen.AdvanceHTTPIdempotencyReservationParams{
		FingerprintVersion: reservation.Attempt.Fingerprint.Version,
		Fingerprint:        fingerprintBytes(reservation.Attempt.Fingerprint),
		RecoveryMicros:     durationMicros(s.recoveryDelay),
		IdentityToken:      identityBytes(reservation.Attempt),
		Generation:         reservation.Generation,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return httpidempotency.Reservation{}, httpidempotency.Decision{Outcome: httpidempotency.OutcomeInProgress}, nil
	}
	if err != nil {
		return httpidempotency.Reservation{}, httpidempotency.Decision{}, unavailable(ctx, "advance reservation")
	}
	return httpidempotency.Reservation{
		Attempt:    reservation.Attempt,
		Generation: generation,
		Recovery:   httpidempotency.ReservationRecoveryNone,
	}, httpidempotency.Decision{Outcome: httpidempotency.OutcomeExecute}, nil
}

func (s *Store) classifyLockedCompleted(
	contract httpidempotency.Contract,
	resolve FingerprintResolver,
	row storedRow,
) (httpidempotency.Reservation, httpidempotency.Decision, error) {
	if row.committedAt == nil {
		return httpidempotency.Reservation{}, httpidempotency.Decision{Outcome: httpidempotency.OutcomeUnavailable}, nil
	}
	if row.fingerprintVersion == "" || len(row.fingerprint) != 32 || len(row.result) == 0 {
		return httpidempotency.Reservation{}, httpidempotency.Decision{Outcome: httpidempotency.OutcomeIntegrityConflict}, nil
	}
	fingerprint, err := resolveFingerprint(resolve, row.fingerprintVersion)
	if err != nil {
		//nolint:nilerr // A resolver failure is an intentional unavailable decision, not a transaction error.
		return httpidempotency.Reservation{}, httpidempotency.Decision{Outcome: httpidempotency.OutcomeUnavailable}, nil
	}
	if !sameFingerprint(row.fingerprintVersion, row.fingerprint, fingerprint) {
		return httpidempotency.Reservation{}, httpidempotency.Decision{Outcome: httpidempotency.OutcomeMismatch}, nil
	}
	result, err := httpidempotency.DecodeResult(contract, row.result)
	if err != nil {
		//nolint:nilerr // An invalid stored result is an intentional integrity decision.
		return httpidempotency.Reservation{}, httpidempotency.Decision{Outcome: httpidempotency.OutcomeIntegrityConflict}, nil
	}
	return httpidempotency.Reservation{}, httpidempotency.Decision{Outcome: httpidempotency.OutcomeReplay, Result: &result}, nil
}

func (s *Store) classifyStaleReservation(
	resolve FingerprintResolver,
	row storedRow,
) httpidempotency.Decision {
	if row.provisionalVersion == "" || len(row.provisionalFingerprint) != 32 {
		return httpidempotency.Decision{Outcome: httpidempotency.OutcomeIntegrityConflict}
	}
	fingerprint, err := resolveFingerprint(resolve, row.provisionalVersion)
	if err != nil || !sameFingerprint(row.provisionalVersion, row.provisionalFingerprint, fingerprint) {
		return httpidempotency.Decision{Outcome: httpidempotency.OutcomeMismatch}
	}
	return httpidempotency.Decision{Outcome: httpidempotency.OutcomeInProgress}
}

func storedRowFromLocked(row sqlcgen.LockHTTPIdempotencyReservationRow) storedRow {
	stored := storedRow{
		exists:                 true,
		writer:                 true,
		generation:             row.Generation,
		phase:                  row.Phase,
		provisionalFingerprint: row.ProvisionalFingerprint,
		fingerprint:            row.Fingerprint,
		result:                 row.Result,
		recoveryDue:            row.RecoveryDue,
	}
	if row.ProvisionalFingerprintVersion != nil {
		stored.provisionalVersion = *row.ProvisionalFingerprintVersion
	}
	if row.FingerprintVersion != nil {
		stored.fingerprintVersion = *row.FingerprintVersion
	}
	if row.CommittedAt.Valid {
		committedAt := row.CommittedAt.Time
		stored.committedAt = &committedAt
	}
	return stored
}

func isLockUnavailable(err error) bool {
	postgresError, ok := errors.AsType[*pgconn.PgError](err)
	return ok && postgresError.Code == pgerrcode.LockNotAvailable
}
