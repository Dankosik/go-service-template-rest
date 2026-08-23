package postgresmigrate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/database"
	gooselock "github.com/pressly/goose/v3/lock"
)

type MigrationOptions struct {
	DSN              string
	SourceFS         fs.FS
	SourcePath       string
	ConnectTimeout   time.Duration
	StatementTimeout time.Duration
	LockTimeout      time.Duration
	CleanupTimeout   time.Duration
	Logger           *slog.Logger
}

const (
	DefaultTimeout          = 5 * time.Minute
	DefaultConnectTimeout   = 3 * time.Second
	DefaultStatementTimeout = 2 * time.Minute
	DefaultLockTimeout      = 15 * time.Second
)

func DefaultOptions(dsn string, source fs.FS, path string, logger *slog.Logger) MigrationOptions {
	return MigrationOptions{
		DSN:              dsn,
		SourceFS:         source,
		SourcePath:       path,
		ConnectTimeout:   DefaultConnectTimeout,
		StatementTimeout: DefaultStatementTimeout,
		LockTimeout:      DefaultLockTimeout,
		CleanupTimeout:   DefaultLockTimeout,
		Logger:           logger,
	}
}

type migrationDirection string

const (
	directionUp   migrationDirection = "up"
	directionDown migrationDirection = "down"
)

// MigrateUp applies every pending canonical migration under the Goose session lock.
func MigrateUp(ctx context.Context, opts MigrationOptions) (RunResult, error) {
	return migrate(ctx, opts, directionUp)
}

// MigrateDown rolls every applied migration back on a disposable database.
//
// Production composition deliberately exposes only MigrateUp.
func MigrateDown(ctx context.Context, opts MigrationOptions) (RunResult, error) {
	return migrate(ctx, opts, directionDown)
}

func migrate(
	ctx context.Context,
	opts MigrationOptions,
	direction migrationDirection,
) (result RunResult, retErr error) {
	started := time.Now()
	defer func() {
		result.Duration = time.Since(started)
	}()

	source, err := rootMigrationSource(opts.SourceFS, opts.SourcePath)
	if err != nil {
		return result, stageError(FailureSource, err)
	}
	if err := validateMigrationOptions(opts); err != nil {
		return result, stageError(FailureConfig, err)
	}

	executionCtx, cancelExecution, err := executionContext(ctx, opts.CleanupTimeout)
	if err != nil {
		return result, stageError(FailureConfig, err)
	}
	defer cancelExecution()

	db, err := openMigrationDB(opts)
	if err != nil {
		return result, stageError(FailureConfig, err)
	}

	locker, err := gooselock.NewPostgresSessionLocker(
		gooselock.WithLockTimeout(1, secondsCeiling(opts.LockTimeout)),
		gooselock.WithUnlockTimeout(1, secondsCeiling(opts.CleanupTimeout)),
	)
	if err != nil {
		_ = db.Close()
		return result, stageError(FailureConfig, fmt.Errorf("build goose session locker: %w", err))
	}
	stagedLocker := migrationSessionLocker{SessionLocker: locker}

	provider, err := goose.NewProvider(
		goose.DialectPostgres,
		db,
		source,
		goose.WithDisableGlobalRegistry(true),
		goose.WithLogger(goose.NopLogger()),
		goose.WithSessionLocker(stagedLocker),
		goose.WithVerbose(false),
	)
	if err != nil {
		if errors.Is(err, goose.ErrNoMigrations) {
			if pingErr := db.PingContext(executionCtx); pingErr != nil {
				_ = db.Close()
				return result, stageError(
					FailureConnect,
					fmt.Errorf("ping postgres migration database: %w", pingErr),
				)
			}
			return migrateEmptySource(ctx, executionCtx, opts, db, stagedLocker, direction, result)
		}
		_ = db.Close()
		return result, stageError(FailureSource, fmt.Errorf("build goose provider: %w", err))
	}
	defer func() {
		if closeErr := provider.Close(); closeErr != nil {
			retErr = stageError(FailureCleanup, errors.Join(retErr, fmt.Errorf("close goose provider: %w", closeErr)))
		}
	}()
	if err := provider.Ping(executionCtx); err != nil {
		return result, stageError(FailureConnect, fmt.Errorf("ping postgres migration database: %w", err))
	}

	current, target, err := provider.GetVersions(executionCtx)
	if err != nil {
		return result, stageError(FailureState, fmt.Errorf("read goose migration versions: %w", err))
	}
	result.Before = current
	result.After = current
	result.Target = target
	if direction == directionDown {
		result.Target = 0
	}
	sources := provider.ListSources()
	logMigrationPlan(executionCtx, opts.Logger, direction, result.Before, result.Target, len(sources))

	var gooseResults []*goose.MigrationResult
	if direction == directionUp {
		gooseResults, err = provider.Up(executionCtx)
	} else {
		gooseResults, err = provider.DownTo(executionCtx, 0)
	}
	applied := gooseResults
	var failed *goose.MigrationResult
	if err != nil {
		if partial, ok := errors.AsType[*goose.PartialError](err); ok {
			applied = partial.Applied
			failed = partial.Failed
		}
		setAfterFromApplied(&result, direction, sources, applied)
		logMigrationResults(executionCtx, opts.Logger, applied, failed)
		if stage := FailureStageOf(err); stage == FailureLock || stage == FailureCleanup {
			return result, fmt.Errorf("run goose migration: %w", err)
		}
		return result, stageError(FailureExecute, err)
	}

	setAfterFromApplied(&result, direction, sources, applied)
	after, _, stateErr := provider.GetVersions(executionCtx)
	if stateErr != nil {
		logMigrationResults(executionCtx, opts.Logger, applied, nil)
		return result, stageError(FailureState, fmt.Errorf("read goose migration versions: %w", stateErr))
	}
	if len(applied) == 0 {
		result.Before = after
	}
	result.After = after
	logMigrationResults(executionCtx, opts.Logger, applied, nil)
	return result, nil
}

