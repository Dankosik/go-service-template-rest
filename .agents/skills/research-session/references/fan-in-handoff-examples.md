# Fan-In Handoff Examples

## When To Load
Load this file when research lanes are complete or blocked and the session needs a clean handoff without writing `spec.md`.

Do not load it to draft specification sections. Handoff language should prepare the next session, not approve decisions.

## Good Fan-In Handoff Artifact
Use a shape like this in `workflow-plans/research.md`, `workflow-plan.md`, or a preserved `research/*.md` note:

```markdown
## Fan-In Summary

Handled questions:
- RQ1 API patterns: evidence gathered from route files and generated handler tests.
- RQ2 state ownership: evidence gathered from repository methods and migrations.
- RQ3 download security: evidence gathered from auth middleware and signing helper references.

Comparable claims:
- API lane found existing create/get async job naming.
- Data lane found job status persistence, but no cancellation terminal state.
- Security lane found tenant auth checks at request time, but no evidence that signed URLs re-check tenant after issuance.

Conflicts or weak evidence:
- Cancellation scope remains unresolved.
- Signed URL post-issuance tenant enforcement has weak repository evidence.

Readiness:
- Research is complete enough for `specification-session` only if cancellation is recorded as an open question or out-of-scope candidate, and signed URL tenant enforcement is treated as a planning-critical question.

Next session starts with:
- specification, with pre-spec challenge expected before approving `spec.md`.

Stop rule:
- Do not create `spec.md`, `workflow-plans/specification.md`, `design/`, `plan.md`, or `tasks.md` in this research session.
```

## Good Blocked Handoff Artifact
```markdown
## Fan-In Summary

Research status: blocked
Blocker: no repository evidence is available for who owns export file lifecycle after job completion.
Evidence gathered: API and auth surfaces were checked; storage lifecycle code was not found under the expected packages.
Next action: targeted local research on storage lifecycle ownership, or a read-only data/reliability lane if the package owner remains unclear.
Ready for next session: no
Next session starts with: targeted research
Stop rule: do not proceed to specification until lifecycle ownership is answered or explicitly recorded as an upstream blocker.
```

## Bad Fan-In Handoff Artifact
```markdown
## Fan-In Summary

Decision: use POST /exports, store jobs in Postgres, and issue signed S3 URLs.
Next: create `spec.md`, `design/`, `plan.md`, and implementation tasks now.
```

Why this is bad:
- final API, data, and storage decisions belong in `spec.md`
- `design/`, `plan.md`, and `tasks.md` are later-phase artifacts
- the research session must stop after routing and evidence handoff

## Evidence Vs Decision Separation
Good:

```markdown
Evidence: all comparable handlers return `202 Accepted` for accepted async work, but no route combines job creation with signed download issuance.
Handoff: future `specification-session` must decide whether download URL issuance is part of job creation, job completion, or a separate endpoint.
```

Bad:

```markdown
Decision: job creation returns `202 Accepted` and includes the signed download URL.
```

The good version gives specification enough context to decide. The bad version approves contract behavior inside research.

## Exa Source Links
These external method links were found through Exa before drafting the examples. Use them as support for transparent evidence handoff only.

- [Penn Libraries, Developing the Research Question](https://guides.library.upenn.edu/c.php?g=475980&p=10914820)
- [University of Tasmania, Documenting Search Strategies](https://utas.libguides.com/SystematicReviews/Documenting)
- [Cochrane Handbook, Chapter 4: Searching for and selecting studies](http://training.cochrane.org/handbook/current/chapter-04)
- [BMJ, The PRISMA 2020 statement](https://www.bmj.com/content/bmj/372/bmj.n71.full.pdf)
