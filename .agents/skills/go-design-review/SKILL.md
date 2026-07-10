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
- Treat an unexplained surviving replaced or unused legacy surface as source-of-truth drift when the approved artifact chain does not retain it with owner, reason, proof, and exit condition.
- Prefer one explicit same-package seam for stable local policy over both scattered copies and vague helper buckets.
- Treat unapproved custom infrastructure, new runtime dependencies, and meaningful helper/abstraction choices as design drift when the approved artifact chain lacks stdlib, repository-pattern, mature-OSS, and custom-code due diligence.
- Treat invented or cargo-culted design/system patterns as design drift when no present requirement and live-alternative evidence justify them, or when implementation violates the selected pattern's real guarantee.
- Treat code-level pattern choices as design drift only when they become ownership, dependency, or maintainability problems: useful local patterns reduce code and clarify seams; pattern-shaped mini-frameworks create accidental complexity.
- Hand off deep API, data, security, reliability, performance, or QA issues when design review only detects the seam.
- Keep output review-shaped: findings, handoffs, design escalations, residual risks, and validation notes. Do not redesign the system from scratch inside the review.

## Evidence Order
Use the strongest local evidence first:
1. Changed diff and directly affected tests or generated outputs.
2. Task-local `spec.md`, `design/`, and `tasks.md` when present.
3. Repository baseline docs such as `docs/repo-architecture.md` plus canonical runtime sources like OpenAPI, config policy, migrations, and generation inputs.
4. External references only to calibrate review patterns, never to override repository-approved intent.

If approved specs or design docs exist, cite them before external style or architecture sources.

## Reference Files Selector
References are compact rubrics and example banks, not exhaustive checklists. Load at most one reference by default: choose the file whose symptom matches the strongest review pressure. Load multiple references only when the diff spans independent decision pressures, such as a dependency-direction bug plus a separate source-of-truth drift.

| Load this file | Symptom | Behavior change when loaded |
| --- | --- | --- |
| `references/boundary-and-ownership-drift.md` | behavior, policy, or construction moved across app/domain/infra/HTTP/config/bootstrap boundaries | choose the owning boundary and smallest move back instead of giving generic layering advice |
| `references/dependency-direction-and-hidden-coupling.md` | imports, callbacks, registration, globals, adapter wiring, or test helpers change who depends on whom | review the coupling mechanism and composition root instead of treating the import as style or demanding interfaces everywhere |
| `references/source-of-truth-seam-drift.md` | generated code, config, migrations, contracts, or stable local policy now have competing owners | route the fix through the canonical source or owning-package seam instead of accepting local copies or global helpers |
| `references/accidental-complexity-and-helper-buckets.md` | helpers, wrappers, premature interfaces, option bags, manager types, or `common` packages obscure ownership | distinguish useful seams from speculative indirection instead of reflexively praising or banning abstraction |
| `references/approved-decision-conformance.md` | code introduces behavior, ownership, lifecycle, fallback, contract, async, or rollout decisions outside approved artifacts | treat implementation as drift or a reopen trigger instead of letting code become the decision record |
| `references/cross-domain-handoff-examples.md` | design review found a seam, but deep correctness belongs to API, chi, data/cache, security, reliability, concurrency, performance, QA, or delivery review | write one design-shaped finding plus a targeted handoff instead of doing every specialist review or handoff spam |

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
- Legacy cleanup integrity: replaced or unused code, tests, fixtures, generated artifacts, configs, docs, skills, agents, or mirrors are removed/refactored, or retained with approved owner/reason/proof/exit condition.
- Dependency/OSS integrity: new dependencies, custom infrastructure, and material abstractions match approved due diligence, including selected and rejected stdlib, repository-pattern, OSS, and custom-code options.
- Design-choice integrity: architecture, workflow, integration, resilience, data-flow, or abstraction shapes match the approved design, and an unsupported material choice is routed back to research, specification, or technical design.
- Code-level pattern integrity: local patterns such as same-package seams, map dispatch, narrow interfaces, functional options, middleware, or table-driven tests reduce code, clarify ownership, or improve proof instead of turning into unapproved mini-frameworks.
- Hidden decisions: new fallback, async, lifecycle, contract, or data-shape behavior is approved rather than smuggled through code.
- Complexity control: abstractions, helpers, wrappers, and interfaces reduce real change risk instead of becoming ownership buckets.
- Cross-domain seams: flag design-shape risk and hand off deep specialist correctness to the owner review.

## Evidence And Shared Finding Envelope
Use the [shared review finding envelope](../../../docs/subagent-contract.md#shared-review-finding-envelope). Each finding adds the concrete design drift, change/regression/operability risk, contract or decision, smallest safe correction, and whether the problem is source-of-truth scattering, over-broad abstraction, an unexplained surviving replaced or unused legacy surface, unsupported dependency/design choice, or a code-level pattern that increases rather than removes complexity. `critical` is a merge-unsafe boundary/ownership violation; `high` includes an executable/importable/generated/validated replaced path or major design drift with material regression risk.

## Escalate When
Escalate when:
- safe correction changes the approved system shape or ownership model (`go-design-spec` or `go-architect-spec`)
- transport or API seam behavior must be redefined (`go-chi-spec` or `api-contract-designer-spec`)
- new data, cache, or consistency decisions are required (`go-db-cache-spec` or `go-data-architect-spec`)
- the issue reveals a missing domain, reliability, security, observability, or delivery contract (`go-domain-invariant-spec`, `go-reliability-spec`, `go-security-spec`, `go-observability-engineer-spec`, or `go-devops-spec`)
