# Ownership Map

## Artifact Ownership To Preserve

- `AGENTS.md`: repository-wide authority and hard invariants.
- `docs/spec-first-workflow.md`: detailed runtime mechanics and artifact shapes.
- Session skills: phase-specific operating protocol and allowed writes.
- Skill reference files: compact behavior-change examples and rubrics.
- `spec.md`: task-local decisions for this pass.
- `design/`: task-local technical context for this pass.
- `tasks.md`: executable task ledger and proof expectations for this pass.

## Boundaries

- Do not make workflow-control files own final decisions.
- Do not make review phase files own raw transcripts or new task ledgers.
- Do not make `tasks.md` absorb `spec.md`, `design/`, or optional `plan.md`.
- Do not introduce new post-code process artifacts outside planning.
