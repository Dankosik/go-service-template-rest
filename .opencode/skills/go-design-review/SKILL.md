---
name: go-design-review
description: "Review Go code changes for architecture alignment, boundary integrity, source-of-truth seam integrity, accidental complexity, and maintainability drift."
---

# Go Design Review

## Purpose
Protect approved design intent in code so boundaries, ownership, maintainability, and cross-domain seams do not drift silently.

## Specialist Stance
- Review design drift as ownership, dependency direction, source-of-truth spread, and accidental complexity.
- Prioritize hidden new decisions and boundary bypasses over subjective cleanup.
- Prefer one explicit same-package seam for stable local policy over both scattered copies and vague helper buckets.
- Hand off deep API, data, security, reliability, performance, or QA issues when design review only detects the seam.

## Scope
- review implementation against approved architecture and design intent
- detect boundary violations, dependency-direction breaks, and hidden coupling
- detect accidental complexity and maintainability regressions
- detect new design decisions introduced implicitly in code
- detect stable policy scattered across files when one same-package seam should own it
- detect seam drift across API, data, security, reliability, observability, delivery, and testing when it changes system shape

## Boundaries
Do not:
- redesign the system from scratch inside review
- absorb deep specialist ownership when the real issue belongs to a dedicated review domain
- block on subjective cleanliness comments without concrete design impact
- treat green tests as proof that architecture and maintainability are still sound

## Core Defaults
- Approved design intent is the source of truth for code structure and boundary ownership.
- Review changed code and directly impacted seams first.
- Treat hidden new decisions in code as design drift until proven deliberate.
- Prefer one explicit same-package source-of-truth seam for stable local policy over both repeated file-local copies and vague helper buckets.
- Prefer the smallest correction that restores explicit ownership and maintainability.
- Escalate design change instead of smuggling it through local code review.

## Expertise

### Boundary And Ownership Integrity
- Verify component responsibility, dependency direction, and composition seams stay explicit.
- Flag hidden cross-layer coupling and implementation shortcuts that redefine ownership.
- Reject undeclared new dependencies that change architecture shape.
- Treat bypass of intended seams as design drift even if tests still pass.
- When normalization, mapping, validation, classification, or section-reading policy is stable inside one package, treat scattering that policy across files as ownership drift unless there is a clear reason to keep it local.

### Approved Decision Conformance
- Review changed behavior against the approved architecture, implementation strategy, and signed design intent.
- Flag “we’ll decide later in code” behavior, TODO-driven design, or new branching semantics that effectively change the plan.
- Treat untracked divergence from approved intent as a design finding, even when locally convenient.

### Complexity Control
- Flag speculative abstractions, wrapper layers, and ceremony that do not remove real duplication or risk.
- Flag duplicated responsibility spread across packages or components.
- Flag vague `util/common/shared/helpers` abstractions that blur ownership instead of giving one seam a clear home.
- Prefer explicit local logic over design that forces readers through multiple indirection layers for basic reasoning.
- Treat complexity as a cost when it increases future change risk, debugging burden, or review ambiguity.

### Maintainability And Evolvability
- Review whether the changed structure keeps ownership obvious and the blast radius of future change bounded.
- Flag code paths that become hard to reason about because control flow, invariants, or side effects are no longer local and explicit.
- Flag stable local policy that remains duplicated across multiple files when one seam-named same-package owner would bound future change more clearly.
- Prefer design that remains testable, operable, and extendable without hidden coupling.

### Cross-Domain Seam Integrity
- API seam: method, status, async, idempotency, and error semantics must stay aligned with the intended contract.
- Data seam: ownership, transaction boundaries, cache role, and evolution safety must stay explicit.
- Security seam: trust-boundary controls must stay enforceable by structure, not luck.
- Reliability seam: timeouts, retries, fallback, and overload behavior must stay explicit where design depends on them.
- Observability seam: correlation, route or operation identity, and telemetry cardinality must remain deliberate.
- Delivery seam: codegen, migrations, and release assumptions must not become undocumented hidden dependencies.
- Testing seam: nontrivial behavior must remain realistically provable.

### Cross-Domain Handoffs
- Hand off deep transport and contract implementation issues to `go-chi-review` or other owner reviews as appropriate.
- Hand off data and cache depth to `go-db-cache-review`.
- Hand off security depth to `go-security-review`.
- Hand off reliability depth to `go-reliability-review`.
- Hand off performance and concurrency depth to `go-performance-review` and `go-concurrency-review`.
- Hand off test-strategy depth to `go-qa-review`.

## Finding Quality Bar
Each finding should include:
- exact `file:line`
- the concrete design drift
- why it increases change, regression, or operability risk
- the smallest safe correction
- the relevant contract or decision when one exists
- whether the issue is local code drift or needs design escalation
- whether the drift is scattered source-of-truth ownership or over-broad helper abstraction

Severity is merge-risk based:
- `critical`: boundary or ownership violation that makes merge unsafe
- `high`: major design drift or complexity growth with meaningful regression risk
- `medium`: bounded maintainability or seam-integrity weakness
- `low`: local design hardening or clarity improvement

## Deliverable Shape
Return review output in this order:
- `Findings`
- `Handoffs`
- `Design Escalations`
- `Residual Risks`
- `Validation Commands`

Use this format for each finding:

```text
[severity] [go-design-review] [file:line]
Issue:
Impact:
Suggested fix:
Reference:
```

## Escalate When
Escalate when:
- safe correction changes the approved system shape or ownership model (`go-design-spec` or `go-architect-spec`)
- transport or API seam behavior must be redefined (`go-chi-spec` or `api-contract-designer-spec`)
- new data, cache, or consistency decisions are required (`go-db-cache-spec` or `go-data-architect-spec`)
- the issue reveals a missing domain, reliability, security, observability, or delivery contract (`go-domain-invariant-spec`, `go-reliability-spec`, `go-security-spec`, `go-observability-engineer-spec`, or `go-devops-spec`)
