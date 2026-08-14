package postgresjobs

import (
	"context"
	"fmt"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgres/sqlcgen"
	"github.com/jackc/pgx/v5/pgxpool"
)

type StoreOptions struct {
	OperationTimeout time.Duration
	StatementTimeout time.Duration
}

type Store struct {
	pool             *postgres.Pool
	operationTimeout time.Duration
	statementTimeout time.Duration
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
	return &Store{
		pool: pool, operationTimeout: options.OperationTimeout,
		statementTimeout: options.StatementTimeout,
	}, nil
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
