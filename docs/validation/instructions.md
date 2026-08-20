# Instruction Validation

Use for instruction, role, skill, mirror, or template-propagation changes.

| Surface | Command |
| --- | --- |
| Canonical role carrier parity | `make agent-roles-check` |
| Codex role registry | `make codex-agents-check` |
| Claude skill discovery | `make claude-skills-check` |
| Template ownership and sync behavior | `make template-owned-purity-check` |
| Structural workflow behavior | `bash scripts/ci/instruction-evals-check.sh` |

Run `make claude-skills-sync` after adding or removing canonical skills, then
check the generated view. Structural checks prove shape and ownership, not
changed model behavior; a model claim requires the authorized live eval runbook.
