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

const (
	poolName = "postgres"

	DefaultMaxOpenConns       = 4
	DefaultHealthcheckTimeout = 3 * time.Second
	DefaultStatementTimeout   = 8 * time.Second
	postgresConnectTimeout    = 3 * time.Second
)

var (
	ErrConfig      = errors.New("postgres config")
	ErrConnect     = errors.New("postgres connect")
	ErrHealthcheck = errors.New("postgres healthcheck")
	ErrTransaction = errors.New("postgres transaction")
	// ErrCommitUnknown means PostgreSQL may have committed even though the
	// client did not receive a definitive result. Callers must reconcile or
	// preserve the original operation identity instead of retrying blindly.
	ErrCommitUnknown = errors.New("postgres commit outcome unknown")

	contextWatcherMarks sync.Map // map[*pgconn.PgConn]*contextWatcherMark
)

type contextWatcherMark struct {
	canceled atomic.Bool
}

type Options struct {
	DSN          string
	MaxOpenConns int
}

func Open(ctx context.Context, opts Options) (*pgxpool.Pool, error) {
	if strings.TrimSpace(opts.DSN) == "" {
		return nil, fmt.Errorf("%w: postgres dsn is empty", ErrConfig)
	}
	if opts.MaxOpenConns <= 0 {
		return nil, fmt.Errorf("%w: max open conns must be > 0", ErrConfig)
	}
	if opts.MaxOpenConns > math.MaxInt32 {
		return nil, fmt.Errorf("%w: max open conns must be <= %d", ErrConfig, math.MaxInt32)
	}
	poolConfig, err := parsePoolConfig(opts.DSN)
	if err != nil {
		return nil, err
	}
	poolConfig.ConnConfig.ConnectTimeout = postgresConnectTimeout
	applyStatementTimeouts(poolConfig.ConnConfig, DefaultStatementTimeout)
	applyContextWatcher(poolConfig.ConnConfig, DefaultStatementTimeout)
	poolConfig.ConnConfig.Tracer = otelpgx.NewTracer(
		otelpgx.WithTrimSQLInSpanName(),
		otelpgx.WithSpanNameFunc(postgresOperationName),
		otelpgx.WithDisableSQLStatementInAttributes(),
		otelpgx.WithDisableConnectionDetailsInAttributes(),
	)
	poolConfig.MaxConns = int32(opts.MaxOpenConns) // #nosec G115 -- validated to be <= math.MaxInt32 above.

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("%w: create pgx pool: %w", ErrConnect, err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, DefaultHealthcheckTimeout)
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

	return pool, nil
}

func applyContextWatcher(connConfig *pgx.ConnConfig, statementTimeout time.Duration) {
	if connConfig == nil {
		return
	}
	connConfig.BuildContextWatcherHandler = func(conn *pgconn.PgConn) ctxwatch.Handler {
		return &contextWatcherHandler{
			conn: conn,
			handler: &pgconn.CancelRequestContextWatcherHandler{
				Conn:               conn,
				CancelRequestDelay: 0,
				DeadlineDelay:      statementTimeout,
			},
		}
	}
}

type contextWatcherHandler struct {
	conn    *pgconn.PgConn
	handler ctxwatch.Handler
}

func (h *contextWatcherHandler) HandleCancel(ctx context.Context) {
	if value, ok := contextWatcherMarks.Load(h.conn); ok {
		if marker, ok := value.(*contextWatcherMark); ok {
			marker.canceled.Store(true)
		}
	}
	h.handler.HandleCancel(ctx)
}

func (h *contextWatcherHandler) HandleUnwatchAfterCancel() {
	h.handler.HandleUnwatchAfterCancel()
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

// RuntimeParamMilliseconds renders a duration as a PostgreSQL runtime-parameter
// value. internal/infra/postgresmigrate publishes its own timeouts through it.
//
// Rounded up rather than truncated so a caller never publishes less time than
// its timeout. One millisecond more cannot fail a statement that would have
// succeeded; one millisecond less can cancel one.
//
// The unit is written out because PostgreSQL reads a bare integer against each
// setting's own default unit, which is milliseconds for these three and not for
// every setting a caller might add next.
func RuntimeParamMilliseconds(duration time.Duration) string {
	milliseconds := math.Ceil(float64(duration) / float64(time.Millisecond))
	return strconv.FormatInt(int64(milliseconds), 10) + "ms"
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
