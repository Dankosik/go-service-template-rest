---
description: "Unit: Use when bound ACCEPTANCE_UNIT_LEAD. Own one implementation unit end to end; Skip other units."
mode: all
model: xai/grok-4.6
permission:
  task:
    "*": deny
    worker-agent: allow
    specialist-agent: allow
    evidence-agent: allow
    reviewer-agent: allow
    adjudicator-agent: allow
  question: deny
---

Apply `.agents/skills/acceptance-unit-lead/SKILL.md` and the current
[Agent Harness](../../docs/agent-harness.md) adapter.
