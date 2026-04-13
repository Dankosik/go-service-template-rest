package postgres

import (
	"context"
	"errors"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMain(m *testing.M) {
	restore := clearPostgresEnvForTests()
	code := m.Run()
	restore()
	os.Exit(code)
}

func clearPostgresEnvForTests() func() {
	type envState struct {
		name  string
		value string
		set   bool
	}

	states := make([]envState, 0, len(recognizedPostgresEnvVars))
	for _, name := range recognizedPostgresEnvVars {
		value, set := os.LookupEnv(name)
		states = append(states, envState{name: name, value: value, set: set})
		_ = os.Unsetenv(name)
	}

	return func() {
		for _, state := range states {
			if state.set {
				_ = os.Setenv(state.name, state.value)
				continue
			}
			_ = os.Unsetenv(state.name)
		}
	}
}

func TestNewRejectsEmptyDSN(t *testing.T) {
	t.Parallel()

	_, err := New(context.Background(), Options{
		DSN:                "   \n\t",
		ConnectTimeout:     time.Second,
		HealthcheckTimeout: time.Second,
		MaxOpenConns:       10,
		MaxIdleConns:       5,
		ConnMaxLifetime:    time.Minute,
	})
	if err == nil {
		t.Fatal("New() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "postgres dsn is empty") {
		t.Fatalf("New() error = %q, want to contain %q", err.Error(), "postgres dsn is empty")
	}
	if !errors.Is(err, ErrConfig) {
		t.Fatalf("New() error = %v, want ErrConfig", err)
	}
}

func TestNewRejectsInvalidOptions(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		opts Options
	}{
		{
			name: "connect timeout",
			opts: Options{
				DSN:                "postgres://user:pass@localhost:5432/db?sslmode=disable",
				HealthcheckTimeout: time.Second,
				MaxOpenConns:       10,
				MaxIdleConns:       5,
				ConnMaxLifetime:    time.Minute,
			},
		},
		{
			name: "healthcheck timeout",
			opts: Options{
				DSN:             "postgres://user:pass@localhost:5432/db?sslmode=disable",
				ConnectTimeout:  time.Second,
				MaxOpenConns:    10,
				MaxIdleConns:    5,
				ConnMaxLifetime: time.Minute,
			},
		},
		{
			name: "max open conns",
			opts: Options{
				DSN:                "postgres://user:pass@localhost:5432/db?sslmode=disable",
				ConnectTimeout:     time.Second,
				HealthcheckTimeout: time.Second,
				MaxIdleConns:       5,
				ConnMaxLifetime:    time.Minute,
			},
		},
		{
			name: "max idle conns",
			opts: Options{
				DSN:                "postgres://user:pass@localhost:5432/db?sslmode=disable",
				ConnectTimeout:     time.Second,
				HealthcheckTimeout: time.Second,
				MaxOpenConns:       10,
				MaxIdleConns:       11,
				ConnMaxLifetime:    time.Minute,
			},
		},
		{
			name: "conn max lifetime",
			opts: Options{
				DSN:                "postgres://user:pass@localhost:5432/db?sslmode=disable",
				ConnectTimeout:     time.Second,
				HealthcheckTimeout: time.Second,
				MaxOpenConns:       10,
				MaxIdleConns:       5,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := New(context.Background(), tc.opts)
			if err == nil {
				t.Fatal("New() error = nil, want non-nil")
			}
			if !errors.Is(err, ErrConfig) {
				t.Fatalf("New() error = %v, want ErrConfig", err)
			}
		})
	}
}

func TestNewInvalidDSNIsRedacted(t *testing.T) {
	t.Parallel()

	rawDSN := "postgres://user:top-secret%@localhost:5432/app"
	_, err := New(context.Background(), Options{
		DSN:                rawDSN,
		ConnectTimeout:     time.Second,
		HealthcheckTimeout: time.Second,
		MaxOpenConns:       10,
		MaxIdleConns:       5,
		ConnMaxLifetime:    time.Minute,
	})
	if err == nil {
		t.Fatal("New() error = nil, want non-nil")
	}
	if !errors.Is(err, ErrConfig) {
		t.Fatalf("New() error = %v, want ErrConfig", err)
	}
	if !strings.Contains(err.Error(), "parse postgres dsn") || !strings.Contains(err.Error(), "redacted") {
		t.Fatalf("New() error = %v, want redacted parse context", err)
	}
	for _, leaked := range []string{rawDSN, "top-secret", "user"} {
		if strings.Contains(err.Error(), leaked) {
			t.Fatalf("New() error = %v, leaked %q", err, leaked)
		}
	}
}

func TestParsePoolConfigAcceptsStrictSingleTargetDSNs(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		dsn  string
	}{
		{
			name: "url",
			dsn:  "postgres://user:pass@localhost:5432/app?sslmode=disable",
		},
		{
			name: "keyword value",
			dsn:  "user='user' password='pass' host='localhost' port='5432' dbname='app' sslmode='disable'",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			config, err := parsePoolConfig(tc.dsn)
			if err != nil {
				t.Fatalf("parsePoolConfig() error = %v", err)
			}
			if config.ConnConfig.Host != "localhost" {
				t.Fatalf("Host = %q, want localhost", config.ConnConfig.Host)
			}
			if config.ConnConfig.Port != 5432 {
				t.Fatalf("Port = %d, want 5432", config.ConnConfig.Port)
			}
			if config.ConnConfig.User != "user" {
				t.Fatalf("User = %q, want user", config.ConnConfig.User)
			}
			if config.ConnConfig.Password != "pass" {
				t.Fatalf("Password = %q, want pass", config.ConnConfig.Password)
			}
			if config.ConnConfig.Database != "app" {
				t.Fatalf("Database = %q, want app", config.ConnConfig.Database)
			}
			if len(config.ConnConfig.Fallbacks) != 0 {
				t.Fatalf("Fallbacks len = %d, want 0", len(config.ConnConfig.Fallbacks))
			}
		})
	}
}

