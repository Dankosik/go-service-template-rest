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

	// rehearsalAcknowledgementEnv gates the validate subcommand.
	//
	// validate applies every migration, rolls all of them back, and applies them
	// again. That is the point on a throwaway database and a schema drop
	// anywhere else — and this binary ships in the production image next to the
	// entrypoint that applies migrations, reading the same APP__POSTGRES__DSN.
	// One mistyped argument is the whole difference, so the destructive path
	// requires a variable no production environment sets.
	rehearsalAcknowledgementEnv   = "MIGRATION_REHEARSAL"
	rehearsalAcknowledgementValue = "1"
)

func run(args []string, stdout io.Writer) error {
	validate, err := parseMigrationCommand(args)
	if err != nil {
		return err
	}
	if validate {
		if err := requireRehearsalAcknowledgement(); err != nil {
			return err
		}
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
		_, _ = fmt.Fprintln(stdout, "no migration files found; skipping migrations")
		return nil
	}
	migrationCtx, cancelMigration := context.WithTimeout(ctx, cfg.Postgres.MigrationTimeout)
	defer cancelMigration()
	options := postgresmigrate.MigrationOptions{
		DSN:              cfg.Postgres.DSN,
		SourceFS:         migrationSourceFS,
		SourcePath:       migrationSourcePath,
		StatementTimeout: cfg.Postgres.MigrationStatementTimeout,
		LockTimeout:      cfg.Postgres.MigrationLockTimeout,
	}
	if validate {
		if err := postgresmigrate.ValidateMigrations(migrationCtx, options); err != nil {
			return fmt.Errorf("validate postgres migrations: %w", err)
		}
		_, _ = fmt.Fprintf(stdout, "validated migrations from %s\n", migrationSourcePath)
		return nil
	}

	changed, err := postgresmigrate.MigrateUp(migrationCtx, options)
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

// requireRehearsalAcknowledgement refuses the rehearsal unless the caller said
// out loud that the target database is disposable. The message names the
// consequence, because "validate" reads as a read-only check and is not one.
func requireRehearsalAcknowledgement() error {
	if os.Getenv(rehearsalAcknowledgementEnv) == rehearsalAcknowledgementValue {
		return nil
	}
	return fmt.Errorf(
		"migrate validate rehearses down-migrations and destroys all data in the target database; "+
			"set %s=%s to confirm the target is disposable",
		rehearsalAcknowledgementEnv,
		rehearsalAcknowledgementValue,
	)
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
