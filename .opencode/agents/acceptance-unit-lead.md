---
description: "Unit: Use when bound ACCEPTANCE_UNIT_LEAD for an implementation unit or final delivery validation. Own the assigned boundary; Skip sibling scheduling."
mode: all
model: xai/grok-4.6
variant: xhigh
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
