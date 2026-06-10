---
name: planning-and-task-breakdown
description: "Turn specification-review-approved `spec.md` plus required compact or split design context into a dependency-ordered, verifiable draft or repaired `tasks.md` ledger for this repository, then prepare the handoff to the separate task-review/readiness gate. Use after `spec.md` is stable, mandatory specification review is reconciled for non-trivial work, lean compact design or triggered system/integration and Go code ownership artifacts are approved or explicitly skipped, mandatory technical design review is reconciled when separate design depth exists, and required challenge gates are reconciled, whenever implementation should be driven from planning artifacts rather than improvised from the decision/design record. Reach for this when executable task order, checkpoints, or parallelism are not obvious. Skip unresolved architecture/API/data/security/reliability/package-ownership decisions and skip actual coding."
---

# Planning And Task Breakdown

## Purpose
Turn stable reviewed decisions plus approved compact or split technical design context into a `tasks.md` executable task ledger that reaches the accepted target state, stays honest about dependencies, names only the checkpoints and proof obligations the work actually needs, and is ready for the separate task-review/readiness gate.

## Outcome-First Operating Rules
- Start by naming the skill-specific outcome, success criteria, constraints, available evidence, and stop rule.
- Treat workflow steps as decision rules, not a ritual checklist. Follow exact order only when this skill or the repository contract makes the sequence an invariant.
- Use the minimum context, references, tools, and validation loops that can change the deliverable; stop expanding when the quality bar is met.
- Before acting, resolve prerequisite discovery, lookup, or artifact reads that the outcome depends on; parallelize only independent evidence gathering and synthesize before the next decision.
- Prefer bounded assumptions and local evidence over broad questioning; ask only when a missing fact would change correctness, ownership, safety, or scope.
- When evidence is missing or conflicting, retry once with a targeted strategy or label the assumption, blocker, or reopen target instead of treating absence as proof.
- Finish only when the requested deliverable is complete in the required shape and verification or a clearly named blocker/residual risk is recorded.

## Scope
- convert reviewed decisions from `spec.md` and task-local technical context from lean `Compact Design`, one `design/overview.md`, or split `design/` into dependency-ordered executable tasks
- make `tasks.md` explicit by default for non-trivial implementation work
- attach acceptance criteria, planned verification, checkpoints, and change-surface hints
- close or route blockers and decision gates before coding starts; a ready `tasks.md` must not contain unresolved open questions
- self-check the completed ledger against reviewed `spec.md`, specification-review obligations, required design context, technical-design-review obligations, and triggered validation or rollout obligations before handing it to task-review/readiness
- preserve coder freedom on local code shape while removing ambiguity about execution order

## Boundaries
Do not:
- make new architecture, API, data, security, reliability, or rollout decisions
- reconstruct missing architecture, system behavior, package/file ownership, data, sequence, cleanup, or test-ownership context from `spec.md` alone when compact or split design context should supply it
- write production code, tests, or migrations as the main deliverable
- dump raw research or repeat the whole spec in planning form
- treat `spec.md` as the place for full task breakdown by default
- let `tasks.md` become a second spec, second design bundle, or bloated strategy memo
- hide blocked work behind optimistic task wording
- leave open questions, `TBD` decisions, or unresolved decision gates in a `tasks.md` marked ready for task review

## Escalate When
Escalate if:
- `spec.md` is not stable enough to derive tasks without reopening design
- mandatory specification review is missing, `FAIL`, stale after repair, or has unresolved findings
- non-trivial work is missing lean compact design answers, `design/overview.md`, triggered system/integration design, triggered Go code ownership design, triggered split design artifacts, or an explicit design-skip/merge rationale
- separate design depth was triggered but mandatory technical design review is missing, `FAIL`, or has unresolved findings
- a conditional design artifact is clearly triggered but missing
- core behavior is still undecided across architecture, API, data, security, reliability, or domain semantics
- the right implementation order depends on a missing migration, compatibility, system ownership, package/file ownership, cleanup, or test-ownership decision
- the change cannot be decomposed without inventing detail the spec does not actually approve

