---
name: workflow-status
description: "Read-only workflow status and next-action helper for this repository. Use when the user asks where a spec-first task stands, what the next action is, whether implementation may start, what gate is missing, or which writes are allowed in the current phase. This skill must infer state from task artifacts, ask for the task path when no active task is obvious, and never edit artifacts, code, git state, task ledgers, or implementation handoffs."
---

# Workflow Status

## Purpose
Report the current status and next action for one task-local spec-first workflow without becoming a new workflow authority.

This helper reads existing artifacts and summarizes what they already say. It does not repair the workflow, approve artifacts, create missing files, advance phases, or replace `workflow-plan.md`, `workflow-plans/<phase>.md`, `spec.md`, `design/`, or `tasks.md`.

## Outcome-First Operating Rules
- Start by naming the skill-specific outcome, success criteria, constraints, available evidence, and stop rule.
- Treat workflow steps as decision rules, not a ritual checklist. Follow exact order only when this skill or the repository contract makes the sequence an invariant.
- Use the minimum context, references, tools, and validation loops that can change the deliverable; stop expanding when the quality bar is met.
- Before acting, resolve prerequisite discovery, lookup, or artifact reads that the outcome depends on; parallelize only independent evidence gathering and synthesize before the next decision.
- Prefer bounded assumptions and local evidence over broad questioning; ask only when a missing fact would change correctness, ownership, safety, or scope.
- When evidence is missing or conflicting, retry once with a targeted strategy or label the assumption, blocker, or reopen target instead of treating absence as proof.
- Finish only when the requested deliverable is complete in the required shape and verification or a clearly named blocker/residual risk is recorded.

## Use When
- the user asks "where are we?", "what is next?", "can implementation start?", or "what is blocked?"
- a session needs a compact task handoff before deciding whether to resume, stop, or reopen an earlier phase
- the orchestrator needs to identify the current phase, artifact status, task-ledger review status, implementation-readiness status, missing gate, allowed writes, next action, or stop rule from task-local artifacts
- a task may be using direct-path or lean-local shortcuts and the helper needs to check whether the waiver, inline `Risk Challenge`, or compact design rationale is explicitly recorded

## Skip When
- the user asks you to write, repair, or advance task artifacts; use the appropriate phase/session skill instead
- no task-local path is provided and no active task path is obvious from the prompt or current working directory
- answering would require guessing from chat memory instead of reading artifacts
- the request is for domain review, task breakdown, validation proof, or code changes rather than status

## Hard Boundaries
This skill is read-only.

Do not:
- edit task artifacts, code, tests, generated files, configs, or docs
- create missing `workflow-plan.md`, `workflow-plans/`, `spec.md`, `design/`, `tasks.md`, `test-plan.md`, or `rollout.md`
- change git state, stage files, commit, push, or run mutating generation commands
- approve, reject, or rewrite the workflow plan
- treat this status report as a new phase, gate, plan, source of truth, or implementation-readiness artifact
- infer state from earlier chat memory when artifacts are missing or contradictory
- treat a missing artifact as intentionally skipped unless an explicit direct-path or lean-local waiver is present in the task artifacts or current user-provided artifact excerpt

The report may say what the current phase permits other sessions to write, but this helper itself still writes nothing.

## Read First
Always read:
- `AGENTS.md`
- `docs/spec-first-workflow.md`

Then identify exactly one task-local path.

Accept a task path only when:
- the user provides one explicitly, such as `specs/<feature-id>`
- the current working directory is already inside a single task-local path
- the prompt includes exactly one task-local artifact path that identifies the task

If no task-local path is provided and no single active task is obvious, ask for the task path and stop. Do not scan broadly and pick a task by recency.

## Artifact Read Order
Read the smallest artifact set needed to answer the status question:

