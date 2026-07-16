# Autonomous Pre-Review Challenge

Focused owner for the autonomous read-only challenge that precedes required review at the workflow boundaries routed here.

## Read When

- A structured or orchestrated non-implementation candidate has met its authoring bar and the router requires a pre-review challenge.
- The root is continuing, resuming, or invalidating that challenge before the separate required reviewer.

## Inputs

- Current phase method and exact latest candidate revision.
- Accepted constraints, evidence boundary, authority boundary, and stop rule.
- Any named open items and their current owning-candidate dispositions.

## Outputs

- Exactly one protocol event from the challenger on each turn.
- Root dispositions recorded in the owning candidate rather than a parallel challenge artifact.
- A completed challenge boundary that remains separate from the required review verdict.

## Protocol

For each candidate routed here by the workflow, the root launches the existing
read-only challenger in internal grilling mode after the candidate meets its
authoring bar. Supply the current phase method, exact candidate revision,
accepted constraints, evidence boundary, authority boundary, and stop rule.
Explicit user-requested grilling remains a root-to-user dialogue; the internal
challenger never relays those questions.

The challenger inspects repository facts rather than asking for them, then
selects the highest-impact unresolved material branch. That branch owns the exchange until the root records `ACCEPT`, `OVERRIDE`, `RECLASSIFY`, `WAIT_HUMAN`, or `REOPEN_OWNER`; only then may the challenger select a sibling branch. Emitting `QUESTION` does not close the branch. The challenger may apply a materially triggered specialist method locally but never delegates recursively. Each turn
returns exactly one event: `QUESTION`, `HUMAN_REQUIRED`, `REOPEN`, or `DONE`.
Do not emit a questionnaire or a readiness verdict.

```text
QUESTION
Changes: <one current-phase decision>
Question: <one root-answerable question>
Recommended: <recommended answer>
Tradeoff: <main cost or risk>
Evidence: <artifact or repository anchor, or bounded assumption>

HUMAN_REQUIRED
Decision: <user-owned decision>
Authority reason: <why the root cannot decide>
Recommended: <recommended option when evidence supports one>
Tradeoff: <main cost or risk>
Dependency impact: <independent continuation or WAIT_HUMAN>

REOPEN
Owner: <evidence or upstream owner>
Gap or conflict: <missing evidence or contradicted decision>
Impact: <choice or readiness that cannot close>
Next evidence or repair: <smallest resolution route>

DONE
Resolved: <material decisions dispositioned in the latest candidate>
Assumptions: <bounded assumptions or none>
Residual risks: <owned risks or none>
Reopen when: <objective invalidation condition or none>
```

## Authority

The root decides mechanism, system/package/file/instruction ownership, proof
strategy, and task order inside accepted behavior and authority. Undecided user
intent, observable behavior or scope, policy, new authority or external action,
and user-owned material risk acceptance return `HUMAN_REQUIRED`. Missing facts,
conflicting evidence, or an upstream decision gap return `REOPEN`.

## State And Continuation

For `QUESTION`, the root verifies the evidence and responds with `ACCEPT`, `OVERRIDE`, or `RECLASSIFY`. Record the selected decision or corrected owner,
strongest basis, destination in the owning candidate, bounded assumption, reopen
condition, and exact latest revision before the next turn. `RECLASSIFY` prevents
the root from answering for the user or hiding an upstream gap.

For `HUMAN_REQUIRED` or `REOPEN`, record the deduplicated item in the existing
owner, then respond with `CONTINUE_INDEPENDENT`, `WAIT_HUMAN`, or `REOPEN_OWNER`,
plus its destination, exact latest revision, and relevant open items.
`CONTINUE_INDEPENDENT` permits only work outside the unresolved challenger branch;
it does not close that branch or permit sibling-branch selection. The other
responses wait for the human answer or owner repair.

Continue dependent turns through the same challenger with the exact latest candidate.
The owning candidate is authoritative; the child transcript is not. If the
runtime cannot resume that child, relaunch the existing challenger from the
exact latest candidate and named open items rather than remembered chat. Do not create a probe transcript, receipt, queue, status, lifecycle field, or review verdict.

## Exhaustion And Invalidation

There is no question quota. `DONE` means no new, evidence-reopened, or readiness-changing current-phase decision remains. A dependent blocked branch retains `HUMAN_REQUIRED` or `REOPEN` until its owner resolves it. Repeated dispositions, generic
category coverage, and questions with no affected choice are no progress.
Wording-only edits and repairs that apply an existing disposition reuse
completion. New evidence or a material change to a decision, assumption,
authority boundary, source of truth, or upstream dependency requires a fresh
probe; uncertain resume state also reruns it.

## Reviewer Separation

After `DONE`, a different read-only child reviews the exact latest candidate
under the owning phase's existing review method. The challenger never supplies
that verdict or replaces the required reviewer.

## Stop Rule

Stop the challenge on `DONE`; wait or reopen the named owner on a dependent `HUMAN_REQUIRED` or `REOPEN`. Continue only independent branches explicitly permitted by the recorded root disposition. Phase movement remains governed by the router and the separate required review.
