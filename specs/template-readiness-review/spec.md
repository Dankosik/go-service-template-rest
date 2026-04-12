# Template Readiness Improvements Spec

## Context

The subagent-backed template-readiness review found the repository mostly ready as a production-business-code template. The durable architecture and placement guidance are strong, but the code examples do not yet demonstrate a production-shaped business feature path end to end.

The main gap is not folder structure. It is onboarding proof: a future team can read where code should go, but the code only shows a trivial `ping` app service and an infra-only SQLC sample. A later implementation should make the intended path harder to misuse without turning the template into a fixed business-domain product.

## Scope / Non-goals

In scope:

- Add or tighten guidance for the first real business feature path: app-owned port and types, adapter mapping, HTTP request/response mapping, bootstrap injection, and test layers.
- Make `ping_history` harder to mistake for production business state while preserving SQLC drift coverage.
- Clarify Redis/Mongo guard-only status so cache/store semantics do not become hidden config/bootstrap behavior before a real feature owns them.
- Clarify the protected-operation seam without adding placeholder auth.
- Add narrow guardrails for manual HTTP route registration and app-to-infra dependency direction.
- Canonicalize the HTTP `Allow` header behavior if implementation touches route policy tests.
- Fix one concrete startup diagnostic helper drift where dependency probe rejection logs omit the error value.
- Update test placement guidance for config keys and real endpoint plus persistence plus bootstrap scenarios.

Out of scope:

- No implementation during the planning session that produced this spec.
- No generated OpenAPI or SQLC hand edits.
- No fake business-domain example such as orders/todos just to have a demo feature.
- No placeholder auth middleware, fake identity model, or broad root auth middleware.
- No migration/table/query rename for `ping_history` without an explicit maintainer decision.
- No Redis/Mongo runtime adapters unless a real app feature needs them.

## Constraints

- Preserve OpenAPI as the REST contract source of truth and regenerate `internal/api` only from `api/openapi/service.yaml`.
- Preserve SQLC source-of-truth flow: migrations and query files first, generated `sqlcgen` second, hand-written repository mapping third.
- Preserve `internal/app` as transport- and driver-agnostic.
- Preserve `internal/domain` as small and empty until a stable shared contract exists.
- Keep concrete adapter wiring in `cmd/service/internal/bootstrap`.
- Keep feature-specific telemetry local to the owning feature or adapter until an instrument is genuinely shared.
- Keep the template domain-neutral unless the maintainer explicitly chooses a runnable exemplar domain.

## Decisions

1. Use a non-breaking guidance-plus-guardrails improvement path.
   The implementation should not add a fake business feature. It should add precise first-feature guidance and small tests/checks that prevent the easiest boundary mistakes.

2. Treat `ping_history` as a SQLC fixture, not a production pattern.
   Strengthen docs and comments around the existing sample. Do not rename active migrations, query names, generated files, or exported repository names in the first implementation pass unless the maintainer explicitly accepts the churn.

3. Add a production-shaped first-feature checklist.
   The checklist should cover the full path: `internal/app/<feature>` app-owned port and types, `internal/infra/postgres` repository mapping, `api/openapi/service.yaml` contract, `internal/infra/http` handler mapping and Problem responses, `cmd/service/internal/bootstrap` injection, and tests at each owner.

4. Keep Redis/Mongo as guard-only stubs.
   Redis/Mongo policy-shaped config keys must not become the source of cache/store semantics. Real cache/store behavior should be introduced only through an app-owned feature need plus `internal/infra/<integration>` adapter design.

5. Clarify protected endpoint wiring without inventing auth.
   Docs should say that the first protected operation needs a real security design and a scoped HTTP-layer/generated/strict seam with public-route non-regression tests. They should not imply placeholder auth or a broad root middleware.

6. Add an actual HTTP route-tree guard.
   Existing manual route tests should be supplemented with a route-tree check that catches future direct manual root routes outside the audited helper, especially `/api/...` registrations and undocumented generated/manual overlaps.

7. Add an app boundary guardrail.
   The guardrail should fail if `internal/app` or `internal/domain` imports `internal/infra/*`, `internal/infra/postgres/sqlcgen`, or concrete DB driver packages such as `github.com/jackc/pgx`.

8. Fix dependency-probe startup rejection logging.
   `recordDependencyProbeRejection` should include the error value in the `startup_blocked` log envelope, matching sibling rejection helpers. Add or adjust a focused bootstrap test.

9. Canonicalize `Allow` header emission.
   The HTTP router currently adds one `Allow` header value per method. That is legal, but a single comma-separated value is easier for clients and `Header.Get`-based tests. If route-policy tests are touched, canonicalize the header and update assertions.

10. Do not rename broad helper files as part of this pass.
   `startup_probe_helpers.go` has a naming smell because it contains general lifecycle helpers, but a rename/split is lower value than the onboarding and guardrail fixes. Defer until the file is next touched for related work.