## Core Defaults
- `spec.md` is for decisions, lean `Compact Design` or triggered `design/` is for technical context, and `tasks.md` is for the executable task ledger and final implementation handoff.
- For lean-local work, plan from reviewed `spec.md` with explicit compact design answers; for full-orchestrated or design-triggered work, plan from reviewed `spec.md` plus triggered `design/` and a reconciled technical design review gate.
- For lean-local and full-orchestrated implementation, default to creating `tasks.md`; direct-path or tiny work may skip a separate ledger only with an explicit waiver.
- Keep `tasks.md` task-local to the active spec-first bundle. Do not use a repository-root or unrelated ledger as the current implementation handoff unless workflow control explicitly reopens it and records the resume route.
- Planning is the last artifact-producing phase before code, but the completed ledger must still pass the separate post-ledger task-review/readiness gate before implementation starts. If later `workflow-plans/review-phase-N.md` or `workflow-plans/validation-phase-N.md` are truly needed for named multi-session routing, create only those files before implementation starts instead of inventing them later. Do not create a coding phase-control file.
- Planning leaves task-ledger review and implementation-readiness fields as `pending_task_review`. The `docs/spec-first-workflow/phases/task-review-readiness.md` phase assigns final `PASS`, `CONCERNS`, `FAIL`, or `WAIVED`.
- Planning may suggest likely task-review lenses, but task-review/readiness owns fan-out, local-only rationale, final readiness status, and implementation handoff approval.
- A draft ledger must not need implementation to choose architecture, system behavior, package/file ownership, contract, sequencing, rollout, validation policy, cleanup, or test ownership. If the planning self-check finds that gap, repair planning or reopen the owning earlier phase before task-review handoff.
- For long-running, multi-slice, or resumable implementation, make the ledger Goal-ready: one objective, one successful completion condition, a separate blocked-stop condition, read-before-coding context, task-specific read context when useful, preserved constraints, checkpoint/progress rules, resume rules, and evidence fields that let Codex audit completion without relying on chat memory. A recorded blocker is a valid stop, not a successful completion claim.
- Planning must not treat a design author's handoff as review. Separate design depth requires a distinct technical design review gate before task breakdown can be approved.
- Planning must not treat a spec author's handoff as review. Non-trivial `spec.md` requires a distinct specification-review gate before task breakdown can be approved.
- If specification review passed with `CONCERNS`, carry each accepted spec risk and proof obligation into `tasks.md`, `test-plan.md`, `rollout.md`, or the task-review handoff; do not leave it only in the review record.
- If technical design review passed with `CONCERNS`, carry each accepted design risk and proof obligation into `tasks.md`, `test-plan.md`, `rollout.md`, or the task-review handoff; do not leave it only in the review record.
- When the planning pass generates or materially changes workflow-control files for full-orchestrated, high-risk, complex, or agent-backed work, expect a read-only `workflow-plan-adequacy-challenge` before handoff; lean-local work without workflow-control artifacts uses the recorded inline/local check instead.
- Prefer a single target-state implementation ledger over phased delivery. Use phase or checkpoint labels only for dependency ordering, reviewability, or proof boundaries, not to defer known in-scope work.
- Prefer dependency-ordered vertical slices over horizontal subsystem dumps when possible.
- Keep tasks small enough to implement, verify, and review in one focused session.
- Planning closes the accepted target-state scope, not just an initial implementation slice. If a downstream issue is in scope and affects production readiness, include it in the ledger or reopen the owning earlier phase. Follow-ups are allowed only for explicit non-goals or proof-only consequences, not target-state cleanup.
- If the approved artifact chain selects a new dependency, OSS integration, or custom implementation after due diligence, carry the required integration, license/security, generation, configuration, drift, and proof tasks into the ledger. If due diligence is missing for custom infrastructure, a new runtime dependency, or a meaningful helper/abstraction, reopen specification or technical design instead of asking implementation to decide.
- If the approved artifact chain selects a design or system-design pattern, carry the pattern-preserving implementation constraints and proof obligations into the ledger. If Pattern Fit Diligence is missing for an invented architecture, workflow, integration, resilience, consistency, data-flow, or abstraction shape, reopen research, specification, or technical design instead of asking implementation to choose the pattern.
- For replacement work, missing in-scope legacy cleanup is a planning-readiness failure: the ledger must task removal/refactor work, retained-surface owner/reason/proof/exit-condition recording, or explicit not-applicable proof instead of leaving the old surface for implementation to interpret.
- For dedicated planning sessions, this pass ends at draft or repaired `tasks.md` ready for task-review/readiness; implementation begins only after that separate gate passes unless an upfront direct/lean waiver was already recorded.
- Put risky or dependency-establishing work early.
- Use checkpoints to create real stop points, not ritual paperwork.
- Do not let `tasks.md` absorb phase strategy, design decisions, or speculative tasking that should reopen design.

