package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"

	"github.com/example/go-service-template-rest/cmd/internal/runtimeopts"
	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/postgresmigrate"
	"github.com/example/go-service-template-rest/internal/observability/logctx"
)

const (
	imageMigrationSourcePath = "/migrations"
	localMigrationSourcePath = "migrations"
	migrationPathEnv         = "MIGRATION_PATH"
	recoveryDocument         = "docs/railway-deployment-profile.md"
)

var errMigrationSourceMissing = errors.New("migration source directory does not exist")

func run(args []string, stdout io.Writer) error {
	// Configuration is not loaded yet, so this carries no service identity; the
	// records before the load below are the only ones that cannot.
	logger := logctx.NewProcessLogger(stdout, slog.LevelInfo)
	if len(args) > 0 {
		err := errors.New("usage: migrate (no arguments)")
		logMigrationTerminal(logger, postgresmigrate.RunResult{}, err, postgresmigrate.FailureConfig)
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, _, err := config.LoadDetailedWithContext(ctx, config.LoadOptions{})
	if err != nil {
		wrapped := fmt.Errorf("load config: %w", err)
		logMigrationTerminal(logger, postgresmigrate.RunResult{}, wrapped, postgresmigrate.FailureConfig)
		return wrapped
	}
	// Rebuilt now that the identity is known, so a migration run is attributable
	// to the same service, version, and environment as the three long-running
	// binaries rather than being the one job whose records carry none of them.
	logger = runtimeopts.Logger(stdout, cfg)
	if !cfg.Postgres.Enabled {
		err := errors.New("postgres is required by the DATABASE=postgres profile")
		logMigrationTerminal(logger, postgresmigrate.RunResult{}, err, postgresmigrate.FailureConfig)
		return err
	}

	migrationSourceFS, migrationSourcePath, err := resolveMigrationSource()
	if err != nil {
		wrapped := fmt.Errorf("resolve migration source: %w", err)
		logMigrationTerminal(logger, postgresmigrate.RunResult{}, wrapped, postgresmigrate.FailureSource)
		return wrapped
	}

	migrationCtx, cancelMigration := context.WithTimeout(ctx, cfg.Postgres.MigrationTimeout)
	defer cancelMigration()
	result, err := postgresmigrate.MigrateUp(migrationCtx, postgresmigrate.MigrationOptions{
		DSN:              cfg.Postgres.DSN,
		SourceFS:         migrationSourceFS,
		SourcePath:       migrationSourcePath,
		ConnectTimeout:   cfg.Postgres.ConnectTimeout,
		StatementTimeout: cfg.Postgres.MigrationStatementTimeout,
		LockTimeout:      cfg.Postgres.MigrationLockTimeout,
		CleanupTimeout:   cfg.Postgres.MigrationLockTimeout,
		Logger:           logger,
	})
	if err != nil {
		logMigrationTerminal(logger, result, err, postgresmigrate.FailureStageOf(err))
		return fmt.Errorf("apply postgres migrations: %w", err)
	}
	logMigrationTerminal(logger, result, nil, "")
	return nil
}

func resolveMigrationSource() (fs.FS, string, error) {
	if configuredPath := strings.TrimSpace(os.Getenv(migrationPathEnv)); configuredPath != "" {
		cleaned := path.Clean(configuredPath)
		if err := requireDirectory(cleaned); err != nil {
			return nil, "", err
		}
		if path.IsAbs(cleaned) {
			return os.DirFS("/"), strings.TrimPrefix(cleaned, "/"), nil
		}
		return os.DirFS("."), cleaned, nil
	}

	if err := requireDirectory(imageMigrationSourcePath); err == nil {
		return os.DirFS("/"), strings.TrimPrefix(imageMigrationSourcePath, "/"), nil
	} else if !errors.Is(err, errMigrationSourceMissing) {
		return nil, "", err
	}
	if err := requireDirectory(localMigrationSourcePath); err == nil {
		return os.DirFS("."), localMigrationSourcePath, nil
	} else if !errors.Is(err, errMigrationSourceMissing) {
		return nil, "", err
	}
	return nil, "", fmt.Errorf(
		"%w: looked in %q and %q",
		errMigrationSourceMissing,
		imageMigrationSourcePath,
		localMigrationSourcePath,
	)
}

func requireDirectory(directory string) error {
	info, err := os.Stat(directory)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("%w: %q", errMigrationSourceMissing, directory)
		}
		return fmt.Errorf("inspect migration directory %q: %w", directory, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("migration source %q is not a directory", directory)
	}
	return nil
}

func logMigrationTerminal(
	logger *slog.Logger,
	result postgresmigrate.RunResult,
	err error,
	stage postgresmigrate.FailureStage,
) {
	outcome := "success"
	if err != nil {
		outcome = "error"
	} else if len(result.Migrations) == 0 {
		outcome = "no_change"
	}
	attrs := []any{
		"migration.before", result.Before,
		"migration.target", result.Target,
		"migration.after", result.After,
		"migration.applied_count", len(result.Migrations),
		"migration.duration", result.Duration,
		"outcome", outcome,
	}
	if err != nil {
		attrs = append(
			attrs,
			"failure.stage", string(stage),
			"failure.context", contextFailureClass(err),
			"failure.sqlstate", postgresmigrate.SQLStateOf(err),
			"recovery.document", recoveryDocument,
		)
		logger.Error("migration_run_finished", attrs...)
		return
	}
	logger.Info("migration_run_finished", attrs...)
}

func contextFailureClass(err error) string {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return "deadline_exceeded"
	case errors.Is(err, context.Canceled):
		return "canceled"
	default:
		return "none"
	}
}
