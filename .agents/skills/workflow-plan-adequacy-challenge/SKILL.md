---
name: workflow-plan-adequacy-challenge
description: "Handoff integrity: Use when a workflow-plan.md has high-impact or contested routing/handoffs, or the user explicitly requests a challenge. Own a read-only, anchored gap result and next owner; Skip when the plan is ordinary and low-risk or when authoring or state changes are requested."
---

# Workflow Plan Adequacy Challenge

Must read [the Artifact Model](../../../docs/spec-first-workflow/shared/artifact-model.md); it owns workflow-plan persistence, status, ownership, and resume context. Reconstruct every transition and handoff from the fixed workflow plan plus its canonical artifacts and current status, then audit the chain of custody read-only so each preserves owner, input artifact/evidence, next action, output checkpoint, and reopen condition. Give each link an anchored finding, evidence-backed no finding, outside-boundary, or proof-blocked disposition. A missing coordination rule ends with a named workflow-planning Decision handoff; adequacy Review restarts separately after acceptance. Otherwise return the complete chain result when the next actor can proceed without chat archaeology.
