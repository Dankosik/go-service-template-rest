---
name: planning-and-task-breakdown
description: "Turn approved `spec.md` plus required compact or split design context into a dependency-ordered, verifiable `tasks.md` ledger for this repository, then run the post-ledger task review/readiness check before implementation handoff. Use after `spec.md` is stable, lean compact design or triggered technical-design artifacts are approved or explicitly skipped, mandatory technical design review is reconciled when separate design depth exists, and required challenge gates are reconciled, whenever implementation should be driven from planning artifacts rather than improvised from the decision/design record. Reach for this when executable task order, checkpoints, or parallelism are not obvious. Skip unresolved architecture/API/data/security/reliability decisions and skip actual coding."
---

# Planning And Task Breakdown

## Purpose
Turn stable decisions plus approved compact or split technical design context into a `tasks.md` executable task ledger that reaches the accepted target state, stays honest about dependencies, names only the checkpoints and proof obligations the work actually needs, and passes a post-ledger review before implementation handoff.

## Outcome-First Operating Rules
- Start by naming the skill-specific outcome, success criteria, constraints, available evidence, and stop rule.
- Treat workflow steps as decision rules, not a ritual checklist. Follow exact order only when this skill or the repository contract makes the sequence an invariant.
- Use the minimum context, references, tools, and validation loops that can change the deliverable; stop expanding when the quality bar is met.
- Before acting, resolve prerequisite discovery, lookup, or artifact reads that the outcome depends on; parallelize only independent evidence gathering and synthesize before the next decision.
- Prefer bounded assumptions and local evidence over broad questioning; ask only when a missing fact would change correctness, ownership, safety, or scope.
- When evidence is missing or conflicting, retry once with a targeted strategy or label the assumption, blocker, or reopen target instead of treating absence as proof.
- Finish only when the requested deliverable is complete in the required shape and verification or a clearly named blocker/residual risk is recorded.

## Scope
- convert approved decisions from `spec.md` and task-local technical context from lean `Compact Design`, one `design/overview.md`, or split `design/` into dependency-ordered executable tasks
- make `tasks.md` explicit by default for non-trivial implementation work
- attach acceptance criteria, planned verification, checkpoints, and change-surface hints
- close or route blockers and decision gates before coding starts; a ready `tasks.md` must not contain unresolved open questions
- review the completed ledger against `spec.md`, required design context, technical-design-review obligations, and triggered validation or rollout obligations before marking implementation readiness
- preserve coder freedom on local code shape while removing ambiguity about execution order

## Boundaries
Do not:
- make new architecture, API, data, security, reliability, or rollout decisions
- reconstruct missing architecture, ownership, data, or sequence context from `spec.md` alone when compact or split design context should supply it
- write production code, tests, or migrations as the main deliverable
- dump raw research or repeat the whole spec in planning form
- treat `spec.md` as the place for full task breakdown by default
- let `tasks.md` become a second spec, second design bundle, or bloated strategy memo
- hide blocked work behind optimistic task wording
- leave open questions, `TBD` decisions, or unresolved decision gates in a `tasks.md` marked ready for implementation

## Escalate When
Escalate if:
- `spec.md` is not stable enough to derive tasks without reopening design
- non-trivial work is missing lean compact design answers, `design/overview.md`, triggered split design artifacts, or an explicit design-skip/merge rationale
- separate design depth was triggered but mandatory technical design review is missing, `FAIL`, or has unresolved findings
- a conditional design artifact is clearly triggered but missing
- core behavior is still undecided across architecture, API, data, security, reliability, or domain semantics
- the right implementation order depends on a missing migration, compatibility, or ownership decision
- the change cannot be decomposed without inventing detail the spec does not actually approve

