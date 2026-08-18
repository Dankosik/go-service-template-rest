# Planning

Turn accepted decisions into the smallest executable sequence. Planning chooses order and proof placement; it does not invent behavior or design.

## Read When

- Work has multiple dependent steps, owners, generated-source order, or validation checkpoints.
- Another actor/session needs a durable implementation ledger.
- Existing `tasks.md` needs repair.

Direct changes may use an inline plan.

## Inputs

- Ready spec and any required design/test/rollout context.
- Current source owners, generated/mirror commands, and repository validation commands.
- Accepted risks and proof obligations.

## Method

### Obligation Reconciliation Contract

1. Build a de-duplicated working set of implementation-changing accepted obligations from the ready inputs. Normalize restatements of the same obligation across specification, design, test, and rollout sources; discard rationale, rejected alternatives, non-normative examples, and future ideas. Treat two statements as one obligation only when their authority, required postcondition, execution-changing constraints, and proof consequence are equivalent. A non-equivalent normative conflict reopens its narrow upstream owner; Planning does not resolve it by normalization.
2. Give every obligation exactly one reconciliation disposition: one task, several named task deltas with a distinct postcondition and proof for each, a proved no-implementation disposition, or a scope exit. Compile those dispositions into the smallest coherent task outcomes under the task-boundary rule below.
3. Reconcile both directions: every task delta and proof maps to one obligation disposition or to the enabling change it serves, every obligation disposition is represented, and task boundaries follow valid postconditions rather than source-document structure.
4. Link-check the working set against the current repository and accepted deployment topology. For each accepted change to a contract, schema, canonical/generated authority, identifier, composition point, migration, or rollout state, confirm the accepted ownership record against every current producer, consumer, mirror, proof carrier, configuration/documentation surface, and replacement surface within its impact boundary. Give each reached surface one auditable boundary disposition: coupled into the outcome task, assigned to named task deltas whose intermediate states and handoffs satisfy the split rule, or proved unchanged. A required surface without accepted ownership or placement reopens its narrow design owner; Planning does not choose it.
5. Record a planned wave only when multiple ready acceptance units will actually run concurrently and current evidence establishes their independence.
6. Prove that the next acceptance unit or real wave is executable from closed inputs; later tasks need owners and dependencies, not prematurely materialized inputs.

