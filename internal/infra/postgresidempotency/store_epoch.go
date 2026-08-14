package postgresidempotency

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/example/go-service-template-rest/internal/httpidempotency"
	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5"
)

// MaterializeEpoch stores the completed row's exact PostgreSQL commit timestamp.
// It never substitutes statement or wall-clock time when the source is absent.
func (s *Store) MaterializeEpoch(ctx context.Context, attempt httpidempotency.Attempt) (time.Time, error) {
	epoch, err := s.materializeEpoch(ctx, attempt)
	if errors.Is(err, ErrEpochLost) || errors.Is(err, ErrIntegrityConflict) {
		s.markTerminal(err)
	}
	return epoch, err
}

func (s *Store) materializeEpoch(ctx context.Context, attempt httpidempotency.Attempt) (time.Time, error) {
	if !s.valid() {
		return time.Time{}, fmt.Errorf("%w: store is required", ErrConfig)
	}
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return time.Time{}, unavailable(ctx, "epoch connection")
	}
	defer conn.Release()
	queries := sqlcgen.New(conn)
	writer, err := queries.CheckHTTPIdempotencyWriter(ctx)
	if err != nil {
		return time.Time{}, unavailable(ctx, "check epoch writer")
	}
	if !writer {
		return time.Time{}, ErrUnavailable
	}

	epoch, err := queries.MaterializeHTTPIdempotencyCommitEpoch(ctx, identityBytes(attempt))
	if err == nil {
		if !epoch.Valid {
			return time.Time{}, ErrEpochLost
		}
		return epoch.Time, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, unavailable(ctx, "materialize epoch")
	}

	row, err := queries.ReadHTTPIdempotency(ctx, identityBytes(attempt))
	if err != nil {
		return time.Time{}, unavailable(ctx, "read epoch")
	}
	if !row.WriterPrimary {
		return time.Time{}, ErrUnavailable
	}
	if !row.RowExists || row.Phase == nil || *row.Phase != phaseCompleted {
		return time.Time{}, ErrIntegrityConflict
	}
	if !row.CommittedAt.Valid {
		return time.Time{}, ErrEpochLost
	}
	return row.CommittedAt.Time, nil
}
