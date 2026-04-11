# Spec-First Handoff Examples

## When To Load
Load this when the idea-refine pass has converged on one direction and needs a compact handoff into `spec-first-brainstorming`.

## Use This Lens
- Hand off a refined product direction, not architecture or tasks.
- Include enough context for `spec-first-brainstorming` to produce a problem frame, behavior delta, constraints, assumptions, prioritized questions, and readiness decision.
- Preserve open questions instead of inventing answers.
- Make the next handoff explicit: usually `spec-first-brainstorming`.

## Good Idea-Refine Output
Input: "Incidents are hard to understand fast. Maybe an incident cockpit with AI summaries, timeline reconstruction, service maps, and Slack integration."

Good:

```markdown
Problem
During the first triage window of a severe incident, primary on-call engineers spend too long reconstructing relevant changes, alert context, and ownership clues before forming the first credible hypothesis.

Target User / Operator
Primary on-call engineer for a service-owning team.

Recommended Direction
Build toward a first-triage incident timeline, not a broad incident cockpit. The first pass should assemble recent deploys, alerts, owning services, and a human-editable hypothesis note for one incident source.

Why This Direction
It targets the earliest diagnostic bottleneck, keeps the MVP narrower than a dashboard-plus-Slack bundle, and lets later spec work compare AI assistance against structured evidence assembly.

Key Assumptions To Validate
- Operators can identify a better first hypothesis when recent changes and alert context are assembled in one place.
- The relevant incident data is available with acceptable latency and source trust.
- A human-editable note is safer than an auto-published AI summary for the first pass.

MVP Scope
One incident source, one service-owning team path, recent deploys, active alerts, service owner hints, and a manually editable hypothesis note.

Not Doing
- No Slack bot.
- No full service map.
- No automated incident commander.
- No cross-organization analytics.
- No auto-published AI summary.

Open Questions
- Which incident source should anchor the first pass?
- What time window best captures useful recent changes?
- What validation signal should define "faster first hypothesis"?

Next Handoff
Use `spec-first-brainstorming` to turn this into a behavior-delta frame with constraints, prioritized open questions, and a challenge-readiness decision.
```

Why it works: It is concrete enough for engineering framing while still avoiding final API, data, architecture, or task decisions.

## Bad Idea-Refine Output
```markdown
Next Handoff
Implement the incident cockpit. Create API endpoints for timelines, add Slack integration, use AI summaries, and build the service map UI.
```

Why it fails: It skips `spec-first-brainstorming`, smuggles in implementation decisions, and does not preserve assumptions for challenge.

## Convergence Example
When final handoff needs to explain rejected directions:

```markdown
Rejected For MVP
Slack digest was rejected as first direction because it distributes context after the context quality problem is solved.
Full service map was rejected because ownership discovery may be a secondary issue, not the first diagnostic bottleneck.
Auto-summary was rejected because trust and review risk would dominate the first pass.
```

## Weak-Assumption Challenges
- Can `spec-first-brainstorming` infer the behavior delta from this handoff without reopening ideation?
- Are any downstream design decisions being presented as product decisions?
- Does the handoff keep enough rejected-option context to prevent scope from re-expanding?
- Are open questions prioritized, or is the next skill being asked to rediscover the whole problem?
- Is the `Not Doing` list specific enough to block the obvious scope creep?

## Exa Source Links
- [SVPG: The Origin of Product Discovery](https://svpg.com/the-origin-of-product-discovery) distinguishes discovery, figuring out what to build, from delivery, building it right.
- [SVPG: Planning Product Discovery](https://svpg.com/planning-product-discovery) supports handing off problem, target user, success signal, and key risks before deeper discovery or design.
- [Shape Up: Write the Pitch](https://basecamp.com/shapeup/1.5-chapter-06) calibrates a concise handoff around problem, appetite, solution, rabbit holes, and no-gos.
- [Product Talk: Opportunity Solution Trees](https://www.producttalk.org/opportunity-solution-trees/) supports handing off outcome, opportunity, candidate solution, and assumption-test context.
