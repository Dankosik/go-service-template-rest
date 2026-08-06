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
	// ErrCommitUnknown means PostgreSQL may have committed even though the
	// client did not receive a definitive result. Callers must reconcile or
	// preserve the original operation identity instead of retrying blindly.
	ErrCommitUnknown = errors.New("postgres commit outcome unknown")

	// ErrSaturated reports that every pooled connection stayed busy for the whole
	// acquire budget. It is a distinct identity because it is the one database
	// failure that is not the database's fault and that the caller should retry:
	// the server is momentarily past capacity, so a repository maps it onto its
	// own "temporarily unavailable" domain error and the transport answers 503
	// with a Retry-After, rather than the 504 an exhausted request budget earns.
	ErrSaturated = errors.New("postgres pool saturated")
)

// Querier is the surface shared by a pooled connection and a transaction, so one
// repository method serves both without knowing which it is running in.
//
// This is the type Acquire and InTx hand out. Without it, a repository takes
// *pgxpool.Pool concretely and therefore cannot be composed into a transaction —
// which is the reach for PGX that InTx exists to prevent. sqlc generates the same
// shape as DBTX, so generated queriers accept these values unchanged.
type Querier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

var (
	_ Querier = (*pgxpool.Conn)(nil)
	_ Querier = (pgx.Tx)(nil)
)

type Options struct {
	DSN                string
	ConnectTimeout     time.Duration
	HealthcheckTimeout time.Duration
	MaxOpenConns       int
	// MinIdleConns keeps connections open through quiet periods. Left at zero,
	// pgxpool closes every idle connection after MaxConnIdleTime, so a
	// low-traffic service pays TCP, TLS, and authentication on the first request
	// of every spike — concurrently, once per connection the spike needs.
	MinIdleConns int
	// AcquireTimeout bounds how long one caller waits for a pooled connection.
	//
	// Nothing else does. pgxpool.Acquire waits until a connection frees or the
	// context is done, and the only deadline on that context is the request
	// budget — so a slow database does not shed, it queues: each waiting request
	// holds an in-flight slot for its whole remaining budget until every slot is
	// held by a connection waiter, and requests that touch no database at all are
	// shed for capacity the database took.
	AcquireTimeout  time.Duration
	ConnMaxLifetime time.Duration
	// StatementTimeout bounds every statement server-side, and bounds how long a
	// session may sit idle inside a transaction. The request budget only cancels
	// client-side, and pgx delivers that cancellation as a separate CancelRequest
	// on a separate connection — so when the network is the thing that went
	// wrong, which is the common case for a slow dependency, the cancel is
	// exactly what fails to arrive and the abandoned query keeps its locks.
	StatementTimeout time.Duration
}