## Core Defaults
- `spec.md` is for decisions, lean `Compact Design` or triggered `design/` is for technical context, and `tasks.md` is for the executable task ledger and final implementation handoff.
- For lean-local work, plan from approved `spec.md` with explicit compact design answers; for full-orchestrated or design-triggered work, plan from approved `spec.md` plus triggered `design/` and a reconciled technical design review gate.
- For lean-local and full-orchestrated implementation, default to creating `tasks.md`; direct-path or tiny work may skip a separate ledger only with an explicit waiver.
- Keep `tasks.md` task-local to the active spec-first bundle. Do not use a repository-root or unrelated ledger as the current implementation handoff unless workflow control explicitly reopens it and records the resume route.
- Planning is the last artifact-producing phase before code, but the completed ledger must still pass the post-ledger task review/readiness gate before implementation starts. If later `workflow-plans/review-phase-N.md` or `workflow-plans/validation-phase-N.md` are truly needed for named multi-session routing, create only those files before implementation starts instead of inventing them later. Do not create a coding phase-control file.
- Planning must leave a task-ledger review and implementation-readiness result for the handoff: `PASS`, `CONCERNS`, `FAIL`, or `WAIVED`. `CONCERNS` needs named accepted risks and proof obligations, not open questions; `FAIL` names the earlier phase to reopen; `WAIVED` stays limited to explicit tiny/direct-path/prototype scope.
- Before marking `tasks.md` approved for non-trivial work, default to a read-only task-ledger review fan-out when implementation readiness depends on independent lenses. Typical lanes are coverage/traceability, dependency ordering, proof/QA, and any triggered API, data, security, reliability, delivery, performance, observability, or rollout lens.
- Each ledger-review lane reviews the draft ledger against approved artifacts only; no lane edits `tasks.md` or makes final readiness decisions. The orchestrator repairs planning or reopens from fan-in.
- Skipping or scoping down ledger-review fan-out requires `Ledger-review fan-out rationale:` that explicitly evaluates coverage/traceability, dependency ordering, proof/QA, and every triggered domain lens, then explains why each omitted lane cannot change readiness.
- Implementation readiness remains `FAIL` or blocked unless task-ledger review fan-out status is recorded or `Ledger-review fan-out rationale:` explains why local review covers every readiness risk.
- A `PASS`, `CONCERNS`, or `WAIVED` handoff must be closed for implementation. Do not approve a ledger that still needs implementation to choose architecture, ownership, contract, sequencing, rollout, or validation policy.
- For long-running, multi-slice, or resumable implementation, make the ledger Goal-ready: one objective, one stopping condition, read-first context, preserved constraints, checkpoint/progress rules, and evidence fields that let Codex audit completion without relying on chat memory.
- Planning must not treat a design author's handoff as review. Separate design depth requires a distinct technical design review gate before task breakdown can be approved.
- If technical design review passed with `CONCERNS`, carry each accepted design risk and proof obligation into `tasks.md`, `test-plan.md`, `rollout.md`, or the implementation-readiness handoff; do not leave it only in the review record.
- When the planning pass generates or materially changes workflow-control files for full-orchestrated, high-risk, complex, or agent-backed work, expect a read-only `workflow-plan-adequacy-challenge` before handoff; lean-local work without workflow-control artifacts uses the recorded inline/local check instead.
- Prefer a single target-state implementation ledger over phased delivery. Use phase or checkpoint labels only for dependency ordering, reviewability, or proof boundaries, not to defer known in-scope work.
- Prefer dependency-ordered vertical slices over horizontal subsystem dumps when possible.
- Keep tasks small enough to implement, verify, and review in one focused session.
- Planning closes the accepted target-state scope, not just an initial implementation slice. If a downstream issue is in scope and affects production readiness, include it in the ledger or reopen the owning earlier phase. Follow-ups are allowed only for explicit non-goals or proof-only consequences, not target-state cleanup.
- If the approved artifact chain selects a new dependency, OSS integration, or custom implementation after due diligence, carry the required integration, license/security, generation, configuration, drift, and proof tasks into the ledger. If due diligence is missing for custom infrastructure, a new runtime dependency, or a meaningful helper/abstraction, reopen specification or technical design instead of asking implementation to decide.
- If the approved artifact chain selects a design or system-design pattern, carry the pattern-preserving implementation constraints and proof obligations into the ledger. If Pattern Fit Diligence is missing for an invented architecture, workflow, integration, resilience, consistency, data-flow, or abstraction shape, reopen research, specification, or technical design instead of asking implementation to choose the pattern.
- For replacement work, missing in-scope legacy cleanup is a planning-readiness failure: the ledger must task removal/refactor work, retained-surface owner/reason/proof/exit-condition recording, or explicit not-applicable proof instead of leaving the old surface for implementation to interpret.
- For dedicated planning sessions, this pass ends at approved `tasks.md`; implementation begins in a new session unless an upfront direct/lean waiver was already recorded.
- Put risky or dependency-establishing work early.
- Use checkpoints to create real stop points, not ritual paperwork.
- Do not let `tasks.md` absorb phase strategy, design decisions, or speculative tasking that should reopen design.

