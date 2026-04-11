# Template Readiness Follow-Up Specification

## Context

The repository is already a strong Go REST service template: it has explicit docs for `cmd/service/internal/bootstrap`, `internal/app`, `internal/domain`, generated `internal/api`, `internal/infra/*`, OpenAPI, sqlc, migrations, and tests.

The latest read-only review still found gaps that would make future production business-code integration depend on memory or reviewer discipline rather than executable conventions. This follow-up turns those review findings into an implementation-ready handoff.

This is a template-readiness task, not a business feature.

## Scope

Harden the template so a future developer or coding agent can add production business code without guessing:

- how OpenAPI request errors map to Problem responses,
- how endpoint security decisions are recorded and checked,
- how public ingress and `/metrics` exposure are declared,
- how browser/CORS/CSRF decisions are handled for future endpoints,
- how panic and telemetry config errors avoid leaking secrets,
- how runtime dependencies join startup admission and readiness,
- how readiness budgets behave when multiple probes are enabled,
- how config keys avoid drift across defaults, snapshot, and validation,
- how persistence ports and migration validation should be used,
- how generated helper drift checks fit feature development,
- how small clone-readiness polish should be handled.

## Non-Goals

- Do not add real auth middleware, an identity provider, tenant context, object-authorization engine, CSRF/session framework, or fake placeholder security layer.
- Do not add a generic `internal/common`, `internal/util`, generic repository interface, generic transaction helper, or map-driven dependency manager.
- Do not remove or rework the sample `ping` service unless directly needed for a listed hardening task.
- Do not change SQL schema or sqlc-generated files unless implementation discovers a concrete blocker.
- Do not treat the older completed `specs/template-readiness-hardening-2026-04-11` bundle as the implementation authority for this follow-up; it is historical context only.

## Decisions

### D1. Normalize generated chi wrapper errors into Problem responses

`internal/infra/http.NewRouter` already normalizes strict handler request errors, but generated chi wrapper errors still use the generated default `http.Error` path.

Implementation should add a local helper that builds `api.ChiServerOptions` with `ErrorHandlerFunc` set to the same sanitized request-error path: log class/request id, write generic malformed-request Problem, and avoid raw parser details in the client response.

If current routes cannot naturally trigger generated parse errors, test the helper or options builder directly rather than adding fake route behavior solely for testing.

### D2. Make endpoint security decisions executable

The OpenAPI contract currently has global `security: []`, so future operations are public by omission. Keep the baseline endpoints public only by explicit decision.

Implementation should require each OpenAPI operation to declare a security decision. Preferred template shape:

- public operations carry an explicit vendor extension such as `x-security-decision` with a reviewed value and rationale,
- protected operations use a real security scheme plus 401/403 Problem responses,
- operations without a decision fail a contract test or lint-like Go test.

Do not add fake `bearerAuth`, no-op auth middleware, or placeholder tenant context in this task.

### D3. Treat public ingress as an explicit deployment declaration

The service binds to `:8080` by default and the network policy currently treats missing `NETWORK_PUBLIC_INGRESS_ENABLED` as false. That is too easy to misread in non-local deployments.

Implementation should distinguish "not declared" from "declared false". For non-local environments with wildcard binds such as `:8080`, `0.0.0.0:...`, or `[::]:...`, require `NETWORK_PUBLIC_INGRESS_ENABLED` to be set explicitly. If declared true, require the existing exception metadata. If declared false, allow startup but document that the operator is asserting private ingress.

Do not add a real listener split or metrics auth in this task.

### D4. Keep `/metrics` operational, not business-public

`/metrics` may remain on the root router as the documented streaming exception, but the docs and OpenAPI security decision marker must make its trust boundary explicit.

Production guidance should say that metrics require a private scrape path/network, internal listener design, or future real auth before internet exposure.

### D5. Add browser-callable endpoint guidance without adding browser security runtime

CORS is fail-closed today, which is good. It is not a CSRF policy.

Docs should add a future browser endpoint checklist: credentialed CORS origin/header/method allowlist, cookie attributes (`Secure`, `HttpOnly`, `SameSite`, Path, Domain), CSRF/origin-token policy, and negative tests. Do not implement any browser-session mechanism until a real feature needs it.

### D6. Redact panic and OTLP header errors

`Recover` must not log raw recovered values. Log panic type/class, request id, method, path, trace id, and span id.

`parseOTLPHeaders` must not include the raw malformed header entry in returned errors. Return a generic error with pair index and, when safe, header key only. Add tests that `secret-value` or authorization-like values are not emitted.

### D7. Add runtime dependency admission rules

Docs should include a "new runtime dependency" checklist covering:

- config keys and validation,
- startup probe budget and retry class,
- criticality and degraded/fail-open behavior,
- readiness participation and feature flag,
- network-policy egress enforcement,
- cleanup registration for partially initialized resources,
- low-cardinality `startup_dependency_status` labels,
- bootstrap tests.

Keep explicit per-dependency functions. Do not introduce a generic dependency manager until multiple real dependencies repeat the same full behavior.

### D8. Make readiness budget semantics coherent

Readiness probes run sequentially under one context. Current validation only compares `http.readiness_timeout` with each individual probe budget.