func TestParsePoolConfigRejectsAmbientPostgresEnv(t *testing.T) {
	validDSN := "postgres://user:pass@localhost:5432/app?sslmode=disable"

	for _, envName := range recognizedPostgresEnvVars {
		t.Run(envName, func(t *testing.T) {
			t.Setenv(envName, "ambient-value")

			_, err := parsePoolConfig(validDSN)
			requirePostgresConfigError(t, err, "postgres dsn uses unsupported ambient PG environment")
			requireErrorDoesNotContain(t, err, "ambient-value", envName)
		})
	}

	t.Run("empty dsn rejects environment only parsing", func(t *testing.T) {
		t.Setenv("PGHOST", "ambient-host")

		_, err := parsePoolConfig("")
		requirePostgresConfigError(t, err, "postgres dsn is empty")
		requireErrorDoesNotContain(t, err, "ambient-host")
	})
}

func TestParsePoolConfigRejectsDisallowedSourcesAndMissingRequiredFields(t *testing.T) {
	testCases := []struct {
		name             string
		dsn              string
		want             string
		forbiddenDetails []string
	}{
		{
			name:             "service",
			dsn:              "service=prodservice user=user password=pass host=localhost port=5432 dbname=app sslmode=disable",
			want:             "postgres dsn uses unsupported service/passfile source",
			forbiddenDetails: []string{"prodservice"},
		},
		{
			name:             "servicefile",
			dsn:              "servicefile=/tmp/pg_service.conf user=user password=pass host=localhost port=5432 dbname=app sslmode=disable",
			want:             "postgres dsn uses unsupported service/passfile source",
			forbiddenDetails: []string{"/tmp/pg_service.conf"},
		},
		{
			name:             "passfile",
			dsn:              "postgres://user:pass@localhost:5432/app?sslmode=disable&passfile=/tmp/.pgpass",
			want:             "postgres dsn uses unsupported service/passfile source",
			forbiddenDetails: []string{"/tmp/.pgpass"},
		},
		{
			name:             "sslcert",
			dsn:              "postgres://user:pass@localhost:5432/app?sslmode=require&sslcert=/tmp/client.crt",
			want:             "postgres dsn uses unsupported TLS file source",
			forbiddenDetails: []string{"/tmp/client.crt"},
		},
		{
			name:             "sslkey",
			dsn:              "postgres://user:pass@localhost:5432/app?sslmode=require&sslkey=/tmp/client.key",
			want:             "postgres dsn uses unsupported TLS file source",
			forbiddenDetails: []string{"/tmp/client.key"},
		},
		{
			name:             "sslpassword",
			dsn:              "postgres://user:pass@localhost:5432/app?sslmode=require&sslpassword=client-secret",
			want:             "postgres dsn uses unsupported TLS file source",
			forbiddenDetails: []string{"client-secret"},
		},
		{
			name:             "sslrootcert",
			dsn:              "postgres://user:pass@localhost:5432/app?sslmode=require&sslrootcert=/tmp/root.crt",
			want:             "postgres dsn uses unsupported TLS file source",
			forbiddenDetails: []string{"/tmp/root.crt"},
		},
		{
			name:             "missing password",
			dsn:              "postgres://user@localhost:5432/app?sslmode=disable",
			want:             "postgres dsn requires explicit host, port, user, password, database, and sslmode",
			forbiddenDetails: []string{"user@localhost"},
		},
		{
			name: "missing host",
			dsn:  "user=user password=pass port=5432 dbname=app sslmode=disable",
			want: "postgres dsn requires explicit host, port, user, password, database, and sslmode",
		},
		{
			name: "missing port",
			dsn:  "postgres://user:pass@localhost/app?sslmode=disable",
			want: "postgres dsn requires explicit host, port, user, password, database, and sslmode",
		},
		{
			name: "missing user",
			dsn:  "postgres://:pass@localhost:5432/app?sslmode=disable",
			want: "postgres dsn requires explicit host, port, user, password, database, and sslmode",
		},
		{
			name: "missing database",
			dsn:  "postgres://user:pass@localhost:5432/?sslmode=disable",
			want: "postgres dsn requires explicit host, port, user, password, database, and sslmode",
		},
		{
			name: "missing sslmode",
			dsn:  "postgres://user:pass@localhost:5432/app",
			want: "postgres dsn requires explicit host, port, user, password, database, and sslmode",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := parsePoolConfig(tc.dsn)
			requirePostgresConfigError(t, err, tc.want)
			requireErrorDoesNotContain(t, err, tc.dsn)
			for _, forbidden := range tc.forbiddenDetails {
				requireErrorDoesNotContain(t, err, forbidden)
			}
		})
	}
}

