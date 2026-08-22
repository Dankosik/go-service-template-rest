# Instruction Validation

Use for instruction, role, skill, mirror, or template-propagation changes.

| Surface | Command |
| --- | --- |
| Canonical role carrier parity | `bash scripts/agent-roles-sync.sh --check --repo .` |
| Codex project runtime and role registry | `bash scripts/codex-agents-sync.sh --check --repo .` |
| Claude skill discovery | `bash scripts/ci/claude-skills-check.sh` |
| Qwen skill discovery | `bash scripts/ci/qwen-skills-check.sh` |
| Template-source ownership and sync behavior | `make template-owned-purity-check` in the template checkout only |
| Structural workflow and hard-skill behavior catalogs, schema ownership, selectors, and links | `bash scripts/ci/instruction-evals-check.sh` |

After adding or removing canonical skills, run `bash
scripts/claude-skills-sync.sh --apply --repo .` and `bash
scripts/qwen-skills-sync.sh --apply --repo .`, then check both generated views.
Cursor, Grok, and OpenCode discover `.agents/skills` directly and have no generated skill
view.
A derived repository verifies adoption with the source template's
`scripts/template-sync.sh --check`; it does not run the template-only purity
gate, which intentionally rejects consumer-local `.service-owned` skills.
Structural checks prove shape and ownership, not changed model behavior; a
model claim requires the authorized live eval runbook.
