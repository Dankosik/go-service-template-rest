package postgres

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
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
		return nil, fmt.Errorf("%w: postgres dsn is empty", ErrConfig)
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
	if opts.MaxOpenConns > math.MaxInt32 {
		return nil, fmt.Errorf("%w: max open conns must be <= %d", ErrConfig, math.MaxInt32)
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
	poolConfig.MaxConns = int32(opts.MaxOpenConns) // #nosec G115 -- validated to be <= math.MaxInt32 above.
	poolConfig.MaxConnLifetime = opts.ConnMaxLifetime
	installMaxIdleConnLimiter(poolConfig, int32(opts.MaxIdleConns)) // #nosec G115 -- validated via MaxIdleConns <= MaxOpenConns <= math.MaxInt32 above.

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("%w: create pgx pool: %w", ErrConnect, err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, opts.HealthcheckTimeout)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("%w: ping postgres: %w", ErrHealthcheck, err)
	}

	return &Pool{pool: pool}, nil
}

type maxIdleConnLimiter struct {
	mu           sync.Mutex
	maxIdleConns int32
	retained     map[*pgx.Conn]struct{}
}

func newMaxIdleConnLimiter(maxIdleConns int32) *maxIdleConnLimiter {
	return &maxIdleConnLimiter{
		maxIdleConns: maxIdleConns,
		retained:     make(map[*pgx.Conn]struct{}),
	}
}

func installMaxIdleConnLimiter(poolConfig *pgxpool.Config, maxIdleConns int32) {
	limiter := newMaxIdleConnLimiter(maxIdleConns)

	beforeAcquire := poolConfig.BeforeAcquire
	afterRelease := poolConfig.AfterRelease
	beforeClose := poolConfig.BeforeClose

	poolConfig.BeforeAcquire = func(ctx context.Context, conn *pgx.Conn) bool {
		limiter.beforeAcquire(conn)
		return beforeAcquire == nil || beforeAcquire(ctx, conn)
	}
	poolConfig.AfterRelease = func(conn *pgx.Conn) bool {
		if afterRelease != nil && !afterRelease(conn) {
			return false
		}
		return limiter.afterRelease(conn)
	}
	poolConfig.BeforeClose = func(conn *pgx.Conn) {
		limiter.beforeClose(conn)
		if beforeClose != nil {
			beforeClose(conn)
		}
	}
}

func (l *maxIdleConnLimiter) afterRelease(conn *pgx.Conn) bool {
	if l.maxIdleConns <= 0 {
		return false
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if _, ok := l.retained[conn]; ok {
		return true
	}
	if int32(len(l.retained)) >= l.maxIdleConns {
		return false
	}
	l.retained[conn] = struct{}{}
	return true
}

func (l *maxIdleConnLimiter) beforeAcquire(conn *pgx.Conn) {
	l.beforeClose(conn)
}

func (l *maxIdleConnLimiter) beforeClose(conn *pgx.Conn) {
	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.retained, conn)
}

func (p *Pool) DB() *pgxpool.Pool {
	if p == nil {
		return nil
	}
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
		return fmt.Errorf("%w: postgres pool is nil", ErrHealthcheck)
	}
	if err := p.pool.Ping(ctx); err != nil {
		return fmt.Errorf("%w: ping postgres: %w", ErrHealthcheck, err)
	}
	return nil
}
