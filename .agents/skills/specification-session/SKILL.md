---
name: specification-session
description: "Own a session dedicated only to specification for this repository. Use when the orchestrator already has framing plus enough researched or explicitly bounded input to finalize `spec.md`, must run or reconcile the non-trivial spec-clarification challenge before approval, and must update task-local `workflow-plan.md` plus `workflow-plans/specification.md` without drifting into `design/`, `plan.md`, `tasks.md`, or implementation. Skip tiny direct-path work and tasks that are still in workflow planning or research."
---

# Specification Session

## Purpose
Run only the specification checkpoint for one task-local session.
This wrapper makes spec-ready input, the autonomous clarification gate, allowed writes, handoff, and stop conditions explicit; it does not assemble `design/`, produce `plan.md`, produce `tasks.md`, or start implementation.

## Use When
- the task already has minimum viable framing and enough evidence or bounded assumptions to support an honest `spec.md`
- prior workflow routing says the next session starts with `specification`
- research, challenge, or direct local analysis already narrowed the open questions enough that stable `Decisions` can now be written
- non-trivial candidate decisions are ready for the required `spec-clarification-challenge` pass before approval
- task-local `spec.md`, `workflow-plan.md`, or `workflow-plans/specification.md` is missing, stale, or inconsistent and the active session should repair the specification checkpoint only

## Skip When
- the work is tiny enough that `AGENTS.md` allows an inline local path and a dedicated specification session would be ceremony
- the task is still at workflow planning or research, or the current evidence is not yet spec-ready
- the task has already moved into `technical design` or later and the current session should not reopen specification casually
- the request tries to combine specification with `design/`, `plan.md`, `tasks.md`, or implementation output in one session

## Required Inputs
Need only the minimum phase-ready inputs from earlier work:
- framed task goal, scope, non-goals, constraints, risk hotspots, and success checks
- current workflow routing and task-local artifact location
- the latest upstream phase output that made specification the next checkpoint
- relevant research findings, comparison notes, or explicit rationale for why more research is not needed
- candidate decisions compact enough to give a read-only clarification challenger a useful input bundle
- unresolved assumptions, blockers, or challenge outcomes that still affect spec honesty
- existing `spec.md`, if this is a continuation or repair rather than a fresh pass

If a required fact is missing, record it as an assumption, blocker, or reopen point instead of inventing detail.

## What Counts As Spec-Ready Input
Treat the session as spec-ready only when all of the following are true:
- the behavior delta and scope cuts are explicit enough to write stable `Decisions`
- material constraints, risks, and validation expectations are visible
- the current evidence is strong enough to avoid fiction, or the remaining uncertainty is small enough to live honestly in `Open Questions / Assumptions`
- prior research or direct analysis already answered the must-answer-now questions, or the workflow plan explicitly records why more research is not required
- for non-trivial work, the `spec-clarification-challenge` gate can be run and reconciled before `spec.md` is marked approved
- the task can hand off to `technical design` without reopening core framing by default

If those conditions are not met, do not force an approval. Reopen the right upstream phase instead.

## Read First
Always read:
- `AGENTS.md`
- `docs/spec-first-workflow.md`
- `.agents/skills/spec-document-designer/SKILL.md`
- `.agents/skills/spec-clarification-challenge/SKILL.md`

Then read current phase context in this order:
1. task-local `workflow-plan.md`, if present
2. task-local `workflow-plans/specification.md`, if present
3. the smallest upstream artifact that explains why specification is next:
   - `workflow-plans/research.md`
   - `workflow-plans/workflow-planning.md`
   - approved framing notes or equivalent task-local context
4. relevant `research/*.md`, when present
5. existing `spec.md`, only to continue or normalize it in this session

Rules:
- follow `AGENTS.md` if other guidance conflicts
- read the master `workflow-plan.md` before the phase-local file when both exist
- load only the smallest evidence set needed to decide whether the task is truly spec-ready
- if phase context shows the task already advanced past specification, stop instead of casually reopening an earlier phase

## Lazy Reference Routing
References are compact rubrics and example banks, not exhaustive checklists, documentation dumps, or second authorities.
Load at most one reference by default: choose the narrowest file whose symptom matches the active decision pressure.
Load more than one only when the session clearly spans independent pressures, such as readiness uncertainty plus a separate clarification-gate result.
If a reference would not change a decision, do not load it.
These references calibrate this wrapper; they do not override `AGENTS.md`, `docs/spec-first-workflow.md`, `spec-document-designer`, or `spec-clarification-challenge`.

