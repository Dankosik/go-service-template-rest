# Template Readiness Hardening Specification

## Context

The prior review asked whether a team can clone this repository and naturally add production business code without guessing folder boundaries, coding style, ownership seams, utility placement, generated-code rules, and validation workflow.

The verdict was: the template has a strong baseline, but several seams still require too much inference before production business-code development. This task turns those findings into implementation-ready decisions.

## Scope

Implement template-hardening changes that make the intended extension path explicit and hard to misuse:

- clone/quickstart config consistency,
- composition-root ownership,
- readiness probe contract ownership,
- startup/readiness/shutdown budget semantics,
- generated HTTP route and `/metrics` ownership,
- auth/security decision points for future endpoints,
- secret-source policy coverage,
- persistence/sqlc sample status,
- migration and integration-test guidance,
- Makefile/docs discoverability,
- local helper/source-of-truth cleanup where duplication would mislead future teams.

## Non-goals

- Do not implement a real auth provider, tenant model, object authorization engine, rate limiter, Redis adapter, Mongo adapter, or business feature.
- Do not turn `ping` into production business logic.
- Do not add a generic `internal/common` or `internal/util` package.
- Do not weaken generated-code ownership by hand-editing generated files.
- Do not solve the native macOS `go tool sqlc` compilation issue unless it directly blocks validation; prefer Docker sqlc proof if needed.

## Constraints

- Keep `cmd/service/main.go` thin; service composition belongs in `cmd/service/internal/bootstrap`.
- `internal/app` must stay transport-agnostic and must not import `internal/infra/http` or concrete driver packages.
- Normal API routes are generated from `api/openapi/service.yaml`; manual chi routes require an explicit root-router exception.
- Config source-of-truth remains `internal/config` defaults/types/snapshot/validation plus `env/config/*.yaml` and `APP__...` environment inputs.
- The implementation must preserve the working baseline proven by prior `make check` and `make openapi-check`.

## Decisions

### D1. Fix clone-time config drift first

`env/.env.example` must agree with the hard-coded/default shutdown baseline. Set `APP__HTTP__SHUTDOWN_TIMEOUT` to the validated default unless the implementation deliberately changes the validation policy.

Add a config fixture test that loads `.env.example` values into the existing config loader path and proves the example does not fail validation. This prevents future template examples from drifting away from config validation.

### D2. Make HTTP adapter stop acting as a composition root

`internal/infra/http.NewRouter` should not create fallback app services or metrics when dependencies are nil. The service binary bootstrap must build and pass the required dependencies.

Preferred shape: change `NewRouter` to return `(http.Handler, error)` and validate required inputs in a narrow local helper. Test convenience should live in test helpers such as `mustNewRouter`, not in production fallback behavior.

### D3. Use consumer-owned interfaces for readiness probes

The readiness probe interface should have one owner. For this repository, the least surprising Go convention is: keep the interface beside the consumer in `internal/app/health`, and reserve `internal/domain` for shared domain types/contracts only when multiple app packages need the same abstraction.

Implementation should remove or replace the unused `internal/domain/readiness.go` example, then update docs so future integrations implement `health.Probe` for readiness checks unless a real shared domain contract is introduced.

### D4. Do not make `ping_history` a hidden production schema

Do not wire `ping_history` into `ping` as a real business side effect. That would overfit the template to the sample endpoint.

Preferred path: make `ping_history` explicitly non-production sample material, or remove it from the default runtime/migration path if the sqlc toolchain can still be proven cleanly. The implementation must choose the smallest safe path after checking sqlc behavior:

- If sqlc and integration proof still work with no default sample schema/query, remove the `ping_history` migration/repository/query/generated code and replace it with docs that show the persistence recipe.
- If the template still needs a checked sqlc sample to prove the toolchain, keep it but label it as a template sample in docs/tests and avoid presenting it as business-owned production schema.

In either case, do not leave an unexplained infra-only repository as the primary persistence exemplar.

### D5. Make migration samples deterministic

Default migrations should not use `IF NOT EXISTS` / `IF EXISTS` unless the migration is intentionally idempotent repair logic. Template examples should teach deterministic migrations that fail on unexpected schema drift.

### D6. Keep generated HTTP route ownership explicit

Normal API endpoints must be added through `api/openapi/service.yaml`, generated bindings, and strict handler methods in `internal/infra/http`.

`/metrics` is currently a root-router shortcut while also being present in the OpenAPI contract. The implementation must make this exception explicit, preferably with a route-owner guard test that rejects future generated/manual path overlaps except documented root exceptions. If the implementation chooses to remove `/metrics` from the OpenAPI contract, it must preserve runtime metrics behavior and update OpenAPI/runtime tests accordingly.

### D7. Do not imply auth exists when it does not

The OpenAPI `bearerAuth` scheme should not remain as a misleading unused hint. Either remove it until real auth is implemented, or add explicit protected-endpoint rules and tests. For this hardening task, prefer removal plus documentation of the future protected-endpoint checklist because there is no identity provider or tenant model yet.

