# Planning Anti-Patterns

## Behavior Change Thesis
When loaded for draft review or smell triage, this file makes the model challenge invented decisions, duplicate authority, vague proof, and artifact-boundary drift instead of treating a plausible-looking plan as ready by checklist momentum.

## When To Load
Load this when reviewing a draft `tasks.md` or optional `plan.md` for invented decisions, duplicate authority, phase-boundary drift, vague verification, false parallelism, or artifact misuse.

Use this as a challenge catalog, not primary planning guidance. Prefer a narrower positive reference when the symptom is specifically phase strategy, dependency order, slicing, proof, or reopen conditions.

## Decision Rubric
- Invented domain work: delete it or reopen the earlier phase; do not add security, data, reliability, rollout, or API phases "for completeness."
- Duplicate authority: move executable detail to `tasks.md`; keep optional `plan.md` limited to supplemental strategy and keep `spec.md` decisions plus `design/` technical context as the sources of truth.
- Artifact misuse: `tasks.md` is a ledger, not a second spec, second design bundle, or raw research dump.
- Vague proof: replace "check everything" with surface-specific commands or manual reads.
- False readiness: use `FAIL` or `CONCERNS` when blockers remain; do not bury them under "implementation can start."

## Imitate
```markdown
## Blockers / Assumptions

- No rollout artifact is expected for this task because the approved scope changes only the skill reference bundle and no deployment, migration, compatibility, or runtime behavior surface is triggered.
- `tasks.md` is expected because this is non-trivial planning output with several reference files; it remains the final handoff ledger and does not restate `spec.md` decisions.
- If an example requires a new architecture, API, data, security, reliability, or rollout decision, reopen the appropriate earlier phase instead of adding it here.
```

Copy the explicit "not expected" reasoning and the refusal to invent conditional artifacts.

## Reject
```markdown
## Extra Safety

Add security, reliability, database, API, and rollout phases just in case. If there is no design for them, make reasonable choices during implementation.
```

This fails because it creates just-in-case work and bypasses the decision/design chain.

## Agent Traps
- Do not flag a compact `SKILL.md` as incomplete when detailed examples live in `references/`.
- Do not flag omitted architecture/API/data/security/reliability/rollout examples when no approved decision in that domain exists.
- Do not flag direct-path or lightweight-local waivers when they are explicit, narrow, and recorded before crossing a phase boundary.
- Do not mistake "not expected" for "forgotten"; conditional artifacts should be absent unless triggered.
