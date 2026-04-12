# Agent Contract Refresh Implementation Phase 2

Phase: implementation-phase-2
Status: completed

## Scope

Resolve the `observability-agent` Claude mirror and README inventory drift.

Task IDs: T010-T013.

## Consumes

- `spec.md`
- `design/`
- `plan.md`
- `tasks.md`
- `workflow-plan.md`

## Allowed Future Writes

- `.claude/agents/observability-agent.md`
- `README.md`
- `.codex/agents/observability-agent.toml` only for a small parity fix if implementation proves it is needed
- existing task-local control/progress artifacts

## Entry Condition

Phase 1 is complete.

## Stop Rule

Do not start Phase 3 in this session. Reopen specification if `observability-agent` is intentionally Codex-only.

## Completion Marker

T010-T013 are complete, focused proof passes, and the next session can start `implementation-phase-3`.

Completion status: satisfied.

## Evidence

- `rtk zsh -lc 'test -f .claude/agents/observability-agent.md'`: passed.
- `rtk rg -n "go-observability-engineer-spec|not a default review|read-only" .claude/agents/observability-agent.md`: passed.
- `rtk rg -n "observability-agent|\\.claude/agents/observability-agent\\.md" README.md`: passed.
- `rtk zsh -lc 'diff -u <(find .codex/agents -maxdepth 1 -name "*.toml" -exec basename {} .toml \; | sort) <(find .claude/agents -maxdepth 1 -name "*.md" -exec basename {} .md \; | sort)'`: passed with no inventory diff.
- `rtk git diff --check`: passed.

## Handoff

Session boundary reached: yes.

Ready for next session: yes.

Next session starts with: `implementation-phase-3`.
