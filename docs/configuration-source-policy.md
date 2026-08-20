# Configuration Source Policy

This template uses a strict split between non-secret and secret configuration.

## Agent Runtime Configuration

Portable Codex behavior is repository state. `.agents/codex-project.toml` owns
the supported project defaults, and `scripts/codex-agents-sync.sh` renders its
marked block plus the project-agent registry into `.codex/config.toml`. A new
Codex session loads that project view only after the user trusts the checkout.

Keep identity, credentials, provider or MCP endpoints, notification commands,
telemetry, selected profiles, UI preferences, host-absolute paths, sandbox, and
approval choices in the user or system Codex config. The generated project view
contains only the universal template runtime and role registry. The installed
Codex version and configured model availability remain host prerequisites; an
unsupported model is a capability gap, not permission to silently change the
repository policy.

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

## Network Admission Ownership

Network reachability is not an application concern here. The service binds
`http.addr` and nothing in this process restricts who can reach it.

Earlier revisions required an operator to set
`NETWORK_PUBLIC_INGRESS_ACKNOWLEDGED` to `true` or `false` before a non-local
wildcard bind would start. That gate is gone. It enforced nothing — both answers
were accepted and neither changed the bind — and it was inverted in practice: it
triggered on `app.env`, which defaults to `local`, so it blocked a correctly
configured production deployment while starting one that had forgotten to set
the environment at all. The startup summary already records `app.env` and
`http.addr` on every boot, which is the same information without a failed
deploy.

Ingress admission belongs to the deployment platform — firewall, security group,
network policy, or service mesh — because that is the only layer that observes
every connection attempt.

Do not migrate feature config into `NETWORK_*`. Outbound network admission belongs to the deployment platform (firewall, network policy, service mesh, or equivalent), where all connection attempts can actually be enforced. Application-specific allowlists, provider authentication, retries, and error mapping belong to the individual service. Go HTTP clients built on `http.DefaultTransport`, including the OTLP HTTP exporter, use the standard `HTTP_PROXY`, `HTTPS_PROXY`, and `NO_PROXY` environment conventions when a proxy is required. The bounded outbound client in `internal/infra/httpclient` is the exception: it sets `Transport.Proxy = nil` because a proxy would resolve and dial on the client's behalf and bypass the post-DNS public-address gate that makes the fixed-authority guarantee meaningful. A service that must reach external providers through a mandatory egress proxy uses a plain `net/http` client in its provider adapter and relies on platform network policy instead.

## OpenTelemetry Environment Policy

This service configures OpenTelemetry from `observability.otel.*` only. It
deliberately does not read the standard `OTEL_*` environment variables, so
resource identity and exporter target cannot be retargeted by ambient process
environment:

- `OTEL_RESOURCE_ATTRIBUTES` and `OTEL_SERVICE_NAME` are ignored; the typed
  config snapshot is the only resource source.
- When `observability.otel.exporter.otlp_endpoint` is set, ambient
  `OTEL_EXPORTER_OTLP_*` variables split in two. Variables the explicit
  exporter options already override — `..._ENDPOINT`, `..._TRACES_ENDPOINT`,
  `..._INSECURE`, `..._TRACES_INSECURE` — plus ones that carry no destination
  or credential, such as `..._PROTOCOL`, `..._TIMEOUT`, and `..._COMPRESSION`,
  are **ignored** and logged as `telemetry_ambient_env_ignored`. Tracing keeps
  working. Credential and trust material — `..._HEADERS`,
  `..._TRACES_HEADERS`, `..._CERTIFICATE`, `..._TRACES_CERTIFICATE`,
  `..._CLIENT_CERTIFICATE`, `..._CLIENT_KEY`, and their `TRACES_` forms — is
  **rejected**, because this service sets no client certificate or CA pool and
  sets headers only from config, so an injected value would otherwise travel to
  the collector unverified. The rejection degrades telemetry and is logged; it
  does not stop the service.
