---
name: technical-design-session
description: "Own a session dedicated only to task-local technical design for this repository when separate design depth is triggered. Use when specification-review-approved `spec.md` must be turned into a review-ready `design/` bundle or `design/overview.md`, with any triggered workflow-control artifacts updated for the mandatory technical-design-review handoff, without drifting into renewed framing, `tasks.md`, planning, or implementation. Skip direct-path work, lean-local work whose compact design fits in reviewed `spec.md`, unstable or unreviewed `spec.md`, and tasks already in technical design review or later."
---

# Technical Design Session

## Purpose
Run only the technical-design checkpoint for one task-local session.
This wrapper makes the `specification-review-approved spec.md -> design/ -> technical design review` handoff explicit: it produces or updates the task-local design bundle, updates workflow control artifacts, and then stops before the mandatory review gate.

Use `.agents/skills/go-design-spec/SKILL.md` as the deeper integrated design method when cross-domain reconciliation or simplification work is needed.
Do not turn this wrapper into a duplicate of `go-design-spec`; this skill owns session protocol, allowed writes, stop rules, and technical-design-review handoff only.

## Outcome-First Operating Rules
- Start by naming the skill-specific outcome, success criteria, constraints, available evidence, and stop rule.
- Treat workflow steps as decision rules, not a ritual checklist. Follow exact order only when this skill or the repository contract makes the sequence an invariant.
- Use the minimum context, references, tools, and validation loops that can change the deliverable; stop expanding when the quality bar is met.
- Before acting, resolve prerequisite discovery, lookup, or artifact reads that the outcome depends on; parallelize only independent evidence gathering and synthesize before the next decision.
- Prefer bounded assumptions and local evidence over broad questioning; ask only when a missing fact would change correctness, ownership, safety, or scope.
- When evidence is missing or conflicting, retry once with a targeted strategy or label the assumption, blocker, or reopen target instead of treating absence as proof.
- Finish only when the requested deliverable is complete in the required shape and verification or a clearly named blocker/residual risk is recorded.

## Use When
- specification-review-approved or explicitly waived direct-path `spec.md` exists and trigger-based design depth is needed before planning
- lean-local `design/overview.md` or full split `design/` is missing, stale, partial, or internally inconsistent after being triggered
- master `workflow-plan.md` needs the `technical-design` checkpoint completed or updated before the technical-design-review stage
- the task needs triggered design artifacts, triggered conditional artifacts, or a clean design handoff for `planning-and-task-breakdown`
- the session should stay bounded to technical design instead of spilling into `tasks.md` or code

## Skip When
- the work is tiny or direct-path and already has an explicit design-skip rationale
- lean-local `spec.md` already contains enough `Compact Design` context for planning and records why separate design is not expected
- `spec.md` is missing, unstable, contradictory, unreviewed, failed specification review, or still needs research, challenge reconciliation, or specification work
- the task is already in `planning` or a later phase and does not intentionally need design repair
- approved compact or split design context already exists and the real next step is technical design review, planning, or execution
- the request tries to combine technical design with technical design review, writing `tasks.md`, implementation, tests, migrations, or post-code review execution

## Required Inputs
Need only the minimum phase-ready inputs:
- specification-review-approved `spec.md`
- specification-review result when non-trivial `spec.md` exists, including accepted risks and proof obligations when status is `CONCERNS`
- current task-local artifact location
- current `workflow-plan.md`, if present
- current `workflow-plans/technical-design.md`, if present
- existing compact or split `design/` artifacts when this is a continuation or repair pass
- known blockers, assumptions, and specialist outputs that still affect design integrity
- approved dependency/OSS due-diligence decisions or blockers when design selects a new dependency, custom infrastructure, or material helper/abstraction
- approved Pattern Fit Diligence decisions, preserved `research/pattern-fit.md`, or blockers when design selects an architecture, workflow, integration, resilience, consistency, data-flow, or abstraction pattern

If a planning-critical input is missing, record it as a blocker or reopen condition instead of inventing detail.

## Read First
Always read:
- `AGENTS.md`
- `docs/spec-first-workflow.md`
- task-local `workflow-plan.md`, if present
- task-local `workflow-plans/technical-design.md`, if present
- specification-review-approved `spec.md`

Load `docs/repo-architecture.md` when at least one is true:
- this is a fresh non-trivial technical-design pass
- the design touches repository boundaries, ownership seams, runtime flow, new or changed packages, transport/app/infra edges, async behavior, data flow, or dependency direction
- the session must rebuild task-local design from `spec.md` rather than only polishing a small existing artifact
- current design artifacts do not already capture the stable repository baseline needed for this change

