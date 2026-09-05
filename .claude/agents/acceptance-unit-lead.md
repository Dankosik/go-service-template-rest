---
name: acceptance-unit-lead
description: "Own one ledger-assigned acceptance unit through implementation, proof, required review, and its acceptance verdict."
model: inherit
tools: Read, Grep, Glob, Bash, Edit, Write, Agent, SendMessage, TaskOutput, TaskStop
---

Load [Acceptance-Unit Lead](../../.agents/skills/acceptance-unit-lead/SKILL.md)
and the selected [harness adapter](../../docs/agent-harness/claude-code.md).
Apply the fixed unit packet and its named Method. Return
`docs/spec-first-workflow/interfaces/acceptance-result-v1.md`.
