package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/example/go-service-template-rest/internal/idempotency"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrIdempotency reports a failure of the idempotency store itself, as opposed to
// the two client-visible outcomes in the idempotency package.
var ErrIdempotency = errors.New("postgres idempotency")

// Statements are single, so every operation is atomic without a transaction and
// the store needs nothing but a Querier.
//
// The claim is the part that cannot be written any other way. INSERT ... ON
// CONFLICT DO NOTHING is one atomic decision: exactly one of two concurrent
// attempts affects a row, and the loser learns it lost. The SELECT-then-INSERT
// shape a service reaches for instead has a window between the two statements
// where both attempts see no row, both insert, one fails on the primary key, and
// the retry it triggers charges the customer twice.
const (
	// make_interval takes the retention in seconds rather than casting a Go
	// duration string to interval: time.Duration.String() produces forms like
	// "24h0m0s", and relying on PostgreSQL's interval parser to accept them is a
	// dependency on a syntax nobody here controls.
	reserveIdempotencyKeyStatement = `
INSERT INTO idempotency_keys (key, fingerprint, expires_at)
VALUES ($1, $2, now() + make_interval(secs => $3))
ON CONFLICT (key) DO NOTHING`

	loadIdempotencyKeyStatement = `
SELECT fingerprint, status, headers, body, replayable, completed_at IS NOT NULL
FROM idempotency_keys
WHERE key = $1`

	completeIdempotencyKeyStatement = `
UPDATE idempotency_keys
SET status = $2, headers = $3, body = $4, replayable = $5, completed_at = now()
WHERE key = $1`

	releaseIdempotencyKeyStatement = `
DELETE FROM idempotency_keys
WHERE key = $1 AND completed_at IS NULL`

	sweepIdempotencyKeysStatement = `
DELETE FROM idempotency_keys
WHERE expires_at <= now()`
)

// IdempotencyStore remembers what the first attempt with a key answered.
//
// It satisfies the store interface internal/infra/http declares, without importing
// it: the shared response value lives in internal/idempotency, so the transport
// package declares the port and this package implements it structurally.
type IdempotencyStore struct {
	// acquire hands out a budgeted Querier and the release that returns it. It is a
	// function rather than the pool so the statement handling is testable without a
	// database; the pool-backed value is what New installs.
	acquire   func(context.Context) (Querier, func(), error)
	retention time.Duration
}

// NewIdempotencyStore builds the store on pool.
//
// retention is how long a completed outcome stays replayable. It is also what
// bounds the table: an entry past it is deleted by Sweep, which is registered as a
// supervised background task. Without expiry the table grows for the life of the
// service, which is the second half of what makes a hand-rolled store wrong.
func NewIdempotencyStore(pool *Pool, retention time.Duration) (*IdempotencyStore, error) {
	if pool == nil || pool.pool == nil {
		return nil, fmt.Errorf("%w: postgres pool is required", ErrConfig)
	}
	if retention <= 0 {
		return nil, fmt.Errorf("%w: idempotency retention must be > 0", ErrConfig)
	}

	return &IdempotencyStore{acquire: pooledQuerier(pool), retention: retention}, nil
}

// pooledQuerier takes connections through Pool.Acquire, so idempotency work is
// subject to the same acquire budget and the same ErrSaturated classification as
// any other query rather than waiting out the caller's whole remaining budget.
func pooledQuerier(pool *Pool) func(context.Context) (Querier, func(), error) {
	return func(ctx context.Context) (Querier, func(), error) {
		conn, err := pool.Acquire(ctx)
		if err != nil {
			return nil, nil, err
		}
		return conn, conn.Release, nil
	}
}

func (s *IdempotencyStore) Reserve(ctx context.Context, key, fingerprint string) (*idempotency.StoredResponse, error) {
	querier, release, err := s.acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer release()

	tag, err := querier.Exec(ctx, reserveIdempotencyKeyStatement, key, fingerprint, s.retention.Seconds())
	if err != nil {
		return nil, fmt.Errorf("%w: reserve key: %w", ErrIdempotency, err)
	}
	if tag.RowsAffected() == 1 {
		return nil, nil //nolint:nilnil // A nil response with a nil error is the contract: this attempt owns the key.
	}

	return s.loadReserved(ctx, querier, key, fingerprint)
}

