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
| `workflow-plan.md` | Cross-session or multi-lane coordination cannot be recovered from the main artifacts. | Current goal, phase, active artifacts, blockers, next action, surviving open decisions and fog. | Duplicate spec/design/task content; the decisions themselves, which stay with their phase owner. |

Split an artifact only when the split creates a real owner or makes review materially easier. Do not create a directory of one-line files.

## Open Decisions And Fog

An **open decision** is a question you can already state precisely: what it can
change, who owns it, and what it blocks. Record one line per open decision in
`workflow-plan.md` at a real handoff, context rollover, or second-lane dispatch,
never in anticipation of one; a phase that finishes in its own session writes
none. An open decision that stops the next action belongs to the `blocked`
status and its `Blockers / assumptions` entry instead. The **frontier** is every
listed decision whose blockers are resolved — what can be worked or dispatched
now, and the surviving form of the question map the [Delegation
Decision](subagents-and-handoff.md#delegation-decision) builds at phase entry.

```markdown
- <question, stated as the decision it can change> — owner: <agent, named external owner, or user> — blocks: <decision, phase, or task; or nothing> — route: <research lane, design owner, probe, or escalation>
```

**Fog** is a decision surface you can see coming but cannot yet phrase
precisely. The test is whether you can state the question precisely now, not
whether you can answer it: an already sharp question is an open decision even
when it is blocked and unworkable. Each `Not yet specified` entry carries both
parts, and an entry that cannot name its second part is deleted rather than
carried:

```markdown
- <suspected area> — sharpens when: <the open decision that resolves it, or the evidence that would let you phrase it>
```

Fog never appears in a readiness or completion claim, and never closes, defers,
or softens a decision the current phase's decision bar has triggered. When the
decision an entry names resolves, that entry graduates into an open decision or
is deleted in the same edit. Work already ruled beyond the accepted outcome
follows the scope-exit record in
[Planning](../phases/planning.md#obligation-reconciliation-contract).

## Minimal Status

When durable status is useful, use one field:

```text
status: draft | ready | blocked | done
```

- `draft`: still being authored or repaired.
- `ready`: the artifact has closed every decision it owns, and its next consumer can act without semantic invention. A ready `tasks.md` additionally closes the inputs and proof for its next executable acceptance unit or real parallel wave plus any decision that could invalidate that work.
- `blocked`: name the missing decision/evidence and reopen owner. This also
  represents `implementation complete; verification incomplete` when the
  implementation is finished but required proof is unavailable; record the
  unverified claim, narrower evidence, and next proof or reopen owner.
- `done`: use for execution/closeout state, not as a substitute for evidence. A
  `tasks.md` ledger reaches `done` only after the final accepted unit also
  passes its global `Completion` condition.

When review is triggered, `PASS` or dispositioned `CONCERNS` can move an artifact to `ready`; `FAIL` requires repair or reopening and fresh review.

For `tasks.md`, the implementation phase's [Acceptance-Unit
Closure](../phases/implementation-validation-closeout.md#acceptance-unit-closure)
authorizes the state transition. Record one compact unit receipt only when
proof must survive a checkout, session, or external-environment boundary:

```markdown
  - Accepted: <unit or task IDs>; evidence: <command or source and result>; candidate: <bounded diff or commit/tree>
```

Use `current bounded diff` for same-checkout proof and a commit/tree identity
only when proof crosses a checkout or integration boundary. A failed triggered
review leaves the unit unchecked for repair. When implementation is complete
but required proof is blocked, leave it unchecked, set the ledger to `status:
blocked`, and append or replace one unit-local line:

```markdown
  - Blocked: <unit or task IDs>; unverified: <claim>; evidence: <narrower evidence>; next proof owner: <owner and condition>; candidate: <bounded diff or commit/tree>
```

Replace that line with the accepted receipt after proof instead of accumulating
attempts. Do not add a second lifecycle field.

The accepted-unit transition changes every member task to `[x]` in one ledger
edit. A receipt alias closes mechanically in that edit once its named accepted
receipt exists; it creates no candidate or proof. Task selection, dependency
movement, and resume follow the phase-owned closure using the persisted
checkboxes. After the final accepted task and any aliases, set `status: done` in
the same edit.

Add a reviewed revision or verdict only when a review actually occurred. Do not maintain parallel fields for phase state, artifact lifecycle, record validity, session boundary, handoff readiness, waiver, and routing revision unless a concrete external consumer requires them.

Status inspection is read-only: inspect the current workspace and Git drift, then report status, owner, evidence, next action, and readiness without changing files or status.

## Compact Shapes

A useful `spec.md` follows the canonical shape and adaptive authoring method in
[Specification](../phases/specification.md). This file owns artifact
persistence and status, not a second Specification template.

A useful `tasks.md` uses the canonical shape, authoring rules, planned-wave
contract, and readiness criterion in [Planning](../phases/planning.md). This
file owns whether and when `tasks.md` is persisted, its lifecycle status,
active-wave resume state, and removal; it does not define a second ledger
schema.

When an active wave must survive compaction, interruption, or session handoff, add one compact `Active wave` block to this same ledger with the acceptance-unit IDs, accepted integration base, unit-to-App-task/worktree state, frozen candidate identity when one exists, and next root action or open causal class. Update it only at a material transition and remove or collapse it into task evidence after atomic unit acceptance; do not create a scheduler file or reconstruct it from chat.

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
- Time-sensitive external evidence records its source and date/version.
- A stale artifact may explain history but cannot override a newer accepted decision or runtime source of truth.

## Resume Order

1. Inspect current workspace and Git status, then read current `tasks.md` first when implementation or validation is active.
2. Otherwise read `workflow-plan.md` when it exists for a real multi-session task.
3. Then read the decision artifact named there: usually `spec.md`, followed by only the design, test, research, or rollout files needed for the next action.
4. If artifacts conflict, stop and reopen the narrowest decision owner; do not merge the conflict silently.
5. Before continuing an implementation unit, compare its recorded tree and proof
   preconditions with the workspace. Reuse the receipt when they match; when they
   drift or the receipt is unavailable, run the smallest ledger proof that can
   detect the affected change and broaden only when its result requires it.

Keep only active task bundles. At closeout, remove execution-only state such as
`tasks.md`, `workflow-plan.md`, and `Active wave`. Retain a completed spec or
design only when another live authority names it as a durable decision source;
otherwise delete the completed bundle after moving durable decisions into
canonical docs or code. Git remains the history, so a completed task ledger is
never kept as an archive.

Moving a decision completes only when it reads as a decision at its canonical
owner: the rule, the stable identity of the change that decided it — merged pull
request or commit — and the condition that would reopen it. Delete the bundle
only after that move. Provenance is for a rule a later reader could reverse by
accident; a rule whose statement already carries its own rationale and reopen
condition needs no link.
