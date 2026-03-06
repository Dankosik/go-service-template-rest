---
name: go-domain-invariant-review
description: "Review Go code changes for business-invariant preservation, state-transition correctness, acceptance semantics, and side-effect safety."
---

# Go Domain Invariant Review

## Purpose
Protect approved business rules in code so critical invariants, forbidden transitions, and acceptance behavior do not drift silently.

## Scope
- review invariant enforcement in changed code paths
- review allowed and forbidden state transitions
- review preconditions, postconditions, and side-effect safety
- review happy-path, fail-path, and corner-case acceptance semantics
- review domain error behavior for invariant violations
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