// loadReserved classifies a key this attempt did not win.
func (s *IdempotencyStore) loadReserved(
	ctx context.Context,
	querier Querier,
	key string,
	fingerprint string,
) (*idempotency.StoredResponse, error) {
	var (
		storedFingerprint string
		status            *int32
		headers           []byte
		body              []byte
		replayable        bool
		complete          bool
	)
	err := querier.QueryRow(ctx, loadIdempotencyKeyStatement, key).
		Scan(&storedFingerprint, &status, &headers, &body, &replayable, &complete)
	if errors.Is(err, pgx.ErrNoRows) {
		// The row was swept between the failed insert and this read, so its
		// retention had already elapsed. Reporting in-flight is the honest answer:
		// this attempt does not own the key and has nothing to replay, and the
		// client's retry claims it cleanly.
		return nil, idempotency.ErrInFlight
	}
	if err != nil {
		return nil, fmt.Errorf("%w: load reserved key: %w", ErrIdempotency, err)
	}

	if storedFingerprint != fingerprint {
		return nil, idempotency.ErrKeyReused
	}
	if !complete {
		return nil, idempotency.ErrInFlight
	}

	response := idempotency.StoredResponse{Replayable: replayable}
	if !replayable {
		// The first attempt's response was too large to hold. The key is spent, and
		// saying so is what keeps the work from running twice.
		return &response, nil
	}
	if status != nil {
		response.Status = int(*status)
	}
	response.Body = body
	response.Header, err = decodeIdempotencyHeaders(headers)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

func (s *IdempotencyStore) Complete(ctx context.Context, key string, response idempotency.StoredResponse) error {
	headers, err := encodeIdempotencyHeaders(response.Header)
	if err != nil {
		return err
	}

	querier, release, err := s.acquire(ctx)
	if err != nil {
		return err
	}
	defer release()

	// #nosec G115 -- an HTTP status is always within int32.
	if _, err := querier.Exec(
		ctx,
		completeIdempotencyKeyStatement,
		key,
		int32(response.Status),
		headers,
		response.Body,
		response.Replayable,
	); err != nil {
		return fmt.Errorf("%w: complete key: %w", ErrIdempotency, err)
	}
	return nil
}

// Release deletes an incomplete reservation, and deliberately leaves a completed
// one alone: the guard keeps a late release from erasing an outcome another attempt
// is entitled to replay.
func (s *IdempotencyStore) Release(ctx context.Context, key string) error {
	querier, release, err := s.acquire(ctx)
	if err != nil {
		return err
	}
	defer release()

	if _, err := querier.Exec(ctx, releaseIdempotencyKeyStatement, key); err != nil {
		return fmt.Errorf("%w: release key: %w", ErrIdempotency, err)
	}
	return nil
}

// Sweep deletes expired entries and reports how many it removed.
//
// This is the half of the contract a store cannot skip. Without it the table keeps
// every key any client ever sent, so the index that makes Reserve one statement
// grows without bound and the vacuum cost grows with it.
func (s *IdempotencyStore) Sweep(ctx context.Context) (int64, error) {
	querier, release, err := s.acquire(ctx)
	if err != nil {
		return 0, err
	}
	defer release()

	tag, err := querier.Exec(ctx, sweepIdempotencyKeysStatement)
	if err != nil {
		return 0, fmt.Errorf("%w: sweep expired keys: %w", ErrIdempotency, err)
	}
	return tag.RowsAffected(), nil
}

func encodeIdempotencyHeaders(header http.Header) ([]byte, error) {
	if len(header) == 0 {
		return []byte("{}"), nil
	}
	encoded, err := json.Marshal(header)
	if err != nil {
		return nil, fmt.Errorf("%w: encode response headers: %w", ErrIdempotency, err)
	}
	return encoded, nil
}

func decodeIdempotencyHeaders(encoded []byte) (http.Header, error) {
	if len(encoded) == 0 {
		return http.Header{}, nil
	}
	header := http.Header{}
	if err := json.Unmarshal(encoded, &header); err != nil {
		return nil, fmt.Errorf("%w: decode response headers: %w", ErrIdempotency, err)
	}
	return header, nil
}

// idempotencyPoolConn keeps the compile-time link between what the pool hands out
// and what this store needs, so a change to Pool.Acquire's return type is a build
// failure here rather than at the composition root.
var _ func(context.Context) (*pgxpool.Conn, error) = (*Pool)(nil).Acquire
