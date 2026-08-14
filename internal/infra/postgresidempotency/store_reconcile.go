package postgresidempotency

import (
	"context"
	"errors"
	"fmt"

	"github.com/example/go-service-template-rest/internal/httpidempotency"
	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5"
)

// Reconcile resolves a caller transaction whose commit result was lost. It
// never assumes rollback from a client-side error alone.
func (s *Store) Reconcile(
	parent context.Context,
	contract httpidempotency.Contract,
	attempt httpidempotency.Attempt,
	resolve FingerprintResolver,
) (httpidempotency.Reservation, httpidempotency.Decision, error) {
	if parent == nil {
		return httpidempotency.Reservation{}, httpidempotency.Decision{}, fmt.Errorf("%w: context is required", ErrConfig)
	}
	if !s.valid() {
		return httpidempotency.Reservation{}, httpidempotency.Decision{}, fmt.Errorf("%w: store is required", ErrConfig)
	}
	if err := validateAttempt(contract, attempt, resolve); err != nil {
		return httpidempotency.Reservation{}, httpidempotency.Decision{}, err
	}
	ctx, cancel, ownBudget := classificationContext(parent, contract.InProgressWait)
	if ctx == nil {
		return httpidempotency.Reservation{}, httpidempotency.Decision{}, fmt.Errorf("%w: classification context is required", ErrConfig)
	}
	defer cancel()
	reservation, decision, err := s.reconcile(ctx, contract, attempt, resolve)
	if err == nil {
		return reservation, decision, nil
	}
	err = classificationError(parent, ctx, ownBudget, err)
	if errors.Is(err, ErrEpochLost) {
		return httpidempotency.Reservation{}, httpidempotency.Decision{Outcome: httpidempotency.OutcomeUnavailable}, nil
	}
	if errors.Is(err, ErrUnavailable) {
		return httpidempotency.Reservation{}, httpidempotency.Decision{Outcome: httpidempotency.OutcomeUnknown}, nil
	}
	if errors.Is(err, ErrIntegrityConflict) {
		return httpidempotency.Reservation{}, httpidempotency.Decision{Outcome: httpidempotency.OutcomeIntegrityConflict}, nil
	}
	return httpidempotency.Reservation{}, httpidempotency.Decision{}, err
}

func (s *Store) reconcile(
	ctx context.Context,
	contract httpidempotency.Contract,
	attempt httpidempotency.Attempt,
	resolve FingerprintResolver,
) (httpidempotency.Reservation, httpidempotency.Decision, error) {
	row, err := s.read(ctx, attempt.Identity)
	if err != nil {
		return httpidempotency.Reservation{}, httpidempotency.Decision{}, err
	}
	if !row.writer {
		return httpidempotency.Reservation{}, httpidempotency.Decision{Outcome: httpidempotency.OutcomeUnknown}, nil
	}
	if !row.exists {
		return s.reserve(ctx, contract, attempt, resolve)
	}
	switch row.phase {
	case phaseCompleted:
		reservation, decision, err := s.classifyCompleted(ctx, contract, attempt, resolve, row)
		if err == nil && decision.Outcome == httpidempotency.OutcomeMismatch {
			return httpidempotency.Reservation{}, httpidempotency.Decision{Outcome: httpidempotency.OutcomeIntegrityConflict}, nil
		}
		return reservation, decision, err
	case phaseReserved:
		if row.generation <= 0 || row.provisionalVersion == "" || len(row.provisionalFingerprint) != 32 {
			return httpidempotency.Reservation{}, httpidempotency.Decision{}, ErrIntegrityConflict
		}
		fingerprint, err := resolveFingerprint(resolve, row.provisionalVersion)
		if err != nil {
			//nolint:nilerr // A resolver failure is an intentional unknown decision after a lost commit result.
			return httpidempotency.Reservation{}, httpidempotency.Decision{Outcome: httpidempotency.OutcomeUnknown}, nil
		}
		if !sameFingerprint(row.provisionalVersion, row.provisionalFingerprint, fingerprint) {
			return httpidempotency.Reservation{}, httpidempotency.Decision{Outcome: httpidempotency.OutcomeIntegrityConflict}, nil
		}
		if err := s.probeReservationLock(ctx, attempt, row.generation); err != nil {
			if isLockUnavailable(err) {
				return httpidempotency.Reservation{}, httpidempotency.Decision{Outcome: httpidempotency.OutcomeUnknown}, nil
			}
			return httpidempotency.Reservation{}, httpidempotency.Decision{}, err
		}
		return httpidempotency.Reservation{
			Attempt:    attempt,
			Generation: row.generation,
			Recovery:   httpidempotency.ReservationRecoveryReconciled,
		}, httpidempotency.Decision{Outcome: httpidempotency.OutcomeExecute}, nil
	default:
		return httpidempotency.Reservation{}, httpidempotency.Decision{}, ErrIntegrityConflict
	}
}

func (s *Store) probeReservationLock(ctx context.Context, attempt httpidempotency.Attempt, generation int64) error {
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return unavailable(ctx, "reconcile connection")
	}
	defer conn.Release()
	tx, err := conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return unavailable(ctx, "begin reconciliation")
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := setTransactionTimeouts(ctx, tx); err != nil {
		return err
	}
	queries := sqlcgen.New(tx)
	writer, err := queries.CheckHTTPIdempotencyWriter(ctx)
	if err != nil {
		return unavailable(ctx, "check reconciliation writer")
	}
	if !writer {
		return ErrUnavailable
	}
	row, err := queries.LockHTTPIdempotencyReservation(ctx, identityBytes(attempt))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrIntegrityConflict
		}
		if isLockUnavailable(err) {
			return err //nolint:wrapcheck // The caller classifies the exact lock error.
		}
		return fmt.Errorf("lock reconciliation: %w", unavailable(ctx, "lock reconciliation"))
	}
	if row.Phase != phaseReserved || row.Generation != generation {
		return ErrIntegrityConflict
	}
	return nil
}
