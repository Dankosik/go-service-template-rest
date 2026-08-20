# Implementation / Validation / Closeout

Close one accepted outcome at the earliest proven state that satisfies it while
preserving already-valid behavior, decisions, candidate state, and proof.

## Read When

- Structured or orchestrated work enters Implementation.
- Direct work reaches non-obvious validation, deployment or remote proof,
  integration, independent review, or blocked closeout.

## Inputs And Outputs

Start from the accepted inline outcome or smallest relevant durable artifact,
the current repository state, canonical sources, and repository-native proof.
Return the bounded change, claim-scoped evidence, and an evidence-clamped
closeout or exact blocker.

## Fixed Unit And Sequence

Direct work keeps one fixed inline unit and the current root owns acceptance.
Structured work binds the current root as one `ACCEPTANCE_UNIT_LEAD`;
orchestrated work receives one Lead from the Ledger Orchestrator.

```text
bind unit -> implement -> self-review -> validate
-> optional independent review -> accept -> persist transition -> close out
```

Keep the unit boundary fixed through the sequence. After acceptance, apply
[Acceptance-Unit Closure](../shared/acceptance-unit-closure.md) before selecting
later work.

## Implement

Direct work follows [Direct Work](../direct-work.md). Structured and
orchestrated work binds one role from the [Execution Role
Tree](implementation-worker-execution.md#execution-role-tree); that role's
skill owns its method. Apply only the Go and domain skills selected by [Go
Change Surface](../../../AGENTS.md#go-change-surface). Evidence changing
accepted behavior, architecture, ownership, proof strategy, or rollout reopens
its narrowest upstream owner.

## Review And Validate

Self-review the bounded diff and observable production path against every
accepted criterion and triggered risk. Use repository [Validation
Routing](../../validation-routing.md) for the smallest matching proof, then
apply the shared [Evidence Contract](../shared/evidence-contract.md) to every
readiness or completion claim.

Use matching domain skills as review lenses. Unrelated or pre-existing defects
and unproven suspicions remain observations outside the fixed unit.

Before deployment or remote proof, apply [Deployment And Remote-Proof
Preflight](../shared/deployment-proof-preflight.md). After the fixed candidate
passes root review and mapped validation, apply [Review
Independence](../shared/review-independence.md). When triggered, load
[Independent Implementation Review](../shared/implementation-review.md).

## Close Out

State the changed outcome, important behavior consequence, proof actually run,
and remaining gap or reopen owner. Move durable decisions to their canonical
owner with the provenance and reopen condition required by [Artifact
Model](../shared/artifact-model.md#resume-order) before removing completed task
artifacts.

## Stop Rule

Close only when every accepted changed or deliberately unchanged behavior has
a terminal disposition; the complete bounded outcome and required real wiring,
cleanup, and canonical/generated synchronization are present; the retained
delta contains no unrelated change and has been reviewed against every
triggered risk; and the [Evidence Contract](../shared/evidence-contract.md)
permits every completion claim. A changed decision reopens its narrowest
upstream owner. After acceptance, only [Acceptance-Unit
Closure](../shared/acceptance-unit-closure.md) may precede closeout or later
work.