A **scope exit** disposes of an obligation or existing task that sits beyond the
accepted outcome — mis-scoped while planning, or exposed by a later result.
Record one line: the gist, the current scope or non-goal wording that already
excludes it, and the owner who could reopen it. When no current wording excludes
it, this is a proposal to narrow rather than a scope exit; narrowing is
user-owned under [Decision
Ownership](../../../AGENTS.md#decision-ownership), so the obligation stays a task
or a `Blocked stop` until that owner answers. Unlike a proved no-implementation
disposition, which claims the obligation is already satisfied and cites its
proving surface, a scope exit is never completion evidence and is stated beside
any completion claim under the [Task
Contract](../../../AGENTS.md#task-contract).

A behavior-preserving restructure required by accepted Go Ownership to leave
one obligation task's touched surface coherent stays in that task when it shares
the same owner and proof boundary. Its surface names the moved declarations and
required callers, tests, generated or manual companions, and documentation; its
proof covers the preserved behavior as well as the task's changed behavior.

An **enabling change** carries no obligation of its own and exists only when a
separately consistent and provable restructure makes named obligation tasks
smaller or safer. Record it as a task ordered before those obligations, name
their task IDs in its outcome, and prove it by the current tests of the moved
surfaces passing unchanged. A restructure that names no enabled obligation task
is not an enabling change and does not enter the ledger.

The contract is complete when every accepted obligation has one auditable
disposition, every task delta and proof maps back to one disposition or to the
enabling change it serves, equivalent restatements alone are normalized, and
every reached current surface has an accepted task placement or an authoritative
unchanged disposition.

### Conditional Planning Branches

When integration is the primary uncertainty, make the next acceptance unit the smallest
production-grade end-to-end slice. The slice establishes one supported behavior
through the real production entry point, every uncertain integration seam, and
the final observable response, effect, or authoritative state, together with
the narrow failure or negative path required to falsify that integration.
Scaffolding, interface-only work, TODOs, mock success, and test-only wiring do
not satisfy the slice outcome; fixtures or test doubles may support proof only
behind an accepted seam. Prove that slice before expanding from it. Otherwise
keep local or already-proven work on its existing direct path.

When one mechanical contract change fans out so broadly that no bounded slice
can remain valid and green, plan `expand -> migrate -> contract`: add the
compatible new form, move bounded caller batches while both forms work, then
remove the old form after every consumer has moved. Keep the contract cleanup
in the same ledger and block it on every migration batch. Use one atomic task
when it can stay valid and provable; do not add compatibility machinery merely
to split work.

## Outputs

A compact `tasks.md`:

```markdown
# Goal
status: draft | ready | blocked | done
Completion: <observable successful condition>
Blocked stop: <what remains incomplete, evidence to record, and owner to reopen>
Global constraints: <exact constraints shared by multiple tasks; omit when none>

- [ ] T1: <verifiable postcondition; execution-changing accepted constraints>
  - Source: <narrow stable spec/design/test/rollout anchor(s)>
  - Owner/surface/resources: <canonical owner for each writable surface; initial authorized writable paths or bounded discovery rule; mutable, exclusive, or non-concurrent resources, or none>
  - Depends on: <ID — output handoff, exact consumed state, or exact safety/proof gate; needed to start, complete, or prove; or none>
  - Handoff: <for an output dependency: exact produced output and consumed input/acceptance condition; omit when none>
  - Alias of: <task ID and exact accepted receipt consumed; use only when this entry has no implementation delta; omit otherwise>
  - External input/gate: <required non-ledger input or rollout gate; named owner; objective availability checkpoint; omit when none>
  - Proof: <claim; command/check; expected observable>
  - Reopen if: <concrete objective future invalidation condition; upstream owner; omit when none>
```

### Ledger Layout

Keep every field inline in `tasks.md` by default. Move each task's execution
detail into one task file under `tasks/<ID>-<postcondition-slug>.md` when an
executing actor would otherwise read materially more ledger than its own unit
needs, or when a task's execution-critical content does not fit its entry. The
split moves detail only:

- `tasks.md` stays the index and the sole owner of lifecycle state: `status`,
  checkboxes, unit receipts, `Completion`, `Blocked stop`, `Global constraints`,
  `## Acceptance units`, `## Planned waves`, and the `Depends on` edges that
  order the work. It keeps each task's ID and postcondition title and links its
  task file.
- The task file owns the outcome body, its constraints, and the remaining entry
  fields. It carries no lifecycle state.

A split index entry keeps the postcondition title only:

```markdown
- [ ] T1: <verifiable postcondition> — file: tasks/T1-<postcondition-slug>.md
  - Depends on: <ID — output handoff, exact consumed state, or exact safety/proof gate; needed to start, complete, or prove; or none>
```

Obligation reconciliation, the acceptance-unit map, and the dependency graph
stay auditable from the index alone; no task file is required to read them.

#### Task File Contract

Write the task file for one actor that receives only [what crosses into a
Worker](../../agent-harness.md#what-crosses-into-a-worker) plus this file.

```markdown
# T1 — <postcondition title>

<the observable behavior that becomes true on the real production path, stated
as the caller sees it rather than as a layer-by-layer plan, with the
execution-changing accepted constraints and any preserved or forbidden behavior>

- Source: <narrow stable anchors>
- Owner/surface/resources: <canonical owner; authorized paths or discovery rule; mutable resources, or none>
- Handoff / Alias of / External input/gate: <when present>
- Proof: <claim; command/check; expected observable>
- Reopen if: <omit when none>
```

Inline a schema, interface, state transition, error body, or other shape when it
fixes an accepted decision more precisely than prose can, and trim it to the
decision-rich part; it is a contract to satisfy, not a solution to copy.

A task file adds no restatement of the index outcome and no skill routing.

### Ledger Entry Contract

Add only fields that change execution. Put a constraint in `Global constraints` only when its exact meaning applies across multiple tasks; keep task-specific constraints in the task outcome. Write each task title as the postcondition that becomes true. Put paths and symbols in `Owner/surface/resources` and commands in `Proof`; neither creates a task boundary. A task may carry prose beyond its fields when the postcondition needs it; keep that prose execution-changing.

### Task Boundary Contract

A split boundary is valid only when the completed task leaves the repository, and every deployment or migration state it creates or assumes, internally consistent, supported by the accepted compatibility or rollback policy, independently reviewable, and provable without unfinished companion work. Group the canonical source, generated or mirrored output, required tests and fixtures, migration/runtime compatibility, required documentation, and replacement cleanup needed for that state in the same task. Prefer the boundary that makes one accepted behavior reachable end to end through its real production entry point over one that completes a single layer of it. A layer-only task is valid when accepted rollout, migration, or `expand -> migrate -> contract` order fixes it, when that layer is the whole accepted outcome, or when it is an enabling change; otherwise its postcondition belongs in the task that makes the behavior reachable. As an oversized-task preflight, identify distinct ownership, review, failure/recovery, rollback, and proof domains inside the outcome. A useful split isolates a distinct owner, review/proof, failure/recovery, or rollback domain; creates a required handoff; enables an actual wave with positive independence evidence; or leaves an independently shippable accepted outcome. Keep the work in the same task when none of those benefits applies. File count, estimated minutes, and desired Worker count do not create an acceptance-unit boundary; a large writable surface still triggers the Implementation Slice DAG decomposition required by [Implementation Worker Execution](implementation-worker-execution.md#implementation-slice-dag).

### Acceptance Unit Contract

An **acceptance unit** is the smallest fixed candidate that one
Acceptance-Unit Lead can deliver through implementation slices, prove, review
when triggered, and integrate without a consumer depending on an intermediate
state. The ledger's acceptance-unit map is
authoritative: every implementation task is a singleton unit unless exactly one
recorded `Acceptance units` entry contains its task ID; membership in more than
one entry is invalid. Group adjacent ready tasks only when they share the same
canonical owner, editable boundary, proof preconditions, and final-state
validity; record a compact entry only for a grouped unit:

```markdown
## Acceptance units
- A1: T2, T3 — <shared owner, boundary, and proof reason>
```

The unit is the Lead, final-proof, review, and integration boundary; internal
Workers own only Slice DAG nodes and create no acceptance state. The unit is not
another task lifecycle. A task whose only postcondition is receipt of another
task's accepted result is a **receipt alias**: record `Alias of`, give it no
writable surface or proof command, and close it mechanically when the named
receipt is present. Receipt aliases never create a Worker, reviewer, validation
run, or integration commit.

### Dependency And Wave Contract

For sequential work, `Depends on` is the complete ordering authority; do not create one-task waves. Record an edge only when the downstream task consumes the upstream task's output or state, or must cross its safety or proof gate, and name whether the edge is required to start, complete, or prove the downstream task. Document order, review preference, and convenient sequencing are not dependencies. For an output edge, record the produced/consumed contract once in `Handoff`; in `Depends on`, write only `<ID> — output handoff — needed to <start|complete|prove>`. For a state or safety/proof gate, omit `Handoff` and name the consumed state or gate in `Depends on`.

Add one compact `Planned waves` section only when at least two ready acceptance units will actually run concurrently:

```markdown
## Planned waves
- W1: A1, T4
  - Base: <same accepted commit, tree, or recorded frozen base>
  - Independence: <current anchors showing pairwise-disjoint writable surfaces and mutable resources, preserved canonical/generated and migration/rollout coupling, and no interface or assumption produced by one member and consumed by another>
```

Use a singleton task ID for its implied unit and a grouped unit ID where one is
recorded. Only positive evidence establishes a wave. A unit whose independence
is unavailable remains dependency-scheduled until current evidence establishes
the boundary. Implementation may narrow a planned wave when current evidence
changes.

### Execution-Ready Task Contract

Cite the narrowest stable source anchor and state enough of the relevant accepted obligation in the task outcome to make execution unambiguous; do not copy source prose. State the verifiable postcondition and only execution-changing accepted constraints, including preserved or forbidden behavior and any accepted state-transition, data-flow, failure/recovery, privacy, or security boundary. Do not prescribe discretionary coding steps; name an exact method or order only when accepted design, generated-source, migration, rollout, or proof dependencies fix it. Do not make implementation recover an execution-critical constraint, invariant, non-goal, exact value, interface, or proof expectation from a broad document link or chat history.

`Owner/surface/resources` names the canonical owner for every writable surface, the initial authorized writable paths or bounded discovery rule, and every mutable, exclusive, or non-concurrent external or proof resource that can affect execution, such as a database, port, environment, migration target, destructive fixture, lock, or generated pair; use `none` when there is no such resource. A discovery rule may resolve exact files only inside an already accepted owner; it names the inspection bound and deterministic placement rule and grants write authority only to the resolved companion surfaces. If the owning repository, package, or generated authority is still a choice, reopen its design owner. `Owner/surface/resources` records authoritative implementation, data, generated-source, and external ownership plus the writable and mutable-resource envelope; it does not select an execution carrier. Root-local versus Worker execution, checkout or worktree, model, and harness control remain Implementation decisions under [Implementation Worker Execution](implementation-worker-execution.md). `External input/gate` records later non-ledger availability; if it is mandatory for the next unit and unavailable, it belongs in `Blocked stop`.

Every Go implementation task carries the owning package or a bounded discovery
rule, the canonical source and any derived generated surfaces, accepted Go
semantic constraints, and the narrowest repository-native proof command with
its expected observable. When accepted Go Ownership fixes exact files and its
inverse file map, `Source` cites that design anchor and
`Owner/surface/resources` preserves those files. A package-wide surface or
discovery rule may replace them only when the design itself recorded the
deterministic implementation-local rule; Planning does not widen a fixed file
map. Do not make the Worker rediscover these from broad context.

A known decision-changing ambiguity or missing input required by a mandatory path through the current completion condition belongs in `Blocked stop` and blocks readiness now. `Reopen if` is optional and records only a concrete objective future condition that would invalidate an input accepted at readiness; omit it when none exists, and do not use it to defer a known question to implementation.

### Proof Placement Contract

Name the claim before its check. A command is not proof unless its expected observable can establish that claim. Prefer the smallest repository-native automated check unless the accepted proof strategy requires manual observation or automation cannot establish the required observable.

Attach each proof to the earliest task whose completed output makes its claim true, and require that proof before accepting the task. A later proof task is valid only for a cross-task, deployed, migration, or environment claim that cannot exist earlier; it names the exact accepted upstream outputs it consumes and proves only that integrated claim. It may inspect the complete integrated candidate, but its acceptance boundary remains its recorded singleton or grouped unit; wider evidence does not accept upstream units or widen the current verdict. Evidence that invalidates an accepted upstream input follows the workflow's narrow reopen contract.

Planning must make these explicit where relevant:

- canonical source before generated/mirrored output;
- accepted regression-proof order inside the task whose outcome makes the behavior true, including test-plan scenario IDs; a deliberately failing intermediate check is not a completed task boundary;
- accepted performance workload/scale boundaries, hot-path amplification or resource constraints, and matching benchmark, load, profile, query-count, or other claim-matched proof;
- migrations/backfills/rollout order and rollback gates;
- cleanup of replaced code, tests, fixtures, config, docs, skills, or mirrors;
- fresh validation and negative proof for retired identifiers;
- a positive independence basis only for an actual parallel wave;
- one successful completion condition distinct from blocked stop.

Preserve an accepted example or scenario when it defines required behavior or proof. Use local obligation keys only when dense inputs cannot otherwise be audited from narrow source anchors. A no-implementation disposition must cite either current authoritative evidence that the obligation is already satisfied or an accepted upstream decision that no implementation change is required, plus its proving surface or objective recheck condition. When one obligation requires several task deltas, its single reconciliation disposition lists those task IDs; each task carries only its distinct postcondition, proof obligation, and actual interface or handoff. Put an unchanged constraint shared by several tasks once in `Global constraints`. Keep reconciliation inline unless the mapping is too dense to audit without a compact table; do not create a separate traceability artifact by default.

### Readiness Dry Run

Before readiness, walk the next acceptance unit or actual parallel wave through its proof using current inputs. Also resolve any later decision that could invalidate that work. A later unavailable input keeps its dependent task pending with an owner and checkpoint; it blocks readiness only when the next accepted result would otherwise be unusable or when final completion is being claimed.

## Readiness Review

Apply focused root self-review before implementation. Run independent [Task Review / Readiness](task-review-readiness.md) only when the shared review trigger applies.

Repair planning-owned findings directly. Reopen an earlier owner when a task would need to choose product behavior, source of truth, runtime mechanism, package ownership, test strategy, or rollout policy.

Task review and planning-owned disposition are internal checkpoints. Fresh review follows only `FAIL` repair or material candidate change.

## Stop Rule

The ledger is ready only when the Obligation Reconciliation, Ledger Entry, Task
Boundary, Acceptance Unit, Dependency And Wave, Execution-Ready Task, and Proof
Placement contracts above all pass. No accepted obligation or reached current
surface lacks one auditable disposition, and no required owner, placement,
resource, gate, order, handoff, proof, or objective reopen condition remains
implicit or duplicated.

The Readiness Dry Run must reach acceptance for the next unit or actual wave
using only the fixed ledger, cited current inputs, and available mandatory gates,
without chat history, unfinished companion work, or a new behavior, mechanism,
placement, ownership, proof, rollout, concurrency, or carrier decision. Every
actual wave has current positive pairwise independence evidence; later work
remains owned and dependency-ordered and cannot invalidate the next accepted
result. Any triggered review has returned `PASS` or dispositioned `CONCERNS`.