| Symptom | Reference | Behavior Change |
| --- | --- | --- |
| Phase ownership is unclear, the input looks under-researched, or the caller asks to approve `spec.md` from partial decisions. | `references/specification-session-readiness.md` | Choose a spec-ready check, bounded assumption, or reopen/block decision instead of approving by momentum. |
| The session is about to edit files, or the request pressures it toward `design/`, `plan.md`, `tasks.md`, tests, migrations, or implementation. | `references/allowed-writes-and-stop-rules.md` | Keep writes specification-only and record the boundary instead of creating downstream starter artifacts. |
| Non-trivial `spec.md` approval depends on running, reconciling, blocking, or waiving the clarification gate. | `references/spec-clarification-gate-flow.md` | Reconcile `spec-clarification-challenge` outcomes into final decisions instead of treating the gate as optional, pasting transcripts, or deferring approval blockers to design. |
| `workflow-plan.md` or `workflow-plans/specification.md` needs repair or handoff updates. | `references/workflow-plan-specification-updates.md` | Keep master routing separate from phase-local orchestration instead of duplicating `spec.md`, adding implementation order, or leaving state in chat. |
| `spec.md` cannot honestly be approved because of under-framed input, contradictory evidence, unresolved challenge questions, product-only policy, or phase drift. | `references/blocked-specification-examples.md` | Leave `spec.md` draft or blocked with a precise reopen target instead of inventing decisions or punting approval-changing gaps to technical design. |
| `spec.md` is approved or near-approved and the next session route is being chosen. | `references/handoff-to-technical-design.md` | Record a clean `technical-design` handoff and stop instead of starting design work or hiding assumptions in chat. |

## Allowed Writes
This session may write or update only:
- task-local `spec.md`
- task-local `workflow-plan.md`
- task-local `workflow-plans/specification.md`
- the `workflow-plans/` directory only when it must be created to hold the phase-local file

## Prohibited Actions
Do not:
- write `research/*.md` except by handing the task back to a research checkpoint instead of continuing here
- assemble or edit `design/`
- write `plan.md`, `tasks.md`, `test-plan.md`, or `rollout.md`
- start implementation, tests, migrations, review, or validation work
- use planning or implementation skills as a backdoor into later phases
- turn `workflow-plans/specification.md` into a second `spec.md`, a design bundle, or a task list
- approve `spec.md` when the input is still under-evidenced, contradictory, or idea-shaped
- approve non-trivial `spec.md` before the clarification challenge is complete, reconciled, or explicitly waived by a direct/local exception; if the gate is blocked, leave `spec.md` unapproved with rationale
- silently continue into `technical design` once the spec feels close enough

## Core Defaults
- this is an orchestrator-facing wrapper, not a domain specialist
- `AGENTS.md` owns the workflow contract; `docs/spec-first-workflow.md` owns the artifact mechanics
- this wrapper owns specification-session protocol only and must not redefine spec shape, design rules, or planning behavior
- use `spec-document-designer` as the deeper method for writing or normalizing `spec.md`
- for non-trivial work, run the clarification challenge with a read-only subagent lane, preferably `challenger-agent`, using exactly one skill: `spec-clarification-challenge`
- the clarification subagent returns questions for orchestrator reconciliation; it never edits files or makes final decisions
- keep the wrapper focused on session readiness, allowed writes, handoff, and stop rules
- a finished specification session ends at approved `spec.md` for non-trivial work unless an earlier recorded waiver already allows phase collapse

## Boundary With `spec-document-designer`
- `spec-document-designer` owns the deeper `spec.md` authoring method: section choice, decision placement, artifact ownership, and technical-design handoff quality
- `specification-session` owns when that method may run inside a dedicated session, what phase inputs are required, what files may be changed, how `workflow-plan.md` and `workflow-plans/specification.md` must be updated, and why the session must stop
- do not duplicate the full `spec-document-designer` section guidance here; reuse it after the phase boundary is confirmed
- if the real need is only local spec shaping without session routing, use `spec-document-designer` directly instead of this wrapper

## Workflow

### 1. Confirm This Session Owns Specification Only
- check the master workflow plan and current phase context first
- if the task is still at workflow planning or research, hand it back to the correct upstream phase instead of spec-writing by momentum
- if the task is already at `technical design` or later, stop and point to the correct reopen point
- if the work is tiny enough for an inline local path, say so directly and stop rather than forcing this wrapper

### 2. Check Spec Readiness
- verify that the task has enough framing, evidence, and bounded unknowns to support an honest decision record
- separate must-answer-now gaps from acceptable open questions
- if a missing answer can change scope, correctness, or ownership, reopen research or challenge instead of writing around it

### 3. Reuse `spec-document-designer` For The Actual Spec Pass
- once the boundary is confirmed, use `spec-document-designer` to draft or normalize `spec.md`
- keep stable decisions in `Decisions`, visible unknowns in `Open Questions / Assumptions`, and validation hooks explicit
- keep technical detail out of `spec.md` when it belongs in a later `design/` artifact
- keep execution sequencing out of `spec.md`; that belongs to later planning