## Lazily Loaded References
Keep this file as the operating contract. References are compact rubrics and example banks, not exhaustive checklists or documentation dumps. Load at most one reference by default; load multiple only when the task clearly spans independent decision pressures, such as dependency ordering plus task-review handoff proof. Treat repository-local `AGENTS.md`, `docs/spec-first-workflow.md`, stable `spec.md`, approved `design/`, and existing task artifacts as higher authority than any example.

| Reference | Load For Symptom | Behavior Change |
| --- | --- | --- |
| `references/phase-strategy-examples.md` | phase boundaries, session stops, review/validation checkpoints, or single-pass versus phased execution are unclear | chooses a target-state ledger with real checkpoints instead of partial phased delivery, a giant "implement everything" phase, or a ceremony-only checkpoint |
| `references/dependency-ordered-task-ledgers.md` | task order, `[P]` markers, generated artifacts, migrations, or source-of-truth-first sequencing is unclear | derives dependencies from approved design artifacts and source-of-truth flow instead of marking everything parallel or starting with derived files |
| `references/task-sizing-and-slicing.md` | tasks are too large, too horizontal, too vague, hard to review, or hard to verify in one focused session | splits work into reviewable, proof-bound slices instead of hiding independent surfaces behind one broad task |
| `references/acceptance-criteria-and-proof-obligations.md` | acceptance criteria, proof commands, manual checks, or `CONCERNS` obligations are vague | states task-specific truths and matching proof commands instead of "looks good", "run tests", or optimistic readiness language |
| `references/checkpoints-and-reopen-conditions.md` | stop points, task-review handoff, blockers, reopen targets, or validation/reconciliation triggers need wording | names executable checkpoints and exact reopen targets instead of asking implementation to improvise or create missing workflow artifacts after coding starts |
| `references/planning-anti-patterns.md` | reviewing a draft plan or ledger for drift, invented decisions, duplicate authority, false parallelism, vague proof, or artifact misuse | challenges smell patterns as triage instead of treating a plausible-looking plan as ready by checklist momentum |

Reference snippets are patterns, not decisions. If an example would require an architecture, API, data, security, reliability, migration, rollout, or ownership choice not already approved in `spec.md` plus required design context, stop and reopen the right earlier phase instead of copying the snippet.

## Planning Workflow

