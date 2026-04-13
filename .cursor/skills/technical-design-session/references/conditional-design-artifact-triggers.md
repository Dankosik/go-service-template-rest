# Conditional Design Artifact Triggers

## Behavior Change Thesis
When loaded for uncertain conditional artifacts, this file makes the model create only artifacts with real data, dependency, contract, validation, or rollout pressure instead of creating all optional files for completeness or skipping planning-critical context as "implementation detail."

## When To Load
Load when deciding whether to add `design/data-model.md`, `design/dependency-graph.md`, `design/contracts/`, `test-plan.md`, or `rollout.md`.

## Decision Rubric
- Trigger `design/data-model.md` for persisted state, schema, migration shape, cache contract, projections, replay behavior, data retention, or correctness-sensitive backfill.
- Trigger `design/dependency-graph.md` for package/module direction changes, generated-code dependency flow, new adapter boundaries, circular-coupling risk, or source-of-truth ambiguity across packages.
- Trigger `design/contracts/` for changed REST resources, event payloads, generated contracts, or material internal interfaces that planning must preserve; runtime authorities remain canonical.
- Trigger `test-plan.md` only when validation obligations are too large or multi-layered for the later `tasks.md`, such as contract plus migration plus reliability fail-path plus e2e smoke proof.
- Trigger `rollout.md` for mixed-version compatibility, expand/backfill/verify/contract sequencing, operational failback, or deploy ordering that affects correctness.
- If no trigger is real, record the artifact as `not expected`; do not create a placeholder file.
- If a triggered artifact needs a missing spec decision, block or reopen rather than drafting filler.
- A triggered `test-plan.md` should stay proof-focused: trigger/scope, proof obligations by changed surface or failure path, planned commands or manual proof shape, exit criteria, and reopen target for missing or failing proof.
- A triggered `rollout.md` should stay choreography-focused: trigger/scope, rollout sequence, safety checks, operator-visible state, rollback or forward-recovery conditions, and links to task IDs for execution detail.

## Imitate
```markdown
Triggered: `design/data-model.md`.
Reason: the approved change adds persisted export-job state, terminal statuses, retry visibility, and a migration path.
Planning impact: tasks must preserve state transition and migration ordering.
```

Copy this shape: it names the trigger and why planning needs the artifact.

```markdown
Triggered: `design/contracts/`.
Reason: OpenAPI request and response shapes change.
Authority note: `design/contracts/` is design-only; `api/openapi/service.yaml` remains canonical.
```

Copy this shape: it prevents design contracts from becoming runtime authority.

```markdown
Not expected: `design/dependency-graph.md`.
Reason: package dependency direction remains unchanged, and the component map introduces no new coupling risk.
```

Copy this shape: it documents a negative decision without creating filler.

```markdown
Triggered: `test-plan.md`.
Reason: validation spans OpenAPI drift, migration compatibility, retry fail-path behavior, and an e2e smoke check; putting every proof branch in `tasks.md` would make the ledger noisy.
Minimum content: scope, proof obligations by surface, planned commands, exit criteria, and reopen target for failing proof.
```

Copy this shape: it creates a proof artifact only because the validation surface is too layered for the ledger.

```markdown
Triggered: `rollout.md`.
Reason: the migration needs expand/backfill/verify sequencing with mixed-version compatibility and explicit failback notes.
Minimum content: rollout sequence, safety checks, rollback or forward-recovery conditions, and task-ID links.
```

Copy this shape: rollout context is choreography, not a second task ledger.

## Reject
```markdown
Create all conditional artifacts so planning has everything available.
```

Failure: optional artifacts become placeholders instead of triggered design context.

```markdown
Skip `design/data-model.md`; the migration can be figured out during coding.
```

Failure: migration ordering affects correctness and planning.

```markdown
Create `test-plan.md` with unit, integration, and e2e headings for completeness.
```

Failure: generic headings do not prove a validation obligation that cannot fit in `tasks.md`.

## Agent Traps
- Treating any API change as needing a large contract design when a small canonical-source note is enough.
- Forgetting `rollout.md` when mixed-version behavior or backfill order changes correctness.
- Treating cache behavior as "just implementation" when staleness, invalidation, or fallback semantics drive correctness.
- Creating conditional artifacts because another task had them, not because this spec triggers them.
