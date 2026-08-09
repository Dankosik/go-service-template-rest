package postgresmigrate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

func openMigrationDB(opts MigrationOptions) (*sql.DB, error) {
	normalizedDSN, err := postgres.NormalizeDSN(opts.DSN)
	if err != nil {
		return nil, fmt.Errorf("validate postgres migration dsn: %w", err)
	}
	config, err := pgx.ParseConfig(normalizedDSN)
	if err != nil {
		return nil, errors.New("parse postgres migration dsn: invalid value redacted")
	}
	config.ConnectTimeout = opts.ConnectTimeout
	if config.RuntimeParams == nil {
		config.RuntimeParams = make(map[string]string)
	}
	config.RuntimeParams["statement_timeout"] = postgresDuration(opts.StatementTimeout)
	config.RuntimeParams["idle_in_transaction_session_timeout"] = postgresDuration(opts.StatementTimeout)
	config.RuntimeParams["lock_timeout"] = postgresDuration(opts.LockTimeout)
	config.RuntimeParams["application_name"] = "goose-migrate"

	db := stdlib.OpenDB(*config)
	db.SetMaxOpenConns(2)
	db.SetMaxIdleConns(0)
	return db, nil
}

func secondsCeiling(duration time.Duration) uint64 {
	return uint64(math.Max(1, math.Ceil(duration.Seconds())))
}

func postgresDuration(duration time.Duration) string {
	milliseconds := math.Ceil(float64(duration) / float64(time.Millisecond))
	return strconv.FormatInt(int64(milliseconds), 10) + "ms"
}

func executionContext(ctx context.Context, reserve time.Duration) (context.Context, context.CancelFunc, error) {
	deadline, hasDeadline := ctx.Deadline()
	if !hasDeadline {
		executionCtx, cancel := context.WithCancel(ctx)
		return executionCtx, cancel, nil
	}
	executionDeadline := deadline.Add(-reserve)
	if !executionDeadline.After(time.Now()) {
		return nil, nil, errors.New("migration deadline leaves no execution budget after cleanup reserve")
	}
	executionCtx, cancel := context.WithDeadline(ctx, executionDeadline)
	return executionCtx, cancel, nil
}

func detachedCleanupContext(ctx context.Context, budget time.Duration) (context.Context, context.CancelFunc) {
	deadline := time.Now().Add(budget)
	if outerDeadline, ok := ctx.Deadline(); ok && outerDeadline.Before(deadline) {
		deadline = outerDeadline
	}
	return context.WithDeadline(context.WithoutCancel(ctx), deadline)
}
