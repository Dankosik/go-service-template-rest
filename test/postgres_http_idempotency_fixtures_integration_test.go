//go:build integration

package integration_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/httpidempotency"
	"github.com/example/go-service-template-rest/internal/infra/postgres"
	"github.com/example/go-service-template-rest/internal/infra/postgres/pgtest"
	"github.com/example/go-service-template-rest/internal/infra/postgresidempotency"
	"github.com/example/go-service-template-rest/internal/infra/postgresmigrate"
	"github.com/example/go-service-template-rest/internal/waittest"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

const httpIDRecoveryDelay = 50 * time.Millisecond

func httpIDStoreOptions() postgresidempotency.StoreOptions {
	return postgresidempotency.StoreOptions{
		OwnerRecoveryDelay:     httpIDRecoveryDelay,
		CleanupBatchSize:       100,
		MaxMaintenanceLag:      time.Minute,
		MaxRelationBytes:       1 << 40,
		AdmissionHeadroomBytes: 1 << 20,
	}
}

type httpIDFixture struct {
	ctx      context.Context
	pool     *pgxpool.Pool
	store    *postgresidempotency.Store
	dsn      string
	contract httpidempotency.Contract
}

func newHTTPIDFixture(t *testing.T, applicationName string, maxOpenConns int) httpIDFixture {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	t.Cleanup(cancel)
	dsn := httpIDDSN(t, pgtest.Migrated(t, os.DirFS(".."), "migrations"), applicationName)
	pool := newHTTPIDPool(t, ctx, dsn, maxOpenConns)
	store, err := postgresidempotency.NewStore(pool, httpIDStoreOptions())
	if err != nil {
		t.Fatalf("postgresidempotency.NewStore(): %v", err)
	}
	var commitTimestamps string
	if err := pool.QueryRow(ctx, "SHOW track_commit_timestamp").Scan(&commitTimestamps); err != nil {
		t.Fatalf("read track_commit_timestamp: %v", err)
	}
	if commitTimestamps != "on" {
		t.Fatalf("track_commit_timestamp = %q, want on", commitTimestamps)
	}
	if err := store.Maintain(ctx); err != nil {
		t.Fatalf("initial idempotency maintenance: %v", err)
	}
	return httpIDFixture{
		ctx:      ctx,
		pool:     pool,
		store:    store,
		dsn:      dsn,
		contract: httpIDContract(),
	}
}

func newRestartableHTTPIDFixture(t *testing.T) (httpIDFixture, *tcpostgres.PostgresContainer) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	t.Cleanup(cancel)
	container, err := tcpostgres.Run(
		ctx,
		pgtest.DefaultImage,
		tcpostgres.WithDatabase("app"),
		tcpostgres.WithUsername("app"),
		tcpostgres.WithPassword("app"),
		tcpostgres.BasicWaitStrategies(),
		testcontainers.WithEnv(map[string]string{"POSTGRES_INITDB_ARGS": "--no-sync"}),
	)
	if err != nil {
		t.Fatalf("start restartable PostgreSQL: %v", err)
	}
	t.Cleanup(func() { _ = testcontainers.TerminateContainer(container) })
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("resolve restartable PostgreSQL DSN: %v", err)
	}
	dsn = setRestartableCommitTimestamps(t, ctx, container, dsn, "on")
	if _, err := postgresmigrate.MigrateUp(
		ctx,
		postgresmigrate.DefaultOptions(dsn, os.DirFS(".."), "migrations", nil),
	); err != nil {
		t.Fatalf("migrate restartable PostgreSQL: %v", err)
	}
	dsn = httpIDDSN(t, dsn, "idempotency-commit-epoch")
	pool := newHTTPIDPool(t, ctx, dsn, 4)
	store, err := postgresidempotency.NewStore(pool, httpIDStoreOptions())
	if err != nil {
		t.Fatalf("postgresidempotency.NewStore(): %v", err)
	}
	if err := store.Maintain(ctx); err != nil {
		t.Fatalf("initial idempotency maintenance: %v", err)
	}
	return httpIDFixture{ctx: ctx, pool: pool, store: store, dsn: dsn, contract: httpIDContract()}, container
}

func setRestartableCommitTimestamps(
	t *testing.T,
	ctx context.Context,
	container *tcpostgres.PostgresContainer,
	dsn, value string,
) string {
	t.Helper()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect restartable PostgreSQL admin: %v", err)
	}
	statement := "ALTER SYSTEM SET track_commit_timestamp = 'on'"
	if value == "off" {
		statement = "ALTER SYSTEM SET track_commit_timestamp = 'off'"
	}
	if _, err := admin.Exec(ctx, statement); err != nil {
		admin.Close()
		t.Fatalf("set track_commit_timestamp=%s: %v", value, err)
	}
	admin.Close()
	stopTimeout := time.Second
	if err := container.Stop(ctx, &stopTimeout); err != nil {
		t.Fatalf("stop restartable PostgreSQL: %v", err)
	}
	if err := container.Start(ctx); err != nil {
		t.Fatalf("restart PostgreSQL: %v", err)
	}
	refreshedDSN, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("refresh restartable PostgreSQL DSN: %v", err)
	}
	var lastPingErr error
	waittest.UntilFunc(t, 30*time.Second, func() bool {
		pool, err := pgxpool.New(ctx, refreshedDSN)
		if err != nil {
			lastPingErr = err
			return false
		}
		defer pool.Close()
		lastPingErr = pool.Ping(ctx)
		return lastPingErr == nil
	}, func() string {
		return fmt.Sprintf("restartable PostgreSQL to accept connections: %v", lastPingErr)
	})
	pool, err := pgxpool.New(ctx, refreshedDSN)
	if err != nil {
		t.Fatalf("connect restarted PostgreSQL: %v", err)
	}
	defer pool.Close()
	var got string
	if err := pool.QueryRow(ctx, "SHOW track_commit_timestamp").Scan(&got); err != nil {
		t.Fatalf("read restarted track_commit_timestamp: %v", err)
	}
	if got != value {
		t.Fatalf("track_commit_timestamp after restart = %q, want %q", got, value)
	}
	return refreshedDSN
}

