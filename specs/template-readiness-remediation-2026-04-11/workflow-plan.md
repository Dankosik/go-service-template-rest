# Template Readiness Remediation Workflow Plan

## Task Frame

Remediate the accepted template-readiness review findings through the approved phased implementation plan.

The prior planning session prepared the handoff bundle without implementation. The implementation sessions completed Phase 1 docs/onboarding tasks T001-T015, Phase 2 config/HTTP guardrail tasks T016-T022, and Phase 3 data/bootstrap/artifact/validation tasks T023-T028.

## Execution Shape

- Shape: phased implementation over the approved lightweight local handoff bundle.
- Current phase: implementation-phase-3.
- Current phase status: complete.
- Research mode: local, using the completed review fan-out from `specs/template-extension-readiness-review-2026-04-11/` plus targeted source reads.
- Subagents: not used in this pass; the user did not request new subagent work, and the prior review already supplied multi-lane evidence.
- Historical phase-collapse waiver: approved for the prior task-local context pass only. Phase 1 implementation consumed the approved handoff and did not reopen planning.
- Coding status: Phase 3 complete; final validation evidence is recorded in `tasks.md`.

## Artifact Status

- `workflow-plan.md`: approved.
- `workflow-plans/planning.md`: approved.
- `workflow-plans/implementation-phase-1.md`: approved for docs/onboarding remediation.
- `workflow-plans/implementation-phase-2.md`: approved for config and HTTP guardrail remediation.
- `workflow-plans/implementation-phase-3.md`: approved for Postgres/bootstrap/artifact cleanup and validation.
- `spec.md`: approved.
- `design/overview.md`: approved.
- `design/component-map.md`: approved.
- `design/sequence.md`: approved.
- `design/ownership-map.md`: approved.
- `plan.md`: approved.
- `tasks.md`: approved; expanded to cover all research findings, including Nice To Have and residual-risk items; T001-T028 complete with Phase 1, Phase 2, and Phase 3 validation evidence.
- `research/finding-coverage.md`: approved; maps every review/research finding to a task or explicit no-op decision; Phase 1, Phase 2, and Phase 3 closeout notes recorded.
- `test-plan.md`: not expected; validation obligations fit in `plan.md` and `tasks.md`.
- `rollout.md`: not expected; this is template/docs/test guardrail work with no runtime rollout.

## Accepted Findings In Scope

1. Add a worked feature path in `docs/project-structure-and-module-organization.md`.
2. Prove config keys reach the runtime snapshot.
3. Widen the API runtime contract gate.
4. Clarify Redis/Mongo as guard-only extension stubs.
5. Add online migration safety guidance.
6. Close all additional review research findings, including docs/onboarding, data, API/HTTP, QA, maintainability, and Nice To Have items recorded in `research/finding-coverage.md`.

## Out Of Scope

- Phase 1 did not include code, test, or Makefile changes; Phase 2 completed the config and HTTP guardrail changes.
- Restoring or editing the already-deleted tracked `specs/template-readiness-*` files visible in the dirty worktree.
- Adding generic architecture layers, `common`/`util` packages, service registries, generic repository interfaces, or generic migration frameworks.
- Promoting Redis or Mongo into real runtime adapters without a separate feature decision.

## Workflow Gate Status

- Workflow plan adequacy challenge: waived under the lightweight local phase-collapse waiver.
- Spec clarification challenge: waived under the same local-prep waiver; the findings are already accepted and do not require new product decisions.
- Implementation readiness: PASS for later phased implementation sessions, constrained to `tasks.md`.
- Coverage repair status: complete; every known review finding is mapped in `research/finding-coverage.md`, and Phase 3 closeout confirms no generic helpers/frameworks/adapters were added.

## Session Boundary

- Session boundary reached: yes.
- Ready for next session: no.
- Next session starts with: none; remediation implementation is complete unless the user requests review, commit, or follow-up work.
- Stop rule: stop after Phase 3 closeout.
