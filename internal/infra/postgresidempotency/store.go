// Package postgresidempotency executes one authorized HTTP mutation and its
// replay evidence in the same PostgreSQL transaction.
package postgresidempotency

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/example/go-service-template-rest/internal/httpidempotency"
	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	cleanupBatchSize    = 500
	maintenanceInterval = time.Minute
)

var ErrConfig = errors.New("postgres idempotency config")

type Store struct {
	pool      *pgxpool.Pool
	retention time.Duration
	inTx      func(context.Context, *pgxpool.Pool, pgx.TxOptions, func(pgx.Tx) error) error
}

// NewExecutor binds a transaction-scoped feature repository before Work runs.
func NewExecutor[Repository, Response any](
	store *Store,
	bind func(pgx.Tx) Repository,
	codec httpidempotency.Codec[Response],
) (httpidempotency.Executor[Repository, Response], error) {
	if store == nil || bind == nil || !codec.Valid() {
		return nil, fmt.Errorf("%w: store, repository binding, and response codec are required", ErrConfig)
	}
	return func(
		ctx context.Context,
		request httpidempotency.Request,
		work httpidempotency.Work[Repository, Response],
	) (Response, bool, error) {
		if work == nil {
			var zero Response
			return zero, false, fmt.Errorf("%w: executor and work are required", ErrConfig)
		}
		var zero Response
		result, replayed, err := store.execute(ctx, request, func(ctx context.Context, tx pgx.Tx) ([]byte, error) {
			response, err := work(ctx, bind(tx))
			if err != nil {
				return nil, err
			}
			return codec.Encode(response)
		})
		if err != nil {
			return zero, false, fmt.Errorf("execute idempotent operation: %w", err)
		}
		decodedResponse, err := codec.Decode(result)
		if err != nil {
			if replayed {
				return decodedResponse, true, fmt.Errorf("%w: decode stored response", httpidempotency.ErrIntegrity)
			}
			return decodedResponse, replayed, fmt.Errorf("decode idempotency response: %w", err)
		}
		return decodedResponse, replayed, nil
	}, nil
}

func NewStore(pool *pgxpool.Pool, retention time.Duration) (*Store, error) {
	if pool == nil {
		return nil, fmt.Errorf("%w: postgres pool is required", ErrConfig)
	}
	if retention <= 0 {
		return nil, fmt.Errorf("%w: retention must be positive", ErrConfig)
	}
	return &Store{
		pool: pool, retention: retention, inTx: postgres.InTx,
	}, nil
}

// execute serializes one scoped key at PostgreSQL, runs work only for the
// winner, and commits its result with the business effect. The boolean reports
// whether the returned result was replayed.
func (s *Store) execute(
	ctx context.Context,
	request httpidempotency.Request,
	work func(context.Context, pgx.Tx) ([]byte, error),
) ([]byte, bool, error) {
	if s == nil || s.pool == nil || s.inTx == nil || !request.Valid() || work == nil {
		return nil, false, fmt.Errorf("%w: store, request, and work are required", ErrConfig)
	}

	var result []byte
	var replayed bool
	err := s.inTx(ctx, s.pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		var transactionErr error
		result, replayed, transactionErr = s.executeTransaction(ctx, tx, request, work)
		return transactionErr
	})
	if err == nil {
		return result, replayed, nil
	}
	// A replay only read evidence committed earlier, so failing to commit its
	// read-only transaction cannot change that result.
	if replayed {
		return result, true, nil
	}
	if !errors.Is(err, postgres.ErrCommitUnknown) {
		return nil, false, fmt.Errorf("execute idempotency transaction: %w", err)
	}

	// The commit may have landed. Committed evidence read back outside the
	// transaction is the result; without it the outcome stays unknown and the
	// work is not retried.
	stored, readErr := s.read(ctx, request)
	if readErr == nil {
		return stored, true, nil
	}
	return nil, false, fmt.Errorf("%w: %w", httpidempotency.ErrOutcomeUnknown, err)
}

func (s *Store) executeTransaction(
	ctx context.Context,
	tx pgx.Tx,
	request httpidempotency.Request,
	work func(context.Context, pgx.Tx) ([]byte, error),
) ([]byte, bool, error) {
	queries := sqlcgen.New(tx)
	// A claim finds no row when a live row blocks it or the session cannot
	// write. If the blocking row is no longer live when read (it expired in
	// between), one more claim decides.
	for range 2 {
		claimed, err := s.claim(ctx, queries, request)
		if err != nil {
			return nil, false, err
		}
		if claimed {
			result, err := s.executeWork(ctx, tx, queries, request, work)
			return result, false, err
		}
		row, err := queries.ReadHTTPIdempotency(ctx, request.Identity())
		if err != nil {
			return nil, false, storageFailure(ctx, "read", err)
		}
		result, found, err := storedEvidence(request, row)
		if err != nil {
			return nil, false, err
		}
		if found {
			return result, true, nil
		}
	}
	return nil, false, httpidempotency.ErrUnavailable
}

