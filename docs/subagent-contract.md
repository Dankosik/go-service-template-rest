# Subagent Contract

Subagents answer narrow questions. The root owns task scope, synthesis, edits, and final claims.

## Use

Delegate only when the question is concrete, bounded, independent of mutable work in other lanes, and materially improved by separate context or review independence. Keep sequential, tiny, or tightly coupled work local.

Default to at most three concurrent lanes and no nested delegation. This is a concurrency limit, not a cap on justified sequential lanes or review iterations.

## Boundaries

- Research, challenge, and review lanes are read-only.
- An implementation worker may write only inside its explicit task/workspace boundary.
- A lane does not broaden scope, invent missing policy, edit workflow authority, approve its own repair, or claim task completion.
- Missing input returns to the root with the smallest useful question or reopen owner.

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

Every review return also names the exact revision or diff and accounts for each materially affected lens as covered, delegated to a named specialist result, or not triggered with a concrete reason. A required gate includes one whole-artifact or whole-diff coherence pass. One reviewer may cover compatible lenses; do not create duplicate lanes merely to increase the count.

## Return

Return a compact answer:

- conclusion;
- strongest evidence;
- uncertainty/conflict;
- recommended root disposition or reopen owner.

Raw lane transcripts are not authoritative artifacts. The root verifies and reconciles the result.