type migrationSessionLocker struct {
	gooselock.SessionLocker
}

func (l migrationSessionLocker) SessionLock(ctx context.Context, conn *sql.Conn) error {
	if err := l.SessionLocker.SessionLock(ctx, conn); err != nil {
		return stageError(FailureLock, err)
	}
	return nil
}

func (l migrationSessionLocker) SessionUnlock(ctx context.Context, conn *sql.Conn) error {
	if err := l.SessionLocker.SessionUnlock(ctx, conn); err != nil {
		return stageError(FailureCleanup, err)
	}
	return nil
}

func migrateEmptySource(
	parent context.Context,
	executionCtx context.Context,
	opts MigrationOptions,
	db *sql.DB,
	locker gooselock.SessionLocker,
	direction migrationDirection,
	result RunResult,
) (RunResult, error) {
	lockConn, err := acquireMigrationLock(executionCtx, db, locker, opts.LockTimeout)
	if err != nil {
		_ = db.Close()
		return result, err
	}

	cleanup := func() error {
		return cleanupMigrationResources(parent, opts.CleanupTimeout, locker, lockConn, db)
	}
	store, err := database.NewStore(database.DialectPostgres, goose.DefaultTablename)
	if err != nil {
		return result, stageError(FailureConfig, errors.Join(
			fmt.Errorf("build goose postgres store: %w", err),
			cleanup(),
		))
	}
	version, err := emptySourceVersion(executionCtx, lockConn, store)
	result.Before = version
	result.After = version
	if err != nil {
		return result, stageError(FailureState, errors.Join(err, cleanup()))
	}
	logMigrationPlan(executionCtx, opts.Logger, direction, version, 0, 0)
	if cleanupErr := cleanup(); cleanupErr != nil {
		return result, stageError(FailureCleanup, cleanupErr)
	}
	return result, nil
}

func acquireMigrationLock(
	ctx context.Context,
	db *sql.DB,
	locker gooselock.SessionLocker,
	timeout time.Duration,
) (*sql.Conn, error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, stageError(
			FailureConnect,
			fmt.Errorf("acquire postgres migration lock connection: %w", err),
		)
	}

	lockCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if err := locker.SessionLock(lockCtx, conn); err != nil {
		return nil, stageError(FailureLock, errors.Join(
			fmt.Errorf("acquire goose session lock: %w", err),
			conn.Close(),
		))
	}
	return conn, nil
}

func validateMigrationOptions(opts MigrationOptions) error {
	if opts.ConnectTimeout <= 0 {
		return errors.New("postgres migration connect timeout must be > 0")
	}
	if opts.StatementTimeout <= 0 {
		return errors.New("postgres migration statement timeout must be > 0")
	}
	if opts.LockTimeout <= 0 {
		return errors.New("postgres migration lock timeout must be > 0")
	}
	if opts.CleanupTimeout <= 0 {
		return errors.New("postgres migration cleanup timeout must be > 0")
	}
	return nil
}

func cleanupMigrationResources(
	parent context.Context,
	budget time.Duration,
	locker gooselock.SessionLocker,
	conn *sql.Conn,
	db *sql.DB,
) error {
	cleanupCtx, cancel := detachedCleanupContext(parent, budget)
	defer cancel()

	var cleanupErr error
	if conn != nil {
		if err := locker.SessionUnlock(cleanupCtx, conn); err != nil {
			cleanupErr = errors.Join(cleanupErr, fmt.Errorf("release goose session lock: %w", err))
		}
	}
	if conn != nil {
		if err := conn.Close(); err != nil {
			cleanupErr = errors.Join(cleanupErr, fmt.Errorf("close postgres migration lock connection: %w", err))
		}
	}
	if db != nil {
		if err := db.Close(); err != nil {
			cleanupErr = errors.Join(cleanupErr, fmt.Errorf("close postgres migration database: %w", err))
		}
	}
	return cleanupErr
}

func logMigrationPlan(
	ctx context.Context,
	logger *slog.Logger,
	direction migrationDirection,
	current int64,
	target int64,
	sourceCount int,
) {
	if logger == nil {
		return
	}
	logger.InfoContext(
		ctx,
		"migration_plan",
		"migration.direction", string(direction),
		"migration.before", current,
		"migration.target", target,
		"migration.source_count", sourceCount,
	)
}

func logMigrationResults(
	ctx context.Context,
	logger *slog.Logger,
	applied []*goose.MigrationResult,
	failed *goose.MigrationResult,
) {
	if logger == nil {
		return
	}
	for _, migration := range applied {
		logMigrationResult(ctx, logger, migration, false)
	}
	if failed != nil {
		logMigrationResult(ctx, logger, failed, true)
	}
}

func logMigrationResult(ctx context.Context, logger *slog.Logger, migration *goose.MigrationResult, failed bool) {
	if migration == nil || migration.Source == nil {
		return
	}
	outcome := "success"
	if failed {
		outcome = "error"
	}
	attrs := []any{
		"migration.version", migration.Source.Version,
		"migration.filename", filepath.Base(migration.Source.Path),
		"migration.direction", migration.Direction,
		"migration.duration", migration.Duration,
		"migration.empty", migration.Empty,
		"outcome", outcome,
	}
	if failed {
		logger.ErrorContext(ctx, "migration_result", attrs...)
		return
	}
	logger.InfoContext(ctx, "migration_result", attrs...)
}
