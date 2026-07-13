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

## Outputs

A compact `tasks.md`:

```markdown
# Goal
status: draft | ready | blocked | done
Completion: <observable successful condition>
Blocked stop: <what remains incomplete, evidence to record, and owner to reopen>

- [ ] T1: <verifiable postcondition; execution-changing accepted constraints>
  - Source: <narrow stable spec/design/test/rollout anchor(s)>
  - Owner/surface: <package/file or discovery boundary>
  - Depends on: <IDs or none>
  - Proof: <claim; command/check; expected observable>
  - Reopen if: <concrete objective future invalidation condition; upstream owner; omit when none>
```

Add only fields that change execution. Treat each task as one coherent outcome that is independently reviewable once its declared prerequisites are satisfied. Split separable outcomes when they have distinct proof, review, rollback, or ownership boundaries; keep coupled source/generated/test/doc edits together when separation would create a broken intermediate state. Use `Depends on` only when an upstream output or gate is required to start, safely complete, or prove the downstream task; document order, review preference, and convenient sequencing are not dependencies.

Cite the narrowest stable source anchor and state enough of the relevant accepted obligation in the task outcome to make execution unambiguous; do not copy source prose. State the verifiable postcondition and only execution-changing accepted constraints, including preserved or forbidden behavior and any accepted state-transition, data-flow, failure/recovery, privacy, or security boundary. Do not prescribe discretionary coding steps; name an exact method or order only when accepted design, generated-source, migration, rollout, or proof dependencies fix it. Do not make implementation recover an execution-critical constraint, invariant, non-goal, exact value, or proof expectation from a broad document link or chat history. Name a concrete owning surface when current repository evidence can establish it. Use a discovery boundary only when the exact files cannot be known before execution; bound what may be inspected and name the deterministic placement rule or canonical source that resolves the file choice.

A known decision-changing ambiguity or missing input required by a mandatory path through the current completion condition belongs in `Blocked stop` and blocks readiness now. `Reopen if` is optional and records only a concrete objective future condition that would invalidate an input accepted at readiness; omit it when none exists, and do not use it to defer a known question to implementation.

Name the claim before its check. A command is not proof unless its expected observable can establish that claim. Prefer the smallest repository-native automated check unless the accepted proof strategy requires manual observation or automation cannot establish the required observable.

Planning must make these explicit where relevant:

- canonical source before generated/mirrored output;
- proof-first regression work and test-plan scenario IDs;
- migrations/backfills/rollout order and rollback gates;
- cleanup of replaced code, tests, fixtures, config, docs, skills, or mirrors;
- fresh validation and negative proof for retired identifiers;
- one successful completion condition distinct from blocked stop.

Before drafting tasks, identify every in-scope accepted obligation across the ready inputs. Treat rationale, rejected alternatives, non-normative examples, and future ideas as context, not implementation work. Preserve an accepted example or scenario when it defines required behavior or proof. Use local obligation keys only when dense inputs cannot otherwise be audited from narrow source anchors.

After drafting, map each accepted obligation to one or more tasks and adequate proofs, or record an evidence-backed no-implementation disposition. A no-implementation disposition must cite either current authoritative evidence that the obligation is already satisfied or an accepted upstream decision that no implementation change is required, plus its proving surface or objective recheck condition. Then verify the reverse direction: every task must trace to an accepted obligation, every proof to its task's claim, and no implementation delta may be duplicated or fall between task boundaries; merge duplicated deltas. An obligation may span multiple tasks when each carries the relevant constraint and states its distinct delta and proof obligation, plus any actual interface or handoff. Keep this reconciliation inline unless the mapping is too dense to audit without a compact table; do not create a separate traceability artifact by default.

Before readiness, cold-walk every mandatory dependency path from each dependency root through final validation. Each root task must be startable from canonical, mechanically derivable, or currently available external inputs; each downstream task and required proof must become startable and completable from those inputs plus the accepted outputs of its completed dependencies. A known unavailable external gate may remain only when its dependent task and claim are excluded from the current completion and routed to a later ledger; it may not sit on a path to final validation. Never approve `PASS subject to gates` for a mandatory path.

## Readiness Review

For structured or orchestrated work, run independent [Task Review / Readiness](task-review-readiness.md) before implementation. Direct work needs only its inline plan and focused self-check unless the user or risk requires independent review.

Repair planning-owned findings directly. Reopen an earlier owner when a task would need to choose product behavior, source of truth, runtime mechanism, package ownership, test strategy, or rollout policy.

Task review, planning-owned repair, and fresh re-review are internal checkpoints in the same root session. They do not produce a next-session prompt.

## Stop Rule

Implementation may start when every mandatory path through the completion condition is dependency-ordered, owned, traceable, provable, executable in dependency order from current inputs, and the required readiness review has returned `PASS`. Do not start implementation merely because a draft ledger exists.