1. task-local `tasks.md`, when present and the question is about implementation, review, validation, closeout, or whether execution may continue
2. task-local `workflow-plan.md`, if present and no approved ledger exists yet or the question is about a pre-code phase
3. current `workflow-plans/<phase>.md`, if the master names a current phase or next phase and no approved ledger supersedes that routing
4. task-local `spec.md`
5. task-local `workflow-plans/specification-review.md` or another recorded specification-review result when non-trivial `spec.md` exists
6. compact design in `spec.md` or task-local `design/overview.md`, then triggered split design files when split design status matters:
   - `design/component-map.md`
   - `design/sequence.md`
   - `design/ownership-map.md`
7. task-local `workflow-plans/technical-design-review.md` or another recorded technical design review result when separate design depth exists
8. optional task-local `test-plan.md`, `rollout.md`, and selected `research/*.md` only when they are present and the status question depends on them

When approved `tasks.md` exists, treat `workflow-plan.md` as historical routing, and treat `workflow-plans/*` as historical unless the ledger explicitly names a pre-created review or validation phase file. When no approved ledger exists and `workflow-plan.md` is missing, infer only the minimum state from the artifact chain and mark workflow control as incomplete unless an explicit direct-path or lean-local rationale explains the missing file.

## Status Inference Rules
- Prefer approved `tasks.md` for implementation, validation, closeout progress, and next executable task once the ledger exists.
- Prefer `workflow-plan.md` for current phase, phase status, session-boundary state, blockers, artifact status, and next-session routing only before approved `tasks.md` exists.
- Prefer the current `workflow-plans/<phase>.md` for phase-local next action, stop rule, completion marker, local blockers, and the planning-phase implementation-readiness gate result when the current or completed phase is `planning`.
- Use `spec.md` and `design/` to confirm approval signals and context. Use `tasks.md` as execution authority after approval; do not let stale master routing invent a different implementation phase.
- Treat absent required artifacts as incomplete unless an explicit waiver or trigger-based `not expected` rationale covers that exact artifact.
- Treat present artifacts with unclear approval state as `present / status unclear`, not `approved`.
- Treat a missing specification review result as incomplete for non-trivial `spec.md` unless an explicit direct-path or prototype waiver covers it.
- Treat a missing task-ledger review or implementation-readiness status as incomplete for non-trivial planned work unless an explicit eligible direct-path, lean-local, or prototype waiver covers it.
- Treat a missing technical design review result as incomplete whenever separate `design/overview.md` or split `design/` exists and no explicit compact-design/waiver rationale covers the task.
- If the master and phase-local file conflict, report the conflict as the blocker instead of choosing a winner silently.
- If `Session boundary reached: yes`, report that the next action belongs to the recorded next session or reopen target; do not continue the prior phase in the same session.
- If `Ready for next session: no`, report the active phase as still needing work unless the artifacts clearly say the master is stale.
- `tasks.md` may be read when present or expected by direct/lean/full workflow. This helper reports its status but must not create, repair, or approve `tasks.md`, the task-ledger review, or the implementation-readiness gate.

## Implementation Start Rule
Answer `Implementation may start` conservatively:

- `Yes` only when task-ledger review and readiness are `PASS`, the required artifact chain is approved or explicitly waived, there are no blocking gates, and approved `tasks.md` points to implementation or the first task.
- `Yes, in the recorded next session` when task-ledger review and readiness are `PASS`, planning is complete, `Session boundary reached: yes`, and `Next session starts with` points at implementation.
- `Yes, with recorded concerns` only when readiness is `CONCERNS`, task-ledger review is `PASS` or `CONCERNS`, named accepted risks and proof obligations are explicit, and routing points to implementation.
- `No` when task-ledger review or readiness is `FAIL`, or when `spec.md`, mandatory specification review, required compact or split design context, mandatory technical design review, expected `tasks.md`, phase control, task-ledger review status, readiness status, or a required review/validation phase file is missing without an explicit waiver.
- `No` when the current phase is workflow planning, research, specification, specification review, technical design, technical design review, planning, review, reconciliation, validation, or done and the artifacts do not route to implementation.
- `No` when task-ledger review or readiness is `CONCERNS` but accepted risks or proof obligations are unnamed.
- `Unknown` only when the task path is identified but the artifacts are too contradictory to make a safe yes or no call; name the contradiction as the blocker.

