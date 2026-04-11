# Planning Handoff And Stop Rules

Use this file before marking a technical-design session complete. The handoff tells the next planning session what it may consume; it does not begin planning.

## When To Load
- Load before setting `Session boundary reached: yes`.
- Load before claiming the design bundle is planning-ready.
- Load when the user asks to continue into `plan.md`, `tasks.md`, implementation, tests, migrations, contract generation, or review.
- Load when deciding whether to block, route back to specification, or hand off to planning.

## Good Design-Session Outputs
- "Planning handoff: approved `spec.md`; approved required design artifacts; triggered `design/data-model.md` and `design/contracts/`; `rollout.md` not expected; unresolved assumptions listed; workflow files agree next session starts with `planning`."
- "Stop rule: do not write `plan.md`, `tasks.md`, code, tests, migrations, generated files, or review output in this session."
- "Blocked handoff: sequence depends on an unresolved event durability decision in `spec.md`; route next session to `specification` and do not mark planning-ready."
- "Accepted assumption: no schema migration is expected because the approved spec keeps existing persisted state unchanged; record `design/data-model.md: not expected`."

## Bad Design-Session Outputs
- Treating a clear handoff as permission to begin planning in the same session.
- Treating a small design as implementation-ready without planning readiness.
- Leaving a missing source-of-truth owner for the planning session to guess.
- Adding next-phase TODO placeholders instead of stopping with a clean handoff or blocker.

## Conditional Artifact Examples
- Planning handoff includes `design/data-model.md` when schema or persisted state changes; otherwise it records the artifact as not expected.
- Planning handoff includes `design/dependency-graph.md` when package direction or coupling risk is part of the design.
- Planning handoff includes `design/contracts/` when contract semantics change, with a note that runtime authorities remain canonical.
- Planning handoff includes `test-plan.md` or `rollout.md` when triggered, or records them as not expected in workflow control.

## Blocked Handoff Examples
- Required design artifacts are missing, stale, or internally contradictory.
- A triggered conditional artifact is missing or only a placeholder.
- A planning-critical open question remains that could change correctness, ownership, rollout, or validation.
- The current session would need upstream decisions or execution sequencing to move forward; stop and route to the right phase instead.

## Exa Source Links
Exa MCP search and fetch were attempted before writing these examples, but the provider returned a 402 credits-limit error. Treat these fallback-verified links as calibration only; repo-local files remain authoritative.

- Google Cloud Architecture decision records: https://docs.cloud.google.com/architecture/architecture-decision-records
- Google Cloud Architecture Framework: https://docs.cloud.google.com/architecture/framework
- Azure Well-Architected Framework: https://learn.microsoft.com/en-us/azure/well-architected/what-is-well-architected-framework
- C4 model: https://c4model.com/