func TestParsePoolConfigRejectsSharedFileDefaultDSNKeys(t *testing.T) {
	for _, key := range postgresFileDefaultDSNKeys {
		t.Run(key.name, func(t *testing.T) {
			dsn := "postgres://user:pass@localhost:5432/app?sslmode=disable&" + key.name + "=file-secret"

			_, err := parsePoolConfig(dsn)
			requirePostgresConfigError(t, err, key.validationMessage)
			requireErrorDoesNotContain(t, err, "file-secret")
		})
	}
}

func TestNormalizePostgresDSNSuppressesFileDefaultKeys(t *testing.T) {
	normalizedURL, err := normalizePostgresURLDSN("postgres://user:pass@localhost:5432/app?sslmode=disable")
	if err != nil {
		t.Fatalf("normalizePostgresURLDSN() error = %v", err)
	}
	parsedURL, err := url.Parse(normalizedURL)
	if err != nil {
		t.Fatalf("url.Parse(normalizedURL) error = %v", err)
	}
	query := parsedURL.Query()
	for _, key := range postgresFileDefaultDSNKeys {
		values, present := query[key.name]
		if !present {
			t.Fatalf("normalized URL query missing %q in %q", key.name, normalizedURL)
		}
		if len(values) != 1 || values[0] != "" {
			t.Fatalf("normalized URL query %q = %#v, want one empty value", key.name, values)
		}
	}
	for _, key := range []string{"service", "servicefile"} {
		if _, present := query[key]; present {
			t.Fatalf("normalized URL query contains disallowed-only key %q in %q", key, normalizedURL)
		}
	}

	normalizedKeywordValue := normalizePostgresKeywordValueDSN("user=user password=pass host=localhost port=5432 dbname=app sslmode=disable")
	settings, err := parsePostgresKeywordValueDSNSettings(normalizedKeywordValue)
	if err != nil {
		t.Fatalf("parsePostgresKeywordValueDSNSettings() error = %v", err)
	}
	for _, key := range postgresFileDefaultDSNKeys {
		value, present := settings[key.name]
		if !present {
			t.Fatalf("normalized keyword/value DSN missing %q in %q", key.name, normalizedKeywordValue)
		}
		if value != "" {
			t.Fatalf("normalized keyword/value DSN %q = %q, want empty", key.name, value)
		}
	}
	for _, key := range []string{"service", "servicefile"} {
		if _, present := settings[key]; present {
			t.Fatalf("normalized keyword/value DSN contains disallowed-only key %q in %q", key, normalizedKeywordValue)
		}
	}
}

