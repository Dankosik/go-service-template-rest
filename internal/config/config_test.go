package config

import (
	"context"
	"errors"
	"log/slog"
	"maps"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/v2"
)

func TestLoadDefaults(t *testing.T) {
	resetConfigEnv(t)

	cfg, report, err := LoadDetailed(LoadOptions{})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v", err)
	}

	if cfg.App.Env != "local" {
		t.Fatalf("App.Env = %q, want local", cfg.App.Env)
	}
	if cfg.App.Version != "dev" {
		t.Fatalf("App.Version = %q, want dev", cfg.App.Version)
	}
	if cfg.HTTP.Addr != ":8080" {
		t.Fatalf("HTTP.Addr = %q, want :8080", cfg.HTTP.Addr)
	}
	if cfg.HTTP.ShutdownTimeout != 30*time.Second {
		t.Fatalf("HTTP.ShutdownTimeout = %s, want 30s", cfg.HTTP.ShutdownTimeout)
	}
	if cfg.HTTP.ReadinessTimeout != 4*time.Second {
		t.Fatalf("HTTP.ReadinessTimeout = %s, want 4s", cfg.HTTP.ReadinessTimeout)
	}
	if cfg.HTTP.ReadinessPropagationDelay != 15*time.Second {
		t.Fatalf("HTTP.ReadinessPropagationDelay = %s, want 15s", cfg.HTTP.ReadinessPropagationDelay)
	}
	if cfg.Redis.Mode != "cache" {
		t.Fatalf("Redis.Mode = %q, want cache", cfg.Redis.Mode)
	}
	if cfg.Redis.AllowStoreMode {
		t.Fatalf("Redis.AllowStoreMode = true, want false")
	}
	if cfg.Postgres.Enabled {
		t.Fatalf("Postgres.Enabled = true, want false")
	}
	if cfg.Postgres.DSN != "" {
		t.Fatalf("Postgres.DSN = %q, want empty", cfg.Postgres.DSN)
	}
	if cfg.Observability.OTel.ServiceName != "service" {
		t.Fatalf("Observability.OTel.ServiceName = %q, want service", cfg.Observability.OTel.ServiceName)
	}
	if cfg.Observability.OTel.TracesSampler != "parentbased_traceidratio" {
		t.Fatalf("Observability.OTel.TracesSampler = %q, want parentbased_traceidratio", cfg.Observability.OTel.TracesSampler)
	}
	if report.LoadDuration <= 0 {
		t.Fatalf("LoadDuration = %s, want > 0", report.LoadDuration)
	}
	if report.ValidateDuration <= 0 {
		t.Fatalf("ValidateDuration = %s, want > 0", report.ValidateDuration)
	}
}

func TestPrecedenceNamespaceWinsOverFileAndOverlay(t *testing.T) {
	resetConfigEnv(t)

	basePath := writeTempConfig(t, `
http:
  addr: ":8081"
`)
	overlayPath := writeTempConfig(t, `
http:
  addr: ":8082"
`)

	t.Setenv("APP__HTTP__ADDR", ":8083")

	cfg, _, err := LoadDetailed(LoadOptions{
		ConfigPath:     basePath,
		ConfigOverlays: []string{overlayPath},
	})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v", err)
	}

	if cfg.HTTP.Addr != ":8083" {
		t.Fatalf("HTTP.Addr = %q, want :8083", cfg.HTTP.Addr)
	}
}

func TestEmptyNamespaceEnvOverridesRequiredDefault(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("APP__HTTP__ADDR", "")

	_, _, err := LoadDetailed(LoadOptions{})
	if err == nil {
		t.Fatalf("LoadDetailed() expected validation error for empty env override")
	}
	if !errors.Is(err, ErrValidate) {
		t.Fatalf("error = %v, want ErrValidate", err)
	}
}

func TestResourceIdentityFieldsCannotBeEmpty(t *testing.T) {
	for _, tc := range []struct {
		name       string
		envKey     string
		wantDetail string
	}{
		{
			name:       "app version",
			envKey:     "APP__APP__VERSION",
			wantDetail: "app.version cannot be empty",
		},
		{
			name:       "otel service name",
			envKey:     "APP__OBSERVABILITY__OTEL__SERVICE_NAME",
			wantDetail: "observability.otel.service_name cannot be empty",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resetConfigEnv(t)
			t.Setenv(tc.envKey, "")

			_, _, err := LoadDetailed(LoadOptions{})
			if err == nil {
				t.Fatal("LoadDetailed() error = nil, want validation error")
			}
			if !errors.Is(err, ErrValidate) {
				t.Fatalf("error = %v, want ErrValidate", err)
			}
			if !strings.Contains(err.Error(), tc.wantDetail) {
				t.Fatalf("error = %v, want %q", err, tc.wantDetail)
			}
		})
	}
}

func TestEmptyNamespaceEnvOverridesConfigFileValue(t *testing.T) {
	resetConfigEnv(t)

	configPath := writeTempConfig(t, `
observability:
  otel:
    exporter:
      otlp_endpoint: "https://otel.example.com:4318"
`)
	t.Setenv("APP__OBSERVABILITY__OTEL__EXPORTER__OTLP_ENDPOINT", "")

	cfg, _, err := LoadDetailed(LoadOptions{ConfigPath: configPath})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v", err)
	}
	if cfg.Observability.OTel.Exporter.OTLPEndpoint != "" {
		t.Fatalf("OTLPEndpoint = %q, want empty env override", cfg.Observability.OTel.Exporter.OTLPEndpoint)
	}
}

func TestNamespaceEnvPreservesRawDataBearingStrings(t *testing.T) {
	resetConfigEnv(t)

	postgresDSN := " postgres://user:pass@localhost:5432/app?sslmode=disable "
	username := " redis user with surrounding whitespace "
	password := " redis password with surrounding whitespace "
	mongoURI := " mongodb://user:pass@localhost:27017/app "
	headers := " authorization=Bearer token, x-trace= spaced value "
	t.Setenv("APP__POSTGRES__DSN", postgresDSN)
	t.Setenv("APP__REDIS__USERNAME", username)
	t.Setenv("APP__REDIS__PASSWORD", password)
	t.Setenv("APP__MONGO__URI", mongoURI)
	t.Setenv("APP__OBSERVABILITY__OTEL__EXPORTER__OTLP_HEADERS", headers)

	cfg, _, err := LoadDetailed(LoadOptions{})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v", err)
	}
	if cfg.Postgres.DSN != postgresDSN {
		t.Fatalf("Postgres.DSN = %q, want exact env value %q", cfg.Postgres.DSN, postgresDSN)
	}
	if cfg.Redis.Username != username {
		t.Fatalf("Redis.Username = %q, want exact env value %q", cfg.Redis.Username, username)
	}
	if cfg.Redis.Password != password {
		t.Fatalf("Redis.Password = %q, want exact env value %q", cfg.Redis.Password, password)
	}
	if cfg.Mongo.URI != mongoURI {
		t.Fatalf("Mongo.URI = %q, want exact env value %q", cfg.Mongo.URI, mongoURI)
	}
	if cfg.Observability.OTel.Exporter.OTLPHeaders != headers {
		t.Fatalf("OTLPHeaders = %q, want exact env value %q", cfg.Observability.OTel.Exporter.OTLPHeaders, headers)
	}
}

func TestNamespaceEnvTrimsSyntaxFields(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("APP__REDIS__MODE", " STORE ")
	t.Setenv("APP__REDIS__ALLOW_STORE_MODE", "true")
	t.Setenv("APP__REDIS__STALE_WINDOW", "0s")
	t.Setenv("APP__REDIS__ADDR", " 127.0.0.1:6379 ")
	t.Setenv("APP__MONGO__DATABASE", " app ")

	cfg, _, err := LoadDetailed(LoadOptions{})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v", err)
	}
	if cfg.Redis.Mode != "store" {
		t.Fatalf("Redis.Mode = %q, want store", cfg.Redis.Mode)
	}
	if cfg.Redis.Addr != "127.0.0.1:6379" {
		t.Fatalf("Redis.Addr = %q, want trimmed address", cfg.Redis.Addr)
	}
	if cfg.Mongo.Database != "app" {
		t.Fatalf("Mongo.Database = %q, want trimmed database", cfg.Mongo.Database)
	}
}

func TestFlatEnvKeysAreIgnored(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("HTTP_ADDR", ":9090")

	cfg, _, err := LoadDetailed(LoadOptions{})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v", err)
	}

	if cfg.HTTP.Addr != ":8080" {
		t.Fatalf("HTTP.Addr = %q, want default :8080", cfg.HTTP.Addr)
	}
}

func TestNamespaceEnvForConfigKey(t *testing.T) {
	t.Parallel()

	if got := namespaceEnvForConfigKey("app.env"); got != "APP__APP__ENV" {
		t.Fatalf("namespaceEnvForConfigKey(app.env) = %q, want APP__APP__ENV", got)
	}
}

func TestEnvExampleLoadsThroughConfigLoader(t *testing.T) {
	resetConfigEnv(t)

	for key, value := range readEnvExample(t, filepath.Join("..", "..", "env", ".env.example")) {
		t.Setenv(key, value)
	}

	cfg, _, err := LoadDetailed(LoadOptions{})
	if err != nil {
		t.Fatalf("LoadDetailed() with env/.env.example values error = %v", err)
	}
	if cfg.HTTP.ShutdownTimeout != 30*time.Second {
		t.Fatalf("HTTP.ShutdownTimeout = %s, want 30s from env/.env.example", cfg.HTTP.ShutdownTimeout)
	}
}

func TestTST001PrecedenceDeterministicSnapshotAcrossRepeatedLoads(t *testing.T) {
	resetConfigEnv(t)

	basePath := writeTempConfig(t, `
http:
  addr: ":8081"
`)
	overlayPath := writeTempConfig(t, `
http:
  addr: ":8082"
`)

	t.Setenv("APP__HTTP__ADDR", ":8083")

	opts := LoadOptions{
		ConfigPath:     basePath,
		ConfigOverlays: []string{overlayPath},
	}

	cfg1, report1, err := LoadDetailed(opts)
	if err != nil {
		t.Fatalf("first LoadDetailed() error = %v", err)
	}
	cfg2, report2, err := LoadDetailed(opts)
	if err != nil {
		t.Fatalf("second LoadDetailed() error = %v", err)
	}

	if cfg1 != cfg2 {
		t.Fatalf("config snapshots differ between repeated loads: first=%+v second=%+v", cfg1, cfg2)
	}
	if !reflect.DeepEqual(report1.UnknownKeyWarnings, report2.UnknownKeyWarnings) {
		t.Fatalf("UnknownKeyWarnings differs between repeated loads: first=%v second=%v", report1.UnknownKeyWarnings, report2.UnknownKeyWarnings)
	}
}

func TestStrictUnknownKeyRejects(t *testing.T) {
	resetConfigEnv(t)

	configPath := writeTempConfig(t, `
unknown:
  field: value
`)

	_, _, err := LoadDetailed(LoadOptions{
		ConfigPath: configPath,
		Strict:     true,
	})
	if err == nil {
		t.Fatalf("LoadDetailed() expected strict unknown key error")
	}
	if !errors.Is(err, ErrStrictUnknownKey) {
		t.Fatalf("error = %v, want ErrStrictUnknownKey", err)
	}
	if got := ErrorType(err); got != "strict_unknown_key" {
		t.Fatalf("ErrorType(error) = %q, want strict_unknown_key", got)
	}
}

