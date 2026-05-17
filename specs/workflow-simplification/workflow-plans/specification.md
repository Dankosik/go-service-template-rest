# Specification Phase Plan

Phase: specification
Phase status: complete; proposal approved for implementation planning
Input sources:
- `workflow-plan.md`
- `workflow-plans/research.md`
- `research/external-agent-workflow-practices.md`
- `research/current-workflow-pain-map.md`
- local workflow contract and Railway auto-migrations example cited in the research notes
- user-supplied parallel audit, reconciled into `spec.md` as `lean local`, inline `Risk Challenge`, delta-style lean specs, and proof-first lean execution

## Spec Readiness

`spec.md` is approved for implementation planning because it has:
- current workflow pain points grounded in repository artifacts
- source-backed external practice synthesis
- a recommended simplified workflow
- keep, merge, remove, and conditional-trigger recommendations
- trigger matrix for direct path, lean local, and full orchestrated work
- artifact model before and after simplification
- lean `spec.md` / `tasks.md` shapes with `Behavior / Contract Delta` and inline `Risk Challenge`
- quality-preservation mechanisms and risks
- resolved approval decisions and reopen conditions

`spec.md` is approved for planning/implementation handoff. Implementation still starts from `tasks.md`, not from this phase file.

## Clarification Gate

Status: locally self-checked, not run as a formal read-only subagent gate.

Rationale:
- The original session produced a proposal, not code.
- The user approved updating the created documents from the parallel audit.
- The current approved choices are recorded in `spec.md`, and the implementation handoff now lives in `tasks.md`.

Former user-decision questions are resolved in `spec.md`. Reopen specification only if implementation needs a new artifact name, a different approval model, or a weakened escalation trigger.

## Completion Marker

Research-backed specification exists, audit additions are reconciled, and implementation planning handoff is explicit.

## Stop Rule

Specification is complete. Do not edit `AGENTS.md`, workflow docs, subagent templates, skills, or examples from this phase file; implementation must follow `tasks.md`.

## Next Action

Implementation from `tasks.md` at `T001`.
