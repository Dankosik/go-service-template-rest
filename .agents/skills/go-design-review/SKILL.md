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
- Keep output review-shaped: findings, handoffs, design escalations, residual risks, and validation notes. Do not redesign the system from scratch inside the review.

## Evidence Order
Use the strongest local evidence first:
1. Changed diff and directly affected tests or generated outputs.
2. Task-local `spec.md`, `design/`, `plan.md`, and `tasks.md` when present.
3. Repository baseline docs such as `docs/repo-architecture.md` plus canonical runtime sources like OpenAPI, config policy, migrations, and generation inputs.
4. External references only to calibrate review patterns, never to override repository-approved intent.

If approved specs or design docs exist, cite them before external style or architecture sources.

## Reference Files Selector
Load only the reference file needed for the active drift pattern:

| Load this file | When the diff suggests |
| --- | --- |
| `references/boundary-and-ownership-drift.md` | package responsibility moved, a component bypassed its owner, or behavior crossed app/infra/bootstrap/domain boundaries |
| `references/dependency-direction-and-hidden-coupling.md` | imports, globals, callbacks, registration, or adapter wiring create a hidden dependency direction change |
| `references/source-of-truth-seam-drift.md` | generated code, config, migrations, contracts, or stable local policy now have multiple competing owners |
| `references/accidental-complexity-and-helper-buckets.md` | helper packages, wrapper layers, premature interfaces, or indirection obscure ownership without reducing real risk |
| `references/approved-decision-conformance.md` | code introduces behavior or architecture decisions not present in the approved spec/design/plan |
| `references/cross-domain-handoff-examples.md` | the design review detects a seam but the deep correctness question belongs to API, data, security, reliability, performance, concurrency, QA, or delivery review |

## Boundaries
Do not:
- redesign the system from scratch inside review
- absorb deep specialist ownership when the real issue belongs to a dedicated review domain
- block on subjective cleanliness comments without concrete design impact
- treat green tests as proof that architecture and maintainability are still sound

## Review Checklist
- Boundary integrity: component ownership, package responsibility, and composition seams stay explicit.
- Dependency direction: concrete adapter dependencies do not leak inward except through approved composition roots.
- Source-of-truth integrity: generated, config, migration, contract, and stable local policy ownership stays singular.
- Hidden decisions: new fallback, async, lifecycle, contract, or data-shape behavior is approved rather than smuggled through code.
- Complexity control: abstractions, helpers, wrappers, and interfaces reduce real change risk instead of becoming ownership buckets.
- Cross-domain seams: flag design-shape risk and hand off deep specialist correctness to the owner review.

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
