# Spec-First Handoff Examples

## Behavior Change Thesis
When loaded for symptom "the idea-refine pass has converged and needs a handoff," this file makes the model hand off a refined product direction to `spec-first-brainstorming` instead of the likely mistake of smuggling architecture, endpoints, or task plans into the next step.

## When To Load
Load this only after convergence, when the output needs to become a compact handoff artifact for engineering framing.

## Decision Rubric
- Hand off a product direction, not architecture or tasks.
- Include problem, target actor, recommended direction, why, assumptions, MVP scope, `Not Doing`, open questions, and next handoff.
- Preserve uncertainty instead of inventing answers for the next skill.
- Include rejected-option context only when it prevents obvious scope re-expansion.
- Make the next handoff explicit: usually `spec-first-brainstorming`.

## Imitate
Input: "Incidents are hard to understand fast. Maybe an incident cockpit with AI summaries, timeline reconstruction, service maps, and Slack integration."

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

Copy the move: it is concrete enough for engineering framing while still avoiding final API, data, architecture, or task decisions.

```markdown
Rejected For MVP
Slack digest was rejected as first direction because it distributes context after the context quality problem is solved.
Full service map was rejected because ownership discovery may be a secondary issue, not the first diagnostic bottleneck.
Auto-summary was rejected because trust and review risk would dominate the first pass.
```

Copy the move: rejected directions are recorded only to keep scope from re-expanding.

## Reject
```markdown
Next Handoff
Implement the incident cockpit. Create API endpoints for timelines, add Slack integration, use AI summaries, and build the service map UI.
```

Reject this because it skips `spec-first-brainstorming`, smuggles in implementation decisions, and drops assumptions.

```markdown
Open Questions
- What should the architecture be?
- Which tables should store incidents?
- Which endpoints should we add?
```

Reject this because idea refinement should hand off product uncertainty, not pretend design questions are ready.

## Agent Traps
- Do not write acceptance criteria, tasks, API shapes, migrations, rollout steps, or validation commands.
- Do not erase rejected options if they are likely to sneak back in during framing.
- Do not leave `Next Handoff` as "build it." Name `spec-first-brainstorming` unless the user chose a different next step.
- Do not make the handoff so polished that weak assumptions disappear.
