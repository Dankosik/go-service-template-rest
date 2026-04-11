# MVP Scope And Not Doing

## Behavior Change Thesis
When loaded for symptom "the idea is growing into a large bundle or the Not Doing list is missing/toothless," this file makes the model choose a learning-shaped MVP boundary and concrete exclusions instead of the likely mistake of calling a platform-sized bundle an MVP.

## When To Load
Load this when scope is expanding, the MVP boundary is vague, the user asks for "complete" coverage, or `Not Doing` would otherwise say only "advanced features."

## Decision Rubric
- Define MVP by the learning it must produce, not by the smallest amount of code.
- Keep safety, trust, and basic quality intact; MVP is not permission to ship sloppy work.
- Compare against the current baseline, not an ideal future product.
- Write `Not Doing` as concrete excluded functionality, audiences, use cases, integrations, or guarantees.
- Preserve the smallest slice that proves the core bet; defer features that assume the answer.

## Imitate
Input: "Build undo for environment deletion with snapshots, restore API, audit timeline, and admin reporting."

```markdown
MVP Scope
Add a safer deletion flow for one environment type: clear target identification, consequence summary, confirmation friction for production-like resources, and an audit event that support can inspect.

Not Doing
- No periodic snapshot system.
- No general restore endpoint.
- No cross-resource undo.
- No admin analytics dashboard.
- No guarantee that every deleted environment can be recovered.
```

Copy the move: it validates whether prevention reduces wrong deletions before committing to a broad recovery platform.

Input: "Let's create full team analytics for onboarding."

```markdown
MVP Scope
Support one onboarding path from blank workspace to first useful project for a single target segment. Track whether the user reaches that first outcome without needing help content.

Not Doing
- No manager analytics dashboard.
- No cross-team benchmarking.
- No video course.
- No personalized AI coach.
```

Copy the move: it narrows the first pass to the behavior change, not the surrounding reporting system.

## Reject
```markdown
MVP Scope
Undo environment deletion with snapshots, full restore, audit reports, and safety prompts.

Not Doing
- Advanced features.
```

Reject this because the MVP is still platform-sized and the `Not Doing` list does not constrain anything.

```markdown
MVP Scope
A quick prototype without auth, logging, or error handling.
```

Reject this when trust or safety is part of the value; cutting basic quality can invalidate the learning.

## Agent Traps
- Do not define MVP as "version 1 of every feature."
- Do not cut the very safety/trust behavior the idea is supposed to validate.
- Do not make `Not Doing` abstract. Name excluded integrations, actors, guarantees, and recovery paths.
- Do not let "complete experience" smuggle in dashboards, notifications, imports, exports, analytics, and admin controls by default.
