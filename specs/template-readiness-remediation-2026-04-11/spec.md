# Template Readiness Remediation Spec

## Context

The prior review found that the template is mostly ready to clone and extend, but five headline P2 gaps and several lower-priority research findings can still lead future contributors or coding agents into ad hoc growth:

1. Missing worked feature path.
2. Config key additions can miss `buildSnapshot`.
3. API runtime contract gate is narrower than the HTTP policy guardrails.
4. Redis/Mongo look more implemented than they are.
5. Migration docs omit online production safety guidance.

The implementation scope must also close the related research findings and Nice To Have items captured in `research/finding-coverage.md`.

This task prepares the implementation context only. Code/docs/test changes are intentionally deferred.

## Scope

Implement later:

- A short worked feature path in `docs/project-structure-and-module-organization.md`.
- A config test or equivalent proof that every leaf config key reaches the runtime snapshot.
- A widened OpenAPI runtime contract check, plus doc updates so contributors understand compatibility proof.
- Explicit Redis/Mongo guard-only extension-stub guidance.
- Online migration safety guidance.
- Related docs, command, test, and maintainability follow-ups from the review research, including Nice To Have items.

## Non-Goals

- No implementation in this session.
- No broad rewrite of README or workflow docs.
- No clean-architecture taxonomy.
- No generic helper packages or new abstraction frameworks.
- No real Redis/Mongo adapter implementation.
- No runtime behavior change except test/guardrail behavior needed to prove template conventions.
- No package rename unless explicitly chosen for a local package-doc clarity fix.

## Constraints

- Keep the repo template-oriented; do not overfit to `ping` or `ping_history`.
- Keep generated artifacts derived from their sources.
- Keep `internal/domain` small and empty until real shared contracts exist.
- Keep app-facing ports beside `internal/app/<feature>` consumers by default.
- Keep manual HTTP root routes exceptional and documented.
- Keep config parsing semantics explicit; do not replace them with reflection-based mapping unless error behavior remains as clear and tested.

## Decisions

### 1. Worked Feature Path

Add one concise `Where to Put New Code` subsection that shows vertical feature flow before the individual endpoint/persistence recipes.

Include three worked paths:

- Simple read-only endpoint: app use case -> OpenAPI contract -> generated API -> strict handler method -> HTTP contract tests.
- Postgres-backed endpoint: app use case and app-owned port -> migration -> sqlc query -> handwritten Postgres repository -> bootstrap injection -> repository and integration tests.
- Background job or worker: `internal/app/<feature>` behavior -> `internal/infra/<integration>` mechanics -> `cmd/<binary>` composition root for lifecycle.

Keep this as a practical map, not a new architecture doctrine.

Also add the missing adjacent guidance from the research pass:

- A short domain-type rule: keep feature-local types in `internal/app/<feature>` until two app packages need the same abstraction or the type represents a stable cross-adapter contract.
- A compact test placement matrix: app/domain unit tests, HTTP contract/policy tests, Postgres repository tests, migration/container tests, and broader `test/` scenarios.
- A small telemetry placement rule: HTTP edge metrics/traces stay in `internal/infra/http`/`internal/infra/telemetry`; feature-specific instrumentation must use low-cardinality labels and should not put feature semantics into shared telemetry code unless the instrument is genuinely shared.

### 2. Config Snapshot Proof

Add same-package test coverage in `internal/config` proving that every leaf key from `defaultValues()` and `Config` koanf tags actually reaches the `Config` snapshot.

Preferred approach:

- Build a sentinel config source that assigns distinguishable, valid values for each known key.
- Load/build a snapshot from that source.
- Flatten the resulting `Config` by koanf tags or explicit test helper and compare observed values to expected sentinels.
- Keep existing explicit parse helpers and validation behavior.

Avoid a generic reflection mapper in production code unless it preserves today-specific parse errors and validation semantics.

### 3. API Runtime Contract Gate

Widen `openapi-runtime-contract-check` so `make openapi-check` includes all API/HTTP policy tests that protect generated-route ownership, fallback behavior, OPTIONS/CORS policy, manual root-route exceptions, and route labels.

Acceptable implementation options:

- Rename relevant tests under the `TestOpenAPIRuntimeContract...` prefix.
- Or widen the Makefile regex to include the existing test names.

