# Problem Vs Solution Framing

## Behavior Change Thesis
When loaded for symptom "the user names a feature, tool, vendor, AI mechanism, or feature bundle before naming the pain," this file makes the model choose a product problem frame instead of the likely mistake of repeating the requested solution as the recommendation.

## When To Load
Load this when the prompt says "we need X," "add Y," or "use Z" and the underlying user/operator breakdown is still blurry.

## Decision Rubric
- Treat the proposed feature as evidence, not as the decision.
- Convert the request into a current user/operator situation, failed job, or costly workaround.
- Test the frame by asking: could two materially different solutions address this same pain?
- Prefer one concrete story, moment, or workflow breakdown over abstract market language.
- Stay upstream of architecture, API, data, and task breakdown. This reference exists to recover the problem, not to design the system.

## Imitate
Input: "We need an AI incident cockpit with Slack summaries, timelines, and service maps."

```markdown
Problem
Primary on-call engineers lose the first 10-15 minutes of a severe incident reconstructing what changed, which alerts matter, and who has current context.

Recommended Direction
Start with a first-triage incident timeline that assembles alerts, recent deploys, ownership hints, and a human-editable hypothesis note. Treat Slack output and broad service maps as follow-ons until the first-triage pain is proven.
```

Copy the move: it keeps the user's ingredients visible, but subordinates them to the real triage breakdown.

Input: "Let's add a customer-health dashboard."

```markdown
Problem
Customer success managers cannot reliably tell which active accounts need intervention before renewal risk becomes visible in a late-stage escalation.

Recommended Direction
Refine toward an early-risk review workflow for one account segment, not a generic dashboard. The first pass should expose the few signals a CSM can act on in a weekly review.
```

Copy the move: it names the actor, timing, and decision moment before deciding what screen or metric set should exist.

## Reject
```markdown
Recommended Direction
Build the AI incident cockpit with Slack summaries, timeline reconstruction, and service maps.

MVP Scope
Implement all three integrations and use AI to summarize incidents.
```

Reject this because it mirrors the requested bundle, never proves the pain, and turns idea refinement into delivery planning.

```markdown
Problem
The product needs better visibility.
```

Reject this because "visibility" is not yet a user problem; it hides who cannot decide or act today.

## Agent Traps
- Do not replace one solution with a cooler solution before naming the pain.
- Do not use "users need a way to..." if the rest of the sentence is still the proposed feature.
- Do not make "AI" the problem. AI is a candidate mechanism unless the actual pain is model governance, trust, or review.
- Do not ask for a full discovery interview when one concrete workflow story would unblock the framing pass.
