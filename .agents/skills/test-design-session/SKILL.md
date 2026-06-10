---
name: test-design-session
description: "Own one session dedicated only to task-local test design for this repository. Use after specification review and required technical design review are approved, when scenario design, proof levels, fail-before expectations, quality gates, or a `test-plan.md` artifact are triggered before planning. Produce or repair `test-plan.md` or record explicit `test-design: not expected` rationale without drifting into `tasks.md`, implementation, or writing tests."
---

# Test Design Session

## Purpose
Run only the test-design checkpoint for one task-local session.
This wrapper makes the `specification-review-approved spec.md -> approved compact/split design -> technical design review -> test-design -> planning` handoff explicit: it designs concrete behavior scenarios and proof obligations before task breakdown, updates workflow control artifacts, and stops before planning.

Use `.agents/skills/go-qa-tester-spec/SKILL.md` as the deeper scenario-design method.
Do not turn this wrapper into a test implementation skill; `go-qa-tester` and coding happen later from an approved `tasks.md`.

## Outcome-First Operating Rules
- Start by naming the skill-specific outcome, success criteria, constraints, available evidence, and stop rule.
- Treat workflow steps as decision rules, not a ritual checklist. Follow exact order only when this skill or the repository contract makes the sequence an invariant.
- Use the minimum context, references, tools, and validation loops that can change the deliverable; stop expanding when the quality bar is met.
- Before acting, resolve prerequisite discovery, lookup, or artifact reads that the outcome depends on; parallelize only independent evidence gathering and synthesize before the next decision.
- Prefer bounded assumptions and local evidence over broad questioning; ask only when a missing fact would change correctness, ownership, safety, or scope.
- When evidence is missing or conflicting, retry once with a targeted strategy or label the assumption, blocker, or reopen target instead of treating absence as proof.
- Finish only when the requested deliverable is complete in the required shape and verification or a clearly named blocker/residual risk is recorded.

## Use When
- the active workflow phase is `test-design`
- a reviewed `spec.md` and required approved design context exist, and planning needs concrete test scenarios before `tasks.md`
- technical design review passed or was explicitly not expected for direct or compact lean work
- the work changes behavior enough that generic validation prose would leave test scenarios, proof level, or pass/fail observables to implementation
- specification review or technical design review produced QA proof obligations that need scenario IDs before planning
- an existing `test-plan.md` is missing, stale, too vague, or inconsistent with approved behavior
- workflow-control files need the test-design phase status, fan-out result, blocker, or next-session route updated

## Skip When
- the work is direct-path or tiny enough that `test-design: not expected` can be recorded with a narrow rationale
- reviewed `spec.md` or required design context is missing, stale, failed, or still under repair
- technical design review is required but missing, failed, or unresolved
- the task is already in planning or implementation and does not intentionally reopen test design
- the request is to write or fix tests from an already approved task ledger; use `go-qa-tester` or implementation skills instead
- the request tries to combine test design with `tasks.md`, production code, test code, migrations, generated artifacts, or validation closeout

## Required Inputs
Need only the minimum phase-ready inputs:
- specification-review-approved `spec.md`
- specification-review result, including accepted risks and proof obligations when status is `CONCERNS`
- approved compact design in `spec.md`, approved `design/overview.md`, or approved split `design/` artifacts
- technical design review result when separate design depth exists
- existing `test-plan.md`, if this is a continuation or repair pass
- current task-local `workflow-plan.md`, if present
- current task-local `workflow-plans/test-design.md`, if present
- repository command surface, normally `docs/build-test-and-development-commands.md`, when proof commands or integration boundaries need current names
- relevant source or test surfaces only when needed to identify observable behavior, current fixture seams, determinism constraints, or proof carriers

If an input is missing, record the blocker and reopen the owning earlier phase instead of inventing behavior or proof semantics.

## Read First
Always read:
- `AGENTS.md`
- `docs/spec-first-workflow.md`
- `docs/spec-first-workflow/phases/test-design.md`
- `.agents/skills/go-qa-tester-spec/SKILL.md`
- reviewed task-local `spec.md`

Then read only what the active task needs:
1. task-local `workflow-plan.md`, if present
2. task-local `workflow-plans/test-design.md`, if present
3. specification-review result
4. approved compact or split design context
5. technical design review result when separate design depth exists
6. existing `test-plan.md`, when present
7. relevant command docs and narrow source/test surfaces needed to design deterministic proof

Rules:
- follow `AGENTS.md` if workflow guidance conflicts
- read the master `workflow-plan.md` before the phase-local file when both exist
- do not broad-read repository code when the scenario frontier is narrower
- do not reopen framing casually; if scenarios cannot be designed honestly, route back to `specification`, `technical design`, or `technical design review`

## Allowed Writes
This session may write or update only:
- task-local `test-plan.md`
- task-local `workflow-plan.md`
- task-local `workflow-plans/test-design.md`
- the `workflow-plans/` directory only when it must be created to hold the phase-local file

## Prohibited Actions
Do not:
- write or repair `tasks.md`
- write production code, test code, migrations, generated artifacts, fixtures, or scripts
- start task-review/readiness, implementation, validation, or closeout
- change approved behavior, contracts, package ownership, rollout decisions, or implementation sequencing
- use planning or implementation skills as a backdoor into later phases
- treat generic validation commands as scenario design
- mark test design approved while scenario classes, proof levels, pass/fail observables, fail-before expectations, or reopen targets are still implicit

