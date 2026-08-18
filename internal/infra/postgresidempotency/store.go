// Package postgresidempotency executes one authorized HTTP mutation and its
// replay evidence in the same PostgreSQL transaction.
package postgresidempotency

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/example/go-service-template-rest/internal/httpidempotency"
	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5"
)

const CleanupBatchSize = 500

var (
	ErrConfig         = errors.New("postgres idempotency config")
	ErrUnavailable    = httpidempotency.ErrUnavailable
	ErrMismatch       = httpidempotency.ErrMismatch
	ErrIntegrity      = httpidempotency.ErrIntegrity
	ErrOutcomeUnknown = httpidempotency.ErrOutcomeUnknown
)

type Store struct {
	pool      *postgres.Pool
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
	if store == nil || bind == nil || codec.Encode == nil || codec.Decode == nil {
		return nil, fmt.Errorf("%w: store, repository binding, and response codec are required", ErrConfig)
	}
	return &Executor[Repository, Response]{store: store, bind: bind, codec: codec}, nil
}

func (e *Executor[Repository, Response]) Execute(
	ctx context.Context,
	request httpidempotency.Request,
	work httpidempotency.Work[Repository, Response],
) (response Response, replayed bool, err error) {
	if e == nil || e.store == nil || e.bind == nil || e.codec.Encode == nil || e.codec.Decode == nil || work == nil {
		return response, false, fmt.Errorf("%w: executor and work are required", ErrConfig)
	}
	result, replayed, err := e.store.Execute(ctx, request, func(ctx context.Context, tx pgx.Tx) (httpidempotency.Result, error) {
		response, err = work(ctx, e.bind(tx))
		if err != nil {
			return httpidempotency.Result{}, err
		}
		return e.codec.Encode(response)
	})
	if err != nil {
		return response, false, err
	}
	response, err = e.codec.Decode(result)
	if err != nil {
		return response, replayed, fmt.Errorf("decode idempotency response: %w", err)
	}
	return response, replayed, nil
}

func NewStore(pool *postgres.Pool, retention time.Duration) (*Store, error) {
	if pool == nil || pool.PGX() == nil {
		return nil, fmt.Errorf("%w: postgres pool is required", ErrConfig)
	}
	if retention <= 0 {
		return nil, fmt.Errorf("%w: retention must be positive", ErrConfig)
	}
	return &Store{pool: pool, retention: retention, inTx: pool.InTx}, nil
}

// Execute serializes one scoped key at PostgreSQL, runs work only for the
// winner, and commits its result with the business effect. The boolean reports
// whether the returned result was replayed.
func (s *Store) Execute(
	ctx context.Context,
	request httpidempotency.Request,
	work func(context.Context, pgx.Tx) (httpidempotency.Result, error),
) (result httpidempotency.Result, replayed bool, err error) {
	if s == nil || s.pool == nil || s.inTx == nil || !request.Valid() || work == nil {
		return httpidempotency.Result{}, false, fmt.Errorf("%w: store, request, and work are required", ErrConfig)
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
		return httpidempotency.Result{}, false, fmt.Errorf("execute idempotency transaction: %w", err)
	}

	result, readErr := s.Read(ctx, request)
	if readErr == nil {
		return result, true, nil
	}
	return httpidempotency.Result{}, false, fmt.Errorf("%w: %w", ErrOutcomeUnknown, err)
}

func (s *Store) executeTransaction(
	ctx context.Context,
	tx pgx.Tx,
	request httpidempotency.Request,
	work func(context.Context, pgx.Tx) (httpidempotency.Result, error),
) (httpidempotency.Result, bool, error) {
	queries := sqlcgen.New(tx)
	for range 2 {
		claimed, err := s.claim(ctx, queries, request)
		if err != nil {
			return httpidempotency.Result{}, false, err
		}
		if claimed {
			result, err := s.executeWork(ctx, tx, queries, request, work)
			return result, false, err
		}
		row, err := queries.ReadHTTPIdempotency(ctx, request.Identity())
		if err != nil {
			return httpidempotency.Result{}, false, unavailable(ctx, "read", err)
		}
		result, found, err := storedResult(request, row)
		if err != nil {
			return httpidempotency.Result{}, false, err
		}
		if found {
			return result, true, nil
		}
	}
	return httpidempotency.Result{}, false, ErrUnavailable
}

