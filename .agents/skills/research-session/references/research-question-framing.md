# Research Question Framing Examples

## When To Load
Load this file when a research session has phase-ready framing but the research questions are still too broad, biased, solution-led, or mixed with future specification decisions.

Do not load it just to run the protocol. `SKILL.md`, `AGENTS.md`, and `docs/spec-first-workflow.md` remain the authority for allowed writes and phase boundaries.

## Method Notes
Use research questions to make the evidence target explicit before searching or spawning read-only lanes.

Good questions usually identify:
- the repository surface or external standard being checked
- the uncertainty that could change later specification
- the evidence needed to answer it
- whether the answer is needed now or only nice to know

Avoid questions that presuppose the answer. For example, prefer "What currently owns job state transitions, and what evidence supports that ownership?" over "How should we implement a Redis job state machine?"

## Good Research Artifact
Use this shape in `workflow-plans/research.md` or in a short `research/*.md` note when preservation is useful:

```markdown
## Research Questions

Must answer before specification:
- RQ1: Which existing package owns tenant filtering for async work, and what files prove the boundary?
  Evidence target: repository routes, service layer, repository methods, existing tenant tests.
- RQ2: What failure states already exist for long-running jobs, and where are retries or terminal states enforced?
  Evidence target: current job model, worker loops, persistence methods, test fixtures.
- RQ3: Which download URL security expectations are dictated by existing auth middleware versus external signed URL guidance?
  Evidence target: auth middleware, file-serving endpoints, external signed URL documentation.

Nice to know:
- NQ1: Naming preferences in unrelated export features.
  Defer unless a local naming conflict appears.
```

Why this is good:
- each question can be answered with evidence
- later `specification-session` can convert evidence into decisions
- no question asks the research session to approve an API, create design artifacts, or implement code

## Bad Research Artifact
Do not use research questions like these:

```markdown
## Research Questions

- Decide whether exports use Redis or Postgres.
- Write the final API contract for export jobs.
- Design the retry sequence and update tasks.md.
- Check everything about security, reliability, data, and tests.
```

Why this is bad:
- it turns research into specification, design, or planning
- "check everything" has no owned evidence target
- it hides which uncertainty blocks spec readiness

## Evidence Vs Decision Separation
Good research language:

```markdown
Finding: Existing repository methods always accept `tenantID` for export-like reads.
Evidence: `internal/...` file references and tests listed below.
Spec handoff: future `specification-session` must decide whether new export jobs use the same tenant argument pattern or need a new ownership rule.
```

Bad research language:

```markdown
Decision: New export jobs must use the existing tenant argument pattern.
Implementation plan: update the repository and tests in phase 1.
```

The first version preserves evidence and names a handoff question. The second version finalizes `spec.md` and begins planning, which is outside a research session.

## Exa Source Links
These external method links were found through Exa before drafting the examples. Use them as framing inspiration only; repository-local workflow rules remain authoritative.

- [University of Wisconsin, Develop Your Research Question](https://researchguides.library.wisc.edu/literature_review/develop_research_question)
- [Penn Libraries, Developing the Research Question](https://guides.library.upenn.edu/c.php?g=475980&p=10914820)
- [University of Minnesota, Frameworks for Systematic Review and Evidence Synthesis](https://libguides.umn.edu/c.php?g=1264119&p=9278660)
- [Mayo Clinic Libraries, Develop and Refine Your Research Question](https://libraryguides.mayo.edu/evidencesynthesis/question)
