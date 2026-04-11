# Assumptions And Kill Criteria

## Behavior Change Thesis
When loaded for symptom "the idea depends on enthusiasm, AI magic, uncertain demand, risky behavior, or untested policy assumptions," this file makes the model choose falsifiable bets and kill/pivot criteria instead of the likely mistake of listing vague assumptions that cannot change the recommendation.

## When To Load
Load this when the recommendation would fail if a belief about user value, trust, feasibility, policy, operations, or business viability turns out false.

## Decision Rubric
- Write assumptions as "we believe..." claims that can be tested or contradicted.
- Prioritize assumptions that are both important to success and weakly evidenced.
- Separate desirability, usability, trust, feasibility, viability, policy, and operations only when the distinction changes the next move.
- Attach kill, shrink, or pivot criteria to assumptions that would change the recommendation.
- Prefer behavioral evidence over stated preference; label anecdote as weak evidence.

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

Input: "Customers will love auto-delete for stale test projects."

```markdown
Key Assumptions To Validate
1. We believe stale test projects create enough cost or clutter that admins want automatic cleanup.
   Risk: desirability.
   Kill criteria: if admins mostly tolerate stale projects or clean them manually during routine reviews, avoid auto-delete and refine a lighter review flow.

2. We believe the product can identify stale projects without deleting active work.
   Risk: trust and operations.
   Pivot criteria: if stale detection creates ambiguous cases, prefer a review queue over automatic deletion.
```

Copy the move: it turns a confident claim into reversible product bets.

## Reject
```markdown
Assumptions
- AI will save time.
- Users want automation.
- Release notes are important.
```

Reject this because the assumptions are vague, unranked, and impossible to disprove cleanly.

```markdown
Kill Criteria
- If the feature is not useful.
```

Reject this because it names no observable result and cannot guide whether to stop, shrink, or pivot.

## Agent Traps
- Do not bury the riskiest assumption under a long list of harmless unknowns.
- Do not treat "stakeholders asked for it" as evidence of user value.
- Do not use kill criteria punitively; they exist to protect focus before spec work hardens.
- Do not invent precise thresholds unless the user supplied a basis. A qualitative kill criterion is better than a fake metric.
