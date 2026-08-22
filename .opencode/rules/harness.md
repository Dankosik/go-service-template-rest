# OpenCode

OpenCode sessions follow the repository contract. When a native control is
needed, load [Agent Harness](../../docs/agent-harness.md) and only its
[OpenCode adapter](../../docs/agent-harness/opencode.md). Sibling bootstrap
files select other harnesses; do not follow their adapter choice.

Task `subagent_type` is a free string. Pass `acceptance-unit-lead` even if the
Task blurb lists only `explore` or `general`. A carrier gap is a failed Task
call, a missing session id, or `subagent_depth` of 1. Ledger dispatch requires
the session footer to show orchestrator (`/orchestrator` or `/agent
orchestrator`).
