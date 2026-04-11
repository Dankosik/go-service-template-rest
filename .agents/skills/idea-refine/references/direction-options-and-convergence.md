# Direction Options And Convergence

## When To Load
Load this when multiple product directions remain plausible, the user asks for brainstorming but needs a recommendation, or the raw idea is a grab bag of unrelated features.

## Use This Lens
- Explore a small set of genuinely different directions, usually two or three.
- Compare options against the same target user, problem, success signal, MVP size, and risk profile.
- Recommend one direction and explain why the rejected options are not first.
- Keep alternative directions alive only as explicit follow-ons or fallback triggers.

## Good Idea-Refine Output
Input: "We need better onboarding: checklists, videos, templates, and AI setup help."

Good:

```markdown
Direction Options
1. Guided checklist: fastest path to reduce first-session confusion, but may become a task list that does not address setup quality.
2. Template-first onboarding: strongest fit if users already know their goal but struggle to start from a blank state.
3. AI setup helper: attractive for flexibility, but riskiest because value and trust are both unproven.

Recommended Direction
Start with template-first onboarding for one high-value setup path. It tests whether users can reach a meaningful first outcome faster without taking on conversational AI risk.
```

Why it works: The options are comparable, the recommendation is clear, and the decision is tied to the core learning bet.

## Bad Idea-Refine Output
```markdown
Recommended Direction
Build checklists, videos, templates, and AI setup help because together they create a complete onboarding experience.
```

Why it fails: It treats a grab bag as strategy and avoids choosing the smallest direction that can validate the core bet.

## Convergence Example
Use this pattern when the options are close:

```markdown
Convergence Rule
Choose the option that tests the riskiest value assumption with the least irreversible scope.

Decision
Pick template-first onboarding because it tests whether pre-shaped starting points drive activation. Defer videos because they can explain a broken flow without fixing it. Defer AI setup until there is evidence that templates are too rigid for the target user.
```

## Weak-Assumption Challenges
- Are these options solving the same opportunity or several unrelated opportunities?
- Which option produces the clearest learning if it fails?
- Which option would still be valuable without the flashiest technology?
- Which option creates the least downstream design and operational commitment?
- What fact would make the recommendation switch to the runner-up?

## Exa Source Links
- [Product Talk: Opportunity Solution Trees](https://www.producttalk.org/opportunity-solution-trees/) supports comparing multiple solutions against one target opportunity and using assumption tests to decide.
- [SVPG: The Origin of Product Discovery](https://svpg.com/the-origin-of-product-discovery) frames discovery as figuring out the right product rather than dictating requirements for a requested feature.
- [SVPG: Planning Product Discovery](https://svpg.com/planning-product-discovery) calibrates option comparison against value, usability, feasibility, viability, and stakeholder risks.
- [Shape Up: Set Boundaries](https://basecamp.com/shapeup/1.2-chapter-03) warns against grab-bag ideas and recommends narrowing the problem.