It is acceptable to skip `docs/repo-architecture.md` only for a narrow continuation where the current approved design bundle already captures the relevant repository baseline and the session is editing one local seam rather than rebuilding design context.

Then load only the smallest task-local context needed to finish this phase:
- `design/overview.md` first when a design bundle already exists
- the smallest affected existing design artifacts
- targeted `research/*.md` or specialist outputs only when they change design integrity or planning readiness
- canonical contract or schema sources only when needed to design around them, while keeping those runtime sources authoritative

Rules:
- follow `AGENTS.md` if workflow guidance conflicts
- read the master `workflow-plan.md` before the phase-local file when both exist
- do not broad-read unrelated repository areas when the design questions are narrower
- do not reopen framing casually; if the design cannot be completed honestly, route back to `specification`

## Reference Files
Keep this `SKILL.md` as the technical-design-session wrapper protocol. References are compact rubrics and example banks, not exhaustive checklists, source-link collections, or substitutes for repo-local authority.

Default loading rule:
- Load at most one reference by default.
- Load a second reference only when the task clearly spans multiple independent decision pressures, such as uncertain entry readiness plus closing handoff, or required artifact shaping plus conditional artifact triggers.
- Do not load the full `references/` directory by default.
- Check repo-local authority first: `AGENTS.md`, `docs/spec-first-workflow.md`, task-local workflow files, reviewed `spec.md`, specification-review result, and `docs/repo-architecture.md` when triggered.
- Do not copy examples blindly; bind them to the current task's phase, artifacts, blockers, and approved decisions.
- If a reference exposes a missing decision, route back to `specification`. If it exposes missing execution sequencing, record it for the technical-design-review handoff instead of writing `tasks.md`.

Routing table:

| Reference | Load When The Symptom Is | Behavior Change |
| --- | --- | --- |
| [references/technical-design-entry-readiness.md](references/technical-design-entry-readiness.md) | `spec.md`, current phase, allowed writes, or user-requested phase mixing is unclear before technical-design writes. | Blocks, reopens, or narrows the write surface instead of starting design from a draft spec, chat momentum, or an obvious implementation path. |
| [references/repo-architecture-loading-rules.md](references/repo-architecture-loading-rules.md) | Repository boundaries, runtime flow, ownership seams, new packages, dependency direction, generated contracts, async work, data flow, or bootstrap/app/infra edges are in scope. | Loads or cites `docs/repo-architecture.md` before boundary decisions instead of relying on memory or treating generated/runtime surfaces as authority. |
| [references/required-design-artifact-examples.md](references/required-design-artifact-examples.md) | The core design bundle needs creation, repair, or boundary cleanup across overview, component map, sequence, or ownership map. | Splits task-local technical context into the four required artifacts instead of writing one design dump or hiding design content in workflow files. |
| [references/conditional-design-artifact-triggers.md](references/conditional-design-artifact-triggers.md) | Optional design artifacts, `test-plan.md`, or `rollout.md` might be needed, or an existing conditional artifact looks like filler. | Creates only artifacts with real pressure and records `not expected` otherwise instead of creating all optional files or skipping planning-critical context. |
| [references/workflow-plan-technical-design-updates.md](references/workflow-plan-technical-design-updates.md) | Workflow-control files need design artifact status, blocker, repair, reopen, session-boundary, or next-session updates. | Records master and phase-local routing state instead of leaving state in chat, duplicating design content, or letting workflow files disagree. |
| [references/planning-handoff-and-stop-rules.md](references/planning-handoff-and-stop-rules.md) | Technical design is being closed, marked review-ready, blocked, or pressured to continue into review, planning, or implementation. | Hands off to the mandatory technical-design-review stage or reopen target instead of drafting `tasks.md`, code, tests, migrations, generated files, or review output. |

## Allowed Writes
This session may write or update only:
- task-local `design/overview.md`
- task-local `design/component-map.md`
- task-local `design/sequence.md`
- task-local `design/ownership-map.md`
- task-local `design/overview.md` as the only design artifact when lean-local design context needs one file
- task-local conditional design artifacts when triggered:
  - `design/data-model.md`
  - `design/dependency-graph.md`
  - `design/contracts/`
