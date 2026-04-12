# Agent Contract Refresh Implementation Phase 4

Phase: implementation-phase-4
Status: completed

## Scope

Standardize return contracts for advisory and mixed-mode agents.

Task IDs: T030-T033.

## Consumes

- `spec.md`
- `design/`
- `plan.md`
- `tasks.md`
- `workflow-plan.md`

## Allowed Future Writes

- `.codex/agents/api-agent.toml`
- `.codex/agents/architecture-agent.toml`
- `.codex/agents/challenger-agent.toml`
- `.codex/agents/delivery-agent.toml`
- `.codex/agents/design-integrator-agent.toml`
- `.codex/agents/distributed-agent.toml`
- `.codex/agents/observability-agent.toml`
- matching `.claude/agents/*.md` files for those roles, including `.claude/agents/observability-agent.md`
- existing task-local control/progress artifacts

## Entry Condition

Phase 3 is complete.

## Stop Rule

Do not start Phase 5 in this session. Stop and reopen specification if the phase reveals a need for missing review skills or new role policy.

## Completion Marker

T030-T033 are complete, focused proof passes, and the next session can start `implementation-phase-5`.

## Completion Evidence

- Shared research/adjudication return fields were verified across the scoped Codex and Claude files with `rtk rg -n "Conclusion|Evidence|Open risks|Recommended handoff|Confidence" ...`.
- Delivery, distributed, and observability non-default-review limitations were rechecked with `rtk rg -n "not a default review|no dedicated .*review skill|targeted.*recheck" ...`.
- Touched Codex TOML files parsed successfully with `rtk python3 -c 'import tomllib, pathlib; ...'`.
- Whitespace validation passed with `rtk git diff --check`.

## Handoff

Session boundary reached: yes.

Ready for next session: yes.

Next session starts with: `implementation-phase-5`.