- When `observability.otel.exporter.otlp_endpoint` is empty, tracing is
  disabled (valid trace IDs are still produced for propagation and log
  correlation, but no span is exported). If ambient `OTEL_EXPORTER_OTLP_*`
  variables are present in that case, startup logs
  `telemetry_ambient_env_ignored` naming the ignored variables.

The override direction is not an assumption: `otlptracehttp` applies ambient
environment first and explicit options second, and this service always passes
`WithEndpointURL`, which owns the endpoint, the URL path, and the TLS scheme.

This matters on platforms that inject the standard variables automatically
(OpenTelemetry Operator auto-instrumentation, Grafana Alloy, vendor add-ons).
Injection alone does **not** enable export here. Map the injected endpoint onto
`APP__OBSERVABILITY__OTEL__EXPORTER__OTLP_ENDPOINT` explicitly, and treat
`telemetry_ambient_env_ignored` as the alarm that this mapping is missing.
Telemetry setup failures never block startup: they log
`startup_dependency_degraded` with `reason` and the underlying `err`.

Trace-export state is queryable rather than log-only. The startup summary
carries `tracing.exporter` as `active`, `disabled`, or `degraded`, and the
Prometheus diagnostics listener exposes
`service_startup_trace_exporter_active`. Alert on that gauge: a service that
exports no traces still answers every request and reports healthy.

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
- Secret-scan `change` mode must cover the reviewable worktree and every commit
  after the base merge point. `history` mode remains mandatory on main, nightly,
  and release; a missing base ref falls back to that full scan.
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

- `http.readiness_timeout` bounds `/health/ready` and startup admission readiness checks. It must not exceed `http.write_timeout`, because readiness handlers still run under the server write deadline. Readiness probes run sequentially, so this timeout must also cover the aggregate budget of every enabled readiness probe. The PostgreSQL probe has one template-owned `3s` budget. Startup admission also requires `150ms` headroom because bootstrap checks the readiness context again after the internal readiness check returns.
- `http.request_timeout` is the per-request handler budget and defaults to `8s`. It is the only bound on how long one request may hold a goroutine and its pooled resources: `http.read_timeout` and `http.write_timeout` are connection deadlines that never cancel the request context, so without this a handler waiting on a slow dependency outlives the client that asked. It must not exceed `http.write_timeout`, because a budget that expires after the write deadline can no longer send the `504` reporting it. Lowering `http.write_timeout` below `8s` therefore requires lowering this too. It bounds handlers, not the process: a handler that ignores its context cannot be interrupted from the transport layer.
- PostgreSQL connection and startup-ping timeouts are template-owned `3s` defaults inside the adapter; the startup stage still caps the whole probe at `5s`.
- The migrator owns a `5m` orchestration deadline. Cancellation stops subsequent
  migration work; one in-flight statement can still take up to its separate
  `2m` server limit. Its `15s` Goose session-lock budget also reserves time for detached
  advisory-lock release and connection cleanup. The statement budget must not
  exceed the overall budget, and the lock budget must be strictly smaller than
  it so the cleanup reserve is non-empty. Reopen these code defaults only when
  rehearsal evidence for the actual schema and largest production table proves
  them insufficient.
- `http.shutdown_timeout` is tunable within validation bounds. `http.readiness_propagation_delay` is counted inside it; the remaining drain budget must still cover `http.write_timeout`.
- The default process-grace expectation is `30s` HTTP shutdown plus the bootstrap telemetry flush window (`5s`) after HTTP drain. Platform termination grace should cover readiness propagation, HTTP drain, and telemetry flush instead of only the HTTP server timeout.

  **This is a deployment precondition on every platform, not only Railway.** The
  default worst-case sequence is 35 seconds, so a grace period shorter than that
  SIGKILLs the service mid-drain and drops in-flight requests. Default grace
  periods of 10–30 seconds are too short. Configure it explicitly:

  | Platform | Setting |
  | --- | --- |
  | Kubernetes | `terminationGracePeriodSeconds: 50` |
  | Docker | `docker run --stop-timeout 45` / `docker stop --time 45` |
  | Compose | `stop_grace_period: 45s` |
  | ECS | `stopTimeout: 45` |
  | Railway | `drainingSeconds` in `railway.toml` (already set to `45`) |

  Changing `http.shutdown_timeout` changes this number; re-derive it and re-run
  the runtime-image shutdown check. See
  [Railway Deployment Profile](railway-deployment-profile.md) for the full
  derivation.
