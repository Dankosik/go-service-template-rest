package main

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/signal"
	"path"
	"syscall"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/postgres"
)

const (
	imageMigrationSourcePath = "/env/migrations"
	localMigrationSourcePath = "env/migrations"
)

func run(stdout io.Writer) error {
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
	return resolveMigrationSourceFrom(imageMigrationSourcePath, localMigrationSourcePath)
}

func resolveMigrationSourceFrom(imagePath string, localPath string) (fs.FS, string) {
	if info, err := os.Stat(imagePath); err == nil && info.IsDir() {
		return nil, imagePath
	}

	if info, err := os.Stat(localPath); err == nil && info.IsDir() {
		return os.DirFS("."), path.Clean(localPath)
	}

	return nil, imagePath
}
