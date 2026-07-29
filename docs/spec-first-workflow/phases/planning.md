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

1. Build a de-duplicated working set of implementation-changing accepted obligations from the ready inputs. Normalize restatements of the same obligation across specification, design, test, and rollout sources; discard rationale, rejected alternatives, non-normative examples, and future ideas.
2. Give every obligation exactly one reconciliation disposition: one task, several named task deltas with a distinct postcondition and proof for each, or a proved no-implementation disposition. Compile those dispositions into the smallest coherent task outcomes under the task-boundary rule below.
3. Reconcile both directions: every task delta and proof maps to one obligation disposition, every obligation disposition is represented, and task boundaries follow valid postconditions rather than source-document structure.
4. Record a planned wave only when multiple ready tasks will actually run concurrently and current evidence establishes their independence.
5. Prove that the next task or real wave is executable from closed inputs; later tasks need owners and dependencies, not prematurely materialized inputs.

When integration is the primary uncertainty, make the next task the smallest
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
  - Depends on: <ID — exact output/state or safety/proof gate consumed; needed to start, complete, or prove; or none>
  - Handoff: <exact consumed/produced contract for an output dependency; omit when none>
  - External input/gate: <required non-ledger input or rollout gate; named owner; objective availability checkpoint; omit when none>
  - Proof: <claim; command/check; expected observable>
  - Reopen if: <concrete objective future invalidation condition; upstream owner; omit when none>
```

Add only fields that change execution. Put a constraint in `Global constraints` only when its exact meaning applies across multiple tasks; keep task-specific constraints in the task outcome. Write each task title as the postcondition that becomes true. Put paths and symbols in `Owner/surface/resources` and commands in `Proof`; neither creates a task boundary.

A split boundary is valid only when the completed task leaves the repository, and every deployment or migration state it creates or assumes, internally consistent, supported by the accepted compatibility or rollback policy, independently reviewable, and provable without unfinished companion work. Group the canonical source, generated or mirrored output, required tests and fixtures, migration/runtime compatibility, required documentation, and replacement cleanup needed for that state in the same task. As an oversized-task preflight, identify distinct ownership, review, failure/recovery, rollback, and proof domains inside the outcome. Split separable domains when each can end at such a boundary. Do not use file count, estimated minutes, or desired Worker count as a sizing rule.

For sequential work, `Depends on` is the complete ordering authority; do not create one-task waves. Record an edge only when the downstream task consumes the upstream task's output or state, or must cross its safety or proof gate, and name whether the edge is required to start, complete, or prove the downstream task. Document order, review preference, and convenient sequencing are not dependencies. Use `Handoff` only for an output edge and name both the produced and consumed sides without copying implementation steps.

Add one compact `Planned waves` section only when at least two ready tasks will actually run concurrently:

```markdown
## Planned waves
- W1: T1, T2
  - Base: <same accepted commit, tree, or recorded frozen base>
  - Independence: <current anchors showing pairwise-disjoint writable surfaces and mutable resources, preserved canonical/generated and migration/rollout coupling, and no interface or assumption produced by one member and consumed by another>
