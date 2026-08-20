# Spec-First Workflow

Router for selecting the smallest execution path and next owner. `AGENTS.md`
owns authority and global invariants; phase and shared files own their methods.

## Choose A Path

Use [Direct Work](spec-first-workflow/direct-work.md) for a local, reversible,
single-owner outcome with bounded proof and no unresolved protected decision.

Use `structured` for a non-trivial outcome whose decisions or proof must survive.
Persist only artifacts needed across an actor or session boundary. Use
`orchestrated` only when coordination is itself material: broad or multi-owner
scope, hard-to-reverse decisions, conflicting evidence, explicit multi-agent
work, dirty-checkout isolation, separate context, or likely multi-session work.

A bounded read-only lane does not by itself make work orchestrated. Re-evaluate
the path only when evidence changes risk, ownership, reversibility, or proof; a
path is not a quality tier.

## Required Spine

Structured and orchestrated work visits these owners in order. A phase may
record that a conditional artifact is unnecessary, but Specification and
Planning remain required.

| Need | Owner | Required outcome |
| --- | --- | --- |
| Clarify interpretation-sensitive input. | [Intake](spec-first-workflow/phases/intake.md) | Routing-sufficient brief or one blocking question. |
| Resolve decision-changing evidence. | [Research](spec-first-workflow/phases/research.md) | Supported findings, limits, conflicts, and implications. |
| Make behavior falsifiable. | [Specification](spec-first-workflow/phases/specification.md) | Ready behavior without an unresolved product choice. |
| Select runtime boundaries and interactions when implementation would choose them. | [System / Integration Design](spec-first-workflow/phases/system-integration-design.md) | Closed components, truth, flows, failures, and rollout. |
| Place the design in Go when ownership is not already forced. | [Go Code / Ownership Design](spec-first-workflow/phases/go-code-ownership-design.md) | Package, file, dependency, composition, cleanup, and proof owners. |
| Define non-obvious proof. | [Test Design](spec-first-workflow/phases/test-design.md) | Risks, observables, proof levels, and commands. |
| Make execution ready. | [Planning](spec-first-workflow/phases/planning.md) | One fixed unit or the smallest dependency-ordered ledger. |
| Change and close the outcome. | [Implementation / Validation / Closeout](spec-first-workflow/phases/implementation-validation-closeout.md) | Working change and evidence-clamped completion. |

One root session owns at most one macro phase. Supporting work, review, repair,
and the smallest upstream reopen stay inside it. A different macro phase starts
from [Resume And Macro-Phase Handoff](spec-first-workflow/shared/resume-and-handoff.md#macro-phase-handoff).

## Conditional Owner Router

Load a conditional owner immediately before the first action in its row.

| Trigger | Owner |
| --- | --- |
| Persist, inspect, or resume task artifacts. | [Artifact Model](spec-first-workflow/shared/artifact-model.md) |
| Dispatch a non-implementation read-only lane. | [Read-Only Lanes](spec-first-workflow/shared/subagents-and-handoff.md) |
| Decide or run an independent review. | [Review Independence](spec-first-workflow/shared/review-independence.md) |
| Resume after interruption or cross an actor or macro-phase boundary. | [Resume And Macro-Phase Handoff](spec-first-workflow/shared/resume-and-handoff.md) |
| Enter or continue Implementation across a session, terminalize a Lead, or route an upstream reopen. | [Implementation Handoff](spec-first-workflow/shared/implementation-handoff.md) |
| Choose or operate a durable control, carrier, model, or reasoning effort. | [Agent Harness](agent-harness.md) |

Re-run routing only when phase movement or current evidence activates a new row.

[Review Independence](spec-first-workflow/shared/review-independence.md#review-router)
owns the phase-specific review map, trigger, and return path. [Planning
obligation reconciliation](spec-first-workflow/phases/planning.md#obligation-reconciliation)
owns implementation input closure.

### Implementation-Input Closure

Implementation reopens the smallest owner when a required input is unavailable
from its named authority.

## Phase Movement

Move forward only when the current owner has dispositioned every triggered
decision and the next phase can act without inventing meaning, mechanism,
ownership, or proof strategy. Reopen only the smallest owner invalidated by new
evidence and preserve unaffected dispositions.

Continue inside the active macro phase unless the user named the boundary, a
required external input is unavailable, the next action needs new authority, or
durable resume state is required. Only a completed, review-cleared macro phase
emits the next-session handoff.
