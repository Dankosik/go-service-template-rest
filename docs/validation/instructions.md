# Instruction Validation

Use for instruction, role, skill, mirror, or template-propagation changes.

| Surface | Command |
| --- | --- |
| Canonical role carrier parity | `bash scripts/agent-roles-sync.sh --check --repo .` |
| Codex project runtime and role registry | `bash scripts/codex-agents-sync.sh --check --repo .` |
| Claude skill discovery | `bash scripts/harness-skills-sync.sh claude --check --repo .` |
| Qwen skill discovery | `bash scripts/harness-skills-sync.sh qwen --check --repo .` |
| Template-source ownership and sync behavior | `make template-owned-purity-check` in the template checkout only |
| Structural workflow and hard-skill behavior catalogs, schema ownership, selectors, and links | `ALLOW_HEAVY=1 make instruction-evals-check` in the template checkout only |

Run one matching carrier leaf while iterating. The template-only purity target
already runs all four carrier checks; it is the final aggregate, not an
additional command beside those leaves.

After adding or removing canonical skills, run `bash
scripts/harness-skills-sync.sh claude --apply --repo .` and `bash
scripts/harness-skills-sync.sh qwen --apply --repo .`, then check both generated views.
Cursor, Grok, and OpenCode discover `.agents/skills` directly and have no generated skill
view.
A derived repository verifies adoption with the source template's
`scripts/template-sync.sh --check`; it does not run the template-only purity
gate, which intentionally rejects consumer-local `.service-owned` skills.
Structural checks prove shape and ownership, not changed model behavior; a
model claim requires the authorized live eval runbook.
