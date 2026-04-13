---
name: workflow-planning-session
description: "Own a session dedicated only to workflow planning for this repository. Use when the orchestrator needs to choose execution shape, research mode, subagent lanes, current-phase routing, and later artifact expectations before research begins, and must write or update task-local `workflow-plan.md` plus `workflow-plans/workflow-planning.md` without drifting into research, `spec.md`, `design/`, `tasks.md`, or implementation. Skip tiny direct-path work and any task whose approved pre-research control artifact already lives under a different phase file."
---

# Workflow Planning Session

## Purpose
Run only the workflow-planning checkpoint for one task-local session.
This wrapper makes the pre-research control pass explicit and stoppable; it does not perform research, specification, technical design, task breakdown, or coding.

## Use When
- non-trivial or agent-backed work needs explicit workflow control before any subagent call or deeper research
- the orchestrator must choose `direct path`, `lightweight local`, or `full orchestrated`
- research mode, subagent lanes, challenge expectations, and later artifact expectations must be written down before the next session
- task-local `workflow-plan.md` or `workflow-plans/workflow-planning.md` is missing, stale, or inconsistent

## Skip When
- the work is tiny enough that `AGENTS.md` allows a short inline workflow-planning note and explicit skip rationale instead of dedicated workflow artifacts
- the task is already in research or a later phase and the current session should not reopen routing
- the active task already has an approved pre-research control artifact under another phase file and creating `workflow-plans/workflow-planning.md` would create a competing source of truth

## Required Inputs
Need only the minimum framing required to route the work:
- task goal or requested change
- known scope and non-goals
- known constraints, risks, and success checks
- whether the task is tiny/direct-path or non-trivial/agent-backed
- task-local artifact location when one already exists

If a routing fact is missing, record it as an assumption or blocker instead of inventing detail.

## Read First
Always read:
- `AGENTS.md`
- `docs/spec-first-workflow.md`

Then load only the smallest task-local context that affects workflow control:
- existing `workflow-plan.md`, if present
- existing `workflow-plans/workflow-planning.md`, if present
- the user request and any already-approved task-local artifact needed to confirm current stage, scope, or blockers

Rules:
- follow `AGENTS.md` if it conflicts with other workflow guidance
- read the master `workflow-plan.md` before the phase-local file when both exist
- do not broad-read the repository or domain references unless workflow routing truly depends on them

## Allowed Writes
This session may write or update only:
- task-local `workflow-plan.md`
- task-local `workflow-plans/workflow-planning.md`
- the `workflow-plans/` directory only when it must be created to hold the phase-local file

## Prohibited Actions
Do not:
- run local or subagent research
- write `research/*.md`
- write or finalize `spec.md`
- write `design/`
- write `tasks.md`, `test-plan.md`, or `rollout.md`
- start implementation, tests, migrations, or review work
- use planning or implementation skills as a backdoor into later phases
- make final domain, architecture, API, data, security, reliability, or rollout decisions that belong to later phases
- create a second active pre-research control artifact when the task already uses another approved phase file for that checkpoint

## Core Defaults
- This is an orchestrator-facing wrapper, not a domain specialist.
- `AGENTS.md` owns the workflow contract; `docs/spec-first-workflow.md` owns the detailed artifact mechanics.
- This skill owns session protocol only. It must not redefine later artifact ownership or phase behavior.
- Use `workflow-planning-session` only when a dedicated workflow-planning session is the intended control shape.
- Later phase-local files from the repository contract still matter: this wrapper does not replace `workflow-plans/specification.md`, `workflow-plans/technical-design.md`, `workflow-plans/planning.md`, or conditional review/validation phase files.
- Before handoff on non-trivial or agent-backed work, run or record the read-only `workflow-plan-adequacy-challenge`; tiny/direct-path work may skip it only with an explicit rationale.
- For non-trivial work, stop after the workflow artifacts are updated. Research or another recorded next phase begins in a new session unless an approved waiver already exists.