## Lazily Loaded References
Keep this file as the operating contract. References are compact rubrics and example banks, not exhaustive checklists or documentation dumps. Load at most one reference by default; load multiple only when the task clearly spans independent decision pressures, such as dependency ordering plus implementation-readiness proof. Treat repository-local `AGENTS.md`, `docs/spec-first-workflow.md`, stable `spec.md`, approved `design/`, and existing task artifacts as higher authority than any example.

| Reference | Load For Symptom | Behavior Change |
| --- | --- | --- |
| `references/phase-strategy-examples.md` | phase boundaries, session stops, review/validation checkpoints, or single-pass versus phased execution are unclear | chooses a target-state ledger with real checkpoints instead of partial phased delivery, a giant "implement everything" phase, or a ceremony-only checkpoint |
| `references/dependency-ordered-task-ledgers.md` | task order, `[P]` markers, generated artifacts, migrations, or source-of-truth-first sequencing is unclear | derives dependencies from approved design artifacts and source-of-truth flow instead of marking everything parallel or starting with derived files |
| `references/task-sizing-and-slicing.md` | tasks are too large, too horizontal, too vague, hard to review, or hard to verify in one focused session | splits work into reviewable, proof-bound slices instead of hiding independent surfaces behind one broad task |
| `references/acceptance-criteria-and-proof-obligations.md` | acceptance criteria, proof commands, manual checks, or `CONCERNS` obligations are vague | states task-specific truths and matching proof commands instead of "looks good", "run tests", or optimistic readiness language |
| `references/checkpoints-and-reopen-conditions.md` | stop points, implementation-readiness handoff, blockers, reopen targets, or validation/reconciliation triggers need wording | names executable checkpoints and exact reopen targets instead of asking implementation to improvise or create missing workflow artifacts after coding starts |
| `references/planning-anti-patterns.md` | reviewing a draft plan or ledger for drift, invented decisions, duplicate authority, false parallelism, vague proof, or artifact misuse | challenges smell patterns as triage instead of treating a plausible-looking plan as ready by checklist momentum |

Reference snippets are patterns, not decisions. If an example would require an architecture, API, data, security, reliability, migration, rollout, or ownership choice not already approved in `spec.md` plus required design context, stop and reopen the right earlier phase instead of copying the snippet.

## Planning Workflow

### 1. Confirm Planning Readiness
- Read the stable `spec.md` and the required compact or split design context, not just the chat.
- Confirm that the main decisions, design constraints, ownership boundaries, and proof obligations are explicit, and that no implementation-blocking open questions remain.
- Confirm dependency/OSS due-diligence decisions are explicit when the implementation would add a dependency, integrate OSS, build custom infrastructure, or introduce a material helper/abstraction.
- Confirm Pattern Fit Diligence decisions are explicit when implementation would rely on a non-trivial architecture, workflow, integration, resilience, consistency, data-flow, or abstraction pattern.
- For lean-local work, require explicit `Compact Design` answers or one `design/overview.md`; for split-design work, require `design/overview.md`, `design/component-map.md`, `design/sequence.md`, and `design/ownership-map.md` unless there is an explicit design-skip/merge rationale.
- If one `design/overview.md` or split `design/` exists, require technical design review `PASS` or `CONCERNS` with named accepted risks and proof obligations before planning.
- Treat review findings classified as `blocks_planning`, `reopens_design`, or `reopens_spec` as planning blockers until the owning phase resolves or explicitly reroutes them.
- If the design or spec is not stable enough, stop and escalate instead of guessing.

### 2. Load Execution-Critical Design Context
- Use lean `Compact Design`, `design/overview.md`, or split `design/component-map.md`, `design/sequence.md`, and `design/ownership-map.md` to understand what must land first and what may move in parallel.
- Load triggered conditional artifacts such as `design/data-model.md`, `design/dependency-graph.md`, `design/contracts/`, `test-plan.md`, or `rollout.md` when they affect sequencing.
- Identify what must exist first: schema or config changes, generated artifacts, interfaces, handlers, background workers, tests, docs, or migration controls.
- Make the ordering explicit when one task truly depends on another.
- Do not confuse implementation taste with real dependency.

### 3. Slice The Work
- Prefer one coherent reviewable increment per checkpoint or target-state work bundle.
- When possible, use vertical slices that land observable behavior.
- If the work must start with enabling seams or migration groundwork, say so directly.
- If two tasks must land together to remain safe, explain the coupling.
- Use the design bundle's ownership and sequence constraints to decide where slices can and cannot be separated.

