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

1. Derive implementation-changing obligations from the ready inputs; discard rationale, rejected alternatives, non-normative examples, and future ideas.
2. Express the smallest coherent task outcomes that can each end in a valid, reviewable, provable state while keeping coupled source/generated/test/doc changes together.
3. Reconcile both directions: each task and proof traces to an accepted obligation, and no accepted implementation delta is lost or duplicated.
4. Record a planned wave only when multiple ready tasks will actually run concurrently and current evidence establishes their independence.
5. Prove that the next task or real wave is executable from closed inputs; later tasks need owners and dependencies, not prematurely materialized inputs.

When a change crosses several layers and integration is the main uncertainty,
make the next task the smallest production-grade end-to-end slice, prove its
real path, then expand from that proven slice. Otherwise keep local or
already-proven work on its existing direct path. The slice contains only
retained implementation and required proof.

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
  - Owner/surface/resources: <package/file or discovery boundary; mutable, exclusive, or non-concurrent resources, or none>
  - Depends on: <IDs or none>
  - Handoff: <exact consumed/produced contract; omit when none>
  - Proof: <claim; command/check; expected observable>
  - Reopen if: <concrete objective future invalidation condition; upstream owner; omit when none>
```

Add only fields that change execution. Put a constraint in `Global constraints` only when its exact meaning applies across multiple tasks; keep task-specific constraints in the task outcome. Treat each task as one coherent outcome that is independently reviewable once its declared prerequisites are satisfied. As an oversized-task preflight, identify distinct ownership, review, failure/recovery, rollback, and proof domains inside the outcome. Split separable domains when each can end in a valid provable state; keep coupled source/generated/test/doc edits together when separation would create a broken intermediate state or an implicit handoff. Do not use file count, estimated minutes, or desired Worker count as a sizing rule. Use `Depends on` only when an upstream output or gate is required to start, safely complete, or prove the downstream task; document order, review preference, and convenient sequencing are not dependencies.

For sequential work, `Depends on` is the complete ordering authority; do not create one-task waves. Add one compact `Planned waves` section only when at least two ready tasks will actually run concurrently. Wave members must start from the same accepted base, have disjoint writable and mutable-resource ownership, preserve canonical/generated and migration/rollout coupling, and not change an interface or assumption consumed by another member. Implementation may narrow a planned wave when current evidence changes.

Cite the narrowest stable source anchor and state enough of the relevant accepted obligation in the task outcome to make execution unambiguous; do not copy source prose. State the verifiable postcondition and only execution-changing accepted constraints, including preserved or forbidden behavior and any accepted state-transition, data-flow, failure/recovery, privacy, or security boundary. Do not prescribe discretionary coding steps; name an exact method or order only when accepted design, generated-source, migration, rollout, or proof dependencies fix it. Do not make implementation recover an execution-critical constraint, invariant, non-goal, exact value, interface, or proof expectation from a broad document link or chat history. `Owner/surface/resources` names the concrete writable owner or bounded discovery rule and every mutable, exclusive, or non-concurrent external or proof resource that can affect execution, such as a database, port, environment, migration target, destructive fixture, lock, or generated pair; use `none` when there is no such resource. Use `Handoff` only when another task consumes an exact produced contract; name both sides without copying implementation steps. Use a discovery boundary only when the exact files cannot be known before execution; bound what may be inspected and name the deterministic placement rule or canonical source that resolves the file choice.

Every Go implementation task carries the owning package or a bounded discovery
rule, the canonical source and any derived generated surfaces, accepted Go
semantic constraints, and the narrowest repository-native proof command with
its expected observable. Do not make the Worker rediscover these from broad
context.

A known decision-changing ambiguity or missing input required by a mandatory path through the current completion condition belongs in `Blocked stop` and blocks readiness now. `Reopen if` is optional and records only a concrete objective future condition that would invalidate an input accepted at readiness; omit it when none exists, and do not use it to defer a known question to implementation.

Name the claim before its check. A command is not proof unless its expected observable can establish that claim. Prefer the smallest repository-native automated check unless the accepted proof strategy requires manual observation or automation cannot establish the required observable.

Planning must make these explicit where relevant:

- canonical source before generated/mirrored output;
- proof-first regression work and test-plan scenario IDs;
- accepted performance workload/scale boundaries, hot-path amplification or resource constraints, and matching benchmark, load, profile, query-count, or other claim-matched proof;
- migrations/backfills/rollout order and rollback gates;
- cleanup of replaced code, tests, fixtures, config, docs, skills, or mirrors;
- fresh validation and negative proof for retired identifiers;
- a positive independence basis only for an actual parallel wave;
- one successful completion condition distinct from blocked stop.

Preserve an accepted example or scenario when it defines required behavior or proof. Use local obligation keys only when dense inputs cannot otherwise be audited from narrow source anchors. A no-implementation disposition must cite either current authoritative evidence that the obligation is already satisfied or an accepted upstream decision that no implementation change is required, plus its proving surface or objective recheck condition. An obligation may span multiple tasks when each carries the relevant constraint and states its distinct delta and proof obligation, plus any actual interface or handoff. Keep reconciliation inline unless the mapping is too dense to audit without a compact table; do not create a separate traceability artifact by default.

Before readiness, walk the next task or actual parallel wave through its proof using current inputs. Also resolve any later decision that could invalidate that work. A later unavailable input keeps its dependent task pending with an owner and checkpoint; it blocks readiness only when the next accepted result would otherwise be unusable or when final completion is being claimed.

## Readiness Review

Apply focused root self-review before implementation. Run independent [Task Review / Readiness](task-review-readiness.md) only when the shared review trigger applies.

Repair planning-owned findings directly. Reopen an earlier owner when a task would need to choose product behavior, source of truth, runtime mechanism, package ownership, test strategy, or rollout policy.

Task review and planning-owned disposition are internal checkpoints. Fresh review follows only `FAIL` repair or material candidate change.

## Stop Rule

The ledger is ready when the next task or real parallel wave can execute and prove its outcome from closed inputs without chat history or a new behavior, mechanism, ownership, proof, or concurrency decision. Later work remains owned and dependency-ordered. Any triggered review must return `PASS` or dispositioned `CONCERNS`.