- task-local `test-plan.md` when the repository contract says validation obligations are too large or multi-layered to fit cleanly inside `tasks.md`
- task-local `rollout.md` when migration, backfill/verify choreography, mixed-version compatibility, or failback notes are planning-critical
- task-local `workflow-plan.md`
- task-local `workflow-plans/technical-design.md`
- the `design/`, `design/contracts/`, or `workflow-plans/` directories only when they must be created to hold those artifacts

## Prohibited Actions
Do not:
- reopen problem framing casually or rewrite approved scope just because design is hard
- rewrite `spec.md` instead of escalating back to `specification`
- write `tasks.md`
- start technical design review, implementation, tests, migrations, contract generation, or post-code review execution
- use planning or implementation skills as a backdoor into later phases
- let `workflow-plans/technical-design.md` become a second design bundle or second `tasks.md`
- treat `design/contracts/` as a runtime source of truth; it is design-only task context and canonical runtime sources still win
- create placeholder design files "just in case" when their trigger is not real

## Core Defaults
- this is an orchestrator-facing wrapper, not a replacement for specialist design skills
- `AGENTS.md` owns the workflow contract and `docs/spec-first-workflow.md` owns artifact mechanics
- `spec.md` owns final decisions, lean `Compact Design` or triggered `design/` owns task-local technical context, and `tasks.md` comes later in a different session
- use `go-design-spec` as the deeper design-integrity method when you need integration, contradiction cleanup, or simplification beyond simple artifact upkeep
- for non-trivial work, this session ends at a review-ready or explicitly blocked design bundle; technical design review begins in a new stage unless an explicit compact-design waiver already covers the task
- for non-trivial triggered technical design, first identify independent planning-critical design questions and run or record narrow read-only specialist fan-out before the integrated design pass
- review-ready handoff is invalid unless the active workflow surface records `Design fan-out: complete | scoped_down | local_only | blocked` with lane summaries, candidate-lane analysis, or blocker state
- if design specialist fan-out is skipped, record `Design fan-out rationale:` in `workflow-plans/technical-design.md` with the reason no independent lane would materially improve correctness
- Missing explicit subagent authorization is not a valid `Design fan-out rationale:`. If required design lanes are blocked only because the current prompt lacks explicit subagent/delegation authorization, record `Design fan-out: blocked` and return a next-session prompt with `Subagent authorization:`.
- a single integrated local design pass is eligible only when the reviewed `spec.md` is stable, the current decision frontier has one primary design question, no adjacent domain has an unresolved live fork, and API/data/security/reliability/observability/QA effects can be recorded as `constraint_only`, `proof_only`, `follow_up_only`, or `no new decision required` without changing ownership, contracts, persistence, failure semantics, validation, or rollout
- dependency-sensitive design must preserve the approved stdlib, repository-pattern, mature-OSS, or custom-code decision. If the design needs a new dependency or custom infrastructure not covered by due diligence, route back to `specification` or targeted research instead of hiding the choice in design prose.
- pattern-sensitive design must preserve the approved Pattern Fit decision. If design needs a new or different architecture, workflow, integration, resilience, consistency, data-flow, or abstraction pattern not covered by Pattern Fit Diligence, route back to research or specification instead of hiding the choice in design prose.
- required and conditional artifacts should be explicit in the workflow files as `approved`, `draft`, `missing`, `blocked`, `conditional`, `waived`, or `not expected`, with trigger rationale for `conditional`, `waived`, or `not expected` rather than guessed into existence
- `lean local` may use `spec.md` `Compact Design` or one `design/overview.md`; split core design files are for triggered depth, not the default for bounded work
- review-ready means the design author believes the current decision frontier is closed, but planning still waits for the mandatory technical design review gate
- triggered design artifacts are required questions, not equal-length documents; concise approved artifacts are valid when they make stable boundaries and current changes explicit enough for planning

## Boundary With `go-design-spec`
`technical-design-session` and `go-design-spec` are complementary, not competing:
- use `technical-design-session` to confirm phase entry, choose the exact writable surface, update `workflow-plan.md`, update `workflow-plans/technical-design.md`, enforce the stop rule, and hand off to technical design review
- use `go-design-spec` inside this session when you need the deeper integrated design method for cross-domain reconciliation, simplification, or design-readiness judgment
- do not copy the full `go-design-spec` discipline into this wrapper; link to it and reuse it
- do not let `go-design-spec` blur the session boundary into `tasks.md` or implementation

