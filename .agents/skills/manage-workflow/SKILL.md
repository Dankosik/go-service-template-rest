---
name: manage-workflow
description: "Workflow loading: Use for path/phase selection, movement/resume, or before change/build/fix. Own the current required read set and next owner; Skip when AGENTS.md fully owns the answer."
---

# Manage Workflow

Reconstruct the accepted outcome, authorization, repository state, and active
task artifacts. A direct `change`, `build`, or `fix` reads
[Implementation](../../../docs/spec-first-workflow/phases/implementation-validation-closeout.md)
before editing. Read the [workflow router](../../../docs/spec-first-workflow.md)
when the path or next owner is open or the route is structured/orchestrated.

Select one current owner and read only its instruction:

- raw or interpretation-sensitive input: [Intake](../../../docs/spec-first-workflow/phases/intake.md)
- decision-changing evidence: [Research](../../../docs/spec-first-workflow/phases/research.md)
- behavior and invariants: [Specification](../../../docs/spec-first-workflow/phases/specification.md)
- runtime, contracts, data, failure, or rollout: [System Design](../../../docs/spec-first-workflow/phases/system-integration-design.md)
- Go placement and dependencies: [Go Ownership](../../../docs/spec-first-workflow/phases/go-code-ownership-design.md)
- non-obvious proof: [Test Design](../../../docs/spec-first-workflow/phases/test-design.md)
- execution order and owners: [Planning](../../../docs/spec-first-workflow/phases/planning.md)
- authorized change and proof: [Implementation](../../../docs/spec-first-workflow/phases/implementation-validation-closeout.md)

Load the [Artifact Model](../../../docs/spec-first-workflow/shared/artifact-model.md)
only for persistence or resume, [Subagents and Review](../../../docs/spec-first-workflow/shared/subagents-and-handoff.md)
only for non-implementation delegation or review, [Review Independence](../../../docs/spec-first-workflow/shared/review-independence.md)
only to decide a fixed review boundary, [Implementation Review](../../../docs/spec-first-workflow/shared/implementation-review.md)
only for its fixed-unit branch, [Resume and Handoff](../../../docs/spec-first-workflow/shared/resume-and-handoff.md)
only for context rollover or a real actor or macro-phase boundary, and [Agent
Harness](../../../docs/agent-harness.md) only for a native control.
Apply independently triggered domain skills directly.

The read set is complete only when every file required before the next governed
action has actually been read; a link or remembered summary does not close it.
Re-evaluate it when evidence changes the phase, risk, ownership, proof, or
harness control.

Finish only at the current phase stop rule. Apply the router's macro-phase
session boundary before movement: continue automatically only inside the active
macro phase; at its boundary return the result and handoff, then stop. Otherwise
return the blocker or named boundary. New evidence reopens the smallest owner.
Report the outcome, proof or gap, and next owner without a routing receipt.
