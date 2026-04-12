# Configuration Source Policy

This template uses a strict split between non-secret and secret configuration.

## Source Of Truth

- YAML (`env/config/*.yaml`) is for baseline non-secret defaults.
- ENV (`APP__...`) is for per-environment overrides and all secret values.
- CLI flags are loader controls today: `--config` selects the base file, `--config-overlay` adds ordered overlays, and `--config-strict` controls unknown-key handling. They do not provide arbitrary runtime config key overrides.

Runtime config value precedence (last wins):
1. code defaults
2. `--config` base file
3. `--config-overlay` files
4. `APP__...` environment variables

## Secret Rules

- Do not place secrets in YAML.
- Secret-like YAML keys may exist only as empty placeholders for schema/default visibility.
- Non-empty secret-like YAML values are rejected at load time (`dsn`, `password`, `token`, `secret`, `authorization`, `otlp_headers`).
- In non-local environments, file-based config is hardened:
  - absolute path only
  - must be under allowed roots (`/etc/config`, `/etc/service/config`, `/run/secrets` by default)
  - symlinks are rejected
  - group/world-writable files are rejected
  - max config file size is 1 MiB

Allowed roots can be overridden with `APP_CONFIG_ALLOWED_ROOTS`.

## Runtime Budget Policy

- `http.readiness_timeout` bounds `/health/ready` and startup admission readiness checks. Readiness probes run sequentially, so this timeout must cover the aggregate budget of every enabled readiness probe: `postgres.healthcheck_timeout`, `redis.dial_timeout` when Redis readiness is enabled or Redis runs in store mode, and `mongo.connect_timeout` when Mongo readiness is enabled.
- `http.shutdown_timeout` is tunable within validation bounds. `http.readiness_propagation_delay` is counted inside it; the remaining drain budget must still cover `http.write_timeout`.
- The default process-grace expectation is `30s` HTTP shutdown plus the bootstrap telemetry flush window (`5s`) after HTTP drain. Platform termination grace should cover readiness propagation, HTTP drain, and telemetry flush instead of only the HTTP server timeout.

## Template Extension Points

Some keys exist as extension points and may be wired later by service authors.
If a key is documented as an extension point, absence of runtime behavior is intentional and non-breaking for the baseline template.

Redis and Mongo keys are guard-only extension stubs in the baseline template. They let bootstrap validate planned dependency exposure, timeout budgets, and readiness policy, but they do not provide cache, store, or database adapters. Add `internal/infra/redis` or `internal/infra/mongo` only when a real app feature needs runtime behavior; do not turn config or bootstrap checks into hidden cache/store semantics.

`MongoProbeAddress` is part of that guard-only path: `internal/config` owns extracting a probe-ready address from the typed config snapshot so validation and bootstrap admission can stay deterministic. A future Mongo adapter should own runtime connection, database, retry, query, and store semantics under `internal/infra/mongo` instead of growing them around the config helper.

## Adding A Config Key

When a feature needs a new runtime config key:

1. Add the typed field and `koanf` tag in `internal/config/types.go`.
2. Add the default in `internal/config/defaults.go` when the key has a baseline value.
3. Thread the value into the immutable runtime snapshot in `internal/config/snapshot.go`.
4. Add validation in `internal/config/validate.go` when the key has bounds, mode-specific rules, or security-sensitive behavior.
5. Update `env/config/default.yaml`, `env/config/local.yaml`, and `env/.env.example` only where the key belongs for non-secret examples or env-driven secrets.
6. Update docs that explain the feature's config behavior, especially secret-source or runtime-budget rules.
7. Add or update `internal/config` tests so the key reaches the built `Config` snapshot and validation rejects invalid values.

Do not list every existing key in this recipe. The source of truth is the typed config shape, defaults, snapshot construction, validation, and tests.