func TestPermissiveUnknownKeyAllows(t *testing.T) {
	resetConfigEnv(t)

	configPath := writeTempConfig(t, `
unknown:
  field: value
`)

	_, report, err := LoadDetailed(LoadOptions{
		ConfigPath: configPath,
		Strict:     false,
	})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v", err)
	}
	if !slices.Contains(report.UnknownKeyWarnings, "unknown.field") {
		t.Fatalf("UnknownKeyWarnings = %v, want unknown.field", report.UnknownKeyWarnings)
	}
}

func TestPermissiveUnknownKeyWarningsPreservedOnValidationError(t *testing.T) {
	resetConfigEnv(t)

	configPath := writeTempConfig(t, `
unknown:
  field: value
`)
	t.Setenv("APP__HTTP__ADDR", "")

	_, report, err := LoadDetailed(LoadOptions{
		ConfigPath: configPath,
		Strict:     false,
	})
	if err == nil {
		t.Fatalf("LoadDetailed() expected validation error")
	}
	if !errors.Is(err, ErrValidate) {
		t.Fatalf("error = %v, want ErrValidate", err)
	}
	if !slices.Contains(report.UnknownKeyWarnings, "unknown.field") {
		t.Fatalf("UnknownKeyWarnings = %v, want unknown.field", report.UnknownKeyWarnings)
	}
}

func TestStrictUnknownKeyRejectsScalarSectionKeys(t *testing.T) {
	for _, tc := range []struct {
		name    string
		envKey  string
		wantKey string
	}{
		{name: "root section", envKey: "APP__HTTP", wantKey: "http"},
		{name: "nested section", envKey: "APP__OBSERVABILITY__OTEL", wantKey: "observability.otel"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resetConfigEnv(t)
			t.Setenv(tc.envKey, "oops")

			_, report, err := LoadDetailed(LoadOptions{Strict: true})
			if err == nil {
				t.Fatalf("LoadDetailed() expected strict unknown key error")
			}
			if !errors.Is(err, ErrStrictUnknownKey) {
				t.Fatalf("error = %v, want ErrStrictUnknownKey", err)
			}
			if !strings.Contains(err.Error(), tc.wantKey) {
				t.Fatalf("error = %v, want unknown section key %q", err, tc.wantKey)
			}
			if report.FailedStage != StageValidate {
				t.Fatalf("FailedStage = %q, want %q", report.FailedStage, StageValidate)
			}
		})
	}
}

func TestPermissiveUnknownKeyWarnsAndIgnoresScalarSectionKey(t *testing.T) {
	resetConfigEnv(t)

	configPath := writeTempConfig(t, `
http: oops
`)

	cfg, report, err := LoadDetailed(LoadOptions{ConfigPath: configPath})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v", err)
	}
	if !slices.Contains(report.UnknownKeyWarnings, "http") {
		t.Fatalf("UnknownKeyWarnings = %v, want http", report.UnknownKeyWarnings)
	}
	if cfg.HTTP.Addr != ":8080" {
		t.Fatalf("HTTP.Addr = %q, want default :8080 after ignored section scalar", cfg.HTTP.Addr)
	}
}

func TestRemovedObservabilityKeysRejectInStrictMode(t *testing.T) {
	resetConfigEnv(t)

	configPath := writeTempConfig(t, `
observability:
  metrics:
    enabled: true
    path: /internal/metrics
  grafana:
    enabled: true
    cloud_otlp_endpoint: "https://example.invalid"
`)

	_, _, err := LoadDetailed(LoadOptions{
		ConfigPath: configPath,
		Strict:     true,
	})
	if err == nil {
		t.Fatalf("LoadDetailed() expected strict unknown key error")
	}
	if !errors.Is(err, ErrStrictUnknownKey) {
		t.Fatalf("error = %v, want ErrStrictUnknownKey", err)
	}
}

func TestRequiredIfEnabledPostgresSecretPolicy(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("APP__POSTGRES__ENABLED", "true")

	_, _, err := LoadDetailed(LoadOptions{})
	if err == nil {
		t.Fatalf("LoadDetailed() expected secret policy error")
	}
	if !errors.Is(err, ErrSecretPolicy) {
		t.Fatalf("error = %v, want ErrSecretPolicy", err)
	}
}

func TestTST003RequiredIfEnabledContracts(t *testing.T) {
	t.Run("postgres_enabled_without_dsn_rejected", func(t *testing.T) {
		resetConfigEnv(t)
		t.Setenv("APP__POSTGRES__ENABLED", "true")

		_, _, err := LoadDetailed(LoadOptions{})
		if err == nil {
			t.Fatalf("LoadDetailed() expected secret policy error")
		}
		if !errors.Is(err, ErrSecretPolicy) {
			t.Fatalf("error = %v, want ErrSecretPolicy", err)
		}
	})

	t.Run("postgres_enabled_with_dsn_allowed", func(t *testing.T) {
		resetConfigEnv(t)
		dsn := "postgres://app:app@localhost:5432/app?sslmode=disable"
		t.Setenv("APP__POSTGRES__ENABLED", "true")
		t.Setenv("APP__POSTGRES__DSN", dsn)

		cfg, _, err := LoadDetailed(LoadOptions{})
		if err != nil {
			t.Fatalf("LoadDetailed() error = %v", err)
		}
		if !cfg.Postgres.Enabled {
			t.Fatalf("Postgres.Enabled = false, want true")
		}
		if cfg.Postgres.DSN != dsn {
			t.Fatalf("Postgres.DSN = %q, want %q", cfg.Postgres.DSN, dsn)
		}
	})

	t.Run("mongo_enabled_without_uri_rejected", func(t *testing.T) {
		resetConfigEnv(t)
		t.Setenv("APP__MONGO__ENABLED", "true")

		_, _, err := LoadDetailed(LoadOptions{})
		if err == nil {
			t.Fatalf("LoadDetailed() expected secret policy error")
		}
		if !errors.Is(err, ErrSecretPolicy) {
			t.Fatalf("error = %v, want ErrSecretPolicy", err)
		}
	})

	t.Run("mongo_enabled_without_database_rejected", func(t *testing.T) {
		resetConfigEnv(t)
		t.Setenv("APP__MONGO__URI", "mongodb://localhost:27017")
		configPath := writeTempConfig(t, `
mongo:
  enabled: true
  database: ""
`)

		_, _, err := LoadDetailed(LoadOptions{ConfigPath: configPath})
		if err == nil {
			t.Fatalf("LoadDetailed() expected validation error")
		}
		if !errors.Is(err, ErrValidate) {
			t.Fatalf("error = %v, want ErrValidate", err)
		}
	})

	t.Run("mongo_disabled_without_uri_allowed", func(t *testing.T) {
		resetConfigEnv(t)

		cfg, _, err := LoadDetailed(LoadOptions{})
		if err != nil {
			t.Fatalf("LoadDetailed() error = %v", err)
		}
		if cfg.Mongo.Enabled {
			t.Fatalf("Mongo.Enabled = true, want false")
		}
		if cfg.Mongo.URI != "" {
			t.Fatalf("Mongo.URI = %q, want empty", cfg.Mongo.URI)
		}
	})

	t.Run("redis_enabled_with_empty_addr_rejected", func(t *testing.T) {
		resetConfigEnv(t)
		configPath := writeTempConfig(t, `
redis:
  enabled: true
  addr: ""
`)

		_, _, err := LoadDetailed(LoadOptions{ConfigPath: configPath})
		if err == nil {
			t.Fatalf("LoadDetailed() expected validation error")
		}
		if !errors.Is(err, ErrValidate) {
			t.Fatalf("error = %v, want ErrValidate", err)
		}
	})
}

func TestRedisStoreGuardRejectsWithoutAllowFlag(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("APP__REDIS__ENABLED", "true")
	t.Setenv("APP__REDIS__ADDR", "127.0.0.1:6379")
	t.Setenv("APP__REDIS__MODE", "store")
	t.Setenv("APP__REDIS__ALLOW_STORE_MODE", "false")

	_, _, err := LoadDetailed(LoadOptions{})
	if err == nil {
		t.Fatalf("LoadDetailed() expected redis store guard rejection")
	}
	if !errors.Is(err, ErrValidate) {
		t.Fatalf("error = %v, want ErrValidate", err)
	}
	if !strings.Contains(err.Error(), "redis.allow_store_mode=true") {
		t.Fatalf("error = %v, want allow_store_mode hint", err)
	}
}

func TestRedisStoreGuardRequiresZeroStaleWindow(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("APP__REDIS__ENABLED", "true")
	t.Setenv("APP__REDIS__ADDR", "127.0.0.1:6379")
	t.Setenv("APP__REDIS__MODE", " STORE ")
	t.Setenv("APP__REDIS__ALLOW_STORE_MODE", "true")
	t.Setenv("APP__REDIS__STALE_WINDOW", "1s")

	_, _, err := LoadDetailed(LoadOptions{})
	if err == nil {
		t.Fatalf("LoadDetailed() expected stale-window validation error")
	}
	if !errors.Is(err, ErrValidate) {
		t.Fatalf("error = %v, want ErrValidate", err)
	}
}

func TestRedisStoreGuardAllowsConfiguredModeForV1GuardPath(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("APP__REDIS__ENABLED", "true")
	t.Setenv("APP__REDIS__ADDR", "127.0.0.1:6379")
	t.Setenv("APP__REDIS__MODE", " STORE ")
	t.Setenv("APP__REDIS__ALLOW_STORE_MODE", "true")
	t.Setenv("APP__REDIS__STALE_WINDOW", "0s")

	cfg, _, err := LoadDetailed(LoadOptions{})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v", err)
	}
	if cfg.Redis.Mode != "store" {
		t.Fatalf("Redis.Mode = %q, want store", cfg.Redis.Mode)
	}
	if !cfg.Redis.AllowStoreMode {
		t.Fatalf("Redis.AllowStoreMode = false, want true")
	}
	if cfg.Redis.StaleWindow != 0 {
		t.Fatalf("Redis.StaleWindow = %s, want 0", cfg.Redis.StaleWindow)
	}
}

func TestRedisModePolicyHelpers(t *testing.T) {
	t.Parallel()

	if got := (RedisConfig{Mode: " STORE "}).ModeValue(); got != RedisModeStore {
		t.Fatalf("ModeValue(STORE) = %q, want %q", got, RedisModeStore)
	}
	if !(RedisConfig{Mode: "store"}).StoreMode() {
		t.Fatal("StoreMode(store) = false, want true")
	}
	if got := (RedisConfig{Mode: "unexpected"}).ModeValue(); got != "unexpected" {
		t.Fatalf("ModeValue(unexpected) = %q, want unexpected", got)
	}
}