## Core Defaults
- `test-plan.md` is conditional but required when scenario design is too dense or risk-bearing for `tasks.md`
- a no-test-plan path must be explicit: `Test design: not expected`, trigger test, proof carrier, and reopen condition
- scenario IDs are stable and task-traceable, using `TD-001`, `TD-002`, and so on
- each scenario names behavior, risk, inputs/setup, proof level, pass/fail observable, fail-before expectation or waiver, determinism constraint, owner layer, and source anchor
- proof levels are explicit: `unit`, `integration`, `contract`, `e2e-smoke`, or a repo-specific named proof level
- failure-path, boundary, regression, and invariant scenarios are first-class; happy paths alone are not enough for non-trivial behavior changes
- test design owns scenario quality, not test implementation details
- planning owns mapping approved scenarios into executable `tasks.md` tasks
- implementation may not invent missing scenario classes or proof levels after planning
- `Test-design fan-out: complete | scoped_down | local_only | blocked | not_expected` must be recorded before approving or skipping this phase
- Missing explicit subagent authorization is not a valid scoped-down or local-only rationale. If required QA or specialist lanes are blocked only because the current prompt lacks explicit subagent/delegation authorization, record `Test-design fan-out: blocked` and return a next-session prompt with `Subagent authorization:`.

## Workflow

### 1. Confirm This Session Owns Test Design Only
- confirm the current phase is `test-design` or that upstream artifacts route here next
- if specification, design, or technical design review is missing or blocking, stop and reopen that phase
- if test design is clearly not triggered, record the no-test-plan rationale and route to planning
- if the real request is to implement tests from an approved ledger, stop and route to implementation instead

### 2. Identify The Scenario Frontier
- extract behavior deltas, preserved invariants, failure semantics, compatibility constraints, and accepted proof obligations
- identify which risks require scenario IDs before planning
- distinguish scenario decisions from implementation choices such as helper names, fixture internals, and file-local test structure
- if a scenario depends on an undecided behavior or ownership question, reopen the owning earlier phase

### 3. Run Or Record Test-Design Fan-Out
- use narrow read-only lanes when independent specialist evidence can change scenario correctness, proof level, determinism, or reopen targets
- use `go-qa-tester-spec` for the main QA scenario design lens
- add focused specialist lanes only when their domain can change proof obligations, such as security, reliability, data, API contract, concurrency, performance, or observability
- if fan-out is skipped, record a local-only rationale naming candidate lanes considered and why none could change test-design readiness
- reconcile lane findings into final scenario decisions; lane output is evidence, not authority

### 4. Write Or Repair `test-plan.md`
When triggered, write a concise task-local `test-plan.md` with:
- `Test-design fan-out` status and evidence pointer
- approved input artifacts and their versions or dates
- scenario matrix with stable `TD-*` IDs
- scenario source anchors to spec, review, design, or technical design review obligations
- proof level and expected proof command or command family
- pass/fail observable and fail-before expectation or waiver
- determinism and fixture/data constraints
- explicit quality gates for planning and implementation
- reopen targets when a scenario cannot be represented without changing behavior, design, rollout, or ownership

Keep the artifact scenario-focused. Do not write task order, implementation steps, or test-code skeletons.

### 5. Update Workflow Control
If workflow-control files exist or are triggered, update them with:
- current phase: `test-design`
- phase status: `approved`, `not_expected`, `blocked`, or `needs_repair`
- `test-plan.md` status and path, or no-test-plan rationale
- `Test-design fan-out` status and summary
- blockers and reopen targets
- next action, normally `planning`
- stop rule and next-session context bundle

Do not store the full ready-to-paste next-session prompt in workflow files; render it only in chat at the phase boundary.

### 6. Stop At The Boundary
- stop after `test-plan.md` or the no-test-plan rationale is recorded and workflow control agrees
- do not start `tasks.md`
- hand off to planning only when scenarios are approved or test design is explicitly not expected

## Required Handoff To Planning
When test design completes successfully, the handoff is:
- reviewed `spec.md`
- specification-review result
- approved compact or split design context
- technical design review result when separate design depth exists
- approved `test-plan.md` with `TD-*` scenario IDs, or explicit `test-design: not expected` rationale
- workflow-control status showing planning may start
- accepted risks and proof obligations that planning must carry into `tasks.md`

## Required Final Chat Shape
At the phase boundary, include:
- whether test design is `approved`, `not_expected`, or `blocked`
- the path to `test-plan.md` when created or repaired
- the `Test-design fan-out` result
- the next phase or reopen target
- a copy-pastable next-session prompt for planning when planning is next

## Completion Criteria
The session is complete when:
- triggered test design has an approved `test-plan.md`, or the no-test-plan rationale is explicit and reviewable
- scenario IDs, proof levels, pass/fail observables, fail-before expectations, determinism constraints, and reopen targets are concrete enough for planning
- no unresolved behavior, ownership, design, or proof-level decision remains hidden for implementation
- workflow-control files, when present, agree on test-design status and next action
- the session has stopped before planning, task review, implementation, or test writing
