# Spec-First Handoff Examples

## Load When
Load this only after convergence, when the output needs to become a compact handoff artifact for engineering framing.

## The Move
Hand off a product direction, not architecture or tasks: problem, target actor, recommended direction, why, assumptions, MVP scope, `Not Doing`, open questions, and an explicit next handoff — `spec-first-brainstorming` unless the user chose another step. Preserve uncertainty so weak assumptions stay visible for the next skill, and record rejected options only where they prevent obvious scope re-expansion.

## Inspect
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
Open Questions
- What should the architecture be?
- Which tables should store incidents?
- Which endpoints should we add?
```

Idea refinement hands off product uncertainty; design questions belong to later phases and are not yet ready to ask.
