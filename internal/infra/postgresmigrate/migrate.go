package postgresmigrate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	pathpkg "path"
	"strings"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgres"
	migrate "github.com/golang-migrate/migrate/v4"
	pgxmigrate "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib" // registers the pgx/v5 database/sql driver
)

type MigrationOptions struct {
	DSN              string
	SourceFS         fs.FS
	SourcePath       string
	StatementTimeout time.Duration
	LockTimeout      time.Duration
}

type migrationExecutor interface {
	Up() error
	Down() error
	Close() (error, error)
}

// MigrateUp applies pending migrations and reports whether anything changed.
func MigrateUp(ctx context.Context, opts MigrationOptions) (bool, error) {
	changed := false
	err := withMigrationRunner(ctx, opts, func(runner migrationExecutor) error {
		applied, upErr := applyMigrations(ctx, runner)
		changed = applied
		return upErr
	})
	if err != nil {
		return false, err
	}
	return changed, nil
}

// MigrateDown rolls every applied migration back.
//
// It destroys the schema and everything in it, and is deliberately not reachable
// from the migrate binary. The version this replaced exposed it as a `validate`
// subcommand that applied, rolled back, and reapplied — shipped in the production
// image, next to the entrypoint that applies migrations, reading the same
// APP__POSTGRES__DSN. It needed an env-var interlock to be safe there, and a
// capability that needs an interlock to be safe in the place it is installed does
// not belong in that place.
//
// Proving that down migrations work is CI's job against a throwaway database, which
// is what test/postgres_migrate_runner_integration_test.go does with this.
func MigrateDown(ctx context.Context, opts MigrationOptions) error {
	return withMigrationRunner(ctx, opts, func(runner migrationExecutor) error {
		downErr := runner.Down()
		if contextErr := ctx.Err(); contextErr != nil {
			return joinMigrationContextError("roll back all postgres migrations", downErr, contextErr)
		}
		if downErr != nil {
			return migrationOperationError("roll back all postgres migrations", downErr)
		}
		return nil
	})
}

// withMigrationRunner opens the source and database drivers, hands the runner to
// operate, and closes both whatever operate did.
func withMigrationRunner(ctx context.Context, opts MigrationOptions, operate func(migrationExecutor) error) error {
	normalizedSourcePath, err := normalizeMigrationSourcePath(opts.SourcePath)
	if err != nil {
		return err
	}
	if err := validateMigrationTimeouts(opts); err != nil {
		return err
	}

	normalizedDSN, err := postgres.NormalizeDSN(opts.DSN)
	if err != nil {
		return fmt.Errorf("validate postgres migration dsn: %w", err)
	}

	sourceFS := opts.SourceFS
	if sourceFS == nil {
		sourceFS = os.DirFS("/")
	}

	sourceDriver, err := iofs.New(sourceFS, normalizedSourcePath)
	if err != nil {
		return fmt.Errorf("open migration source %q: %w", opts.SourcePath, err)
	}

	databaseHandle, err := sql.Open("pgx/v5", normalizedDSN)
	if err != nil {
		sourceCloseErr := sourceDriver.Close()
		if sourceCloseErr != nil {
			return errors.Join(
				fmt.Errorf("open postgres migration database: %w", err),
				fmt.Errorf("close migration source: %w", sourceCloseErr),
			)
		}
		return fmt.Errorf("open postgres migration database: %w", err)
	}

	// The migration driver owns databaseHandle only after WithInstance
	// succeeds; every failure path here must close it directly.
	databaseDriver, err := pgxmigrate.WithInstance(databaseHandle, &pgxmigrate.Config{
		StatementTimeout: opts.StatementTimeout,
	})
	if err != nil {
		closeErr := closeMigrationResources(sourceDriver, databaseHandle)
		if closeErr != nil {
			return errors.Join(
				fmt.Errorf("open postgres migration driver: %w", err),
				closeErr,
			)
		}
		return fmt.Errorf("open postgres migration driver: %w", err)
	}

	runner, err := migrate.NewWithInstance("iofs", sourceDriver, "pgx", databaseDriver)
	if err != nil {
		closeErr := closeMigrationResources(sourceDriver, databaseDriver)
		if closeErr != nil {
			return errors.Join(
				fmt.Errorf("build migration runner: %w", err),
				closeErr,
			)
		}
		return fmt.Errorf("build migration runner: %w", err)
	}
	runner.LockTimeout = opts.LockTimeout
	stopSignals := make(chan bool, 1)
	runner.GracefulStop = stopSignals
	stopWatcher := context.AfterFunc(ctx, func() { stopSignals <- true })
	defer stopWatcher()

	return runMigrationOperation(ctx, runner, operate)
}

