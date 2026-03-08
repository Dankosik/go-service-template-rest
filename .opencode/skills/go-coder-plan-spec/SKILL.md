---
name: go-coder-plan-spec
description: "Turn approved Go requirements and specs into the explicit implementation plan that unblocks coding. Use whenever a Go task is past design/synthesis and needs `Implementation Plan` content for `spec.md` or `plan.md`, including when the user asks to break a spec into tasks, sequence work, map validation, add checkpoints, or prepare a handoff to `go-coder`, even if they do not explicitly ask for a 'planning' skill. Skip unresolved architecture/API/data/security decisions and skip actual coding."
---

# Go Coder Plan Spec

## Purpose
Turn approved Go change intent into a compact, execution-ready implementation plan that downstream coding can follow without rediscovering scope, blockers, sequencing, or proof obligations.

## Scope And Boundaries
In scope:
- convert approved decisions, requirements, invariants, and constraints into dependency-ordered implementation work
- make the `Implementation Plan` explicit in `spec.md` or `plan.md`
- map each meaningful step to traceability, expected change surface, and validation
- expose blockers, assumptions, and decision dependencies before coding starts
- preserve coder autonomy on local decomposition, file layout, and low-level implementation shape

Out of scope:
- making new architecture, API, data, security, reliability, or rollout decisions
- writing production code, tests, migrations, or review comments as the main deliverable
- forcing workflow theater, mandatory subagent choreography, or one rigid schema for every task size
- freezing exact file paths, function names, or micro-order unless they are real constraints

## Workflow Compatibility
- This skill is a planning aid, not the top-level workflow manager.
- Use it only in planning, after intent is stable enough to decompose and before implementation starts.
- Treat `spec.md` as the canonical decisions artifact; put the plan there unless a separate `plan.md` is clearer because the work is long, parallelized, or noisy.
- Keep the plan compatible with downstream `go-coder`, review skills, and validation work.
- If implementation later reveals a real design gap or touched scope drifts beyond approved intent, reopen planning instead of silently improvising a new plan in code.
- Do not duplicate specialist decision narratives; reference approved decisions and translate them into executable work.

## Context Loading
- Load the smallest sufficient set of approved artifacts: canonical spec sections, issue text, acceptance criteria, constraints, and the relevant implementation surface.
- Prefer canonical artifacts over chat paraphrases when both exist.
- If a critical fact is missing, mark it as `[assumption]` or block the affected work instead of inventing it.
- If design intent is still moving, say planning is premature for the affected slice.

## Working Method
1. Frame the implementation scope, non-goals, constraints, and risky unknowns.
2. Confirm which decisions are already approved and which gaps must stay blocked.
3. Slice the work by dependency and observable outcome, not by coding style or arbitrary file boundaries.
4. Attach each meaningful slice to traceability and smallest-sufficient validation.
5. Add checkpoints only where they reduce execution or review risk.
6. End with explicit blockers, assumptions, and execution handoff notes.

## Planning Defaults

### Plan Depth Scales With Task Risk
- For small or low-risk work, a short numbered plan plus validation notes is enough.
- For medium or high-risk work, use task cards or a compact table only when they add clarity.
- Do not force heavyweight matrices, per-task status fields, or formal contracts when a simpler plan is clearer.

### Task Design
- Each task should land one observable behavior change or one tightly coupled implementation slice.
- Keep tasks small enough that completion can be proven without immediately running the whole repository unless integration risk truly requires it.
- Separate enabling or refactoring work from behavior changes when that improves verification and rollback clarity.
- If two changes must land together to remain safe, say why they are coupled instead of pretending they are independent.

### Sequencing
- Order tasks by dependency, risk retirement, and feedback speed.
- Front-load tasks that unblock other work, expose risky assumptions, or establish cross-cutting scaffolding.
- Avoid sequences based on function-by-function trivia or guessed implementation aesthetics.

