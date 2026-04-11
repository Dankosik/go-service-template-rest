# Target User And Success Signal

## When To Load
Load this when the idea has no clear actor, says "users" generically, confuses buyer and operator, or measures success by shipping a feature instead of changing behavior or outcomes.

## Use This Lens
- Pick the primary target user or operator for the MVP, even if other audiences exist.
- Name the situation, job, or operational moment where the pain appears.
- Prefer behavior, outcome, or value-creation signals over output completion.
- Keep success signals inside the team's likely span of influence where possible.

## Good Idea-Refine Output
Input: "Admins keep deleting the wrong environment. Maybe snapshots every 15 minutes plus undo."

Good:

```markdown
Target User / Operator
Workspace administrators deleting test or staging environments under time pressure.

Success Signal
Fewer irreversible wrong-environment deletions and fewer support-assisted restore requests. A useful discovery signal would be: admins can identify the target environment and consequence before confirming deletion in moderated tests, without materially slowing routine safe deletions.
```

Why it works: It separates operator safety from the proposed snapshot mechanism and ties success to a concrete destructive-action moment.

## Bad Idea-Refine Output
```markdown
Target User / Operator
All admins and developers.

Success Signal
Ship an undo button and restore endpoint.
```

Why it fails: The actor is too broad and the signal is an output, not evidence that the risk got smaller.

## Convergence Example
When there are several audiences, converge by choosing the audience whose pain determines MVP correctness.

```markdown
Audience Options
1. Admin who triggers deletion: highest leverage because prevention can happen before harm.
2. Platform support engineer: important for recovery, but only after harm occurs.
3. Engineering manager: wants reporting, but does not operate the workflow.

Recommendation
Optimize the first pass for the admin at the point of deletion. Capture support recovery as an assumption and keep manager reporting out of MVP.
```

## Weak-Assumption Challenges
- Who feels the pain first: the operator taking action, the owner of the deleted resource, support, or management?
- What customer behavior should change if the idea works?
- Is the success signal within product control, or is it a lagging business metric that needs translation?
- Could the target user be a narrower segment, such as new admins or teams with many similarly named environments?
- What would a happy customer never need to do, and are we accidentally measuring that action?

## Exa Source Links
- [Product Talk: Shifting from Outputs to Outcomes](https://www.producttalk.org/shifting-from-outputs-to-outcomes/) calibrates output-vs-outcome language and product outcomes as customer behavior or sentiment.
- [Product Talk: Defining Product Outcomes](https://www.producttalk.org/defining-product-outcomes/) highlights common outcome mistakes, including outputs disguised as outcomes and metrics outside the team's control.
- [SVPG: Planning Product Discovery](https://svpg.com/planning-product-discovery) calls for clarity on problem, user or customer, and how success will be known.
- [HBS: Know Your Customers' Jobs to Be Done](https://hbs.edu/faculty/Pages/item.aspx?num=51553) emphasizes the circumstances and job the customer is trying to get done.
