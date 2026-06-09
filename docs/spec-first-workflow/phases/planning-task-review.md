# Planning And Task Review Phase

Detailed phase companion for `docs/spec-first-workflow.md`. Read this when drafting or repairing `tasks.md`, running task-ledger review, or approving implementation readiness.

## Read When

- Specification review has passed and the next phase is compact tasking or planning.
- A draft `tasks.md` must be created, repaired, or reviewed for implementation readiness.
- Task-ledger review fan-out, Goal-ready completion semantics, proof obligations, or legacy cleanup tasking must be checked.

## Inputs

- Reviewed `spec.md`, specification-review result, compact design or `design/` bundle, and technical-design-review result when present.
- Triggered `test-plan.md`, `rollout.md`, review phase, or validation phase obligations.
- Subagent gate, design fan-out, and task-ledger review evidence or valid scoped-down/local-only rationale.

## Outputs

- Draft or approved `tasks.md` with Goal Contract, checkpoint gates, traceable task sources, evidence fields, cleanup audit, proof obligations, and resume rule.
- Task-ledger review/readiness status of `PASS`, `CONCERNS`, `FAIL`, or eligible `WAIVED`.
- Implementation handoff when readiness allows coding, or the smallest owning reopen target.

## Stop Rule

Stop after task-ledger review/readiness is recorded. Do not start implementation unless a later implementation session is explicitly starting from an approved, reviewed ledger.

## Lean `tasks.md`

Lean `tasks.md` is the main execution surface.

Recommended shape:

```markdown
# Tasks

## Goal Contract

Goal objective: Complete <feature/change> by executing this ledger from `T001` through final validation.
Completion condition: all required tasks are checked, required proof passes, task evidence is current, and ledger-owned closeout updates are complete.
Blocked-stop condition: if required proof cannot pass, a required command cannot run, an implementation-blocking decision is missing, or an approved artifact is insufficient, stop with the Goal blocked, leave affected tasks unchecked, record `Blocked:` with evidence and the owning reopen target, and do not claim completion.
Read before coding:
- `tasks.md` because it is the approved implementation ledger.
- `spec.md` because it is the canonical decision record.
- <required design/review artifact> because <one-line reason>.
Read before relevant tasks:
- <artifact path> before <task IDs> because <one-line reason>.
Do not read `workflow-plan.md` for implementation unless no approved ledger exists yet or this ledger explicitly names a pre-created review/validation phase file.
Do not change: <non-goals and preserved constraints from `spec.md`>
Task-local implementation quality bar:
- Before starting each task, bind the task ID to its `Source`, owner file/package, proof, and stop/reopen condition; repair the ledger or reopen if any of those are missing.
- Before checking a task, self-review that the diff still traces to the `Source`, introduces no unapproved decision/dependency/pattern, keeps file responsibility focused, and has proof that covers this task rather than a neighboring surface.
- Before substantial edits, identify the owning package/file and avoid catch-all files or mixed abstraction levels.
- Change owning sources before generated or mirrored outputs; prove drift after regeneration or sync.
- Add no unapproved runtime dependency, custom infrastructure, helper framework, or pattern-like abstraction.
- Preserve context cancellation, bounded concurrency, deterministic errors, safe logs/metrics, and repository observability conventions where the touched surface uses them.
- Keep tests at the smallest layer that proves the approved behavior, including negative proof for retired or forbidden surfaces when text search is reliable.
Progress log: after each checkpoint, update the checkbox/evidence lines; if blocked, stop and record `Blocked:` with the missing input or failing proof.
Resume rule: on resume, read git status and this ledger first, then continue at the first unchecked task whose dependencies are satisfied; re-run only the proof needed to detect drift unless the ledger or failing evidence requires broader validation.

## Implementation Handoff

Task ledger review: PASS | CONCERNS | FAIL | WAIVED
Implementation readiness: PASS | CONCERNS | FAIL | WAIVED
Consumes: reviewed `spec.md`, specification-review result, compact design or `design/`, technical-design-review result when present
Design fan-out status: <complete | scoped_down | local_only | blocked | not expected with rationale>
Subagent gates consumed: <gate status and artifact/evidence pointer, or not expected with rationale>
Ledger-review fan-out: <complete | scoped_down | local_only | not_expected | blocked>
Ledger-review fan-out rationale: <required when local_only, scoped_down, or not_expected>
Proof: <smallest sufficient proof command or manual proof>
Reopen target: <none | planning | specification | specification-review | technical-design | technical-design-review>

## Checkpoint Gates

| Checkpoint | Tasks | Gate before continuing |
| --- | --- | --- |
| CP1 <name> | T001-T00N | <specific proof/currentness required before later tasks rely on this surface> |

## Tasks

Legacy cleanup audit:
| Surface | Status | Evidence | Retention owner/reason/exit |
| --- | --- | --- | --- |
| <old identifier/path/config/doc> | removed/refactored/retained/not_applicable | <command/read> | <only if retained> |

- [ ] T001 Add failing proof for <behavior>
  Files: `internal/...`
  Source: `spec.md` <decision/section>, <review finding or design artifact when relevant>.
  Proof: targeted proof fails for the expected reason before implementation.
  Evidence:
  - Command/read: Pending.
  - Result: Pending.
  - Key output/ref: Pending.
  - Changed proof files: Pending.
  - Residual blocker/narrower claim: Pending.

- [ ] T002 Implement scoped production behavior
  Files: `internal/...`
  Source: `design/overview.md` <decision/section>, <review finding when relevant>.
  Proof: targeted proof passes.
  Evidence:
  - Command/read: Pending.
  - Result: Pending.
  - Key output/ref: Pending.
  - Changed proof files: Pending.
  - Residual blocker/narrower claim: Pending.

- [ ] T003 Run validation and record outcome
  Proof: `go test ./...`, `rtk make check`, or the smallest relevant command.
  Evidence:
  - Command/read: Pending.
  - Result: Pending.
  - Key output/ref: Pending.
  - Changed proof files: Pending.
  - Residual blocker/narrower claim: Pending.
```

