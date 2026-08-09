// Package pgtest gives a test an isolated, migrated PostgreSQL database.
//
// The container is shared per test binary and the database is per test, so
// startup is paid once and parallel cases still cannot see each other's rows.
//
// Usage from any package that owns a repository:
//
//	func TestMain(m *testing.M) { os.Exit(pgtest.Main(m, "")) }
//
//	func TestOrderRepository(t *testing.T) {
//		dsn := pgtest.Migrated(t, os.DirFS("../../.."), "migrations")
//		// ...
//	}
//
// Docker is optional. Without it, DSN and Migrated skip the test, unless
// REQUIRE_DOCKER is set — which is what CI sets so a missing daemon fails the
// build instead of silently reporting a green suite that ran nothing.
package pgtest

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/postgresmigrate"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

// DefaultImage is the PostgreSQL image tests run against. It is pinned by digest
// so a suite cannot start passing or failing because a tag moved.
const DefaultImage = "postgres:17@sha256:a426e44bac0b759c95894d68e1a0ac03ecc20b619f498a91aae373bf06d8508d"

const (
	externalDSNEnv        = "PGTEST_POSTGRES_DSN"
	containerStartTimeout = 2 * time.Minute
	databaseOpTimeout     = time.Minute
	migrationTimeout      = time.Minute
	migrationLockTimeout  = 15 * time.Second
)

var (
	adminDSN    string
	databaseSeq atomic.Int64
)

// Main starts one PostgreSQL container for the whole test binary, runs m against
// it, and terminates it afterwards. An empty image uses DefaultImage.
//
// It returns an exit code rather than calling os.Exit, so the caller's TestMain
// stays the one place the process ends.
func Main(m *testing.M, image string) int {
	if externalDSN := strings.TrimSpace(os.Getenv(externalDSNEnv)); externalDSN != "" {
		adminDSN = externalDSN
		return m.Run()
	}
	if strings.TrimSpace(image) == "" {
		image = DefaultImage
	}

	ctx, cancel := context.WithTimeout(context.Background(), containerStartTimeout)
	defer cancel()

	if !requireDocker() && !dockerIsHealthy(ctx) {
		// Tests skip through DSN, so a developer without Docker still gets a
		// useful run of everything else in the package.
		return m.Run()
	}

	container, err := tcpostgres.Run(
		ctx,
		image,
		tcpostgres.WithDatabase("app"),
		tcpostgres.WithUsername("app"),
		tcpostgres.WithPassword("app"),
		tcpostgres.BasicWaitStrategies(),
		// initdb runs before the server starts, so the module's own `-c fsync=off`
		// never reaches it, and a default initdb spends most of container startup
		// syncing a cluster this binary throws away minutes later.
		testcontainers.WithEnv(map[string]string{"POSTGRES_INITDB_ARGS": "--no-sync"}),
	)
	if err != nil {
		terminate(container)
		slog.Error("start shared postgres container", "err", err)
		return 1
	}
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		terminate(container)
		slog.Error("resolve shared postgres dsn", "err", err)
		return 1
	}

	adminDSN = dsn
	code := m.Run()
	terminate(container)
	return code
}

// DSN returns a connection string for an empty database owned by this test,
// created on the shared container and dropped when the test finishes.
//
// It skips the test when Docker is unavailable and REQUIRE_DOCKER is unset.
func DSN(tb testing.TB) string {
	tb.Helper()

	if adminDSN == "" {
		tb.Skip("docker provider is not healthy; set REQUIRE_DOCKER=1 to require it")
	}

	ctx, cancel := context.WithTimeout(tb.Context(), databaseOpTimeout)
	defer cancel()

	name := fmt.Sprintf("test_db_%d", databaseSeq.Add(1))
	admin, err := pgxpool.New(ctx, adminDSN)
	if err != nil {
		tb.Fatalf("connect shared postgres admin database: %v", err)
	}
	defer admin.Close()
	// The name is generated here from a counter, never from test input, so it
	// cannot carry anything an identifier quote would have to defend against.
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		tb.Fatalf("create test database %s: %v", name, err)
	}
	tb.Cleanup(func() { dropDatabase(tb, name) })

	dsn, err := databaseURL(adminDSN, name)
	if err != nil {
		tb.Fatalf("build test database dsn: %v", err)
	}
	return dsn
}

// Migrated returns a DSN for an isolated database with the repository's
// migrations already applied.
//
// source and path name the migrations directory relative to the calling package,
// for example os.DirFS("../../..") and "migrations". They are explicit because
// the depth differs per package, and a guess would fail as "no migrations found"
// rather than as a compile error.
func Migrated(tb testing.TB, source fs.FS, path string) string {
	tb.Helper()

	dsn := DSN(tb)
	ctx, cancel := context.WithTimeout(tb.Context(), migrationTimeout)
	defer cancel()

	if _, err := postgresmigrate.MigrateUp(ctx, postgresmigrate.MigrationOptions{
		DSN:              dsn,
		SourceFS:         source,
		SourcePath:       path,
		ConnectTimeout:   databaseOpTimeout,
		StatementTimeout: migrationTimeout,
		LockTimeout:      migrationLockTimeout,
		CleanupTimeout:   migrationLockTimeout,
	}); err != nil {
		tb.Fatalf("apply migrations from %s: %v", path, err)
	}
	return dsn
}

func dropDatabase(tb testing.TB, name string) {
	tb.Helper()

	// Detached from the test's context, which is already done by the time cleanup
	// runs.
	ctx, cancel := context.WithTimeout(context.Background(), databaseOpTimeout)
	defer cancel()

	drop, err := pgxpool.New(ctx, adminDSN)
	if err != nil {
		tb.Errorf("connect to drop test database %s: %v", name, err)
		return
	}
	defer drop.Close()
	// FORCE, because a pool the test left open would otherwise block the drop and
	// leak the database for the life of the container.
	if _, err := drop.Exec(ctx, "DROP DATABASE "+name+" WITH (FORCE)"); err != nil {
		tb.Errorf("drop test database %s: %v", name, err)
	}
}

func databaseURL(base, name string) (string, error) {
	parsed, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("parse shared postgres dsn: %w", err)
	}
	parsed.Path = "/" + name
	return parsed.String(), nil
}

func dockerIsHealthy(ctx context.Context) bool {
	provider, err := testcontainers.NewDockerProvider()
	if err != nil {
		return false
	}
	defer func() { _ = provider.Close() }()
	return provider.Health(ctx) == nil
}

func terminate(container *tcpostgres.PostgresContainer) {
	if container == nil {
		return
	}
	if err := testcontainers.TerminateContainer(container); err != nil {
		slog.Error("terminate shared postgres container", "err", err)
	}
}

// requireDocker reports whether a missing Docker daemon must fail the run rather
// than skip it. CI sets it, so a suite that ran no integration test at all cannot
// report green.
func requireDocker() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("REQUIRE_DOCKER"))) {
	case "1", "true", "yes":
		return true
	default:
		return false
	}
}