func runMigrationOperation(
	ctx context.Context,
	runner migrationExecutor,
	operate func(migrationExecutor) error,
) (err error) {
	defer func() {
		sourceErr, databaseErr := runner.Close()
		if closeErr := errors.Join(sourceErr, databaseErr); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close migration runner: %w", closeErr))
		}
	}()

	if ctxErr := ctx.Err(); ctxErr != nil {
		return fmt.Errorf("run postgres migrations canceled: %w", ctxErr)
	}
	return operate(runner)
}

// applyMigrations runs Up and reports whether the schema moved. ErrNoChange is not
// a failure: an already-current schema is the ordinary outcome of every deploy after
// the one that introduced a migration.
func applyMigrations(ctx context.Context, runner migrationExecutor) (bool, error) {
	upErr := runner.Up()
	if contextErr := ctx.Err(); contextErr != nil {
		return false, joinMigrationContextError("run postgres migrations", upErr, contextErr)
	}
	if upErr != nil {
		if errors.Is(upErr, migrate.ErrNoChange) {
			return false, nil
		}
		return false, migrationOperationError("run postgres migrations", upErr)
	}
	return true, nil
}

func validateMigrationTimeouts(opts MigrationOptions) error {
	if opts.StatementTimeout <= 0 {
		return fmt.Errorf("postgres migration statement timeout must be > 0")
	}
	if opts.LockTimeout <= 0 {
		return fmt.Errorf("postgres migration lock timeout must be > 0")
	}
	return nil
}

func joinMigrationContextError(operation string, operationErr, contextErr error) error {
	canceled := fmt.Errorf("%s canceled: %w", operation, contextErr)
	if operationErr == nil {
		return canceled
	}
	return errors.Join(migrationOperationError(operation, operationErr), canceled)
}

func migrationOperationError(operation string, err error) error {
	if dirty, ok := errors.AsType[migrate.ErrDirty](err); ok {
		return fmt.Errorf(
			"%s: postgres migration state is dirty at version %d; automatic force is disabled; repair and verify the schema, then follow docs/railway-deployment-profile.md: %w",
			operation,
			dirty.Version,
			err,
		)
	}
	return fmt.Errorf("%s: %w", operation, err)
}

func normalizeMigrationSourcePath(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("migration source path is empty")
	}

	normalized := pathpkg.Clean("/" + trimmed)
	normalized = strings.TrimPrefix(normalized, "/")
	if normalized == "." || normalized == "" {
		return "", fmt.Errorf("migration source path is empty")
	}
	if !fs.ValidPath(normalized) {
		return "", fmt.Errorf("migration source path %q is invalid", raw)
	}

	return normalized, nil
}

func closeMigrationResources(sourceCloser interface{ Close() error }, databaseCloser interface{ Close() error }) error {
	var sourceErr error
	if sourceCloser != nil {
		sourceErr = sourceCloser.Close()
	}

	var databaseErr error
	if databaseCloser != nil {
		databaseErr = databaseCloser.Close()
	}

	return errors.Join(sourceErr, databaseErr)
}
