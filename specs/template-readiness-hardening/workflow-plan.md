# Template Readiness Hardening Workflow Plan

## Task Frame

- Goal: prepare implementation-ready context for fixing the actionable template-readiness findings from `specs/template-readiness-review` without changing production code in this session.
- Source review bundle: `specs/template-readiness-review/workflow-plan.md` and `specs/template-readiness-review/workflow-plans/research.md`.
- Must-fix findings in scope:
  - OpenAPI runtime-contract Makefile target misses the security-decision guard test.
  - `internal/api/README.md` underspecifies protected endpoint auth placement.
  - `PingHistoryRepository.ListRecent` does not model a bounded SQL `LIMIT`.
  - Redis store-mode readiness policy is split between config validation and bootstrap runtime wiring.
- Additional research points are covered in `research/coverage-audit.md` as planned implementation, explicit deferral, already-covered guidance, or rejected over-abstraction.
- Non-goals:
  - No implementation in this session.
  - No new business feature.
  - No generic DI container, repository registry, transaction manager, `common` package, broad auth framework, or Redis adapter.
  - No broad README rewrite, startup failure recorder, log-label taxonomy change, degraded dependency logging helper, or bootstrap assembly cleanup in this task.

## Execution Control

- Execution shape: lightweight local with explicit phase-collapse waiver.
- Waiver rationale: the task is a bounded hardening follow-up from completed read-only fan-out. It touches several files, but the decisions are local, source evidence is already available, and the user explicitly asked for pre-implementation context now with implementation deferred to a later session.
- Current phase: implementation-phase-1.
- Current phase status: complete.
- Research mode: local reuse of prior subagent-backed review evidence; no new subagent fan-out requested in this turn.
- Active phase workflow plan: `workflow-plans/implementation-phase-1.md`.
- Next phase workflow plan: not expected; implementation and validation proof are complete.
- Workflow plan adequacy challenge: waived under lightweight local scope; no new subagent was requested, and the next implementation session has explicit `plan.md` plus `tasks.md`.

## Artifact Status

- `spec.md`: approved.
- `design/overview.md`: approved.
- `design/component-map.md`: approved.
- `design/sequence.md`: approved.
- `design/ownership-map.md`: approved.
- `plan.md`: approved.
- `tasks.md`: approved.
- `research/coverage-audit.md`: approved.
- `test-plan.md`: not expected; validation obligations fit in `plan.md` and `tasks.md`.
- `rollout.md`: not expected; no runtime rollout or migration choreography.
- `workflow-plans/implementation-phase-1.md`: complete.

## Implementation Readiness

- Status: PASS.
- Rationale: all planned research findings have explicit selected fixes, file surfaces, non-goals, deferrals, and verification expectations.
- Accepted risks: none that block implementation.
- Proof obligations:
  - OpenAPI guard is selected by `make openapi-runtime-contract-check`.
  - Protected endpoint guidance clearly preserves OpenAPI source-of-truth, generated routing, scoped auth placement, Problem responses, and public route non-regression.
  - Ping history sample rejects or otherwise bounds excessive list limits before SQL.
  - Redis mode/readiness policy has one narrow config-owned predicate or method consumed by both validation and bootstrap.
  - Documentation updates are reviewed against `research/coverage-audit.md` so every planned research point is either implemented or explicitly left deferred/already-covered/rejected.

## Handoff

- Session boundary reached: yes.
- Ready for next session: no; task is complete.
- Next session starts with: none.
- Stop rule: no further work planned unless a new issue is opened.
- Implementation proof:
  - `make openapi-runtime-contract-check`
  - `go test ./internal/infra/http -count=1`
  - `go test ./internal/infra/postgres -count=1`
  - `go test ./internal/config ./cmd/service/internal/bootstrap -count=1`
  - `make openapi-check`
  - `make check`