func TestParsePoolConfigRejectsFallbackProducingDSNs(t *testing.T) {
	testCases := []struct {
		name string
		dsn  string
		want string
	}{
		{
			name: "multi-host url",
			dsn:  "postgres://user:pass@first:5432,second:5432/app?sslmode=disable",
			want: "postgres dsn fallback targets are not supported",
		},
		{
			name: "multi-host keyword value",
			dsn:  "user=user password=pass host=first,second port=5432 dbname=app sslmode=disable",
			want: "postgres dsn fallback targets are not supported",
		},
		{
			name: "omitted sslmode",
			dsn:  "postgres://user:pass@localhost:5432/app",
			want: "postgres dsn requires explicit host, port, user, password, database, and sslmode",
		},
		{
			name: "sslmode prefer",
			dsn:  "postgres://user:pass@localhost:5432/app?sslmode=prefer",
			want: "postgres dsn fallback targets are not supported",
		},
		{
			name: "sslmode allow",
			dsn:  "postgres://user:pass@localhost:5432/app?sslmode=allow",
			want: "postgres dsn fallback targets are not supported",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := parsePoolConfig(tc.dsn)
			requirePostgresConfigError(t, err, tc.want)
		})
	}
}

func TestProbeAddress(t *testing.T) {
	t.Parallel()

	t.Run("valid dsn", func(t *testing.T) {
		t.Parallel()

		address, err := ProbeAddress("postgres://user:pass@localhost:5432/app?sslmode=disable")
		if err != nil {
			t.Fatalf("ProbeAddress() error = %v", err)
		}
		if address != "localhost:5432" {
			t.Fatalf("ProbeAddress() = %q, want localhost:5432", address)
		}
	})

	t.Run("invalid dsn is redacted", func(t *testing.T) {
		t.Parallel()

		rawDSN := "postgres://user:top-secret%@localhost:5432/app"
		_, err := ProbeAddress(rawDSN)
		if err == nil {
			t.Fatal("ProbeAddress() error = nil, want non-nil")
		}
		if !errors.Is(err, ErrConfig) {
			t.Fatalf("ProbeAddress() error = %v, want ErrConfig", err)
		}
		if !strings.Contains(err.Error(), "parse postgres dsn") || !strings.Contains(err.Error(), "redacted") {
			t.Fatalf("ProbeAddress() error = %v, want redacted parse context", err)
		}
		for _, leaked := range []string{rawDSN, "top-secret", "user"} {
			if strings.Contains(err.Error(), leaked) {
				t.Fatalf("ProbeAddress() error = %v, leaked %q", err, leaked)
			}
		}
	})

	t.Run("invalid probe target shape", func(t *testing.T) {
		t.Parallel()

		_, err := ProbeAddress("user=user password=pass host=/var/run/postgresql port=5432 dbname=app sslmode=disable")
		if err == nil {
			t.Fatal("ProbeAddress() error = nil, want non-nil")
		}
		if !errors.Is(err, ErrConfig) {
			t.Fatalf("ProbeAddress() error = %v, want ErrConfig", err)
		}
		if !strings.Contains(err.Error(), "postgres dsn requires valid single tcp host and port") {
			t.Fatalf("ProbeAddress() error = %v, want invalid target context", err)
		}
	})
}

func requirePostgresConfigError(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want %q", want)
	}
	if !errors.Is(err, ErrConfig) {
		t.Fatalf("error = %v, want ErrConfig", err)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want to contain %q", err, want)
	}
}

func requireErrorDoesNotContain(t *testing.T, err error, forbidden ...string) {
	t.Helper()
	if err == nil {
		t.Fatal("error = nil, want non-nil")
	}
	for _, value := range forbidden {
		if value == "" {
			continue
		}
		if strings.Contains(err.Error(), value) {
			t.Fatalf("error = %v, leaked %q", err, value)
		}
	}
}