### 4. Run The Autonomous Clarification Gate
- for non-trivial work, prepare a compact input bundle: problem frame, scope and non-goals, candidate decisions, constraints, validation expectations, known assumptions or open questions, and relevant research links
- invoke one read-only subagent lane, preferably `challenger-agent`, with exactly one skill: `spec-clarification-challenge`
- answer each returned question from existing evidence when possible
- if an answer requires expert work, reopen targeted research or fan-out with one read-only lane per expert question and one skill per lane; in a dedicated specification session, record the reopen and stop unless an upfront direct/local waiver already allowed same-session collapse
- if a question is truly external product or business policy and cannot be answered from repo evidence or safe assumptions, record `requires_user_decision` and leave `spec.md` blocked or partially draft instead of inventing the answer
- if material decisions changed or a major seam was reopened and then resolved, rerun the clarification challenge once on the updated candidate synthesis
- store final resolved outcomes in `spec.md` sections: stable outcomes in `Decisions`, remaining assumptions in `Open Questions / Assumptions`, and proof consequences in `Validation`; do not paste raw subagent transcript into `spec.md`

### 5. Write Or Repair `workflow-plans/specification.md`
- record phase-local orchestration only:
  - readiness check outcome
  - input sources used
  - whether the pass is fresh, continuation, or repair
  - clarification challenge status
  - subagent lane used for the clarification challenge, or the direct/local waiver rationale
  - whether targeted research was reopened
  - clarification resolution status
  - why `spec.md` is approved, draft, or blocked
  - phase status
  - completion marker
  - stop rule
  - next action
  - blockers
  - what can run in parallel
- keep this file routing-only; do not turn it into `spec.md`, `design/`, `plan.md`, or `tasks.md`

### 6. Write Or Repair `workflow-plan.md`
- update master phase status, artifact status, blockers, and next-session routing
- make it explicit whether `spec.md` is approved, still draft, or blocked
- record clarification gate status and whether `technical design` is the next session or whether upstream re-research, expert subagent work, or challenge reopened the flow
- keep the handoff ready for the future `technical-design` session without beginning that work here

### 7. Stop At The Boundary
- once `spec.md`, `workflow-plan.md`, and `workflow-plans/specification.md` agree on state and handoff, stop
- do not start `design/`, `plan.md`, `tasks.md`, or implementation in the same session

## What To Hand Off To Technical Design
When specification completes successfully, the handoff is:
- approved `spec.md` as the canonical decisions artifact
- updated `workflow-plan.md` with the next session routed to `technical-design`
- updated `workflow-plans/specification.md` showing the specification checkpoint is complete, the clarification gate is resolved or explicitly waived, and why the session stopped
- explicit blockers, accepted assumptions, and reopen conditions that technical design must honor instead of rediscover

Do not hand off a hidden design bundle, task breakdown, or implementation starter patch.

## Required Master `workflow-plan.md` Updates
Every completed or blocked pass must update the master file with:
- current phase set to this specification checkpoint and current phase status
- link or status for `workflow-plans/specification.md`
- status for `spec.md` as `approved`, `draft`, or `blocked`
- clarification gate status
- whether the task is spec-ready for `technical design`, or whether research, expert subagent work, or challenge reopened
- `Session boundary reached`
- `Ready for next session`
- `Next session starts with`
- blockers, accepted assumptions, and open points that still affect the handoff
- artifact status for `design/`, `plan.md`, `tasks.md`, and any triggered `test-plan.md` or `rollout.md` as `approved`, `draft`, `missing`, or not expected

Do not leave spec approval or handoff state implicit in chat.

## Expected Outputs
A finished specification session produces only specification-phase artifacts and routing:
- updated or newly created `spec.md`
- updated or newly created `workflow-plan.md`
- updated or newly created `workflow-plans/specification.md`
- an honest `complete`, `blocked`, or `reopened` specification-phase state with the next session start point made explicit

It does not produce `design/`, `plan.md`, `tasks.md`, or implementation output.

## Stop Condition
The session is complete when:
- the spec-ready check is satisfied or explicitly failed with an honest reopen point
- the clarification gate is resolved, explicitly waived by an eligible direct/local exception, or clearly blocked
- `spec.md` is approved or clearly left unapproved for a documented reason
- master and phase-local workflow artifacts agree on phase status, blockers, and handoff
- the next session start point is explicit, including whether it is `technical-design`, challenge, or more research
- the session stops before `design/`, `plan.md`, `tasks.md`, or implementation begins

## Escalate When
Escalate instead of forcing output when:
- the task is still idea-shaped, under-framed, or under-researched
- material contradictions remain unresolved across domains
- the clarification challenge produces a planning-critical question that needs targeted research, expert subagent work, or a `requires_user_decision` answer before approval
- the request tries to combine specification with `design/`, planning, or implementation
- the task already advanced to `technical design` or later
- the task is so small that a dedicated specification session would be ceremony
- creating `workflow-plans/specification.md` would conflict with an already-approved later-phase control file

## Anti-Patterns
- turning this wrapper into a second copy of `spec-document-designer`
- restating full `spec.md` section rules here instead of reusing the deeper skill
- approving a spec that still depends on invented answers
- treating the clarification challenge as optional ceremony for non-trivial work
- hiding research or challenge gaps under generic wording
- stuffing design detail, task sequencing, or implementation hints into `spec.md` to avoid later phases
- treating `workflow-plans/specification.md` as a second decision record instead of a phase-control artifact
- starting `technical design` "just to get ahead" before the session ends
