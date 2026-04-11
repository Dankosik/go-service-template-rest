# MVP Scope And Not Doing

## When To Load
Load this when the idea is growing into a large feature bundle, the MVP boundary is vague, the user asks for "complete" coverage, or the `Not Doing` list is missing or toothless.

## Use This Lens
- Define MVP by the learning it must produce, not by the smallest amount of code.
- Keep safety, trust, and basic quality intact; MVP is not permission to ship sloppy work.
- Compare against the current baseline instead of an ideal future product.
- Write `Not Doing` as concrete excluded functionality, audiences, use cases, integrations, or guarantees.

## Good Idea-Refine Output
Input: "Build undo for environment deletion with snapshots, restore API, audit timeline, and admin reporting."

Good:

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

Why it works: The MVP validates whether prevention reduces wrong deletions before committing to a broad recovery platform.

## Bad Idea-Refine Output
```markdown
MVP Scope
Undo environment deletion with snapshots, full restore, audit reports, and safety prompts.

Not Doing
- Advanced features.
```

Why it fails: The MVP is still a platform-sized bundle and the `Not Doing` list does not constrain anything.

## Convergence Example
When scope is contested, converge by linking each item to the core bet:

```markdown
Core Bet
Wrong-environment deletion is primarily a pre-action comprehension problem, not a post-action recovery problem.

Keep In MVP
Confirmation flow and audit event, because they test comprehension and provide a minimal support trail.

Defer
Snapshots and restore API, because they assume recovery is the right product direction before prevention has been tested.
```

## Weak-Assumption Challenges
- What is the one learning goal this MVP must produce?
- Which part of the proposed scope is only there to make the idea feel complete?
- What can users already live with today, even if imperfectly?
- What is explicitly out of bounds for this first pass?
- Are we cutting scope or accidentally cutting necessary safety and trust?

## Exa Source Links
- [Lean Startup Co.: What Is an MVP?](https://leanstartup.co/resources/articles/what-is-an-mvp/) defines MVP around validated learning with least effort and warns that it is not merely a minimal product.
- [Shape Up: Set Boundaries](https://basecamp.com/shapeup/1.2-chapter-03) frames appetite, narrowed problems, and fixed time with variable scope.
- [Shape Up: Write the Pitch](https://basecamp.com/shapeup/1.5-chapter-06) defines no-gos as explicit exclusions that make a concept tractable.
- [Shape Up: Decide When to Stop](https://basecamp.com/shapeup/3.5-chapter-13) calibrates scope decisions against baseline and core use cases rather than an ideal solution.
