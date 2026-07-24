# Make derived services safe and small by default
status: implemented candidate

## Scope and non-goals

Implement the accepted adoption-review recommendations:

- initialize a complete derived-service identity;
- serve Prometheus metrics only on a separate diagnostics listener and permit
  explicitly declared public application ingress;
- reject unknown application configuration keys;
- prove the first production feature workflow against the real service owners;
- make PostgreSQL and the full multi-agent workflow explicit derived-service
  profiles instead of unavoidable defaults;
- make GHCR publication opt-in;
- reduce startup and toolchain maintenance surface;
- repair configuration and upgrade documentation.

The production API remains health-only. This change does not add fictional
business routes, authentication, an ORM, retries, a DI framework, Kubernetes,
brokers, caches, or public debug endpoints.

## Behavior and contract delta

- `template-init` derives one service name from the module basename and applies
  it to the binary/image name, bootstrap logs, telemetry defaults, example
  environment, OpenAPI title, and derived README. Template repository badges
  and marketing do not survive initialization.
- Derived repositories default to `DATABASE=none`; `DATABASE=postgres` retains
  the optional database capabilities. The complete agent workflow is retained
  by every derived repository.
- The application listener never serves `/metrics`. Prometheus exposition uses
  `observability.metrics.addr`, which defaults to `127.0.0.1:9090`; an empty
  address disables HTTP exposition. Explicit public application ingress is
  allowed after separation.
- Every unknown config key from defaults, files, overlays, or `APP__...`
  environment variables fails validation. There is no permissive flag.
- GHCR publication runs only when repository variable
  `ENABLE_GHCR_PUBLISH=true`; digest signing, attestation, and release preflight
  remain unchanged after admission.
- The isolated reference feature remains the canonical runnable sample, and a
  maintained production-porting guide maps it to the real OpenAPI, handler,
  bootstrap, config, and proof owners.

## Invariants and edge cases

- Metrics remain unavailable from the public application listener.
- A configured diagnostics listener that cannot bind fails startup; a disabled
  listener creates no goroutine or socket.
- Readiness becomes true only after startup probes pass. Shutdown marks
  readiness false before the propagation delay and bounded server shutdown.
- Liveness never depends on PostgreSQL.
- Derived-profile removal is exact, tested against the full template, and
  followed by `go mod tidy`; it must not leave PostgreSQL commands,
  dependencies, examples, generated owners, or delivery promises. The retained
  typed PostgreSQL vocabulary fails closed if enabled in a database-none
  service.
- Generated OpenAPI Go remains derived only from its YAML source.
- Tool isolation must not change runtime dependency selection or command
  behavior.

## Decisions, constraints, and authorities

- `api/openapi/service.yaml` remains the production API authority.
- `internal/config.Config` tags remain the configuration schema authority.
- `internal/infra/telemetry.Metrics` owns the Prometheus handler;
  `cmd/service/internal/bootstrap` owns diagnostics listener lifecycle.
- Existing PostgreSQL implementation remains the optional PostgreSQL profile.
- Existing `.agents` content remains the full workflow profile; the minimal
  profile keeps only a short service-local `AGENTS.md`.
- No new runtime dependency or generator is introduced.

## Success criteria and proof expectations

- Template-init tests prove complete identity replacement, minimal defaults,
  explicit full-profile retention, and absence of retired template markers.
- Focused HTTP/bootstrap tests prove application `/metrics` rejection,
  diagnostics success, public-ingress acceptance, bind failure, readiness, and
  bounded shutdown; race and leak checks pass.
- Config tests prove unknown YAML, overlay, and environment keys fail.
- `go list -deps ./cmd/service` excludes the migration implementation.
- CD admission and version-consistency checks pass without publishing.
- The feature guide is structurally checked and the reference service remains
  generated, compiled, and tested.
- `make check`, the affected broader CI-local gates, and committed-tree proof
  pass before publication.

## Risks, assumptions, and reopen conditions

- A non-loopback diagnostics address relies on deployment-network access
  control; reopen only for a repository-owned authentication design.
- Reopen a shared dependency-probe framework only after a second dependency has
  materially equivalent lifecycle behavior.
- Reopen a separate tools module if it increases command or upgrade complexity
  more than it removes from the runtime module graph.
