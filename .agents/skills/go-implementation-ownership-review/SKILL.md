---
name: go-implementation-ownership-review
description: "Use when changed Go may violate accepted package or file ownership, dependency direction, source-of-truth seams, or implementation boundaries; Own architecture-conformance defects in code placement and coupling; Skip when topology policy, Go semantics, local readability, or explicit whole-diff structural overbuild is primary."
---

# Go Implementation Ownership Review

Load the [shared specialist contract](../specialist-contract.md) for common selection, scope, evidence, reference, return, and handoff mechanics; apply the domain-specific rules below.

## Target And Invariants

Protect accepted implementation ownership: package/file responsibility, dependency direction, singular sources of truth, and explicit seams. Use evidence in this order: changed code and affected tests/generated output; task `spec.md`, `design/`, and `tasks.md`; repository architecture and canonical OpenAPI/config/migration/generation sources; external calibration that never overrides accepted intent.

- Keep behavior, policy, and construction in their accepted app/domain/infra/HTTP/config/bootstrap owner; composition roots are the explicit exception for wiring concrete adapters.
- Keep dependencies pointing in the accepted direction. Review imports, callbacks, registration, globals, adapter wiring, and test helpers by the coupling they create, not by an interface-everywhere rule.
- Keep generated inputs, config, migrations, contracts, and stable local policy singular. Prefer one seam-named same-package owner over scattered copies or a vague global helper bucket.
- Remove or refactor replaced code, tests, fixtures, generated artifacts, config, docs, skills, agents, and mirrors; retention requires accepted owner, reason, proof, and exit condition.
- Treat hidden fallback, async, lifecycle, contract, data-shape, integration, resilience, or workflow decisions as drift from accepted artifacts, not as implementation precedent.
- Treat custom infrastructure, runtime dependencies, or material abstractions as drift when accepted live-choice evidence required by the [research method](../../../docs/spec-first-workflow/phases/research.md#method) is absent.
- Judge helpers, wrappers, interfaces, options, managers, and patterns here only when they obscure ownership, reverse dependencies, split authority, or violate an accepted seam. Local cognitive cost belongs to simplification; explicit harsh whole-diff overbuild belongs to structural quality.
- Do not redesign topology or absorb deep API, data/cache, security, reliability, performance, test, or delivery correctness. Green tests do not prove ownership conformance.

## Symptom-Driven References

| Pressure | Load |
| --- | --- |
| Behavior, policy, or construction crossed package or layer ownership. | [boundary-and-ownership-drift.md](references/boundary-and-ownership-drift.md) |
| Imports, callbacks, registration, globals, wiring, or test helpers changed dependency direction. | [dependency-direction-and-hidden-coupling.md](references/dependency-direction-and-hidden-coupling.md) |
| Generated/config/migration/contract/local-policy authority is duplicated or bypassed. | [source-of-truth-seam-drift.md](references/source-of-truth-seam-drift.md) |
| Helpers, wrappers, interfaces, options, managers, or common packages obscure an owner or seam. | [accidental-complexity-and-helper-buckets.md](references/accidental-complexity-and-helper-buckets.md) |
| Code introduced an unapproved behavior, ownership, lifecycle, fallback, contract, async, or rollout decision. | [approved-decision-conformance.md](references/approved-decision-conformance.md) |
| An ownership seam exposes deeper specialist correctness. | [cross-domain-handoff-examples.md](references/cross-domain-handoff-examples.md) |

## Findings And Escalation

Each finding names the ownership/dependency/source-of-truth/seam drift, governing accepted contract or decision, and concrete change, regression, or operability risk. State whether the mechanism is scattered authority, an ownership-obscuring abstraction, an active replaced path, an unsupported dependency/design choice, or a pattern that violates the seam. `critical` means merge-unsafe boundary or ownership violation; `high` includes an executable/importable/generated/validated replaced path or major drift with material regression risk.

Hand off only the deeper correctness exposed by the seam. Escalate placement, package ownership, or implementation-seam decisions to `go-implementation-ownership-spec`; topology or system-authority changes to `go-system-architecture-spec`; transport/API seams to `go-chi-spec` or `go-api-contract-spec`; data/cache authority to `go-data-architecture-spec` or `go-db-cache-spec`; and missing domain, reliability, security, observability, or delivery policy to its specification owner.
