# Assumptions And Kill Criteria

## Behavior Change Thesis
When loaded for symptom "the idea depends on enthusiasm, AI magic, uncertain demand, risky behavior, or untested policy assumptions," this file makes the model choose falsifiable bets and kill/pivot criteria instead of listing vague assumptions that cannot change the recommendation.

## When To Load
Load this when the recommendation would fail if a belief about user value, trust, feasibility, policy, operations, or business viability turns out false.

## The Move
Write each assumption as a "we believe..." claim specific enough to be contradicted, and lead with the assumption that is both most important to success and most weakly evidenced. Attach kill, shrink, or pivot criteria to the assumptions that would change the recommendation, naming the observable result that triggers them — a qualitative criterion beats an invented threshold, and a user-supplied basis is what earns a number. Weigh behavioral evidence over stated preference, and label anecdote — including "stakeholders asked for it" — as weak evidence.

## Imitate
Input: "Let's add AI-generated release notes for every deploy."

```markdown
Key Assumptions To Validate
1. We believe service owners lose meaningful time drafting release notes, not merely polishing them.
   Risk: desirability.
   Weak evidence: anecdotal.
   Kill criteria: if recent service owners do not name release-note drafting as a meaningful deploy pain, do not build an AI-notes MVP.

2. We believe generated notes can be trusted after lightweight owner review.
   Risk: value and trust.
   Weak evidence: no sample review yet.
   Pivot criteria: if reviewers must rewrite most generated notes from recent deploys, switch to a structured note template instead of generation.
```

Copy the move: each assumption is specific enough to be wrong and strong enough to affect the direction.

## Reject
```markdown
Kill Criteria
- If the feature is not useful.
```

It names no observable result, so it cannot guide whether to stop, shrink, or pivot.
