# Direction Options And Convergence

## Behavior Change Thesis
When loaded for symptom "several product directions remain plausible or the raw idea is a grab bag," this file makes the model choose one defensible direction with rejected-option rationale instead of the likely mistake of preserving every option as a combined strategy.

## When To Load
Load this when brainstorming needs convergence, when the idea contains unrelated features, or when two or three directions could each plausibly be first.

## Decision Rubric
- Compare two or three genuinely different directions; do not list minor variants as options.
- Hold options against the same target user, problem, success signal, MVP size, and risk profile.
- Prefer the direction that tests the riskiest value assumption with the least irreversible scope.
- Explain why runner-up options are not first, not why they are bad forever.
- Keep alternatives only as follow-ons or switch triggers.

## Imitate
Input: "We need better onboarding: checklists, videos, templates, and AI setup help."

```markdown
Direction Options
1. Guided checklist: fastest path to reduce first-session confusion, but may become a task list that does not improve setup quality.
2. Template-first onboarding: strongest fit if users know their goal but struggle to start from a blank state.
3. AI setup helper: attractive for flexibility, but riskiest because value and trust are both unproven.

Recommended Direction
Start with template-first onboarding for one high-value setup path. It tests whether users can reach a meaningful first outcome faster without taking on conversational AI risk.
```

Copy the move: it compares options on the same bet and recommends one first move.

Input: "We could improve admin safety with warnings, approvals, audit logs, or auto-rollback."

```markdown
Recommended Direction
Start with pre-action comprehension for the riskiest admin operation: clearer target identification, consequence summary, and confirmation friction. Defer approvals, audit reporting, and rollback until the team validates whether mistakes happen before the action or during recovery.
```

Copy the move: it converges by choosing the earliest point in the failure chain.

## Reject
```markdown
Recommended Direction
Build checklists, videos, templates, and AI setup help because together they create a complete onboarding experience.
```

Reject this because it treats a grab bag as strategy and avoids choosing the smallest direction that can validate the core bet.

```markdown
Direction Options
1. Dashboard with AI.
2. AI-powered dashboard.
3. Dashboard plus summaries.
```

Reject this because the options are not materially different.

## Agent Traps
- Do not call a bundle "comprehensive" as a substitute for convergence.
- Do not choose the flashiest technology unless it is the smallest way to learn the core bet.
- Do not keep all options alive to avoid disappointing the user; the deliverable should reduce ambiguity.
- Do not reject runner-ups so hard that later spec work loses useful fallback triggers.