### 1. Confirm Planning Readiness
- Read the stable reviewed `spec.md`, specification-review result, and the required compact or split design context, not just the chat.
- Confirm that the main decisions, design constraints, ownership boundaries, and proof obligations are explicit, and that no implementation-blocking open questions remain.
- Confirm dependency/OSS due-diligence decisions are explicit when the implementation would add a dependency, integrate OSS, build custom infrastructure, or introduce a material helper/abstraction.
- Confirm Pattern Fit Diligence decisions are explicit when implementation would rely on a non-trivial architecture, workflow, integration, resilience, consistency, data-flow, or abstraction pattern.
- Confirm specification review is `PASS` or `CONCERNS` with named obligations for non-trivial `spec.md`.
- For lean-local work, require explicit `Compact Design` answers or one `design/overview.md`; for split-design work, require `design/overview.md`, triggered `design/system-integration.md`, triggered `design/go-code-ownership.md`, `design/component-map.md`, `design/sequence.md`, and `design/ownership-map.md` unless there is an explicit design-skip/merge rationale.
- When Go code ownership design was triggered, confirm the design packet has a source responsibility audit, rejected owner locations, owner package/file or approved placement rule, cleanup owner, and test owner before planning.
- If one `design/overview.md` or split `design/` exists, require technical design review `PASS` or `CONCERNS` with named accepted risks and proof obligations before planning.
- Treat review findings classified as `blocks_planning`, `reopens_design`, or `reopens_spec` as planning blockers until the owning phase resolves or explicitly reroutes them.
- If the design or spec is not stable enough, stop and escalate instead of guessing.

### 2. Load Execution-Critical Design Context
- Use lean `Compact Design`, `design/overview.md`, triggered `design/system-integration.md`, triggered `design/go-code-ownership.md`, or split `design/component-map.md`, `design/sequence.md`, and `design/ownership-map.md` to understand what must land first and what may move in parallel.
- Load triggered conditional artifacts such as `design/data-model.md`, `design/dependency-graph.md`, `design/contracts/`, `test-plan.md`, or `rollout.md` when they affect sequencing.
- Identify what must exist first: schema or config changes, generated artifacts, interfaces, handlers, background workers, tests, docs, or migration controls.
- Make the ordering explicit when one task truly depends on another.
- Do not confuse implementation taste with real dependency.

### 3. Slice The Work
- Prefer one coherent reviewable increment per checkpoint or target-state work bundle.
- Keep one reviewable diff story per task. If the title needs "and" to be accurate, split it unless the approved design makes the coupling inseparable.
- When possible, use vertical slices that land observable behavior.
- If the work must start with enabling seams or migration groundwork, say so directly.
- If two tasks must land together to remain safe, explain the coupling.
- Use the design bundle's ownership and sequence constraints to decide where slices can and cannot be separated.

