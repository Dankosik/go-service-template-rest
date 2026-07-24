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

	"github.com/example/go-service-template-rest/internal/infra/postgres"
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
	Down() error
	Close() (error, error)
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

	normalizedDSN, err := postgres.NormalizeDSN(opts.DSN)
	if err != nil {
		return false, fmt.Errorf("validate postgres migration dsn: %w", err)
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
	stopSignals := make(chan bool, 1)
	runner.GracefulStop = stopSignals
	stopWatcher := context.AfterFunc(ctx, func() { stopSignals <- true })
	defer stopWatcher()

	return executeMigrations(runner, rehearse)
}

func executeMigrations(runner migrationExecutor, rehearse bool) (changed bool, err error) {
	defer func() {
		sourceErr, databaseErr := runner.Close()
		if closeErr := errors.Join(sourceErr, databaseErr); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close migration runner: %w", closeErr))
		}
	}()

	changed = true
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
	if err := runner.Down(); err != nil {
		return false, fmt.Errorf("roll back all postgres migrations: %w", err)
	}
	if err := runner.Up(); err != nil {
		return false, fmt.Errorf("reapply all postgres migrations: %w", err)
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