type Pool struct {
	pool           *pgxpool.Pool
	acquireTimeout time.Duration
	commitTx       func(context.Context, pgx.Tx) error
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
	if opts.MinIdleConns < 0 || opts.MinIdleConns > opts.MaxOpenConns {
		return nil, fmt.Errorf("%w: min idle conns must be in range [0,%d]", ErrConfig, opts.MaxOpenConns)
	}
	if opts.AcquireTimeout <= 0 {
		return nil, fmt.Errorf("%w: acquire timeout must be > 0", ErrConfig)
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
	poolConfig.MaxConns = int32(opts.MaxOpenConns)     // #nosec G115 -- validated to be <= math.MaxInt32 above.
	poolConfig.MinIdleConns = int32(opts.MinIdleConns) // #nosec G115 -- validated to be <= MaxOpenConns, itself <= math.MaxInt32.
	poolConfig.MaxConnLifetime = opts.ConnMaxLifetime
	poolConfig.MaxConnLifetimeJitter = opts.ConnMaxLifetime / 10

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

	return &Pool{
		pool:           pool,
		acquireTimeout: opts.AcquireTimeout,
		commitTx:       commitTx,
	}, nil
}

// Acquire takes a pooled connection, bounded by the acquire budget rather than by
// whatever is left of the caller's request budget, and returns ErrSaturated when
// the budget runs out.
//
// This is how a repository gets a Querier outside a transaction. The budget is
// the reason it exists: an unbounded acquire turns a slow database into a
// service-wide outage, because every waiting request keeps its in-flight slot
// until the whole request budget is spent and the shedding limiter then rejects
// traffic that never needed the database. Failing one caller in a second, with a
// retryable identity, keeps the rest of the service serving.
//
// The returned connection must be released. Callers that want the pool's own
// unbudgeted convenience methods use PGX.
func (p *Pool) Acquire(ctx context.Context) (*pgxpool.Conn, error) {
	if p == nil || p.pool == nil {
		return nil, fmt.Errorf("%w: postgres pool is nil", ErrConfig)
	}

	acquireCtx, cancel := context.WithTimeout(ctx, p.acquireTimeout)
	defer cancel()

	conn, err := p.pool.Acquire(acquireCtx)
	if err == nil {
		return conn, nil
	}
	// Only the sub-budget expiring means saturation. A caller whose own context
	// was already done is reporting a spent request budget or a canceled client,
	// and calling that a capacity problem would hide both.
	if errors.Is(err, context.DeadlineExceeded) && ctx.Err() == nil {
		return nil, fmt.Errorf("%w: no connection available within %s", ErrSaturated, p.acquireTimeout)
	}
	return nil, fmt.Errorf("%w: acquire connection: %w", ErrConnect, err)
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
// This is the seam that keeps a service from reaching for PGX to compose two
// repository calls atomically. fn receives a pgx.Tx, which satisfies Querier, so a
// repository method written against Querier works both inside and outside a
// transaction. The connection is taken through Acquire, so opening a transaction
// is subject to the same acquire budget as any other database work.
func (p *Pool) InTx(ctx context.Context, opts pgx.TxOptions, fn func(pgx.Tx) error) error {
	if p == nil {
		return fmt.Errorf("%w: postgres pool is nil", ErrConfig)
	}
	if fn == nil {
		return fmt.Errorf("%w: transaction function is required", ErrConfig)
	}

	conn, err := p.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	tx, err := conn.BeginTx(ctx, opts)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrTransaction, err)
	}

	commit := p.commitTx
	if commit == nil {
		commit = commitTx
	}
	if err := runInTx(ctx, tx, fn, commit); err != nil {
		return fmt.Errorf("%w: %w", ErrTransaction, err)
	}
	return nil
}

func runInTx(
	ctx context.Context,
	tx pgx.Tx,
	fn func(pgx.Tx) error,
	commit func(context.Context, pgx.Tx) error,
) (err error) {
	defer func() {
		rollbackErr := tx.Rollback(ctx)
		if rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			err = rollbackErr
		}
	}()

	if err := fn(tx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}

	if err := commit(ctx, tx); err != nil {
		if commitDefinitelyFailed(err) {
			return err
		}
		return fmt.Errorf("%w: %w", ErrCommitUnknown, err)
	}
	return nil
}

func commitTx(ctx context.Context, tx pgx.Tx) error {
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

func commitDefinitelyFailed(err error) bool {
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
		return pgErr.Code != pgerrcode.TransactionResolutionUnknown &&
			pgErr.Code != pgerrcode.StatementCompletionUnknown
	}
	return errors.Is(err, pgx.ErrTxCommitRollback) || pgconn.SafeToRetry(err)
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

// Check reports whether the database is reachable, and deliberately does not
// report whether the pool is busy.
//
// A saturated pool is evidence that the database is answering — every connection
// is in use serving it. Treating that as a readiness failure is how a slow
// dependency becomes a fleet-wide outage: the refresher's own ping queues behind
// the traffic, readiness flips, the orchestrator evicts the instance, and its
// traffic moves to instances that are already saturated. This package's readiness
// is served from cache for exactly that reason, and the cache is worth nothing if
// what fills it is itself a pool waiter.
//
// The masking window is bounded and worth it. A pool that is saturated because
// the database died drains within one statement timeout: in-flight queries fail,
// their connections are destroyed, an acquire then succeeds, and the ping reports
// the truth.
func (p *Pool) Check(ctx context.Context) error {
	if p == nil || p.pool == nil {
		return fmt.Errorf("%w: postgres pool is nil", ErrHealthcheck)
	}

	conn, err := p.Acquire(ctx)
	if err != nil {
		if errors.Is(err, ErrSaturated) {
			return nil
		}
		return fmt.Errorf("%w: %w", ErrHealthcheck, err)
	}
	defer conn.Release()

	if err := conn.Ping(ctx); err != nil {
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
