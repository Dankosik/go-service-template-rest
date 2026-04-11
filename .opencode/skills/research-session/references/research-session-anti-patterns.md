# Research Session Anti-Patterns

## Behavior Change Thesis
When loaded for concrete research-session drift smells that do not fit a narrower positive reference, this file makes the model stop, repair the boundary, or route back to the right phase instead of likely mistake: treating `research-session` as a catch-all path into specification, design, planning, implementation, or generic note sprawl.

## When To Load
Load as challenge or smell triage when a research session is already drifting, has multiple overlapping smells, or no narrower reference matches.

Prefer narrower references first:
- broad or biased questions: `research-question-framing.md`
- mode uncertainty: `local-vs-fanout-mode-selection.md`
- vague lanes: `research-lane-planning.md`
- note sprawl or poor source notes: `evidence-note-structure.md`
- fan-in or blocked handoff: `fan-in-handoff-examples.md`

Do not load this as the default checklist for every research session.

## Decision Rubric
| Smell | Likely mistake | Repair |
| --- | --- | --- |
| `## Decisions` appears in research output | Research is approving `spec.md` content | Rename to evidence or handoff language and route final decisions to future `specification-session` |
| Component diagrams or runtime sequences appear | Research is doing technical design | Keep only repository evidence that components may be involved; route mapping to future design |
| Markdown checkboxes or task IDs appear | Research is doing planning | Replace with next-session routing and leave `plan.md`/`tasks.md` missing by design |
| Prototype or test edits are proposed | Research is implementing | Stop and reopen a later approved implementation phase if code is truly required |
| Lanes are role labels only | Fan-out cannot be synthesized | Rewrite each lane around one owned question and evidence target |
| `research/all-notes.md` grows with everything found | Note sprawl hides signal | Preserve only evidence that affects spec readiness, conflict resolution, challenge, or resume |
| External docs are cited vaguely | Source-note rot | Record URL or source name plus one-sentence relevance and limitation, or drop it |

## Imitate
Spec drift repair:

```markdown
## Research Handoff
- Evidence found existing async creation routes and timeout helpers.
- Cancellation has weak evidence and must be decided in future `specification-session`.
- Candidate questions for specification are listed, but no decisions are approved here.
```

Vague fan-out repair:

```markdown
Lanes:
- L1 api-agent/no-skill: Which existing routes use async acceptance semantics, and what statuses do they return?
- L2 security-agent/no-skill: Which existing trust boundaries apply to signed download URL issuance? Read-only research only.
- L3 data-agent/no-skill: Which tables or repository methods already model tenant-scoped job state?
```

Boundary-respecting completion:

```markdown
Research status: complete with one accepted limitation
Research mode: fan-out
Preserved notes:
- `research/export-api-evidence.md`
- `research/export-state-evidence.md`
Not preserved:
- naming notes from unrelated examples, because they do not affect spec readiness
Next session starts with: specification
Stop rule: no later-phase artifacts created
```

## Reject
```markdown
Research status: complete
Spec status: approved
Design status: draft
Plan status: ready
Implementation: started
```

Reject because it claims later-phase progress from inside a research-only session and creates competing authority.

```markdown
Decision: signed URLs do not need revocation.
```

Reject because weak or absent evidence can expose a decision, but cannot make it final in research.

## Agent Traps
- Using this broad file instead of a narrower reference that would give a positive shape.
- "Repairing" drift by deleting uncertainty instead of routing it.
- Treating the stop rule as optional once enough evidence has been gathered.
- Creating a new artifact mid-session to hold what should be a future `spec.md`, `design/`, `plan.md`, or `tasks.md`.