11. Do not centralize telemetry failure reason vocabulary in this pass.
   Bootstrap and telemetry currently duplicate the small `setup_error` / `deadline_exceeded` / `canceled` vocabulary. This is low-risk relative to the template-readiness gaps. Leave it as a deferred cleanup unless the implementation already touches both bootstrap telemetry init and telemetry metrics.

## Open Questions / Assumptions

- [assumption] The maintainer prefers the template to remain domain-neutral rather than shipping a runnable fake business domain.
- [assumption] The first pass should avoid `ping_history` schema/query renames because it would create migration/codegen churn without changing runtime behavior.
- [requires_user_decision] If the maintainer wants the SQLC fixture to be impossible to mistake for real state, a future task can rename the migration/table/query/repository sample deliberately.
- [requires_user_decision] Real auth policy remains outside this task; protected endpoint guidance must not choose identity, tenant, token, session, or authorization behavior.

## Research Coverage Map

| Review point | Disposition |
| --- | --- |
| Strong app/domain/infra/bootstrap folder model | Preserve in constraints and `design/ownership-map.md`. |
| Empty `internal/domain` resists speculative abstraction | Preserve in constraints and ownership rules. |
| OpenAPI-first generated routing boundary | Preserve; no generated-code hand edits. |
| SQLC generated boundary and repository mapping | Preserve; no SQLC hand edits. |
| Missing production-shaped vertical feature path | Planned in T001-T002 and T006. |
| `ping_history` active schema and production-looking names | Planned as stronger fixture/replacement guidance in T003; schema/query/generated renames require user decision. |
| No runnable app-owned port plus Postgres adapter example | Planned as checklist/docs guidance in T002; no fake domain implementation. |
| DB-required feature bootstrap proof shape is not demonstrated | Planned as docs/test guidance in T002 and T006. |
| Redis/Mongo cache/store policy-shaped config stubs | Planned as guard-only clarification in T004; adapters remain out of scope. |
| `MongoProbeAddress` exposes probe-address derivation from `internal/config` | Covered by T004 as guard-only/temporary semantics; moving helper ownership is deferred until real Mongo adapter work exists. |
| Protected endpoint middleware seam not demonstrated | Planned as security-design-dependent guidance in T005; no placeholder auth. |
| Manual route guard checks helper, not final chi tree | Planned as route-tree guard in T008. |
| Multiple `Allow` header values | Planned as canonical header behavior in T009. |
| App/domain imports could drift into infra/sqlcgen/pgx | Planned as guardrail in T007. |
| `recordDependencyProbeRejection` omits `err` in logs | Planned as implementation plus test in T010-T011. |
| `startup_probe_helpers.go` filename invites helper junk drawer | Explicitly deferred in Decision 10 until the file is touched for related work. |
| Telemetry init reason vocabulary duplicated | Explicitly deferred in Decision 11 as low risk for this pass. |
| Feature telemetry could drift into shared central metrics | Covered by T002 documentation and `design/ownership-map.md`; feature metrics start with the owning feature or adapter. |
| Integration trigger for endpoint plus persistence plus bootstrap | Planned in T006. |
| `internal/config` missing from test placement matrix | Planned in T006. |
| Production migration safety beyond rehearsal | Already documented in `docs/project-structure-and-module-organization.md`; no new task unless migrations change. |
| Preserve test layering and validation commands | Preserved in `plan.md`, `tasks.md`, and validation phase file. |

## Plan Summary / Link

Implementation should follow `plan.md` and the executable ledger in `tasks.md`.

## Validation

Required proof for the planned implementation:

- `go test ./cmd/service/internal/bootstrap ./internal/infra/http`
- `go test ./...`
- `make guardrails-check` if the repository guardrail script changes
- `make openapi-check` only if `api/openapi/service.yaml`, OpenAPI generation config, or generated API artifacts change
- `make sqlc-check` only if migrations, SQLC queries, or generated SQLC artifacts change

Completed evidence on 2026-04-12:

- `make guardrails-check`: passed.
- `go test ./cmd/service/internal/bootstrap`: passed.
- `go test ./internal/infra/http`: passed.
- `go test -count=1 ./cmd/service/internal/bootstrap ./internal/infra/http`: passed.
- `go test -count=1 ./...`: passed.
- `make openapi-check`: not run because no OpenAPI source, generation config, or generated API artifact changed.
- `make sqlc-check`: not run because no migration, SQL query, or generated SQLC artifact changed.

## Outcome

Implemented. The template now links future feature authors to a first-production-feature checklist, clarifies first-feature placement and proof obligations, documents `ping_history` as a replaceable SQLC fixture, clarifies Redis/Mongo guard-only semantics and `MongoProbeAddress` ownership, strengthens protected-operation guidance, adds app/domain import guardrails, tightens root route-tree policy tests, canonicalizes `Allow` header emission, and logs dependency-probe rejection errors in the startup-blocked envelope.

No fake business domain, placeholder auth, `ping_history` rename, Redis/Mongo runtime adapter, OpenAPI source change, migration/query change, or generated-code hand edit was introduced.
