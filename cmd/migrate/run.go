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
	"github.com/example/go-service-template-rest/internal/infra/postgres"
)

const (
	imageMigrationSourcePath = "/migrations"
	localMigrationSourcePath = "migrations"
)

func run(args []string, stdout io.Writer) error {
	validate, err := parseMigrationCommand(args)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if !cfg.Postgres.Enabled {
		_, _ = fmt.Fprintln(stdout, "postgres is disabled; skipping migrations")
		return nil
	}

	migrationSourceFS, migrationSourcePath, found, err := resolveMigrationSource()
	if err != nil {
		return fmt.Errorf("resolve migration source: %w", err)
	}
	if !found {
		_, _ = fmt.Fprintln(stdout, "no migration files found; skipping migrations")
		return nil
	}
	if validate {
		if err := postgres.ValidateMigrations(ctx, postgres.MigrationOptions{
			DSN:        cfg.Postgres.DSN,
			SourceFS:   migrationSourceFS,
			SourcePath: migrationSourcePath,
		}); err != nil {
			return fmt.Errorf("validate postgres migrations: %w", err)
		}
		_, _ = fmt.Fprintf(stdout, "validated migrations from %s\n", migrationSourcePath)
		return nil
	}

	changed, err := postgres.MigrateUp(ctx, postgres.MigrationOptions{
		DSN:        cfg.Postgres.DSN,
		SourceFS:   migrationSourceFS,
		SourcePath: migrationSourcePath,
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

func resolveMigrationSource() (fs.FS, string, bool, error) {
	if configuredPath := strings.TrimSpace(os.Getenv("MIGRATION_PATH")); configuredPath != "" {
		if path.IsAbs(configuredPath) {
			return nil, path.Clean(configuredPath), true, nil
		}
		return os.DirFS("."), path.Clean(configuredPath), true, nil
	}
	return resolveMigrationSourceFrom(imageMigrationSourcePath)
}

func parseMigrationCommand(args []string) (bool, error) {
	switch len(args) {
	case 0:
		return false, nil
	case 1:
		if args[0] == "validate" {
			return true, nil
		}
	}
	return false, fmt.Errorf("usage: migrate [validate]")
}

func resolveMigrationSourceFrom(imagePath string) (fs.FS, string, bool, error) {
	found, err := hasMigrationFiles(imagePath)
	if err != nil {
		return nil, "", false, fmt.Errorf("inspect image migration source %q: %w", imagePath, err)
	}
	if found {
		return nil, imagePath, true, nil
	}

	found, err = hasMigrationFiles(localMigrationSourcePath)
	if err != nil {
		return nil, "", false, fmt.Errorf("inspect local migration source %q: %w", localMigrationSourcePath, err)
	}
	if found {
		return os.DirFS("."), localMigrationSourcePath, true, nil
	}

	return nil, "", false, nil
}

func hasMigrationFiles(directory string) (bool, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
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