## Lazily Loaded References
Keep `SKILL.md` as the wrapper protocol. References are compact rubrics and example banks, not exhaustive checklists or alternate authority. After reading `AGENTS.md` and `docs/spec-first-workflow.md`, load at most one reference by default: choose the symptom whose behavior change matches the current uncertainty. Load multiple references only when the task clearly spans independent decision pressures, and never bulk-load the whole directory.

| Reference | Load When The Symptom Is... | Behavior Change |
| --- | --- | --- |
| [execution-shape-selection.md](references/execution-shape-selection.md) | choosing or checking `direct path`, `lightweight local`, or `full orchestrated` | chooses the smallest defensible execution shape with escalation triggers instead of forcing ceremony or under-routing cross-domain work |
| [research-mode-and-fanout-lanes.md](references/research-mode-and-fanout-lanes.md) | deciding `local` versus `fan-out`, or writing lane rows | plans read-only lanes by owned evidence question and one skill per lane instead of broad owner lanes, worker lanes, multi-skill lanes, or same-session research |
| [artifact-expectation-matrix.md](references/artifact-expectation-matrix.md) | marking later artifacts as expected, missing, draft, approved, conditional, not expected, or waived | records trigger-aware artifact status instead of inventing completeness, creating later artifacts early, or marking everything "not applicable" |
| [control-file-authoring-split.md](references/control-file-authoring-split.md) | deciding what belongs in `workflow-plan.md` versus `workflow-plans/workflow-planning.md` | keeps cross-phase status in the master and session-local orchestration in the phase file instead of duplicating details or drifting into later artifacts |
| [adequacy-challenge-and-stop-boundary.md](references/adequacy-challenge-and-stop-boundary.md) | routing the workflow-plan adequacy challenge, recording a skip, stopping at the boundary, or avoiding an existing phase-control collision | keeps the gate read-only and boundary-safe instead of treating short waits as failure, spawning research early, or creating competing control files |

If any reference example conflicts with `AGENTS.md` or `docs/spec-first-workflow.md`, follow the repo-local contract. Do not use an example as permission to start research, write `spec.md`, create `design/`, write `tasks.md`, or create implementation artifacts.

## Workflow

### 1. Confirm This Session Owns Workflow Planning Only
- Check whether the task is still at the routing stage.
- If the task is already in research or later, stop and hand back the correct reopen point instead of rewriting phase control.
- If the work is tiny enough for inline workflow planning only, say so directly and stop instead of forcing this wrapper.

### 2. Normalize Minimum Framing
- Capture what must change, scope cuts, constraints, risk hotspots, success checks, blockers, and visible assumptions.
- Keep missing facts visible instead of filling them in.

### 3. Choose Execution Shape And Research Mode
- Choose `direct path`, `lightweight local`, or `full orchestrated`.
- Decide whether the next research pass should be `local` or `fan-out`.
- If `fan-out` is expected, enumerate lanes by owned question, role, and one chosen skill or explicit `no-skill`.
- Decide whether a later pre-spec challenge pass is expected.
- Decide whether later `design/`, `tasks.md`, `test-plan.md`, or `rollout.md` artifacts are expected.
- Decide whether later review or validation phase workflow files may be expected, with the rule that planning creates only named files that are genuinely needed before implementation starts.

### 4. Set Session Routing
- Treat this session's local checkpoint as `workflow-planning`.
- Record the completion marker, stop rule, next action, blockers, and what can run in parallel.
- Record what the next session starts with. Default to `research` when workflow planning completes and no earlier waiver allows same-session collapse.
- Keep the wrapper narrow: it prepares later phase work, it does not begin that work.

### 5. Write Or Repair `workflow-plan.md`
- Record the execution shape and why it fits.
- Record research mode when later research is expected.
- Record current phase, phase status, session-boundary state, next-session routing, blockers, and phase workflow plan links or status.
- Record artifact status for `spec.md`, `design/`, `tasks.md`, and conditional later artifacts as `approved`, `draft`, `missing`, `blocked`, `conditional`, `waived`, or `not expected`, with trigger rationale for `conditional`, `waived`, or `not expected` instead of a bare label.
- Record whether later review or validation phase workflow files are expected and must be created during planning because named multi-session routing needs them, rather than mid-implementation or mid-validation.

