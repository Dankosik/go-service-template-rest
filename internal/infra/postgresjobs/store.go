package postgresjobs

import (
	"context"
	"fmt"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	infratelemetry "github.com/example/go-service-template-rest/internal/infra/telemetry"
	"github.com/example/go-service-template-rest/internal/jobs"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type StoreOptions struct {
	OperationTimeout time.Duration
	StatementTimeout time.Duration
}

type Store struct {
	pool             *postgres.Pool
	operationTimeout time.Duration
	statementTimeout time.Duration
	events           metric.Int64Counter
	operationTime    metric.Float64Histogram
}

func NewStore(pool *postgres.Pool, options StoreOptions) (*Store, error) {
	if options.OperationTimeout <= 0 {
		return nil, fmt.Errorf("%w: operation timeout must be positive", ErrConfig)
	}
	if options.StatementTimeout <= 0 {
		return nil, fmt.Errorf("%w: statement timeout must be positive", ErrConfig)
	}
	if pool == nil || pool.PGX() == nil {
		return nil, fmt.Errorf("%w: postgres pool is required", ErrConfig)
	}
	events, err := infratelemetry.MeterOrGlobal(nil, jobsMeterName).Int64Counter("postgres.jobs.events")
	if err != nil {
		return nil, fmt.Errorf("create PostgreSQL jobs acceptance telemetry: %w", err)
	}
	operationTime, err := infratelemetry.MeterOrGlobal(nil, jobsMeterName).Float64Histogram("postgres.jobs.store.operation.duration", metric.WithUnit("s"))
	if err != nil {
		return nil, fmt.Errorf("create PostgreSQL jobs Store telemetry: %w", err)
	}
	return &Store{
		pool: pool, operationTimeout: options.OperationTimeout,
		statementTimeout: options.StatementTimeout, events: events, operationTime: operationTime,
	}, nil
}

func (s *Store) recordOperation(ctx context.Context, operation string, err error, duration time.Duration) {
	if s == nil || s.operationTime == nil {
		return
	}
	outcome := jobs.OutcomeSuccess
	if err != nil {
		outcome = jobs.OutcomeUnknown
	}
	s.operationTime.Record(ctx, duration.Seconds(), metric.WithAttributes(
		attribute.String("operation", metricOperation(operation)),
		attribute.String("outcome", metricOutcome(outcome)),
	))
}

func (s *Store) recordAcceptance(ctx context.Context, result jobs.StageResult, err error) {
	if s == nil || s.events == nil {
		return
	}
	event := "acceptance_rejected"
	outcome := jobs.OutcomeUnknown
	if err == nil {
		switch result.Outcome {
		case jobs.StageNew:
			event = "acceptance"
			outcome = jobs.OutcomeSuccess
		case jobs.StageExisting:
			event = "acceptance_duplicate"
			outcome = jobs.OutcomeSuccess
		case jobs.StageConflict:
			event = "acceptance_conflict"
			outcome = jobs.OutcomePermanent
		case jobs.StageRejected:
			event = "acceptance_rejected"
			outcome = jobs.OutcomeUnknown
		}
	}
	s.events.Add(ctx, 1, metric.WithAttributes(
		attribute.String("event", metricEvent(event)),
		attribute.String("outcome", metricOutcome(outcome)),
	))
}

func (s *Store) valid() bool {
	return s != nil && s.pool != nil && s.pool.PGX() != nil && s.operationTimeout > 0 && s.statementTimeout > 0
}

func (s *Store) AcquireSession(ctx context.Context) (*Session, error) {
	if !s.valid() {
		return nil, fmt.Errorf("%w: Store is required", ErrConfig)
	}
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquire postgres jobs Session: %w", err)
	}
	return &Session{store: s, conn: conn, queries: sqlcgen.New(conn)}, nil
}

func (s *Store) CheckSchema(ctx context.Context) error {
	session, err := s.AcquireSession(ctx)
	if err != nil {
		return err
	}
	defer session.Release(ctx)
	return session.CheckSchema(ctx)
}

type Session struct {
	store    *Store
	conn     *pgxpool.Conn
	queries  *sqlcgen.Queries
	released bool
	terminal bool
}

func (s *Session) valid() bool {
	return s != nil && !s.released && s.store != nil && s.store.valid() && s.conn != nil && s.queries != nil
}

func (s *Session) BackendPID() uint32 {
	if !s.valid() || s.conn.Conn() == nil || s.conn.Conn().PgConn() == nil {
		return 0
	}
	return s.conn.Conn().PgConn().PID()
}

func (s *Session) Release(ctx context.Context) {
	if s == nil || s.released || s.conn == nil {
		return
	}
	s.released = true
	if s.terminal && s.conn.Conn() != nil && !s.conn.Conn().IsClosed() {
		_ = s.conn.Hijack().Close(ctx)
		return
	}
	s.conn.Release()
}
