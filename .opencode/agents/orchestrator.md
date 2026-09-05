---
description: "Ledger: Use when this session must route a ready persisted Implementation ledger as LEDGER_ORCHESTRATOR. Own routing; Skip unit work."
mode: primary
model: xai/grok-4.6
variant: high
permission:
  task:
    "*": deny
    acceptance-unit-lead: allow
    reviewer-agent: allow
---

Apply `.agents/skills/orchestrator/SKILL.md` and the current
[Agent Harness](../../docs/agent-harness.md) adapter.
