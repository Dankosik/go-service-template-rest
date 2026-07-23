package postgres

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5/pgxpool"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

const poolName = "postgres"

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
	if opts.ConnMaxLifetime <= 0 {
		return nil, fmt.Errorf("%w: conn max lifetime must be > 0", ErrConfig)
	}

	poolConfig, err := parsePoolConfig(opts.DSN)
	if err != nil {
		return nil, err
	}
	poolConfig.ConnConfig.ConnectTimeout = opts.ConnectTimeout
	poolConfig.ConnConfig.Tracer = otelpgx.NewTracer(
		otelpgx.WithTrimSQLInSpanName(),
		otelpgx.WithSpanNameFunc(postgresOperationName),
		otelpgx.WithDisableSQLStatementInAttributes(),
		otelpgx.WithDisableConnectionDetailsInAttributes(),
	)
	poolConfig.MaxConns = int32(opts.MaxOpenConns) // #nosec G115 -- validated to be <= math.MaxInt32 above.
	poolConfig.MaxConnLifetime = opts.ConnMaxLifetime

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

	if err := otelpgx.RecordStats(
		pool,
		otelpgx.WithStatsAttributes(semconv.DBClientConnectionPoolName(poolName)),
	); err != nil {
		pool.Close()
		return nil, fmt.Errorf("record postgres pool metrics: %w", err)
	}

	return &Pool{pool: pool}, nil
}

func (p *Pool) Close() {
	if p == nil || p.pool == nil {
		return
	}
	p.pool.Close()
}

func (p *Pool) Name() string {
	return poolName
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

func postgresOperationName(statement string) string {
	for line := range strings.Lines(statement) {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "--") {
			continue
		}
		for operation := range strings.FieldsSeq(line) {
			return strings.ToUpper(operation)
		}
	}
	return "UNKNOWN"
}
