# Problem Vs Solution Framing

## When To Load
Load this when the user starts with a proposed solution, feature bundle, vendor/tool idea, or implementation mechanism and the real user problem is not yet explicit.

## Use This Lens
- Translate the proposal into the customer or operator situation that makes it worth discussing.
- Test whether the "problem" is really a solution in disguise by asking whether more than one solution could address it.
- Prefer one concrete story or current workaround over abstract market language.
- Keep the output upstream of architecture. The goal is a product problem frame, not a build design.

## Good Idea-Refine Output
Input: "We need an AI incident cockpit with Slack summaries, timelines, and service maps."

Good:

```markdown
Problem
On-call engineers lose the first 10-15 minutes of a severe incident reconstructing what changed, which alerts matter, and who has current context.

Target User / Operator
Primary on-call engineer during the first triage window.

Recommended Direction
Start with a first-10-minutes incident triage view that assembles alerts, recent deploys, owner hints, and a human-editable hypothesis note. Keep Slack output as a later distribution channel, not the core bet.
```

Why it works: The output names the operator, current breakdown, and a target opportunity before committing to the user's proposed feature bundle.

## Bad Idea-Refine Output
```markdown
Recommended Direction
Build an AI incident cockpit with Slack summaries, timeline reconstruction, and service maps.

MVP Scope
Implement all three integrations and use AI to summarize the incident.
```

Why it fails: It repeats the proposed implementation, never proves the problem, and turns the refinement pass into a delivery plan.

## Convergence Example
If the input contains three solutions, converge by asking what concrete breakdown they all try to fix.

```markdown
Options Considered
1. Slack incident digest: good for broadcast, weak for first diagnosis.
2. Service dependency map: useful when ownership is unclear, but may not explain what changed.
3. First-triage incident timeline: strongest fit for the stated pain because it reduces reconstruction time before the team decides on next action.

Recommendation
Choose the first-triage incident timeline. Treat Slack digest and service maps as follow-on options if discovery shows broadcast or ownership is the dominant pain.
```

## Weak-Assumption Challenges
- What is one recent incident story where the current workflow broke down?
- Is the pain "summarize everything" or "identify the first credible hypothesis faster"?
- Could a non-AI timeline or better alert grouping solve the same opportunity?
- Which proposed feature would still matter if Slack were out of scope?
- What baseline workaround are operators using today?

## Exa Source Links
- [Product Talk: Opportunity Solution Trees](https://www.producttalk.org/opportunity-solution-trees/) frames outcomes, opportunities, solutions, and assumption tests, and distinguishes opportunities from solutions.
- [SVPG: Planning Product Discovery](https://svpg.com/planning-product-discovery) emphasizes agreeing on the specific problem, target user or customer, success signal, and risks before discovery work.
- [Shape Up: Set Boundaries](https://basecamp.com/shapeup/1.2-chapter-03) shows how raw ideas need appetite and a narrowed problem before shaping.
- [HBS: Know Your Customers' Jobs to Be Done](https://hbs.edu/faculty/Pages/item.aspx?num=51553) calibrates problem framing around what customers are trying to accomplish in context.
