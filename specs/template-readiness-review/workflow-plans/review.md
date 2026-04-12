# Review Phase Plan

## Phase Scope

- Phase: review.
- Status: complete.
- Research mode: fan-out plus local repository inspection.
- Deliverable: concise but substantive template-readiness review report in chat.
- Out of scope: code edits, generated-code edits, implementation planning, broad rewrite proposals, or generic Go style review detached from template readiness.

## Order And Parallelism

1. Run `workflow-adequacy` first and reconcile any blocking findings before treating this phase plan as sufficient.
2. Run architecture, maintainability, API transport, data, and QA lanes in parallel.
3. Perform local repository inspection of top-level docs, package tree, bootstrap/app/domain/infra/config/API/data/test surfaces, and selected tests while subagents run.
4. Synthesize comparable claims across lanes, separating evidence-backed issues from opportunities and existing strengths.
5. Run `synthesis-challenge` after initial synthesis to pressure-test missing template-readiness risks.
6. Reconcile challenger output and produce the final review report.

## Lanes

| Lane | Role | Skill | Scope |
| --- | --- | --- | --- |
| workflow-adequacy | `challenger-agent` | `workflow-plan-adequacy-challenge` | Check only whether `workflow-plan.md` and this phase plan are sufficient for the requested agent-backed review. |
| architecture | `architecture-agent` | `go-design-review` | Review `docs/repo-architecture.md`, `docs/project-structure-and-module-organization.md`, `cmd/service/internal/bootstrap`, `internal/app`, `internal/domain`, and relevant `internal/infra/*` boundaries for future business-use-case placement. |
| maintainability | `quality-agent` | `go-language-simplifier-review` | Review helper/source-of-truth clarity, naming, package-local patterns, and whether future contributors can imitate style without duplicating utilities. |
| api-transport | `api-agent` | `go-chi-review` | Review `api/openapi/service.yaml`, `internal/api`, and `internal/infra/http` for OpenAPI/generated/chi integration and endpoint extension clarity. |
| data | `data-agent` | `go-db-cache-review` | Review `internal/infra/postgres`, SQLC query/generated boundaries, `env/migrations`, and data-related docs for persistence extension clarity. |
| qa | `qa-agent` | `go-qa-review` | Review package tests and `test` support for obvious proving layers for business logic, HTTP transport, persistence, and integration tests. |
| synthesis-challenge | `challenger-agent` | `pre-spec-challenge` | Challenge the orchestrator's synthesized findings for missing risks, overfitting to `ping`, weak evidence, and misplaced recommendations. |

## Local Inspection Checklist

- Top-level tree, README, and docs that claim where new code goes.
- `cmd/service/internal/bootstrap`.
- `internal/app`, `internal/domain`, `internal/infra/http`, `internal/infra/postgres`, `internal/infra/telemetry`, and `internal/config`.
- `api/openapi/service.yaml` and generated `internal/api` ownership signals.
- `env/migrations`, SQLC query and generated-code surfaces.
- Package tests and integration test structure under `test`.

## Stop Rule

Stop after the final review report and workflow-control closeout updates. If a finding would require code changes, record it as a recommendation instead of implementing it.

## Completion Marker

- Adequacy gate: reconciled.
- Review lanes: returned and compared, or explicitly limited.
- Challenger lane: returned and reconciled.
- Fresh evidence: repository inspection commands recorded in the final report.
- Output: final review report with concrete path references, recommendations ordered by impact, suggested "where new code goes" guidance, and open questions.

## Current Status

- Adequacy gate: complete; no blocking findings, recordable wording findings reconciled in `workflow-plan.md`.
- Initial fan-out: complete; architecture, maintainability, API transport, data, and QA lanes returned.
- Local inspection: complete; focused tree, docs, code, tests, Makefile, and generated-boundary surfaces inspected.
- Synthesis challenge: complete; recommendations reconciled into the final report.
- Final report: complete; returned in chat.

## Closeout

- Fresh command evidence: `go test -count=1 ./...` passed for non-integration packages.
- Mutation boundary: no implementation, generated-code, git-state, or runtime config changes were made.
- Next action: if maintainers accept any recommendation, reopen into a new spec/planning task rather than treating this advisory review as an implementation plan.