Docs must say that every new business endpoint needs an explicit security decision: public by design, protected by a real auth middleware/contract, or blocked pending security spec.

### D8. Treat `/metrics` as an operational trust-boundary decision

Metrics exposure must be documented as operational, not ordinary public business API. If the route remains on the same server, document that production deployments must expose it only on a private scrape path/network or add a future auth/internal-listener design before public internet exposure.

Do not add fake auth to metrics in this task.

### D9. Gate external readiness on startup admission

Serving may start before startup admission completes, but external `/health/ready` should not report ready until admission has marked the runtime ready.

Preferred shape:

- keep an internal startup dependency/readiness check that does not depend on `admission.Ready()`,
- pass an external readiness gate into HTTP handlers,
- make `/health/ready` check admission before reporting ready,
- document whether non-health traffic relies only on platform readiness or also needs a pre-ready middleware.

### D10. Make readiness and shutdown budgets explicit and coherent

Remove the hard-coded external readiness timeout from HTTP handlers. Add configuration for the readiness timeout, or derive it from an explicit config field that is validated.

Couple shutdown-related budgets so the template does not validate impossible combinations. At minimum, validate or document the relationship among:

- `http.write_timeout`,
- `http.shutdown_timeout`,
- readiness propagation delay,
- telemetry flush timeout,
- platform termination grace.

Preferred validation: sync HTTP `write_timeout` should not exceed the effective HTTP drain budget after readiness propagation delay.

### D11. Clean up dependency admission ownership

Dependency admission should register runtime handles, readiness probes, status, and cleanup together. If a later dependency fails, already-created resources must be cleaned up before returning.

Keep degraded-mode decisions explicit. Redis cache and Mongo degraded modes currently have no runtime handles or app-facing behavior. The hardening task should document them as probe/extension-point behavior unless a future adapter design defines cache bypass, stale read, feature-off, or fail-closed semantics.

### D12. Extract stable same-package helper sources of truth

Do only narrow same-package extraction where duplication creates future template drift:

- `networkPolicy.EnforceIngress` and `ValidateIngressRuntime` should share one ingress validation helper.
- dependency probe rejection telemetry should use a narrow helper for common span/metric/log rejection fields while keeping fail-open/degraded branches explicit.
- `resetConfigEnv` in config tests should derive `APP__...` keys from config known keys plus a small non-namespace list, instead of hand-maintaining a stale list.

Do not create broad helper buckets.

### D13. Tighten secret-like config policy

The secret-source policy should catch common future key shapes such as `client_secret`, `jwt_secret`, `api_key`, `private_key`, and top-level `token`. Implement this as segment/suffix matching over normalized `.`, `_`, and `-` separators or as an explicit schema-owned sensitive-key registry.

Add tests for both allowed non-secret keys and rejected secret-like keys.

### D14. Sanitize generated request error details at the HTTP edge

Do not expose raw strict-handler request parse errors as client Problem `detail`. Return a stable generic detail for malformed requests and log only sanitized error class plus request ID.

### D15. Document outbound integration security expectations

Because the current egress/network policy is bootstrap-local, docs must tell future outbound adapters what to do:

- fixed outbound targets are validated by bootstrap before wiring,
- dynamic/user-controlled outbound URLs require a separate security design,
- adapters must declare target source, timeout, redirect behavior, DNS/IP-class behavior, and egress allowlist policy.

Do not move network policy into a new shared package unless implementation finds a concrete adapter-facing use in this task.

### D16. Upgrade docs and Make discoverability

Docs must include a compact feature placement convention:

- OpenAPI contract and generated bindings,
- app behavior,
- app-owned ports/contracts,
- HTTP mapping,
- Postgres/sqlc repository flow,
- bootstrap wiring,
- tests by layer,
- validation commands by change type.

`make help` should surface feature validation targets: `openapi-check`, `sqlc-check`, `test-integration`, and Docker equivalents.

## Open Questions / Assumptions

- Assumption: this task should harden the template without adding real business domain semantics.
- Assumption: if native `make sqlc-check` remains blocked, Docker sqlc validation is acceptable proof.
- Resolved implementation check: the current sqlc setup fails generation when `queries/*.sql` is empty, so `ping_history` is retained as an explicit template SQLC sample and documented as non-production sample state.

## Plan Summary

Use the ordered plan in `plan.md`. Implementation starts with clone/config correctness, then ownership seams, then HTTP/security/readiness policy, then persistence/docs/tests cleanup.

## Validation

- `go test ./internal/infra/postgres -count=1`: passed.
- `make help`: passed and includes OpenAPI, SQLC, integration-test, and Docker validation targets.
- `make docker-sqlc-check`: passed.
- `make docker-migration-validate`: passed.
- `make test-integration`: passed.
- `make check`: passed.
- `make openapi-check`: passed with a temporary git index containing the current OpenAPI source/generated pair; the real git index was not changed.

## Outcome

Implemented through Phase 4 and final validation. `ping_history` remains only as an explicit template SQLC sample, retained migrations are deterministic, docs/test placement/Make discoverability are updated, and required validation is green.
