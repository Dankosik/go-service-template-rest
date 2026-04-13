---
name: research-session
description: "Own a session dedicated only to research for this repository. Use when the orchestrator already has task framing and needs one bounded session to run local research or read-only subagent fan-out, preserve evidence in `research/*.md` when useful, and update task-local `workflow-plan.md` plus `workflow-plans/research.md` without drifting into `spec.md`, `design/`, `tasks.md`, optional `plan.md`, or implementation. Skip tiny direct-path work and tasks that have already moved into `specification` or later."
---

# Research Session

## Purpose
Run only the research checkpoint for one task-local session.
This wrapper makes evidence gathering and handoff explicit; it does not finalize `spec.md`, start `technical design`, produce `tasks.md`, produce optional `plan.md`, or implement code.

## Use When
- the task already has minimum viable framing and workflow routing, and now needs one research-only session
- the orchestrator must choose between `local` research and read-only subagent `fan-out` for this session
- repository evidence, external references, comparisons, or specialist reads must be gathered before specification
- preserved `research/*.md` notes would reduce later guesswork, make fan-in easier, or support a future resume
- master `workflow-plan.md` needs the research checkpoint completed or updated before a later `specification-session`

## Skip When
- the work is tiny enough that inline local reasoning plus a short note is sufficient and a dedicated research session would be ceremony
- the task is still at workflow planning; use `workflow-planning-session`
- the task has already moved into `specification` or later, or `workflow-plans/specification.md` is already the active phase-control file
- the request tries to combine research with final `spec.md`, `design/`, `tasks.md`, optional `plan.md`, or implementation output in one session

## Required Inputs
Need only the minimum phase-ready inputs:
- framed task goal plus scope and non-goals
- known constraints, risk hotspots, and success checks
- current workflow routing and task-local artifact location
- any already-known research questions, blockers, or assumptions
- existing research artifacts or lane outputs when this is a continuation

If a research prerequisite is missing, record it as an assumption or blocker instead of inventing facts.

## Read First
Always read:
- `AGENTS.md`
- `docs/spec-first-workflow.md`

Then read current phase context in this order:
1. task-local `workflow-plan.md`, if present
2. task-local `workflow-plans/research.md`, if present
3. the smallest task-local artifact that explains current routing, blockers, or prior research:
   - the user request or approved framing artifact
   - relevant `research/*.md`
   - existing `spec.md` only to confirm whether specification already started, never to edit it in this session
4. targeted repository or external sources needed to answer the chosen research questions

Rules:
- follow `AGENTS.md` if other workflow guidance conflicts
- read the master `workflow-plan.md` before the phase-local file when both exist
- do not broad-read the repository or unrelated references when narrow evidence is enough
- if phase context shows the task already advanced past research, stop instead of casually reopening an earlier phase

## Allowed Writes
This session may write or update only:
- task-local `workflow-plan.md`
- task-local `workflow-plans/research.md`
- task-local `research/*.md` when preserved evidence, comparisons, or source notes will materially help later synthesis, challenge, or resume
- the `workflow-plans/` or `research/` directories only when they must be created to hold those artifacts

## Prohibited Actions
Do not:
- finalize or approve `spec.md`
- create `workflow-plans/specification.md`
- assemble `design/`
- write `tasks.md`, optional `plan.md`, `test-plan.md`, or `rollout.md`
- start implementation, tests, migrations, review, or validation work
- use planning or implementation skills as a backdoor into later phases
- turn `research/*.md` into a second source of truth for decisions that belong in `spec.md`
- let subagents write code, files, git state, or the task ledger or implementation handoff
- silently continue into specification once research feels "close enough"

## Core Defaults
- this is an orchestrator-facing wrapper, not a domain specialist
- `AGENTS.md` owns the workflow contract; `docs/spec-first-workflow.md` owns the detailed artifact mechanics
- this wrapper owns research-session protocol only and must not redefine later artifact ownership
- use `research-session` only when a dedicated research session is the intended control shape for the task
- support both `local` research and read-only subagent `fan-out`
- each subagent lane owns one question and at most one skill, or explicit `no-skill`
- preserve `research/*.md` only when the evidence will help later fan-in, challenge, auditability, or multi-session resume
- a finished research session hands off evidence and routing; it does not convert that evidence into approved `Decisions`