### 4. Write The Task Ledger
- Use `tasks.md` for executable task checkboxes and the final implementation handoff.
- Treat non-trivial `tasks.md` as Goal-ready by default in this repository: start the ledger with a compact Goal contract unless the work is explicitly tiny/direct and not meant to become a Goal handoff.
- The Goal contract includes objective, completion condition, separate blocked-stop condition, read-before-coding artifacts, task-specific read context when useful, preserved constraints, progress-log rule, resume rule, and blocked-stop rule.
- Keep the Goal contract derivative of approved artifacts. It must not introduce new scope, new design decisions, or permission to create missing pre-code workflow artifacts during implementation.
- Write the Goal contract so the later task-review/readiness phase can reuse it directly when rendering a Codex Goal objective after readiness approval, covering every executable task in the approved ledger through final validation.
- When the later readiness phase renders the final chat implementation handoff from that Goal contract, it uses `.agents/skills/codex-goal-prompt-composer/SKILL.md` and `docs/spec-first-workflow/shared/subagents-and-handoff.md`.
- Keep detailed artifact lists, constraints, accepted concerns, proof commands, and progress rules in the Goal contract fields and handoff metadata, not inside the objective sentence itself.
- For each executable task, make the action, dependency marker when nontrivial, change surface, and planned verification explicit.
- Include structured evidence fields for each task or checkpoint when the implementer is expected to update progress during execution. Prefer `Command/read`, `Result`, `Key output/ref`, `Changed proof files`, and `Residual blocker/narrower claim` unless a one-line evidence field is enough for tiny/direct-path work.
- Do not let the evidence shape imply that skipped, unavailable, failing, stale, or too-narrow proof can satisfy a task. The affected task must stay unchecked with `Blocked:` or a narrower claim.
- Include `Source:` anchors for tasks whose requirements come from non-obvious spec decisions, review findings, design sections, `test-plan.md`, or `rollout.md`, so task-ledger review can trace material work back to approved artifacts.
- For behavior changes and bug fixes, include proof-first or test-first tasking by default. If a failing proof is not useful, add `Proof-first waiver: <reason>` to the relevant task or checkpoint.
- Include a compact task-local implementation quality bar when the work is broad enough that clean code, package ownership, generated-source discipline, dependency/custom-helper discipline, concurrency/lifecycle behavior, observability, or test-layer choices could otherwise be implicit.
- Include dependency tasks when relevant: module change, license/security check, transitive dependency review, generated or config updates, integration tests, and any proof obligations from the approved dependency/OSS due diligence.
- Include pattern-preserving tasks when relevant: package or boundary placement, runtime sequence, idempotency/dedup/recovery hooks, validation proof, documentation updates, or negative checks required by the approved Pattern Fit decision.
- When approved decisions replace old code or artifacts, include cleanup audit/removal tasking for old identifiers, routes, configs, commands, tests, fixtures, generated artifacts, scripts, docs, skills, agents, and mirrors that belong to the replaced path.
- For replacement work, add a compact `Legacy cleanup audit` table with columns `Surface`, `Status`, `Evidence`, and `Retention owner/reason/exit`; use exactly `removed`, `refactored`, `retained`, or `not_applicable` for status.
- Name exact file paths when known. When exact file choice is genuinely design-time unknown, name a narrow package or artifact surface plus the approved placement rule and first-task inspection bounds instead of vague subsystem labels.
- Do not use `implementation decides`, `choose appropriate file`, `place where it fits`, `refactor as needed`, `split if necessary`, `cleanup later`, or equivalent wording for owner, sequence, generated authority, cleanup/test ownership, validation policy, or accepted-risk proof paths.
- Do not add a task if tasking it requires inventing a missing system/integration or Go code ownership decision; reopen the owning design checkpoint instead.
- Do not add a task if tasking it requires resolving an unreviewed or blocking design-review finding; reopen `technical design review` instead.
- Do not add an open-question or decision-gate section to a ready `tasks.md`; route unresolved decisions to the owning earlier phase.
- Add only a short readiness reference in `tasks.md` when useful.

### 5. Add Checkpoints
- Add review and validation checkpoints at natural risk boundaries.
- Each checkpoint should say what must be true before the next phase starts.
- When checkpoints exist, add a compact `Checkpoint Gates` table or equivalent wording that maps checkpoint names to task ranges and required proof/currentness before later tasks rely on the checkpoint.
- Keep checkpoints proportional; tiny work may need one final checkpoint only.