- `http.access_log_health_probes` defaults to `false`, so matched `GET /health/live` and `GET /health/ready` requests are served without an access-log line. Orchestrator probes otherwise produce continuous no-signal volume that every log backend bills. The exclusion is route-based: an unmatched path that merely resembles a probe is still recorded, and span route attribution is unchanged. Set it to `true` while debugging readiness.

Postgres DSN driver parsing belongs to `internal/infra/postgres`. `internal/config` validates required presence and generic bounds, while bootstrap asks the Postgres adapter for a sanitized probe address before egress admission. A malformed `postgres.dsn` is therefore classified as dependency initialization during bootstrap address resolution, not as generic config validation, and adapter parse errors must not echo credentials.

The baseline Postgres DSN contract is intentionally strict. The DSN must come from the typed `postgres.dsn` value, which in deployments means `APP__POSTGRES__DSN`, and it must use a `postgres://` or `postgresql://` URL with explicit host, port, database, user, non-empty password, and `sslmode`. Keyword/value DSNs are not accepted. The adapter rejects empty DSNs; non-empty libpq/pgx PostgreSQL environment variables such as `PGHOST`, `PGOPTIONS`, and `PGTZ`; `service`, `servicefile`, and `passfile`; caller-provided TLS file keys such as `sslcert`, `sslkey`, `sslrootcert`, and `sslpassword`; multi-host or fallback targets; default, `prefer`, or `allow` `sslmode`; and Unix socket hosts. Unrelated process variables whose names merely begin with `PG`, such as `PGO_ENABLED`, do not affect parsing. Before calling pgx, the adapter also clears pgx's implicit passfile and TLS file defaults so `.pgpass` and `~/.postgresql/*` files cannot become side-effect config sources.

## Adding A Config Key

When a feature needs a new runtime config key:

1. Add the typed field and `koanf` tag, its default when the key has a baseline value, and its validation — all three in the section's own `internal/config/<section>_config.go`. One key is one file: the reason a value was chosen sits beside the rule that enforces it, and a section a build profile removes leaves with its file. `types.go` and `defaults.go` hold the `Config` shape, the merge over it, and only the sections a dedicated file would not pay for (`app`, `health`, `log`, `runtime`); `http` and `observability` are always present too and still have their own files, because size rather than removability is what earns one. `validate.go` owns the order the sections run in and the helpers more than one section shares. A rule spanning two sections goes to whichever section depends on the other and takes that other section as a parameter — `validatePostgres` against the request budget, `validateOutbox` against Postgres — so no rule outlives the section it is about.
2. Add the key to both `sentinelConfigSourceValues` and `expectedSentinelSnapshotValues` in `internal/config/snapshot_contract_test.go`. `TestBuildSnapshotMapsEveryKnownConfigLeafKey` walks the shape by reflection and fails on a leaf that either map is missing, so a key skipped here fails the suite rather than silently going unproven.
3. Update `env/config/local.yaml` or `env/.env.example` only where the key belongs for non-secret examples or env-driven secrets.
4. Update docs that explain the feature's config behavior, especially secret-source or runtime-budget rules.
5. Add or update `internal/config` tests so the tagged field is decoded into the immutable `Config` snapshot and validation rejects invalid values.

`internal/config/snapshot.go` decodes the tagged `Config` shape through Koanf,
so adding a field does not require a second manual mapping. The source of truth
is the typed config shape, Go defaults, validation, and tests.
