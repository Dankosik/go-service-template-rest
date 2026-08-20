# Planning Ledger Contract

Read only when Planning must persist `tasks.md` because work has multiple
acceptance units, dependencies or waves, crosses an actor/session boundary, or
needs durable resume state.

## Ledger shape

```markdown
# Goal
status: draft | ready | blocked | done
Completion: <observable successful condition>
Blocked stop: <what remains incomplete, evidence to record, and owner to reopen>
Global constraints: <exact constraints shared by multiple tasks; omit when none>

- [ ] T1: <verifiable postcondition; execution-changing accepted constraints>
  - Source: <narrow stable spec/design/test/rollout/code anchor(s)>
  - Owner/surface/resources: <canonical owner for each writable surface; initial authorized writable paths or bounded discovery rule; mutable, exclusive, or non-concurrent resources, or none>
  - Depends on: <ID — output handoff, exact consumed state, or exact safety/proof gate; needed to start, complete, or prove; or none>
  - Handoff: <for an output dependency: exact produced output and consumed input/acceptance condition; omit when none>
  - Alias of: <task ID and exact accepted receipt consumed; use only when this entry has no implementation delta; omit otherwise>
  - External input/gate: <required non-ledger input or rollout gate; named owner; objective availability checkpoint; omit when none>
  - Proof: <claim; command/check; expected observable>
  - Reopen if: <concrete objective future invalidation condition; upstream owner; omit when none>
```

## Ledger layout

Keep every field inline in `tasks.md` by default. Move each task's execution
detail into one task file under `tasks/<ID>-<postcondition-slug>.md` when an
executing actor would otherwise read materially more ledger than its own unit
needs, or when a task's execution-critical content does not fit its entry. The
split moves detail only:

- `tasks.md` stays the index and sole owner of lifecycle state, checkboxes,
  receipts, completion and blocked conditions, global constraints, acceptance
  units, planned waves, and dependency edges.
- The task file owns the outcome body, its constraints, and remaining entry
  fields. It carries no lifecycle state.

A split index entry keeps the postcondition title and dependency only:

```markdown
- [ ] T1: <verifiable postcondition> — file: tasks/T1-<postcondition-slug>.md
  - Depends on: <ID — output handoff, exact consumed state, or exact safety/proof gate; needed to start, complete, or prove; or none>
```

Obligation reconciliation, the acceptance-unit map, and dependency graph stay
auditable from the index alone.

### Task file

