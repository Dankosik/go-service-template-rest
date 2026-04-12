# Agent Contract Refresh Implementation Phase 5

Phase: implementation-phase-5
Status: completed

## Scope

Add `Inspect first` blocks for runtime and domain roles.

Task IDs: T040-T043.

## Consumes

- `spec.md`
- `design/`
- `plan.md`
- `tasks.md`
- `workflow-plan.md`
- `docs/repo-architecture.md`

## Allowed Future Writes

- Codex and Claude runtime files for `api-agent`, `concurrency-agent`, `data-agent`, `domain-agent`, `observability-agent`, `performance-agent`, `reliability-agent`, and `security-agent`
- existing task-local control/progress artifacts

## Entry Condition

Phase 4 is complete.

## Stop Rule

Do not start Phase 6 in this session. Stop and reopen technical design if a role's inspect-first list depends on a missing repository ownership decision.

## Completion Marker

T040-T043 are complete, focused proof passes, and the next session can start `implementation-phase-6`.

## Completion Evidence

- Codex `Inspect first` check passed: `rtk rg -n "Inspect first" .codex/agents/api-agent.toml .codex/agents/concurrency-agent.toml .codex/agents/data-agent.toml .codex/agents/domain-agent.toml .codex/agents/observability-agent.toml .codex/agents/performance-agent.toml .codex/agents/reliability-agent.toml .codex/agents/security-agent.toml`.
- Claude `Inspect first` check passed: `rtk rg -n "Inspect first" .claude/agents/api-agent.md .claude/agents/concurrency-agent.md .claude/agents/data-agent.md .claude/agents/domain-agent.md .claude/agents/observability-agent.md .claude/agents/performance-agent.md .claude/agents/reliability-agent.md .claude/agents/security-agent.md`.
- Path spot check passed against approved repository surfaces: `rtk zsh -lc 'for p in api/openapi/service.yaml internal/api internal/infra/http internal/app cmd/service/main.go cmd/service/internal/bootstrap internal/config internal/app/health internal/infra/postgres internal/infra/telemetry internal/observability/otelconfig env/migrations; do test -e "$p" || { echo missing:$p; exit 1; }; done; echo phase5-inspect-paths-present'`.
- Scoped Codex TOML parse check passed.
- Whitespace check passed: `rtk git diff --check`.

## Exit State

Session boundary reached: yes.
Ready for next session: yes.
Next session starts with: `implementation-phase-6`.
