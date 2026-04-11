# Research Session Anti-Patterns

## When To Load
Load this file when a research session starts drifting into later phases, spawns vague lanes, preserves too much, or hides blockers in chat.

Do not load it as a generic checklist. Use it to diagnose a concrete drift risk.

## Anti-Pattern: Spec Drift
Bad artifact:

```markdown
## Decisions
- Export jobs use POST /exports.
- Download URLs expire after 10 minutes.
- Cancellation is out of scope.
```

Repair:

```markdown
## Research Handoff
- Evidence found existing async creation routes and timeout helpers.
- Cancellation has weak evidence and must be decided in future `specification-session`.
- Candidate questions for specification are listed, but no decisions are approved here.
```

## Anti-Pattern: Design Drift
Bad artifact:

```markdown
## Component Design
Handler -> Service -> Repository -> Worker -> Object Storage.
```

Repair:

```markdown
Finding: repository evidence suggests handler, service, repository, and worker surfaces may all be involved.
Handoff: future technical design must map the runtime sequence after `spec.md` approves the behavior.
```

## Anti-Pattern: Planning Drift
Bad artifact:

```markdown
## Tasks
- [ ] T001 Add migration.
- [ ] T002 Implement worker.
- [ ] T003 Add tests.
```

Repair:

```markdown
Research status: complete enough to move to specification.
Next session starts with: specification.
Planning artifacts remain missing by design until approved `spec.md + design/` exist.
```

## Anti-Pattern: Implementation Drift
Bad artifact:

```markdown
I will prototype the repository method to see what is needed.
```

Repair:

```markdown
The research session can read repository methods and tests. If implementation is needed to answer the question, stop and route to a later approved implementation phase instead of writing code now.
```

## Anti-Pattern: Vague Fan-Out
Bad artifact:

```markdown
Lanes:
- api-agent: API
- security-agent: security
- data-agent: data
```

Repair:

```markdown
Lanes:
- L1 api-agent/no-skill: Which existing routes use async acceptance semantics, and what statuses do they return?
- L2 security-agent/no-skill: Which existing trust boundaries apply to signed download URL issuance? Read-only research only.
- L3 data-agent/no-skill: Which tables or repository methods already model tenant-scoped job state?
```

## Anti-Pattern: Evidence Sprawl
Bad artifact:

```markdown
research/all-notes.md contains every command output, every unrelated package mention, and every external link found during browsing.
```

Repair:

```markdown
Preserve only evidence that affects spec readiness, conflict resolution, challenge, or resume. Leave disposable observations in chat or omit them.
```

## Anti-Pattern: Source-Note Rot
Bad artifact:

```markdown
External docs say use frameworks. Existing code seems to do this.
```

Repair:

```markdown
External source: URL plus one-sentence relevance.
Repository source: exact file path or symbol plus why it matters.
Limitation: what was not checked.
```

## Evidence Vs Decision Separation
Good:

```markdown
Evidence: signed URL guidance and repository auth middleware both emphasize request-time authorization, but repository evidence is weak for post-issuance tenant revocation.
Open point: future specification must decide whether revocation is required or accepted as out of scope.
```

Bad:

```markdown
Decision: signed URLs do not need revocation.
```

The research session can identify weak evidence and expose the decision. It cannot make the decision final.

## Good Research Artifact
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

## Bad Research Artifact
```markdown
Research status: complete
Spec status: approved
Design status: draft
Plan status: ready
Implementation: started
```

Why this is bad:
- it claims later-phase progress from inside a research-only session
- it creates competing authority and makes resume unsafe

## Exa Source Links
These external method links were found through Exa before drafting the examples. Use them as support for anti-pattern repair only.

- [University of Wisconsin, Develop Your Research Question](https://researchguides.library.wisc.edu/literature_review/develop_research_question)
- [Penn Libraries, Developing the Research Question](https://guides.library.upenn.edu/c.php?g=475980&p=10914820)
- [University of Tasmania, Documenting Search Strategies](https://utas.libguides.com/SystematicReviews/Documenting)
- [University of Oxford, Reading, note-taking and library skills](https://www.ox.ac.uk/students/academic/guidance/skills/research)
- [University of Reading, Researching your assignments](https://libguides.reading.ac.uk/academicintegrity/researching)
- [Cochrane Handbook, Chapter 4: Searching for and selecting studies](http://training.cochrane.org/handbook/current/chapter-04)