func (s *Store) executeWork(
	ctx context.Context,
	tx pgx.Tx,
	queries *sqlcgen.Queries,
	request httpidempotency.Request,
	work func(context.Context, pgx.Tx) ([]byte, error),
) ([]byte, error) {
	encoded, err := work(ctx, tx)
	if err != nil {
		return nil, err
	}
	completed, err := queries.CompleteHTTPIdempotency(ctx, sqlcgen.CompleteHTTPIdempotencyParams{
		Result:          encoded,
		RetentionMicros: durationMicros(s.retention),
		IdentityToken:   request.Identity(),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, httpidempotency.ErrIntegrity
	}
	if err != nil {
		return nil, storageFailure(ctx, "complete", err)
	}
	if len(completed) == 0 {
		return nil, httpidempotency.ErrIntegrity
	}
	return encoded, nil
}

func storedEvidence(
	request httpidempotency.Request,
	row sqlcgen.ReadHTTPIdempotencyRow,
) ([]byte, bool, error) {
	// Only a writable primary proves current evidence; a standby or read-only
	// session may lag or be unable to claim.
	if !row.WriterPrimary {
		return nil, false, httpidempotency.ErrUnavailable
	}
	if !row.RowExists || !row.Live {
		return nil, false, nil
	}
	if row.FingerprintVersion == nil || !request.MatchesFingerprint(*row.FingerprintVersion, row.Fingerprint) {
		return nil, false, httpidempotency.ErrMismatch
	}
	if len(row.Result) == 0 {
		return nil, false, httpidempotency.ErrIntegrity
	}
	return row.Result, true, nil
}

func (s *Store) claim(ctx context.Context, queries *sqlcgen.Queries, request httpidempotency.Request) (bool, error) {
	version, fingerprint := request.Fingerprint()
	_, err := queries.ClaimHTTPIdempotency(ctx, sqlcgen.ClaimHTTPIdempotencyParams{
		IdentityToken:      request.Identity(),
		FingerprintVersion: version,
		Fingerprint:        fingerprint,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, storageFailure(ctx, "claim", err)
	}
	return true, nil
}

func (s *Store) read(ctx context.Context, request httpidempotency.Request) ([]byte, error) {
	if s == nil || s.pool == nil || !request.Valid() {
		return nil, fmt.Errorf("%w: store and request are required", ErrConfig)
	}
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return nil, storageFailure(ctx, "read connection", err)
	}
	defer conn.Release()
	row, err := sqlcgen.New(conn).ReadHTTPIdempotency(ctx, request.Identity())
	if err != nil {
		return nil, storageFailure(ctx, "read", err)
	}
	result, found, err := storedEvidence(request, row)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, httpidempotency.ErrOutcomeUnknown
	}
	return result, nil
}

// Cleanup removes expired replay rows in bounded, lock-skipping batches.
func (s *Store) Cleanup(ctx context.Context) (int64, error) {
	if s == nil || s.pool == nil {
		return 0, fmt.Errorf("%w: store is required", ErrConfig)
	}
	var total int64
	for {
		rows, err := s.cleanupBatch(ctx)
		total += rows
		if err != nil || rows < cleanupBatchSize {
			return total, err
		}
	}
}

func (s *Store) cleanupBatch(ctx context.Context) (int64, error) {
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return 0, storageFailure(ctx, "cleanup connection", err)
	}
	defer conn.Release()
	rows, err := sqlcgen.New(conn).CleanupHTTPIdempotency(ctx, cleanupBatchSize)
	if err != nil {
		return 0, storageFailure(ctx, "cleanup", err)
	}
	return rows, nil
}

// Maintain removes expired replay rows until ctx is canceled. Cleanup failures
// are degraded maintenance, not a reason to stop serving idempotent requests.
func (s *Store) Maintain(ctx context.Context, log *slog.Logger) error {
	ticker := time.NewTicker(maintenanceInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("stop HTTP idempotency cleanup: %w", ctx.Err())
		case <-ticker.C:
			if _, err := s.Cleanup(ctx); err != nil && log != nil {
				log.WarnContext(ctx, "http idempotency cleanup failed", "error", err)
			}
		}
	}
}

// storageFailure classifies a storage error as ErrUnavailable, except that a
// canceled or expired ctx returns its own error instead.
func storageFailure(ctx context.Context, stage string, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return fmt.Errorf("%s: %w", stage, ctxErr)
	}
	return fmt.Errorf("%w: %s: %w", httpidempotency.ErrUnavailable, stage, err)
}

func durationMicros(duration time.Duration) int64 {
	micros := duration / time.Microsecond
	if duration%time.Microsecond != 0 {
		micros++
	}
	return int64(micros)
}
