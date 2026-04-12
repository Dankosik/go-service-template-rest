package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
)

// ErrPingHistoryRepository classifies errors from the template ping_history sample repository.
var ErrPingHistoryRepository = errors.New("ping history repository")

const (
	pingHistorySampleMaxListLimit int32 = 100
)

// PingHistoryRecord is the adapter-safe representation of one template ping_history sample row.
// It is sample-local; future app-facing records and ports belong beside internal/app/<feature>.
type PingHistoryRecord struct {
	ID        int64
	Payload   string
	CreatedAt time.Time
}

type pingHistoryQuerier interface {
	CreatePingHistory(ctx context.Context, payload string) (sqlcgen.PingHistory, error)
	ListRecentPingHistory(ctx context.Context, limit int32) ([]sqlcgen.PingHistory, error)
}

// PingHistoryRepository is a template SQLC sample repository, not production ping behavior.
type PingHistoryRepository struct {
	queries pingHistoryQuerier
}

// NewPingHistoryRepository builds the template sample repository backed by sqlc generated queries.
func NewPingHistoryRepository(pool *Pool) (*PingHistoryRepository, error) {
	if pool == nil || pool.pool == nil {
		return nil, fmt.Errorf("%w: postgres pool is required", ErrPingHistoryRepository)
	}
	return newPingHistoryRepository(sqlcgen.New(pool.pool))
}

func newPingHistoryRepository(queries pingHistoryQuerier) (*PingHistoryRepository, error) {
	if queries == nil {
		return nil, fmt.Errorf("%w: queries are required", ErrPingHistoryRepository)
	}
	return &PingHistoryRepository{
		queries: queries,
	}, nil
}

func (r *PingHistoryRepository) Create(ctx context.Context, payload string) (PingHistoryRecord, error) {
	if err := r.requireQueries(); err != nil {
		return PingHistoryRecord{}, err
	}

	row, err := r.queries.CreatePingHistory(ctx, payload)
	if err != nil {
		return PingHistoryRecord{}, fmt.Errorf("%w: create ping history: %w", ErrPingHistoryRepository, err)
	}

	record, err := mapPingHistoryRecord(row)
	if err != nil {
		return PingHistoryRecord{}, fmt.Errorf("create ping history: %w", err)
	}

	return record, nil
}

func (r *PingHistoryRepository) ListRecent(ctx context.Context, limit int32) ([]PingHistoryRecord, error) {
	if err := r.requireQueries(); err != nil {
		return nil, err
	}
	if err := validatePingHistoryListLimit(limit); err != nil {
		return nil, err
	}

	rows, err := r.queries.ListRecentPingHistory(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("%w: list recent ping history: %w", ErrPingHistoryRepository, err)
	}

	records := make([]PingHistoryRecord, 0, len(rows))
	for _, row := range rows {
		record, err := mapPingHistoryRecord(row)
		if err != nil {
			return nil, fmt.Errorf("list recent ping history: %w", err)
		}
		records = append(records, record)
	}

	return records, nil
}

func (r *PingHistoryRepository) requireQueries() error {
	if r == nil || r.queries == nil {
		return fmt.Errorf("%w: queries are not configured", ErrPingHistoryRepository)
	}
	return nil
}

func validatePingHistoryListLimit(limit int32) error {
	if limit <= 0 {
		return fmt.Errorf("%w: limit must be > 0", ErrPingHistoryRepository)
	}
	if limit > pingHistorySampleMaxListLimit {
		return fmt.Errorf("%w: limit must be <= %d", ErrPingHistoryRepository, pingHistorySampleMaxListLimit)
	}
	return nil
}

func mapPingHistoryRecord(row sqlcgen.PingHistory) (PingHistoryRecord, error) {
	if !row.CreatedAt.Valid {
		return PingHistoryRecord{}, fmt.Errorf("%w: created_at is null", ErrPingHistoryRepository)
	}

	return PingHistoryRecord{
		ID:        row.ID,
		Payload:   row.Payload,
		CreatedAt: row.CreatedAt.Time,
	}, nil
}
