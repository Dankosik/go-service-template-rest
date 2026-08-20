# Planning

Use when structured work lacks one ready fixed implementation unit or the
smallest dependency-ordered ledger. Own order, unit boundaries, dependencies,
and proof placement; never invent behavior, design, ownership, or proof strategy.

Consume ready behavior, design, proof, rollout, risk, repository-owner, and
validation inputs. Reconcile every accepted implementation-changing obligation
to one unit, named unit deltas, proved no-implementation, or scope exit. Split
only for a distinct postcondition, owner, consumed dependency, rollout sequence,
valid handoff, or independently acceptable proof boundary.

Return one fixed inline unit when no durable boundary exists. Otherwise persist
[Task Ledger V1](../interfaces/task-ledger-v1.md) and apply the [Planning Ledger
Contract](planning/ledger-contract.md). Load [Conditional Planning
Branches](planning/conditional-branches.md) only for integration-first
uncertainty or broad contract fan-out, and [Planning Proof And
Readiness](planning/proof-and-readiness.md) before declaring readiness.

Apply [Task Review / Readiness](task-review-readiness.md) through shared
[Review](../shared/review.md). Ready when the next unit can reach acceptance from
closed inputs without chat history, companion work, or a new decision. Reopen
the smallest upstream owner of any missing choice or input.
