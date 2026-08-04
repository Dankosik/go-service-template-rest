# Clarifying Questions And Safe Assumptions

## Behavior Change Thesis
When loaded for symptom "the idea is underspecified but not truly blocked," this file makes the model choose one or two gating questions plus labeled safe assumptions instead of turning idea refinement into a long discovery questionnaire or inventing certainty.

## When To Load
Load this when the prompt is thin, the model is tempted to ask many questions, or a small assumption would allow a useful first refinement.

## The Move
Ask only a question whose answer could change the recommended direction, target actor, or MVP boundary; cover ordinary missing detail with labeled safe assumptions the user can correct, and keep refining after either — the user receives progress, never a homework list. When the missing fact is high-impact policy, money movement, identity, privacy, safety, destructive action, or irreversible state, ask instead of assuming; implementation questions wait until the product direction is chosen.

## Imitate
Input: "Make onboarding better."

```markdown
I can refine this with one assumption: onboarding means a new workspace admin trying to reach their first useful project, not enterprise procurement or team training.

Problem
New workspace admins can reach the product but do not reliably know the first concrete setup path that produces value.

Open Question
Which first outcome matters most: inviting a team, creating the first project, connecting data, or completing a deployment?
```

Copy the move: it makes progress while naming the one assumption most likely to need correction.

Input: "Let's automate cleanup of stale environments."

```markdown
Before choosing a direction, one question matters: would cleanup be allowed to delete resources automatically, or must a human approve destructive actions?
```

Copy the move: destructive behavior is high-impact enough to ask before recommending an MVP.

## Reject
```markdown
I will assume automatic deletion is safe and proceed with an auto-cleanup recommendation.
```

Destructive behavior is never a safe assumption.
