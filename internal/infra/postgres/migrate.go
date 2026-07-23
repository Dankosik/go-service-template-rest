package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	pathpkg "path"
	"strings"

	migrate "github.com/golang-migrate/migrate/v4"
	pgxmigrate "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib" // registers the pgx/v5 database/sql driver
)

type MigrationOptions struct {
	DSN        string
	SourceFS   fs.FS
	SourcePath string
}

type migrationExecutor interface {
	Up() error
	Steps(int) error
}

// MigrateUp applies pending migrations and reports whether anything changed.
func MigrateUp(ctx context.Context, opts MigrationOptions) (bool, error) {
	return runPostgresMigrations(ctx, opts, false)
}

func ValidateMigrations(ctx context.Context, opts MigrationOptions) error {
	_, err := runPostgresMigrations(ctx, opts, true)
	return err
}

func runPostgresMigrations(ctx context.Context, opts MigrationOptions, rehearse bool) (bool, error) {
	normalizedSourcePath, err := normalizeMigrationSourcePath(opts.SourcePath)
	if err != nil {
		return false, err
	}

	normalizedDSN, err := preflightPostgresDSN(opts.DSN)
	if err != nil {
		return false, err
	}

	sourceFS := opts.SourceFS
	if sourceFS == nil {
		sourceFS = os.DirFS("/")
	}

	sourceDriver, err := iofs.New(sourceFS, normalizedSourcePath)
	if err != nil {
		return false, fmt.Errorf("open migration source %q: %w", opts.SourcePath, err)
	}

	databaseHandle, err := sql.Open("pgx/v5", normalizedDSN)
	if err != nil {
		sourceCloseErr := sourceDriver.Close()
		if sourceCloseErr != nil {
			return false, errors.Join(
				fmt.Errorf("open postgres migration database: %w", err),
				fmt.Errorf("close migration source: %w", sourceCloseErr),
			)
		}
		return false, fmt.Errorf("open postgres migration database: %w", err)
	}

	// The migration driver owns databaseHandle only after WithInstance
	// succeeds; every failure path here must close it directly.
	databaseDriver, err := pgxmigrate.WithInstance(databaseHandle, &pgxmigrate.Config{})
	if err != nil {
		closeErr := closeMigrationResources(sourceDriver, databaseHandle)
		if closeErr != nil {
			return false, errors.Join(
				fmt.Errorf("open postgres migration driver: %w", err),
				closeErr,
			)
		}
		return false, fmt.Errorf("open postgres migration driver: %w", err)
	}

	runner, err := migrate.NewWithInstance("iofs", sourceDriver, "pgx", databaseDriver)
	if err != nil {
		closeErr := closeMigrationResources(sourceDriver, databaseDriver)
		if closeErr != nil {
			return false, errors.Join(
				fmt.Errorf("build migration runner: %w", err),
				closeErr,
			)
		}
		return false, fmt.Errorf("build migration runner: %w", err)
	}
	defer func() {
		_ = closeMigrationRunner(runner)
	}()

	stopSignals := make(chan bool, 1)
	runner.GracefulStop = stopSignals
	stopWatcher := context.AfterFunc(ctx, func() { stopSignals <- true })
	defer stopWatcher()

	return executeMigrations(runner, rehearse)
}

func executeMigrations(runner migrationExecutor, rehearse bool) (bool, error) {
	changed := true
	if err := runner.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			changed = false
		} else {
			return false, fmt.Errorf("run postgres migrations: %w", err)
		}
	}

	if !rehearse {
		return changed, nil
	}
	if err := runner.Steps(-1); err != nil {
		return false, fmt.Errorf("roll back latest postgres migration: %w", err)
	}
	if err := runner.Steps(1); err != nil {
		return false, fmt.Errorf("reapply latest postgres migration: %w", err)
	}

	return changed, nil
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

func closeMigrationRunner(runner *migrate.Migrate) error {
	if runner == nil {
		return nil
	}

	sourceErr, databaseErr := runner.Close()
	return errors.Join(sourceErr, databaseErr)
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
