---
name: go-domain-invariant-review
description: "Review Go code changes for business-invariant preservation, state-transition correctness, acceptance semantics, and side-effect safety."
---

# Go Domain Invariant Review

## Purpose
Protect approved business rules in code so critical invariants, forbidden transitions, and acceptance behavior do not drift silently.

## Specialist Stance
- Review behavior through business invariants and legal transitions, not through implementation shape alone.
- Prioritize invalid state acceptance, side effects before preconditions, duplicate effects, and silent invariant drift.
- Treat domain wording, state names, and acceptance semantics as correctness surfaces when code changes them.
- Hand off API, data, reliability, or security depth when the domain issue depends on those seams.

## Scope
- review invariant enforcement in changed code paths
- review allowed and forbidden state transitions
- review preconditions, postconditions, and side-effect safety
- review happy-path, fail-path, and corner-case acceptance semantics
- review domain error behavior for invariant violations
- review domain-language drift when changed terminology alters business meaning
- review whether critical invariant and transition behavior remains testable

## Boundaries
Do not:
- redesign the domain model during review unless local correction is impossible
- take primary ownership of transport, DB/cache, security, or reliability depth when those are only supporting causes
- accept “eventual correction later” for hard domain invariants without an explicit process contract
- reduce domain review to happy-path-only validation

## Core Defaults
- Approved domain behavior is the source of truth.
- Hidden transition logic and implicit business assumptions are defects until proven aligned.
- Observable behavior matters, not only internal state shape.
- Preconditions should protect side effects, not explain them afterward.
- Prefer the smallest safe fix that restores invariant enforcement and deterministic behavior.

## Source Authority
Use repo-local evidence before general domain modeling advice:
- approved `spec.md`, domain docs, plans, task artifacts, and task-local design files
- existing tests, fixtures, and accepted behavior examples
- changed code and adjacent code when no approved artifact is attached

If no approved artifact is present, say the rule is inferred from code-visible behavior, tests, names, or caller expectations. Do not treat external DDD or workflow sources as business-rule authority; use them only to calibrate review questions and finding quality.

## Lazy-Loaded Review References
References are compact rubrics and example banks, not exhaustive checklists. Load at most one reference by default: choose the one that best matches the changed risk. Load a second only when the diff clearly spans independent decision pressures, such as side-effect ordering plus missing proof.

| Reference | Symptom | Behavior change |
| --- | --- | --- |
| `references/invariant-preservation-review.md` | Mutation, construction, repository save, handler guard, or direct field update may accept impossible business state. | Makes the model prove a local invariant bypass instead of asking for generic DDD reshaping. |
| `references/state-transition-review.md` | Status enum, lifecycle guard, transition table, terminal state, or event-driven state update changed. | Makes the model check legal movement and terminal-state semantics instead of redesigning a state machine. |
| `references/acceptance-and-rejection-semantics.md` | Command, domain error, no-op, duplicate, event consumer, or validation placement changes whether input is accepted, rejected, ignored, or already applied. | Makes the model preserve deterministic business acceptance semantics instead of commenting on error style. |
| `references/preconditions-side-effects-and-partial-failure.md` | Payment, refund, inventory, entitlement, event, webhook, email, or save can outlive a rejected operation. | Makes the model review guard-before-effect ordering and mixed outcomes instead of prescribing sagas by default. |
| `references/retry-duplicate-and-reorder-domain-risks.md` | Retry, replay, idempotency key, stale event, backfill, optimistic concurrency, or out-of-order consumer path changed. | Makes the model tie duplicate/reorder handling to a concrete business effect instead of saying "add dedupe." |
| `references/domain-language-and-meaning-drift.md` | Renames or vocabulary changes touch domain states, obligations, ownership, eligibility, totals, or lifecycle terms. | Makes the model distinguish behavior-changing semantic drift from pure naming taste. |
| `references/domain-test-traceability.md` | A changed invariant, transition, rejection, duplicate path, or side-effect rule has missing or weak proof. | Makes the model report missing proof only when a specific business regression can slip through instead of asking for more tests generally. |

