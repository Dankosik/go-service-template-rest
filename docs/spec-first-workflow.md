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
| Definition | Intake, supporting or standalone Research, Specification, required review of the fixed macro result |
| Technical Design | supporting Research, System / Integration Design, Go Code / Ownership Design, required Technical Design Review |
| Planning | Planning and required Task Review / Readiness |
| Implementation | Parallel implementation, then final validation and any required delivery review |

Research is a standalone macro phase only when the accepted boundary is
`research only`; otherwise it supports the active phase. Review, repair, and
the smallest upstream reopen remain inside the active macro phase. Each phase
actor owns at most one macro phase. For a request spanning phases, the current
root retains responsibility as the continuation coordinator and delegates the
selected phase to a fresh actor through the authorized harness. The coordinator
routes results; it does not perform phase work or accept implementation units.
[Transition](spec-first-workflow/shared/transition.md) owns continuation,
including before an Implementation ledger exists. An explicit phase-only or
research-only request stops at that reviewed boundary.

Test cases, fixtures, assertions, and test commands are chosen by the executor
while implementing each task. They require no preceding phase, approved test
plan, or separate review. Implementation's final-validation boundary owns when
the tests run.

Traverse applicable macro phases in table order. Skip one only when Phase
Selection evidence shows that its decision is already closed or untriggered;
never fold its open decision into a neighboring phase or the same phase actor.

## Phase Selection

Evaluate these triggers before loading a phase owner:

| Observable trigger | Owner |
| --- | --- |
| New structured work lacks a complete [Intent](spec-first-workflow/interfaces/intent.md), or outcome, scope, authority, observable success, or first owner is ambiguous | [Intake](spec-first-workflow/phases/intake.md) |
| Current or external evidence can change a named decision | [Research](spec-first-workflow/phases/research.md) |
| Structured work lacks a ready behavior delta | [Specification](spec-first-workflow/phases/specification.md) |
| Implementation would otherwise choose a runtime boundary, truth, material flow, failure/recovery behavior, or rollout mechanism | [System / Integration Design](spec-first-workflow/phases/system-integration-design.md) |
| Package/file ownership is not mechanically forced by current code and accepted design | [Go Code / Ownership Design](spec-first-workflow/phases/go-code-ownership-design.md) |
| Structured work lacks one ready fixed unit or the smallest dependency-ordered ledger | [Planning](spec-first-workflow/phases/planning.md) |
| One accepted implementation unit is ready and authorized | [Implementation](spec-first-workflow/phases/implementation.md) |

Intake is required for new structured work unless a complete Intent already
exists. Specification and Planning are required for structured work unless
ready equivalent authority already supplies their output. Do not backfill
active work whose accepted upstream authority predates Intent: that authority
remains its requester-meaning input, including on a narrow reopen. Missing or
changed requester meaning still reopens Intake. Other phases are conditional.
Load only the selected owner; do not load a phase merely to
declare it unnecessary. Record `skipped` only when a durable [Transition
result](spec-first-workflow/shared/transition.md) needs that disposition.

## Conditional Owners

Load a conditional owner immediately before its trigger:

| Trigger | Owner |
| --- | --- |
| Decide whether or what to persist | [Artifacts](spec-first-workflow/shared/artifacts.md) |
| Interpret or update artifact status | [Artifact Lifecycle V1](spec-first-workflow/interfaces/artifact-lifecycle-v1.md) |
| Dispatch a non-implementation read-only lane | [Read-Only Delegation](spec-first-workflow/shared/read-only-delegation.md) |
| Decide or run independent review | [Review](spec-first-workflow/shared/review.md) |
| Resume after interruption or actor/session change | [Resume](spec-first-workflow/shared/resume.md) |
| Move, reopen, or cross an actor/session/macro-phase boundary | [Transition](spec-first-workflow/shared/transition.md) |
| Close a completed task bundle | [Cleanup](spec-first-workflow/shared/cleanup.md) |

### Implementation-Input Closure

Implementation reopens the smallest owner when a required input is unavailable
from its named authority.
