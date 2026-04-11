# Approved Decision Conformance

## Behavior Change Thesis
When loaded for symptom "the diff introduces behavior or structure not in the approved artifacts," this file makes the model treat code as design drift or a reopen trigger instead of letting implementation become the new decision record.

## When To Load
Load this when a diff appears to introduce behavior, ownership, lifecycle, fallback, contract, async, rollout, or delivery decisions not present in approved `spec.md`, `design/`, `plan.md`, `tasks.md`, or repository baseline.

Use this only when approved intent exists or the repo baseline clearly owns the decision. Do not demand artifacts for eligible tiny/direct-path fixes.

## Decision Rubric
- Flag code that changes structural, lifecycle, contract, durability, fallback, or validation obligations outside approved scope.
- Do not flag every local helper or refactor as "needs ADR"; conformance matters for decisions with real ownership, behavior, rollout, or proof impact.
- If the code reveals a better design, make a design escalation instead of rewriting the plan inside the review.
- If approved artifacts are absent because the task is legitimately tiny, review against local correctness and do not invent a documentation gate.
- Cite the exact section when possible: `spec.md` Decisions, Scope / Non-goals, `design/sequence.md`, `plan.md`, or `tasks.md`.

## Imitate
```text
[critical] [go-design-review] internal/infra/http/imports.go:97
Issue: The endpoint now defers work to an in-memory background queue, but the approved spec describes synchronous request handling.
Impact: The diff silently changes durability, shutdown, retry, and response semantics without a design decision or proof path.
Suggested fix: Restore the synchronous flow, or reopen the spec/design to decide async ownership, lifecycle, persistence, and validation.
Reference: task `spec.md` Decisions and `design/sequence.md` if present.
```

Copy this shape when implementation changes the runtime model.

```text
[high] [go-design-review] cmd/service/internal/bootstrap/dependencies.go:58
Issue: Startup admission now falls back to serving when a dependency probe times out, but the approved lifecycle model requires dependency validation before accepting traffic.
Impact: This changes availability and correctness semantics in code rather than in the reliability/design decision record.
Suggested fix: Remove the fallback or route the new fail-open policy through approved reliability/design work.
Reference: `docs/repo-architecture.md` startup path and task reliability decisions if present.
```

Copy this shape when a fallback or lifecycle policy is smuggled through code.

```text
[high] [go-design-review] api/openapi/service.yaml:142
Issue: The API schema changes despite the approved scope excluding contract behavior.
Impact: Clients, generated handlers, and validation obligations change without the contract review path.
Suggested fix: Remove the schema change from this diff or reopen the spec for an API-contract decision and hand off to the API reviewer.
Reference: task `spec.md` Scope / Non-goals.
```

Copy this shape when a scoped-out decision appears in implementation.

## Reject
```text
[low] [go-design-review] internal/app/widgets/format.go:18
Issue: This helper was not mentioned in the spec.
```

Reject because local implementation details do not need explicit approval unless they change a decision surface.

```text
[medium] [go-design-review] internal/infra/http/imports.go:97
Issue: Async is bad.
Suggested fix: Make it sync.
```

Reject because the finding must be about conformance to approved behavior, not generic architectural preference.

## Agent Traps
- Do not override approved repo intent with outside best-practice links.
- Do not accept TODO-driven ownership after merge when code already branches on that decision.
- Do not bury a conformance issue as a handoff; make the design drift visible, then hand off specialist proof if needed.

## Validation Shape
Compare the diff against approved scope, decisions, sequence, and task ledger. Proof is not "tests pass"; proof is that the implementation still matches the approved behavior or that the task has been explicitly reopened for a new decision.
