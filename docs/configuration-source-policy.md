# Configuration Source Policy

This template uses a strict split between non-secret and secret configuration.

## Source Of Truth

- YAML (`env/config/*.yaml`) is for baseline non-secret defaults.
- ENV (`APP__...`) is for per-environment overrides and all secret values.
- Flags are for explicit runtime overrides (`--config`, `--config-overlay`, `--config-strict`).

Precedence (last wins):
1. code defaults
2. `--config` base file
3. `--config-overlay` files
4. `APP__...` environment variables
5. flags

## Secret Rules

- Do not place secrets in YAML.
- Secret-like keys in YAML are rejected at load time (`dsn`, `password`, `token`, `secret`, `authorization`, `otlp_headers`).
- In non-local environments, file-based config is hardened:
  - absolute path only
  - must be under allowed roots (`/etc/config`, `/etc/service/config`, `/run/secrets` by default)
  - symlinks are rejected
  - group/world-writable files are rejected
  - max config file size is 1 MiB

Allowed roots can be overridden with `APP_CONFIG_ALLOWED_ROOTS`.

## Runtime Budget Policy

- `http.readiness_timeout` bounds `/health/ready` and startup admission readiness checks. When a dependency readiness probe is enabled, this timeout must be at least that probe's configured budget (`postgres.healthcheck_timeout`, `redis.dial_timeout`, or `mongo.connect_timeout`).
- `http.readiness_propagation_delay` is counted inside `http.shutdown_timeout`; the remaining drain budget must still cover `http.write_timeout`.
- The default process-grace expectation is `30s` HTTP shutdown plus the bootstrap telemetry flush window (`5s`) after HTTP drain. Platform termination grace should cover both instead of only the HTTP server timeout.

## Template Extension Points

Some keys exist as extension points and may be wired later by service authors.
If a key is documented as an extension point, absence of runtime behavior is intentional and non-breaking for the baseline template.