func TestPoolHelpersWithoutConnection(t *testing.T) {
	t.Parallel()

	var nilPool *Pool
	nilPool.Close()

	if err := nilPool.Check(context.Background()); err == nil {
		t.Fatal("(*Pool)(nil).Check() error = nil, want non-nil")
	} else if !errors.Is(err, ErrHealthcheck) {
		t.Fatalf("(*Pool)(nil).Check() error = %v, want ErrHealthcheck", err)
	}

	pool := &Pool{}
	if got := pool.Name(); got != "postgres" {
		t.Fatalf("Name() = %q, want %q", got, "postgres")
	}

	pool.Close()
	if err := pool.Check(context.Background()); err == nil {
		t.Fatal("Check() error = nil, want non-nil for nil internal pool")
	} else if !errors.Is(err, ErrHealthcheck) {
		t.Fatalf("Check() error = %v, want ErrHealthcheck", err)
	}
}

func TestMaxIdleConnLimiter(t *testing.T) {
	t.Parallel()

	limiter := newMaxIdleConnLimiter(2)
	first := &pgx.Conn{}
	second := &pgx.Conn{}
	third := &pgx.Conn{}

	if !limiter.afterRelease(first) {
		t.Fatal("afterRelease(first) = false, want true")
	}
	if !limiter.afterRelease(second) {
		t.Fatal("afterRelease(second) = false, want true")
	}
	if limiter.afterRelease(third) {
		t.Fatal("afterRelease(third) = true, want false when max idle is full")
	}

	limiter.beforeAcquire(first)
	if !limiter.afterRelease(third) {
		t.Fatal("afterRelease(third) after first acquire = false, want true")
	}

	limiter.beforeClose(second)
	if !limiter.afterRelease(first) {
		t.Fatal("afterRelease(first) after second close = false, want true")
	}

	disabled := newMaxIdleConnLimiter(0)
	if disabled.afterRelease(&pgx.Conn{}) {
		t.Fatal("afterRelease() with max idle 0 = true, want false")
	}
}

func TestMaxIdleConnLimiterConcurrentReleases(t *testing.T) {
	t.Parallel()

	limiter := newMaxIdleConnLimiter(2)
	conns := make([]*pgx.Conn, 10)
	for i := range conns {
		conns[i] = &pgx.Conn{}
	}

	var wg sync.WaitGroup
	kept := make(chan bool, len(conns))
	for _, conn := range conns {
		wg.Add(1)
		go func() {
			defer wg.Done()
			kept <- limiter.afterRelease(conn)
		}()
	}
	wg.Wait()
	close(kept)

	var keepCount int
	for keep := range kept {
		if keep {
			keepCount++
		}
	}
	if keepCount != 2 {
		t.Fatalf("kept releases = %d, want 2", keepCount)
	}
}

func TestInstallMaxIdleConnLimiterComposesPoolHooks(t *testing.T) {
	t.Parallel()

	var beforeAcquireCalled bool
	var afterReleaseCalled bool
	var beforeCloseCalled bool
	poolConfig := &pgxpool.Config{
		BeforeAcquire: func(context.Context, *pgx.Conn) bool {
			beforeAcquireCalled = true
			return true
		},
		AfterRelease: func(*pgx.Conn) bool {
			afterReleaseCalled = true
			return true
		},
		BeforeClose: func(*pgx.Conn) {
			beforeCloseCalled = true
		},
	}
	first := &pgx.Conn{}
	second := &pgx.Conn{}

	installMaxIdleConnLimiter(poolConfig, 1)

	if !poolConfig.AfterRelease(first) {
		t.Fatal("AfterRelease(first) = false, want true")
	}
	if !afterReleaseCalled {
		t.Fatal("original AfterRelease was not called")
	}
	if poolConfig.AfterRelease(second) {
		t.Fatal("AfterRelease(second) = true, want false when max idle is full")
	}

	if !poolConfig.BeforeAcquire(context.Background(), first) {
		t.Fatal("BeforeAcquire(first) = false, want true")
	}
	if !beforeAcquireCalled {
		t.Fatal("original BeforeAcquire was not called")
	}
	if !poolConfig.AfterRelease(second) {
		t.Fatal("AfterRelease(second) after first acquire = false, want true")
	}

	poolConfig.BeforeClose(second)
	if !beforeCloseCalled {
		t.Fatal("original BeforeClose was not called")
	}
}