### 4. Write The Task Ledger
- Use `tasks.md` for executable task checkboxes and the final implementation handoff.
- Treat non-trivial `tasks.md` as Goal-ready by default in this repository: start the ledger with a compact Goal contract unless the work is explicitly tiny/direct and not meant to become a Goal handoff.
- The Goal contract includes objective, stopping condition, read-first artifacts, preserved constraints, progress-log rule, and blocked-stop rule.
- Keep the Goal contract derivative of approved artifacts. It must not introduce new scope, new design decisions, or permission to create missing pre-code workflow artifacts during implementation.
- Write the Goal contract so the planning handoff can reuse it directly as a concise Codex Goal objective for the next implementation session, covering every executable task in the approved ledger through final validation.
- When rendering the final chat implementation handoff from that Goal contract, use `.agents/skills/codex-goal-prompt-composer/SKILL.md`.
- Keep detailed artifact lists, constraints, accepted concerns, proof commands, and progress rules in the Goal contract fields and handoff metadata, not inside the objective sentence itself.
- For each executable task, make the action, dependency marker when nontrivial, change surface, and planned verification explicit.
- Include an evidence field for each task or checkpoint when the implementer is expected to update progress during execution.
- Include dependency tasks when relevant: module change, license/security check, transitive dependency review, generated or config updates, integration tests, and any proof obligations from the approved dependency/OSS due diligence.
- Include pattern-preserving tasks when relevant: package or boundary placement, runtime sequence, idempotency/dedup/recovery hooks, validation proof, documentation updates, or negative checks required by the approved Pattern Fit decision.
- When approved decisions replace old code or artifacts, include cleanup audit/removal tasking for old identifiers, routes, configs, commands, tests, fixtures, generated artifacts, scripts, docs, skills, agents, and mirrors that belong to the replaced path.
- For replacement work, add a compact `Legacy cleanup audit` table with columns `Surface`, `Status`, `Evidence`, and `Retention owner/reason/exit`; use exactly `removed`, `refactored`, `retained`, or `not_applicable` for status.
- Name exact file paths when known. When exact file choice is genuinely design-time unknown, name a narrow package or artifact surface instead of vague subsystem labels.
- Do not add a task if tasking it requires inventing a missing design decision; reopen `technical design` instead.
- Do not add a task if tasking it requires resolving an unreviewed or blocking design-review finding; reopen `technical design review` instead.
- Do not add an open-question or decision-gate section to a ready `tasks.md`; route unresolved decisions to the owning earlier phase.
- Add only a short readiness reference in `tasks.md` when useful.

### 5. Add Checkpoints
- Add review and validation checkpoints at natural risk boundaries.
- Each checkpoint should say what must be true before the next phase starts.
- Keep checkpoints proportional; tiny work may need one final checkpoint only.

### 6. Review The Draft Ledger Before Handoff
- Compare the completed `tasks.md` against the approved `spec.md`, required compact or split design context, technical-design-review result, and triggered `test-plan.md` or `rollout.md`.
- Run read-only task-ledger review fan-out by default; otherwise record `Ledger-review fan-out rationale:` that evaluates coverage/traceability, dependency ordering, proof/QA, and every triggered domain lens.
- Keep ledger-review lanes read-only and advisory; they return findings and proof obligations for orchestrator fan-in, not edits or final readiness decisions.
- Confirm every in-scope behavior and preserved constraint is represented in tasking, proof, or explicit non-task rationale.
- Confirm every approved dependency/OSS due-diligence outcome is represented in tasking or explicit non-task rationale, and that missing due diligence for custom code or a new dependency reopens specification or technical design rather than passing to implementation.
- Confirm every approved Pattern Fit outcome is represented in tasking, proof, or explicit non-task rationale, and that missing Pattern Fit Diligence for an invented design shape reopens research, specification, or technical design rather than passing to implementation.
- Confirm every known in-scope legacy surface is represented as remove/refactor work, retained with owner/reason/proof/exit condition, or proven not applicable; missing cleanup tasking reopens planning unless it changes spec or design scope.
- Confirm replacement ledgers use the `Legacy cleanup audit` table; prose-only cleanup classification is too easy to miss during implementation and closeout.
- Confirm task order matches ownership, sequence, dependency, migration, validation, and rollout constraints from the design context.
- Confirm accepted design-review `CONCERNS` are carried as named risks and proof obligations, and that no unresolved `FAIL`, `blocks_planning`, `reopens_design`, or `reopens_spec` finding remains.
- Confirm subagent gates consumed by planning are listed, no lane blocker or material severity conflict remains unresolved, and subagent-derived proof obligations are mapped into `tasks.md`, `test-plan.md`, or `rollout.md`.
- Confirm non-trivial implementation readiness stays `FAIL` or blocked when ledger-review fan-out status and a valid `Ledger-review fan-out rationale:` are both missing.
- If the ledger has only execution-quality gaps, repair planning and review it again; if the gap changes accepted behavior or design, reopen `specification`, `technical design`, or `technical design review`.

