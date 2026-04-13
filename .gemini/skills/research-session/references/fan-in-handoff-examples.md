# Fan-In Handoff

## Behavior Change Thesis
When loaded after research lanes are complete, partial, or blocked, this file makes the model hand off comparable evidence, conflicts, readiness, and next-session routing instead of likely mistake: converting research into approved `spec.md` decisions or drifting into design and planning.

## When To Load
Load when research is done enough to summarize fan-in or when a blocked session needs a clean stop rule.

Do not load to draft specification sections. Handoff language prepares the next session; it does not approve decisions.

## Decision Rubric
A research handoff should answer:
- Which research questions were handled, by whom, and from what evidence surface?
- Which lane claims are comparable, conflicting, weak, or still missing?
- Is research complete enough for `specification-session`, pre-spec challenge, targeted re-research, or blocked stop?
- What exact next session starts, and what must it decide or challenge?
- What later-phase artifacts remain missing by design?

Keep handoff verbs evidence-oriented: "found", "did not find", "conflicts with", "needs decision", "ready only if", "blocked by". Avoid "approved", "decided", "designed", "planned", or "implemented".

## Imitate
Use this shape in `workflow-plans/research.md`, `workflow-plan.md`, or a preserved `research/*.md` note:

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
- Do not create `spec.md`, `workflow-plans/specification.md`, `design/`, `tasks.md`, or optional `plan.md` in this research session.
```

Copy the comparable-claims and readiness shape when fan-out produced multiple lane outputs.

For blocked research:

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

## Reject
```markdown
## Fan-In Summary

Decision: use POST /exports, store jobs in Postgres, and issue signed S3 URLs.
Next: create `spec.md`, `design/`, `plan.md`, and implementation tasks now.
```

Reject because final API, data, and storage decisions belong in `spec.md`, and design/planning artifacts are later-phase outputs.

## Agent Traps
- Treating "ready for specification" as "specification has started".
- Flattening lane conflicts into a single confident story.
- Omitting the stop rule because the next action feels obvious.
- Hiding a blocker in prose instead of setting `Ready for next session: no` or equivalent routing.
