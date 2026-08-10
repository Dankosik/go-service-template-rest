package postgresmigrate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
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

	source, err := admitSource(opts.SourceFS, opts.SourcePath)
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

	var (
		lockConn *sql.Conn
		locked   bool
	)
	defer func() {
		cleanupErr := cleanupMigrationResources(ctx, opts.CleanupTimeout, locker, lockConn, locked, db)
		if cleanupErr != nil {
			retErr = stageError(FailureCleanup, errors.Join(retErr, cleanupErr))
		}
	}()

	if err := db.PingContext(executionCtx); err != nil {
		return result, stageError(FailureConnect, fmt.Errorf("ping postgres migration database: %w", err))
	}
	lockConn, err = acquireMigrationLock(executionCtx, db, locker, opts.LockTimeout)
	if err != nil {
		return result, err
	}
	locked = true

	store, err := database.NewStore(database.DialectPostgres, goose.DefaultTablename)
	if err != nil {
		return result, stageError(FailureConfig, fmt.Errorf("build goose postgres store: %w", err))
	}
	before, err := loadMigrationState(executionCtx, lockConn, store, source.Migrations)
	if err != nil {
		return result, stageError(FailureState, err)
	}
	result.Before = before.Current
	result.After = before.Current
	result.Target = before.Target
	if direction == directionDown {
		result.Target = 0
		before.Target = 0
	}
	logMigrationPlan(executionCtx, opts.Logger, direction, before, len(source.Migrations))

	if len(source.Migrations) == 0 {
		return result, nil
	}

	provider, err := goose.NewProvider(
		goose.DialectPostgres,
		db,
		source.FS,
		goose.WithDisableGlobalRegistry(true),
		goose.WithLogger(goose.NopLogger()),
		goose.WithVerbose(false),
	)
	if err != nil {
		return result, stageError(FailureSource, fmt.Errorf("build goose provider: %w", err))
	}

	var gooseResults []*goose.MigrationResult
	switch direction {
	case directionUp:
		gooseResults, err = provider.Up(executionCtx)
	case directionDown:
		gooseResults, err = provider.DownTo(executionCtx, 0)
	default:
		return result, stageError(FailureConfig, fmt.Errorf("unsupported migration direction %q", direction))
	}
	result.Migrations = migrationResultsFromGoose(gooseResults)

	if err != nil {
		if partial, ok := errors.AsType[*goose.PartialError](err); ok {
			result.Migrations = migrationResultsFromGoose(partial.Applied)
			failed := migrationResultFromGoose(partial.Failed)
			result.Failed = &failed
		}
		setPartialAfter(&result, direction, source.Migrations)
		logMigrationResults(executionCtx, opts.Logger, result)
		return result, stageError(FailureExecute, err)
	}

	setPartialAfter(&result, direction, source.Migrations)
	after, stateErr := loadMigrationState(executionCtx, lockConn, store, source.Migrations)
	if stateErr != nil {
		logMigrationResults(executionCtx, opts.Logger, result)
		return result, stageError(FailureState, stateErr)
	}
	result.After = after.Current
	logMigrationResults(executionCtx, opts.Logger, result)
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
		return conn, stageError(FailureLock, fmt.Errorf("acquire goose session lock: %w", err))
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
	locked bool,
	db *sql.DB,
) error {
	cleanupCtx, cancel := detachedCleanupContext(parent, budget)
	defer cancel()

	var cleanupErr error
	if locked && conn != nil {
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
	state migrationState,
	sourceCount int,
) {
	if logger == nil {
		return
	}
	logger.InfoContext(
		ctx,
		"migration_plan",
		"migration.direction", string(direction),
		"migration.before", state.Current,
		"migration.target", state.Target,
		"migration.source_count", sourceCount,
		"migration.applied_count", len(state.Applied),
	)
}

func logMigrationResults(ctx context.Context, logger *slog.Logger, result RunResult) {
	if logger == nil {
		return
	}
	for _, migration := range result.Migrations {
		logger.InfoContext(
			ctx,
			"migration_result",
			"migration.version", migration.Version,
			"migration.filename", migration.Filename,
			"migration.direction", migration.Direction,
			"migration.duration", migration.Duration,
			"migration.empty", migration.Empty,
			"outcome", "success",
		)
	}
	if result.Failed != nil {
		logger.ErrorContext(
			ctx,
			"migration_result",
			"migration.version", result.Failed.Version,
			"migration.filename", result.Failed.Filename,
			"migration.direction", result.Failed.Direction,
			"migration.duration", result.Failed.Duration,
			"migration.empty", result.Failed.Empty,
			"outcome", "error",
		)
	}
}