## Task Sizing
- `XS`: one tiny local step; prefer keeping it inline unless the surrounding work is already non-trivial
- `S`: one focused task, usually one behavior seam or one enabling change
- `M`: a small feature slice or tightly coupled implementation bundle
- `L+`: break it down further unless the coupling is unavoidable and explicitly named

Break a task down further when:
- it would take more than one focused coding session
- acceptance criteria cannot stay short and concrete
- it touches multiple independent subsystems
- the title needs `and` to stay accurate

## Preferred `tasks.md` Shape
Return ledger text that can drop directly into `tasks.md` with minimal rewriting.

Write tasks as outcome slices, not process scripts:
- each task should name the observable result, the owning surface, and the proof that result needs
- include exact sequence only when dependency, source-of-truth flow, generation order, migration safety, or rollout safety requires it
- avoid telling `go-coder` how to implement internals unless the approved design made that path part of the contract
- use proof obligations and reopen conditions to control risk instead of adding speculative subtasks

Use markdown checkboxes. Each task should include:
- a compact `Goal Contract` header for long-running or resumable work, limited to objective, stopping condition, read-first artifacts, preserved constraints, progress-log rule, and blocked-stop rule;
- an optional compact `Implementation Handoff` header when it helps the next implementation session, limited to consumed artifacts, task-ledger review/readiness status, first task or checkpoint, named `CONCERNS` proof obligations, and reopen target;
- `Subagent gates consumed`, task-ledger review fan-out status or `Ledger-review fan-out rationale:`, and `Subagent-derived proof obligations` lines for non-trivial implementation readiness;
- stable task ID such as `T001`
- checkpoint label, or phase label only when the approved design/user request explicitly uses phases
- optional `[P]` marker only when safe to parallelize
- short action
- exact file path when known, or a narrow package/artifact surface when exact file choice is genuinely design-time unknown
- dependency marker when nontrivial, such as `Depends on: T001`
- proof/verification expectation
- concise continuation lines when dependency, proof, accepted concern, or reopen detail would make a one-line checkbox hard to scan; continuation lines must support the same task item, not turn `tasks.md` into a design note or strategy memo

Example:

```markdown
## Goal Contract

Goal objective: Complete the approved request-ID behavior change by executing this ledger through final validation.
Stopping condition: all tasks are checked, required proof passes or records a concrete blocker, and ledger-owned closeout evidence is current.
Read first: approved `spec.md`, `design/`, and this task ledger.
Do not change: public HTTP semantics other than the approved request-ID echo behavior.
Progress log: update each task's `Evidence` line after running its proof; if blocked, stop and record `Blocked:` under the task.

## Implementation Handoff

Consumes: approved `spec.md`, `design/`, and this task ledger.
Task ledger review: PASS.
Implementation readiness: PASS.
First task: T001.
Accepted concerns: none.
Reopen target: planning if required artifact context is missing.

## Tasks

- [ ] T001 [Checkpoint 1] Update `internal/http/handler.go` to preserve request ID echo behavior. Depends on: none. Proof: `go test ./internal/http`. Evidence: Pending.
- [ ] T002 [Checkpoint 1] [P] Add regression coverage in `internal/http/handler_test.go`. Depends on: T001. Proof: `go test ./internal/http`. Evidence: Pending.
```

Prefer vertical, reviewable slices. Avoid generic tasks like `implement feature`. Keep headers short; if the Goal contract or handoff starts carrying phase strategy or design rationale, trim it back or reopen `design/`. Use multi-line items for readability, not as permission to hide new decisions or broad subplans inside a checkbox.