func (s *Store) executeWork(
	ctx context.Context,
	tx pgx.Tx,
	queries *sqlcgen.Queries,
	request httpidempotency.Request,
	work func(context.Context, pgx.Tx) (httpidempotency.Result, error),
) (httpidempotency.Result, error) {
	result, err := work(ctx, tx)
	if err != nil {
		return httpidempotency.Result{}, err
	}
	encoded, err := httpidempotency.EncodeResult(result)
	if err != nil {
		return httpidempotency.Result{}, fmt.Errorf("encode idempotency result: %w", err)
	}
	completed, err := queries.CompleteHTTPIdempotency(ctx, sqlcgen.CompleteHTTPIdempotencyParams{
		Result:          encoded,
		RetentionMicros: durationMicros(s.retention),
		IdentityToken:   request.Identity(),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return httpidempotency.Result{}, ErrIntegrity
	}
	if err != nil {
		return httpidempotency.Result{}, unavailable(ctx, "complete", err)
	}
	if len(completed) == 0 {
		return httpidempotency.Result{}, ErrIntegrity
	}
	return result, nil
}

func storedResult(
	request httpidempotency.Request,
	row sqlcgen.ReadHTTPIdempotencyRow,
) (httpidempotency.Result, bool, error) {
	if !row.WriterPrimary {
		return httpidempotency.Result{}, false, ErrUnavailable
	}
	if !row.RowExists || !row.Live {
		return httpidempotency.Result{}, false, nil
	}
	if row.FingerprintVersion == nil || !request.MatchesFingerprint(*row.FingerprintVersion, row.Fingerprint) {
		return httpidempotency.Result{}, false, ErrMismatch
	}
	if len(row.Result) == 0 {
		return httpidempotency.Result{}, false, ErrIntegrity
	}
	result, err := httpidempotency.DecodeResult(row.Result)
	if err != nil {
		return httpidempotency.Result{}, false, ErrIntegrity
	}
	return result, true, nil
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

// Read resolves a committed result from the authoritative writer.
func (s *Store) Read(ctx context.Context, request httpidempotency.Request) (httpidempotency.Result, error) {
	if s == nil || s.pool == nil || !request.Valid() {
		return httpidempotency.Result{}, fmt.Errorf("%w: store and request are required", ErrConfig)
	}
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return httpidempotency.Result{}, unavailable(ctx, "read connection", err)
	}
	defer conn.Release()
	row, err := sqlcgen.New(conn).ReadHTTPIdempotency(ctx, request.Identity())
	if err != nil {
		return httpidempotency.Result{}, unavailable(ctx, "read", err)
	}
	result, found, err := storedResult(request, row)
	if err != nil {
		return httpidempotency.Result{}, err
	}
	if !found {
		return httpidempotency.Result{}, ErrOutcomeUnknown
	}
	return result, nil
}

// Cleanup removes one bounded batch of expired replay rows.
func (s *Store) Cleanup(ctx context.Context) (int64, error) {
	if s == nil || s.pool == nil {
		return 0, fmt.Errorf("%w: store is required", ErrConfig)
	}
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return 0, unavailable(ctx, "cleanup connection", err)
	}
	defer conn.Release()
	rows, err := sqlcgen.New(conn).CleanupHTTPIdempotency(ctx, CleanupBatchSize)
	if err != nil {
		return 0, unavailable(ctx, "cleanup", err)
	}
	return rows, nil
}

func unavailable(ctx context.Context, stage string, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return fmt.Errorf("%s: %w", stage, ctxErr)
	}
	return fmt.Errorf("%w: %s: %w", ErrUnavailable, stage, err)
}

func durationMicros(duration time.Duration) int64 {
	micros := duration / time.Microsecond
	if duration%time.Microsecond != 0 {
		micros++
	}
	return int64(micros)
}
