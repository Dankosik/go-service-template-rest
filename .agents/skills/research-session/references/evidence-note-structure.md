# Evidence Note Structure Examples

## When To Load
Load this file when deciding whether to create or update `research/*.md`, or when a preserved research note needs better source-note hygiene.

Do not load it to force a universal template. Research notes should stay flexible and evidence-oriented.

## Good Evidence Note Shape
Use a shape like this when the note will materially help later synthesis, challenge, or resume:

```markdown
# Export Job State Ownership Research

## Question
Which existing repository surfaces own long-running job state, tenant isolation, and terminal status transitions?

## Scope
In scope: current repository code, tests, and relevant external state-machine guidance if repository evidence is insufficient.
Out of scope: final schema decisions, migration design, implementation tasks.

## Findings
- Finding 1: Existing job-like records include `tenant_id` in repository method arguments.
  Evidence: file references and test names.
  Confidence: medium, because only one comparable feature exists.
- Finding 2: No existing terminal state named `cancelled` was found.
  Evidence: enum or status constants searched, tests searched.
  Confidence: high for current code, low for product intent.

## Source Notes
- Repository: `path/to/file.go`, why it matters, relevant symbol names.
- External: source URL, accessed during this session, one-sentence relevance.

## Conflicts Or Weak Evidence
- API naming suggests cancellation might exist later, but persisted state does not.

## Handoff
Future `specification-session` must decide whether cancellation is in scope. This note does not decide it.
```

## Bad Evidence Note Shape
```markdown
# Export Job Design

Decision: use a `jobs` table with statuses `queued`, `running`, `succeeded`, `failed`.
Plan:
- create migration
- update repository
- add worker
Tests:
- write integration tests
```

Why this is bad:
- the title and content turn a research note into a design and task plan
- no evidence is attached to the claims
- it finalizes decisions that belong in `spec.md`

## Source-Note Hygiene
Good source notes:

```markdown
- Repository: `internal/.../handler.go`, symbol `createExportJob`, shows existing request auth boundary.
- External: [University of Tasmania, Documenting Search Strategies](https://utas.libguides.com/SystematicReviews/Documenting), used only for the practice of recording databases, dates, exact searches, limits, and reused strategies.
- External: [University of Oxford, Reading, note-taking and library skills](https://www.ox.ac.uk/students/academic/guidance/skills/research), used only for the practice of recording full source details and URLs for later revisit.
```

Bad source notes:

```markdown
- Looked at some docs.
- Cochrane says do comprehensive research.
- The internet agrees.
```

## Evidence Vs Decision Separation
Good:

```markdown
Finding: Search found no repository evidence of a retry budget constant for export-like workers.
Limitation: only current code was searched; no production config was available.
Handoff: specification or targeted research must decide whether retry policy is a requirement or out of scope.
```

Bad:

```markdown
Decision: retries are out of scope because no constant exists.
```

Absence of evidence can be a finding, but it is not automatically a product or architecture decision.

## Exa Source Links
These external method links were found through Exa before drafting the examples. Use them for note-taking and documentation hygiene only.

- [University of Oxford, Reading, note-taking and library skills](https://www.ox.ac.uk/students/academic/guidance/skills/research)
- [University of Reading, Researching your assignments](https://libguides.reading.ac.uk/academicintegrity/researching)
- [Yale Library, Note Taking and Citation Management](https://guides.library.yale.edu/c.php?g=1391702&p=10293822)
- [Elmira College Library, Take Notes](https://libguides.elmira.edu/research/take_notes)
- [University of Tasmania, Documenting Search Strategies](https://utas.libguides.com/SystematicReviews/Documenting)
