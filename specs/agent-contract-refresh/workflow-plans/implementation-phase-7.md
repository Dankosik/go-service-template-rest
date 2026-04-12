# Agent Contract Refresh Implementation Phase 7

Phase: implementation-phase-7
Status: completed

## Scope

Run safe global-policy deduplication and the drift-policy checkpoint.

Task IDs: T060-T064.

## Consumes

- `spec.md`
- `design/`
- `plan.md`
- `tasks.md`
- `workflow-plan.md`

## Allowed Future Writes

- `.codex/agents/*.toml`
- `.claude/agents/*.md`
- `README.md` only if an existing section needs a short drift-policy note
- existing task-local control/progress artifacts

## Entry Condition

Phase 6 is complete.

## Stop Rule

Do not start validation in this session. Reopen specification if the user wants canonical-source generation, CI drift checks, new review skills, nickname additions, or model/reasoning overrides in this same task cycle.

## Completion Marker

T060-T064 are complete, focused proof passes, and the next session can start `validation-phase-1`.

Completion status: met.

Proof summary:

- Removed the repeated global `Never use` block from Codex and Claude agent runtime files while preserving role-local mission, boundaries, mode routing, skill policy, return contract, inspect-first guidance, and escalation rules.
- Rechecked guardrails with targeted `rtk rg` for read-only/advisory/one-skill-per-pass wording.
- Rechecked README `observability-agent` inventory and Codex/Claude runtime inventory parity.
- Confirmed `.agents/skills`, CI files, and `.codex/config.toml` had no status changes during the drift-policy checkpoint.
- Passed full Codex TOML parse check, final section checks, no-`Never use` dedupe check, and `rtk git diff --check`.
