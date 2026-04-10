# Session-Phase Skills Direction

Status: future workflow direction, not yet part of the approved repository contract

## Why This Note Exists

Capture a possible next step for the artifact-driven workflow redesign:

- one session should complete one workflow phase
- if work needs to move to the next phase, that next phase should begin in a new session
- the orchestrator should have a dedicated phase/session skill that defines the scope, allowed writes, required artifact output, and stop condition for that session

This note exists so the idea does not get lost while the current artifact-driven workflow changes are landing.

## Problem To Solve

Even with the richer artifact chain:

`workflow-plan.md -> spec.md -> design/ -> plan.md -> implementation`

the workflow still relies heavily on orchestrator discipline to stop at the end of a phase and hand off to the next session.

That leaves a gap:
- a session may casually drift from `specification` into `technical design`
- a planning session may start implementation work
- implementation may continue into the next phase instead of stopping at a reviewable checkpoint

The goal of phase/session skills would be to make those boundaries explicit and repeatable.

## Working Hypothesis

If the repository later adopts a session-boundary rule such as `one session = one phase` for non-trivial work, then the orchestrator should have phase-specific skills that:

- define what phase the session is in
- define which artifacts must be read first
- define which artifacts may be written in that session
- define which work is explicitly out of scope for that session
- define what marks the phase complete
- define how `workflow-plan.md` must be updated before stopping
- force a handoff into the next session instead of continuing into the next phase inline

## Recommended Design Direction

Do not let phase/session skills become the new source of truth for workflow policy.

Keep responsibilities split:

- `AGENTS.md`
  - owns the repository workflow contract
  - owns the phase model, gates, artifact ownership, and session-boundary policy
- `docs/spec-first-workflow.md`
  - owns the detailed workflow mechanics, artifact order, resume rules, and artifact contracts
- phase/session skills
  - own the protocol for one specific session-phase only
  - should not redefine the workflow differently from the repository contract

## Candidate Session-Phase Skills

Recommended future shape:

1. `workflow-planning-session`
   - session output:
     - approved `workflow-plan.md`
   - must not:
     - run research
     - write `spec.md`
     - write `design/`
     - write `plan.md`

2. `research-session`
   - session output:
     - updated `workflow-plan.md`
     - optional `research/*.md`
   - must not:
     - finalize `spec.md`
     - start `technical design`
     - start `planning`

3. `specification-session`
   - likely an adaptation of `spec-document-designer`
   - session output:
     - approved `spec.md`
     - updated `workflow-plan.md`
   - must not:
     - assemble `design/`
     - write `plan.md`

4. `technical-design-session`
   - likely an adaptation of `go-design-spec`
   - session output:
     - approved task-local `design/`
     - updated `workflow-plan.md`
   - must not:
     - reopen framing casually
     - write `plan.md`
     - start implementation

5. `planning-session`
   - likely an adaptation of `planning-and-task-breakdown`
   - session output:
     - approved `plan.md`
     - optional `test-plan.md`
     - optional `rollout.md`
     - updated `workflow-plan.md`
   - must not:
     - start implementation

6. `implementation-phase-session`
   - new orchestrator-facing wrapper
   - session output:
     - implementation of exactly one planned phase or one explicit checkpoint
     - updated artifact status
     - phase-local validation evidence
   - must not:
     - begin the next implementation phase in the same session unless that was explicitly the session goal and remains within one bounded checkpoint

7. `validation-closeout-session`
   - new orchestrator-facing wrapper
   - session output:
     - final validation evidence
     - updated `Outcome`
     - updated `workflow-plan.md`
   - must not:
     - reopen implementation silently

## Guardrails

If this direction is implemented later:

- do not duplicate file-format rules in every skill
- keep artifact contracts in one repository-level place, preferably `docs/spec-first-workflow.md` or another explicit artifact-contract reference
- let each phase/session skill focus on:
  - allowed inputs
  - allowed writes
  - must-not-do rules
  - required outputs
  - stop condition
  - handoff update to `workflow-plan.md`

Each phase/session skill should answer:

- what phase this session is in
- what files must be read
- what files may be changed
- what marks the phase complete
- what must be written to `workflow-plan.md`
- why the session must stop instead of continuing into the next phase

## Expected Benefits

- stronger support for `one session = one phase`
- less cross-phase drift
- cleaner multi-session handoff
- more predictable artifact completion
- easier resume behavior for future orchestrators

## Main Risks

- workflow drift if phase/session skills contradict `AGENTS.md`
- duplication if every skill restates artifact formats
- too much ceremony for tiny or direct-path work
- wrapper-skill proliferation if the phase list becomes too granular

## Recommended Next Step

Run a dedicated research/spec session focused on session-boundary policy before implementing this direction.

If later implementation work is approved, a prompt pack for that direction is preserved in [`session-phase-skills-prompts.md`](session-phase-skills-prompts.md).

That future work should decide:
- whether `one session = one phase` is mandatory, default, or only for certain execution shapes
- which phases are session-bounded
- which exceptions exist for direct-path work
- which artifacts mark phase completion
- how `workflow-plan.md` should represent `phase complete` and `ready for next session`

Until that work is done, this note is only a preserved design direction, not an active workflow rule.
