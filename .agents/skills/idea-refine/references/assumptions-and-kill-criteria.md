# Assumptions And Kill Criteria

## When To Load
Load this when the idea depends on enthusiasm, AI magic, uncertain demand, risky operator behavior, compliance or policy assumptions, or a claim that users "will want" something without evidence.

## Use This Lens
- Convert hidden beliefs into "we believe..." assumptions.
- Separate desirability, viability, feasibility, and business or stakeholder risks when useful.
- Prioritize assumptions that are important to success and weakly evidenced.
- Add kill or pivot criteria for assumptions that would change the recommendation.

## Good Idea-Refine Output
Input: "Let's add AI-generated release notes for every deploy."

Good:

```markdown
Key Assumptions To Validate
1. We believe service owners lose meaningful time writing release notes, not merely polishing them.
   Risk: desirability.
   Weak evidence: anecdotal.
   Kill criteria: if 5 recent service owners say release-note drafting is not a top deploy pain, do not build an AI notes MVP.

2. We believe generated notes can be trusted after lightweight owner review.
   Risk: value and trust.
   Weak evidence: no sample review yet.
   Kill criteria: if reviewers must rewrite most generated notes in a sample of recent deploys, switch to a structured note template instead of generation.
```

Why it works: The assumptions are specific enough to test and strong enough to change the direction if false.

## Bad Idea-Refine Output
```markdown
Assumptions
- AI will save time.
- Users want automation.
- Release notes are important.
```

Why it fails: The assumptions are vague, unranked, and impossible to disprove cleanly.

## Convergence Example
Use risk to choose between directions:

```markdown
Option A: Generate full release notes.
Riskiest assumption: generated content is trusted enough to publish.

Option B: Draft a structured note from deploy metadata and owner-selected changes.
Riskiest assumption: metadata covers the facts owners need.

Recommendation
Choose Option B because it validates the workflow and value assumption while reducing trust risk. Move to full generation only if structured drafts prove valuable and owners still spend too much time on wording.
```

## Weak-Assumption Challenges
- What must be true for this idea to work?
- Which assumption would sink the recommendation if false?
- Is there behavioral evidence, or only stated preference?
- Is the riskiest assumption about customer value, usability, feasibility, business viability, policy, trust, or operations?
- What result would make us stop, shrink, or change direction before spec work?

## Exa Source Links
- [Precoil: How To Map Risk With A Mission Model Canvas](https://www.precoil.com/articles/how-to-map-risk) explains extracting assumptions, mapping importance against evidence, and testing the riskiest items.
- [David J. Bland: About](https://davidjbland.com/about/) describes assumption mapping across desirability, viability, and feasibility with evidence-based commit, correct, or cut decisions.
- [SVPG: Planning Product Discovery](https://svpg.com/planning-product-discovery) lists product discovery risks beyond technology, including value and stakeholder risks.
- [Lean Startup Co.: What Is an MVP?](https://leanstartup.co/resources/articles/what-is-an-mvp/) frames MVPs as validated learning with least effort, not minimal product theater.
- [Product Talk: Opportunity Solution Trees](https://www.producttalk.org/opportunity-solution-trees/) places assumption tests under candidate solutions.
