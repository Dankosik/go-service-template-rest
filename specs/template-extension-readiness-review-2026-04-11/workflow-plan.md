# Template Extension Readiness Review Workflow Plan

## Task Frame

Review whether this Go REST service template is ready to be cloned and extended with production business code. The review is read-only and must assess folder/package clarity, future feature placement, duplicated helper patterns, boundary crispness, style consistency, and practical improvement opportunities without implementing fixes.

## Execution Shape

- Shape: full orchestrated, read-only review.
- Rationale: the request is explicitly subagent-backed and spans architecture, Go maintainability, API/HTTP, data, QA, and onboarding/documentation surfaces.
- Current phase: review-phase-1.
- Current phase status: complete.
- Research mode: fan-out.
- Code changes: not expected.
- Implementation readiness: not applicable; no implementation is planned in this task.

## Artifact Status

- `workflow-plan.md`: complete in this task path.
- `workflow-plans/review-phase-1.md`: complete in this task path.
- `spec.md`: not expected for this read-only review.
- `design/`: not expected for this read-only review.
- `plan.md`: not expected for this read-only review.
- `tasks.md`: not expected for this read-only review.
- `test-plan.md`: not expected.
- `rollout.md`: not expected.
- Post-code phase workflow files: not expected beyond this review-phase control file.

## Planned Read-Only Lanes

| Lane | Agent | Skill | Owned question |
| --- | --- | --- | --- |
| Adequacy challenge | challenger-agent | workflow-plan-adequacy-challenge | Check whether this master plan and the active review-phase plan are sufficient before review fan-out. |
| Architecture/design | architecture-agent | go-design-review | Are package boundaries, ownership, and extension seams clear enough for future business features? |
| Go maintainability | quality-agent | go-language-simplifier-review | Are helpers, naming, cohesion, and abstraction choices likely to drift or duplicate as features grow? |
| API/HTTP | api-agent | go-chi-review | Is the OpenAPI/chi/generated-handler path clear for new endpoints, middleware, fallback policy, and handler placement? |
| Data | data-agent | go-db-cache-review | Is the Postgres/sqlc/migration/repository extension path clear and well-bounded for future persistence code? |
| QA | qa-agent | go-qa-review | Are test placement, validation expectations, and examples sufficient for future business-feature work? |
| Docs/onboarding | explorer | no-skill | Do README/docs explain where future production code belongs, and are any extension recipes missing or inconsistent? |

## Gate Status

- Workflow plan adequacy challenge: complete.
- Blocking findings: reconciled.
- Resolution: clarified that the adequacy challenge runs before review fan-out; only after reconciliation may the Architecture/design, Go maintainability, API/HTTP, Data, QA, and Docs/onboarding lanes run in parallel.
- Accepted assumptions:
  - The review should not restore or edit previously deleted `specs/template-readiness-*` paths in the dirty worktree.
  - This task path is new to avoid overwriting unrelated user changes.
  - The final review report will be delivered in chat, not persisted as a new design/spec artifact.

## Session Boundary

- Session boundary reached: yes.
- Ready for next session: yes, if the user asks for an implementation follow-up.
- Next session starts with: implementation planning for selected review recommendations, if requested.
- Stop rule: do not implement fixes or create non-review planning/design artifacts in this pass.
- Closeout note: read-only review lanes returned, local evidence was gathered, no tests were run, and the final review report is prepared for chat delivery.
