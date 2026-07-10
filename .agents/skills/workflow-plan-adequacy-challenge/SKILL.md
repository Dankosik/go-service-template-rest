---
name: workflow-plan-adequacy-challenge
description: "Read-only falsification of recorded workflow routing and handoff adequacy when a canonical ADEQUACY-* trigger is true. Test the existing route and control artifacts; never classify, reclassify, edit, or approve state."
---

# Workflow Plan Adequacy Challenge

## Eligibility And Outcome

Use only when the canonical artifact model says formal adequacy is required: full shape, true or approval-relevant unknown FULL-* evidence, durable workflow-planning control, or a reclassification that could invalidate a prior challenged route.

The outcome is an independent falsification report over the recorded route and handoff. It is advisory evidence for orchestrator reconciliation, not a workflow phase or approval.

## Canonical Owners

- [AGENTS.md](../../../AGENTS.md) owns SHAPE-*, FULL-*, agent-request, direct-writer, and ADEQUACY-* policy.
- [Artifact model](../../../docs/spec-first-workflow/shared/artifact-model.md) owns routing identity, transitions, typed state, artifact expectations, phase-control eligibility, and guarded reclassification.
- [Subagents and handoff](../../../docs/spec-first-workflow/shared/subagents-and-handoff.md) owns subagent gates, resume order, and handoff rendering.
- The active phase file selected by the [workflow router](../../../docs/spec-first-workflow.md) owns phase-local entry, output, stop, and reopen behavior.

Load at most one bundled rubric by default, selected by the suspected gap: master/phase consistency, artifact status, lane ownership, stop boundary, planning readiness, finding severity, or duplicated authority. Load another only for a genuinely independent gap.

## Side Effects And Boundaries

This skill is read-only. Do not edit files, classify or reclassify shape, run a transition, approve readiness, choose product/design/test/planning decisions, create artifacts, change the ledger, or mutate git.

Do not demand more prose or artifacts merely because a valid route is compact. Do not turn phase control into a second spec, design bundle, test plan, or task ledger.

## Required Bundle

Require only the recorded inputs needed to test the active ADEQUACY-* reason:

- accepted brief and current route audit, including matched SHAPE-* rule, agent_request, actor, proof obligation, and decisive trigger evidence;
- current routing identity, validity, transition/reclassification history, and applicable workflow-control files;
- typed artifact/gate/verdict/handoff state consumed from canonical owners;
- the active phase, stop rule, blocker/reopen state, next action, and context bundle;
- lane/audit summaries when a required gate depends on them.

If a decisive input is missing, return an input-gap finding instead of reconstructing state.

## Unique Method

1. Re-evaluate the recorded route against the canonical first-match rules without writing a replacement classification.
2. Test that routing identity and validity agree across current carriers and that any transition or reclassification used the owning guarded transaction.
3. Test only the phase-local fields and gates required by the active canonical owner, including subagent disposition, proof obligations, stop boundary, and actionable reopen route.
4. Reject duplicated authority, stale approvals, hidden blockers, unsupported local-only reasoning, and handoffs that require chat archaeology.
5. Preserve concise valid control. Findings must identify the exact missing or contradictory evidence and the smallest orchestrator repair.

## Deliverable

Return:

- Adequacy Summary;
- Matrix Falsification: recorded rule, decisive evidence, ADEQUACY-* reason, routing identity/validity, and whether the route survived;
- Findings;
- Handoff Recommendation;
- Confidence and evidence boundary.

Each finding contains Gap, Why It Matters, What Could Fail, Evidence, one classification, one recommended action, and the exact orchestrator addition:

- classifications: `blocks_phase_handoff`, `blocks_specific_lane`, `non_blocking_but_record`;
- actions: `add_missing_routing`, `clarify_artifact_status`, `clarify_readiness_status`, `clarify_lane_ownership`, `clarify_stop_or_completion_rule`, `record_blocker_or_reopen`, `record_skip_or_accepted_risk`, `trim_duplicate_authority`, `no_action`.

Never say the plan is approved. If no finding survives, say no blocking adequacy gap was found within the named evidence boundary.

## Success, Blocked Stop, And Reopen

Success means the recorded route survived or every material falsification has actionable evidence, impact classification, and the smallest repair/reopen owner.

Stop blocked on a decisive input gap, stale/conflicting routing identity, unresolved required lane, or missing owner evidence. The orchestrator reopens workflow planning for route/control repair or the active phase for phase-local repair; this challenger never performs the repair.
