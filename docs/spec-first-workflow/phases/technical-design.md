# Technical Design Phase

Detailed phase companion for `docs/spec-first-workflow.md`. Read this when authoring compact or split technical design and recording `Design fan-out`.

## Read When

- Specification review has passed and separate technical design depth is triggered.
- Compact design in `spec.md` is too dense, contested, or insufficient for planning.
- Design-specialist fan-out, Pattern Fit, source-of-truth ownership, sequence, rollout, or validation decisions are planning-critical.

## Inputs

- Specification-review-approved `spec.md` and review obligations.
- `docs/repo-architecture.md` when stable boundaries, ownership, dependency direction, or runtime flows matter.
- Current `workflow-plan.md` or `workflow-plans/technical-design.md` when phase-local routing exists.

## Outputs

- Compact `design/overview.md` or split design bundle with triggered conditional artifacts.
- Recorded `Design fan-out: complete | scoped_down | local_only | blocked` decision and fan-in result.
- Next route to technical design review, or a reopen target.

## Stop Rule

Stop after the design bundle is review-ready or blocked. Do not draft `tasks.md`, approve implementation readiness, or start implementation in this phase.

## Design Depth

Design is content-triggered.

Lean local may keep design answers in `spec.md` `Compact Design` when affected surfaces, ownership, and sequence/failure behavior are concise and uncontested.

Use one `design/overview.md` when lean design context needs more room but still fits one artifact.

Split into design artifacts when the task needs durable, planning-critical context by dimension:

- `design/component-map.md`: affected packages, modules, generated surfaces, adapters, responsibility changes, stable surfaces, and intentional non-touches.
- `design/sequence.md`: runtime order, sync/async boundaries, side effects, failure points, retry/recovery behavior, and parallel versus sequential behavior.
- `design/ownership-map.md`: source-of-truth ownership, allowed dependency direction, generated-code authority, adapter responsibility, and explicit non-owners.

Conditional design artifacts:

- `design/data-model.md`: persisted state, schema, cache contract, projections, replay behavior, retention, or migration shape.
- `design/dependency-graph.md`: module/package dependency shape, generated-code dependency flow, coupling risk, or source-of-truth ambiguity.
- `design/pattern-fit.md`: selected and rejected design or system-design patterns, source descriptions, real-use examples, task applicability, Go-fit, repository-fit, and custom-design justification when the comparison is too dense for `design/overview.md`.
- `design/contracts/`: API/event/generated/material internal interface design context. Runtime authorities like `api/openapi/service.yaml` still win.
- `test-plan.md`: validation obligations are too large or layered for `tasks.md`.
- `rollout.md`: migration sequencing, compatibility window, deploy order, rollback, or failback matters.

If a design trigger is real but the required decision is missing, reopen specification or technical design instead of burying it in `tasks.md`.

### Technical Design Authoring Fan-Out

Separate technical design is not eligible for a private integrated design pass by default. Before writing or marking `design/` review-ready, identify the planning-critical design frontier and record the design-specialist fan-out decision in `workflow-plans/technical-design.md`, `workflow-plan.md`, or the lean-local artifact that owns the design checkpoint.

Use read-only specialist lanes for every unresolved live fork or domain-owned design decision that could change ownership, interfaces, persistence, async or sync semantics, failure behavior, observability, rollout, validation, package boundaries, dependency choice, or Pattern Fit outcome. Typical lane families are architecture/integration, API or contracts, data/source-of-truth, security, reliability/lifecycle, observability, delivery/rollout, performance, QA/proof, dependency/OSS, and Pattern Fit. Each lane owns one concrete question and returns advisory evidence for orchestrator fan-in.

Record the authoring gate in this shape:

```text
Design fan-out: complete | scoped_down | local_only | blocked
Candidate seams: <planning-critical seams considered>
Lane table: <lane id, lens/domain, owned question, skill/no-skill, inspect-first target, read-only enforcement, status>
Collapsed seams: <duplicate or consequence-only seams folded into the integrated design pass>
Escalation seams: <seams that require another lane, specification reopen, research, or user decision>
Fan-in outcome: <orchestrator reconciliation that changes or confirms the design bundle>
Review-ready consequence: <ready for technical design review | blocked | reopen specification/research>
```

`local_only` is valid only when the record lists candidate lanes or lenses considered, the evidence checked for each, why each omitted lane cannot change design correctness or planning readiness, and the seam that would reopen fan-out. Generic "single domain", "bounded", or "obvious" wording is not enough. Missing `Design fan-out` status, skipped candidate-lane analysis, unresolved lane blockers, or material severity conflicts block review-ready handoff and technical design review.

For full-orchestrated, protected-domain, high-impact, or user-requested agent-backed technical design, `local_only` is not an eligible authoring result. A scoped-down gate must still run at least one read-only specialist lane unless read-only execution is unavailable; unavailable read-only execution records `Design fan-out: blocked` and routes to the smallest unblock path.