## Lazy-Loaded References
Keep this `SKILL.md` as the protocol. References are compact rubrics and example banks, not exhaustive checklists or documentation dumps. Load at most one reference by default; load more only when the task clearly spans multiple independent decision pressures, such as both mode selection and final fan-in handoff.

Treat every reference as non-authoritative support under `AGENTS.md` and `docs/spec-first-workflow.md`. Use live repository files or current external sources when freshness matters instead of keeping link dumps in the skill.

| Symptom | Behavior change | Reference |
| --- | --- | --- |
| Research questions are too broad, solution-led, biased, or mixed with future decisions. | Makes the model write answerable evidence-targeted questions instead of asking research to decide the solution, check everything, or draft later-phase artifacts. | `references/research-question-framing.md` |
| The hard choice is whether the session should stay `local` or use read-only `fan-out`. | Makes the model choose mode by evidence surface, risk, and independent-question shape instead of choosing fan-out because "more agents is safer" or staying local to hide cross-domain uncertainty. | `references/local-vs-fanout-mode-selection.md` |
| Mode is chosen or likely, but `workflow-plans/research.md` has vague lanes, unclear roles, weak source targets, or missing parallelism. | Makes the model assign one owned question, one role, one skill, one evidence target, and a fan-in path per lane instead of using broad domain labels, multi-skill lanes, write-capable workers, or decision-approving subagents. | `references/research-lane-planning.md` |
| The session must decide whether to preserve `research/*.md`, or an existing research note has poor source hygiene. | Makes the model write compact evidence notes with source relevance, limitations, and handoff value instead of dumping generic notes, command output, links, or `spec.md` decisions. | `references/evidence-note-structure.md` |
| Research lanes are complete, partial, or blocked and need a clean boundary handoff. | Makes the model hand off comparable evidence, conflicts, readiness, and next-session routing instead of converting research into approved `spec.md` decisions or drifting into design and planning. | `references/fan-in-handoff-examples.md` |
| A research session has concrete drift smells across multiple surfaces, or no narrower positive reference matches. | Makes the model stop, repair the boundary, or route back to the right phase instead of treating `research-session` as a catch-all path into specification, design, planning, implementation, or note sprawl. | `references/research-session-anti-patterns.md` |

## Boundary With Future `specification-session`
- `research-session` may write `workflow-plan.md`, `workflow-plans/research.md`, and optional `research/*.md`
- the future `specification-session` owns approved `spec.md`, `workflow-plans/specification.md`, and the handoff into `technical design`
- if the task is ready to move forward, record `Next session starts with: specification` and stop instead of drafting spec sections here

## Workflow

### 1. Confirm This Session Owns Research Only
- check the master workflow plan and active phase context first
- if the task is still at workflow planning, send it back to `workflow-planning-session`
- if the task is already at specification or later, stop and point to the correct reopen point instead of reopening research casually
- if the work is tiny enough for inline local handling, say so directly and stop rather than forcing this wrapper

### 2. Read Current Phase Context
- confirm current phase, phase status, blockers, assumptions, and expected next session
- reuse existing research artifacts or unfinished lane outputs before planning new work
- detect whether this session is a fresh research pass, a continuation, or targeted re-research after challenge or recheck

### 3. Define The Research Questions
- list only the questions whose answers can change scope, correctness, constraints, risk handling, or spec readiness
- separate must-answer-now from nice-to-know
- keep unknowns visible instead of filling them in

### 4. Choose Research Mode And Plan Lanes
- choose `local` when the work is bounded enough that the orchestrator can gather the evidence directly without losing clarity
- choose `fan-out` when cross-domain coverage, second opinions, ambiguity reduction, or preserved specialist evidence would materially improve the outcome
- for each lane, record:
  - owned question
  - local or subagent execution
  - role
  - one chosen skill or explicit `no-skill`
  - evidence target or expected source surface
  - order or parallelism
- if `fan-out` is used, keep every lane read-only and prefer enough coverage over artificial subagent minimization
- record whether a later pre-spec challenge pass is expected after research fan-in

### 5. Run Research And Preserve Only What Helps
- gather repository or external evidence for the chosen questions
- create `research/*.md` only when the evidence, comparisons, or source notes need to survive beyond this session
- keep research artifacts evidence-oriented: question, findings, sources or file references, assumptions, open points, and why the note matters
- do not treat `research/*.md` as the decision record

