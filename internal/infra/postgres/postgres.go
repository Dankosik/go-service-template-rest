package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Pool struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, dsn string) (*Pool, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, fmt.Errorf("postgres dsn is empty")
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("create pgx pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return &Pool{pool: pool}, nil
}

func (p *Pool) DB() *pgxpool.Pool {
	return p.pool
}

func (p *Pool) Close() {
	if p == nil || p.pool == nil {
		return
	}
	p.pool.Close()
}

func (p *Pool) Name() string {
	return "postgres"
}

func (p *Pool) Check(ctx context.Context) error {
	if p == nil || p.pool == nil {
		return fmt.Errorf("postgres pool is nil")
	}
	return p.pool.Ping(ctx)
}
