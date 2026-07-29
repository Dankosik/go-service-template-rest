---
name: manage-workflow
description: "Workflow: Use when repository work needs path or phase selection, movement across research, specification, technical design, test design, planning, implementation, or resume from task artifacts. Own the smallest valid path, current decision owner, exact instruction loading, transition, and reopen decision; Skip when a direct task or an already-owned domain action can proceed without workflow routing."
---

# Manage Workflow

Reconstruct the accepted outcome, authorization, repository state, and active
task artifacts. Read the [workflow router](../../../docs/spec-first-workflow.md)
when the path or next owner is open.

Select one current owner and read only its instruction:

- decision-changing evidence: [Research](../../../docs/spec-first-workflow/phases/research.md)
- behavior and invariants: [Specification](../../../docs/spec-first-workflow/phases/specification.md)
- runtime, contracts, data, failure, or rollout: [System Design](../../../docs/spec-first-workflow/phases/system-integration-design.md)
- Go placement and dependencies: [Go Ownership](../../../docs/spec-first-workflow/phases/go-code-ownership-design.md)
- non-obvious proof: [Test Design](../../../docs/spec-first-workflow/phases/test-design.md)
- execution order and owners: [Planning](../../../docs/spec-first-workflow/phases/planning.md)
- authorized change and proof: [Implementation](../../../docs/spec-first-workflow/phases/implementation-validation-closeout.md)

Load the [Artifact Model](../../../docs/spec-first-workflow/shared/artifact-model.md)
only for persistence or resume, [Subagents and Handoff](../../../docs/spec-first-workflow/shared/subagents-and-handoff.md)
only for triggered review, delegation, or handoff, and
[Agent Harness](../../../docs/agent-harness.md) only for a native control.
Apply independently triggered domain skills directly.

Finish only at the phase stop rule. Move when authorized inputs are closed;
otherwise return the blocker or named boundary. New evidence reopens the
smallest owner. Report the outcome, proof or gap, and next owner without a
routing receipt.
