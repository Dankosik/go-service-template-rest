# Template Readiness Review Workflow Plan

## Master Control

- Task: repository-grounded review of readiness as a reusable production Go REST service template for future business-code development.
- Execution shape: full orchestrated, because the user explicitly requested subagents and the review crosses architecture, HTTP/API, data, reliability, QA, and code-quality boundaries.
- Current phase: review-phase-1.
- Phase status: complete.
- Session boundary reached: yes.
- Ready for next session: yes.
- Next session starts with: user decision on which recommended template-readiness improvements to implement, if requested later.

## Scope

- In scope: read-only repository inspection, domain-specific subagent review lanes, lightweight validation commands when useful, and final template-readiness synthesis with concrete recommendations.
- Out of scope: implementation, refactors, generated-code updates, migrations, or repository behavior changes.
- Allowed writes: this workflow-control file and `workflow-plans/review-phase-1.md` only.

## Artifact Status

- `workflow-plan.md`: draft, current review pass complete.
- `workflow-plans/review-phase-1.md`: draft, current review pass complete.
- `spec.md`: not expected for this review-only pass; no implementation decision record is being approved.
- `design/`: not expected; the review consumes stable repository docs instead of creating task-local design.
- `plan.md`: not expected; no implementation plan is requested.
- `tasks.md`: not expected; no executable coding ledger is requested.
- `test-plan.md`: not expected.
- `rollout.md`: not expected.
- `research/*.md`: not expected by default; final synthesis is delivered in chat unless the phase is interrupted.

## Review Lanes

- Workflow adequacy: `challenger-agent` with `workflow-plan-adequacy-challenge`; review this master file and `workflow-plans/review-phase-1.md` before treating the fan-out plan as sufficient.
- Architecture: `architecture-agent` with `go-design-review`; package boundaries, ownership, extension seams, template guidance for new business code.
- Quality: `quality-agent` with `go-language-simplifier-review`; duplicated helpers, scattered policy, naming, helper extraction risk, same-package source-of-truth opportunities.
- HTTP/API: `api-agent` with `go-chi-review`; chi/router/OpenAPI integration boundaries and endpoint-extension path.
- Data: `data-agent` with `go-db-cache-review`; Postgres/sqlc/migration/repository boundaries and persistence-extension path.
- Reliability: `reliability-agent` with `go-reliability-review`; bootstrap/config/shutdown/readiness template guidance.
- QA: `qa-agent` with `go-qa-review`; test layout, validation commands, integration-test guidance, future feature-test placement.
- Docs/Make local inspection: orchestrator-owned `no-skill` lane for README, project-structure docs, command discoverability, and whether documented generated-code/test commands match template extension paths.
- Security: `security-agent` with `go-security-review`; required for completion because architecture and local inspection raised auth/CORS/trust-boundary extension concerns.

## Validation Expectations

- Fresh repository evidence gathered with `rg`, `sed`, `nl`, and targeted file reads across the requested paths.
- Local inspection used `rg`, `find`, `sed`, `nl`, `go list`, and targeted file reads across the requested paths.
- `go test ./cmd/service/internal/bootstrap ./internal/config ./internal/infra/http ./internal/infra/postgres ./internal/infra/telemetry ./internal/app/...`: passed.
- `make check`, `make openapi-check`, `make sqlc-check`, integration tests, and migration rehearsal: not run in this pass; the review did not make full baseline, OpenAPI drift, sqlc drift, or live migration correctness claims.

## Blockers

- None; review phase complete.

## Workflow Plan Adequacy Challenge

- Status: complete; blocking handoff/lane findings reconciled.
- Resolution: master next-session routing now starts before fan-out/fan-in, the security lane is required rather than conditional, and stale Russian-report wording was removed from the active phase plan.

## Resume Order

1. Read this file.
2. Read `workflow-plans/review-phase-1.md`.
3. Read `AGENTS.md`, `docs/spec-first-workflow.md`, and `docs/repo-architecture.md`.
4. If implementation is requested later, start from the final review report and decide which recommendations to turn into a separate spec/planning pass.
