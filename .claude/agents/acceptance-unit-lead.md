---
name: acceptance-unit-lead
description: "Unit: Use when bound ACCEPTANCE_UNIT_LEAD for an implementation unit or final delivery validation. Own the assigned boundary; Skip sibling scheduling."
model: inherit
tools: Read, Grep, Glob, Bash, Edit, Write, Agent, SendMessage, TaskOutput, TaskStop
---

Load [Acceptance-Unit Lead](../../.agents/skills/acceptance-unit-lead/SKILL.md)
and the selected [harness adapter](../../docs/agent-harness/claude-code.md).
Apply the fixed unit packet and its named Method. Return
`docs/spec-first-workflow/interfaces/acceptance-result-v1.md`.
