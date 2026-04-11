# Template Readiness Hardening Spec

## Context

The template readiness review found four must-fix hardening items plus supporting documentation and maintainability recommendations. The findings are not requests for new business behavior; they are template-readiness fixes that make existing conventions enforceable and easier to copy safely.

This task prepares the implementation context only. Code and documentation changes are deferred to a later implementation session.

## Scope / Non-goals

In scope:
- Align the OpenAPI runtime-contract test selector with the security-decision guard.
- Clarify protected endpoint auth placement in the generated API README.
- Make the `ping_history` SQLC sample demonstrate bounded list limits before SQL.
- Consolidate Redis store-mode readiness policy under a narrow config-owned source of truth.
- Add the supporting onboarding and placement-guide updates captured in `research/coverage-audit.md` so future authors can find and apply the conventions.

Non-goals:
- No new protected endpoint, security scheme, or auth middleware implementation.
- No unused OpenAPI 401/403 response components unless the implementation session explicitly decides they are necessary for the README guidance or guardrail tests.
- No generic transaction manager, repository registry, DI container, `common` package, or global helper bucket.
- No Redis runtime adapter.
- No broad README rewrite, startup failure recorder extraction, log-label taxonomy change, degraded dependency logging helper, or bootstrap assembly helper.

## Constraints

- Preserve OpenAPI-first ownership: `api/openapi/service.yaml` remains the REST contract source of truth; `internal/api/openapi.gen.go` remains generated output.
- Preserve generated-route ownership: normal API endpoints stay on the generated strict-server path, not manual `/api/...` chi routes.
- Keep auth guidance scoped and future-proof without inventing a real auth design before the first protected endpoint exists.
- Keep the `ping_history` repository clearly marked as a sample, not production business state.
- Keep Redis policy consolidation narrow: the owner should be `internal/config`, while concrete dependency probing and telemetry labels remain in bootstrap.

## Decisions

1. Prefer renaming `TestOpenAPIOperationsDeclareSecurityDecisions` to a `TestOpenAPIRuntimeContract...` name over broadening the Makefile regex.
   - Reason: this keeps `openapi-runtime-contract-check` intentionally narrow and avoids accidentally selecting unrelated future OpenAPI tests.
   - Expected name: `TestOpenAPIRuntimeContractOperationsDeclareSecurityDecisions`.

2. Update `internal/api/README.md` to make the strict-server endpoint recipe app-first and auth-aware.
   - Add an explicit app-use-case step before the handler step.
   - State that `strictHandlers.<Operation>` maps transport to app behavior and should not own business logic.
   - State that protected endpoints require OpenAPI `security`, 401/403 Problem responses, scoped generated/strict middleware or an explicitly designed equivalent, and negative tests for unauthenticated protected calls plus public-route non-regression.
   - Do not add a generic auth framework in this task.

2a. Add documentation discoverability and placement updates where the research found authors could miss the canonical conventions.
   - Add early README pointers to the placement guide for both human and agent quickstarts.
   - Add a CONTRIBUTING pointer to the placement guide.
   - Add command-doc validation scope wording so green commands are not mistaken for architecture readiness.
   - Align `internal/domain` wording in the architecture baseline with the stricter consumer-owned-first rule.
   - Add compact placement-guide recipes for app-only features, first SQL feature replacement of `ping_history`, existing examples to inspect, DB-required bootstrap, and test placement links.
   - Add README migration rehearsal commands and the caveat that skip output is not proof.

3. Tighten protected-operation Problem-response guardrails where they support the endpoint convention.
   - Require canonical `#/components/schemas/Problem` for protected 401/403 checks in the existing helper.
   - Do not add unused OpenAPI response components solely for completeness unless the implementation needs them to express the convention clearly.

4. Add a local maximum for the `ping_history` sample list limit and reject values outside the allowed range.
   - Reason: explicit rejection teaches contract ownership better than silently clamping in an adapter-only sample with no API/app caller.
   - The maximum should be small and template-safe, such as `100`, and owned in `internal/infra/postgres/ping_history_repository.go`.
   - Apply the same validation to `ListRecent` and the transaction sample helper so the sample stays internally consistent.
   - Strengthen sample comments so real production app-facing records/ports are still expected beside `internal/app/<feature>`.

5. Consolidate Redis mode/readiness policy in `internal/config`.
   - Add narrow exported constants and/or methods such as `RedisModeCache`, `RedisModeStore`, `RedisConfig.ModeValue()`, `RedisConfig.StoreMode()`, and `Config.RedisReadinessProbeRequired()`.
   - Use the config-owned API in both validation and bootstrap.
   - Remove or stop using bootstrap's independent `redisStartupMode` normalizer.
   - Keep bootstrap-owned dependency mode labels where they describe startup telemetry, not config policy.

## Open Questions / Assumptions

- Assumption: no current protected endpoint exists, so auth middleware implementation remains out of scope.
- Assumption: a max `ping_history` sample limit of `100` is sufficient unless implementation discovers an existing test expectation that makes a different small value clearer.
- Assumption: adding exported methods inside `internal/config` is acceptable because `internal` keeps the API repository-private.

## Plan Summary / Link

Execution plan: `plan.md`.
Task ledger: `tasks.md`.
Research coverage audit: `research/coverage-audit.md`.

## Validation

Required targeted proof:
- `make openapi-runtime-contract-check` - passed.
- `go test ./internal/infra/http -count=1` - passed.
- `go test ./internal/infra/postgres -count=1` - passed.
- `go test ./internal/config ./cmd/service/internal/bootstrap -count=1` - passed.

Recommended broader proof when local tools are available:
- `make openapi-check` - passed.
- `make check` - passed.

## Outcome

Implemented in `implementation-phase-1`.
