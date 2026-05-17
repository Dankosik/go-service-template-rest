# Workflow Plan

Task: workflow simplification research, specification, planning, implementation, and closeout
Execution shape: full orchestrated analysis with local approval review and planning handoff
Execution rationale: the change affects repository-wide workflow rules, skills, docs, and agent handoff behavior; it needed external comparison and repo-grounded synthesis before implementation, then a bounded executable planning handoff after approval.

Current phase: validation / closeout
Phase status: complete
Session boundary reached: yes
Ready for next session: no
Next session starts with: N/A; workflow simplification implementation is done
Next session context bundle:
- No next session is expected for this task.
- `spec.md`: approved decisions plus implementation validation and outcome.
- `tasks.md`: all implementation tasks checked complete.
- `workflow-plans/planning.md`: historical planning handoff and accepted concerns.
- Research notes remain available for audit context if the workflow is reopened later.

Default resume order:
1. `workflow-plan.md`
2. `workflow-plans/planning.md`
3. `spec.md`
4. `tasks.md`
5. research notes named in the context bundle when source detail is needed

## Artifact Status

| Artifact | Status | Rationale |
| --- | --- | --- |
| `workflow-plan.md` | complete | Cross-phase state closed after implementation validation. |
| `workflow-plans/research.md` | complete | Records local research tracks, sources, fan-in, and limits. |
| `workflow-plans/specification.md` | complete | Records spec proposal status and original approval boundary. |
| `workflow-plans/planning.md` | complete | Records approval choices, design-skip rationale, accepted concerns, and planning stop rule. |
| `research/external-agent-workflow-practices.md` | complete | Preserves current external evidence and interpretation boundaries. |
| `research/current-workflow-pain-map.md` | complete | Preserves repo-grounded pain map and quality-carrier map. |
| `spec.md` | approved / validated | Trigger-based direction approved; validation and outcome updated with fresh evidence. |
| `design/` | waived / merged into `spec.md` | Planning records a design-skip rationale because the approved change is docs/skill contract work and `spec.md` contains the implementation-fit answers. |
| `tasks.md` | complete | All T001-T008 checkboxes are complete with closeout evidence. |
| `test-plan.md` | not expected | Implementation proof obligations fit in `tasks.md`; no separate validation strategy is needed for docs/skills-only work. |
| `rollout.md` | not expected | No staged rollout or compatibility window is required for docs/skills-only implementation; historical bundles remain valid by docs contract. |

## Gates

- Workflow-plan adequacy challenge: not run as a formal subagent gate; local self-check and implementation consistency sweeps were used for this docs/skills-only task.
- Spec clarification challenge: not run as a formal subagent gate; local approval review resolved the open questions in `spec.md`, including audit-driven `lean local`, inline `Risk Challenge`, and delta-spec additions.
- Implementation readiness: `CONCERNS` accepted and satisfied by targeted consistency sweeps plus required validation commands.
- Validation: `rtk make agents-check`, `rtk make skills-check`, and `rtk git diff --check` passed on 2026-05-17.

## Blockers

None. Workflow is complete.

## Accepted Assumptions

- The target is lower real-world workflow overhead, not removal of core quality safeguards.
- Compatibility with existing task-local bundles matters, so implementation should prefer evolutionary changes over replacing every artifact name at once.
- External workflow systems are useful as design evidence, but this repository's Go-service template constraints remain the controlling context.
- Docs and skills can be updated together without touching runtime code or generated artifacts.

## Reopen Targets

- Reopen specification if the approved trigger matrix, `lean local` artifact model, or inline `Risk Challenge` model is later rejected or materially changed.
- Reopen planning if a future workflow-related skill surface is discovered that was not covered by this implementation and cannot fit the current task grouping.
- Open a new task for any future `workflow-next` / `go-next` helper; it was explicitly out of scope for this implementation.
