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
	inTx      func(context.Context, pgx.TxOptions, func(pgx.Tx) error) error
}

// Executor binds a transaction-scoped feature repository before Work runs.
type Executor[Repository, Response any] struct {
	store *Store
	bind  func(pgx.Tx) Repository
	codec httpidempotency.Codec[Response]
}

func NewExecutor[Repository, Response any](
	store *Store,
	bind func(pgx.Tx) Repository,
	codec httpidempotency.Codec[Response],
) (*Executor[Repository, Response], error) {
	if store == nil || bind == nil || !codec.Valid() {
		return nil, fmt.Errorf("%w: store, repository binding, and response codec are required", ErrConfig)
	}
	return &Executor[Repository, Response]{store: store, bind: bind, codec: codec}, nil
}

func (e *Executor[Repository, Response]) Execute(
	ctx context.Context,
	request httpidempotency.Request,
	work httpidempotency.Work[Repository, Response],
) (response Response, replayed bool, err error) {
	if e == nil || e.store == nil || e.bind == nil || !e.codec.Valid() || work == nil {
		return response, false, fmt.Errorf("%w: executor and work are required", ErrConfig)
	}
	result, replayed, err := e.store.execute(ctx, request, func(ctx context.Context, tx pgx.Tx) ([]byte, error) {
		response, err = work(ctx, e.bind(tx))
		if err != nil {
			return nil, err
		}
		return e.codec.Encode(response)
	})
	if err != nil {
		return response, false, fmt.Errorf("execute idempotent operation: %w", err)
	}
	response, err = e.codec.Decode(result)
	if err != nil {
		if replayed {
			return response, true, fmt.Errorf("%w: decode stored response", httpidempotency.ErrIntegrity)
		}
		return response, replayed, fmt.Errorf("decode idempotency response: %w", err)
	}
	return response, replayed, nil
}

func NewStore(pool *pgxpool.Pool, retention time.Duration) (*Store, error) {
	if pool == nil {
		return nil, fmt.Errorf("%w: postgres pool is required", ErrConfig)
	}
	if retention <= 0 {
		return nil, fmt.Errorf("%w: retention must be positive", ErrConfig)
	}
	return &Store{
		pool: pool, retention: retention,
		inTx: func(ctx context.Context, opts pgx.TxOptions, fn func(pgx.Tx) error) error {
			return postgres.InTx(ctx, pool, opts, fn)
		},
	}, nil
}

// execute serializes one scoped key at PostgreSQL, runs work only for the
// winner, and commits its result with the business effect. The boolean reports
// whether the returned result was replayed.
func (s *Store) execute(
	ctx context.Context,
	request httpidempotency.Request,
	work func(context.Context, pgx.Tx) ([]byte, error),
) (result []byte, replayed bool, err error) {
	if s == nil || s.pool == nil || s.inTx == nil || !request.Valid() || work == nil {
		return nil, false, fmt.Errorf("%w: store, request, and work are required", ErrConfig)
	}

	err = s.inTx(ctx, pgx.TxOptions{}, func(tx pgx.Tx) error {
		var transactionErr error
		result, replayed, transactionErr = s.executeTransaction(ctx, tx, request, work)
		return transactionErr
	})
	if err == nil || replayed {
		return result, replayed, nil
	}
	if !errors.Is(err, postgres.ErrCommitUnknown) {
		return nil, false, fmt.Errorf("execute idempotency transaction: %w", err)
	}

	result, readErr := s.read(ctx, request)
	if readErr == nil {
		return result, true, nil
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
			return nil, false, unavailable(ctx, "read", err)
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
		return nil, unavailable(ctx, "complete", err)
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
		return false, unavailable(ctx, "claim", err)
	}
	return true, nil
}

func (s *Store) read(ctx context.Context, request httpidempotency.Request) ([]byte, error) {
	if s == nil || s.pool == nil || !request.Valid() {
		return nil, fmt.Errorf("%w: store and request are required", ErrConfig)
	}
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return nil, unavailable(ctx, "read connection", err)
	}
	defer conn.Release()
	row, err := sqlcgen.New(conn).ReadHTTPIdempotency(ctx, request.Identity())
	if err != nil {
		return nil, unavailable(ctx, "read", err)
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
		return 0, unavailable(ctx, "cleanup connection", err)
	}
	defer conn.Release()
	rows, err := sqlcgen.New(conn).CleanupHTTPIdempotency(ctx, cleanupBatchSize)
	if err != nil {
		return 0, unavailable(ctx, "cleanup", err)
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

func unavailable(ctx context.Context, stage string, err error) error {
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
