# Agent Contract Refresh Implementation Phase 3

Phase: implementation-phase-3
Status: completed

## Scope

Standardize return contracts for review-focused agents.

Task IDs: T020-T023.

## Consumes

- `spec.md`
- `design/`
- `plan.md`
- `tasks.md`
- `workflow-plan.md`

## Allowed Future Writes

- `.codex/agents/concurrency-agent.toml`
- `.codex/agents/data-agent.toml`
- `.codex/agents/domain-agent.toml`
- `.codex/agents/performance-agent.toml`
- `.codex/agents/qa-agent.toml`
- `.codex/agents/quality-agent.toml`
- `.codex/agents/reliability-agent.toml`
- `.codex/agents/security-agent.toml`
- matching `.claude/agents/*.md` files for those roles
- existing task-local control/progress artifacts

## Entry Condition

Phase 2 is complete.

## Stop Rule

Do not start Phase 4 in this session. Stop and reopen technical design if a review-focused role cannot preserve its safety boundary with the shared return contract.

## Completion Marker

T020-T023 are complete, focused proof passes, and the next session can start `implementation-phase-4`.

Completion status: satisfied.

## Evidence

- `rtk rg -n "Findings by severity|Evidence:|Why it matters|Validation gap|Handoff:|Confidence:" .codex/agents/concurrency-agent.toml .codex/agents/data-agent.toml .codex/agents/domain-agent.toml .codex/agents/performance-agent.toml .codex/agents/qa-agent.toml .codex/agents/quality-agent.toml .codex/agents/reliability-agent.toml .codex/agents/security-agent.toml .claude/agents/concurrency-agent.md .claude/agents/data-agent.md .claude/agents/domain-agent.md .claude/agents/performance-agent.md .claude/agents/qa-agent.md .claude/agents/quality-agent.md .claude/agents/reliability-agent.md .claude/agents/security-agent.md`: passed.
- `rtk rg -n "Use at most one skill per pass|read-only|advisory" .codex/agents/concurrency-agent.toml .codex/agents/data-agent.toml .codex/agents/domain-agent.toml .codex/agents/performance-agent.toml .codex/agents/qa-agent.toml .codex/agents/quality-agent.toml .codex/agents/reliability-agent.toml .codex/agents/security-agent.toml .claude/agents/concurrency-agent.md .claude/agents/data-agent.md .claude/agents/domain-agent.md .claude/agents/performance-agent.md .claude/agents/qa-agent.md .claude/agents/quality-agent.md .claude/agents/reliability-agent.md .claude/agents/security-agent.md`: passed.
- `rtk python3 -c 'import sys,tomllib; [tomllib.load(open(path,"rb")) for path in sys.argv[1:]]; print("toml ok", len(sys.argv)-1)' .codex/agents/concurrency-agent.toml .codex/agents/data-agent.toml .codex/agents/domain-agent.toml .codex/agents/performance-agent.toml .codex/agents/qa-agent.toml .codex/agents/quality-agent.toml .codex/agents/reliability-agent.toml .codex/agents/security-agent.toml`: passed.
- `rtk git diff --check`: passed.

## Handoff

Session boundary reached: yes.

Ready for next session: yes.

Next session starts with: `implementation-phase-4`.
