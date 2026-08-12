package postgresidempotency

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/example/go-service-template-rest/internal/httpidempotency"
	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5"
)

type publicationGroup struct {
	mu      sync.Mutex
	waiting map[[32]byte]chan struct{}
}

// run elects one local publisher. The map carries only a completion signal;
// each follower re-reads PostgreSQL after the signal.
func (g *publicationGroup) run(ctx context.Context, identity [32]byte, publish func() error) (bool, error) {
	g.mu.Lock()
	if done, found := g.waiting[identity]; found {
		g.mu.Unlock()
		select {
		case <-done:
			return false, nil
		case <-ctx.Done():
			return false, ctx.Err()
		}
	}
	done := make(chan struct{})
	g.waiting[identity] = done
	g.mu.Unlock()

	defer func() {
		g.mu.Lock()
		delete(g.waiting, identity)
		close(done)
		g.mu.Unlock()
	}()
	return true, publish()
}

// Reserve classifies a request on the writer and publishes a reservation when
// the identity is absent. It starts exactly one in-progress sub-budget.
func (s *Store) Reserve(
	parent context.Context,
	contract httpidempotency.Contract,
	attempt httpidempotency.Attempt,
	resolve FingerprintResolver,
) (httpidempotency.Reservation, httpidempotency.Decision, error) {
	if !s.valid() {
		return httpidempotency.Reservation{}, httpidempotency.Decision{}, fmt.Errorf("%w: store is required", ErrConfig)
	}
	if err := validateAttempt(contract, attempt, resolve); err != nil {
		return httpidempotency.Reservation{}, httpidempotency.Decision{}, err
	}

	ctx, cancel, ownBudget := classificationContext(parent, contract.InProgressWait)
	defer cancel()
	reservation, decision, err := s.reserve(ctx, contract, attempt, resolve)
	if err == nil {
		return reservation, decision, nil
	}
	err = classificationError(parent, ctx, ownBudget, err)
	if decision, handled := decisionForClassificationError(err); handled {
		return httpidempotency.Reservation{}, decision, nil
	}
	return httpidempotency.Reservation{}, httpidempotency.Decision{}, err
}

func (s *Store) reserve(
	ctx context.Context,
	contract httpidempotency.Contract,
	attempt httpidempotency.Attempt,
	resolve FingerprintResolver,
) (httpidempotency.Reservation, httpidempotency.Decision, error) {
	for publication := 0; publication < 2; publication++ {
		reservation, decision, absent, err := s.classify(ctx, contract, attempt, resolve)
		if err != nil || !absent {
			return reservation, decision, err
		}

		var published httpidempotency.Reservation
		leader, err := s.flights.run(ctx, attempt.Identity, func() error {
			var publishErr error
			published, publishErr = s.publish(ctx, attempt)
			return publishErr
		})
		if err != nil {
			return httpidempotency.Reservation{}, httpidempotency.Decision{}, err
		}
		if leader && validReservation(published) {
			return published, httpidempotency.Decision{Outcome: httpidempotency.OutcomeExecute}, nil
		}
	}
	return httpidempotency.Reservation{}, httpidempotency.Decision{}, unavailable(ctx, "publish reservation")
}

func (s *Store) classify(
	ctx context.Context,
	contract httpidempotency.Contract,
	attempt httpidempotency.Attempt,
	resolve FingerprintResolver,
) (httpidempotency.Reservation, httpidempotency.Decision, bool, error) {
	row, err := s.read(ctx, attempt.Identity)
	if err != nil {
		return httpidempotency.Reservation{}, httpidempotency.Decision{}, false, err
	}
	if !row.writer {
		return httpidempotency.Reservation{}, httpidempotency.Decision{}, false, ErrUnavailable
	}
	if !row.exists {
		return httpidempotency.Reservation{}, httpidempotency.Decision{}, true, nil
	}
	reservation, decision, err := s.classifyRow(ctx, contract, attempt, resolve, row)
	return reservation, decision, false, err
}

