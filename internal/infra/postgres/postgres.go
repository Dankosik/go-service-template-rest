package postgres

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

const poolName = "postgres"

var (
	ErrConfig      = errors.New("postgres config")
	ErrConnect     = errors.New("postgres connect")
	ErrHealthcheck = errors.New("postgres healthcheck")
	ErrTransaction = errors.New("postgres transaction")
)

type Options struct {
	DSN                string
	ConnectTimeout     time.Duration
	HealthcheckTimeout time.Duration
	MaxOpenConns       int
	ConnMaxLifetime    time.Duration
	// StatementTimeout bounds every statement server-side, and bounds how long a
	// session may sit idle inside a transaction. The request budget only cancels
	// client-side, and pgx delivers that cancellation as a separate CancelRequest
	// on a separate connection — so when the network is the thing that went
	// wrong, which is the common case for a slow dependency, the cancel is
	// exactly what fails to arrive and the abandoned query keeps its locks.
	StatementTimeout time.Duration
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
	if opts.StatementTimeout <= 0 {
		return nil, fmt.Errorf("%w: statement timeout must be > 0", ErrConfig)
	}

	poolConfig, err := parsePoolConfig(opts.DSN)
	if err != nil {
		return nil, err
	}
	poolConfig.ConnConfig.ConnectTimeout = opts.ConnectTimeout
	applyStatementTimeouts(poolConfig.ConnConfig, opts.StatementTimeout)
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

// applyStatementTimeouts publishes the budget as session defaults on every
// pooled connection.
//
// These are startup parameters rather than a per-query SET, so they apply to
// connections the pool opens later without any query path having to remember.
// idle_in_transaction_session_timeout covers the case statement_timeout cannot:
// a transaction that was opened, ran a fast statement, and then lost its client
// holds its locks indefinitely while no statement is running at all.
func applyStatementTimeouts(connConfig *pgx.ConnConfig, statementTimeout time.Duration) {
	if connConfig == nil {
		return
	}
	if connConfig.RuntimeParams == nil {
		connConfig.RuntimeParams = make(map[string]string, 2)
	}

	// PostgreSQL reads a bare integer for these settings as milliseconds.
	milliseconds := strconv.FormatInt(statementTimeout.Milliseconds(), 10)
	connConfig.RuntimeParams["statement_timeout"] = milliseconds
	connConfig.RuntimeParams["idle_in_transaction_session_timeout"] = milliseconds
}

func (p *Pool) Close() {
	if p == nil || p.pool == nil {
		return
	}
	p.pool.Close()
}

// InTx runs fn inside one transaction, committing when it returns nil and
// rolling back otherwise.
//
// This is the seam that keeps a service from reaching for PGX() to compose two
// repository calls atomically. fn receives pgx.Tx, which carries the same
// Query/Exec/QueryRow surface as the pool, so a repository method written against
// that surface works both inside and outside a transaction without knowing which
// it is in.
func (p *Pool) InTx(ctx context.Context, opts pgx.TxOptions, fn func(pgx.Tx) error) error {
	if p == nil || p.pool == nil {
		return fmt.Errorf("%w: postgres pool is nil", ErrConfig)
	}
	if fn == nil {
		return fmt.Errorf("%w: transaction function is required", ErrConfig)
	}

	if err := pgx.BeginTxFunc(ctx, p.pool, opts, fn); err != nil {
		return fmt.Errorf("%w: %w", ErrTransaction, err)
	}
	return nil
}

// Retryable reports whether err is a PostgreSQL failure that the same request
// could succeed at if it ran again.
//
// There is deliberately no retry loop here. Whether a retry is safe depends on
// what the caller already did — a serialization failure inside a read-only query
// is free to retry, the same failure after an outbound side effect is not — and
// how many attempts fit the request's remaining budget. Classification is the
// part every service needs and gets wrong; the policy is the part that differs.
func Retryable(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	switch pgErr.Code {
	case pgerrcode.SerializationFailure, pgerrcode.DeadlockDetected:
		return true
	default:
		return false
	}
}

// PGX returns the concrete pool for sqlc and transaction wiring at composition.
func (p *Pool) PGX() *pgxpool.Pool {
	if p == nil {
		return nil
	}
	return p.pool
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
