package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/postgresmigrate"
)

const (
	imageMigrationSourcePath = "/migrations"
	localMigrationSourcePath = "migrations"

	// migrationPathEnv overrides where migrations are read from. A path named here
	// must exist: the operator said where they are.
	migrationPathEnv = "MIGRATION_PATH"
)

// errMigrationSourceMissing reports that no migration directory exists where one
// was expected.
//
// The distinction it carries is the whole point. A directory that exists and holds
// no *.sql is a service that has not written its first migration yet, which is a
// normal state. A directory that does not exist at all, in an image whose build
// always creates it, means the migrations were not packaged — and the version this
// replaced reported both as "no migration files found; skipping migrations" and
// exited 0. So a Dockerfile edit, a wrong MIGRATION_PATH, or a pre-deploy hook
// running from the wrong directory produced a green migration step, a service that
// started, a readiness probe that passed, and 500s at the first query against a
// schema that was never created.
var errMigrationSourceMissing = errors.New("migration source directory does not exist")

func run(args []string, stdout io.Writer) error {
	if len(args) > 0 {
		// There is deliberately no subcommand. This binary applies migrations and
		// nothing else; see postgresmigrate.MigrateDown for where the rollback
		// rehearsal went and why it is not reachable from here.
		return fmt.Errorf("usage: migrate (no arguments)")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, _, err := config.LoadDetailedWithContext(ctx, config.LoadOptions{})
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if !cfg.Postgres.Enabled {
		return errors.New("postgres is required by the DATABASE=postgres profile")
	}

	migrationSourceFS, migrationSourcePath, found, err := resolveMigrationSource()
	if err != nil {
		return fmt.Errorf("resolve migration source: %w", err)
	}
	if !found {
		_, _ = fmt.Fprintf(stdout, "migration directory %s is empty; no migrations to apply\n", migrationSourcePath)
		return nil
	}
	migrationCtx, cancelMigration := context.WithTimeout(ctx, cfg.Postgres.MigrationTimeout)
	defer cancelMigration()

	changed, err := postgresmigrate.MigrateUp(migrationCtx, postgresmigrate.MigrationOptions{
		DSN:              cfg.Postgres.DSN,
		SourceFS:         migrationSourceFS,
		SourcePath:       migrationSourcePath,
		StatementTimeout: cfg.Postgres.MigrationStatementTimeout,
		LockTimeout:      cfg.Postgres.MigrationLockTimeout,
	})
	if err != nil {
		return fmt.Errorf("apply postgres migrations: %w", err)
	}

	if changed {
		_, _ = fmt.Fprintf(stdout, "applied migrations from %s\n", migrationSourcePath)
		return nil
	}

	_, _ = fmt.Fprintln(stdout, "database schema is already up to date")
	return nil
}

// resolveMigrationSource reports where migrations are, and whether there are any.
//
// found is false only for a directory that exists and holds no migration files. An
// absent directory is an error; see errMigrationSourceMissing.
func resolveMigrationSource() (fs.FS, string, bool, error) {
	if configuredPath := strings.TrimSpace(os.Getenv(migrationPathEnv)); configuredPath != "" {
		cleaned := path.Clean(configuredPath)
		found, err := hasMigrationFiles(cleaned)
		if err != nil {
			return nil, "", false, err
		}
		if path.IsAbs(cleaned) {
			return nil, cleaned, found, nil
		}
		return os.DirFS("."), cleaned, found, nil
	}
	return resolveMigrationSourceFrom(imageMigrationSourcePath)
}

// resolveMigrationSourceFrom prefers the image path and falls back to the working
// directory, which is how one binary serves a container and a developer shell. Only
// when neither exists is it the packaging failure.
func resolveMigrationSourceFrom(imagePath string) (fs.FS, string, bool, error) {
	imageFound, imageErr := hasMigrationFiles(imagePath)
	if imageErr == nil {
		return nil, imagePath, imageFound, nil
	}
	if !errors.Is(imageErr, errMigrationSourceMissing) {
		return nil, "", false, imageErr
	}

	localFound, localErr := hasMigrationFiles(localMigrationSourcePath)
	if localErr == nil {
		return os.DirFS("."), localMigrationSourcePath, localFound, nil
	}
	if !errors.Is(localErr, errMigrationSourceMissing) {
		return nil, "", false, localErr
	}

	return nil, "", false, fmt.Errorf(
		"%w: looked in %q and %q",
		errMigrationSourceMissing,
		imagePath,
		localMigrationSourcePath,
	)
}

// hasMigrationFiles reports whether directory holds any migration file, and returns
// errMigrationSourceMissing when the directory itself is absent.
func hasMigrationFiles(directory string) (bool, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, fmt.Errorf("%w: %q", errMigrationSourceMissing, directory)
		}
		return false, fmt.Errorf("read migration directory %q: %w", directory, err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".up.sql") || strings.HasSuffix(name, ".down.sql") {
			return true, nil
		}
	}
	return false, nil
}