### 6. Self-Check The Draft Ledger Before Task-Review Handoff
- Compare the completed `tasks.md` against the reviewed `spec.md`, specification-review result, required compact or split design context including triggered system/integration and Go code ownership decisions, design fan-out result when separate technical design depth was triggered, technical-design-review result, and triggered `test-plan.md` or `rollout.md`.
- Leave `Task ledger review`, `Implementation readiness`, `Ledger-review fan-out`, and `Ledger-review fan-out rationale` as `pending_task_review`; the separate task-review/readiness phase owns those fields.
- Record likely task-review lenses or risk notes only as handoff context. Do not run review lanes, assign readiness, or render an implementation prompt from this method.
- Confirm every in-scope behavior and preserved constraint is represented in tasking, proof, or explicit non-task rationale.
- Confirm every approved dependency/OSS due-diligence outcome is represented in tasking or explicit non-task rationale, and that missing due diligence for custom code or a new dependency reopens specification or technical design rather than passing to implementation.
- Confirm every approved Pattern Fit outcome is represented in tasking, proof, or explicit non-task rationale, and that missing Pattern Fit Diligence for an invented design shape reopens research, specification, or technical design rather than passing to implementation.
- Confirm design fan-out is `complete`, valid `scoped_down`, or eligible `local_only` for every triggered design checkpoint when separate technical design depth was triggered; missing, `blocked`, or ineligible `local_only` design fan-out reopens the owning design checkpoint instead of passing to implementation.
- Confirm the Goal Contract separates successful completion from blocked-stop behavior; if "recorded blocker" can be read as successful completion, repair planning before task-review handoff.
- Confirm each task has one reviewable diff story, or records why coupled changes cannot be split safely.
- Confirm read context is split into required start context and task-specific context when the artifact list is long or domain-specific; implementation should not have to read every companion artifact before the first safe edit.
- Confirm checkpoint gates identify the proof/currentness required before dependent later tasks proceed.
- Confirm material task requirements have `Source:` anchors or equivalent traceability to approved spec, review, design, test-plan, or rollout artifacts.
- Confirm behavior-change and bug-fix tasks include proof-first/test-first tasking, or an explicit task-level `Proof-first waiver:` with rationale.
- Confirm evidence fields are structured enough for closeout to distinguish passed proof, unavailable proof, blocked proof, and narrower claims.
- Confirm skipped, unavailable, stale, failing, or too-narrow proof cannot satisfy a task checkbox, checkpoint, or completion claim.
- Confirm long-running ledgers have a resume rule that tells a context-blind implementation session where to continue and how much proof to rerun.
- Confirm the task-local implementation quality bar is present or explicitly unnecessary for the task shape.
- Confirm every known in-scope legacy surface is represented as remove/refactor work, retained with owner/reason/proof/exit condition, or proven not applicable; missing cleanup tasking reopens planning unless it changes spec or design scope.
- Confirm replacement ledgers use the `Legacy cleanup audit` table; prose-only cleanup classification is too easy to miss during implementation and closeout.
- Confirm task order matches ownership, sequence, dependency, migration, validation, and rollout constraints from the design context.
- Confirm accepted specification-review and design-review `CONCERNS` are carried as named risks and proof obligations, and that no unresolved `FAIL`, spec-review blocker, `blocks_planning`, `reopens_design`, or `reopens_spec` finding remains.
- Confirm subagent gates consumed by planning are listed, no lane blocker or material severity conflict remains unresolved, and subagent-derived proof obligations are mapped into `tasks.md`, `test-plan.md`, or `rollout.md`.
- Confirm non-trivial implementation readiness remains `pending_task_review` until the separate gate runs.
- If the ledger has only execution-quality gaps, repair planning and self-check it again; if the gap changes accepted behavior or design, reopen `specification`, `specification review`, `technical design`, or `technical design review`.

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
- a compact `Goal Contract` header for long-running or resumable work, limited to objective, completion condition, separate blocked-stop condition, read-before-coding artifacts, task-specific read context when useful, preserved constraints, task-local implementation quality bar when needed, progress-log rule, resume rule, and blocked-stop rule;
- an optional compact `Task-Review Handoff` header when it helps the next phase, limited to consumed artifacts, pending task-ledger review/readiness fields, likely review lenses, accepted proof obligations, and reopen target;
- `Subagent gates consumed`, `Task ledger review: pending_task_review`, `Implementation readiness: pending_task_review`, and task-review handoff notes for non-trivial ledgers;
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
Completion condition: all tasks are checked, required proof passes, task evidence is current, and ledger-owned closeout updates are complete.
Blocked-stop condition: if required proof cannot pass, a required command cannot run, an implementation-blocking decision is missing, or an approved artifact is insufficient, stop with the Goal blocked, leave affected tasks unchecked, record `Blocked:` with evidence and the owning reopen target, and do not claim completion.
Read before coding:
- this task ledger because it is the approved implementation authority.
- reviewed `spec.md` because it is the canonical decision record.
- specification-review result because it records accepted proof obligations.
Read before relevant tasks:
- `design/` before tasks that touch approved package or runtime boundaries.
Do not change: public HTTP semantics other than the approved request-ID echo behavior.
Task-local implementation quality bar:
- before starting each task, bind task ID, `Source`, owner file/package, proof, and stop/reopen condition; repair the ledger or reopen if any of those are missing.
- before checking a task, self-review that the diff still traces to the `Source`, introduces no unapproved decision/dependency/pattern, keeps file responsibility focused, and has proof that covers this task rather than a neighboring surface.
- choose the owning package/file from the approved placement rule before substantial edits and avoid catch-all growth.
- change owning sources before generated output and prove drift when generation applies.
- add no unapproved dependency, custom helper framework, or new pattern-like abstraction.
Resume rule: on resume, read git status and this ledger first, then continue at the first unchecked task whose dependencies are satisfied.
Progress log: update each task's `Evidence` line after running its proof; if blocked, stop and record `Blocked:` under the task.