### 6. Capture Research Fan-In Without Writing `spec.md`
- summarize what is now known, what remains uncertain, and whether the task is ready for specification, challenge, or more research
- keep conclusions in research or handoff language, not as approved spec wording
- if evidence is still weak in a high-impact seam, route to targeted re-research or challenge instead of pretending the task is spec-ready

### 7. Write Or Repair `workflow-plans/research.md`
- record phase-local orchestration only:
  - research mode
  - lane plan or executed lanes
  - order or parallelism
  - fan-in path
  - whether later challenge is expected
  - phase status
  - completion marker
  - stop rule
  - next action
  - blockers
  - what can run in parallel
- keep this file routing-only; do not turn it into `spec.md`, `design/`, `tasks.md`, or optional `plan.md`

### 8. Write Or Repair `workflow-plan.md`
- update master phase status, research mode, artifact status, blockers, and next-session routing
- make it explicit whether research is complete, blocked, or reopened
- keep the handoff ready for the future `specification-session` without drafting it here

### 9. Stop At The Boundary
- once research artifacts and routing are consistent, stop
- do not start `spec.md`, `workflow-plans/specification.md`, `design/`, `tasks.md`, optional `plan.md`, or implementation in the same session

## Research Lane Planning Rules
When a dedicated research phase file is used, `workflow-plans/research.md` should make lane ownership obvious at a glance.
Each lane should name:
- the question it owns
- whether it is local or subagent work
- role
- single skill or `no-skill`
- evidence target
- status

Good defaults:
- keep unrelated questions in separate lanes
- reuse the same role in multiple lanes when the questions differ
- use `primary + challenger` or second-opinion coverage when impact is high or assumptions are fragile
- avoid asking one lane to handle both evidence gathering and final decision writing

## Required Master `workflow-plan.md` Updates
Every completed or blocked pass must update the master file with:
- current phase set to this research checkpoint and current phase status
- research mode and why
- lane summary or an explicit note that research stayed local
- link or status for `workflow-plans/research.md`
- status for preserved `research/*.md`, or an explicit note that none were needed
- whether the evidence is sufficient for the next session to start with `specification`, or whether challenge, recheck, or targeted re-research is next
- `Session boundary reached`
- `Ready for next session`
- `Next session starts with`
- blockers, accepted assumptions, and open points that still affect spec readiness
- artifact status for `spec.md`, `design/`, `tasks.md`, optional `plan.md`, and any triggered later artifacts as `approved`, `draft`, `missing`, or not expected

Do not leave spec readiness or handoff state implicit in chat.

## Expected Outputs
A finished research session produces only research-phase artifacts and routing:
- updated or newly created `workflow-plan.md`
- updated or newly created `workflow-plans/research.md`
- optional `research/*.md` for preserved evidence or comparisons
- an honest `complete`, `blocked`, or `reopened` research-phase state with the next session start point made explicit

It does not produce approved `spec.md`, `workflow-plans/specification.md`, `design/`, `tasks.md`, optional `plan.md`, or implementation output.

## Stop Condition
The session is complete when:
- the current research questions and evidence sources were handled or explicitly deferred
- research mode and lane ownership are explicit
- useful preserved research has been written only where it helps later synthesis or resume
- master and phase-local workflow artifacts agree on phase status, blockers, and handoff
- the next session start point is explicit, including whether it is `specification`, challenge, or targeted re-research
- no specification, technical design, planning, or implementation work has started

## Escalate When
Escalate instead of forcing output when:
- minimum framing or workflow routing is missing and research scope cannot be chosen honestly
- the request tries to combine research with spec authoring, technical design, planning, or implementation
- the task already advanced to `specification` or later
- the task is so small that a dedicated research session would be ceremony
- a requested fan-out lane cannot be kept read-only
- evidence conflicts remain unresolved and require challenger or targeted follow-up before handoff
- creating `workflow-plans/research.md` would conflict with an already-approved later-phase control file

## Anti-Patterns
- treating this wrapper as a way to sneak into `spec.md`
- creating research lanes without naming the question each lane owns
- launching write-capable or multi-skill subagent passes
- preserving every note in `research/*.md` even when the evidence is disposable
- restating artifact file formats that belong in repository-level workflow docs
- marking research complete while spec readiness, blockers, or handoff routing stay implicit
- writing stable `Decisions` in research artifacts instead of handing them to the future `specification-session`
