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

| Artifact | Create when | Owns | Does not own |
| --- | --- | --- | --- |
| `spec.md` | Required for structured/orchestrated work; direct work uses it only when decisions must survive. | Outcome, behavior delta, invariants, constraints, accepted risks, proof expectations. | Runtime implementation order. |
| `design/overview.md` or focused `design/*` | Implementation would otherwise choose architecture, contract, data, failure, rollout, or package ownership. | Selected mechanism and ownership decisions. | Task progress. |
| `test-plan.md` | Proof spans meaningful scenarios or levels. | Scenario obligations, observables, proof levels, residual gaps. | Test implementation. |
| `tasks.md` | Required for structured/orchestrated work; direct work may keep its plan inline. | Executable order, planned waves, owners, evidence, progress, completion condition. | New product or design decisions. |
| `research/*.md` | Evidence must be reused, audited, or refreshed. | Findings, source limits, conflicts, decision impact. | Final task decisions. |
| `rollout.md` | Deployment, migration, backfill, compatibility, or rollback has a non-trivial sequence. | Operational order, gates, rollback/failback, observables. | Product scope. |
| `workflow-plan.md` | Cross-session or multi-lane coordination cannot be recovered from the main artifacts. | Current goal, phase, active artifacts, blockers, next action. | Duplicate spec/design/task content. |

Split an artifact only when the split creates a real owner or makes review materially easier. Do not create a directory of one-line files.

## Minimal Status

When durable status is useful, use one field:

```text
status: draft | ready | blocked | done
```

- `draft`: still being authored or repaired.
- `ready`: the artifact has closed every decision it owns, and its next consumer can act without semantic invention. A ready `tasks.md` additionally closes the inputs and proof for its next executable task or real parallel wave plus any decision that could invalidate that work.
- `blocked`: name the missing decision/evidence and reopen owner.
- `done`: use for execution/closeout state, not as a substitute for evidence.

When review is triggered, `PASS` or dispositioned `CONCERNS` can move an artifact to `ready`; `FAIL` requires repair or reopening and fresh review.

Add a reviewed revision or verdict only when a review actually occurred. Do not maintain parallel fields for phase state, artifact lifecycle, record validity, session boundary, handoff readiness, waiver, and routing revision unless a concrete external consumer requires them.

Status inspection is read-only: inspect the current workspace and Git drift, then report status, owner, evidence, next action, and readiness without changing files or status.

## Compact Shapes

A useful `spec.md` follows the canonical shape and adaptive authoring method in
[Specification](../phases/specification.md). This file owns artifact
persistence and status, not a second Specification template.

A useful `tasks.md` usually needs:

```markdown
Global constraints: <exact constraints shared by multiple tasks; omit when none>
- [ ] ID: outcome
  - Source: <narrow anchors>
  - Owner/surface/resources: <writable owner; execution-exclusive resources or none>
  - Depends on: <IDs or none>
  - Handoff: <exact consumes/produces contract; omit when none>
  - Proof: <claim; check; expected observable>
  - Reopen if: <objective invalidation condition; omit when none>
```

Add `Planned waves` only when at least two ready tasks will actually be dispatched concurrently; dependency order is sufficient for sequential work.

When an active wave must survive compaction, interruption, or session handoff, add one compact `Active wave` block to this same ledger with the adjusted member IDs, accepted integration base, task-to-App-task/worktree state, disposable candidate identity when one exists, and next root action or open causal class. Update it only at a material transition and remove or collapse it into task evidence after atomic wave acceptance; do not create a scheduler file or reconstruct it from chat.

A useful `workflow-plan.md` usually needs:

```markdown
# Goal
status: draft | ready | blocked | done
Current phase:
Active artifacts:
Blockers / assumptions:
Next action:
Completion proof:
```

Use additional fields only when they change an action or verdict.

## Review And Freshness

- Identify the exact artifact revision or diff reviewed when that distinction matters.
- After a material repair, old findings remain history; the repaired surface needs a fresh check proportionate to the change.
- Time-sensitive external evidence records its source and date/version.
- A stale artifact may explain history but cannot override a newer accepted decision or runtime source of truth.

## Resume Order

1. Inspect current workspace and Git status, then read current `tasks.md` first when implementation or validation is active.
2. Otherwise read `workflow-plan.md` when it exists for a real multi-session task.
3. Then read the decision artifact named there: usually `spec.md`, followed by only the design, test, research, or rollout files needed for the next action.
4. If artifacts conflict, stop and reopen the narrowest decision owner; do not merge the conflict silently.
5. Before continuing an implementation task, rerun the smallest ledger proof that can detect workspace drift affecting the next unchecked task; broaden only when the result or changed surface requires it.

Keep only active task bundles. At closeout, remove execution-only state such as
`tasks.md`, `workflow-plan.md`, and `Active wave`. Retain a completed spec or
design only when another live authority names it as a durable decision source;
otherwise delete the completed bundle after moving durable decisions into
canonical docs or code. Git remains the history, so a completed task ledger is
never kept as an archive.