func TestConfigReadinessProbeRequiredPolicyHelpers(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		cfg          Config
		wantPostgres bool
		wantMongo    bool
		wantRedis    bool
	}{
		{
			name: "disabled dependencies ignore readiness flags",
			cfg: Config{
				FeatureFlags: FeatureFlagsConfig{
					PostgresReadinessProbe: true,
					MongoReadinessProbe:    true,
					RedisReadinessProbe:    true,
				},
			},
		},
		{
			name: "disabled redis store mode does not require readiness",
			cfg: Config{
				Redis: RedisConfig{Mode: RedisModeStore},
			},
		},
		{
			name: "enabled dependencies without readiness flags",
			cfg: Config{
				Postgres: PostgresConfig{Enabled: true},
				Mongo:    MongoConfig{Enabled: true},
				Redis:    RedisConfig{Enabled: true, Mode: RedisModeCache},
			},
		},
		{
			name: "enabled dependencies with readiness flags",
			cfg: Config{
				Postgres: PostgresConfig{Enabled: true},
				Mongo:    MongoConfig{Enabled: true},
				Redis:    RedisConfig{Enabled: true, Mode: RedisModeCache},
				FeatureFlags: FeatureFlagsConfig{
					PostgresReadinessProbe: true,
					MongoReadinessProbe:    true,
					RedisReadinessProbe:    true,
				},
			},
			wantPostgres: true,
			wantMongo:    true,
			wantRedis:    true,
		},
		{
			name: "enabled redis store mode requires readiness without flag",
			cfg: Config{
				Redis: RedisConfig{Enabled: true, Mode: RedisModeStore},
			},
			wantRedis: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := tc.cfg.PostgresReadinessProbeRequired(); got != tc.wantPostgres {
				t.Fatalf("PostgresReadinessProbeRequired() = %v, want %v", got, tc.wantPostgres)
			}
			if got := tc.cfg.MongoReadinessProbeRequired(); got != tc.wantMongo {
				t.Fatalf("MongoReadinessProbeRequired() = %v, want %v", got, tc.wantMongo)
			}
			if got := tc.cfg.RedisReadinessProbeRequired(); got != tc.wantRedis {
				t.Fatalf("RedisReadinessProbeRequired() = %v, want %v", got, tc.wantRedis)
			}
		})
	}
}

func TestConfigReadinessProbeBudgetsUseRequiredRuntimeProbes(t *testing.T) {
	t.Parallel()

	cfg := Config{
		HTTP: HTTPConfig{
			ReadinessTimeout: 10 * time.Second,
		},
		Postgres: PostgresConfig{
			Enabled:            true,
			HealthcheckTimeout: 2 * time.Second,
		},
		Redis: RedisConfig{
			Enabled:     true,
			Mode:        RedisModeStore,
			DialTimeout: 3 * time.Second,
		},
		Mongo: MongoConfig{
			Enabled:        true,
			ConnectTimeout: 4 * time.Second,
		},
		FeatureFlags: FeatureFlagsConfig{
			PostgresReadinessProbe: true,
			MongoReadinessProbe:    true,
		},
	}

	budgets := cfg.ReadinessProbeBudgets()
	want := []ReadinessProbeBudget{
		{ConfigKey: "postgres.healthcheck_timeout", Budget: 2 * time.Second},
		{ConfigKey: "redis.dial_timeout", Budget: 3 * time.Second},
		{ConfigKey: "mongo.connect_timeout", Budget: 4 * time.Second},
	}
	if len(budgets) != len(want) {
		t.Fatalf("ReadinessProbeBudgets() len = %d, want %d", len(budgets), len(want))
	}
	for i := range want {
		if budgets[i] != want[i] {
			t.Fatalf("ReadinessProbeBudgets()[%d] = %+v, want %+v", i, budgets[i], want[i])
		}
	}

	budgets[0].Budget = time.Nanosecond
	if got := cfg.ReadinessProbeBudgets()[0].Budget; got != 2*time.Second {
		t.Fatalf("ReadinessProbeBudgets() returned aliased slice; first budget = %s, want 2s", got)
	}
}

func TestLocalAllowsSymlinkConfig(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation can require elevated privileges on Windows")
	}

	resetConfigEnv(t)

	target := writeTempConfig(t, `
http:
  addr: ":18080"
`)

	linkPath := filepath.Join(t.TempDir(), "config-link.yaml")
	if err := os.Symlink(target, linkPath); err != nil {
		t.Fatalf("os.Symlink() error = %v", err)
	}

	cfg, _, err := LoadDetailed(LoadOptions{ConfigPath: linkPath})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v", err)
	}
	if cfg.HTTP.Addr != ":18080" {
		t.Fatalf("HTTP.Addr = %q, want :18080", cfg.HTTP.Addr)
	}
}

func TestNonLocalRejectsSymlinkConfig(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation can require elevated privileges on Windows")
	}

	resetConfigEnv(t)
	t.Setenv("APP__APP__ENV", "prod")

	target := writeTempConfig(t, `
http:
  addr: ":8080"
`)

	tempDir := t.TempDir()
	t.Setenv("APP_CONFIG_ALLOWED_ROOTS", tempDir)

	linkPath := filepath.Join(tempDir, "config-link.yaml")
	if err := os.Symlink(target, linkPath); err != nil {
		t.Fatalf("os.Symlink() error = %v", err)
	}

	_, _, err := LoadDetailed(LoadOptions{ConfigPath: linkPath})
	if err == nil {
		t.Fatalf("LoadDetailed() expected secret policy error for non-local symlink config")
	}
	if !errors.Is(err, ErrSecretPolicy) {
		t.Fatalf("error = %v, want ErrSecretPolicy", err)
	}
}

func TestTST005NonLocalRejectsWorldWritableConfig(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits are not reliable on Windows")
	}

	resetConfigEnv(t)
	t.Setenv("APP__APP__ENV", "prod")

	configPath := writeTempConfig(t, `
http:
  addr: ":8080"
`)
	t.Setenv("APP_CONFIG_ALLOWED_ROOTS", filepath.Dir(configPath))

	if err := os.Chmod(configPath, 0o666); err != nil {
		t.Fatalf("os.Chmod() error = %v", err)
	}

	_, _, err := LoadDetailed(LoadOptions{ConfigPath: configPath})
	if err == nil {
		t.Fatalf("LoadDetailed() expected secret policy error for world-writable config")
	}
	if !errors.Is(err, ErrSecretPolicy) {
		t.Fatalf("error = %v, want ErrSecretPolicy", err)
	}
}

func TestInvalidDurationParseError(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("APP__HTTP__READ_TIMEOUT", "oops")

	_, _, err := LoadDetailed(LoadOptions{})
	if err == nil {
		t.Fatalf("LoadDetailed() expected parse error")
	}
	if !errors.Is(err, ErrParse) {
		t.Fatalf("error = %v, want ErrParse", err)
	}
	if !strings.Contains(err.Error(), "invalid duration syntax") {
		t.Fatalf("error = %v, want sanitized duration parse detail", err)
	}
}

func TestParseErrorsExposeSanitizedDetail(t *testing.T) {
	tests := []struct {
		name       string
		envKey     string
		envValue   string
		wantDetail string
	}{
		{
			name:       "duration missing unit",
			envKey:     "APP__HTTP__READ_TIMEOUT",
			envValue:   "150",
			wantDetail: "missing duration unit",
		},
		{
			name:       "int format",
			envKey:     "APP__HTTP__MAX_HEADER_BYTES",
			envValue:   "many",
			wantDetail: "invalid integer format",
		},
		{
			name:       "float finite check",
			envKey:     "APP__OBSERVABILITY__OTEL__TRACES_SAMPLER_ARG",
			envValue:   "NaN",
			wantDetail: "non-finite numeric value",
		},
		{
			name:       "bool format",
			envKey:     "APP__FEATURE_FLAGS__REDIS_READINESS_PROBE",
			envValue:   "maybe",
			wantDetail: "invalid boolean format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetConfigEnv(t)
			t.Setenv(tt.envKey, tt.envValue)

			_, _, err := LoadDetailed(LoadOptions{})
			if err == nil {
				t.Fatal("LoadDetailed() error = nil, want parse error")
			}
			if !errors.Is(err, ErrParse) {
				t.Fatalf("error = %v, want ErrParse", err)
			}
			if !strings.Contains(err.Error(), tt.wantDetail) {
				t.Fatalf("error = %v, want sanitized detail %q", err, tt.wantDetail)
			}
			if strings.Contains(err.Error(), tt.envValue) {
				t.Fatalf("error = %v, leaked raw value %q", err, tt.envValue)
			}
		})
	}
}

func TestNonFiniteSamplerArgReturnsParseError(t *testing.T) {
	for _, value := range []string{"NaN", "+Inf"} {
		t.Run(value, func(t *testing.T) {
			resetConfigEnv(t)
			t.Setenv("APP__OBSERVABILITY__OTEL__TRACES_SAMPLER_ARG", value)

			_, _, err := LoadDetailed(LoadOptions{})
			if err == nil {
				t.Fatal("LoadDetailed() error = nil, want parse error")
			}
			if !errors.Is(err, ErrParse) {
				t.Fatalf("error = %v, want ErrParse", err)
			}
			if got := ErrorType(err); got != "parse" {
				t.Fatalf("ErrorType(error) = %q, want parse", got)
			}
		})
	}
}

func TestMalformedYAMLReturnsParseError(t *testing.T) {
	resetConfigEnv(t)

	configPath := writeTempConfig(t, `
http:
  addr: ":8080"
broken: [
`)

	_, _, err := LoadDetailed(LoadOptions{ConfigPath: configPath})
	if err == nil {
		t.Fatalf("LoadDetailed() expected parse error for malformed YAML")
	}
	if !errors.Is(err, ErrParse) {
		t.Fatalf("error = %v, want ErrParse", err)
	}
	if got := ErrorType(err); got != "parse" {
		t.Fatalf("ErrorType(error) = %q, want parse", got)
	}
}

func TestConfigFileWithoutEnvironmentHintFailsClosed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits are not reliable on Windows")
	}

	resetConfigEnv(t)
	t.Setenv("APP__APP__ENV", "")

	configPath := writeTempConfig(t, `
http:
  addr: ":8080"
`)
	t.Setenv("APP_CONFIG_ALLOWED_ROOTS", filepath.Dir(configPath))

	if err := os.Chmod(configPath, 0o666); err != nil {
		t.Fatalf("os.Chmod() error = %v", err)
	}

	_, _, err := LoadDetailed(LoadOptions{ConfigPath: configPath})
	if err == nil {
		t.Fatalf("LoadDetailed() expected fail-closed hardening error without explicit local env hint")
	}
	if !errors.Is(err, ErrSecretPolicy) {
		t.Fatalf("error = %v, want ErrSecretPolicy", err)
	}
}

func TestLoadConfigFileRejectsWhitespaceOnlyPath(t *testing.T) {
	t.Parallel()

	err := loadConfigFile(context.Background(), koanf.New(keyDelimiter), " \t\n ", configFilePolicyLocal)
	if err == nil {
		t.Fatal("loadConfigFile() error = nil, want non-nil")
	}
	if !errors.Is(err, ErrLoad) {
		t.Fatalf("error = %v, want ErrLoad", err)
	}
	if !strings.Contains(err.Error(), "empty config path") {
		t.Fatalf("error = %v, want empty config path detail", err)
	}
}

