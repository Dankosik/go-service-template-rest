# Implementation Obstacle Recovery

## Read When

Read only after a leaf returns `NEEDS_PARENT` or current evidence invalidates an
accepted input during Implementation. The [Role
Tree](implementation-worker-execution.md#execution-role-tree) owns authority.

## Bottom-Up Resolution

| Current actor | Resolution boundary | Result |
| --- | --- | --- |
| Specialist, Worker, or Reviewer | Acceptance-Unit Lead | `NEEDS_PARENT` |
| Acceptance-Unit Lead | Same structured root or Ledger Orchestrator | Internal reopen or canonical `Blocked:` transition |
| Upstream Reopen Lead | Ledger Orchestrator | Review-cleared phase result or exact boundary |
| Ledger Orchestrator | AGENTS-owned boundary | Outcome stop |

An obstacle is not a blocker while an evidence-changing remedy remains inside
the actor's authority. Diagnose it, take the narrowest eligible remedy, and do
not repeat a route under unchanged inputs and hypothesis. A leaf that exhausts
its remedies returns evidence, attempted actions, the boundary, and one
parent-owned action through `NEEDS_PARENT`.

The Lead re-diagnoses a leaf return and may close a unit-level decision, change
the execution strategy, obtain evidence, integrate a valid candidate, or route
a same-Worker correction. It records one canonical blocker only after safe
unit-local routes and applicable upstream recovery are exhausted. The blocker
names evidence, boundary, reopen owner and condition, and preserved candidate;
it is not an attempt transcript.

## Upstream Reopen

An agent-owned reopen is internal recovery. A structured root suspends its Lead
role, closes the smallest owning phase, completes that phase's review loop, and
then resumes the same unit. An orchestrated Lead preserves its candidate and
returns the boundary to the Ledger Orchestrator, which loads [Artifact
Model](../shared/artifact-model.md), [Resume And
Handoff](../shared/resume-and-handoff.md), [Implementation
Handoff](../shared/implementation-handoff.md), and the selected harness recovery
branch. If recovery creates a prerequisite unit, accept that unit before
resuming the blocked one. Planning changes ledger scope or dependencies only
through a separate Planning reopen.