## Required Design Artifacts
For full-orchestrated or otherwise triggered split-design work, the technical-design session should leave these artifacts approved or explicitly blocked with reasons:
- `design/overview.md` for chosen approach, artifact index with review-bound artifact status and conditional trigger rationale, unresolved seams, and readiness summary
- `design/component-map.md` for affected packages, modules, generated surfaces, adapters, and components; what changes; what remains stable; and which plausible surfaces are intentionally not touched
- `design/sequence.md` for call order, sync or async boundaries, failure points, side effects, recovery or retry boundaries when relevant, and parallel versus sequential behavior
- `design/ownership-map.md` for source-of-truth ownership, allowed dependency direction, generated-code authority, adapter responsibility, and explicit non-owners for critical behavior

These are required as durable answers to design questions only when split design is triggered, not as matched prose quotas. If a narrow change leaves one seam largely stable, the corresponding artifact may stay short and explicitly say so.

Minimum context-first answers:
- `design/component-map.md`: affected packages, generated surfaces, adapters, responsibility changes, stable surfaces, and intentional non-touches
- `design/sequence.md`: runtime order, sync or async boundaries, side effects, failure points, retry or recovery behavior, and parallel versus sequential behavior
- `design/ownership-map.md`: source-of-truth owners, allowed dependency direction, generated-code authority, adapter responsibility, and explicit non-owners

Do not move split-design technical context back into `spec.md`. Lean-local compact design answers may remain in `spec.md` when no split-design trigger is present.

## Conditional Design Artifacts
Create these only when their trigger is real:
- `design/data-model.md` when persisted state, schema, cache contract, projections, replay behavior, or migration shape changes
- `design/dependency-graph.md` when package or module dependency shape changes, generated-code flow changes, or coupling risk must be made explicit
- `design/pattern-fit.md` when selected and rejected design or system-design patterns, source examples, task applicability, Go-fit, and repository-fit are too dense for `design/overview.md`
- `design/contracts/` when API contracts, event contracts, generated contracts, or material internal interfaces change
  - keep this folder design-only; authoritative runtime contracts stay in canonical repository-owned sources such as `api/openapi/service.yaml`, generation inputs, or other contract authorities
- `test-plan.md` when validation obligations are too large or multi-layered to fit cleanly inside `tasks.md`
- `rollout.md` when the task needs migration sequencing, backfill/verify choreography, mixed-version compatibility, or deploy/failback notes

If a trigger is not real, record the artifact as `not expected` with trigger rationale instead of creating filler.

Technical design owns the trigger decision for `test-plan.md` and `rollout.md` when validation or rollout shape affects later planning readiness. Create them here only when the specification-review-approved `spec.md` and current design context are enough to write the artifact honestly. If the trigger is plausible but planning must decide from execution detail, record it as `conditional` with the decision point instead of creating a placeholder.

## Workflow

### 1. Confirm This Session Owns Technical Design Only
- confirm the current phase is `technical-design` or that the task is explicitly resuming this phase
- if the task is still in research or specification, stop and route it back instead of starting design early
- if approved design already exists and the real next step is technical design review or planning, stop and route to the recorded next phase instead of rewriting design
- if the task is direct path or lean local with enough compact design context already recorded, say so directly and do not force this wrapper

### 2. Verify Design Entry Preconditions
- confirm that specification review is `PASS` or `CONCERNS` with named accepted risks and proof obligations, and that `spec.md` is stable enough to derive task-local design without casually reopening framing
- reuse existing design artifacts and specialist outputs before inventing new files
- surface blockers when missing inputs would make the design dishonest

### 3. Load Stable Repository Baseline Only When Needed
- apply the `docs/repo-architecture.md` load rule above
- load it before rebuilding task-local design from scratch when stable boundaries or runtime flow matter
- keep the read set narrow when the session is only repairing one known design seam

### 4. Plan And Run Design Specialist Fan-Out
- first identify the decision frontier as owned questions, not domain labels
- for each candidate domain seam, ask whether a real live fork exists that would change the design bundle or later task ledger
- default to multiple narrow read-only specialist lanes when API, data, security, reliability, observability, delivery, performance, QA, architecture, or dependency seams have unresolved live forks or domain-owned decisions
- open a dependency/OSS research or review lane when a selected library or custom implementation depends on current external maintenance, adoption, license, security, or integration evidence that could change the design
- open a Pattern Fit research or review lane when a selected design/system pattern depends on external descriptions, real-use examples, competing known patterns, or Go/repository-fit evidence that could change the design
- each lane owns one question and one skill or `no-skill`; lanes return evidence, risks, and recommended handoffs only
- collapse duplicate or consequence-only seams into one `go-design-spec` pass and record the classification
- the orchestrator performs fan-in, then uses `go-design-spec` to integrate outcomes into the writable design bundle
- if fan-out is skipped or scoped down, record `Design fan-out rationale:`, `Collapsed seams:`, and `Escalation seams:` in `workflow-plans/technical-design.md`; skipped design fan-out without candidate-lane analysis blocks review-ready status