func (s *Store) classifyRow(
	ctx context.Context,
	contract httpidempotency.Contract,
	attempt httpidempotency.Attempt,
	resolve FingerprintResolver,
	row storedRow,
) (httpidempotency.Reservation, httpidempotency.Decision, error) {
	switch row.phase {
	case "reserved":
		if row.generation <= 0 || row.provisionalVersion == "" || len(row.provisionalFingerprint) != 32 {
			return httpidempotency.Reservation{}, httpidempotency.Decision{}, ErrIntegrityConflict
		}
		if row.recoveryDue {
			return httpidempotency.Reservation{
				Attempt:    attempt,
				Generation: row.generation,
				Recovery:   httpidempotency.ReservationRecoveryDue,
			}, httpidempotency.Decision{Outcome: httpidempotency.OutcomeExecute}, nil
		}
		fingerprint, err := resolveFingerprint(resolve, row.provisionalVersion)
		if err != nil {
			return httpidempotency.Reservation{}, httpidempotency.Decision{}, err
		}
		if !sameFingerprint(row.provisionalVersion, row.provisionalFingerprint, fingerprint) {
			return httpidempotency.Reservation{}, httpidempotency.Decision{Outcome: httpidempotency.OutcomeMismatch}, nil
		}
		return httpidempotency.Reservation{}, httpidempotency.Decision{Outcome: httpidempotency.OutcomeInProgress}, nil
	case "completed":
		return s.classifyCompleted(ctx, contract, attempt, resolve, row)
	default:
		return httpidempotency.Reservation{}, httpidempotency.Decision{}, ErrIntegrityConflict
	}
}

func (s *Store) classifyCompleted(
	ctx context.Context,
	contract httpidempotency.Contract,
	attempt httpidempotency.Attempt,
	resolve FingerprintResolver,
	row storedRow,
) (httpidempotency.Reservation, httpidempotency.Decision, error) {
	if row.committedAt == nil {
		if _, err := s.MaterializeEpoch(ctx, attempt); err != nil {
			return httpidempotency.Reservation{}, httpidempotency.Decision{}, err
		}
		var err error
		row, err = s.read(ctx, attempt.Identity)
		if err != nil {
			return httpidempotency.Reservation{}, httpidempotency.Decision{}, err
		}
		if !row.writer {
			return httpidempotency.Reservation{}, httpidempotency.Decision{}, ErrUnavailable
		}
		if !row.exists || row.phase != "completed" || row.committedAt == nil {
			return httpidempotency.Reservation{}, httpidempotency.Decision{}, ErrEpochLost
		}
	}
	if row.fingerprintVersion == "" || len(row.fingerprint) != 32 || len(row.result) == 0 {
		return httpidempotency.Reservation{}, httpidempotency.Decision{}, ErrIntegrityConflict
	}
	fingerprint, err := resolveFingerprint(resolve, row.fingerprintVersion)
	if err != nil {
		return httpidempotency.Reservation{}, httpidempotency.Decision{}, err
	}
	if !sameFingerprint(row.fingerprintVersion, row.fingerprint, fingerprint) {
		return httpidempotency.Reservation{}, httpidempotency.Decision{Outcome: httpidempotency.OutcomeMismatch}, nil
	}
	result, err := httpidempotency.DecodeResult(contract, row.result)
	if err != nil {
		return httpidempotency.Reservation{}, httpidempotency.Decision{}, ErrIntegrityConflict
	}
	return httpidempotency.Reservation{}, httpidempotency.Decision{
		Outcome: httpidempotency.OutcomeReplay,
		Result:  &result,
	}, nil
}

