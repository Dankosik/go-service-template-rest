# Planning

Use when structured work lacks one ready fixed implementation unit or the
smallest dependency-ordered ledger. Own order, unit boundaries, dependencies,
and proof placement; never invent behavior, design, or proof strategy.

## Inputs

Consume ready behavior, design, proof, rollout, and risk decisions plus current
canonical/generated owners and repository validation commands.

## Method

1. Build a de-duplicated set of implementation-changing accepted obligations.
   Normalize only equivalent authority, postcondition, constraints, and proof;
   a conflict reopens its narrow owner.
2. Give every obligation one disposition: one unit, named unit deltas with
   distinct postconditions/proof, proved no-implementation, or scope exit backed
   by current accepted scope.
3. Reconcile both directions: every retained delta/proof maps to an obligation
   or enabling change, and every obligation plus reached producer, consumer,
   mirror, configuration, replacement, and proof carrier has one disposition.
4. Split only for a distinct postcondition, owner, consumed dependency, rollout
   sequence, valid handoff, or independently acceptable proof boundary. Record
   only real output/safety dependencies.
5. Dry-run the next unit from current closed inputs through its claim-matched
   proof. Later work needs owners/dependencies, not invented inputs.

Load [Conditional Planning Branches](planning/conditional-branches.md) only for
integration-first uncertainty or broad mechanical contract fan-out. Load
[Planning Proof And Readiness](planning/proof-and-readiness.md) before declaring
the result ready.

## Output

Return one fixed inline acceptance unit when no durable boundary exists.
Otherwise persist [Task Ledger V1](../interfaces/task-ledger-v1.md) and apply the
[Planning Ledger Contract](planning/ledger-contract.md). Keep ordinary paths and
repository facts in their canonical owners.

## Review

Self-review every fixed inline or persisted result, then apply shared
[Review](../shared/review.md) through [Task Review /
Readiness](task-review-readiness.md) before returning `ready`.

## Exit And Reopen

Exit when obligation reconciliation passes and the next unit can reach
acceptance without chat history, unfinished companion work, or a new behavior,
mechanism, placement, ownership, proof, rollout, authority, concurrency, or
carrier decision. Reopen the smallest upstream owner of any missing choice or
input.
