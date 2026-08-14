package postgres

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgconn/ctxwatch"
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
	// acquire budget. It has its own identity because it is the one database
	// failure the caller should retry: a repository maps it onto "temporarily
	// unavailable" and the transport answers 503, not the 504 an exhausted
	// request budget earns.
	ErrSaturated = errors.New("postgres pool saturated")
)

// Querier is the surface shared by a pooled connection and a transaction, so one
// repository method serves both without knowing which it is running in.
//
// Without it a repository takes *pgxpool.Pool concretely and cannot be composed
// into a transaction — the reach for PGX that InTx exists to prevent. sqlc
// generates the same shape as DBTX, so generated queriers accept these unchanged.
type Querier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

var (
	_ Querier = (*pgxpool.Conn)(nil)
	_ Querier = pgx.Tx(nil)
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
	// Nothing else does: pgxpool.Acquire waits until a connection frees or the
	// context is done, so without this a slow database queues rather than sheds —
	// every in-flight slot ends up held by a connection waiter, and requests that
	// touch no database at all are shed for capacity the database took.
	AcquireTimeout  time.Duration
	ConnMaxLifetime time.Duration
	// StatementTimeout bounds every statement server-side, and how long a session
	// may sit idle inside a transaction. It is needed because pgx delivers the
	// request budget's cancellation as a separate CancelRequest on a separate
	// connection: when the network is what went wrong, that cancel is exactly what
	// fails to arrive and the abandoned query keeps its locks.
	StatementTimeout time.Duration
}

type Pool struct {
	pool                *pgxpool.Pool
	acquireTimeout      time.Duration
	commitTx            func(context.Context, pgx.Tx) error
	contextWatcherMarks sync.Map // map[*pgconn.PgConn]*contextWatcherMark
}

type contextWatcherMark struct {
	canceled atomic.Bool
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
	p := &Pool{
		acquireTimeout: opts.AcquireTimeout,
		commitTx:       commitTx,
	}
	applyContextWatcher(poolConfig.ConnConfig, opts.StatementTimeout, &p.contextWatcherMarks)
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

	p.pool = pool
	return p, nil
}

// Acquire takes a pooled connection, bounded by the acquire budget rather than by
// whatever is left of the caller's request budget, and returns ErrSaturated when
// the budget runs out.
//
// This is how a repository gets a Querier outside a transaction; see
// PoolConfig.AcquireTimeout for why the budget is separate from the request's.
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
// connections the pool opens later. idle_in_transaction_session_timeout covers
// what statement_timeout cannot: a transaction that ran a fast statement and then
// lost its client holds its locks while no statement is running at all.
func applyStatementTimeouts(connConfig *pgx.ConnConfig, statementTimeout time.Duration) {
	if connConfig == nil {
		return
	}
	if connConfig.RuntimeParams == nil {
		connConfig.RuntimeParams = make(map[string]string, 2)
	}

	milliseconds := RuntimeParamMilliseconds(statementTimeout)
	connConfig.RuntimeParams["statement_timeout"] = milliseconds
	connConfig.RuntimeParams["idle_in_transaction_session_timeout"] = milliseconds
}

func applyContextWatcher(connConfig *pgx.ConnConfig, statementTimeout time.Duration, marks *sync.Map) {
	if connConfig == nil {
		return
	}
	connConfig.BuildContextWatcherHandler = func(conn *pgconn.PgConn) ctxwatch.Handler {
		return &contextWatcherHandler{
			marks: marks,
			conn:  conn,
			handler: &pgconn.CancelRequestContextWatcherHandler{
				Conn:               conn,
				CancelRequestDelay: 0,
				DeadlineDelay:      statementTimeout,
			},
		}
	}
}

type contextWatcherHandler struct {
	marks   *sync.Map
	conn    *pgconn.PgConn
	handler ctxwatch.Handler
}

func (h *contextWatcherHandler) HandleCancel(ctx context.Context) {
	if h.marks != nil {
		if value, ok := h.marks.Load(h.conn); ok {
			if marker, ok := value.(*contextWatcherMark); ok {
				marker.canceled.Store(true)
			}
		}
	}
	h.handler.HandleCancel(ctx)
}

func (h *contextWatcherHandler) HandleUnwatchAfterCancel() {
	h.handler.HandleUnwatchAfterCancel()
}

// RuntimeParamMilliseconds renders a duration as a PostgreSQL runtime-parameter
// value. internal/infra/postgresmigrate publishes its own timeouts through it.
//
// Rounded up rather than truncated, because these carry timeouts: a duration
// with a sub-millisecond remainder — 100500us is inside what configuration
// accepts — would otherwise be published as less time than the operator asked
// for. One millisecond more cannot fail a statement that would have succeeded;
// one millisecond less can cancel one.
//
// The unit is written out because PostgreSQL reads a bare integer against each
// setting's own default unit, which is milliseconds for these three and not for
// every setting a caller might add next.
func RuntimeParamMilliseconds(duration time.Duration) string {
	milliseconds := math.Ceil(float64(duration) / float64(time.Millisecond))
	return strconv.FormatInt(int64(milliseconds), 10) + "ms"
}

func (p *Pool) Close() {
	if p == nil || p.pool == nil {
		return
	}
	p.pool.Close()
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
// is in use serving it — so reporting it as unready would evict the instance for
// being busy. The masking window is bounded: a pool saturated because the
// database died drains within one statement timeout, and the ping then reports
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
