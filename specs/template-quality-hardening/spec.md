# Make the template a trustworthy implementation reference
status: ready

## Scope and non-goals

Implement the complete accepted quality-audit outcome:

- reject only PostgreSQL environment variables that can actually change pgx
  connection parsing, while allowing unrelated names such as `PGO_ENABLED`;
- remove the unusable public-ingress exception path while operational metrics
  share the application listener;
- remove config load/validation stage budgets that do not interrupt blocking
  file I/O, while retaining the real overall startup deadline and timing
  report;
- keep string normalization with the semantic validator that owns each value;
- replace the generic one-dependency startup probe framework with a concrete
  PostgreSQL startup flow;
- add one runnable, generated-contract-first reference feature slice outside
  the production service.

The production service must not regain fictional business routes. This change
does not add authentication, a database schema, a second production binary, or
a promise that public ingress is supported.

## Behavior and contract delta

- An explicitly configured PostgreSQL DSN still fails closed when a supported
  libpq/pgx ambient variable could alter it. Unrelated process variables whose
  names merely start with `PG` do not affect startup.
- `NETWORK_PUBLIC_INGRESS_ENABLED=true` is rejected directly with an actionable
  explanation that the shared metrics listener prevents public exposure.
  `NETWORK_INGRESS_EXCEPTION_*` no longer forms a supported configuration
  family. Explicit `false` remains required for non-local wildcard binds.
- `config.LoadOptions` no longer advertises independent load and validation
  timeouts. `LoadDetailedWithContext` remains cooperatively cancellable between
  synchronous stages and reports their elapsed time.
- Loaded string values keep their current trim semantics. Trimming occurs in
  the app, HTTP, and observability validation owners before those owners
  validate or publish the values.
- PostgreSQL startup preserves retry, jitter, tracing, error classification,
  readiness registration, and cleanup behavior without a reusable probe-spec
  abstraction.
- `examples/reference-service` exposes a public-by-design, read-only
  `GET /api/v1/articles/{slug}` example backed by an in-memory adapter. Its
  OpenAPI document is canonical, generated strict-server bindings are checked
  for drift, the handler maps feature errors, and `main` is only a composition
  root. The example is compiled and tested by repository-wide checks but is not
  wired into `cmd/service`.

## Invariants and edge cases

- PostgreSQL errors remain sanitized: rejected environment variable names and
  values are not returned.
- Empty PostgreSQL environment values do not change parsing.
- Ingress stays fail-closed; egress exception behavior is unchanged.
- Config cancellation does not claim to interrupt `os.Open`, `Stat`,
  `io.ReadAll`, or YAML parsing.
- The reference handler does not own business behavior or repository details.
  Missing articles map to a stable Problem response; malformed slugs are
  rejected by OpenAPI request validation before feature execution.
- Generated Go is never hand-edited.

## Decisions, constraints, and authorities

- The pgx source used by the current module graph is the authority for ambient
  PostgreSQL variables that can affect parsing.
- `api/openapi/service.yaml` remains the only production HTTP contract.
  `examples/reference-service/api/openapi.yaml` owns only the isolated example
  contract.
- The existing overall bootstrap context owns the startup deadline. Per-probe
  budgets remain where their operations actually consume a context.
- No compatibility layer is retained for ingress exceptions or config stage
  budget fields because both are repository-internal template APIs with no
  viable behavior to preserve.

## Success criteria and proof expectations

- Regression tests prove `PGHOST` is rejected and `PGO_ENABLED` is accepted.
- Network policy tests prove wildcard declaration requirements and direct
  rejection of enabled public ingress; no ingress-exception path remains.
- Config tests prove normalized returned values and parent-context
  cancellation without stage-budget fields.
- Focused bootstrap tests prove PostgreSQL disabled, success, failure, retry,
  tracing, readiness, and cleanup behavior after simplification.
- The reference service has use-case and full HTTP-path tests, including
  generated validation, success, and not-found behavior.
- Generated drift, OpenAPI lint/validation, focused tests, race-relevant tests,
  repository lint, and the repository's broad local check pass.

## Risks, assumptions, and reopen conditions

- Reopen ingress design only when a real private or authenticated metrics
  listener is selected and can be proven end to end.
- Reopen a reusable dependency-probe abstraction only after a second production
  dependency has materially equivalent lifecycle behavior.
- Reopen reference persistence or authentication only for a concrete example
  requirement; neither is implied by this read-only public sample.
