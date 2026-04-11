# Research Question Framing

## Behavior Change Thesis
When loaded for a research session whose questions are broad, solution-led, biased, or mixed with future decisions, this file makes the model write answerable evidence-targeted questions instead of likely mistake: asking research to decide the solution, check everything, or draft later-phase artifacts.

## When To Load
Load when phase-ready framing exists, but the research questions still do not say what evidence would change spec readiness.

Do not load just to run the research-session protocol. `SKILL.md`, `AGENTS.md`, and `docs/spec-first-workflow.md` remain authoritative for allowed writes and phase boundaries.

## Decision Rubric
- Ask only questions whose answers can change scope, correctness, constraints, risk handling, or readiness for `specification-session`.
- Name the evidence target: repository surface, artifact, external standard, or prior lane output.
- Split must-answer-now from nice-to-know so disposable curiosity does not become a lane.
- Phrase uncertainty without presupposing the answer. Prefer "What currently owns job state transitions, and what proves it?" over "How should we implement a Redis job state machine?"
- Keep decisions out of the question. Research can surface candidates; it cannot approve API, data, design, planning, or implementation choices.

## Imitate
Use this shape in `workflow-plans/research.md` or in a preserved `research/*.md` note when preservation is useful:

```markdown
## Research Questions

Must answer before specification:
- RQ1: Which existing package owns tenant filtering for async work, and what files prove the boundary?
  Evidence target: repository routes, service layer, repository methods, existing tenant tests.
- RQ2: What failure states already exist for long-running jobs, and where are retries or terminal states enforced?
  Evidence target: current job model, worker loops, persistence methods, test fixtures.
- RQ3: Which download URL security expectations come from existing auth middleware versus external signed URL guidance?
  Evidence target: auth middleware, file-serving endpoints, external signed URL documentation.

Nice to know:
- NQ1: Naming preferences in unrelated export features.
  Defer unless a local naming conflict appears.
```

Copy the separation between blocking research and nice-to-know checks, plus the evidence target under each question.

## Reject
```markdown
## Research Questions

- Decide whether exports use Redis or Postgres.
- Write the final API contract for export jobs.
- Design the retry sequence and update tasks.md.
- Check everything about security, reliability, data, and tests.
```

Reject because these are specification, design, planning, and vague audit requests disguised as research questions.

## Agent Traps
- Turning a candidate implementation into the research question.
- Asking one lane to answer "API/data/security/reliability" without a concrete uncertainty.
- Treating absence of repo evidence as an approved decision rather than a finding to hand off.
- Keeping a question because it is interesting even though it cannot affect the next session.
