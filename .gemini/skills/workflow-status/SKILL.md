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
- the orchestrator needs to identify the current phase, artifact status, implementation-readiness status, missing gate, allowed writes, next action, or stop rule from task-local artifacts
- a task may be using direct-path or lightweight-local shortcuts and the helper needs to check whether the waiver is explicitly recorded

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
- treat a missing artifact as intentionally skipped unless an explicit direct-path or lightweight-local waiver is present in the task artifacts or current user-provided artifact excerpt

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

1. task-local `workflow-plan.md`, if present
2. current `workflow-plans/<phase>.md`, if the master names a current phase or next phase
3. task-local `spec.md`
4. task-local `design/overview.md`, then required core design files when design status matters:
   - `design/component-map.md`
   - `design/sequence.md`
   - `design/ownership-map.md`
5. task-local `tasks.md` when present or expected by workflow status
6. optional task-local `test-plan.md`, `rollout.md`, and selected `research/*.md` only when they are present and the status question depends on them

When `workflow-plan.md` is missing, infer only the minimum state from the artifact chain and mark workflow control as incomplete unless an explicit direct-path or lightweight-local waiver explains the missing file.

## Status Inference Rules
- Prefer `workflow-plan.md` for current phase, phase status, session-boundary state, blockers, artifact status, and next-session routing.
- Prefer the current `workflow-plans/<phase>.md` for phase-local next action, stop rule, completion marker, local blockers, and the planning-phase implementation-readiness gate result when the current or completed phase is `planning`.
- Use `spec.md`, `design/`, and `tasks.md` only to confirm artifact presence and approval signals, not to invent a different phase than the master file records.
- Treat absent required artifacts as incomplete unless an explicit waiver covers that exact artifact.
- Treat present artifacts with unclear approval state as `present / status unclear`, not `approved`.
- Treat a missing implementation-readiness status as incomplete for non-trivial planned work unless an explicit eligible direct-path, lightweight-local, or prototype waiver covers it.
- If the master and phase-local file conflict, report the conflict as the blocker instead of choosing a winner silently.
- If `Session boundary reached: yes`, report that the next action belongs to the recorded next session or reopen target; do not continue the prior phase in the same session.
- If `Ready for next session: no`, report the active phase as still needing work unless the artifacts clearly say the master is stale.
- `tasks.md` may be read when present or expected by the workflow. This helper reports its status but must not create, repair, or approve `tasks.md` or the implementation-readiness gate.

## Implementation Start Rule
Answer `Implementation may start` conservatively:

- `Yes` only when readiness is `PASS`, the required artifact chain is approved or explicitly waived, there are no blocking gates, and workflow routing points to implementation or the first task in `tasks.md`.
- `Yes, in the recorded next session` when readiness is `PASS`, planning is complete, `Session boundary reached: yes`, and `Next session starts with` points at implementation.
- `Yes, with recorded concerns` only when readiness is `CONCERNS`, named accepted risks and proof obligations are explicit, and routing points to implementation.
- `No` when readiness is `FAIL`, or when `spec.md`, required `design/`, expected `tasks.md`, phase control, readiness status, or a required review/validation phase file is missing without an explicit waiver.
- `No` when the current phase is workflow planning, research, specification, technical design, planning, review, reconciliation, validation, or done and the artifacts do not route to implementation.
- `No` when readiness is `CONCERNS` but accepted risks or proof obligations are unnamed.
- `Unknown` only when the task path is identified but the artifacts are too contradictory to make a safe yes or no call; name the contradiction as the blocker.

For direct-path, lightweight-local, or prototype work, `WAIVED` allows implementation only if the waiver, rationale, scope, and inline tasking are explicit in the current task record. Do not infer a waiver from task size alone.

## Allowed Writes Reference
Report the phase's allowed write surface using the repository contract, while making clear that this helper writes nothing:

- `workflow planning`: `workflow-plan.md` and `workflow-plans/workflow-planning.md`
- `research`: `research/*.md`, task-local `workflow-plan.md`, and the active research phase-control file when the session owns research
- `specification`: `spec.md`, task-local `workflow-plan.md`, and `workflow-plans/specification.md`
- `technical design`: task-local `design/` core and triggered conditional design files, task-local `workflow-plan.md`, and `workflow-plans/technical-design.md`
- `planning`: `tasks.md` when expected, triggered `test-plan.md` or `rollout.md`, named review/validation phase-control files when needed, task-local `workflow-plan.md`, and `workflow-plans/planning.md`
- `implementation`: code, tests, migrations, configs, generation inputs, generated outputs required by the approved task ledger, plus existing `workflow-plan.md` routing and existing `tasks.md` progress only
- `review`: read-only review output only; no code or artifact mutation by review agents
- `reconciliation`: approved code/test/runtime fixes required by the task ledger plus existing control/checkpoint artifacts only
- `validation`: fresh verification plus existing closeout surfaces only, such as `spec.md` `Validation`/`Outcome`, existing `workflow-plan.md`, existing `tasks.md` progress when used, and an existing validation phase-control file when one was created before implementation
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
- Implementation readiness: <PASS / CONCERNS / FAIL / WAIVED / missing / unknown, with one reason>
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
- turning missing `design/` or expected `tasks.md` into a harmless note for non-trivial work
- creating a second source of truth for "implementation readiness"
