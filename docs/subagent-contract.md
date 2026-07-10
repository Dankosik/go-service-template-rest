# Subagent Contract

Subagents answer narrow questions. The root owns task scope, synthesis, edits, and final claims.

## Use

Delegate only when the question is concrete, bounded, independent of mutable work in other lanes, and materially improved by separate context or review independence. Keep sequential, tiny, or tightly coupled work local.

Default to at most three concurrent lanes and no nested delegation.

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

## Return

Return a compact answer:

- conclusion;
- strongest evidence;
- uncertainty/conflict;
- recommended root disposition or reopen owner.

Raw lane transcripts are not authoritative artifacts. The root verifies and reconciles the result.