### 6. Write Or Repair `workflow-plans/workflow-planning.md`
- Record only the local orchestration for this session.
- Include research mode when relevant, planned subagent lanes, order or parallelism, fan-in or challenge path, phase status, completion marker, next action, blockers, what can run in parallel, and the local stop rule.
- Keep this file routing-only. Do not turn it into `spec.md`, `design/`, or `tasks.md`.

### 7. Run Or Record The Workflow Plan Adequacy Challenge
- For non-trivial or agent-backed work, invoke one read-only challenger lane with exactly one skill: `workflow-plan-adequacy-challenge`.
- Give it the task frame, execution shape, master workflow plan, `workflow-plans/workflow-planning.md`, planned lanes, artifact expectations, and next-session handoff.
- Reconcile blocking findings by repairing the workflow-control artifacts, recording an accepted risk or waiver, or leaving the session blocked.
- For tiny/direct-path work, record the skip rationale instead of forcing the challenge.

### 8. Stop At The Boundary
- Once the two workflow artifacts are consistent and the next session can start without re-planning, stop.
- Do not begin research, specification, technical design, planning, or implementation in this session.

## Required Master `workflow-plan.md` Updates
Every completed pass must update the master file with:
- execution shape and why
- research mode, or an explicit note that later research is not expected
- current phase for this session and current phase status
- `Session boundary reached`
- `Ready for next session`
- `Next session starts with`
- `Next session context bundle` as an always-present field: say default resume order is sufficient, or list exact artifact paths and one-line reasons for task-specific resume context
- blockers and accepted assumptions that still affect routing
- phase workflow plan links or status, including `workflow-plans/workflow-planning.md`
- workflow plan adequacy challenge status and resolution, or an explicit direct/local skip rationale
- artifact status for `spec.md`, `design/`, `tasks.md`, and any triggered `test-plan.md` or `rollout.md`, with trigger rationale for `not expected`, `conditional`, or `waived` statuses
- phased-delivery policy, including whether later review and validation phase files are expected or still unknown

Do not leave those fields implicit in chat.

## Expected Outputs
Produce only workflow-control output:
- updated or newly created `workflow-plan.md`
- updated or newly created `workflow-plans/workflow-planning.md`
- an honest blocked state when routing cannot be completed without contradicting the repository contract

No research notes, `spec.md`, `design/`, `tasks.md`, or implementation output belongs to this session.

## Stop Condition
The session is complete when:
- execution shape and research mode are explicit
- the current workflow-planning checkpoint has a written completion marker and stop rule
- the next session start point is explicit
- master and phase-local workflow artifacts agree on phase status, blockers, and handoff
- required workflow plan adequacy challenge findings are reconciled, or an eligible skip rationale is explicit
- later required artifacts are marked as expected, draft, missing, or not expected instead of guessed into existence
- no research, specification, technical design, planning, or implementation work has started

## Escalate When
Escalate instead of forcing output when:
- the work is so small that a dedicated workflow-planning session would be pure ceremony
- the task is already in research or a later phase
- critical routing facts are missing and cannot be safely assumed
- the requested `workflow-plans/workflow-planning.md` would conflict with an already-approved current control file such as `workflow-plans/specification.md`
- the task needs real domain research before execution shape or lane planning can be chosen honestly
- the request tries to combine workflow planning with research, specification, technical design, planning, or implementation in one session

## Anti-Patterns
- using this wrapper as a substitute for domain research or `spec.md` authoring
- turning workflow planning into hidden task breakdown
- creating `workflow-plans/workflow-planning.md` plus another active pre-research control file for the same checkpoint
- inventing artifact status or blocker resolution for completeness theater
- selecting fan-out lanes without naming the question each lane owns
- marking the workflow-planning handoff ready while blocking adequacy challenge findings are unreconciled
- starting research "just to get ahead" before closing the session
