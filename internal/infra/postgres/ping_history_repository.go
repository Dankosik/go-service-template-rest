package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var ErrPingHistoryRepository = errors.New("ping history repository")

const txRollbackTimeout = 5 * time.Second

// PingHistoryRecord is the adapter-safe representation of one ping_history row.
type PingHistoryRecord struct {
	ID        int64
	Payload   string
	CreatedAt time.Time
}

type pingHistoryQuerier interface {
	CreatePingHistory(ctx context.Context, payload string) (sqlcgen.PingHistory, error)
	ListRecentPingHistory(ctx context.Context, limit int32) ([]sqlcgen.PingHistory, error)
}

type pingHistoryDB interface {
	Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error)
	Query(context.Context, string, ...interface{}) (pgx.Rows, error)
	QueryRow(context.Context, string, ...interface{}) pgx.Row
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

type PingHistoryRepository struct {
	queries pingHistoryQuerier
	db      pingHistoryDB
}

// NewPingHistoryRepository builds a repository backed by sqlc generated queries.
func NewPingHistoryRepository(db pingHistoryDB) *PingHistoryRepository {
	return &PingHistoryRepository{
		queries: sqlcgen.New(db),
		db:      db,
	}
}

func newPingHistoryRepositoryWithQuerier(queries pingHistoryQuerier) *PingHistoryRepository {
	return &PingHistoryRepository{queries: queries}
}

func (r *PingHistoryRepository) Create(ctx context.Context, payload string) (PingHistoryRecord, error) {
	row, err := r.queries.CreatePingHistory(ctx, payload)
	if err != nil {
		return PingHistoryRecord{}, fmt.Errorf("create ping history: %w", err)
	}

	record, err := mapPingHistoryRecord(row)
	if err != nil {
		return PingHistoryRecord{}, fmt.Errorf("create ping history: %w", err)
	}

	return record, nil
}

func (r *PingHistoryRepository) ListRecent(ctx context.Context, limit int32) ([]PingHistoryRecord, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("%w: limit must be > 0", ErrPingHistoryRepository)
	}

	rows, err := r.queries.ListRecentPingHistory(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("list recent ping history: %w", err)
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

func (r *PingHistoryRepository) createAndListRecentInTx(ctx context.Context, payload string, limit int32) (PingHistoryRecord, []PingHistoryRecord, error) {
	if limit <= 0 {
		return PingHistoryRecord{}, nil, fmt.Errorf("%w: limit must be > 0", ErrPingHistoryRepository)
	}
	if r.db == nil {
		return PingHistoryRecord{}, nil, fmt.Errorf("%w: transaction starter is not configured", ErrPingHistoryRepository)
	}

	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return PingHistoryRecord{}, nil, fmt.Errorf("begin ping history transaction: %w", err)
	}
	defer func() {
		rollbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), txRollbackTimeout)
		defer cancel()
		_ = tx.Rollback(rollbackCtx)
	}()

	txQueries := sqlcgen.New(tx)

	createdRow, err := txQueries.CreatePingHistory(ctx, payload)
	if err != nil {
		return PingHistoryRecord{}, nil, fmt.Errorf("create ping history in transaction: %w", err)
	}
	created, err := mapPingHistoryRecord(createdRow)
	if err != nil {
		return PingHistoryRecord{}, nil, fmt.Errorf("create ping history in transaction: %w", err)
	}

	recentRows, err := txQueries.ListRecentPingHistory(ctx, limit)
	if err != nil {
		return PingHistoryRecord{}, nil, fmt.Errorf("list recent ping history in transaction: %w", err)
	}

	recent := make([]PingHistoryRecord, 0, len(recentRows))
	for _, row := range recentRows {
		record, err := mapPingHistoryRecord(row)
		if err != nil {
			return PingHistoryRecord{}, nil, fmt.Errorf("list recent ping history in transaction: %w", err)
		}
		recent = append(recent, record)
	}

	if err := tx.Commit(ctx); err != nil {
		return PingHistoryRecord{}, nil, fmt.Errorf("commit ping history transaction: %w", err)
	}

	return created, recent, nil
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