Write a task file for one actor that receives only the [Worker dispatch
contract](../implementation-worker-contract.md#dispatch-contract) plus this
file.

```markdown
# T1 — <postcondition title>

<the observable behavior that becomes true on the real production path, stated
as the caller sees it, with execution-changing constraints and preserved or
forbidden behavior>

- Source: <narrow stable anchors>
- Owner/surface/resources: <canonical owner; authorized paths or discovery rule; mutable resources, or none>
- Handoff / Alias of / External input/gate: <when present>
- Proof: <claim; command/check; expected observable>
- Reopen if: <omit when none>
```

Inline a schema, interface, state transition, error body, or other shape only
when it fixes an accepted decision more precisely than prose. A task file adds
no restatement of the index outcome and no skill routing.

## Ledger entry

Add only fields that change execution. Put a constraint in `Global constraints`
only when its exact meaning applies across multiple tasks; keep task-specific
constraints in the task outcome. Write each task title as the postcondition that
becomes true. Put paths and symbols in `Owner/surface/resources` and commands in
`Proof`; neither creates a task boundary.

## Task boundary

A split boundary is valid only when the completed task leaves the repository
and every deployment or migration state it creates or assumes internally
consistent, supported by accepted compatibility or rollback policy,
independently reviewable, and provable without unfinished companion work. Group
canonical source, generated or mirrored output, required tests and fixtures,
migration/runtime compatibility, documentation, and replacement cleanup needed
for that state in the same task.

Prefer a boundary that makes one accepted behavior reachable end to end through
its real production entry point. A layer-only task is valid when accepted
rollout, migration, or `expand -> migrate -> contract` order fixes it, when the
layer is the whole outcome, or when it is an enabling change. Split an oversized
outcome only for a distinct owner, review/proof, failure/recovery, rollback,
required handoff, evidence-backed parallel wave, or independently shippable
accepted result. File count, estimated minutes, and desired Worker count do not
create a boundary.

## Acceptance units

An **acceptance unit** is the smallest fixed candidate that one Acceptance-Unit
Lead can deliver through implementation slices, prove, review when triggered,
and integrate without a consumer depending on an intermediate state. Every
implementation task is a singleton unit unless exactly one recorded
`Acceptance units` entry contains its task ID; overlapping membership is
invalid. Group adjacent ready tasks only when they share the canonical owner,
editable boundary, proof preconditions, and final-state validity:

```markdown
## Acceptance units
- A1: T2, T3 — <shared owner, boundary, and proof reason>
```

The unit is the Lead, final-proof, review, and integration boundary. Internal
Workers create no acceptance state. A receipt alias names `Alias of`, has no
writable surface or proof command, and closes mechanically when the named
accepted receipt exists.

## Implementation transitions

A leaf or reviewer `NEEDS_PARENT` return is a one-level message to its direct
parent, never ledger state. Only the Acceptance-Unit Lead may promote the issue
to a unit `Blocked:` record after [bottom-up
resolution](../implementation-obstacle-recovery.md#bottom-up-resolution) is
exhausted; the Ledger Orchestrator never transcribes a child return.

[Acceptance-Unit
Closure](../../shared/acceptance-unit-closure.md)
authorizes one transition. Record a receipt only when proof must survive a
checkout, session, or external-environment boundary:

```markdown
  - Accepted: <unit or task IDs>; evidence: <command or source and result>; candidate: <bounded diff or commit/tree>
```

Use `current bounded diff` for same-checkout proof and a commit/tree only across
a checkout or integration boundary. When implementation is complete but
required proof is unavailable, leave the unit unchecked and keep one line:

```markdown
  - Blocked: <unit or task IDs>; unverified: <claim>; evidence: <narrower evidence>; next proof owner: <owner and condition>; candidate: <bounded diff or commit/tree>
```

Replace that line with the accepted receipt after proof rather than accumulating
attempts. A blocked unit blocks its dependants. Keep ledger `status: ready` while
another unit or authorized recovery is executable; use `status: blocked` only
when neither remains.

The accepted transition checks every member task in one edit. Receipt aliases
close mechanically in that edit and create no candidate or proof. After the
final task and aliases, set `status: done` in the same edit. Add no second
lifecycle field.

## Dependencies and waves

For sequential work, `Depends on` is the complete ordering authority; do not
create one-task waves. Record an edge only when the downstream task consumes an
upstream output or state, or must cross its safety/proof gate, and name whether
the edge is required to start, complete, or prove the downstream task. Put an
output contract in `Handoff`; keep `Depends on` to the edge and need.

Add `Planned waves` only when at least two ready acceptance units will actually
run concurrently:

```markdown
## Planned waves
- W1: A1, T4
  - Base: <same accepted commit, tree, or recorded frozen base>
  - Independence: <current anchors proving disjoint writes/resources, preserved canonical/generated and migration/rollout coupling, and no produced/consumed interface or assumption>
```

Only positive evidence establishes a wave. A unit without that evidence remains
dependency-scheduled.

## Execution-ready entries

Cite the narrowest stable source anchor and state enough of the accepted
obligation to make execution unambiguous without copying source prose. Record
only execution-changing constraints and name an exact method or order only when
accepted design, generated-source, migration, rollout, or proof dependencies
fix it. Do not make implementation recover a critical invariant, non-goal,
value, interface, or proof expectation from a broad link or chat history.

`Owner/surface/resources` names every canonical writable owner, the initial
authorized paths or bounded discovery rule, and mutable or exclusive resources.
It does not select a carrier. A missing owner or generated authority reopens
design. `External input/gate` records later non-ledger availability; a mandatory
unavailable input belongs in `Blocked stop`.

Every Go task carries its owning package or bounded discovery rule, canonical
and derived surfaces, accepted semantic constraints, and the narrowest
repository-native proof with its expected observable. A known decision-changing
ambiguity blocks readiness now. `Reopen if` records only objective future
invalidation of an input accepted at readiness.
