# Subagent Contract

Built-in subagents answer narrow read-only research, challenge, and review questions outside implementation/validation/closeout. The root owns task scope, synthesis, integration, task acceptance, and final claims. External implementation Workers are separate `codex exec` processes governed by the [implementation phase](spec-first-workflow/phases/implementation-validation-closeout.md#cli-worker-launch-and-resume), not by this contract. An explicitly requested independent review of completed implementation is a separate read-only boundary after that macro phase.

## Use

Delegate only when the question is concrete, bounded, independent of mutable work in other lanes, and materially improved by separate context or review independence. Keep sequential, tiny, or tightly coupled work local.

Default to at most three concurrent lanes, one bounded wave, and no nested delegation. Additional sequential lanes require distinct decision-changing questions that could not be covered locally or in the first wave.

## Boundaries

- Every subagent lane is read-only and limited to research, challenge, or independent review; it never implements or repairs code, config, docs, or tests.
- No built-in subagent lane runs inside implementation/validation/closeout for acceptance, final-diff review, specialist analysis, or re-review.
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

Every review return also names the exact revision or diff and accounts for each materially affected lens as covered, supplied by one concrete specialist result, or not triggered with a concrete reason. A non-implementation gate uses one whole-artifact reviewer applying compatible methods locally in one coherence pass. A standalone review may inspect a fixed completed-implementation diff, but it is not part of implementation acceptance or closeout. A skill handoff is advisory evidence for the root, not an automatic lane; do not create duplicate reviewers merely to increase confidence.

## Return

Return a compact answer:

- conclusion;
- strongest evidence;
- uncertainty/conflict;
- recommended root disposition or reopen owner.

Raw lane transcripts are not authoritative artifacts. The root verifies and reconciles the result.
