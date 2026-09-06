---
name: acceptance-unit-lead
description: "Unit: Use when bound ACCEPTANCE_UNIT_LEAD for an implementation unit or final delivery validation. Own the assigned boundary; Skip sibling scheduling."
model: inherit
tools:
  - read_file
  - grep_search
  - glob
  - list_directory
  - run_shell_command
  - write_file
  - edit
  - agent
  - list_agents
  - send_message
  - task_stop
---

Load [Acceptance-Unit Lead](../../.agents/skills/acceptance-unit-lead/SKILL.md)
and the selected [harness adapter](../../docs/agent-harness/qwen-code.md).
Apply the fixed unit packet and its named Method. Return
`docs/spec-first-workflow/interfaces/acceptance-result-v1.md`.
