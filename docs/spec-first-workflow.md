# Workflow Router

Select the smallest non-direct execution path and current decision owner. This
file defines routing, not phase methods. If `AGENTS.md` selected Direct Work,
follow [Direct Work](spec-first-workflow/direct-work.md) and stop routing here.

## Path

Use `structured` when accepted decisions or proof must survive the current
reasoning pass. Use `orchestrated` only when coordination is material: several
independent owners, separate mutable candidates, required fresh context,
hard-to-reverse decisions, dirty-checkout isolation, or durable multi-session
execution.

A bounded read-only lane does not by itself make work orchestrated. Path is not
a quality tier. Re-route only when current evidence changes coordination,
ownership, reversibility, or proof.

## Macro Phases

| Macro phase | Possible owners |
| --- | --- |
| Definition | Intake, supporting or standalone Research, Specification |
| Technical Design | supporting Research, System / Integration Design, Go Code / Ownership Design, triggered Technical Design Review |
| Test Design | Test Design and its triggered review |
| Planning | Planning and triggered Task Review / Readiness |
| Implementation | Implementation and triggered acceptance review |

Research is a standalone macro phase only when the accepted boundary is
`research only`; otherwise it supports the active phase. Review, repair, and
the smallest upstream reopen remain inside the active macro phase. One root
session owns at most one macro phase; cross-phase continuation uses the
[Handoff](spec-first-workflow/shared/handoff.md) owner.

## Phase Selection

Evaluate these triggers before loading a phase owner:

| Observable trigger | Owner |
| --- | --- |
| Outcome, scope, authority, observable success, or first owner is ambiguous | [Intake](spec-first-workflow/phases/intake.md) |
| Current or external evidence can change a named decision | [Research](spec-first-workflow/phases/research.md) |
| Structured work lacks a ready behavior delta | [Specification](spec-first-workflow/phases/specification.md) |
| Implementation would otherwise choose a runtime boundary, truth, material flow, failure/recovery behavior, or rollout mechanism | [System / Integration Design](spec-first-workflow/phases/system-integration-design.md) |
| Package/file ownership is not mechanically forced by current code and accepted design | [Go Code / Ownership Design](spec-first-workflow/phases/go-code-ownership-design.md) |
| Material proof obligations, oracles, or proof levels are non-obvious | [Test Design](spec-first-workflow/phases/test-design.md) |
| Structured work lacks one ready fixed unit or the smallest dependency-ordered ledger | [Planning](spec-first-workflow/phases/planning.md) |
| One accepted implementation unit is ready and authorized | [Implementation](spec-first-workflow/phases/implementation.md) |

Specification and Planning are required for structured work unless ready
equivalent authority already supplies their output. Other phases are
conditional. Load only the selected owner; do not load a phase merely to
declare it unnecessary. Record `skipped` only when a durable
[Phase Result V1](spec-first-workflow/interfaces/phase-result-v1.md) needs that
disposition.

## Conditional Owners

Load a conditional owner immediately before its trigger:

| Trigger | Owner |
| --- | --- |
| Decide whether or what to persist | [Artifacts](spec-first-workflow/shared/artifacts.md) |
| Interpret or update artifact status | [Artifact Lifecycle V1](spec-first-workflow/interfaces/artifact-lifecycle-v1.md) |
| Dispatch a non-implementation read-only lane | [Delegation](spec-first-workflow/shared/delegation.md) |
| Decide or run independent review | [Review](spec-first-workflow/shared/review.md) |
| Resume after interruption or actor/session change | [Resume](spec-first-workflow/shared/resume.md) |
| Cross an actor or macro-phase boundary | [Handoff](spec-first-workflow/shared/handoff.md) |
| Close a completed task bundle | [Cleanup](spec-first-workflow/shared/cleanup.md) |
| Move or reopen a macro phase | [Phase Movement](spec-first-workflow/shared/phase-movement.md) |
| Choose or operate a durable control, carrier, model, or effort | [Agent Harness](agent-harness.md) |

### Implementation-Input Closure

Implementation reopens the smallest owner when a required input is unavailable
from its named authority.
