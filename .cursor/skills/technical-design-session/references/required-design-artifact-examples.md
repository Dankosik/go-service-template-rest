# Required Design Artifact Examples

## Behavior Change Thesis
When loaded for shaping the core design bundle, this file makes the model split task-local technical context into the four required design artifacts instead of writing one design dump, hiding design content in workflow files, or leaving ownership and sequence for planning to guess.

## When To Load
Load when creating or repairing `design/overview.md`, `design/component-map.md`, `design/sequence.md`, or `design/ownership-map.md`.

## Decision Rubric
- `design/overview.md` is the entrypoint: chosen approach, artifact index, unresolved seams, readiness summary, and links to triggered conditional artifacts.
- `design/component-map.md` owns affected packages, modules, generated surfaces, adapters, and stable areas; it is not an implementation task list.
- `design/sequence.md` owns runtime order: request, async, startup, shutdown, recovery, failure points, side effects, and sync/async boundaries.
- `design/ownership-map.md` owns source-of-truth, dependency direction, generated-code authority, adapter responsibility, and what must not own the behavior.
- Keep final product decisions in `spec.md`; keep execution sequencing in the later `plan.md`; keep workflow status in workflow-control files.
- If a required artifact cannot be completed without a missing spec decision, block or reopen instead of writing "decide during implementation."

## Imitate
```markdown
`design/overview.md`
- Approach: add the async export capability behind app-owned job orchestration and infra-owned HTTP/download adapters.
- Bundle index: component map, sequence, ownership map, data model, contracts.
- Readiness: planning can start after `design/contracts/` is approved.
```

Copy this shape: overview points to the bundle and keeps readiness honest.

```markdown
`design/sequence.md`
- Handler validates request and delegates to app orchestration.
- App creates a pending job record before enqueueing work.
- Worker completes generation, records terminal state, and emits the outbox event.
- Failure points: enqueue failure before acceptance, worker retry, download URL generation, event publish.
```

Copy this shape: sequence includes side effects and failure points, not only a happy path.

```markdown
`design/ownership-map.md`
- Job state source of truth: Postgres via infra repository.
- Business transition rules: app orchestration.
- HTTP response mapping: infra/http.
- Generated contract authority: `api/openapi/service.yaml`; generated output is not hand-edited.
```

Copy this shape: ownership names who owns each decision and who does not.

## Reject
```markdown
`design/sequence.md`: handler calls service and saves the job.
```

Failure: it omits side effects, failure points, and sync/async finality.

```markdown
`workflow-plans/technical-design.md`: component map follows...
```

Failure: workflow control becomes the design bundle.

```markdown
`design/ownership-map.md`: implementation should choose whether cache or DB owns invoice status.
```

Failure: planning-critical source-of-truth ownership is deferred to coding.

## Agent Traps
- Creating a polished `design/overview.md` while the required map or sequence files stay empty.
- Repeating `spec.md` decisions without adding repository fit, runtime order, or ownership.
- Turning component map bullets into T001/T002 execution steps.
- Marking the bundle planning-ready while required artifacts disagree with each other.
