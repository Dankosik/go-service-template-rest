# Decision Placement And Artifact Ownership

## Behavior Change Thesis
When loaded for a spec draft that mixes decisions with evidence, design detail, implementation tasks, or transcripts, this file makes the model move each fact to its owning artifact instead of using `spec.md` as an all-purpose dump.

## When To Load
Load this when a draft includes research notes, benchmark output, file topology, component maps, implementation steps, review transcripts, or task IDs inside `spec.md`.

## Decision Rubric

| Content | Home |
|---|---|
| final behavior, scope, accepted constraints, rejected alternatives | `spec.md` |
| raw comparisons, source links, benchmark output, transcript evidence | `research/*.md` |
| component map, runtime sequence, ownership, source-of-truth design | `design/` |
| execution order, phase breakdown, task IDs, coder instructions | `tasks.md`, plus optional `plan.md` only when supplemental strategy is justified |
| unresolved assumptions, blockers, or accepted risks | `spec.md` `Open Questions / Assumptions` |

- Keep only the selected outcome in `Decisions`; link or summarize preserved evidence elsewhere.
- If a fact is useful but not final, do not promote it into `Decisions`.
- If a design detail matters, route it to `design/` and leave the decision-level boundary in `spec.md`.
- If a task list exists, keep only the planning summary or plan link in `spec.md`.

## Imitate

```markdown
## Decisions
- The import endpoint remains synchronous for files below the existing request-size limit.
- Duplicate external IDs are rejected with the existing validation-error response shape.
- Audit events are emitted only after the import transaction commits.
- Reject asynchronous import for this change because the current API contract is synchronous and no queue ownership exists in the approved scope.

## Validation
- API tests cover duplicate external IDs and transaction rollback.
- Audit-event tests prove no event is emitted for rejected imports.
```

Copy this: the spec fixes acceptance semantics and a rejected path without describing SQL statements, handler call order, or task IDs.

## Reject

```markdown
## Decisions
- Researcher A said option 2 looked safer.
- The database package has files `store.go`, `tx.go`, and `queries.go`.
- Task 1: edit the handler. Task 2: edit repository tests.
- Use the following transaction sequence: begin, insert rows, insert audit, commit.
```

Failure: evidence belongs in `research/`, topology and sequence belong in design when material, and task order belongs in planning.

## Agent Traps
- Keeping raw research in `Decisions` because it influenced the choice.
- Turning "this package exists" into a spec decision.
- Preserving task IDs copied from a prompt.
- Using `spec.md` to avoid creating a required downstream artifact.
- Loading `spec-handoff-to-technical-design.md` for the same symptom by default; use that narrower reference only when the immediate question is handoff readiness or design leakage at approval time.