The examples are not reusable business rules. Adapt only the review lens and finding shape, then cite the local contract or state the local inference.

## Expertise

### Invariant Preservation
- Verify each affected invariant still has a clear enforcement point.
- Flag bypass paths where alternate flows can skip critical guards.
- Reject “repair later” logic for hard invariants unless a deliberate process-level guarantee exists.
- Keep invariant ownership explicit at runtime.

### State Transition Correctness
- Verify changed logic permits only approved transitions and blocks forbidden ones.
- Require preconditions before side effects and clear postconditions afterward.
- Flag hidden transitions introduced through retries, duplicates, reorder, or side-channel updates.
- Treat incorrect transition guards as domain defects even when tests are green.

### Acceptance Behavior
- Review externally visible behavior on success, failure, and corner cases.
- Verify domain errors remain deterministic and semantics-preserving.
- Flag behavior that changes business meaning without an explicit contract or decision.

### Domain Language And Meaning Drift
- Treat domain vocabulary as evidence only when a changed term alters state, obligation, ownership, eligibility, amount meaning, or caller interpretation.
- Flag collapses of distinct business concepts, such as turning `cancelled` and `expired` into one branch, only when the local contract keeps them distinct.
- Avoid taste-only naming findings; hand off readability-only naming issues to `go-language-simplifier-review`.

### Invariant Violation Semantics
- Invariant violation must fail predictably; it must not silently continue.
- Review whether partial side effects are prevented, compensated, or otherwise kept safe.
- Treat silent corruption, silent loss, or mixed outcomes as blocker-level domain risk.

### Corner Cases And Failure Paths
- Review retry, duplicate, reorder, delay, and dependency-failure paths when they can affect business validity.
- Require these paths to preserve the same domain rules as the happy path.
- Flag undefined or contradictory state outcomes.

### Test Traceability
- Review whether critical invariants and transitions still have clear validating tests or evidence.
- Flag missing coverage for fail paths and corner cases when changed behavior depends on them.
- Keep QA ownership separate: this skill identifies domain-risky gaps, while `go-qa-review` owns test-strategy depth.

### Cross-Domain Handoffs
- Hand off transaction, query, and cache mechanics to `go-db-cache-review`.
- Hand off retry, timeout, and degradation policy depth to `go-reliability-review`.
- Hand off authz, tenant, or object-ownership root causes to `go-security-review`.
- Hand off broader architecture drift to `go-design-review`.
- Hand off coverage completeness and test-shape depth to `go-qa-review`.

## Finding Quality Bar
Each finding should include:
- exact `file:line`
- the violated invariant, transition, or acceptance rule
- concrete business impact
- the smallest safe correction
- a relevant contract or decision when one exists
- whether the issue is local code drift or needs design escalation

Keep findings local and review-shaped. Do not redesign the domain model unless the smallest safe correction cannot preserve the approved rule. If the only honest fix changes the invariant set, transition model, acceptance contract, or ownership boundary, escalate instead of smuggling a redesign into a review comment.

Severity is merge-risk based:
- `critical`: confirmed invariant violation or forbidden transition
- `high`: high-likelihood fail-path or corner-case invariant break
- `medium`: bounded but meaningful domain behavior risk
- `low`: local traceability or hardening improvement

## Deliverable Shape
Return review output in this order:
- `Findings`
- `Handoffs`
- `Design Escalations`
- `Residual Risks`
- `Validation Commands`

Use this format for each finding:

```text
[severity] [go-domain-invariant-review] [file:line]
Issue:
Impact:
Suggested fix:
Reference:
```

## Escalate When
Escalate when:
- safe correction changes the approved invariant set, state model, or acceptance rules (`go-domain-invariant-spec`)
- API-visible behavior or error semantics must change (`api-contract-designer-spec`)
- the fix depends on new transaction, cache, or consistency design (`go-db-cache-spec`)
- the right answer requires a new retry, reconciliation, or degradation model (`go-reliability-spec` or `go-distributed-architect-spec`)
- local repair exposes broader design drift (`go-design-spec`)
