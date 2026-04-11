# Technical Design Entry Readiness

Use these examples after reading repo-local authority: `AGENTS.md`, `docs/spec-first-workflow.md`, task-local `workflow-plan.md`, task-local `workflow-plans/technical-design.md`, and approved `spec.md`. Examples are fragments, not templates.

## When To Load
- Load before starting or resuming a technical-design session.
- Load when `spec.md` is present but you need to verify whether it is approved, planning-stable, or blocked.
- Load when the current phase or allowed write surface is unclear.
- Load when the user asks to combine design with planning, coding, tests, migrations, or review.

## Good Design-Session Outputs
- "Entry readiness: `spec.md` is approved, the clarification gate is recorded as resolved, current phase is `technical-design`, and `workflow-plans/technical-design.md` is active. This session may update `design/`, `workflow-plan.md`, and `workflow-plans/technical-design.md` only."
- "The design pass will load `docs/repo-architecture.md` because the change touches HTTP transport, app orchestration, persistence, and generated contracts."
- "The design pass is blocked: `spec.md` still disagrees on durable completion semantics. Record the blocker and route to `specification`; do not draft a design bundle to hide the contradiction."
- "Design already approved and workflow control says next session starts with `planning`; stop and hand off instead of reworking the design."

## Bad Design-Session Outputs
- Treating a nearly approved spec as permission to rewrite decisions during design.
- Treating obvious design choices as permission to enter planning in the same session.
- Leaving workflow-control updates only in chat.
- Starting runtime artifact edits, generation, tests, migrations, or review execution during technical design.

## Conditional Artifact Examples
- Create `design/data-model.md` when the approved spec adds persisted job state, replay behavior, schema changes, or cache correctness rules.
- Create `design/contracts/` when the approved spec changes OpenAPI, events, generated contracts, or material internal interfaces. Keep runtime authority in the canonical contract source.
- Create `design/dependency-graph.md` when new packages or generated-code flow could create coupling risk.
- Create `test-plan.md` only when validation spans enough layers that it would bloat `plan.md`.
- Create `rollout.md` only when migration order, mixed-version behavior, backfill/verify choreography, or failback is planning-critical.
- Record "not expected" instead of creating placeholder conditional artifacts when no trigger exists.

## Blocked Handoff Examples
- `spec.md` is draft, contradictory, or missing clarification-gate resolution, so technical design cannot start honestly.
- `workflow-plan.md` says `Current phase: planning` and there is no recorded reopen target for technical design.
- Repository boundary changes are material, but `docs/repo-architecture.md` has not been loaded and current design artifacts do not already capture the stable baseline.
- The user asks to continue into planning or implementation edits in the same session; complete or block technical design, then stop at the boundary.

## Exa Source Links
Exa MCP search and fetch were attempted before writing these examples, but the provider returned a 402 credits-limit error. Treat these fallback-verified links as calibration only; repo-local files remain authoritative.

- C4 model: https://c4model.com/
- arc42 documentation: https://docs.arc42.org/home/
- Google Cloud Architecture decision records: https://docs.cloud.google.com/architecture/architecture-decision-records
- Azure Well-Architected Framework: https://learn.microsoft.com/en-us/azure/well-architected/what-is-well-architected-framework