### 5. Run The Integrated Design Pass
- use `go-design-spec` when cross-domain design integrity, simplification, or contradiction cleanup is the hard part
- keep the work inside technical design: reconcile architecture, API, data, reliability, security, observability, testing, and rollout implications only as far as they shape the design bundle or force a new current decision
- if the design exposes a planning-critical spec gap, stop treating the session as review-ready and route back to `specification`

### 6. Write Or Repair The Design Bundle
- produce or tighten the required core artifacts
- create only the conditional artifacts whose trigger is real
- include dependency/OSS due-diligence consequences in the design bundle when they affect package boundaries, adapter shape, configuration, generation, security/license proof, or validation
- include Pattern Fit consequences in the design bundle when the selected pattern affects ownership, sequence, package boundaries, failure behavior, data flow, validation, rollout, or Go implementation shape
- keep `design/overview.md` as the entrypoint and link surface for the bundle, with required artifact status and conditional trigger rationale visible when the bundle is review-bound
- for `test-plan.md` and `rollout.md`, write the artifact only when the validation or rollout shape is design-ready; otherwise record the conditional trigger and decision point for planning
- keep technical design in `design/`, `test-plan.md`, or `rollout.md` where appropriate; do not absorb it into `spec.md` or phase-control files

### 7. Write Or Repair `workflow-plans/technical-design.md`
- record only the local orchestration for this phase:
  - phase status
  - whether the pass is fresh, resumed, or repair work
  - `Design fan-out: complete | scoped_down | local_only | blocked`
  - design specialist lane set, fan-in result, or `Design fan-out rationale:`
  - candidate seams, collapsed seams, and escalation seams when lanes are skipped or scoped down
  - completion marker
  - local stop rule
  - next action
  - blockers
  - what can run in parallel
  - design artifacts written, approved, blocked, or still pending
- keep this file routing-only; it must not replace the design bundle or turn into `tasks.md`

### 8. Write Or Repair `workflow-plan.md`
- update current phase status, design artifact status, blockers, reopen conditions, and next-session routing
- record whether technical design review can start next, whether the task must return to `specification`, or whether technical design remains blocked
- keep the master file as routing/control, not as a second design document

### 9. Stop At The Boundary
- once the design bundle and workflow artifacts are consistent, stop
- do not begin technical design review, `tasks.md`, implementation work, or validation execution in this session

## Required Master `workflow-plan.md` Updates
Every completed or blocked pass must update the master file with:
- current phase set to `technical-design` and current phase status
- link or status for `workflow-plans/technical-design.md`
- status for each triggered design artifact, including whether design stayed merged in `spec.md`, one `design/overview.md`, or split design
- status for each triggered conditional artifact, including `test-plan.md` or `rollout.md` when applicable; include a short trigger rationale for `not expected`, `conditional`, or `waived` statuses rather than a bare label
- design specialist fan-out status, lane summary, fan-in result, or explicit `Design fan-out rationale:`
- blockers, accepted assumptions, reopen conditions, and any reason the next session cannot start with technical design review
- `Session boundary reached`
- `Ready for next session`
- `Next session starts with`
- `Next session context bundle` as an always-present field: say default resume order is sufficient, or list exact artifact paths and one-line reasons for task-specific resume context
- updated artifact status for `spec.md`, `design/`, `tasks.md`, and any triggered later artifacts

Do not leave review readiness or the reopen point implicit in chat.

## Expected Outputs
A finished technical-design session produces only technical-design-phase artifacts and routing:
- updated or newly created triggered `design/` artifacts
- updated or newly created conditional design artifacts when their trigger is real
- updated or newly created `workflow-plan.md`
- updated or newly created `workflow-plans/technical-design.md`
- an honest technical-design phase status such as `complete` or `blocked`, plus a separate routing state such as `reopen specification` when relevant

It does not produce `tasks.md`, implementation code, migration scripts, or technical design review output.

## Required Final Chat Handoff
When this session ends with `Session boundary reached: yes` and `Ready for next session: yes`, the final chat response must include a `Recommended next-session prompt` section with one copy-pastable fenced text block.

