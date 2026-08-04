# Problem Vs Solution Framing

## Behavior Change Thesis
When loaded for symptom "the user names a feature, tool, vendor, AI mechanism, or feature bundle before naming the pain," this file makes the model recover the product problem instead of repeating the requested solution as the recommendation.

## When To Load
Load this when the prompt says "we need X," "add Y," or "use Z" and the underlying user/operator breakdown is still blurry.

## The Move
Treat the proposed feature as evidence about the pain and convert it into one concrete user/operator situation — who cannot decide or act, at which moment, at what cost. The frame is right when two materially different solutions could address the same pain; keep it upstream of architecture, API, data, and task breakdown. Name the pain before choosing any mechanism, including a cooler one, and keep "AI" a candidate mechanism unless the pain itself is model governance, trust, or review. One concrete workflow story unblocks framing faster than a discovery interview.

## Imitate
Input: "We need an AI incident cockpit with Slack summaries, timelines, and service maps."

```markdown
Problem
Primary on-call engineers lose the first 10-15 minutes of a severe incident reconstructing what changed, which alerts matter, and who has current context.

Recommended Direction
Start with a first-triage incident timeline that assembles alerts, recent deploys, ownership hints, and a human-editable hypothesis note. Treat Slack output and broad service maps as follow-ons until the first-triage pain is proven.
```

Copy the move: the user's ingredients stay visible but subordinate to the real triage breakdown.

## Reject
```markdown
Problem
The product needs better visibility.
```

"Visibility" is not yet a user problem; it hides who cannot decide or act today.