```

Only recorded positive evidence establishes a wave. A task whose independence is unavailable remains dependency-scheduled until current evidence establishes the boundary. Implementation may narrow a planned wave when current evidence changes.

Cite the narrowest stable source anchor and state enough of the relevant accepted obligation in the task outcome to make execution unambiguous; do not copy source prose. State the verifiable postcondition and only execution-changing accepted constraints, including preserved or forbidden behavior and any accepted state-transition, data-flow, failure/recovery, privacy, or security boundary. Do not prescribe discretionary coding steps; name an exact method or order only when accepted design, generated-source, migration, rollout, or proof dependencies fix it. Do not make implementation recover an execution-critical constraint, invariant, non-goal, exact value, interface, or proof expectation from a broad document link or chat history.

`Owner/surface/resources` names the canonical owner for every writable surface, the initial authorized writable paths or bounded discovery rule, and every mutable, exclusive, or non-concurrent external or proof resource that can affect execution, such as a database, port, environment, migration target, destructive fixture, lock, or generated pair; use `none` when there is no such resource. A discovery rule may resolve exact files only inside an already accepted owner; it names the inspection bound and deterministic placement rule and grants write authority only to the resolved companion surfaces. If the owning repository, package, or generated authority is still a choice, reopen its design owner. `External input/gate` records later non-ledger availability; if it is mandatory for the next task and unavailable, it belongs in `Blocked stop`.

Every Go implementation task carries the owning package or a bounded discovery
rule, the canonical source and any derived generated surfaces, accepted Go
semantic constraints, and the narrowest repository-native proof command with
its expected observable. Do not make the Worker rediscover these from broad
context.

A known decision-changing ambiguity or missing input required by a mandatory path through the current completion condition belongs in `Blocked stop` and blocks readiness now. `Reopen if` is optional and records only a concrete objective future condition that would invalidate an input accepted at readiness; omit it when none exists, and do not use it to defer a known question to implementation.

Name the claim before its check. A command is not proof unless its expected observable can establish that claim. Prefer the smallest repository-native automated check unless the accepted proof strategy requires manual observation or automation cannot establish the required observable.

Attach each proof to the earliest task whose completed output makes its claim true, and require that proof before accepting the task. A later proof task is valid only for a cross-task, deployed, migration, or environment claim that cannot exist earlier; it names the exact prior outputs it consumes and proves only that integrated claim.

Planning must make these explicit where relevant:

- canonical source before generated/mirrored output;
- proof-first regression work and test-plan scenario IDs;
- accepted performance workload/scale boundaries, hot-path amplification or resource constraints, and matching benchmark, load, profile, query-count, or other claim-matched proof;
- migrations/backfills/rollout order and rollback gates;
- cleanup of replaced code, tests, fixtures, config, docs, skills, or mirrors;
- fresh validation and negative proof for retired identifiers;
- a positive independence basis only for an actual parallel wave;
- one successful completion condition distinct from blocked stop.

Preserve an accepted example or scenario when it defines required behavior or proof. Use local obligation keys only when dense inputs cannot otherwise be audited from narrow source anchors. A no-implementation disposition must cite either current authoritative evidence that the obligation is already satisfied or an accepted upstream decision that no implementation change is required, plus its proving surface or objective recheck condition. When one obligation requires several task deltas, its single reconciliation disposition lists those task IDs; each task carries only its distinct postcondition, proof obligation, and actual interface or handoff. Put an unchanged constraint shared by several tasks once in `Global constraints`. Keep reconciliation inline unless the mapping is too dense to audit without a compact table; do not create a separate traceability artifact by default.

Before readiness, walk the next task or actual parallel wave through its proof using current inputs. Also resolve any later decision that could invalidate that work. A later unavailable input keeps its dependent task pending with an owner and checkpoint; it blocks readiness only when the next accepted result would otherwise be unusable or when final completion is being claimed.

## Readiness Review

Apply focused root self-review before implementation. Run independent [Task Review / Readiness](task-review-readiness.md) only when the shared review trigger applies.

Repair planning-owned findings directly. Reopen an earlier owner when a task would need to choose product behavior, source of truth, runtime mechanism, package ownership, test strategy, or rollout policy.

Task review and planning-owned disposition are internal checkpoints. Fresh review follows only `FAIL` repair or material candidate change.

## Stop Rule

The ledger is ready only when every implementation-changing accepted obligation has one reconciliation disposition; every task is an outcome-shaped valid boundary with its accepted writable owner or discovery rule, consumed outputs and gates, coupled companion changes, claim-matched task-local proof, and objective reopen condition when one exists; and an executor dry-run of the next task or recorded wave reaches acceptance without chat history, an unavailable mandatory input, or a new behavior, mechanism, placement, ownership, test/proof strategy, rollout, or concurrency decision.

Every planned wave carries current positive independence evidence. Later work remains owned and dependency-ordered; later unavailable inputs have named owners and objective checkpoints and cannot invalidate the next accepted result. Any triggered review must return `PASS` or dispositioned `CONCERNS`.