Rules:

- Use markdown checkboxes and stable task IDs.
- Keep one reviewable diff story per task. If the title needs "and" to be accurate, split it unless the approved design makes the coupling inseparable.
- Name one objective and one completion condition so the ledger can drive a long-running `/goal` without extra chat context.
- Keep completion and blocked-stop semantics separate. Passing proof is required for successful completion; recording a blocker is a valid stop that leaves the Goal blocked and the affected task unchecked.
- Treat non-trivial `tasks.md` as Goal-ready by default in this repository. That means the ledger should contain the Goal Contract fields a later handoff needs; it does not mean a Goal prompt is rendered before the ledger passes review/readiness.
- Keep the Goal contract derivative: it may summarize approved scope, constraints, proof, and stop rules, but must not introduce new decisions or weaken implementation readiness.
- Write the objective and completion condition so a later implementation handoff can explicitly ask the next session to set a Codex Goal covering all executable ledger tasks.
- Point implementation at the files, docs, plans, or logs it must read first. Split read context into `Read before coding` for the minimum start set and `Read before relevant tasks` for task-specific design, contract, test-plan, rollout, or review artifacts.
- Include a task-local implementation quality bar when the work is broad enough that clean code, package ownership, generated-source discipline, dependency discipline, concurrency/lifecycle behavior, or testing layer choices could otherwise be implicit.
- Include checkpoint/progress rules when the ledger spans multiple tasks, sessions, or proof loops.
- Add a compact `Checkpoint Gates` table when checkpoints exist; each gate must state what proof/currentness is required before later tasks can safely rely on that checkpoint.
- Name dependencies when task order matters.
- Include `Subagent gates consumed` and ledger-review fan-out status for non-trivial ledgers; missing gate state keeps `tasks.md` draft.
- Name exact files when known, or narrow artifact/package surfaces when exact file choice is not knowable yet.
- Include `Source:` anchors for tasks whose requirements come from non-obvious spec decisions, review findings, design sections, `test-plan.md`, or `rollout.md`; task-ledger review should be able to trace every material task back to approved artifacts.
- Include proof expectations and a structured evidence slot per task or checkpoint. Prefer `Command/read`, `Result`, `Key output/ref`, `Changed proof files`, and `Residual blocker/narrower claim` unless a smaller evidence line is enough for tiny/direct-path work.
- Do not check a task or call the ledger complete when required proof is skipped, unavailable, failing, stale, or narrower than the task claim. Leave the task unchecked and record `Blocked:` or a narrower claim in the evidence field.
- Include a `Resume rule` for long-running or resumable ledgers so a context-blind Goal continuation can restart from artifacts, not chat memory.
- For replacement work, include executable cleanup audit/removal tasks for every known in-scope old surface: code, tests, fixtures, generated artifacts, configs, docs, scripts, examples, skills, agents, and mirrors. A retained legacy surface must carry owner, reason, proof, and exit condition instead of becoming an implicit follow-up.
- For replacement work, include a `Legacy cleanup audit` table. Each known old surface must have one status: `removed`, `refactored`, `retained`, or `not_applicable`; retained rows must include owner, reason, proof, and exit condition.
- Do not include unresolved open questions, `TBD` decisions, or pending decision gates in `tasks.md`. A ready ledger may carry accepted risks and proof obligations, but any implementation-blocking question must reopen specification, technical design, or technical design review.
- Treat a newly written `tasks.md` as a draft until the task-ledger review has compared it against reviewed `spec.md`, specification-review obligations, required design context, technical-design-review obligations, and triggered validation or rollout obligations.
- For behavior changes and bug fixes, proof-first or test-first is the default.
- For docs, config, or mechanical changes where a failing test is not useful, record `Proof-first waiver: <reason>` on the relevant task or checkpoint, not only in chat.

