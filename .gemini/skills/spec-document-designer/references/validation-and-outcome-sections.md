# Validation And Outcome Sections

## Behavior Change Thesis
When loaded for vague proof language or closeout text, this file makes the model separate forward-looking validation obligations from evidence-backed outcome claims instead of writing "run tests" or "done."

## When To Load
Load this when writing proof expectations before handoff, repairing vague acceptance criteria, or closing a spec after validation evidence exists.

## Decision Rubric
- `Validation` is forward-looking proof intent before implementation or validation runs.
- `Outcome` is closeout after evidence exists; omit it or leave it clearly pending before proof.
- Each validation bullet should name the behavior or failure path plus the kind of proof that would demonstrate it.
- Do not dictate test file names, exact commands, or task order unless they are already known and materially constrain planning.
- Do not claim broader validation than the fresh evidence supports.
- Convert behavior-level acceptance criteria into `Decisions`; convert proof-level acceptance criteria into `Validation`.
- If proof obligations are too layered for `spec.md`, trigger `test-plan.md` during planning instead of bloating `Validation`.

## Imitate

Forward-looking proof:

```markdown
## Validation
- Unit tests prove token reload keeps last known-good values after a failed config read.
- Integration smoke proves reload works without process restart.
- Log assertions prove secret values are redacted on reload failure.
```

Copy this: each item is specific enough for planning without dictating task order.

Evidence-backed closeout:

```markdown
## Outcome
- Implemented runtime token reload with last known-good fallback.
- Fresh validation: `go test ./internal/auth ./internal/config` passed on 2026-04-11.
- Follow-up: none; rollout risk remains limited to the existing config source.
```

Copy this: it is written after proof exists and states exactly what evidence ran.

## Reject

```markdown
## Validation
- Run tests.
- Make sure it works.
```

Failure: planning gets no proof obligation and important failure paths stay hidden.

```markdown
## Outcome
- Done.
```

Failure: it claims completion without evidence and does not say what was validated.

## Agent Traps
- Writing `Outcome` during specification as if implementation already happened.
- Treating acceptance criteria as validation when they are actually behavior decisions.
- Listing every possible test layer rather than the proof that matters for this spec.
- Saying "all tests pass" without fresh command evidence and scope.
