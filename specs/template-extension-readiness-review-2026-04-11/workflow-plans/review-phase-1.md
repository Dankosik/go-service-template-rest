# Review Phase 1 Workflow Plan

## Scope

Read-only, subagent-backed review of template readiness for future production business-code integration.

## Local Orchestration

- Phase: review-phase-1.
- Phase status: complete.
- Research mode: fan-out.
- Fan-out style: independent read-only lanes, then orchestrator synthesis.
- Order: run Adequacy challenge first; reconcile any blocking findings in this file and `workflow-plan.md`; only then launch Architecture/design, Go maintainability, API/HTTP, Data, QA, and Docs/onboarding lanes in parallel; fan-in after those review lanes return.
- Next action: launch review lanes in parallel after adequacy reconciliation.
- Completion marker: adequacy challenge reconciled, review lanes returned, repository evidence checked locally, and final report prepared for chat delivery with prioritized recommendations.
- Stop rule: no code edits, no refactors, no generated artifacts, no tests unless the orchestrator determines a lightweight read-only baseline check is useful.

## Lanes

| Lane | Agent | Skill | Required evidence |
| --- | --- | --- | --- |
| Adequacy challenge | challenger-agent | workflow-plan-adequacy-challenge | Findings on task-specific sufficiency of `workflow-plan.md` and this file; exact additions if blocked. |
| Architecture/design | architecture-agent | go-design-review | File/path evidence for boundary clarity or drift across `cmd/service/internal/bootstrap`, `internal/app`, `internal/domain`, `internal/infra/*`, docs, and extension seams. |
| Go maintainability | quality-agent | go-language-simplifier-review | Evidence of helper duplication, unclear package cohesion, over/under-abstraction, naming/style drift, and actionable simplification opportunities. |
| API/HTTP | api-agent | go-chi-review | Evidence from `api/openapi`, `internal/api`, and `internal/infra/http` about generated-route ownership, handler extension path, middleware boundaries, errors, and route policy. |
| Data | data-agent | go-db-cache-review | Evidence from `env/migrations`, `internal/infra/postgres`, sqlc query/generated layout, repository mapping, transaction/context/resource safety, and extension guidance. |
| QA | qa-agent | go-qa-review | Evidence from test layout, `test/README.md`, existing unit/integration tests, and validation docs for future business feature coverage. |
| Docs/onboarding | explorer | no-skill | Evidence from README, CONTRIBUTING, architecture docs, project-structure docs, and internal READMEs on whether new contributors know where to add common code. |

## Fan-In And Synthesis

The orchestrator will compare subagent claims against local repository evidence, resolve conflicts, avoid overfitting to the sample `ping` feature, and produce the final review report with:

- overall readiness verdict;
- strongest parts;
- concrete gaps;
- duplicated or unclear helper patterns;
- boundary issues with file/path evidence;
- must-fix, should-fix, and nice-to-have recommendations;
- suggested "where to put new code" conventions;
- over-abstractions to avoid.

## Local Blockers

- None known.
- The worktree already contains deleted tracked files under older `specs/template-readiness-*` paths; this review must not restore or modify those paths.

## Adequacy Challenge

- Status: complete.
- Resolution: blocking sequencing finding reconciled by making the adequacy challenge an explicit pre-fan-out gate and updating the local next action.

## Closeout

- Review lanes returned: Architecture/design, Go maintainability, API/HTTP, Data, QA, and Docs/onboarding.
- Local validation: static/read-only evidence gathering with `rg`, `nl`, `git ls-files`, and targeted file reads.
- Tests: not run; this was not a behavioral change and test success would not prove architectural readiness.
- Next action: user decides which recommendations to implement in a later pass.
