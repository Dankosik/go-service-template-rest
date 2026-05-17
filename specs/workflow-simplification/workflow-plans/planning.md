# Planning Phase Plan

Phase: planning
Phase status: complete for implementation handoff
Input sources:
- `workflow-plan.md`
- `workflow-plans/specification.md`
- approved `spec.md`
- `research/external-agent-workflow-practices.md`
- `research/current-workflow-pain-map.md`
- current target surfaces: `AGENTS.md`, `docs/spec-first-workflow.md`, `docs/subagent-contract.md`, `docs/subagent-brief-template.md`, `.agents/skills/`

## Approval Decision

The trigger-based simplification direction is approved for implementation planning.

Approved choices:
- keep `workflow-plan.md`; do not introduce `workflow.md`
- make `lean local` the default for bounded non-trivial single-domain work, with `lightweight local` retained as a compatibility alias
- allow inline `Risk Challenge` approval for lean local specs when no escalation trigger is present
- allow lean design answers in `spec.md` or one `design/overview.md`
- require `tasks.md` for lean implementation and for multi-slice, multi-surface, dependency-bearing, or resumable implementation; allow inline plan only for tiny direct-path work
- use delta-style `Behavior / Contract Delta` in lean specs
- make proof-first or test-first execution the default for behavior changes and bug fixes, with explicit waiver only for surfaces where a failing test is not useful
- update docs and workflow-related skills in one coordinated implementation pass

## Design-Skip Rationale

No separate `design/` bundle is created for this planning pass.

Rationale:
- The approved change is a workflow-documentation and skill-contract rewrite, not a runtime, API, data, security, reliability, or deployment behavior change.
- `spec.md` already names the affected surfaces, before/after artifact model, trigger matrix, lean spec/task shapes, inline `Risk Challenge`, preserved invariants, risks, and validation obligations.
- The implementation sequence is source-of-truth-first: update `AGENTS.md`, then detailed docs, then subagent docs, then skills, then consistency validation.
- If implementation discovers a new ownership model, artifact name, required compatibility bridge, or validation rule not covered by `spec.md`, reopen specification instead of deciding during docs or skill edits.

## Task Ledger

`tasks.md` is the executable implementation handoff.

Implementation readiness: `CONCERNS`.

Accepted concerns and proof obligations:
- Cross-surface drift risk: `AGENTS.md`, detailed workflow docs, subagent docs, and skills must describe the same direct / lean local / full orchestrated trigger model. Proof: targeted diff/read sweep plus `rtk make agents-check` and `rtk make skills-check`.
- Over-simplification risk: direct path and lean local rules must not weaken high-risk/full-orchestrated escalation triggers. Proof: trigger matrix and preserved invariants remain near the top of `AGENTS.md` and `docs/spec-first-workflow.md`, and challenge skills still require formal gates for triggered high-risk work.
- Lean-shortcut risk: `lean local` must still require canonical decisions, inline risk check, executable tasks, and proof. Proof: `docs/spec-first-workflow.md` and specification/planning skills include the lean `spec.md` and `tasks.md` shapes.
- Backward-compatibility risk: old task bundles must remain valid. Proof: implementation must avoid historical bundle rewrites and docs must state old full bundles are accepted.
- Validation-scope risk: this is docs/skills only. Proof: final validation uses docs/skill checks and `rtk git diff --check`; broader Go/runtime checks are not required unless implementation touches code or generated runtime artifacts.

## Stop Rule

Stop after creating the planning handoff. Do not edit `AGENTS.md`, workflow docs, subagent docs, or skill files in this planning session.

## Next Action

Start implementation from `tasks.md` at `T001`.