## Task-Review Handoff

Consumes: reviewed `spec.md`, specification-review result, `design/`, and this task ledger.
Task ledger review: pending_task_review.
Implementation readiness: pending_task_review.
Next phase: task-review/readiness.
Suggested first implementation task after readiness approval: T001.
Accepted concerns: none.
Reopen target: planning if required artifact context is missing.

## Checkpoint Gates

| Checkpoint | Tasks | Gate before continuing |
| --- | --- | --- |
| CP1 request ID behavior | T001-T002 | Targeted HTTP proof is current before final validation. |

## Tasks

- [ ] T001 [Checkpoint 1] Update `internal/http/handler.go` to preserve request ID echo behavior.
  Depends on: none.
  Files: `internal/http/handler.go`.
  Source: `spec.md` request-ID behavior decision.
  Proof: `go test ./internal/http`.
  Evidence:
  - Command/read: Pending.
  - Result: Pending.
  - Key output/ref: Pending.
  - Changed proof files: Pending.
  - Residual blocker/narrower claim: Pending.

- [ ] T002 [Checkpoint 1] [P] Add regression coverage in `internal/http/handler_test.go`.
  Depends on: T001.
  Files: `internal/http/handler_test.go`.
  Source: specification-review request-ID proof obligation.
  Proof: `go test ./internal/http`.
  Evidence:
  - Command/read: Pending.
  - Result: Pending.
  - Key output/ref: Pending.
  - Changed proof files: Pending.
  - Residual blocker/narrower claim: Pending.