For direct-path, lean-local, compatibility `lightweight-local`, or prototype work, `WAIVED` allows implementation only if the waiver, rationale, scope, and inline tasking are explicit in the current task record. Do not infer a waiver from task size alone.

## Allowed Writes Reference
Report the phase's allowed write surface using the repository contract, while making clear that this helper writes nothing:

- `workflow planning`: `workflow-plan.md` and `workflow-plans/workflow-planning.md`
- `research`: `research/*.md`, task-local `workflow-plan.md`, and the active research phase-control file when the session owns research
- `specification`: `spec.md`, task-local `workflow-plan.md`, and `workflow-plans/specification.md`
- `specification review`: read-only review output plus task-local `workflow-plan.md` and `workflow-plans/specification-review.md`; review agents do not edit `spec.md`, `design/`, `tasks.md`, or implementation handoffs, and the orchestrator records `PASS`, `CONCERNS`, or `FAIL`
- `technical design`: compact design in `spec.md`, task-local `design/overview.md`, split `design/` core and triggered conditional design files as applicable, task-local `workflow-plan.md`, and triggered `workflow-plans/technical-design.md`
- `technical design review`: read-only review output and workflow-control updates only; review agents do not edit design artifacts, `tasks.md`, or implementation handoffs, and the orchestrator records `PASS`, `CONCERNS`, or `FAIL`
- `planning`: `tasks.md` when expected, triggered `test-plan.md` or `rollout.md`, named review/validation phase-control files when needed, task-ledger review/readiness status in task-local `workflow-plan.md`, and `workflow-plans/planning.md`
- `implementation`: code, tests, migrations, configs, generation inputs, generated outputs required by the approved task ledger, plus existing `tasks.md` progress only
- `review`: read-only review output only; no code or artifact mutation by review agents
- `reconciliation`: approved code/test/runtime fixes required by the task ledger plus existing control/checkpoint artifacts only
- `validation`: fresh verification plus ledger-owned closeout surfaces only, such as `spec.md` `Validation`/`Outcome`, existing `tasks.md` progress when used, and an existing validation phase-control file only when approved `tasks.md` explicitly names it
- `done`: no writes unless a new task or explicit reopen starts
- `unknown`: no writes until the task path and phase are clarified

## Report Shape
Keep the answer compact and use this shape unless the user asked for a narrower answer:

```text
Workflow Status
- Task path: <path or not identified>
- Current phase: <phase or unknown>
- Phase status: <status or unknown>
- Routing/task state: <done / reopened / N/A / unknown, when distinct from phase status>
- Session boundary: <reached / not reached / unknown, plus next-session signal if present>
- Artifact status: <compact list>
- Task-ledger review: <PASS / CONCERNS / FAIL / WAIVED / missing / unknown, with one reason>
- Implementation readiness: <PASS / CONCERNS / FAIL / WAIVED / missing / unknown, with one reason>
- Specification review: <PASS / CONCERNS / FAIL / missing / not expected / unknown, with one reason>
- Missing gate or blocker: <first meaningful blocker, or none found>
- Allowed writes for current phase: <phase write surface; status helper writes nothing>
- Next action: <from artifacts, or first safe action>
- Stop rule: <from phase file or inferred contract stop>
- Implementation may start: <yes / yes in recorded next session / yes with recorded concerns / no / unknown, with one reason>
```

## Stop Rules
Stop after the status report.

If the task path is not identified, ask for it and stop:

```text
I need the task-local path, such as `specs/<feature-id>`, before I can report workflow status from artifacts.
```

If artifacts are missing or contradictory, report the missing gate or conflict and stop. Do not repair them in the same pass.

## Anti-Patterns
- guessing the active task from a vague memory of the conversation
- treating this helper's status report as an approval record
- saying implementation may start because a task "looks small" without an explicit waiver and inline tasking
- treating `workflow-plans/<phase>.md` as a replacement for `workflow-plan.md`
- turning missing required compact/split design context or expected `tasks.md` into a harmless note for non-trivial work
- creating a second source of truth for "implementation readiness"
