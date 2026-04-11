# Template Readiness Review Research Plan

## Phase Control

- Phase: research.
- Status: complete.
- Research mode: fan-out.
- Output boundary: structured review report in chat; no code, refactor, generated output, `spec.md`, `design/`, `plan.md`, or `tasks.md`.
- Stop rule: stop after the read-only review report; if the user wants fixes, open a later implementation-planning task.

## Research Questions

- Is the package and folder structure clear enough for production business-code extension?
- Would a new developer or coding agent know where to place use cases, domain types, handlers, DB adapters/queries, config options, migrations, telemetry, and tests?
- Are helpers, wiring patterns, or generic utilities repeated in ways that should become clearer local package seams?
- Are boundaries between `cmd`, `internal/app`, `internal/domain`, `internal/infra/*`, generated API code, migrations, and tests crisp enough?
- Does the template establish a style frame that prevents ad hoc growth?
- Which improvements are high-value, and which would be over-abstraction?

## Lanes

| Lane | Execution | Role | Skill | Owned Question | Evidence Targets | Status |
| --- | --- | --- | --- | --- | --- | --- |
| workflow-adequacy | subagent | challenger-agent | workflow-plan-adequacy-challenge | Are the workflow-control artifacts sufficient before fan-out? | `workflow-plan.md`, this file | complete; no blocking adequacy gaps found |
| architecture-design | subagent | architecture-agent | go-design-review | Are boundaries and extension seams clear for future business code? | `docs/repo-architecture.md`, `docs/project-structure-and-module-organization.md`, `cmd/service/internal/bootstrap/`, `internal/app/`, `internal/domain/`, `internal/infra/*` | complete |
| maintainability-helper | subagent | quality-agent | go-language-simplifier-review | Which repeated helper or naming patterns risk ad hoc growth or unclear cohesion? | `internal/config/`, `cmd/service/internal/bootstrap/`, `internal/infra/http/`, `internal/infra/postgres/`, `internal/infra/telemetry/` | complete |
| api-http | subagent | api-agent | go-chi-review | Is the OpenAPI/generated/chi handler extension path and transport boundary clear? | `api/openapi/`, `internal/api/`, `internal/infra/http/` | complete |
| data | subagent | data-agent | go-db-cache-review | Is the Postgres/sqlc/migration extension path clear and safely bounded? | `internal/infra/postgres/`, `env/migrations/`, `test/`, docs mentioning persistence | complete |
| qa | subagent | qa-agent | go-qa-review | Does the test layout teach future feature validation well enough? | package tests, `test/`, docs mentioning checks and integration tests | complete |
| docs-onboarding | subagent | quality-agent | no-skill | Do onboarding docs explain where new production code should go? | `README.md`, `CONTRIBUTING.md`, `docs/*.md`, `internal/api/README.md`, `test/README.md` | complete |

## Order And Fan-In

- Run workflow adequacy first and reconcile blocking findings.
- Run domain lanes in parallel after adequacy is reconciled.
- The orchestrator will compare lane findings, resolve conflicts, and produce the final recommendation set.

## Blockers

- None known.

## Completion Marker

- Complete: all required lanes returned, and the orchestrator is synthesizing the final report with readiness verdict, strengths, concrete gaps, duplicated or unclear helper patterns, boundary issues with path evidence, prioritized recommendations, suggested placement conventions, and over-abstraction warnings.
