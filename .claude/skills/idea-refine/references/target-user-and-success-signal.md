# Target User And Success Signal

## Behavior Change Thesis
When loaded for symptom "the idea says users broadly, mixes buyer/operator audiences, or defines success as shipping," this file makes the model choose one primary actor and a behavior/outcome signal instead of the likely mistake of accepting a vague audience and output metric.

## When To Load
Load this when the actor, operator, job, or success signal is vague, plural, business-only, or feature-completion-shaped.

## Decision Rubric
- Pick the primary target user or operator for the MVP, even if other audiences matter later.
- Name the situation where the pain appears: timing, trigger, workflow, and pressure.
- Separate buyer, approver, operator, affected user, and support role when they could pull scope in different directions.
- Prefer behavior, outcome, risk reduction, or value-creation signals over "ship X."
- Keep success signals inside the team's plausible span of influence; translate lagging business metrics into nearer product evidence.

## Imitate
Input: "Admins keep deleting the wrong environment. Maybe snapshots every 15 minutes plus undo."

```markdown
Target User / Operator
Workspace administrators deleting staging or production-like environments under time pressure.

Success Signal
Fewer irreversible wrong-environment deletions and fewer support-assisted restore requests. A useful early signal is that admins can identify the target environment and consequence before confirming deletion, without materially slowing routine safe deletions.
```

Copy the move: it separates operator safety from the proposed snapshot mechanism and ties success to a destructive-action moment.

Input: "Managers need a compliance dashboard."

```markdown
Target User / Operator
Compliance owner preparing an evidence review before an external audit, not every engineering manager.

Success Signal
The compliance owner can identify missing evidence for one audit area before the review meeting, reducing last-minute evidence chasing.
```

Copy the move: it narrows the audience to the role whose job determines MVP correctness.

## Reject
```markdown
Target User / Operator
All admins and developers.

Success Signal
Ship an undo button and restore endpoint.
```

Reject this because the actor is too broad and the signal is delivery output, not evidence that the risk got smaller.

```markdown
Success Signal
Increase retention.
```

Reject this unless the pass also names the nearer behavior that would plausibly affect retention; otherwise the signal is too lagging to guide refinement.

## Agent Traps
- Do not keep "all users" because several groups are affected. Choose the one whose pain determines the first direction.
- Do not confuse the person who buys or requests the idea with the person who must succeed in the workflow.
- Do not measure a prevention feature by how often users need recovery; a successful prevention flow may reduce use of recovery.
- Do not invent numeric targets unless the user supplied evidence or the number is clearly labeled as a discovery placeholder.
