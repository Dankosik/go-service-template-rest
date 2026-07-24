# Configuration Source Policy

This template uses a strict split between non-secret and secret configuration.

## Source Of Truth

- YAML (`env/config/*.yaml`) is for baseline non-secret defaults.
- ENV (`APP__...`) is for per-environment overrides and all secret values.
- CLI flags are loader controls: `--config` selects the base file and
  `--config-overlay` adds ordered overlays. They do not provide arbitrary
  runtime config key overrides.

Runtime config value precedence (last wins):
1. code defaults
2. `--config` base file
3. `--config-overlay` files
4. `APP__...` environment variables

An empty `APP__...` value is still an explicit final override. Empty values for required keys are not treated as absent; they flow through parsing or validation and fail fast when the key cannot be empty.
Unknown keys from files, overlays, or `APP__...` variables always fail
validation; there is no permissive mode.

## Operational Network Policy Channel

`NETWORK_PUBLIC_INGRESS_ENABLED` is a separate operator policy value owned by bootstrap, not ordinary app runtime config. Bootstrap reads it directly from the process environment after the typed `internal/config.Config` snapshot is built from YAML, `APP__...`, and loader flags.

There is no YAML overlay or `APP__...` precedence chain for this value: the effective value is the process environment value visible to the service at startup. It exists because explicit declaration matters for public ingress. Missing `NETWORK_PUBLIC_INGRESS_ENABLED` is not the same as setting it to `false`; in non-local wildcard-bind deployments, missing public-ingress declaration fails closed.

`NETWORK_PUBLIC_INGRESS_ENABLED` declares whether a non-local wildcard
  application listener is public (`true`) or private (`false`). Both values are
  valid when explicit; Prometheus metrics use a separate diagnostics listener.

Do not migrate feature config into `NETWORK_*`. Outbound network admission belongs to the deployment platform (firewall, network policy, service mesh, or equivalent), where all connection attempts can actually be enforced. Application-specific allowlists, provider authentication, retries, and error mapping belong to the individual service. Go HTTP clients and the OTLP HTTP exporter use the standard `HTTP_PROXY`, `HTTPS_PROXY`, and `NO_PROXY` environment conventions when a proxy is required.

`observability.metrics.addr` owns the Prometheus diagnostics listener. It
defaults to `127.0.0.1:9090`; an empty value disables HTTP exposition. Binding
failure blocks startup. Do not use a non-loopback value without deployment
network policy that keeps the listener private.

## Secret Rules

- Do not place secrets in YAML.
- Do not baseline a new or active secret-scanner finding. A historical
  credential may remain in the gitleaks baseline only after its owner confirms
  revocation or rotation and records the owner, date, and rationale in the
  approving pull request. History rewriting is a separate repository-owner
  decision and does not replace credential revocation.
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

The baseline Postgres DSN contract is intentionally strict. The DSN must come from the typed `postgres.dsn` value, which in deployments means `APP__POSTGRES__DSN`, and it must use a `postgres://` or `postgresql://` URL with explicit host, port, database, user, non-empty password, and `sslmode`. Keyword/value DSNs are not accepted. The adapter rejects empty DSNs; non-empty libpq/pgx PostgreSQL environment variables such as `PGHOST`, `PGOPTIONS`, and `PGTZ`; `service`, `servicefile`, and `passfile`; caller-provided TLS file keys such as `sslcert`, `sslkey`, `sslrootcert`, and `sslpassword`; multi-host or fallback targets; default, `prefer`, or `allow` `sslmode`; and Unix socket hosts. Unrelated process variables whose names merely begin with `PG`, such as `PGO_ENABLED`, do not affect parsing. Before calling pgx, the adapter also clears pgx's implicit passfile and TLS file defaults so `.pgpass` and `~/.postgresql/*` files cannot become side-effect config sources.

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
