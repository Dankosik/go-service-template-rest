# Template Readiness Review Workflow Plan

## Task Frame

- Goal: produce a read-only, subagent-backed review of whether this Go REST service template is ready to be cloned and extended with production business code.
- Scope: package and folder organization, extension seams, helper duplication, generated API/HTTP boundaries, Postgres/sqlc/migration path, config and telemetry placement, test layout, onboarding documentation, and practical recommendations for template users.
- Non-goals: no business feature implementation, no refactor, no code generation, no migration changes, no test rewrites, and no generic clean-architecture ceremony unless it materially improves the template.
- Constraints: subagents must remain read-only and advisory; final recommendations belong to the orchestrator; findings must be evidence-backed with repository paths; do not invent missing conventions.

## Execution Control

- Execution shape: full orchestrated, because the user explicitly requested subagent-backed review and the question spans architecture, HTTP/API, data, QA, maintainability, and docs/onboarding surfaces.
- Current phase: research.
- Current phase status: complete.
- Research mode: fan-out.
- Phase workflow plan: `workflow-plans/research.md` active.
- Session-boundary note: this is a review-only task rather than an implementation/specification task. The phase collapse is limited to research fan-out plus orchestrator synthesis into the requested chat report; no `spec.md`, `design/`, `plan.md`, `tasks.md`, implementation, or validation phase is expected in this pass.
- Workflow plan adequacy challenge: complete; no blocking adequacy gaps found by read-only challenger.
- Domain fan-out: complete; architecture/design, maintainability/helper, API/HTTP, data, QA, and docs/onboarding lanes returned.

## Artifact Status

- `spec.md`: not expected for this analysis-only review.
- `design/`: not expected; the output is recommendations, not a task-local design.
- `plan.md`: not expected unless the user later asks to implement fixes.
- `tasks.md`: not expected unless the user later asks to implement fixes.
- `test-plan.md`: not expected.
- `rollout.md`: not expected.
- `research/*.md`: optional; preserve only if subagent evidence needs durable handoff beyond the final report.
- Implementation/review/validation phase-control files: not expected for this pass.

## Blockers And Assumptions

- Blockers: none.
- Assumption: because the user requested the full review outcome in this pass, the research phase may synthesize directly into the final chat report while remaining read-only and avoiding implementation artifacts.
- Assumption: repository documentation should be treated as part of the template surface, not as an external explanation.

## Planned Lanes

- Workflow adequacy challenge: `challenger-agent`; question: are these workflow-control artifacts sufficient for this read-only fan-out; skill: `workflow-plan-adequacy-challenge`.
- Architecture/design lane: `architecture-agent`; question: are package boundaries and extension seams clear for future business code; skill: `go-design-review`.
- Maintainability/helper lane: `quality-agent`; question: where do repeated helper patterns, naming, or cohesion gaps risk ad hoc growth; skill: `go-language-simplifier-review`.
- API/HTTP lane: `api-agent`; question: is the OpenAPI/generated/chi handler extension path and transport boundary clear; skill: `go-chi-review`.
- Data lane: `data-agent`; question: is the Postgres/sqlc/migration/repository extension path clear and cohesive; skill: `go-db-cache-review`.
- QA lane: `qa-agent`; question: does the test layout teach future business-feature validation well enough; skill: `go-qa-review`.
- Docs/onboarding lane: `quality-agent`; question: do README/docs explain where new production code goes; skill: no-skill.

## Handoff

- Next action: deliver synthesized review report.
- Completion marker: domain lane evidence has been synthesized into a structured template-readiness report with prioritized recommendations and non-recommendations.
- Ready for next session: yes, if the user asks to turn recommendations into implementation work.
- Next session starts with: new implementation-planning task for selected recommendations, or done if no follow-up is requested.