Derive that prompt from the recorded workflow handoff state, not memory:
- `Next session starts with`
- `Next session context bundle`
- this phase's stop rule
- blockers, accepted assumptions, accepted risks, or reopen conditions that still matter
- the expected artifact or output for the next session

Assume the next session cannot see this chat. Make the prompt self-contained for the next phase but selective: include the recorded objective and current state, exact paths, phase names, task IDs, blocker names, accepted decisions, accepted assumptions or risks, proof obligations, and one-line reasons for non-obvious context files. Omit generic repo rules, resolved history, broad summaries, and artifact dumps that the next agent can read from the named files.

Rules:
- keep the prompt chat-only; do not write it into workflow artifacts or create a new artifact for it
- target the recorded next phase or reopen route exactly
- tell the next agent which files to read first, the immediate objective, important constraints, and expected outputs
- for any non-trivial next phase or reopen target that may use read-only lanes, include `Subagent authorization: I explicitly request and authorize read-only subagents, delegation, and parallel agent work for every repository workflow gate that requires or benefits from fan-out in this session. Spawn the required read-only lanes without asking again; the orchestrator retains final authority and reconciles results.`
- if there is no next session or `Ready for next session: no`, do not invent a prompt

## Phase-Local Stop Condition
The session is complete when one of these is true:
- review-ready handoff:
  - design fan-out status is `complete`, `scoped_down`, or `local_only` with lane evidence or valid candidate-lane analysis; `blocked` or absent status is not review-ready
  - triggered design artifacts are approved, or design is explicitly merged/waived by the lean-local rationale
  - triggered conditional artifacts are approved or explicitly not expected
  - planning-critical contradictions are resolved or made explicit
  - remaining downstream effects are classified as `forces new decision`, `forces handoff`, `forces proof obligation`, or `no new decision required`
  - master and phase-local workflow artifacts agree that the next session starts with `technical design review`
- blocked or loop-back handoff:
  - the missing decision or contradiction is explicit
  - the workflow artifacts record why technical design cannot finish honestly
  - the next session is routed back to `specification` or remains in `technical-design`

In all cases:
- `workflow-plan.md` and `workflow-plans/technical-design.md` must agree on phase status, blockers, and next action
- the stop rule must remain "do not begin technical design review, planning, or implementation in this session"

## Technical Design Review Handoff
When the session is review-ready, hand technical design review exactly this:
- specification-review-approved `spec.md`
- specification-review result and any accepted risks or proof obligations
- approved lean compact design context, one `design/overview.md`, or triggered split design artifacts
- approved triggered conditional design artifacts, or explicit `not expected` rationale
- any triggered `design/data-model.md`, `design/dependency-graph.md`, or `design/contracts/`
- any triggered `design/pattern-fit.md` or preserved `research/pattern-fit.md` when pattern comparison affects design readiness
- `test-plan.md` or `rollout.md` when triggered, or an explicit workflow note that they are not expected
- unresolved assumptions, accepted trade-offs, non-goals, and reopen conditions that review and later planning must preserve
- the expected review question, gate output, and any proof obligations already forced by the design
- master and phase-local workflow artifacts updated so the next session starts with `technical design review`

The handoff is the review packet. It should be complete enough for the reviewer to classify findings and recommend `PASS`, `CONCERNS`, or `FAIL` without rediscovering which artifacts are authoritative or which conditional artifacts are expected.

If that handoff is not honest yet, route backward or stay blocked instead of drafting `tasks.md` as a workaround.

## Escalate When
Escalate instead of forcing output when:
- `spec.md` is missing, unstable, or still contradicts itself in a planning-critical way
- a requested design artifact would be filler because its trigger is not real
- the task is so small that a dedicated design session would be ceremony and a recorded design-skip rationale is the right answer
- the request tries to combine technical design with technical design review, planning, or implementation
- stable repository architecture context materially matters and has not been loaded yet
- a required design artifact cannot be completed honestly without reopening `spec.md`
- phase control already shows that the task has moved to planning or later

## Anti-Patterns
- using this wrapper as a way to rewrite `spec.md` or sneak into `tasks.md`
- copying `go-design-spec` into this file instead of treating it as the deeper method
- letting `design/contracts/` drift into runtime source-of-truth authority
- creating every conditional artifact "just in case"
- marking technical design complete while artifact status, blockers, or next-session routing remain implicit
- starting implementation work because the design "already feels clear"