Preferred implementation: keep sequential probing and validate the aggregate configured readiness probe budget against `http.readiness_timeout`. If implementation instead changes to bounded parallel probing, stop for concurrency review before coding that behavior.

Docs must also say `/health/live` stays process-only and must not call external dependencies; dependency, startup-admission, drain, and ingress-policy checks belong under readiness.

### D9. Make `http.shutdown_timeout` ordinary config or explicitly code-locked

The current validation requires exactly 30s, while docs read more like a default process-grace policy.

Preferred implementation: replace exact validation with relationship/range validation so services can tune shutdown timeout while preserving:

- `http.readiness_propagation_delay < http.shutdown_timeout`,
- `http.write_timeout <= shutdown_timeout - readiness_propagation_delay`,
- docs note that platform termination grace should also cover the telemetry flush window.

If implementation keeps exact validation, update docs to state that `http.shutdown_timeout` is intentionally code-locked and requires a code/design change.

### D10. Clarify YAML secret placeholder policy

Docs currently say secret-like keys in YAML are rejected, but the shipped YAML uses empty secret-like placeholders and the runtime rejects non-empty values.

Implementation should choose one rule and make code/docs/tests agree. Preferred template rule: empty placeholders are allowed only for schema/default visibility; non-empty secret-like YAML values are rejected.

### D11. Add config key drift protection

Avoid a production reflection mapper. Keep explicit parse/validate code.

Add package-local drift protection, preferably a test deriving expected config leaf keys from `Config` `koanf` tags and comparing them with `knownConfigKeys()` / defaults. This catches default keys that are not represented in the runtime snapshot and snapshot fields that are not defaulted.

If constants are introduced for key names, keep them package-local under `internal/config`; do not create a shared constants package.

### D12. Extract only stable local helper vocabulary

Use package-local constants for startup dependency names, modes, and stage label strings that are metrics/log contracts. A narrow helper for repeated degraded/rejection telemetry is acceptable.

Remove `startupLifecycleStartedAt` plumbing if it remains unused, and remove `runtimeIngressAdmissionGuard.violationOnce` unless a real once-only log/metric is implemented.

Do not collapse Postgres/Redis/Mongo startup functions into one generic callback runner because their policies differ.

### D13. Improve persistence and validation guidance

The persistence boundary is already mostly correct. Add a docs-only port sketch showing that app-owned interfaces belong beside `internal/app/<feature>` when the app layer needs inversion, and `internal/domain` is only for shared stable contracts.

Add `make migration-validate` / `make docker-migration-validate` to the primary Postgres feature checklist. Add conditional `mocks-drift-check` / `stringer-drift-check` guidance for generated helper changes.

Do not extract generic repository or transaction helpers from `ping_history`.

### D14. Track the reserved `internal/domain` seam or document it as create-on-demand

Either add a tiny `internal/domain/doc.go` that explains the package is intentionally empty until a shared stable contract exists, or adjust docs to say the directory is created when needed.

Preferred implementation: add `doc.go` because it gives clone users and agents a concrete local seam without introducing runtime behavior.

### D15. Remove obvious clone-readiness noise

Remove the stray "Hello from claude code" line from `README.md`.

## Open Questions / Assumptions

- Assumption: the implementation should prioritize template clarity and guardrails over broad architecture changes.
- Assumption: public baseline endpoints may remain public when explicitly marked; no auth feature is being introduced.
- Assumption: no SQL schema changes are needed for this follow-up unless implementation discovers a direct test or docs blocker.
- Assumption: `make openapi-check` is required if OpenAPI vendor extensions change generated embedded spec output.

## Validation

Claim: the template-readiness follow-up implementation is ready for closeout after phases 1 through 4.

Scope: approved template hardening changes for HTTP/OpenAPI guardrails, ingress/redaction, config/readiness semantics, contributor placement, persistence guidance, generated-helper guidance, and clone-readiness polish. SQLC, migration, and migration-backed runtime behavior changes are out of scope for this closeout because no `env/migrations`, `internal/infra/postgres/queries`, or `internal/infra/postgres/sqlcgen` files changed.

Verification commands:

- `go test ./cmd/service/internal/bootstrap ./internal/config ./internal/infra/http ./internal/infra/telemetry ./internal/app/... -count=1` passed.
- `make check` passed, including `golangci-lint config verify`, `golangci-lint run --timeout=3m`, and `go test ./...`.
- `make openapi-check` passed using a temporary `GIT_INDEX_FILE` with the current generated `internal/api` output added only to the temporary index, leaving the real staging area unchanged. The command ran `go generate ./internal/api`, OpenAPI drift check, `go test ./internal/api`, `go test ./internal/infra/http -run '^TestOpenAPIRuntimeContract' -count=1`, Redocly lint, and `go tool validate -- api/openapi/service.yaml`.

Observed result: all required closeout commands exited 0. `make sqlc-check` / `make docker-sqlc-check`, `make test-integration`, and `make migration-validate` were not run because their task-local triggers were not met.

Conclusion: final validation passed for the approved follow-up scope.

Next action: none for this task.

## Outcome

Complete. The follow-up is implemented and validated within the approved non-goal boundaries: no real auth/session/CSRF runtime, generic repository or dependency-manager abstraction, metrics listener/auth redesign, SQL schema change, or SQLC surface change was introduced.