func TestNonLocalRejectsConfigOutsideAllowedRoots(t *testing.T) {
	resetConfigEnv(t)
	t.Setenv("APP__APP__ENV", "prod")

	allowedRoot := t.TempDir()
	t.Setenv("APP_CONFIG_ALLOWED_ROOTS", allowedRoot)

	otherRoot := t.TempDir()
	path := filepath.Join(otherRoot, "config.yaml")
	if err := os.WriteFile(path, []byte("http:\n  addr: \":8080\"\n"), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	_, _, err := LoadDetailed(LoadOptions{ConfigPath: path})
	if err == nil {
		t.Fatalf("LoadDetailed() expected allowed-root policy rejection")
	}
	if !errors.Is(err, ErrSecretPolicy) {
		t.Fatalf("error = %v, want ErrSecretPolicy", err)
	}
}

func TestNonLocalDefaultRootsDoNotAllowRepositoryConfigDir(t *testing.T) {
	resetConfigEnv(t)
	t.Setenv("APP__APP__ENV", "prod")
	t.Setenv("APP_CONFIG_ALLOWED_ROOTS", "")

	repoRoot := filepath.Join(t.TempDir(), "repo")
	repoConfigDir := filepath.Join(repoRoot, "env", "config")
	if err := os.MkdirAll(repoConfigDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}
	configPath := filepath.Join(repoConfigDir, "nonlocal-default-root-test.yaml")
	content := "app:\n  env: prod\nhttp:\n  addr: \":8080\"\n"
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	_, _, err := LoadDetailed(LoadOptions{ConfigPath: configPath})
	if err == nil {
		t.Fatalf("LoadDetailed() expected allowed-root policy rejection for repository config path in non-local mode")
	}
	if !errors.Is(err, ErrSecretPolicy) {
		t.Fatalf("error = %v, want ErrSecretPolicy", err)
	}
}

func TestNonLocalAllowedRootsRejectsUnsafeInputs(t *testing.T) {
	testCases := []struct {
		name             string
		allowedRoots     string
		wantErrorMessage string
	}{
		{
			name:             "relative root",
			allowedRoots:     "relative-config-root",
			wantErrorMessage: "APP_CONFIG_ALLOWED_ROOTS entries must be absolute paths",
		},
		{
			name:             "empty value uses default roots",
			allowedRoots:     "",
			wantErrorMessage: "outside allowed roots",
		},
		{
			name:             "delimiter only value produces no roots",
			allowedRoots:     ",;;",
			wantErrorMessage: "outside allowed roots",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resetConfigEnv(t)
			t.Setenv("APP__APP__ENV", "prod")
			t.Setenv("APP_CONFIG_ALLOWED_ROOTS", tc.allowedRoots)

			configPath := filepath.Join(t.TempDir(), "config.yaml")
			if err := os.WriteFile(configPath, []byte("http:\n  addr: \":8080\"\n"), 0o600); err != nil {
				t.Fatalf("os.WriteFile() error = %v", err)
			}

			_, _, err := LoadDetailed(LoadOptions{ConfigPath: configPath})
			if err == nil {
				t.Fatal("LoadDetailed() error = nil, want allowed-root policy rejection")
			}
			if !errors.Is(err, ErrSecretPolicy) {
				t.Fatalf("error = %v, want ErrSecretPolicy", err)
			}
			if !strings.Contains(err.Error(), tc.wantErrorMessage) {
				t.Fatalf("error = %v, want %q", err, tc.wantErrorMessage)
			}
		})
	}
}

func TestNonLocalRejectsSymlinkPathComponents(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation can require elevated privileges on Windows")
	}

	resetConfigEnv(t)
	t.Setenv("APP__APP__ENV", "prod")

	allowedRoot := t.TempDir()
	t.Setenv("APP_CONFIG_ALLOWED_ROOTS", allowedRoot)

	realRoot := t.TempDir()
	configPath := filepath.Join(realRoot, "config.yaml")
	if err := os.WriteFile(configPath, []byte("http:\n  addr: \":8080\"\n"), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	linkedDir := filepath.Join(allowedRoot, "linked")
	if err := os.Symlink(realRoot, linkedDir); err != nil {
		t.Fatalf("os.Symlink() error = %v", err)
	}

	pathViaSymlink := filepath.Join(linkedDir, "config.yaml")
	_, _, err := LoadDetailed(LoadOptions{ConfigPath: pathViaSymlink})
	if err == nil {
		t.Fatalf("LoadDetailed() expected symlink-path rejection")
	}
	if !errors.Is(err, ErrSecretPolicy) {
		t.Fatalf("error = %v, want ErrSecretPolicy", err)
	}
}

func TestNonLocalRejectsSecretLikeValuesInConfigFile(t *testing.T) {
	resetConfigEnv(t)
	t.Setenv("APP__APP__ENV", "prod")

	allowedRoot := t.TempDir()
	t.Setenv("APP_CONFIG_ALLOWED_ROOTS", allowedRoot)

	path := filepath.Join(allowedRoot, "config.yaml")
	content := `
app:
  env: prod
postgres:
  enabled: true
  dsn: "postgres://app:secret@localhost:5432/app?sslmode=disable"
`
	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	_, _, err := LoadDetailed(LoadOptions{ConfigPath: path})
	if err == nil {
		t.Fatalf("LoadDetailed() expected secret source policy rejection")
	}
	if !errors.Is(err, ErrSecretPolicy) {
		t.Fatalf("error = %v, want ErrSecretPolicy", err)
	}
}

func TestLocalRejectsSecretLikeValuesInConfigFile(t *testing.T) {
	resetConfigEnv(t)

	path := writeTempConfig(t, `
postgres:
  enabled: true
  dsn: "postgres://app:secret@localhost:5432/app?sslmode=disable"
`)

	_, _, err := LoadDetailed(LoadOptions{ConfigPath: path})
	if err == nil {
		t.Fatalf("LoadDetailed() expected secret source policy rejection")
	}
	if !errors.Is(err, ErrSecretPolicy) {
		t.Fatalf("error = %v, want ErrSecretPolicy", err)
	}
}

func TestConfigFileAllowsEmptySecretLikePlaceholders(t *testing.T) {
	resetConfigEnv(t)

	path := writeTempConfig(t, `
postgres:
  dsn: ""
mongo:
  uri: ""
redis:
  password: ""
observability:
  otel:
    exporter:
      otlp_headers: ""
`)

	if _, _, err := LoadDetailed(LoadOptions{ConfigPath: path}); err != nil {
		t.Fatalf("LoadDetailed() error = %v, want nil for empty secret-like placeholders", err)
	}
}

func TestConfigFileRejectsCommonFutureSecretLikeKeys(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantKey string
	}{
		{
			name: "client secret",
			content: `
oauth:
  client_secret: "secret"
`,
			wantKey: "oauth.client_secret",
		},
		{
			name: "jwt secret",
			content: `
security:
  jwt_secret: "secret"
`,
			wantKey: "security.jwt_secret",
		},
		{
			name: "api key",
			content: `
provider:
  api_key: "secret"
`,
			wantKey: "provider.api_key",
		},
		{
			name: "private key",
			content: `
tls:
  private_key: "secret"
`,
			wantKey: "tls.private_key",
		},
		{
			name: "top level token",
			content: `
token: "secret"
`,
			wantKey: "token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetConfigEnv(t)

			path := writeTempConfig(t, tt.content)
			_, _, err := LoadDetailed(LoadOptions{ConfigPath: path})
			if err == nil {
				t.Fatalf("LoadDetailed() expected secret policy rejection for %s", tt.wantKey)
			}
			if !errors.Is(err, ErrSecretPolicy) {
				t.Fatalf("error = %v, want ErrSecretPolicy", err)
			}
			if !strings.Contains(err.Error(), tt.wantKey) {
				t.Fatalf("error = %v, want rejected key %q", err, tt.wantKey)
			}
		})
	}
}

func TestSecretLikeConfigKeyPolicyAllowsNonSecretShapes(t *testing.T) {
	keys := []string{
		"http.addr",
		"redis.key_prefix",
		"feature_flags.redis_readiness_probe",
		"metadata.public_key",
	}

	for _, key := range keys {
		if isSecretLikeConfigKey(key) {
			t.Fatalf("isSecretLikeConfigKey(%q) = true, want false", key)
		}
	}
}

func TestParseErrorDoesNotLeakRawValue(t *testing.T) {
	resetConfigEnv(t)

	secretLikeValue := "supersecret-token-value"
	t.Setenv("APP__HTTP__READ_TIMEOUT", secretLikeValue)

	_, _, err := LoadDetailed(LoadOptions{})
	if err == nil {
		t.Fatalf("LoadDetailed() expected parse error")
	}
	if !errors.Is(err, ErrParse) {
		t.Fatalf("error = %v, want ErrParse", err)
	}
	if strings.Contains(err.Error(), secretLikeValue) {
		t.Fatalf("error unexpectedly contains raw secret-like value: %v", err)
	}
	if strings.Contains(err.Error(), "time: invalid duration") {
		t.Fatalf("error unexpectedly wraps raw time.ParseDuration detail: %v", err)
	}
}

func TestFlatPostgresDSNIsIgnored(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("POSTGRES_DSN", "postgres://app:app@localhost:5432/app?sslmode=disable")

	cfg, _, err := LoadDetailed(LoadOptions{})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v", err)
	}
	if cfg.Postgres.Enabled {
		t.Fatalf("Postgres.Enabled = true, want false when only flat key is set")
	}
	if cfg.Postgres.DSN != "" {
		t.Fatalf("Postgres.DSN = %q, want empty when only flat key is set", cfg.Postgres.DSN)
	}
}

func TestErrorTypeMapping(t *testing.T) {
	if got := ErrorType(nil); got != "" {
		t.Fatalf("ErrorType(nil) = %q, want empty", got)
	}
	if got := ErrorType(ErrStrictUnknownKey); got != "strict_unknown_key" {
		t.Fatalf("ErrorType(strict) = %q", got)
	}
	if got := ErrorType(ErrSecretPolicy); got != "secret_policy" {
		t.Fatalf("ErrorType(secret_policy) = %q", got)
	}
	if got := ErrorType(ErrValidate); got != "validate" {
		t.Fatalf("ErrorType(validate) = %q", got)
	}
	if got := ErrorType(ErrParse); got != "parse" {
		t.Fatalf("ErrorType(parse) = %q", got)
	}
	if got := ErrorType(ErrLoad); got != "load" {
		t.Fatalf("ErrorType(load) = %q", got)
	}
	if got := ErrorType(errors.New("new config error class")); got != "unknown" {
		t.Fatalf("ErrorType(unknown) = %q, want unknown", got)
	}
}

