# Shared Specialist Contract

Use this result protocol for a delegated Decision or Review, or when the active
artifact requires its interface. Applying a domain method locally does not
create a specialist handoff; keep the judgment and evidence in the code or
existing task artifact under the active workflow.

Select the domain by the absent/changing decision or violated accepted contract.
Use **Decision** for policy and **Review** for conformance. Load one domain
reference by default and another only for an independent pressure.

Interpret domain proof requirements through the [Evidence
Contract](../../docs/spec-first-workflow/shared/evidence-contract.md#design-proof)
for the claim and active phase. During ledger implementation, a domain method's
proof requirements describe final evidence; they do not expand the execution
boundary owned by [Implementation](../../docs/spec-first-workflow/phases/implementation.md).

A Decision returns [Decision Result
V1](../../docs/spec-first-workflow/interfaces/decision-result-v1.md). A Review
returns the shared [Review Result
V1](../../docs/spec-first-workflow/interfaces/review-result-v1.md). Missing
policy returns to that domain's Decision branch or nearest decision owner. Load
[specialist arbitration](specialist-arbitration.md) only for ambiguous overlap.

Stay inside the selected domain and return only its judgment, proof, or gap to
the requesting parent. A gap names the missing input, dependent action, nearest
owner, and next discriminating evidence or decision. Preserve unknowns instead
of inventing values or proof. The parent applies [Parent-Owned
Recovery](../../docs/spec-first-workflow/shared/transition.md#parent-owned-recovery).
