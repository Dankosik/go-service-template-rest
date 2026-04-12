# Agent Contract Refresh Implementation Phase 1

Phase: implementation-phase-1
Status: completed

## Scope

Implement the challenger three-mode contract fix only.

Task IDs: T001-T004.

## Consumes

- `spec.md`
- `design/`
- `plan.md`
- `tasks.md`
- `workflow-plan.md`

## Allowed Future Writes

- `.codex/agents/challenger-agent.toml`
- `.claude/agents/challenger-agent.md`
- `README.md` only if challenger wording conflicts after the runtime fix
- existing task-local control/progress artifacts

## Entry Condition

Implementation readiness remains PASS from planning.

## Stop Rule

Do not start Phase 2 in this session. Stop and reopen specification or technical design if the three-mode challenger contract cannot be expressed equivalently in both runtime formats.

## Completion Marker

T001-T004 are complete, focused proof passes, and the next session can start `implementation-phase-2`.

Completion status: satisfied.

## Evidence

- `rtk python3 -c 'import pathlib, tomllib; tomllib.loads(pathlib.Path(".codex/agents/challenger-agent.toml").read_text()); print("toml ok")'`: passed.
- `rtk rg -n "workflow-plan-adequacy-challenge|pre-spec-challenge|spec-clarification-challenge" .codex/agents/challenger-agent.toml`: passed.
- `rtk rg -n "workflow-plan-adequacy-challenge|pre-spec-challenge|spec-clarification-challenge" .claude/agents/challenger-agent.md`: passed.
- `rtk sed -n '108,116p' README.md`: challenger row now includes workflow-plan adequacy, pre-spec challenge, and spec clarification challenge.
- `rtk git diff --check`: passed.

## Handoff

Session boundary reached: yes.

Ready for next session: yes.

Next session starts with: `implementation-phase-2`.
