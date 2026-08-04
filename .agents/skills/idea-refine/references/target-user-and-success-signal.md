# Target User And Success Signal

## Behavior Change Thesis
When loaded for symptom "the idea says users broadly, mixes buyer/operator audiences, or defines success as shipping," this file makes the model choose one primary actor and a behavior/outcome signal instead of accepting a vague audience and an output metric.

## When To Load
Load this when the actor, operator, job, or success signal is vague, plural, business-only, or feature-completion-shaped.

## The Move
Pick the one actor whose success determines MVP correctness — the operator inside the workflow, distinct from the buyer who requested the idea — and name the moment the pain appears: timing, trigger, and pressure. State success as behavior, outcome, or risk reduction inside the team's plausible span of influence, translating lagging business metrics into nearer product evidence. Measure a prevention feature by the risk shrinking rather than by recovery usage — a successful prevention flow reduces use of recovery. A numeric target needs supplied evidence or an explicit discovery-placeholder label.

## Imitate
Input: "Admins keep deleting the wrong environment. Maybe snapshots every 15 minutes plus undo."

```markdown
Target User / Operator
Workspace administrators deleting staging or production-like environments under time pressure.

Success Signal
Fewer irreversible wrong-environment deletions and fewer support-assisted restore requests. A useful early signal is that admins can identify the target environment and consequence before confirming deletion, without materially slowing routine safe deletions.
```

Copy the move: it separates operator safety from the proposed snapshot mechanism and ties success to a destructive-action moment.

## Reject
```markdown
Target User / Operator
All admins and developers.

Success Signal
Ship an undo button and restore endpoint.
```

The actor is too broad to determine correctness, and the signal is delivery output, not evidence that the risk got smaller.