func newHTTPIDPool(t *testing.T, ctx context.Context, dsn string, maxOpenConns int) *pgxpool.Pool {
	t.Helper()
	pool, err := postgres.Open(ctx, postgres.Options{
		DSN: dsn,

		MaxOpenConns: maxOpenConns,
	})
	if err != nil {
		t.Fatalf("postgres.Open(): %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func httpIDDSN(t *testing.T, dsn, applicationName string) string {
	t.Helper()
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("parse PostgreSQL DSN: %v", err)
	}
	query := parsed.Query()
	query.Set("application_name", applicationName)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func httpIDDSNParam(t *testing.T, dsn, name, value string) string {
	t.Helper()
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("parse PostgreSQL DSN: %v", err)
	}
	query := parsed.Query()
	query.Set(name, value)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func httpIDContract() httpidempotency.Contract {
	return httpidempotency.Contract{
		OperationID:         "test.create",
		APIVersion:          "v1",
		KeyMaxBytes:         128,
		FingerprintVersions: []string{"v1", "v2"},
		ResultCodecs:        []string{"test/v1"},
		ReplayStatuses:      []int{http.StatusCreated},
		StableHeaders:       []string{"location"},
		ResultMaxBytes:      1024,
		ReplayTTL:           time.Hour,
		DuplicateRisk: httpidempotency.DuplicateRiskPolicy{
			Duration: 2 * time.Hour,
		},
		InProgressWait: time.Second,
		RetryAfter:     time.Second,
		ExternalEffect: httpidempotency.ExternalEffectNone,
	}
}

func httpIDAttempt(t *testing.T, version string, canonical string) httpidempotency.Attempt {
	return httpIDAttemptWithKey(t, version, canonical, "key-a")
}

func httpIDAttemptWithKey(t *testing.T, version string, canonical string, key string) httpidempotency.Attempt {
	t.Helper()
	fingerprint, err := httpidempotency.NewFingerprint(version, []byte(canonical))
	if err != nil {
		t.Fatalf("new fingerprint: %v", err)
	}
	attempt, err := httpidempotency.NewAttempt(httpidempotency.Scope{
		Authority:   "authority-a",
		OperationID: "test.create",
		APIVersion:  "v1",
		Resource:    "resource-a",
		Environment: "test",
		Region:      "local",
	}, key, fingerprint)
	if err != nil {
		t.Fatalf("new attempt: %v", err)
	}
	return attempt
}

func httpIDResolver(values map[string]httpidempotency.Fingerprint) postgresidempotency.FingerprintResolver {
	return func(version string) (httpidempotency.Fingerprint, error) {
		fingerprint, ok := values[version]
		if !ok {
			return httpidempotency.Fingerprint{}, errors.New("retained version is unavailable")
		}
		return fingerprint, nil
	}
}

func httpIDResult(payload string) httpidempotency.Result {
	return httpidempotency.Result{
		Status:    http.StatusCreated,
		MediaType: "application/json",
		Codec:     "test/v1",
		Headers:   http.Header{"Location": {"/resources/1"}},
		Payload:   []byte(payload),
	}
}

func mustHTTPIDReserve(
	t *testing.T,
	fixture httpIDFixture,
	attempt httpidempotency.Attempt,
	resolve postgresidempotency.FingerprintResolver,
) httpidempotency.Reservation {
	t.Helper()
	reservation, decision, err := fixture.store.Reserve(fixture.ctx, fixture.contract, attempt, resolve)
	if err != nil {
		t.Fatalf("Reserve(): %v", err)
	}
	if decision.Outcome != httpidempotency.OutcomeExecute {
		t.Fatalf("Reserve() outcome = %v, want execute", decision.Outcome)
	}
	return reservation
}

func mustHTTPIDComplete(
	t *testing.T,
	fixture httpIDFixture,
	reservation httpidempotency.Reservation,
	resolve postgresidempotency.FingerprintResolver,
	result httpidempotency.Result,
) {
	t.Helper()
	if err := postgres.InTx(fixture.ctx, fixture.pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		acquired, decision, err := fixture.store.Acquire(fixture.ctx, tx, fixture.contract, reservation, resolve)
		if err != nil {
			return err
		}
		if decision.Outcome != httpidempotency.OutcomeExecute {
			return fmt.Errorf("acquire outcome %v", decision.Outcome)
		}
		return fixture.store.Complete(fixture.ctx, tx, fixture.contract, acquired, result)
	}); err != nil {
		t.Fatalf("caller transaction: %v", err)
	}
}
