# Workflow Artifact Model

Persist only what must survive the current reasoning pass. Stop at the smallest artifact set that lets the next actor act, prove the result, and identify the reopen owner without chat archaeology; artifacts are communication tools, not phase receipts.

## Read When

- Deciding whether a result belongs in chat, `spec.md`, `design/`, `test-plan.md`, `tasks.md`, research notes, `rollout.md`, or `workflow-plan.md`.
- Resuming work or deciding which file owns current state.
- Reopening a decision after new evidence.

## Inputs

- Accepted user outcome and current execution path from `AGENTS.md`.
- Existing task-local artifacts and repository sources of truth.
- The current phase's unique decisions and proof needs.

## Outputs

- The smallest useful artifact set.
- One clear owner for each decision, task, and proof obligation.
- A compact current status or blocker when resume is needed.

## Stop Rule

Stop adding artifact structure when the next actor can act correctly, prove the result, and identify the reopen owner without chat archaeology.

## When To Persist

Persist a result only when at least one is true:

- another phase or person must consume it;
- work will likely resume in a later session;
- the evidence is time-sensitive, contested, or too dense to summarize safely;
- implementation needs a stable decision or ordered ledger;
- rollout or validation has a real operational sequence.

Otherwise keep the result inline.

## Artifact Owners

Task-local artifacts live in the task bundle at `specs/<task>/` named by
[Project Structure](../../project-structure-and-module-organization.md).

| Artifact | Create when | Owns | Does not own |
| --- | --- | --- | --- |
| `spec.md` | The behavior delta or accepted decision is not already owned by a stable authoritative reference, or must cross an actor/session boundary. | Outcome, behavior delta, invariants, constraints, accepted risks, proof expectations. | Runtime implementation order or unchanged behavior already owned by cited OpenAPI, tests, code, mockups, or external contracts. |
| `design/overview.md` or focused `design/*` | Implementation would otherwise choose architecture, contract, data, failure, rollout, or package ownership. | Selected mechanism and ownership decisions. | Task progress. |
| `test-plan.md` | Proof spans meaningful scenarios or levels. | Scenario obligations, observables, proof levels, residual gaps. | Test implementation. |
| `tasks.md` (+ `tasks/` task files when [split](../phases/planning/ledger-contract.md#ledger-layout)) | Work has multiple acceptance units, dependencies or waves, crosses an actor/session boundary, or needs durable resume state. A single fixed unit may stay inline. | Executable order, planned waves, owners, evidence, progress, completion condition. Lifecycle state stays in the index even when detail moves to task files. | New product or design decisions. |
| `research/*.md` | Evidence must be reused, audited, or refreshed. | Findings, source limits, conflicts, decision impact. | Final task decisions. |
| `rollout.md` | Deployment, migration, backfill, compatibility, or rollback has a non-trivial sequence. | Operational order, gates, rollback/failback, observables. | Product scope. |
| `workflow-plan.md` | Cross-session or multi-lane coordination cannot be recovered from the main artifacts. | Current goal, phase, active artifacts, blockers, next action, surviving open decisions and fog. | Duplicate spec/design/task content; the decisions themselves, which stay with their phase owner. |

Split an artifact only when the split creates a real owner or makes review materially easier. Do not create a directory of one-line files.

## Minimal Status

When durable status is useful, use one field:

```text
status: draft | ready | blocked | done
```

- `draft`: still being authored or repaired.
- `ready`: the artifact has closed every decision it owns, and its next consumer can act without semantic invention. A ready `tasks.md` additionally closes the inputs and proof for its next executable acceptance unit or real parallel wave plus any decision that could invalidate that work.
- `blocked`: name the missing decision or evidence and reopen owner.
- `done`: use for execution/closeout state, not as a substitute for evidence. A
  `tasks.md` ledger reaches `done` only after the final accepted unit also
  passes its global `Completion` condition.

When review is triggered, `PASS` or dispositioned `CONCERNS` can move an artifact to `ready`; `FAIL` requires repair or reopening and fresh review.

For `tasks.md` receipts, blockers, and implementation state transitions, load
the [Planning Ledger Contract](../phases/planning/ledger-contract.md#implementation-transitions).

Add a reviewed revision or verdict only when a review actually occurred. Do not maintain parallel fields for phase state, artifact lifecycle, record validity, session boundary, handoff readiness, waiver, and routing revision unless a concrete external consumer requires them.

Status inspection is read-only: inspect the current workspace and Git drift, then report status, owner, evidence, next action, and readiness without changing files or status.

## Compact Shapes

A useful `spec.md` follows the canonical shape and adaptive authoring method in
[Specification](../phases/specification.md). This file owns artifact
persistence and status, not a second Specification template.

A useful `tasks.md` uses the canonical shape and execution contracts in
[Planning Ledger Contract](../phases/planning/ledger-contract.md), with readiness
owned by [Planning](../phases/planning.md). This
file owns whether and when `tasks.md` is persisted, its semantic lifecycle, and
removal; native task lifecycle and Git candidate identity remain with their own
systems.

A useful `workflow-plan.md` usually needs:

```markdown
# Goal
status: draft | ready | blocked | done
Current phase:
Active artifacts:
Blockers / assumptions:
Next action:
Completion proof:

## Open decisions      <!-- only when a question must survive its phase -->
## Not yet specified   <!-- only when fog is worth carrying forward -->
```

Use additional fields only when they change an action or verdict.

## Review And Freshness

- Identify the exact artifact revision or diff reviewed when that distinction matters.
- After a material repair, old findings remain history; the repaired surface needs a fresh check proportionate to the change.
- Time-sensitive external evidence carries the claim-level locator and freshness
  required by [Research](../phases/research.md#outputs).
- A stale artifact may explain history but cannot override a newer accepted decision or runtime source of truth.
- When a ready artifact's review predates a material change to its owning phase
  contract, recheck only the affected next action against the current Stop Rule
  and retain unaffected decisions and proof.

## Resume Order

1. Inspect current workspace and Git status, then read current `tasks.md` first when implementation or validation is active. When the ledger is split, read the index first and then only the task files for the next ready unit.
2. Otherwise read `workflow-plan.md` when it exists for a real multi-session task.
3. Then read the decision artifact named there: usually `spec.md`, followed by only the design, test, research, or rollout files needed for the next action.
4. If artifacts conflict, stop and reopen the narrowest decision owner; do not merge the conflict silently.
5. Before continuing Implementation, apply [Implementation
   Handoff](implementation-handoff.md) and its [Evidence
   Contract](../phases/implementation-validation-closeout.md#evidence-contract)
   to candidate and proof reuse.

Keep only active task bundles. At closeout, remove execution-only state such as
`tasks.md` with any `tasks/` directory and `workflow-plan.md`. Retain a completed spec or
design only when another live authority names it as a durable decision source;
otherwise delete the completed bundle after moving durable decisions into
canonical docs or code. A research note whose refresh trigger is still live
moves to the canonical owner of the decision it supports, or is deleted with the
bundle. Git remains the history, so a completed task ledger is never kept as an
archive.

Moving a decision completes only when it reads as a decision at its canonical
owner: the rule, the stable identity of the change that decided it — merged pull
request or commit — and the condition that would reopen it. Delete the bundle
only after that move. Provenance is for a rule a later reader could reverse by
accident; a rule whose statement already carries its own rationale and reopen
condition needs no link.