func TestLoadDetailedWithContextCanceled(t *testing.T) {
	resetConfigEnv(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := LoadDetailedWithContext(ctx, LoadOptions{})
	if err == nil {
		t.Fatalf("LoadDetailedWithContext() expected context cancellation error")
	}
	if !errors.Is(err, ErrLoad) {
		t.Fatalf("error = %v, want ErrLoad", err)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}

func TestLoadDetailedFailedStageReporting(t *testing.T) {
	t.Run("parse_stage", func(t *testing.T) {
		resetConfigEnv(t)
		t.Setenv("APP__HTTP__READ_TIMEOUT", "oops")

		_, report, err := LoadDetailed(LoadOptions{})
		if err == nil {
			t.Fatalf("LoadDetailed() expected parse error")
		}
		if !errors.Is(err, ErrParse) {
			t.Fatalf("error = %v, want ErrParse", err)
		}
		if report.FailedStage != StageParse {
			t.Fatalf("FailedStage = %q, want %q", report.FailedStage, StageParse)
		}
		if report.FailedStageDuration <= 0 {
			t.Fatalf("FailedStageDuration = %s, want > 0", report.FailedStageDuration)
		}
	})

	t.Run("validate_stage", func(t *testing.T) {
		resetConfigEnv(t)
		configPath := writeTempConfig(t, `
unknown:
  field: value
`)

		_, report, err := LoadDetailed(LoadOptions{
			ConfigPath: configPath,
			Strict:     true,
		})
		if err == nil {
			t.Fatalf("LoadDetailed() expected strict unknown key error")
		}
		if !errors.Is(err, ErrStrictUnknownKey) {
			t.Fatalf("error = %v, want ErrStrictUnknownKey", err)
		}
		if report.FailedStage != StageValidate {
			t.Fatalf("FailedStage = %q, want %q", report.FailedStage, StageValidate)
		}
		if report.FailedStageDuration <= 0 {
			t.Fatalf("FailedStageDuration = %s, want > 0", report.FailedStageDuration)
		}
	})

	t.Run("load_file_stage", func(t *testing.T) {
		resetConfigEnv(t)
		t.Setenv("APP__APP__ENV", "prod")
		t.Setenv("APP_CONFIG_ALLOWED_ROOTS", t.TempDir())

		_, report, err := LoadDetailed(LoadOptions{ConfigPath: "/nonexistent/config.yaml"})
		if err == nil {
			t.Fatalf("LoadDetailed() expected load error")
		}
		if !errors.Is(err, ErrLoad) && !errors.Is(err, ErrSecretPolicy) {
			t.Fatalf("error = %v, want ErrLoad or ErrSecretPolicy", err)
		}
		if report.FailedStage != StageLoadFile {
			t.Fatalf("FailedStage = %q, want %q", report.FailedStage, StageLoadFile)
		}
		if report.FailedStageDuration <= 0 {
			t.Fatalf("FailedStageDuration = %s, want > 0", report.FailedStageDuration)
		}
	})

	t.Run("validate_context_stage", func(t *testing.T) {
		resetConfigEnv(t)

		_, report, err := LoadDetailed(LoadOptions{ValidateBudget: time.Nanosecond})
		if err == nil {
			t.Fatalf("LoadDetailed() expected validate context deadline error")
		}
		if !errors.Is(err, ErrValidate) {
			t.Fatalf("error = %v, want ErrValidate", err)
		}
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("error = %v, want context.DeadlineExceeded", err)
		}
		if got := ErrorType(err); got != "validate" {
			t.Fatalf("ErrorType(error) = %q, want validate", got)
		}
		if report.FailedStage != StageValidate {
			t.Fatalf("FailedStage = %q, want %q", report.FailedStage, StageValidate)
		}
		if report.FailedStageDuration <= 0 {
			t.Fatalf("FailedStageDuration = %s, want > 0", report.FailedStageDuration)
		}
	})
}

func TestOTLPExporterValuesFromNamespaceEnv(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("APP__OBSERVABILITY__OTEL__EXPORTER__OTLP_ENDPOINT", "https://otel.example.com:4318")
	t.Setenv("APP__OBSERVABILITY__OTEL__EXPORTER__OTLP_TRACES_ENDPOINT", "https://otel.example.com:4318/v1/traces")
	t.Setenv("APP__OBSERVABILITY__OTEL__EXPORTER__OTLP_HEADERS", "authorization=Bearer token")
	t.Setenv("APP__OBSERVABILITY__OTEL__EXPORTER__OTLP_PROTOCOL", "http/protobuf")

	cfg, _, err := LoadDetailed(LoadOptions{})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v", err)
	}
	if cfg.Observability.OTel.Exporter.OTLPEndpoint != "https://otel.example.com:4318" {
		t.Fatalf("OTLPEndpoint = %q, want %q", cfg.Observability.OTel.Exporter.OTLPEndpoint, "https://otel.example.com:4318")
	}
	if cfg.Observability.OTel.Exporter.OTLPTracesEndpoint != "https://otel.example.com:4318/v1/traces" {
		t.Fatalf("OTLPTracesEndpoint = %q, want %q", cfg.Observability.OTel.Exporter.OTLPTracesEndpoint, "https://otel.example.com:4318/v1/traces")
	}
	if cfg.Observability.OTel.Exporter.OTLPHeaders != "authorization=Bearer token" {
		t.Fatalf("OTLPHeaders = %q, want %q", cfg.Observability.OTel.Exporter.OTLPHeaders, "authorization=Bearer token")
	}
	if cfg.Observability.OTel.Exporter.OTLPProtocol != "http/protobuf" {
		t.Fatalf("OTLPProtocol = %q, want %q", cfg.Observability.OTel.Exporter.OTLPProtocol, "http/protobuf")
	}
}

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	return path
}

func readEnvExample(t *testing.T, path string) map[string]string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", path, err)
	}

	values := make(map[string]string)
	for lineNumber, rawLine := range strings.Split(string(content), "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			t.Fatalf("%s:%d is not KEY=VALUE", path, lineNumber+1)
		}
		key = strings.TrimSpace(key)
		if key == "" {
			t.Fatalf("%s:%d has an empty env key", path, lineNumber+1)
		}
		if !strings.HasPrefix(key, namespacePrefix) {
			continue
		}
		values[key] = strings.TrimSpace(value)
	}

	return values
}

func resetConfigEnv(t *testing.T) {
	t.Helper()

	previousValues := make(map[string]string)
	for _, key := range configEnvResetKeys() {
		if value, ok := os.LookupEnv(key); ok {
			previousValues[key] = value
			t.Setenv(key, value)
		}
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("os.Unsetenv(%q) error = %v", key, err)
		}
	}
	t.Cleanup(func() {
		for _, key := range configEnvResetKeys() {
			if _, ok := previousValues[key]; !ok {
				_ = os.Unsetenv(key)
			}
		}
	})
	t.Setenv("APP__APP__ENV", "local")
}