### Traceability
For each meaningful task or task group, carry forward:
- source requirement, approved decision, or invariant
- affected change surface by package, layer, or module instead of rigid file lock-in
- validation obligations and expected proof of completion

Maintain a clean closure path:
- approved intent -> implementation step -> validation -> observed evidence

### Validation And Evidence
- Define the smallest sufficient command set that can honestly prove the planned slice.
- Call out stronger checks when the risk surface demands them: contract tests, generated-code drift checks, migration checks, race checks, or compatibility checks.
- Prefer observable evidence over vague `verify manually` phrasing.
- If a step intentionally defers some validation until a later checkpoint, say what remains open and why.

### Checkpoints
- Add checkpoints at natural risk boundaries, not by ritual cadence.
- A checkpoint should say what must be true before proceeding: completed validations, drift check, blocker resolution, or review-ready state.
- Small plans may need one final checkpoint only; larger plans may need multiple grouped checkpoints.

### Blockers, Assumptions, And Clarifications
- When a task cannot proceed safely, mark it blocked with the missing decision or fact, why it matters, and what unblocks it.
- Use plain language first; do not force request IDs or ticket-shaped metadata unless an external workflow actually needs them.
- Distinguish:
  - `[blocked]` for missing decisions or facts
  - `[assumption]` for bounded temporary premises
  - `[follow-up]` for non-blocking later work

### Coder Autonomy
- The plan governs outcomes, dependencies, guardrails, and proof, not exact code shape.
- Leave room for the coder to choose local refactors, helper extraction, file movement, and implementation order inside a task when that does not violate approved intent.
- Only lock down exact paths, APIs, or sequence when correctness, generated artifacts, migration order, or operational safety requires it.

### Handoffs And Reopen Conditions
- Name adjacent specialist follow-ups only when a real unresolved question remains.
- If a task should pause for a new architecture, API, data, security, or reliability decision, say that explicitly instead of filling the gap with planning guesses.
- Make the coding handoff explicit when it matters: what `go-coder` can execute now, what must stay aligned with `spec.md` or `plan.md`, and which conditions require returning to planning.
- Record reopen conditions for high-risk assumptions that could invalidate later steps.

## Deliverable Shape
Return plan text that can drop directly into `Implementation Plan` in `spec.md` or into `plan.md` with minimal rewriting.

Use the smallest structure that stays clear. A strong default is:
- `Execution Context`
- `Implementation Plan`
- `Validation Plan`
- `Blockers / Assumptions`
- `Checkpoint Plan` only when the work is large enough to need it
- `Handoffs / Reopen Conditions` only when they materially matter

For medium or large work, task cards should usually include:
- `Task ID`
- `Objective`
- `Depends On` when nontrivial
- `Traceability`
- `Change Surface`
- `Planned Verification`
- `Done Evidence`
- `Ambiguity / Stop Conditions`

## Definition Of Done
The planning pass is complete when:
- implementation is no longer blocked by avoidable ambiguity
- the plan is explicit in `spec.md` or `plan.md`
- tasks are sequenced by dependency and observable outcome
- validation expectations match the real risk surface
- blockers, assumptions, and reopen conditions are visible
- the plan is compact enough for `go-coder` to execute without re-planning the whole task

## Escalate Or Reject
- intent is still unstable or contradictory
- the next steps depend on unresolved architecture, contract, data, security, reliability, or invariant decisions
- the work cannot be decomposed without pretending uncertain decisions are already made
- the requested plan format would create fake precision or duplicate the canonical spec
- implementation would start without an explicit plan

## Anti-Patterns
Do not:
- act as the workflow orchestrator instead of a planning aid
- dump raw research or repeated decision narratives into the plan
- force every plan into identical task-card or matrix ceremony
- prescribe exact low-level coding moves when the constraint is not real
- hide uncertainty inside optimistic task wording
- separate plan and validation so far that completion proof becomes ambiguous
