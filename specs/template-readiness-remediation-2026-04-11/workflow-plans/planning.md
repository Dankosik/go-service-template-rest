# Planning Phase Workflow Plan

## Scope

Turn the accepted review findings, follow-up research findings, Nice To Have items, and explicit no-op/avoid decisions into an implementation-ready context bundle, without changing production repository behavior.

## Local Orchestration

- Phase: planning.
- Phase status: complete.
- Research mode: local.
- Phase-collapse waiver: lightweight local; specification, compact technical design, and implementation planning are intentionally produced together because the findings are already accepted and bounded.
- Completion marker: `spec.md`, required compact `design/` files, `plan.md`, `tasks.md`, `research/finding-coverage.md`, and `workflow-plans/implementation-phase-1.md` through `workflow-plans/implementation-phase-3.md` exist and agree.
- Stop rule: do not implement code/docs/test changes in this session.

## Inputs Used

- User-provided accepted P2 findings.
- Additional review research findings and Nice To Have items from the completed read-only review synthesis.
- Completed review workflow: `specs/template-extension-readiness-review-2026-04-11/`.
- Repository baseline docs: `docs/spec-first-workflow.md`, `docs/repo-architecture.md`, `docs/project-structure-and-module-organization.md`.
- Targeted reads of `Makefile`, `.github/workflows/ci.yml`, `internal/config`, `internal/infra/http`, and Postgres sample tests.

## Planning Decisions

- Use three implementation phases because the full task now covers docs/onboarding, config/HTTP guardrails, and Postgres/bootstrap/artifact cleanup. This keeps each implementation session reviewable while still covering all research findings.
- Keep docs and guardrail edits close to existing source-of-truth files rather than adding new broad policy documents.
- Prefer test/guardrail changes that make future drift fail loudly.
- Do not introduce new abstraction packages.

## Implementation Readiness

- Status: PASS for later implementation.
- Constraint: implementation must follow the active implementation phase file, `tasks.md`, and `research/finding-coverage.md`, and must not restore or edit unrelated deleted `specs/template-readiness-*` paths.
- Proof path: targeted checks per task, then `make check` if feasible.