func configEnvResetKeys() []string {
	knownKeys := knownConfigKeys()
	knownSections := knownConfigSections()
	keySet := make(map[string]struct{}, len(knownKeys)+len(knownSections)+1)
	for key := range knownKeys {
		keySet[namespaceEnvForConfigKey(key)] = struct{}{}
	}
	for key := range knownSections {
		keySet[namespaceEnvForConfigKey(key)] = struct{}{}
	}
	keySet[allowedConfigRootsEnvVar] = struct{}{}

	keys := make([]string, 0, len(keySet))
	for key := range keySet {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func TestBuildSnapshotMapsEveryKnownConfigLeafKey(t *testing.T) {
	t.Parallel()

	sourceValues := sentinelConfigSourceValues()
	knownKeys := sortedStringSetKeys(knownConfigKeys())
	sourceKeys := sortedStringSetKeys(sourceValues)
	if !reflect.DeepEqual(sourceKeys, knownKeys) {
		t.Fatalf("sentinel source keys = %v, want known config keys %v", sourceKeys, knownKeys)
	}

	k := koanf.New(keyDelimiter)
	if err := k.Load(confmap.Provider(sourceValues, keyDelimiter), nil); err != nil {
		t.Fatalf("load sentinel config source: %v", err)
	}

	cfg, err := buildSnapshot(k)
	if err != nil {
		t.Fatalf("buildSnapshot() error = %v", err)
	}

	observedValues := flattenConfigSnapshotValues(t, reflect.ValueOf(cfg), "")
	observedKeys := sortedStringSetKeys(observedValues)
	if !reflect.DeepEqual(observedKeys, knownKeys) {
		t.Fatalf("flattened Config keys = %v, want known config keys %v", observedKeys, knownKeys)
	}

	expectedValues := expectedSentinelSnapshotValues()
	expectedKeys := sortedStringSetKeys(expectedValues)
	if !reflect.DeepEqual(expectedKeys, knownKeys) {
		t.Fatalf("expected sentinel keys = %v, want known config keys %v", expectedKeys, knownKeys)
	}

	for _, key := range knownKeys {
		if got, want := observedValues[key], expectedValues[key]; !reflect.DeepEqual(got, want) {
			t.Fatalf("buildSnapshot() value for %s = %#v (%T), want %#v (%T)", key, got, got, want, want)
		}
	}
}

func TestKnownConfigKeysMatchSnapshotTags(t *testing.T) {
	t.Parallel()

	knownKeys := sortedStringSetKeys(knownConfigKeys())

	tagKeys := configLeafKeysFromType(t, reflect.TypeFor[Config](), "")
	sort.Strings(tagKeys)
	if !reflect.DeepEqual(knownKeys, tagKeys) {
		t.Fatalf("knownConfigKeys() = %v, want Config koanf leaf keys %v", knownKeys, tagKeys)
	}
}

func TestKnownConfigSectionsMatchSnapshotTags(t *testing.T) {
	t.Parallel()

	knownSections := sortedStringSetKeys(knownConfigSections())

	tagSections := configSectionKeysFromType(t, reflect.TypeFor[Config](), "")
	sort.Strings(tagSections)
	if !reflect.DeepEqual(knownSections, tagSections) {
		t.Fatalf("knownConfigSections() = %v, want Config koanf section keys %v", knownSections, tagSections)
	}
}

func TestDefaultValuesAreSubsetOfKnownConfigKeys(t *testing.T) {
	t.Parallel()

	knownKeys := knownConfigKeys()
	for key := range defaultValues() {
		if _, ok := knownKeys[key]; !ok {
			t.Fatalf("defaultValues() contains %q, which is not a known Config koanf leaf key", key)
		}
	}
}

func TestDefaultConfigYAMLMatchesCodeDefaults(t *testing.T) {
	t.Parallel()

	defaultKoanf := koanf.New(keyDelimiter)
	if err := defaultKoanf.Load(confmap.Provider(defaultValues(), keyDelimiter), nil); err != nil {
		t.Fatalf("load code defaults: %v", err)
	}
	defaultSnapshot, err := buildSnapshot(defaultKoanf)
	if err != nil {
		t.Fatalf("buildSnapshot(defaultValues()) error = %v", err)
	}

	yamlKoanf := koanf.New(keyDelimiter)
	path := filepath.Join("..", "..", "env", "config", "default.yaml")
	if err := loadConfigFile(context.Background(), yamlKoanf, path, configFilePolicyLocal); err != nil {
		t.Fatalf("load default config yaml: %v", err)
	}

	defaultKeys := sortedStringSetKeys(defaultValues())
	yamlKeys := yamlKoanf.Keys()
	sort.Strings(yamlKeys)
	if !reflect.DeepEqual(yamlKeys, defaultKeys) {
		t.Fatalf("env/config/default.yaml keys = %v, want code default keys %v", yamlKeys, defaultKeys)
	}

	yamlSnapshot, err := buildSnapshot(yamlKoanf)
	if err != nil {
		t.Fatalf("buildSnapshot(env/config/default.yaml) error = %v", err)
	}
	if !reflect.DeepEqual(yamlSnapshot, defaultSnapshot) {
		t.Fatalf("env/config/default.yaml snapshot = %+v, want code defaults %+v", yamlSnapshot, defaultSnapshot)
	}
}

func configLeafKeysFromType(t *testing.T, typ reflect.Type, prefix string) []string {
	t.Helper()

	if typ.Kind() != reflect.Struct {
		t.Fatalf("configLeafKeysFromType(%s) called with non-struct type", typ)
	}

	keys := make([]string, 0)
	for field := range typ.Fields() {
		tag := strings.TrimSpace(field.Tag.Get("koanf"))
		if tag == "" || tag == "-" {
			t.Fatalf("%s.%s must declare a concrete koanf tag", typ.Name(), field.Name)
		}

		key := tag
		if prefix != "" {
			key = prefix + keyDelimiter + tag
		}

		if hasKoanfTaggedFields(field.Type) {
			keys = append(keys, configLeafKeysFromType(t, field.Type, key)...)
			continue
		}
		keys = append(keys, key)
	}
	return keys
}

func configSectionKeysFromType(t *testing.T, typ reflect.Type, prefix string) []string {
	t.Helper()

	if typ.Kind() != reflect.Struct {
		t.Fatalf("configSectionKeysFromType(%s) called with non-struct type", typ)
	}

	keys := make([]string, 0)
	for field := range typ.Fields() {
		tag := strings.TrimSpace(field.Tag.Get("koanf"))
		if tag == "" || tag == "-" {
			t.Fatalf("%s.%s must declare a concrete koanf tag", typ.Name(), field.Name)
		}

		key := tag
		if prefix != "" {
			key = prefix + keyDelimiter + tag
		}

		if !hasKoanfTaggedFields(field.Type) {
			continue
		}
		keys = append(keys, key)
		keys = append(keys, configSectionKeysFromType(t, field.Type, key)...)
	}
	return keys
}

func hasKoanfTaggedFields(typ reflect.Type) bool {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return false
	}
	for field := range typ.Fields() {
		if strings.TrimSpace(field.Tag.Get("koanf")) != "" {
			return true
		}
	}
	return false
}

func flattenConfigSnapshotValues(t *testing.T, value reflect.Value, prefix string) map[string]any {
	t.Helper()

	for value.Kind() == reflect.Pointer {
		if value.IsNil() {
			t.Fatalf("flattenConfigSnapshotValues(%s) got nil pointer", value.Type())
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		t.Fatalf("flattenConfigSnapshotValues(%s) called with non-struct value", value.Type())
	}

	typ := value.Type()
	values := make(map[string]any)
	for field := range typ.Fields() {
		tag := strings.TrimSpace(field.Tag.Get("koanf"))
		if tag == "" || tag == "-" {
			t.Fatalf("%s.%s must declare a concrete koanf tag", typ.Name(), field.Name)
		}

		key := tag
		if prefix != "" {
			key = prefix + keyDelimiter + tag
		}

		fieldValue := value.FieldByIndex(field.Index)
		if hasKoanfTaggedFields(field.Type) {
			maps.Copy(values, flattenConfigSnapshotValues(t, fieldValue, key))
			continue
		}
		values[key] = fieldValue.Interface()
	}
	return values
}

func sentinelConfigSourceValues() map[string]any {
	return map[string]any{
		"app.env":     "stage",
		"app.version": "v-snapshot-test",

		"http.addr":                        ":18080",
		"http.shutdown_timeout":            "31s",
		"http.readiness_timeout":           "4s",
		"http.readiness_propagation_delay": "16s",
		"http.read_header_timeout":         "6s",
		"http.read_timeout":                "7s",
		"http.write_timeout":               "11s",
		"http.idle_timeout":                "61s",
		"http.max_header_bytes":            20 << 10,
		"http.max_body_bytes":              int64(2 << 20),

		"log.level": "warn",

		"postgres.enabled":             true,
		"postgres.dsn":                 "postgres://app:secret@db:5432/app?sslmode=disable",
		"postgres.connect_timeout":     "17s",
		"postgres.healthcheck_timeout": "18s",
		"postgres.max_open_conns":      26,
		"postgres.max_idle_conns":      11,
		"postgres.conn_max_lifetime":   "45m",

		"redis.enabled":                  true,
		"redis.mode":                     "cache",
		"redis.allow_store_mode":         true,
		"redis.addr":                     "127.0.0.1:6380",
		"redis.username":                 "redis-user",
		"redis.password":                 "redis-secret",
		"redis.db":                       2,
		"redis.dial_timeout":             "8s",
		"redis.read_timeout":             "9s",
		"redis.write_timeout":            "10s",
		"redis.pool_size":                21,
		"redis.key_prefix":               "snapshot",
		"redis.fresh_ttl":                "70s",
		"redis.stale_window":             "71s",
		"redis.negative_ttl":             "72s",
		"redis.ttl_jitter_percent":       12,
		"redis.enable_singleflight":      false,
		"redis.max_fallback_concurrency": 33,

		"mongo.enabled":                  true,
		"mongo.uri":                      "mongodb://localhost:27017",
		"mongo.database":                 "snapshot_app",
		"mongo.connect_timeout":          "73s",
		"mongo.server_selection_timeout": "74s",
		"mongo.max_pool_size":            101,

		"observability.otel.service_name":                  "snapshot-service",
		"observability.otel.traces_sampler":                "always_on",
		"observability.otel.traces_sampler_arg":            0.25,
		"observability.otel.exporter.otlp_endpoint":        "https://otel.example.com:4318",
		"observability.otel.exporter.otlp_traces_endpoint": "https://otel.example.com:4318/v1/traces",
		"observability.otel.exporter.otlp_headers":         "authorization=Bearer snapshot",
		"observability.otel.exporter.otlp_protocol":        "grpc",

		"feature_flags.postgres_readiness_probe": false,
		"feature_flags.mongo_readiness_probe":    true,
		"feature_flags.redis_readiness_probe":    true,
	}
}

func expectedSentinelSnapshotValues() map[string]any {
	return map[string]any{
		"app.env":     "stage",
		"app.version": "v-snapshot-test",

		"http.addr":                        ":18080",
		"http.shutdown_timeout":            31 * time.Second,
		"http.readiness_timeout":           4 * time.Second,
		"http.readiness_propagation_delay": 16 * time.Second,
		"http.read_header_timeout":         6 * time.Second,
		"http.read_timeout":                7 * time.Second,
		"http.write_timeout":               11 * time.Second,
		"http.idle_timeout":                61 * time.Second,
		"http.max_header_bytes":            20 << 10,
		"http.max_body_bytes":              int64(2 << 20),

		"log.level": slog.LevelWarn,

		"postgres.enabled":             true,
		"postgres.dsn":                 "postgres://app:secret@db:5432/app?sslmode=disable",
		"postgres.connect_timeout":     17 * time.Second,
		"postgres.healthcheck_timeout": 18 * time.Second,
		"postgres.max_open_conns":      26,
		"postgres.max_idle_conns":      11,
		"postgres.conn_max_lifetime":   45 * time.Minute,

		"redis.enabled":                  true,
		"redis.mode":                     "cache",
		"redis.allow_store_mode":         true,
		"redis.addr":                     "127.0.0.1:6380",
		"redis.username":                 "redis-user",
		"redis.password":                 "redis-secret",
		"redis.db":                       2,
		"redis.dial_timeout":             8 * time.Second,
		"redis.read_timeout":             9 * time.Second,
		"redis.write_timeout":            10 * time.Second,
		"redis.pool_size":                21,
		"redis.key_prefix":               "snapshot",
		"redis.fresh_ttl":                70 * time.Second,
		"redis.stale_window":             71 * time.Second,
		"redis.negative_ttl":             72 * time.Second,
		"redis.ttl_jitter_percent":       12,
		"redis.enable_singleflight":      false,
		"redis.max_fallback_concurrency": 33,

		"mongo.enabled":                  true,
		"mongo.uri":                      "mongodb://localhost:27017",
		"mongo.database":                 "snapshot_app",
		"mongo.connect_timeout":          73 * time.Second,
		"mongo.server_selection_timeout": 74 * time.Second,
		"mongo.max_pool_size":            101,

		"observability.otel.service_name":                  "snapshot-service",
		"observability.otel.traces_sampler":                "always_on",
		"observability.otel.traces_sampler_arg":            0.25,
		"observability.otel.exporter.otlp_endpoint":        "https://otel.example.com:4318",
		"observability.otel.exporter.otlp_traces_endpoint": "https://otel.example.com:4318/v1/traces",
		"observability.otel.exporter.otlp_headers":         "authorization=Bearer snapshot",
		"observability.otel.exporter.otlp_protocol":        "grpc",

		"feature_flags.postgres_readiness_probe": false,
		"feature_flags.mongo_readiness_probe":    true,
		"feature_flags.redis_readiness_probe":    true,
	}
}

func sortedStringSetKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func TestPostgresDurationBounds(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("APP__POSTGRES__CONNECT_TIMEOUT", "50ms")

	_, _, err := LoadDetailed(LoadOptions{})
	if err == nil {
		t.Fatalf("LoadDetailed() expected validation error for connect timeout")
	}
	if !errors.Is(err, ErrValidate) {
		t.Fatalf("error = %v, want ErrValidate", err)
	}
}

func TestShutdownTimeoutCanBeTunedWhenDrainBudgetIsValid(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("APP__HTTP__SHUTDOWN_TIMEOUT", "45s")
	t.Setenv("APP__HTTP__READINESS_PROPAGATION_DELAY", "20s")
	t.Setenv("APP__HTTP__WRITE_TIMEOUT", "10s")

	cfg, _, err := LoadDetailed(LoadOptions{})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v, want nil for tuned shutdown timeout", err)
	}
	if cfg.HTTP.ShutdownTimeout != 45*time.Second {
		t.Fatalf("HTTP.ShutdownTimeout = %s, want 45s", cfg.HTTP.ShutdownTimeout)
	}
}

func TestShutdownTimeoutMustStayWithinRange(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("APP__HTTP__SHUTDOWN_TIMEOUT", "500ms")
	t.Setenv("APP__HTTP__READINESS_PROPAGATION_DELAY", "0s")
	t.Setenv("APP__HTTP__WRITE_TIMEOUT", "100ms")

	_, _, err := LoadDetailed(LoadOptions{})
	if err == nil {
		t.Fatalf("LoadDetailed() expected validation error for shutdown timeout range")
	}
	if !errors.Is(err, ErrValidate) {
		t.Fatalf("error = %v, want ErrValidate", err)
	}
	if !strings.Contains(err.Error(), "http.shutdown_timeout must be in range") {
		t.Fatalf("error = %v, want shutdown timeout range policy", err)
	}
}

func TestHTTPShutdownBudgetMustLeaveWriteDrainTime(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("APP__HTTP__READINESS_PROPAGATION_DELAY", "25s")
	t.Setenv("APP__HTTP__WRITE_TIMEOUT", "10s")

	_, _, err := LoadDetailed(LoadOptions{})
	if err == nil {
		t.Fatalf("LoadDetailed() expected validation error for write timeout beyond drain budget")
	}
	if !errors.Is(err, ErrValidate) {
		t.Fatalf("error = %v, want ErrValidate", err)
	}
	if !strings.Contains(err.Error(), "http.write_timeout must be <= effective drain budget") {
		t.Fatalf("error = %v, want explicit drain budget policy", err)
	}
}

func TestReadinessTimeoutMustNotExceedWriteTimeout(t *testing.T) {
	t.Run("greater readiness timeout rejects", func(t *testing.T) {
		resetConfigEnv(t)
		t.Setenv("APP__HTTP__READINESS_TIMEOUT", "6s")
		t.Setenv("APP__HTTP__WRITE_TIMEOUT", "5s")

		_, _, err := LoadDetailed(LoadOptions{})
		if err == nil {
			t.Fatalf("LoadDetailed() expected validation error for readiness timeout beyond write timeout")
		}
		if !errors.Is(err, ErrValidate) {
			t.Fatalf("error = %v, want ErrValidate", err)
		}
		if !strings.Contains(err.Error(), "http.readiness_timeout must be <= http.write_timeout") {
			t.Fatalf("error = %v, want readiness/write timeout compatibility policy", err)
		}
	})

	for _, tc := range []struct {
		name             string
		readinessTimeout string
		writeTimeout     string
	}{
		{name: "equal timeout allows", readinessTimeout: "5s", writeTimeout: "5s"},
		{name: "lower readiness timeout allows", readinessTimeout: "4s", writeTimeout: "5s"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resetConfigEnv(t)
			t.Setenv("APP__HTTP__READINESS_TIMEOUT", tc.readinessTimeout)
			t.Setenv("APP__HTTP__WRITE_TIMEOUT", tc.writeTimeout)

			_, _, err := LoadDetailed(LoadOptions{})
			if err != nil {
				t.Fatalf("LoadDetailed() error = %v", err)
			}
		})
	}
}

