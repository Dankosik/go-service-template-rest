# Subagent Contract

Subagents answer narrow questions. The root owns task scope, synthesis, integration, task acceptance, and final claims.

## Use

Delegate only when the question is concrete, bounded, independent of mutable work in other lanes, and materially improved by separate context or review independence. Keep sequential, tiny, or tightly coupled work local.

Default to at most three concurrent research/review lanes, one bounded wave, and no nested delegation. Additional sequential research/review lanes require distinct decision-changing questions that could not be covered locally or in the first wave. The one-at-a-time implementation task loop is separate from that lane limit.

## Boundaries

- Research, challenge, and review lanes are read-only.
- An implementation worker owns exactly one assigned task and may write only inside that task/workspace boundary.
- The worker returns its diff, acceptance-criteria mapping, proof, and blockers to the root; it does not mark the task complete, start the next task, or launch a reviewer.
- The root either accepts the task or returns concrete gaps to the same worker. Until acceptance, the root does not advance the ledger or repair the assigned task itself.
- After acceptance, the next task goes to a fresh worker; the previous task worker is reused only for corrections to its own task.
- A lane does not broaden scope, invent missing policy, edit workflow authority, approve its own repair, or claim task completion.
- Missing input returns to the root with the smallest useful question or reopen owner.
- Internal macro-phase grilling follows the canonical [Autonomous Pre-Review Challenge](spec-first-workflow/shared/subagents-and-handoff.md#autonomous-pre-review-challenge): the challenger returns one protocol event and no verdict, while the root records decisions and continues the same lane.

## Evidence

Inspect the named artifacts and sources before concluding. Keep facts, inferences, conflicts, and missing proof distinct. Cite tight file/line, artifact, command, or external-source anchors.

## Shared Review Finding Envelope

Lead with findings in severity order. Each actionable finding contains:

```text
Title
Anchor/evidence
Impact on the accepted outcome
Classification: blocker | bounded concern/proof obligation | non-blocking
Recommended action and owner
```

If no finding survives, say so and state the evidence boundary. Do not pad a clean review.

Every review return also names the exact revision or diff and accounts for each materially affected lens as covered, supplied by one concrete specialist result, or not triggered with a concrete reason. The default gate is one whole-artifact or whole-diff reviewer applying compatible methods locally in one coherence pass. A skill handoff is advisory evidence for the root, not an automatic lane; do not create duplicate reviewers merely to increase confidence.

## Return

Return a compact answer:

- conclusion;
- strongest evidence;
- uncertainty/conflict;
- recommended root disposition or reopen owner.

Raw lane transcripts are not authoritative artifacts. The root verifies and reconciles the result.
