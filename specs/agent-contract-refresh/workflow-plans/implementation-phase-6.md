# Agent Contract Refresh Implementation Phase 6

Phase: implementation-phase-6
Status: completed

## Scope

Add `Inspect first` blocks for workflow and meta roles.

Task IDs: T050-T053.

## Consumes

- `spec.md`
- `design/`
- `plan.md`
- `tasks.md`
- `workflow-plan.md`
- `docs/spec-first-workflow.md`

## Allowed Future Writes

- Codex and Claude runtime files for `architecture-agent`, `challenger-agent`, `delivery-agent`, `design-integrator-agent`, `distributed-agent`, `qa-agent`, and `quality-agent`
- existing task-local control/progress artifacts

## Entry Condition

Phase 5 is complete.

## Stop Rule

Do not start Phase 7 in this session. Stop and reopen specification if the challenger inspect-first split exposes missing policy for a challenge gate.

## Completion Marker

T050-T053 are complete, focused proof passes, and the next session can start `implementation-phase-7`.

## Completion Evidence

- Codex `Inspect first` check passed: `rtk rg -n "Inspect first" .codex/agents/architecture-agent.toml .codex/agents/challenger-agent.toml .codex/agents/delivery-agent.toml .codex/agents/design-integrator-agent.toml .codex/agents/distributed-agent.toml .codex/agents/qa-agent.toml .codex/agents/quality-agent.toml`.
- Claude `Inspect first` check passed: `rtk rg -n "Inspect first" .claude/agents/architecture-agent.md .claude/agents/challenger-agent.md .claude/agents/delivery-agent.md .claude/agents/design-integrator-agent.md .claude/agents/distributed-agent.md .claude/agents/qa-agent.md .claude/agents/quality-agent.md`.
- Challenger three-mode inspect-first guidance was rechecked with `rtk rg -n "workflow-plan-adequacy-challenge|pre-spec-challenge|spec-clarification-challenge" .codex/agents/challenger-agent.toml .claude/agents/challenger-agent.md`.
- Path spot check passed against referenced existing repository surfaces: `rtk zsh -lc 'for p in docs/repo-architecture.md docs/ci-cd-production-ready.md docs/build-test-and-development-commands.md docs/project-structure-and-module-organization.md cmd/service/internal/bootstrap internal/app internal/infra build/ci scripts/ci Makefile build/docker env/docker-compose.yml railway.toml env/migrations api/openapi/service.yaml api/proto go.mod; do test -e "$p" || { print -r -- missing:$p; exit 1; }; done; print -r -- phase6-inspect-paths-present'`.
- Scoped Codex TOML parse check passed.
- Whitespace check passed: `rtk git diff --check`.

## Exit State

Session boundary reached: yes.
Ready for next session: yes.
Next session starts with: `implementation-phase-7`.
