# Clarifying Questions And Safe Assumptions

## Behavior Change Thesis
When loaded for symptom "the idea is underspecified but not truly blocked," this file makes the model choose one or two gating questions plus labeled safe assumptions instead of the likely mistake of turning idea refinement into a long discovery questionnaire or inventing certainty.

## When To Load
Load this when the prompt is thin, the model is tempted to ask many questions, or a small assumption would allow a useful first refinement.

## Decision Rubric
- Ask only questions whose answers could change the recommended direction, target actor, or MVP boundary.
- Use safe assumptions for ordinary missing detail; label them so the user can correct them.
- If the missing fact is high-impact policy, money movement, identity, privacy, safety, destructive action, or irreversible state, ask instead of assuming.
- Prefer "I will assume X for now" over stalling when the assumption is reversible.
- Continue refining after the question or assumption; do not hand the user a homework list.

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
Before I can help, please answer:
1. Who are the users?
2. What are their goals?
3. What is the business objective?
4. What is the budget?
5. What is the timeline?
6. What integrations do you need?
```

Reject this because it exports the work to the user and asks questions that may not change the first refinement.

```markdown
I will assume automatic deletion is safe and proceed with an auto-cleanup recommendation.
```

Reject this because destructive behavior is not a safe assumption.

## Agent Traps
- Do not ask questions to protect yourself from making any recommendation.
- Do not turn every unknown into a blocker; idea refinement often starts with partial information.
- Do not use assumptions to skip high-impact policy or safety decisions.
- Do not ask implementation questions before the product direction is chosen.