## Planning, Task Review, And Implementation Readiness

Planning turns reviewed decisions and required design context into `tasks.md`.

Direct path may use an inline plan.

Lean local and full orchestrated work use `tasks.md` for non-trivial implementation.

Planning must not invent missing design context. If exact tasking requires a missing decision, reopen the earlier concern.

`tasks.md` is a draft until the task-ledger review/readiness gate checks it against the approved artifact chain. This gate must run after the ledger is written or materially repaired and before implementation starts.

Task-ledger review must verify:

- specification review is `PASS` or `CONCERNS` with named accepted risks and proof obligations; a missing, `FAIL`, stale-after-repair, or unresolved specification-review gate blocks handoff;
- every in-scope behavior, non-goal, constraint, and accepted decision from reviewed `spec.md` is represented in executable tasking, preserved constraints, or explicit non-task rationale;
- every accepted specification-review `CONCERNS` proof obligation is represented in executable tasking, design constraints, `test-plan.md`, `rollout.md`, or explicit non-task rationale;
- every approved dependency/OSS due-diligence decision is represented in executable dependency, integration, license/security, generation, or proof tasks where relevant; if due diligence is missing for custom infrastructure or a new dependency, reopen specification or technical design instead of letting implementation decide;
- every approved Pattern Fit decision is represented in executable tasking, design-preserving constraints, validation, or explicit non-task rationale; if pattern comparison is missing for an invented design shape, reopen research, specification, or technical design instead of asking implementation to choose a pattern;
- when separate technical design depth was triggered, design fan-out is `complete`, valid `scoped_down`, or eligible `local_only`; a missing, `blocked`, or ineligible `local_only` authoring gate reopens technical design before planning can approve `tasks.md`;
- the Goal Contract names one objective, one successful completion condition, and a separate blocked-stop condition; a ledger that treats "recorded blocker" as successful completion reopens planning;
- each task has one reviewable diff story, or the ledger records why coupled changes cannot be split safely;
- read context is selective enough to start work without flooding implementation: required start artifacts are listed under `Read before coding`, while task-specific design, contract, test-plan, rollout, or review artifacts are listed under `Read before relevant tasks`;
- checkpointed ledgers include a compact gate table or equivalent wording that states what proof/currentness must hold before later tasks rely on that checkpoint;
- material tasks include `Source:` anchors or equivalent traceability to approved spec decisions, review findings, design sections, `test-plan.md`, or `rollout.md`;
- behavior-change and bug-fix tasks include proof-first/test-first tasking, or an explicit task-level `Proof-first waiver:` with rationale;
- evidence fields are concrete enough for closeout: command/read, result, key output or evidence ref, changed proof files when relevant, and residual blocker or narrower-claim state;
- skipped, unavailable, stale, failing, or too-narrow proof cannot satisfy a task checkbox, checkpoint, or completion claim;
- long-running ledgers include a resume rule that tells a context-blind implementation session where to continue and how much proof to rerun;
- the task-local implementation quality bar is present or explicitly not needed; broad code work must carry package/file responsibility, generated-source, dependency/custom-helper, lifecycle/observability, and smallest-proof expectations into implementation;
- file and package placement is narrow enough that implementation will not have to choose where a substantial code block belongs; when work touches a large or mixed-responsibility hand-written file, the ledger names the owning file, focused new seam file, package boundary, or approved rationale for keeping the code together;
- known in-scope legacy surfaces are represented as removal/refactor work, retained-surface rationale with owner/reason/proof/exit condition, or explicit not-applicable proof; missing cleanup coverage is a planning blocker, not implementation discretion;
- replacement ledgers include a per-surface cleanup audit table; generic prose is not enough when known old surfaces exist;
- required compact design, `design/overview.md`, or split `design/` ownership, sequence, dependency, failure, and conditional-artifact rules are reflected in task order and proof expectations;
- technical-design-review `CONCERNS` are carried as named accepted risks and proof obligations, and any `FAIL`, unresolved `blocks_planning`, `reopens_design`, or `reopens_spec` finding blocks handoff;
- triggered `test-plan.md`, `rollout.md`, review phase, or validation phase obligations are either represented in the ledger or explicitly marked not expected with rationale before code starts;
- the ledger contains no open-question section, unresolved decision gate, `TBD`, hidden design work, or instruction for implementation to decide architecture, ownership, contract, sequencing, rollout, or validation policy.
- subagent gates consumed by planning are listed, no lane blocker or material severity conflict remains unresolved, and subagent-derived proof obligations are mapped into `tasks.md`, `test-plan.md`, or `rollout.md`.

