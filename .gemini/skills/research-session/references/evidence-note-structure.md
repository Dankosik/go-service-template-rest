# Evidence Note Structure

## Behavior Change Thesis
When loaded for deciding whether or how to preserve `research/*.md`, this file makes the model write compact evidence notes with source hygiene, limitations, and handoff value instead of likely mistake: dumping generic notes, command output, links, or decisions that belong in `spec.md`.

## When To Load
Load when a preserved research note would materially help later synthesis, challenge, auditability, or multi-session resume, or when an existing note lacks source hygiene.

Do not load to force a universal template. Research notes stay flexible and evidence-oriented.

## Decision Rubric
Create or update `research/*.md` only when at least one is true:
- later `specification-session` needs reusable evidence, not just chat memory
- fan-in needs comparable claims or conflicts captured across lanes
- a high-impact assumption needs a source trail and limitation
- a future resume would otherwise have to rediscover the same repository surfaces

Keep the note shaped around evidence:
- question and scope
- findings with source references and confidence or limits
- conflicts, weak evidence, and absence-of-evidence caveats
- handoff implications for future specification

Do not store:
- final decisions
- task lists
- design sequences
- raw command transcripts without interpretation
- external link dumps

## Imitate
```markdown
# Export Job State Ownership Research

## Question
Which existing repository surfaces own long-running job state, tenant isolation, and terminal status transitions?

## Scope
In scope: current repository code, tests, and relevant external state-machine guidance if repository evidence is insufficient.
Out of scope: final schema decisions, migration design, implementation tasks.

## Findings
- Existing job-like records include `tenant_id` in repository method arguments.
  Evidence: file references and test names.
  Confidence: medium, because only one comparable feature exists.
- No existing terminal state named `cancelled` was found.
  Evidence: enum or status constants searched, tests searched.
  Confidence: high for current code, low for product intent.

## Conflicts Or Weak Evidence
- API naming suggests cancellation might exist later, but persisted state does not.

## Handoff
Future `specification-session` must decide whether cancellation is in scope. This note does not decide it.
```

Copy the evidence/confidence/limitation/handoff structure, not the exact headings.

Good source notes:

```markdown
- Repository: `internal/.../handler.go`, symbol `createExportJob`, shows existing request auth boundary.
- External: vendor signed URL documentation, checked during this session, used only for expiry and revocation semantics relevant to RQ3.
```

## Reject
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

Reject because it turns research into design and planning, attaches no evidence, and finalizes decisions that belong in `spec.md`.

```markdown
- Looked at some docs.
- The internet agrees.
- Existing code seems to do this.
```

Reject because the sources cannot be revisited and the claim cannot be compared at fan-in.

## Agent Traps
- Treating absence of evidence as a product decision.
- Preserving notes because research happened, not because later sessions need them.
- Keeping external source links without explaining why they affect the research question.
- Copying a template so rigidly that the note hides the actual uncertainty.