## Planning Rules
- For direct-path work, a short inline plan may still be enough; do not force `tasks.md` for a tiny change just to satisfy ceremony.
- For lean-local and full-orchestrated non-trivial work, default to `tasks.md` and consume approved `spec.md` plus required design context.
- Create or repair `test-plan.md` or `rollout.md` during planning only when the approved design already contains the needed validation or rollout context. If the companion artifact would require a missing design, compatibility, migration, or rollout decision, reopen technical design instead of filling the gap inside the plan.
- If a required technical design review gate is missing or blocking, reopen technical design review instead of filling the gap inside the plan.
- When later review or validation phase-control files are genuinely needed for named multi-session routing, planning should leave them ready to be created or linked before implementation begins; post-code work should not need to invent new workflow/process artifacts.
- The workflow-control handoff must be challenge-ready: master and phase-local plans should make phase status, blockers, stop rules, next-session start, the next-session context bundle, `tasks.md` status, artifact expectations with trigger rationale, and any named review or validation phase files clear enough for an adequacy challenger to review without reconstructing intent from chat.
- The task-ledger review and implementation-readiness handoff must be explicit: `PASS` may proceed only when the accepted target-state ledger matches the approved artifact chain and needs no hidden architecture, ownership, contract, sequencing, or rollout decisions; `CONCERNS` may proceed only with named risks and proof obligations the implementation can satisfy without replanning; `FAIL` must route to the named earlier phase; and `WAIVED` must remain a narrow tiny/direct-path/prototype exception.
- The task-ledger review must also record consumed subagent gates, lane-derived proof obligations, accepted risks or waivers, and whether unresolved lane blockers or severity conflicts remain.
- If required compact or split design context is missing or inconsistent, reopen specification or technical design instead of inferring the missing context locally.
- If required technical design review is missing or inconsistent, reopen technical design review instead of inferring approval locally.
- Keep planning aligned with repository realities: OpenAPI drift checks, `sqlc` regeneration, migrations, race tests, integration checks, or other real verification surfaces when they actually apply.
- Keep dependency decisions aligned with repository realities: prefer existing repo patterns and current stdlib where sufficient, and include module, license/security, transitive dependency, and drift checks when introducing OSS.
- Keep Pattern Fit decisions aligned with repository realities: prefer established patterns only when their forces match the task, translate them into idiomatic Go and explicit package ownership, and include proof that validates the selected pattern's guarantee rather than only its vocabulary.
- Keep generated and mirrored cleanup source-of-truth order explicit: update owning sources first, regenerate or sync derived artifacts, and add drift proof instead of hand-editing mirrors or generated output as primary cleanup.
- If a phase is not independently mergeable or testable, name the coupling explicitly.
- Prefer sequential phases unless change surfaces are truly disjoint.
- Make the handoff explicit: the planning session stops at approved `tasks.md`, and the first implementation task starts in a new session unless a recorded waiver says otherwise.
- State what should trigger a reopen back into specification or technical design instead of letting coding discover it silently.

## Definition Of Done
The planning pass is complete when:
- the execution order is explicit
- each meaningful task in `tasks.md` has concrete action, dependency/proof context, and planned verification
- checkpoints exist where the risk actually changes
- blocked work is routed to the owning earlier phase instead of left as an open question in ready work
- the next implementation or validation session can start without creating new workflow/process artifacts or missing `tasks.md` to compensate for incomplete planning output
- any mandatory technical design review gate is reconciled before implementation-readiness is marked ready
- task-ledger review confirms `tasks.md` matches approved `spec.md`, required design context, technical-design-review obligations, and triggered validation or rollout obligations before implementation-readiness is marked ready
- implementation-readiness status is explicit and is not `FAIL` unless the planning result is honestly blocked or reopened
- the workflow-control artifacts are ready for the read-only adequacy challenge, or the direct-path skip rationale is explicit
- the next session can start implementation without re-planning or guessing where this planning pass was supposed to stop
- out-of-scope implications are visible as non-goals, accepted risks, or proof-only follow-ups rather than hidden target-state cleanup
- the task ledger is specific enough for `go-coder` to execute without recreating strategy or reverse-engineering missing design context
- selected pattern constraints and proof obligations are clear enough that `go-coder` does not need to reinterpret or choose a design/system pattern
- no unresolved decision gate, `TBD`, or implementation-blocking open question remains in `tasks.md`

## Escalate Or Reject
- task breakdown derived from an unstable spec
- task breakdown that assumes missing compact or split design context instead of escalating
- a phase list with no acceptance criteria or verification
- a `tasks.md` ledger with open questions, unresolved gates, or implementation-time design decisions
- a generic task like `implement the feature`
- horizontal slicing that hides risk and postpones integration until the end
- a `tasks.md` ledger that turns into a strategy memo instead of listing executable, proof-bound work
- planning output that leaves workflow-control routing too vague for adequacy review before handoff
- planning output that duplicates the entire spec instead of turning it into execution work