func TestReadinessTimeoutMustCoverAggregateEnabledProbeBudget(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("APP__HTTP__READINESS_TIMEOUT", "6s")
	t.Setenv("APP__POSTGRES__ENABLED", "true")
	t.Setenv("APP__POSTGRES__DSN", "postgres://user:pass@localhost:5432/app?sslmode=disable")
	t.Setenv("APP__POSTGRES__HEALTHCHECK_TIMEOUT", "3s")
	t.Setenv("APP__REDIS__ENABLED", "true")
	t.Setenv("APP__REDIS__MODE", "store")
	t.Setenv("APP__REDIS__ALLOW_STORE_MODE", "true")
	t.Setenv("APP__REDIS__DIAL_TIMEOUT", "2s")
	t.Setenv("APP__MONGO__ENABLED", "true")
	t.Setenv("APP__MONGO__URI", "mongodb://localhost:27017/app")
	t.Setenv("APP__MONGO__CONNECT_TIMEOUT", "2s")
	t.Setenv("APP__FEATURE_FLAGS__MONGO_READINESS_PROBE", "true")

	_, _, err := LoadDetailed(LoadOptions{})
	if err == nil {
		t.Fatalf("LoadDetailed() expected validation error for aggregate readiness timeout")
	}
	if !errors.Is(err, ErrValidate) {
		t.Fatalf("error = %v, want ErrValidate", err)
	}
	if !strings.Contains(err.Error(), "aggregate sequential readiness probe budget") {
		t.Fatalf("error = %v, want aggregate readiness dependency budget policy", err)
	}
	if !strings.Contains(err.Error(), "postgres.healthcheck_timeout + redis.dial_timeout + mongo.connect_timeout") {
		t.Fatalf("error = %v, want enabled readiness probe names", err)
	}
}

func TestPostgresDSNParseIsAdapterOwned(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("APP__POSTGRES__ENABLED", "true")
	t.Setenv("APP__POSTGRES__DSN", "postgres://%zz")

	cfg, _, err := LoadDetailed(LoadOptions{})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v, want nil because driver-specific parsing is adapter-owned", err)
	}
	if cfg.Postgres.DSN != "postgres://%zz" {
		t.Fatalf("Postgres.DSN = %q, want raw invalid DSN preserved for adapter-owned parsing", cfg.Postgres.DSN)
	}
}

func TestRedisEnabledRequiresHostPortAddress(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("APP__REDIS__ENABLED", "true")
	t.Setenv("APP__REDIS__ADDR", "redis-without-port")

	_, _, err := LoadDetailed(LoadOptions{})
	if err == nil {
		t.Fatalf("LoadDetailed() expected validation error for redis addr without port")
	}
	if !errors.Is(err, ErrValidate) {
		t.Fatalf("error = %v, want ErrValidate", err)
	}
}

func TestRedisEnabledRejectsEmptyHostAddress(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("APP__REDIS__ENABLED", "true")
	t.Setenv("APP__REDIS__ADDR", ":6379")

	_, _, err := LoadDetailed(LoadOptions{})
	if err == nil {
		t.Fatalf("LoadDetailed() expected validation error for redis addr with empty host")
	}
	if !errors.Is(err, ErrValidate) {
		t.Fatalf("error = %v, want ErrValidate", err)
	}
	if !strings.Contains(err.Error(), "redis.addr must include non-empty host") {
		t.Fatalf("error = %v, want empty redis host policy", err)
	}
}

func TestRedisEnabledRequiresNumericTCPPort(t *testing.T) {
	for _, addr := range []string{"127.0.0.1:notaport", "127.0.0.1:0", "127.0.0.1:65536"} {
		t.Run(addr, func(t *testing.T) {
			resetConfigEnv(t)
			t.Setenv("APP__REDIS__ENABLED", "true")
			t.Setenv("APP__REDIS__ADDR", addr)

			_, _, err := LoadDetailed(LoadOptions{})
			if err == nil {
				t.Fatal("LoadDetailed() error = nil, want validation error")
			}
			if !errors.Is(err, ErrValidate) {
				t.Fatalf("error = %v, want ErrValidate", err)
			}
		})
	}
}

func TestMongoURIMustContainValidProbeTarget(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("APP__MONGO__ENABLED", "true")
	t.Setenv("APP__MONGO__URI", "mongo://bad-uri")
	t.Setenv("APP__MONGO__DATABASE", "app")

	_, _, err := LoadDetailed(LoadOptions{})
	if err == nil {
		t.Fatalf("LoadDetailed() expected validation error for mongo uri without a valid probe target")
	}
	if !errors.Is(err, ErrValidate) {
		t.Fatalf("error = %v, want ErrValidate", err)
	}
	if !strings.Contains(err.Error(), "mongo.uri must contain a valid probe target") {
		t.Fatalf("error = %v, want mongo probe-target detail", err)
	}
}

func TestMongoURIRejectsSeedlists(t *testing.T) {
	resetConfigEnv(t)

	rawURI := "mongodb://user:secret@mongo-a.example.com:27017,mongo-b.example.com:27017/app"
	t.Setenv("APP__MONGO__ENABLED", "true")
	t.Setenv("APP__MONGO__URI", rawURI)
	t.Setenv("APP__MONGO__DATABASE", "app")

	_, _, err := LoadDetailed(LoadOptions{})
	if err == nil {
		t.Fatalf("LoadDetailed() expected validation error for mongo seedlist")
	}
	if !errors.Is(err, ErrValidate) {
		t.Fatalf("error = %v, want ErrValidate", err)
	}
	if !strings.Contains(err.Error(), "mongo seedlists are not supported by guard-only probe path") {
		t.Fatalf("error = %v, want seedlist policy", err)
	}
	for _, leaked := range []string{rawURI, "user", "secret", "mongo-a.example.com", "mongo-b.example.com"} {
		if strings.Contains(err.Error(), leaked) {
			t.Fatalf("error = %v, leaked %q", err, leaked)
		}
	}
}

func TestMongoURIWithSurroundingWhitespaceRejected(t *testing.T) {
	resetConfigEnv(t)

	rawURI := " mongodb://user:secret@localhost:27017/app "
	t.Setenv("APP__MONGO__ENABLED", "true")
	t.Setenv("APP__MONGO__URI", rawURI)
	t.Setenv("APP__MONGO__DATABASE", "app")

	_, _, err := LoadDetailed(LoadOptions{})
	if err == nil {
		t.Fatalf("LoadDetailed() expected validation error for whitespace-padded mongo uri")
	}
	if !errors.Is(err, ErrValidate) {
		t.Fatalf("error = %v, want ErrValidate", err)
	}
	for _, leaked := range []string{rawURI, "user", "secret"} {
		if strings.Contains(err.Error(), leaked) {
			t.Fatalf("error = %v, leaked %q", err, leaked)
		}
	}
}

func TestMongoURIMustUseNumericTCPPort(t *testing.T) {
	for _, uri := range []string{
		"mongodb://localhost:notaport/app",
		"mongodb://localhost:0/app",
		"mongodb://localhost:65536/app",
	} {
		t.Run(uri, func(t *testing.T) {
			resetConfigEnv(t)
			t.Setenv("APP__MONGO__ENABLED", "true")
			t.Setenv("APP__MONGO__URI", uri)
			t.Setenv("APP__MONGO__DATABASE", "app")

			_, _, err := LoadDetailed(LoadOptions{})
			if err == nil {
				t.Fatal("LoadDetailed() error = nil, want validation error")
			}
			if !errors.Is(err, ErrValidate) {
				t.Fatalf("error = %v, want ErrValidate", err)
			}
		})
	}
}

func TestMongoURIRejectsColonRichNonIPHost(t *testing.T) {
	resetConfigEnv(t)

	rawURI := "mongodb://user:secret@foo:bar:baz/app"
	t.Setenv("APP__MONGO__ENABLED", "true")
	t.Setenv("APP__MONGO__URI", rawURI)
	t.Setenv("APP__MONGO__DATABASE", "app")

	_, _, err := LoadDetailed(LoadOptions{})
	if err == nil {
		t.Fatalf("LoadDetailed() expected validation error for colon-rich non-IP mongo host")
	}
	if !errors.Is(err, ErrValidate) {
		t.Fatalf("error = %v, want ErrValidate", err)
	}
	if !strings.Contains(err.Error(), "mongo.uri must contain a valid probe target") {
		t.Fatalf("error = %v, want mongo probe-target detail", err)
	}
	if !strings.Contains(err.Error(), "invalid mongo host") {
		t.Fatalf("error = %v, want invalid mongo host detail", err)
	}
	for _, leaked := range []string{rawURI, "user", "secret", "foo:bar:baz"} {
		if strings.Contains(err.Error(), leaked) {
			t.Fatalf("error = %v, leaked %q", err, leaked)
		}
	}
}

func TestMongoDisabledAllowsInvalidURI(t *testing.T) {
	resetConfigEnv(t)

	t.Setenv("APP__MONGO__ENABLED", "false")
	t.Setenv("APP__MONGO__URI", "mongo://bad-uri")

	cfg, _, err := LoadDetailed(LoadOptions{})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v, want nil when mongo is disabled", err)
	}
	if cfg.Mongo.Enabled {
		t.Fatalf("Mongo.Enabled = true, want false")
	}
}

