package main

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	pathpkg "path"
	"syscall"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/postgres"
)

const imageMigrationSourcePath = "/env/migrations"

func main() {
	if err := run(os.Stdout); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run(stdout *os.File) error {
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

	migrationSourceFS, migrationSourcePath := resolveMigrationSource()
	result, err := postgres.MigrateUp(ctx, postgres.MigrationOptions{
		DSN:        cfg.Postgres.DSN,
		SourceFS:   migrationSourceFS,
		SourcePath: migrationSourcePath,
	})
	if err != nil {
		return fmt.Errorf("apply postgres migrations: %w", err)
	}

	if result.Changed {
		_, _ = fmt.Fprintf(stdout, "applied migrations from %s\n", migrationSourcePath)
		return nil
	}

	_, _ = fmt.Fprintln(stdout, "database schema is already up to date")
	return nil
}

func resolveMigrationSource() (fs.FS, string) {
	if info, err := os.Stat(imageMigrationSourcePath); err == nil && info.IsDir() {
		return nil, imageMigrationSourcePath
	}

	const localMigrationSourcePath = "env/migrations"
	if info, err := os.Stat(localMigrationSourcePath); err == nil && info.IsDir() {
		return os.DirFS("."), pathpkg.Clean(localMigrationSourcePath)
	}

	return nil, imageMigrationSourcePath
}