```

Prefer vertical, reviewable slices. Avoid generic tasks like `implement feature`. Keep headers short; if the Goal contract or handoff starts carrying phase strategy or design rationale, trim it back or reopen `design/`. Use multi-line items for readability, not as permission to hide new decisions or broad subplans inside a checkbox.

## Planning Rules
- For direct-path work, a short inline plan may still be enough; do not force `tasks.md` for a tiny change just to satisfy ceremony.
- For lean-local and full-orchestrated non-trivial work, default to `tasks.md` and consume reviewed `spec.md` plus required design context.
- Create or repair `test-plan.md` or `rollout.md` during planning only when the approved design already contains the needed validation or rollout context. If the companion artifact would require a missing design, compatibility, migration, or rollout decision, reopen `system-integration-design` or `go-code-ownership-design` according to the missing decision owner instead of filling the gap inside the plan.
- If a required specification review gate is missing or blocking, reopen specification review instead of filling the gap inside the plan.
- If a required technical design review gate is missing or blocking, reopen technical design review instead of filling the gap inside the plan.
- When later review or validation phase-control files are genuinely needed for named multi-session routing, planning should leave them ready to be created or linked before implementation begins; post-code work should not need to invent new workflow/process artifacts.
- The workflow-control handoff must be challenge-ready: master and phase-local plans should make phase status, blockers, stop rules, next-session start, the next-session context bundle, `tasks.md` status, artifact expectations with trigger rationale, and any named review or validation phase files clear enough for an adequacy challenger to review without reconstructing intent from chat. It must not store the full ready-to-paste next-session prompt; render that prompt only in the final chat response.
- The task-review handoff must be explicit: leave review and readiness fields as `pending_task_review`, name the consumed artifact chain, and identify likely review lenses or proof obligations for the separate gate.
- The task-review/readiness phase records consumed subagent gates, lane-derived proof obligations, accepted risks or waivers, and whether unresolved lane blockers or severity conflicts remain.
- If required compact or split design context is missing or inconsistent, reopen specification, `system-integration-design`, or `go-code-ownership-design` instead of inferring the missing context locally.
- If required specification review is missing or inconsistent, reopen specification review instead of inferring approval locally.
- If required technical design review is missing or inconsistent, reopen technical design review instead of inferring approval locally.
- Keep planning aligned with repository realities: OpenAPI drift checks, `sqlc` regeneration, migrations, race tests, integration checks, or other real verification surfaces when they actually apply.
- Keep dependency decisions aligned with repository realities: prefer existing repo patterns and current stdlib where sufficient, and include module, license/security, transitive dependency, and drift checks when introducing OSS.
- Keep Pattern Fit decisions aligned with repository realities: prefer established patterns only when their forces match the task, translate them into idiomatic Go and explicit package ownership, and include proof that validates the selected pattern's guarantee rather than only its vocabulary.
- Keep generated and mirrored cleanup source-of-truth order explicit: update owning sources first, regenerate or sync derived artifacts, and add drift proof instead of hand-editing mirrors or generated output as primary cleanup.
- If a phase is not independently mergeable or testable, name the coupling explicitly.
- Prefer sequential phases unless change surfaces are truly disjoint.
- Make the handoff explicit: the planning session stops at draft or repaired `tasks.md`, and the first implementation task starts only after task-review/readiness passes unless a recorded waiver says otherwise.
- State what should trigger a reopen back into specification or technical design instead of letting coding discover it silently.

## Definition Of Done
The planning pass is complete when:
- the execution order is explicit
- each meaningful task in `tasks.md` has concrete action, dependency/proof context, and planned verification
- checkpoints exist where the risk actually changes
- blocked work is routed to the owning earlier phase instead of left as an open question in ready work
- the separate task-review/readiness phase can start without creating new workflow/process artifacts or missing `tasks.md` to compensate for incomplete planning output
- any mandatory specification review gate is reconciled before planning handoff
- any mandatory design fan-out gate is reconciled before planning handoff
- any mandatory technical design review gate is reconciled before planning handoff
- `tasks.md` is ready for task-ledger review to compare it with reviewed `spec.md`, specification-review obligations, required design context, design fan-out obligations, technical-design-review obligations, and triggered validation or rollout obligations
- task-ledger review and implementation-readiness status are explicit as `pending_task_review`, unless planning is honestly blocked, reopened, or an eligible waiver is recorded
- the workflow-control artifacts are ready for the read-only adequacy challenge, or the direct-path skip rationale is explicit
- required read-only fan-out was run, validly scoped down, or explicitly blocked; Missing explicit subagent authorization is not a valid `Ledger-review fan-out rationale:`
- the next session can start task-review/readiness without re-planning or guessing where this planning pass was supposed to stop
- out-of-scope implications are visible as non-goals, accepted risks, or proof-only follow-ups rather than hidden target-state cleanup
- the task ledger is specific enough for `go-coder` to execute without recreating strategy or reverse-engineering missing design context
- owner package/file, placement evidence, cleanup owner, and test owner are concrete enough that `go-coder` does not need to make a design choice
- selected pattern constraints and proof obligations are clear enough that `go-coder` does not need to reinterpret or choose a design/system pattern
- no unresolved decision gate, `TBD`, or implementation-blocking open question remains in `tasks.md`

## Escalate Or Reject
- task breakdown derived from an unstable spec
- task breakdown that assumes missing compact or split design context instead of escalating
- a phase list with no acceptance criteria or verification
- a `tasks.md` ledger with open questions, unresolved gates, or implementation-time design decisions
- a `tasks.md` ledger that hides owner/file choice behind vague wording such as `choose appropriate file`, `place where it fits`, `split if necessary`, or `implementation decides`
- a generic task like `implement the feature`
- horizontal slicing that hides risk and postpones integration until the end
- a `tasks.md` ledger that turns into a strategy memo instead of listing executable, proof-bound work
- planning output that leaves workflow-control routing too vague for adequacy review before handoff
- planning output that duplicates the entire spec instead of turning it into execution work
