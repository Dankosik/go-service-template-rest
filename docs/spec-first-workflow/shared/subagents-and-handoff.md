# Subagents And Handoff

Use a separate read-only lane only when it changes a decision, evidence boundary, independence, or resume outcome. Keep everything else local.

## Read When

- One bounded research or review question benefits from independent context.
- A high-impact, hard-to-reverse, cross-owner, or explicitly requested review needs independence.
- Another actor or session genuinely needs a compact handoff.

## Inputs

- One question or fixed candidate revision.
- The smallest source/artifact bundle that can answer it.
- Clear read-only, authority, and external-action boundaries.

## Outputs

- A conclusion, strongest evidence, uncertainty, and root disposition.
- A compact handoff only when another actor or session is needed.

## Delegation

Delegate only a concrete, bounded, independently answerable question whose separate context or independence materially improves the result. Do not delegate tiny lookups, tightly coupled reasoning, broad “review everything,” duplicate lenses, or implementation repair. A skill name or domain label is not a lane.

Use at most three useful concurrent lanes and no nested delegation. Built-in subagents are read-only; they do not edit, accept, or claim completion. Local direct implementation needs no lane. An App Worker is optional execution isolation under the implementation phase, not a built-in review lane.

## Independent Review

One whole-artifact reviewer is enough when independent review is triggered. Trigger it only for an explicit request or a high-impact, hard-to-reverse, cross-owner, or weakly falsifiable fixed decision. The reviewer anchors findings, states the evidence boundary, and classifies each affected lens as covered, delegated to concrete evidence, or not triggered with a reason.

The owning author repairs findings. Re-review only the repair and transitively affected lenses. `PASS`, `CONCERNS`, and `FAIL` are evidence summaries: move when the applicable decision and proof are closed, not because an untriggered review lacks a particular word. Do not add a default challenger, a second reviewer, or a review receipt for confidence.

Explicit user grilling remains a root-to-user dialogue through the `grilling` skill. It is not an internal workflow gate.

## Fan-In

Keep only the conclusion, strongest evidence, uncertainty, root disposition, and destination owner. Do not paste transcripts into authoritative artifacts.

## Resume

Resume from the smallest current source: `tasks.md` only when an active ledger exists, otherwise `workflow-plan.md` only for real multi-session coordination, then the artifact named by the next action. Inspect workspace and Git drift first. If sources conflict, reopen the narrowest owner.

## Handoff

Hand off only when the current session cannot safely finish. Name the accepted source, next outcome, authority boundary, proof, first action, and stop/reopen owner. Keep the prompt outcome-first and omit workflow manuals, model catalogs, and repeated repository rules.

## Stop Rule

Do not create a lane or handoff when local work can safely finish. Stop a reviewer after its fixed evidence boundary is answered; stop a handoff when the receiver can act without chat archaeology.