Before marking `tasks.md` approved for non-trivial work, use a read-only task-ledger review fan-out by default. Typical lanes are coverage and traceability, dependency ordering, proof and QA, and any triggered API, data, security, reliability, delivery, performance, observability, or rollout lens. Each lane reviews the draft ledger against approved artifacts only; no lane edits `tasks.md` or makes final readiness decisions. A local-only or scoped-down ledger review must explicitly evaluate the default lanes and explain why each omitted lane cannot change readiness. Missing explicit subagent authorization is not a valid `Ledger-review fan-out rationale:`. Without recorded task-ledger review fan-out status or `Ledger-review fan-out rationale:`, implementation readiness remains `FAIL` or blocked.

If the review finds a blocker, use the smallest owning reopen target:

- `planning` for missing task coverage, wrong ordering, vague proof, missing evidence fields, or workflow-control handoff gaps that do not change approved decisions or design;
- `specification review` when a required review verdict is missing, stale after repair, or has unresolved blocking findings;
- `technical design review` when a required review verdict is missing, stale after repair, or has unresolved blocking findings;
- `technical design` when the ledger needs ownership, sequence, dependency, rollout, validation, or conditional-artifact context the design does not provide;
- `specification` when the missing or contradictory point changes accepted scope, behavior, invariant, public contract, non-goal, or approval boundary.

Task-ledger review and implementation readiness use the same status vocabulary:

- `PASS`: coding may start; no hidden architecture, ownership, contract, sequencing, rollout, or validation decision is needed for the next slice.
- `CONCERNS`: coding may start only with named accepted risks and explicit proof obligations; these concerns must be closed as decisions, not open questions.
- `FAIL`: coding must not start; route to the named earlier phase.
- `WAIVED`: allowed only for tiny direct-path or explicitly user-requested prototype scope with explicit rationale.

Readiness belongs in the planning handoff when planning artifacts exist. `workflow-plan.md` and `workflow-plans/planning.md` record the gate status when those artifacts are used; `tasks.md` may carry a short reference. Implementation may start only after task-ledger review produces `PASS`, eligible `CONCERNS`, or eligible `WAIVED`.

Planning consumes the specification review result for all non-trivial work. Missing review, blocking review, or repaired spec after `FAIL` without a follow-up verdict is a planning-entry failure and a task-review blocker, not a detail to infer inside `tasks.md`. When the review result is `CONCERNS`, planning must copy the accepted spec risks and proof obligations into the task-ledger review/readiness handoff and the relevant ledger or companion artifacts.

Planning also consumes the technical design review result whenever separate design depth was triggered. Missing review, blocking review, or repaired design after `FAIL` without a follow-up verdict is a planning-entry failure and a task-review blocker, not a detail to infer inside `tasks.md`. When the review result is `CONCERNS`, planning must copy the accepted design risks and proof obligations into the task-ledger review/readiness handoff and the relevant ledger or companion artifacts.
