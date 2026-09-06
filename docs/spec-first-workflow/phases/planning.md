# Planning

Use when structured work lacks one ready fixed implementation unit or the
smallest dependency-ordered ledger. Own order, unit boundaries, dependencies,
and final success conditions; never invent product behavior or architecture.
Leave test cases, fixtures, assertions, proving layers, and exact test commands
to Implementation. Do not require their prior design or approval.

Consume ready behavior, design, rollout, risk, and repository-owner inputs.
Reconcile every accepted implementation-changing obligation
to one independently acceptable task, named task deltas, proved
no-implementation, or scope exit.

Delta: Make the current-to-target change explicit and anchor its accepted
mechanism and ownership. Leave routine coding choices to Implementation.

Before expanding task packets, establish each Outcome, dependency, and
final observable, then apply [Task Review / Readiness's atomicity
criterion](task-review-readiness.md#atomicity-gate) as author self-check. Correct
unit boundaries in the same plan before detailing their packets; this adds no
intermediate artifact or review gate. The completed plan still receives the
independent review below.

Return one fixed inline unit when no durable boundary exists. Otherwise persist
[Task Ledger V1](../interfaces/task-ledger-v1.md), keep dependency, status, and
results in `tasks.md`, and put each task packet in `tasks/<ID>-<slug>.md` using
[Task Packet V1](../interfaces/task-packet-v1.md). Packets contain code and test-writing work.
Test execution and reviews belong
to Completion, not schedulable ledger tasks. The packet defines what must become
true; Implementation chooses how. Apply the [Planning Ledger
Contract](planning/ledger-contract.md). Load [Conditional Planning Branches](planning/conditional-branches.md)
only for integration-first uncertainty or broad contract fan-out, and [Planning
Proof And Readiness](planning/proof-and-readiness.md) before declaring
readiness.

Apply [Task Review / Readiness](task-review-readiness.md) through shared
[Review](../shared/review.md). Ready when the next unit can implement from
closed inputs without chat history or a new decision, with proof placed in
final validation and later dependencies annotated under the [Ready
Frontier](planning/ledger-contract.md#ready-frontier). Reopen the smallest
upstream owner of an input missing for the next authorized action.
