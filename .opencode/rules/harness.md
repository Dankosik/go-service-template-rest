# OpenCode

OpenCode sessions follow the repository contract. When a native control is
needed, load [Agent Harness](../../docs/agent-harness.md) and only its
[OpenCode adapter](../../docs/agent-harness/opencode.md). Sibling bootstrap
files select other harnesses; do not follow their adapter choice.

A user request to orchestrate a persisted Implementation ledger is enough.
This session routes: spawn each ready unit with Task `subagent_type`
`acceptance-unit-lead`. Do not implement unit work here. Do not ask the user
to Tab, `/agent`, or restart. Do not wait for the footer to say orchestrator.
Pass that name even if the Task blurb lists only `explore` or `general`. A
carrier gap is a failed Task call, a missing session id, or `subagent_depth`
of 1.
