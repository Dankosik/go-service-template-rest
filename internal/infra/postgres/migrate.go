package postgres

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	pathpkg "path"
	"strings"

	migrate "github.com/golang-migrate/migrate/v4"
	pgxmigrate "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

type MigrationOptions struct {
	DSN        string
	SourceFS   fs.FS
	SourcePath string
}

type MigrationResult struct {
	Changed bool
}

func MigrateUp(ctx context.Context, opts MigrationOptions) (MigrationResult, error) {
	normalizedSourcePath, err := normalizeMigrationSourcePath(opts.SourcePath)
	if err != nil {
		return MigrationResult{}, err
	}

	normalizedDSN, err := preflightPostgresDSN(opts.DSN)
	if err != nil {
		return MigrationResult{}, err
	}

	sourceFS := opts.SourceFS
	if sourceFS == nil {
		sourceFS = os.DirFS("/")
	}

	sourceDriver, err := iofs.New(sourceFS, normalizedSourcePath)
	if err != nil {
		return MigrationResult{}, fmt.Errorf("open migration source %q: %w", opts.SourcePath, err)
	}

	databaseDriver, err := (&pgxmigrate.Postgres{}).Open(normalizedDSN)
	if err != nil {
		sourceCloseErr := sourceDriver.Close()
		if sourceCloseErr != nil {
			return MigrationResult{}, errors.Join(
				fmt.Errorf("open postgres migration driver: %w", err),
				fmt.Errorf("close migration source: %w", sourceCloseErr),
			)
		}
		return MigrationResult{}, fmt.Errorf("open postgres migration driver: %w", err)
	}

	runner, err := migrate.NewWithInstance("iofs", sourceDriver, "pgx", databaseDriver)
	if err != nil {
		closeErr := closeMigrationResources(sourceDriver, databaseDriver)
		if closeErr != nil {
			return MigrationResult{}, errors.Join(
				fmt.Errorf("build migration runner: %w", err),
				closeErr,
			)
		}
		return MigrationResult{}, fmt.Errorf("build migration runner: %w", err)
	}
	defer func() {
		_ = closeMigrationRunner(runner)
	}()

	stopSignals := make(chan bool, 1)
	stopWatcherStop := make(chan struct{})
	runner.GracefulStop = stopSignals
	go func() {
		select {
		case <-ctx.Done():
			stopSignals <- true
		case <-stopWatcherStop:
		}
	}()
	defer close(stopWatcherStop)

	if err := runner.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			return MigrationResult{Changed: false}, nil
		}
		return MigrationResult{}, fmt.Errorf("run postgres migrations: %w", err)
	}

	return MigrationResult{Changed: true}, nil
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