Also update docs so `make openapi-check` is described accurately. It proves generation, drift, runtime contract, lint, and schema validation. Breaking compatibility remains `BASE_OPENAPI=<base> make openapi-breaking` or the PR CI breaking-change job.

Tighten manual root-route policy as part of the same API/HTTP guardrail work. Manual root routes should be exceptional, registered through a single documented exception surface, and covered even when they do not overlap generated OpenAPI routes.

When documenting route-label guardrails, note that future parameterized routes must prove logs, metrics, and spans use route templates rather than concrete IDs.

### 4. Redis/Mongo Stub Clarity

Document Redis and Mongo as guard-only extension stubs unless and until a real feature owns adapter behavior.

The wording should say:

- Config keys and bootstrap probes validate planned dependency exposure and readiness policy.
- They are not full cache/store/database adapters.
- Production behavior must add `internal/infra/redis` or `internal/infra/mongo` only when a real app feature needs it.
- Do not add more cache/store semantics to `internal/config` or bootstrap alone.

### 5. Online Migration Safety

Extend migration guidance with production safety rules:

- Prefer additive/expand-first migrations for online systems.
- Destructive changes, column type changes, large backfills, and new constraints/indexes need an explicit plan.
- Consider lock behavior and table size before adding indexes or constraints.
- `make migration-validate` rehearses migration mechanics; it does not prove lock, backfill, or mixed-version safety.
- Escalate schema ownership, retention, backfill, or rollout questions to data-architecture design before coding.

### 6. Data Sample And Repository Guidance

Close the lower-priority data findings by documenting or tightening sample behavior:

- Add or document an upper-bound rule for list limits before passing values to SQL `LIMIT`.
- Add a short transaction recipe: begin with caller context, bind sqlc queries to the transaction, defer rollback with a bounded cleanup context, commit with caller context, and avoid a generic transaction helper until real repetition exists.
- Add DB-backed feature guidance: construct repositories only after the dependency is enabled and initialized, inject through app-owned ports, and add bootstrap tests for disabled, ready, and cleanup paths.
- Strengthen sample Postgres assertions for payload and timestamp mapping if implementation touches those tests.

### 7. Maintainability And Test Hygiene

Close the Nice To Have and maintainability findings:

- Remove or clarify `enforceSecretSourcePolicy`'s ignored local-environment parameter so secret-file rejection remains visibly environment-independent.
- Reduce startup dependency label drift by replacing clusters of same-typed label strings with a small same-package label/spec structure or another compiler-aided local shape.
- Move the test-only `newPingHistoryRepositoryWithQuerier` helper out of production code or otherwise remove it as a visible runtime extension surface.
- Fix or explicitly harden `TestServerRunAndShutdown` so it does not teach reserve-close-listen polling as a lifecycle-test pattern.
- Fix or explicitly harden `TestNonLocalDefaultRootsDoNotAllowRepositoryConfigDir` so it does not write temporary files into the real repository path.
- Refresh the stale project tree in the structure doc, including `specs/` and generated/report artifacts, without turning the tree into a maintenance burden.
- Decide `.artifacts/test/*` tracking: untrack and ignore generated reports unless there is a deliberate sample-artifact reason.
- Decide the `internal/infra/http` package-name convention. Preferred closeout is a short package doc or docs note if renaming is not worth the churn; do not rename casually.

## Open Questions / Assumptions

- Assume docs should stay concise and live in existing docs, not a new documentation tree.
- Assume Redis/Mongo stay stubs for this remediation; no runtime adapters will be added.
- Assume the next implementation session should address all mapped findings in `research/finding-coverage.md`, not just the five inline P2 findings.
- Assume `httpx` package renaming is not required unless implementation chooses it deliberately; documenting the convention is acceptable closeout.

## Plan Summary / Link

See `plan.md` and `tasks.md` in this task directory.

## Validation

Expected after implementation:

- Focused config tests for snapshot round-trip proof.
- Focused HTTP tests via the widened `openapi-runtime-contract-check`.
- Focused Postgres sample tests if assertion improvements are included.
- Focused HTTP server/config tests if lifecycle-test or config-test hygiene is changed.
- Docs review for consistency of `openapi-check`, `openapi-breaking`, migration safety, config extension, and feature placement.
- `make check` when feasible.
