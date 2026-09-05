# MVP Scope And Not Doing

## Load When
Load this when scope is expanding, the MVP boundary is vague, the user asks for "complete" coverage, or `Not Doing` would otherwise say only "advanced features."

## The Move
Define MVP by the learning it must produce — the smallest slice that proves the core bet against the current baseline — while keeping the safety, trust, and quality behavior the idea is supposed to validate. Defer every feature that assumes the answer. Write `Not Doing` as concrete excluded functionality, audiences, integrations, and guarantees; an abstract exclusion constrains nothing, and "complete experience" quietly re-admits dashboards, notifications, imports, exports, and admin controls.

## Inspect
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

## Reject
```markdown
MVP Scope
Undo environment deletion with snapshots, full restore, audit reports, and safety prompts.

Not Doing
- Advanced features.
```

The MVP is still platform-sized and the `Not Doing` list does not constrain anything.
