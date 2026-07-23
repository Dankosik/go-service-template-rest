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

An empty `APP__...` value is still an explicit final override. Empty values for required keys are not treated as absent; they flow through parsing or validation and fail fast when the key cannot be empty.

## Operational Network Policy Channel

`NETWORK_*` variables are a separate operator policy channel owned by bootstrap, not ordinary app runtime config. Bootstrap reads them directly from the process environment after the typed `internal/config.Config` snapshot is built from YAML, `APP__...`, and loader flags.

There is no YAML overlay or `APP__...` precedence chain for `NETWORK_*`: the effective value is the process environment value visible to the service at startup. These variables are intended for deployment/network admission controls where explicit declaration matters. For example, missing `NETWORK_PUBLIC_INGRESS_ENABLED` is not the same as setting it to `false`; in non-local wildcard-bind deployments, missing public-ingress declaration fails closed.

Example key families:

- `NETWORK_PUBLIC_INGRESS_ENABLED` declares whether public ingress is expected.
- `NETWORK_EGRESS_ALLOWLIST` and `NETWORK_EGRESS_ALLOWED_SCHEMES` constrain allowed outbound targets.
- `NETWORK_INGRESS_EXCEPTION_*` and `NETWORK_EGRESS_EXCEPTION_*` carry temporary exception metadata such as owner, reason, scope, expiry, and rollback plan.

Do not migrate new feature config into `NETWORK_*`. Use this channel only for bootstrap-owned network policy controls that must remain fail-closed and operator-controlled outside normal application config.

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

Allowed roots can be overridden with `APP_CONFIG_ALLOWED_ROOTS`. In non-local environments, every non-empty root entry must be an absolute path. An empty value keeps the default roots, while a delimiter-only value produces no allowed roots and rejects explicit config files.

## Runtime Budget Policy

- `http.readiness_timeout` bounds `/health/ready` and startup admission readiness checks. It must not exceed `http.write_timeout`, because readiness handlers still run under the server write deadline. Readiness probes run sequentially, so this timeout must also cover the aggregate budget of every enabled readiness probe. In the baseline template that means `postgres.healthcheck_timeout` when Postgres readiness is enabled. Startup admission also requires headroom above the aggregate readiness probe budget, currently the bootstrap fail-fast threshold (`150ms`), because bootstrap checks the readiness context again after the internal readiness check returns.
- Dependency startup timeout fields are per-attempt maxima, not guarantees that every retry gets that much time. When a dependency is enabled, bootstrap rejects values that exceed its startup probe envelope: `postgres.connect_timeout` and `postgres.healthcheck_timeout` must be at most `5s`.
- `http.shutdown_timeout` is tunable within validation bounds. `http.readiness_propagation_delay` is counted inside it; the remaining drain budget must still cover `http.write_timeout`.
- The default process-grace expectation is `30s` HTTP shutdown plus the bootstrap telemetry flush window (`5s`) after HTTP drain. Platform termination grace should cover readiness propagation, HTTP drain, and telemetry flush instead of only the HTTP server timeout.

Postgres DSN driver parsing belongs to `internal/infra/postgres`. `internal/config` validates required presence and generic bounds, while bootstrap asks the Postgres adapter for a sanitized probe address before egress admission. A malformed `postgres.dsn` is therefore classified as dependency initialization during bootstrap address resolution, not as generic config validation, and adapter parse errors must not echo credentials.

The baseline Postgres DSN contract is intentionally strict. The DSN must come from the typed `postgres.dsn` value, which in deployments means `APP__POSTGRES__DSN`, and it must explicitly include host, port, database, user, non-empty password, and `sslmode`. URL and keyword/value DSNs are accepted when they describe one TCP target. The adapter rejects empty DSNs; any non-empty `PG*` environment input; `service`, `servicefile`, and `passfile`; caller-provided TLS file keys such as `sslcert`, `sslkey`, `sslrootcert`, and `sslpassword`; multi-host or fallback targets; default, `prefer`, or `allow` `sslmode`; and Unix socket hosts. Before calling pgx, the adapter also clears pgx's implicit passfile and TLS file defaults so `.pgpass` and `~/.postgresql/*` files cannot become side-effect config sources.

## Adding A Config Key

When a feature needs a new runtime config key:

1. Add the typed field and `koanf` tag in `internal/config/types.go`.
2. Add the default in `internal/config/defaults.go` when the key has a baseline value.
3. Add validation in `internal/config/validate.go` when the key has bounds, mode-specific rules, or security-sensitive behavior.
4. Update `env/config/local.yaml` or `env/.env.example` only where the key belongs for non-secret examples or env-driven secrets.
5. Update docs that explain the feature's config behavior, especially secret-source or runtime-budget rules.
6. Add or update `internal/config` tests so the tagged field is decoded into the immutable `Config` snapshot and validation rejects invalid values.

`internal/config/snapshot.go` decodes the tagged `Config` shape through Koanf,
so adding a field does not require a second manual mapping. The source of truth
is the typed config shape, Go defaults, validation, and tests.