func TestMongoProbeAddress(t *testing.T) {
	t.Parallel()

	t.Run("mongodb scheme", func(t *testing.T) {
		t.Parallel()

		address, err := MongoProbeAddress("mongodb://user:pass@localhost:27017/app?replicaSet=rs0")
		if err != nil {
			t.Fatalf("MongoProbeAddress() error = %v", err)
		}
		if address != "localhost:27017" {
			t.Fatalf("MongoProbeAddress() = %q, want localhost:27017", address)
		}
	})

	t.Run("mongodb srv scheme defaults port", func(t *testing.T) {
		t.Parallel()

		address, err := MongoProbeAddress("mongodb+srv://cluster.example.com/app")
		if err != nil {
			t.Fatalf("MongoProbeAddress() error = %v", err)
		}
		if address != "cluster.example.com:27017" {
			t.Fatalf("MongoProbeAddress() = %q, want cluster.example.com:27017", address)
		}
	})

	t.Run("bare host defaults port", func(t *testing.T) {
		t.Parallel()

		address, err := MongoProbeAddress("mongodb://cluster.example.com/app")
		if err != nil {
			t.Fatalf("MongoProbeAddress() error = %v", err)
		}
		if address != "cluster.example.com:27017" {
			t.Fatalf("MongoProbeAddress() = %q, want cluster.example.com:27017", address)
		}
	})

	t.Run("bare ipv6 host defaults port", func(t *testing.T) {
		t.Parallel()

		address, err := MongoProbeAddress("mongodb://2001:db8::1/app")
		if err != nil {
			t.Fatalf("MongoProbeAddress() error = %v", err)
		}
		if address != "[2001:db8::1]:27017" {
			t.Fatalf("MongoProbeAddress() = %q, want [2001:db8::1]:27017", address)
		}
	})

	t.Run("bracketed ipv6 host defaults port", func(t *testing.T) {
		t.Parallel()

		address, err := MongoProbeAddress("mongodb://[2001:db8::1]/app")
		if err != nil {
			t.Fatalf("MongoProbeAddress() error = %v", err)
		}
		if address != "[2001:db8::1]:27017" {
			t.Fatalf("MongoProbeAddress() = %q, want [2001:db8::1]:27017", address)
		}
	})

	t.Run("bracketed ipv6 host keeps explicit port", func(t *testing.T) {
		t.Parallel()

		address, err := MongoProbeAddress("mongodb://[2001:db8::1]:27018/app")
		if err != nil {
			t.Fatalf("MongoProbeAddress() error = %v", err)
		}
		if address != "[2001:db8::1]:27018" {
			t.Fatalf("MongoProbeAddress() = %q, want [2001:db8::1]:27018", address)
		}
	})

	t.Run("rejects empty and malformed bracket hosts", func(t *testing.T) {
		t.Parallel()

		for _, uri := range []string{
			"mongodb://:27017/app",
			"mongodb://[]/app",
			"mongodb://[]:27017/app",
			"mongodb://[2001:db8::1/app",
			"mongodb://2001:db8::1]/app",
			"mongodb://[2001:db8::1]]/app",
			"mongodb://local[host]/app",
			"mongodb://foo:bar:baz/app",
			"mongodb://[localhost]/app",
			"mongodb://[127.0.0.1]/app",
			"mongodb://[localhost]:27017/app",
			"mongodb://[foo:bar:baz]:27017/app",
		} {
			t.Run(uri, func(t *testing.T) {
				t.Parallel()

				if _, err := MongoProbeAddress(uri); err == nil {
					t.Fatal("MongoProbeAddress() error = nil, want malformed host error")
				} else if !errors.Is(err, ErrValidate) {
					t.Fatalf("error = %v, want ErrValidate", err)
				}
			})
		}
	})

	t.Run("rejects seedlists without leaking uri parts", func(t *testing.T) {
		t.Parallel()

		rawURI := "mongodb://leaky-user:top-secret@mongo-a.example.com:27017,mongo-b.example.com:27017/app"

		_, err := MongoProbeAddress(rawURI)
		if err == nil {
			t.Fatal("MongoProbeAddress() error = nil, want seedlist error")
		}
		if !errors.Is(err, ErrValidate) {
			t.Fatalf("error = %v, want ErrValidate", err)
		}
		for _, leaked := range []string{rawURI, "leaky-user", "top-secret", "mongo-a.example.com", "mongo-b.example.com"} {
			if strings.Contains(err.Error(), leaked) {
				t.Fatalf("error = %v, leaked %q", err, leaked)
			}
		}
	})

	t.Run("rejects surrounding whitespace", func(t *testing.T) {
		t.Parallel()

		rawURI := " mongodb://user:top-secret@localhost:27017/app "

		_, err := MongoProbeAddress(rawURI)
		if err == nil {
			t.Fatal("MongoProbeAddress() error = nil, want whitespace error")
		}
		if !errors.Is(err, ErrValidate) {
			t.Fatalf("error = %v, want ErrValidate", err)
		}
		for _, leaked := range []string{rawURI, "user", "top-secret"} {
			if strings.Contains(err.Error(), leaked) {
				t.Fatalf("error = %v, leaked %q", err, leaked)
			}
		}
	})

	t.Run("redacts malformed credential uri", func(t *testing.T) {
		t.Parallel()

		rawURI := "mongodb://leaky-user:top-secret@local[host]/app"

		_, err := MongoProbeAddress(rawURI)
		if err == nil {
			t.Fatal("MongoProbeAddress() error = nil, want malformed host error")
		}
		if !errors.Is(err, ErrValidate) {
			t.Fatalf("error = %v, want ErrValidate", err)
		}
		for _, leaked := range []string{rawURI, "leaky-user", "top-secret", "local[host]"} {
			if strings.Contains(err.Error(), leaked) {
				t.Fatalf("error = %v, leaked %q", err, leaked)
			}
		}
	})
}

func TestReadDurationParsesDefaultDurations(t *testing.T) {
	resetConfigEnv(t)

	cfg, _, err := LoadDetailed(LoadOptions{})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v", err)
	}
	if cfg.HTTP.ReadTimeout != 5*time.Second {
		t.Fatalf("HTTP.ReadTimeout = %s, want 5s", cfg.HTTP.ReadTimeout)
	}
	if cfg.Postgres.ConnMaxLifetime != 30*time.Minute {
		t.Fatalf("Postgres.ConnMaxLifetime = %s, want 30m", cfg.Postgres.ConnMaxLifetime)
	}
}

func TestParseInt(t *testing.T) {
	t.Parallel()

	t.Run("supports mixed numeric inputs", func(t *testing.T) {
		t.Parallel()

		value, err := parseInt("42")
		if err != nil {
			t.Fatalf("parseInt(string) error = %v", err)
		}
		if value != 42 {
			t.Fatalf("parseInt(string) = %d, want 42", value)
		}

		value, err = parseInt(float64(7))
		if err != nil {
			t.Fatalf("parseInt(float64) error = %v", err)
		}
		if value != 7 {
			t.Fatalf("parseInt(float64) = %d, want 7", value)
		}
	})

	t.Run("rejects non integer floats", func(t *testing.T) {
		t.Parallel()

		if _, err := parseInt(1.25); err == nil {
			t.Fatalf("parseInt() expected non-integer error")
		}
	})

	t.Run("rejects non finite floats", func(t *testing.T) {
		t.Parallel()

		if _, err := parseInt(math.NaN()); err == nil {
			t.Fatalf("parseInt() expected non-finite error for NaN")
		}
		if _, err := parseInt(math.Inf(1)); err == nil {
			t.Fatalf("parseInt() expected non-finite error for +Inf")
		}
	})

	t.Run("rejects conversion unsafe float upper bound", func(t *testing.T) {
		t.Parallel()

		if _, err := parseInt(math.Ldexp(1, strconv.IntSize-1)); err == nil {
			t.Fatalf("parseInt() expected overflow error at first unsafe upper bound")
		}
	})

	t.Run("rejects float above exact integer range on wide int", func(t *testing.T) {
		t.Parallel()

		if strconv.IntSize <= 53 {
			t.Skip("parseInt target range is already narrower than float64 exact integer range")
		}
		if _, err := parseInt(math.Ldexp(1, 53) + 2); err == nil {
			t.Fatalf("parseInt() expected unsafe float integer error")
		}
	})

	t.Run("rejects overflow from unsigned values", func(t *testing.T) {
		t.Parallel()

		overflow := uint(math.MaxInt) + 1
		if _, err := parseInt(overflow); err == nil {
			t.Fatalf("parseInt() expected overflow error for uint value")
		}
		if _, err := parseInt(uint64(math.MaxUint64)); err == nil {
			t.Fatalf("parseInt() expected overflow error for uint64 value")
		}
	})
}

func TestParseInt64(t *testing.T) {
	t.Parallel()

	t.Run("supports mixed numeric inputs", func(t *testing.T) {
		t.Parallel()

		value, err := parseInt64("922")
		if err != nil {
			t.Fatalf("parseInt64(string) error = %v", err)
		}
		if value != 922 {
			t.Fatalf("parseInt64(string) = %d, want 922", value)
		}

		value, err = parseInt64(uint32(11))
		if err != nil {
			t.Fatalf("parseInt64(uint32) error = %v", err)
		}
		if value != 11 {
			t.Fatalf("parseInt64(uint32) = %d, want 11", value)
		}
	})

	t.Run("rejects non integer floats", func(t *testing.T) {
		t.Parallel()

		if _, err := parseInt64(float64(2.5)); err == nil {
			t.Fatalf("parseInt64() expected non-integer error")
		}
	})

	t.Run("rejects non finite floats", func(t *testing.T) {
		t.Parallel()

		if _, err := parseInt64(math.NaN()); err == nil {
			t.Fatalf("parseInt64() expected non-finite error for NaN")
		}
		if _, err := parseInt64(math.Inf(-1)); err == nil {
			t.Fatalf("parseInt64() expected non-finite error for -Inf")
		}
	})

	t.Run("rejects conversion unsafe float upper bound", func(t *testing.T) {
		t.Parallel()

		if _, err := parseInt64(math.Ldexp(1, 63)); err == nil {
			t.Fatalf("parseInt64() expected overflow error at first unsafe upper bound")
		}
	})

	t.Run("rejects float above exact integer range", func(t *testing.T) {
		t.Parallel()

		if _, err := parseInt64(math.Ldexp(1, 53) + 2); err == nil {
			t.Fatalf("parseInt64() expected unsafe float integer error")
		}
	})

	t.Run("rejects overflow from unsigned values", func(t *testing.T) {
		t.Parallel()

		if _, err := parseInt64(uint64(math.MaxUint64)); err == nil {
			t.Fatalf("parseInt64() expected overflow error")
		}
	})
}

func TestParseBool(t *testing.T) {
	t.Parallel()

	value, err := parseBool("true")
	if err != nil {
		t.Fatalf("parseBool(true) error = %v", err)
	}
	if !value {
		t.Fatalf("parseBool(true) = false, want true")
	}

	if _, err := parseBool(1); err == nil {
		t.Fatalf("parseBool() expected unsupported type error")
	}
}

func TestValidateRangeHelpers(t *testing.T) {
	t.Parallel()

	t.Run("int range is inclusive", func(t *testing.T) {
		t.Parallel()

		if err := validateIntRange("redis.pool_size", 1, 1, 100); err != nil {
			t.Fatalf("validateIntRange(min) error = %v", err)
		}
		if err := validateIntRange("redis.pool_size", 100, 1, 100); err != nil {
			t.Fatalf("validateIntRange(max) error = %v", err)
		}
	})

	t.Run("int range out of bounds returns ErrValidate", func(t *testing.T) {
		t.Parallel()

		err := validateIntRange("redis.pool_size", 101, 1, 100)
		if err == nil {
			t.Fatalf("validateIntRange() expected error")
		}
		if !errors.Is(err, ErrValidate) {
			t.Fatalf("error = %v, want ErrValidate", err)
		}
		if !strings.Contains(err.Error(), "redis.pool_size") {
			t.Fatalf("error = %v, want field name in message", err)
		}
	})

	t.Run("duration range is inclusive", func(t *testing.T) {
		t.Parallel()

		if err := validateDurationRange("http.read_timeout", time.Second, time.Second, 10*time.Second); err != nil {
			t.Fatalf("validateDurationRange(min) error = %v", err)
		}
		if err := validateDurationRange("http.read_timeout", 10*time.Second, time.Second, 10*time.Second); err != nil {
			t.Fatalf("validateDurationRange(max) error = %v", err)
		}
	})

	t.Run("duration range out of bounds returns ErrValidate", func(t *testing.T) {
		t.Parallel()

		err := validateDurationRange("http.read_timeout", 11*time.Second, time.Second, 10*time.Second)
		if err == nil {
			t.Fatalf("validateDurationRange() expected error")
		}
		if !errors.Is(err, ErrValidate) {
			t.Fatalf("error = %v, want ErrValidate", err)
		}
		if !strings.Contains(err.Error(), "http.read_timeout") {
			t.Fatalf("error = %v, want field name in message", err)
		}
	})
}
