package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrConfig      = errors.New("postgres config")
	ErrConnect     = errors.New("postgres connect")
	ErrHealthcheck = errors.New("postgres healthcheck")
)

type Options struct {
	DSN                string
	ConnectTimeout     time.Duration
	HealthcheckTimeout time.Duration
	MaxOpenConns       int
	MaxIdleConns       int
	ConnMaxLifetime    time.Duration
}

type Pool struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, opts Options) (*Pool, error) {
	if strings.TrimSpace(opts.DSN) == "" {
		return nil, fmt.Errorf("postgres dsn is empty")
	}
	if opts.ConnectTimeout <= 0 {
		return nil, fmt.Errorf("%w: connect timeout must be > 0", ErrConfig)
	}
	if opts.HealthcheckTimeout <= 0 {
		return nil, fmt.Errorf("%w: healthcheck timeout must be > 0", ErrConfig)
	}
	if opts.MaxOpenConns <= 0 {
		return nil, fmt.Errorf("%w: max open conns must be > 0", ErrConfig)
	}
	if opts.MaxIdleConns < 0 || opts.MaxIdleConns > opts.MaxOpenConns {
		return nil, fmt.Errorf("%w: max idle conns must be in range [0,max_open_conns]", ErrConfig)
	}
	if opts.ConnMaxLifetime <= 0 {
		return nil, fmt.Errorf("%w: conn max lifetime must be > 0", ErrConfig)
	}

	poolConfig, err := pgxpool.ParseConfig(opts.DSN)
	if err != nil {
		return nil, fmt.Errorf("%w: parse postgres dsn: %w", ErrConfig, err)
	}
	poolConfig.ConnConfig.ConnectTimeout = opts.ConnectTimeout
	poolConfig.MaxConns = int32(opts.MaxOpenConns)
	poolConfig.MaxConnLifetime = opts.ConnMaxLifetime
	// Enforce max_idle_conns as an upper bound for retained idle connections.
	// pgxpool does not expose a direct MaxIdleConns knob.
	var poolRef atomic.Pointer[pgxpool.Pool]
	poolConfig.AfterRelease = func(_ *pgx.Conn) bool {
		pool := poolRef.Load()
		if pool == nil {
			return true
		}
		return shouldKeepReleasedConn(int32(opts.MaxIdleConns), pool.Stat().IdleConns())
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("%w: create pgx pool: %w", ErrConnect, err)
	}
	poolRef.Store(pool)

	pingCtx, cancel := context.WithTimeout(ctx, opts.HealthcheckTimeout)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("%w: ping postgres: %w", ErrHealthcheck, err)
	}

	return &Pool{pool: pool}, nil
}

func shouldKeepReleasedConn(maxIdleConns int32, idleConnsBeforeRelease int32) bool {
	if maxIdleConns <= 0 {
		return false
	}
	return idleConnsBeforeRelease < maxIdleConns
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