func (s *Store) publish(ctx context.Context, attempt httpidempotency.Attempt) (httpidempotency.Reservation, error) {
	row, err := s.read(ctx, attempt.Identity)
	if err != nil {
		return httpidempotency.Reservation{}, err
	}
	if !row.writer {
		return httpidempotency.Reservation{}, ErrUnavailable
	}
	if row.exists {
		return httpidempotency.Reservation{}, nil
	}

	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return httpidempotency.Reservation{}, unavailable(ctx, "reserve connection")
	}
	defer conn.Release()

	tx, err := conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return httpidempotency.Reservation{}, unavailable(ctx, "begin reservation")
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := setTransactionTimeouts(ctx, tx); err != nil {
		return httpidempotency.Reservation{}, err
	}

	generation, err := sqlcgen.New(tx).InsertHTTPIdempotencyReservation(ctx, sqlcgen.InsertHTTPIdempotencyReservationParams{
		IdentityToken:      identityBytes(attempt),
		FingerprintVersion: attempt.Fingerprint.Version,
		Fingerprint:        fingerprintBytes(attempt.Fingerprint),
		RecoveryMicros:     durationMicros(s.recoveryDelay),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return httpidempotency.Reservation{}, nil
	}
	if err != nil {
		return httpidempotency.Reservation{}, unavailable(ctx, "publish reservation")
	}
	if err := tx.Commit(ctx); err == nil {
		return httpidempotency.Reservation{
			Attempt:    attempt,
			Generation: generation,
			Recovery:   httpidempotency.ReservationRecoveryNone,
		}, nil
	}

	row, err = s.read(ctx, attempt.Identity)
	if err != nil {
		return httpidempotency.Reservation{}, err
	}
	if row.writer && row.exists && row.phase == "reserved" && row.generation == generation &&
		sameFingerprint(row.provisionalVersion, row.provisionalFingerprint, attempt.Fingerprint) {
		return httpidempotency.Reservation{
			Attempt:    attempt,
			Generation: generation,
			Recovery:   httpidempotency.ReservationRecoveryNone,
		}, nil
	}
	return httpidempotency.Reservation{}, unavailable(ctx, "reconcile reservation commit")
}

func (s *Store) read(ctx context.Context, identity [32]byte) (storedRow, error) {
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return storedRow{}, unavailable(ctx, "read connection")
	}
	defer conn.Release()

	row, err := sqlcgen.New(conn).ReadHTTPIdempotency(ctx, identity[:])
	if err != nil {
		return storedRow{}, unavailable(ctx, "read reservation")
	}
	return storedRowFromRead(row), nil
}

type storedRow struct {
	exists                 bool
	writer                 bool
	generation             int64
	phase                  string
	provisionalVersion     string
	provisionalFingerprint []byte
	fingerprintVersion     string
	fingerprint            []byte
	result                 []byte
	recoveryDue            bool
	committedAt            *time.Time
}

func storedRowFromRead(row sqlcgen.ReadHTTPIdempotencyRow) storedRow {
	stored := storedRow{exists: row.RowExists, writer: row.WriterPrimary, recoveryDue: row.RecoveryDue}
	if !stored.exists {
		return stored
	}
	if row.Generation != nil {
		stored.generation = *row.Generation
	}
	if row.Phase != nil {
		stored.phase = *row.Phase
	}
	if row.ProvisionalFingerprintVersion != nil {
		stored.provisionalVersion = *row.ProvisionalFingerprintVersion
	}
	if row.FingerprintVersion != nil {
		stored.fingerprintVersion = *row.FingerprintVersion
	}
	stored.provisionalFingerprint = row.ProvisionalFingerprint
	stored.fingerprint = row.Fingerprint
	stored.result = row.Result
	if row.CommittedAt.Valid {
		committedAt := row.CommittedAt.Time
		stored.committedAt = &committedAt
	}
	return stored
}

func setTransactionTimeouts(ctx context.Context, tx pgx.Tx) error {
	deadline, ok := ctx.Deadline()
	if !ok {
		return nil
	}
	remaining := time.Until(deadline)
	if remaining <= time.Millisecond {
		return context.DeadlineExceeded
	}
	timeout := postgres.RuntimeParamMilliseconds(remaining - time.Millisecond)
	if _, err := tx.Exec(ctx, "SELECT set_config('lock_timeout', $1, true), set_config('statement_timeout', $1, true)", timeout); err != nil {
		return unavailable(ctx, "set reservation timeout")
	}
	return nil
}

func durationMicros(duration time.Duration) int64 {
	micros := duration / time.Microsecond
	if duration%time.Microsecond != 0 {
		micros++
	}
	return int64(micros)
}
