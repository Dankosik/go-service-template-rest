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

- [ ] T1: <outcome>
  - Source: <spec/design/test anchor>
  - Owner/surface: <package/file or discovery boundary>
  - Depends on: <IDs or none>
  - Proof: <command/manual observable>
  - Reopen if: <missing decision/evidence and owner>
```

Add only fields that change execution. Group work into reviewable diff stories; split when a checkpoint must pass before dependent work begins. Keep coupled source/generated/test/doc edits together when separation would create a broken intermediate state.

Planning must make these explicit where relevant:

- canonical source before generated/mirrored output;
- proof-first regression work and test-plan scenario IDs;
- migrations/backfills/rollout order and rollback gates;
- cleanup of replaced code, tests, fixtures, config, docs, skills, or mirrors;
- fresh validation and negative proof for retired identifiers;
- one successful completion condition distinct from blocked stop.

## Readiness Review

Perform a focused ledger self-check. Use independent [Task Review / Readiness](task-review-readiness.md) when the implementation is high-impact, broad, delegated, hard to reverse, or explicitly gated.

Repair planning-owned findings directly. Reopen an earlier owner when a task would need to choose product behavior, source of truth, runtime mechanism, package ownership, test strategy, or rollout policy.

Task review, planning-owned repair, and fresh re-review are internal checkpoints in the same root session. They do not produce a next-session prompt.

## Stop Rule

Implementation may start when tasks are dependency-ordered, owned, traceable, and provable, and the required readiness review has no blocker. Do not start implementation merely because a draft ledger exists.
