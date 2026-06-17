# Planning Phase

Detailed phase companion for `docs/spec-first-workflow.md`. Read this when drafting or repairing `tasks.md` from reviewed specification and required design context.

## Read When

- Specification review has passed and the next phase is compact tasking or planning.
- A draft `tasks.md` must be created or repaired before read-only task review.
- Goal-ready completion semantics, proof obligations, checkpoint gates, source traceability, generated or mirrored output discipline, or legacy cleanup tasking must be written into the ledger.

## Inputs

- Reviewed `spec.md`, specification-review result, compact design or `design/` bundle, technical-design-review result when present, and approved or explicitly not-expected test design.
- Triggered `test-plan.md`, `rollout.md`, review phase, or validation phase obligations.
- Subagent gate, design fan-out, accepted risks, proof obligations, and source-of-truth evidence needed to make implementation tasking executable.

## Outputs

- Draft or repaired `tasks.md` with Goal Contract, checkpoint gates, traceable task sources, test-design scenario references when present, evidence fields, cleanup audit, proof obligations, and resume rule.
- A planning handoff that identifies the completed draft, consumed artifacts, unresolved blockers if any, and the next phase: task review/readiness.

## Stop Rule

Stop when `tasks.md` is review-ready or blocked. During planning, `Task ledger review`, `Implementation readiness`, `Ledger-review fan-out`, and `Ledger-review fan-out rationale` remain `pending_task_review`; do not approve implementation readiness, run task-ledger review as author self-certification, render an implementation Goal prompt, or start implementation in this phase.

## Planning Responsibility

Planning turns reviewed decisions, required design context, and approved test-design scenarios into executable `tasks.md`.

Direct path may use an inline plan.

Lean local and full orchestrated work use `tasks.md` for non-trivial implementation.

Planning must not invent missing design context. If exact tasking requires a missing decision, reopen the earlier concern.

Planning invention test: if `tasks.md` would need to decide source of truth, contract shape, runtime sequence, failure policy, rollout mechanism, owner package/file, cleanup owner, proof owner, test scenario class, proof level, fail-before signal, or test owner before it can name task source, order, proof, evidence, checkpoint, or stop/reopen condition, planning is blocked. Do not convert that gap into an implementation task; reopen specification, system/integration design, Go code ownership design, technical design review, or test design according to the missing owner.

`tasks.md` is a draft until the task-ledger review/readiness phase checks it against the approved artifact chain. A planning session may make the draft easy to review, but it must not treat its own authoring pass as the read-only readiness gate. `task-review-readiness.md` owns the readiness verdict.

Planning consumes the specification review result for all non-trivial work. Missing review, blocking review, or repaired spec after `FAIL` without a follow-up verdict is a planning-entry failure, not a detail to infer inside `tasks.md`. When the review result is `CONCERNS`, planning must copy the accepted spec risks and proof obligations into the task-review handoff and the relevant ledger or companion artifacts.

Planning also consumes the technical design review result whenever separate design depth was triggered. Missing system/integration design, missing Go code ownership design, missing review, blocking review, or repaired design after `FAIL` without a follow-up verdict is a planning-entry failure, not a detail to infer inside `tasks.md`. When the review result is `CONCERNS`, planning must copy the accepted design risks and proof obligations into the task-review handoff and the relevant ledger or companion artifacts.

Planning consumes test design whenever `test-plan.md` is triggered. Missing test design, missing or stale `test-plan.md`, blocked test-design fan-out, or scenario rows without source anchors, proof levels, pass/fail observables, fail-before expectations or waivers, and reopen targets are planning-entry failures, not details to infer inside `tasks.md`. When test design is not triggered, planning must point to the explicit `not expected` rationale before placing proof directly in tasks.

Planning preflight is complete only when the author can name the specification-review status, system/integration design status or not-expected rationale, contract-design checkpoint result when contract surfaces are plausible, Go code ownership design status or not-expected rationale, technical-design-review status or not-expected rationale, test-design status or not-expected rationale, consumed subagent/design/test-design fan-out gates, accepted concerns, proof-obligation map, source-anchor set, source responsibility audit when code ownership was triggered, package/file placement evidence, checkpoint triggers, generated or mirrored source plan, legacy cleanup audit input, and reopen target for every missing prerequisite. If a prerequisite is absent or stale, stop with `tasks.md` blocked or leave it uncreated and route to the owning earlier phase.

## Planning Entry Pack

Before writing or repairing `tasks.md`, assemble the implementation inputs as a compact planning entry pack. This can live in the planning notes, workflow-control phase file when one exists, or directly in `tasks.md` as traceability tables. It is not a new decision artifact.

The pack should identify:

- approved behavior, non-goals, constraints, and accepted risks that implementation must preserve;
- specification-review and technical-design-review verdicts, including finding IDs, accepted proof obligations, carrying artifacts, and reopen targets;
- exact source anchors for every material behavior, owner, interface, data, rollout, validation, generated-output, mirrored-output, or cleanup obligation;
- contract-design anchors for every changed REST/API, event, generated, or material internal interface surface, including runtime source of truth, generated or derived outputs, compatibility class, proof carrier, and reopen target;
- test-design scenario IDs, selected proof levels, fail-before expectations or waivers, and pass/fail observables that must become proof-first tasks;
- authoritative owner package, exact file when known, or narrow package/artifact surface plus the approved placement rule that will decide the file before coding;
- placement evidence from the approved source responsibility audit, current file responsibilities, sibling files, rejected owner locations, package ownership, generated/manual authority, or design ownership maps for substantial code tasks;
- generated and mirrored surfaces, their authoritative source, generator or sync command, expected output paths, and drift proof;
- known legacy or replacement surfaces, including code, tests, fixtures, generated artifacts, configs, scripts, examples, docs, skills, agents, and mirrors;
- required proof shape for each material claim, including freshness and negative proof when relevant;
- dependency order, checkpoint gates, and the smallest reopen target for each gate failure.

If any row cannot be traced to an approved source, review finding, system/integration decision, Go code ownership decision, test-design scenario, triggered companion artifact, or accepted proof obligation, do not hide the gap in task wording. Reopen specification, specification review, system/integration design, Go code ownership design, technical design, technical design review, or test design according to the missing owner.

## Authoring Procedure

Draft `tasks.md` in this order:

1. Write the `Goal Contract` from approved scope only: one durable objective, one successful completion condition, one blocked-stop condition, the start read set, task-specific read set, and resume rule.
2. Add an implementation handoff that keeps `Task ledger review`, `Implementation readiness`, `Ledger-review fan-out`, and `Ledger-review fan-out rationale` at `pending_task_review`.
3. Map every accepted concern, proof obligation, and approved test-design scenario to one task, checkpoint, `rollout.md`, or explicit non-task rationale. Include claim, evidence, freshness or negative proof, carrying artifact, and reopen target.
4. Record checkpoint triggers before slicing: material risk boundaries such as public/API compatibility, contract source-of-truth updates, data/migration/cache changes, auth/security, concurrency/lifecycle, rollout, generated drift, mirrored sync, accepted-risk proof, and legacy cleanup need a checkpoint gate or a short rationale for why no material boundary exists.
5. Slice tasks by reviewable diff story in dependency order. Prefer proof-first tasks for behavior changes, implementation tasks that own one surface, generated or mirrored sync tasks after authoritative-source changes, cleanup tasks for retired surfaces, and a final validation task that catches integration or drift.
6. Bind each task, checkpoint, cleanup row, generated-source row, mirrored-source row, test-design scenario, and proof-obligation row to exact `Source:` anchors. Use stable section headings, decision IDs, review finding IDs, design seam names, `test-plan.md` scenario IDs, conditional artifact rows, or proof-obligation IDs. A generic path such as `spec.md` is not enough for a material item.
7. Bind each task to `Files:`, `Owner package/file:`, and `Placement evidence:`. When exact files are known, name them. When the final file choice depends on local inspection during coding, name the owning package or artifact surface plus the allowed placement rule from approved Go code ownership design, such as existing focused owner file, new focused same-package seam file, or approved package boundary, and include a first implementation task that performs that placement inspection inside approved boundaries. Unknown owner package, package boundary, generated/manual authority, cleanup owner, or test owner reopens Go code ownership design.
8. Add proof and evidence slots before implementation starts. Each material task needs enough evidence fields that a later closeout can prove the checkbox without chat history. Proof commands should come from repository-owned command docs or an explicit narrower-proof rationale that names what final validation still covers.
9. Add checkpoint gates wherever later tasks rely on earlier behavior, generated drift, migration order, cleanup, rollout, or proof freshness. Each checkpoint names the blocking proof and reopen target.
10. Add a legacy cleanup audit when replacement or retirement is in scope. Every known old surface must be removed, refactored, retained with owner/reason/proof/exit, or explicitly `not_applicable` with bounded search/read evidence.
11. Run the planning self-check below, then stop at review-ready draft status or blocked status. Planning may state why the draft is ready for task review, but it must not mark implementation readiness as approved or render an implementation Goal prompt.

Planning self-check before `draft_review_ready`: no material task, checkpoint, cleanup row, generated or mirrored source row, or proof-obligation row may be review-ready unless it has an exact `Source`, owner package/file or approved placement rule, proof, evidence slot, dependency/checkpoint relationship when relevant, and stop/reopen condition.

Do not write task wording that says or implies implementation will decide owner, architecture, contract shape, sequence, generated authority, rollout, validation policy, or an accepted-risk proof path. Red-flag phrases include `implementation decides`, `choose appropriate file`, `place where it fits`, `define the API while implementing`, `fill in OpenAPI as needed`, `refactor as needed`, `split if necessary`, and `cleanup later` when they carry owner, source, proof, order, cleanup, or completion semantics. If a ledger would need that phrase to be honest, planning is blocked and must reopen the owning phase.

## Lean `tasks.md`

Lean `tasks.md` is the main execution surface.

Recommended shape:

```markdown
# Tasks

Ledger status: draft_review_ready | blocked

## Goal Contract

Goal objective: Complete <feature/change> by executing this ledger from `T001` through final validation.
Goal scope: all required tasks, checkpoint gates, accepted proof obligations, and ledger-owned closeout surfaces named in this file.
Completion condition: all required tasks are checked, required proof passes, task evidence is current, and ledger-owned closeout updates are complete.
Completion evidence: <final command/read set and closeout artifacts that prove the completion condition>.
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
Resume rule: on resume, read git status and this ledger first, then the artifacts named in `Read before coding`, then the task-specific artifacts named in `Read before relevant tasks` before the first unchecked task whose dependencies are satisfied; re-run only the proof needed to detect drift unless the ledger or failing evidence requires broader validation.

## Planning Traceability

Accepted concern/proof obligation mapping:
| Stable ID | Source finding/obligation | Claim or risk | Carrying artifact/row | Task/checkpoint IDs | Required evidence | Freshness/negative proof | Reopen target |
| --- | --- | --- | --- | --- | --- | --- | --- |
| PO-001 | <spec-review or design-review finding ID> | <claim/risk> | <tasks.md/test-plan.md/rollout.md/non-task rationale row> | <task/checkpoint IDs or not_applicable> | <command/read/manual proof> | <freshness or negative proof rule> | <phase/artifact> |

Generated and mirrored source plan:
| Source | Authoritative source | Generated/mirrored outputs | Generator/sync command | Drift proof | Direct-edit rationale |
| --- | --- | --- | --- | --- | --- |
| <exact decision/source anchor> | <source path/artifact> | <expected paths or not_applicable> | <command or not_applicable> | <command/read> | <none unless approved> |

## Implementation Handoff

Ledger status: draft_review_ready | blocked
Task ledger review: pending_task_review
Implementation readiness: pending_task_review
Consumes: reviewed `spec.md`, specification-review result, compact design or `design/`, technical-design-review result when present
Technical-design-review consumed: <not expected with rationale | PASS | CONCERNS; obligations mapped to tasks.md/test-plan.md/rollout.md with finding IDs>
Test-design consumed: <not expected with rationale | approved `test-plan.md` with scenario IDs | blocked with reopen target>
Design fan-out status: <complete | scoped_down | local_only | blocked | not expected with rationale>
Subagent gates consumed: <gate status and artifact/evidence pointer, or not expected with rationale>
Ledger-review fan-out: pending_task_review
Ledger-review fan-out rationale: pending_task_review
Proof: <smallest sufficient proof command or manual proof>
Reopen target: <none | planning | specification | specification-review | system-integration-design | go-code-ownership-design | technical-design-review>

## Checkpoint Gates

| Checkpoint | Tasks | Gate before continuing |
| --- | --- | --- |
| CP1 <name> | T001-T00N | <specific proof/currentness required before later tasks rely on this surface; reopen target if the gate fails> |

Checkpoint trigger rationale: <material risk boundary that requires each checkpoint, or one sentence naming why no material risk boundary needs a checkpoint>.

## Tasks

Legacy cleanup audit:
| Source | Surface | Status | Evidence | Retention owner/reason/exit |
| --- | --- | --- | --- | --- |
| <exact spec/design/review cleanup anchor> | <old identifier/path/config/doc> | removed/refactored/retained/not_applicable | <command/read> | <only if retained> |

- [ ] T001 Add failing proof for <behavior>
  Files: `internal/...`
  Owner package/file: `<package or exact file>` because <approved source/design owner>.
  Placement evidence: <required for substantial hand-written code; optional for tiny docs/config/mechanical tasks; use source responsibility audit, current file responsibility, sibling/package read, rejected owner locations, generated/manual authority, or first-task inspection rule>.
  Source: `spec.md` <exact decision/section>, <review finding ID, design artifact section, or `test-plan.md` scenario ID when relevant>.
  Test scenario: <TD-001 or not_applicable with no-test-design rationale>.
  Depends on: <none | task/checkpoint IDs>.
  Proof: targeted proof fails for the expected reason before implementation.
  Expected fail-before signal: <error/assertion/output proving the approved behavior is absent before implementation>.
  Proof command source: <docs/build-test-and-development-commands.md anchor or narrower-proof rationale plus final validation coverage>.
  Stop/reopen condition: <what failed/missing proof reopens planning/specification/design/review>.
  Evidence:
  - Command/read: Pending.
  - Result: Pending.
  - Key output/ref: Pending.
  - Changed proof files: Pending.
  - Residual blocker/narrower claim: Pending.

- [ ] T002 Implement scoped production behavior
  Files: `internal/...`
  Owner package/file: `<package or exact file>` because <approved source/design owner>.
  Placement evidence: <required for substantial hand-written code; optional for tiny docs/config/mechanical tasks; use source responsibility audit, current file responsibility, sibling/package read, rejected owner locations, generated/manual authority, or first-task inspection rule>.
  Source: `design/overview.md` <exact decision/section>, <review finding ID or `test-plan.md` scenario ID when relevant>.
  Test scenario: <TD-001 or not_applicable with rationale>.
  Depends on: <task/checkpoint IDs>.
  Proof: targeted proof passes.
  Proof command source: <docs/build-test-and-development-commands.md anchor or narrower-proof rationale plus final validation coverage>.
  Stop/reopen condition: <what failed/missing proof reopens planning/specification/design/review>.
  Evidence:
  - Command/read: Pending.
  - Result: Pending.
  - Key output/ref: Pending.
  - Changed proof files: Pending.
  - Residual blocker/narrower claim: Pending.

- [ ] T003 Run validation and record outcome
  Source: implementation handoff proof, accepted review obligations, approved test-design scenarios when present, and triggered validation artifacts with exact finding/artifact IDs.
  Depends on: all required implementation tasks and checkpoint gates.
  Proof: `go test ./...`, `rtk make check`, or the smallest relevant command.
  Proof command source: <docs/build-test-and-development-commands.md anchor or narrower-proof rationale plus final validation coverage>.
  Stop/reopen condition: <failed final proof reopens planning/specification/design/review or implementation fix inside approved ledger>.
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
- A reviewable task names the intended surface, owner package/file, exact `Source:`, dependency or checkpoint, proof, evidence fields, and reopen/stop condition. Omnibus implementation tasks are not reviewable unless the approved design records why the diff must stay coupled.
- Name one objective and one completion condition so the ledger can drive a long-running Codex Goal without extra chat context.
- The Goal scope must cover every executable task and checkpoint through final validation; do not scope a Goal-ready ledger to the first slice, first checkpoint, or "start implementation" unless the approved ledger is intentionally partial.
- Keep completion and blocked-stop semantics separate. Passing proof is required for successful completion; recording a blocker is a valid stop that leaves the Goal blocked and the affected task unchecked.
- Treat non-trivial `tasks.md` as Goal-ready by default in this repository. That means the ledger should contain the Goal Contract fields a later handoff needs; it does not mean a Goal prompt is rendered before the ledger passes review/readiness.
- Keep the Goal contract derivative: it may summarize approved scope, constraints, proof, and stop rules, but must not introduce new decisions or weaken implementation readiness.
- Write the objective and completion condition so a later implementation handoff can explicitly ask the next session to set a Codex Goal covering all executable ledger tasks.
- Point implementation at the files, docs, plans, or logs it must read first. Split read context into `Read before coding` for the minimum start set and `Read before relevant tasks` for task-specific design, contract, test-plan, rollout, or review artifacts.
- Include a task-local implementation quality bar when the work is broad enough that clean code, package ownership, generated-source discipline, dependency discipline, concurrency/lifecycle behavior, or testing layer choices could otherwise be implicit.
- Include checkpoint/progress rules when the ledger spans multiple tasks, sessions, proof loops, or material risk boundaries.
- Add a compact `Checkpoint Gates` table when checkpoints exist; each gate must state what proof/currentness is required before later tasks can safely rely on that checkpoint. Material public/API, data/migration/cache, auth/security, concurrency/lifecycle, rollout, generated drift, mirrored sync, accepted-risk proof, or cleanup boundaries need a checkpoint or an explicit no-checkpoint rationale.
- Later tasks must not depend on a checkpoint until its gate evidence is current. If the gate fails, the ledger must name whether to reopen planning, test design, specification, specification review, system/integration design, Go code ownership design, technical design, or technical design review.
- Name dependencies when task order matters.
- Include `Subagent gates consumed` and leave task-ledger review fields as `pending_task_review` until the separate task-review/readiness phase runs.
- Name exact files when known, or narrow artifact/package surfaces when exact file choice is not knowable yet. For substantial hand-written code tasks, include placement evidence from current file responsibility, sibling/package reads, generated/manual authority, or an approved first-task placement-inspection rule; this is optional for tiny docs, config, or mechanical tasks.
- Include exact `Source:` anchors for every material task, checkpoint, cleanup row, generated-source row, mirrored-source row, and proof-obligation row; task-ledger review should be able to trace each material item back to approved artifacts.
- A generic artifact path is not a sufficient `Source:` anchor for a material item. Point to the exact spec decision, review finding, design seam, conditional artifact row, or proof obligation that the item preserves.
- Accepted concerns and proof obligations must have stable IDs such as `PO-001`; do not rely on prose-only summaries that cannot be referenced by tasks, checkpoints, review findings, implementation evidence, or closeout.
- Include proof expectations and a structured evidence slot per task or checkpoint. Prefer `Command/read`, `Result`, `Key output/ref`, `Changed proof files`, and `Residual blocker/narrower claim` unless a smaller evidence line is enough for tiny/direct-path work.
- Proof commands should cite `docs/build-test-and-development-commands.md` or another repo-owned command source when one exists. If a narrower task-local proof is sufficient, record why and name what final validation still covers.
- Do not make final validation the first meaningful proof for earlier behavior tasks when a narrower proof can be written; use final validation to catch integration or drift after task-local proof exists.
- Every proof obligation carried into `tasks.md` must name claim, evidence, freshness, negative proof, and carrying artifact when those fields are relevant. If any field is not relevant, record why in the task, checkpoint, or handoff instead of leaving the proof vague.
- Do not check a task or call the ledger complete when required proof is skipped, unavailable, failing, stale, or narrower than the task claim. Leave the task unchecked and record `Blocked:` or a narrower claim in the evidence field.
- Include a `Resume rule` for long-running or resumable ledgers so a context-blind Goal continuation can restart from artifacts, not chat memory.
- For replacement work, include executable cleanup audit/removal tasks for every known in-scope old surface: code, tests, fixtures, generated artifacts, configs, docs, scripts, examples, skills, agents, and mirrors. A retained legacy surface must carry owner, reason, proof, and exit condition instead of becoming an implicit follow-up.
- For replacement work, include a `Legacy cleanup audit` table. Each known old surface must have one status: `removed`, `refactored`, `retained`, or `not_applicable`; retained rows must include owner, reason, proof, and exit condition.
- `not_applicable` is allowed only with bounded evidence checked, such as `No known replacement surface` from the reviewed spec plus targeted reads or searches for plausible old identifiers.
- Do not include unresolved open questions, `TBD` decisions, or pending decision gates in `tasks.md`. A ready-for-review ledger may carry accepted risks and proof obligations, but any implementation-blocking question must reopen specification, system/integration design, Go code ownership design, technical design, technical design review, or test design.
- Do not use `implementation decides`, `choose appropriate file`, `place where it fits`, `refactor as needed`, `split if necessary`, `cleanup later`, or equivalent wording for owner, architecture, sequence, generated authority, rollout, validation policy, cleanup/test ownership, or accepted-risk proof paths. Those are missing upstream decisions, not task-ledger flexibility.
- Treat a newly written `tasks.md` as a draft until the task-ledger review has compared it against reviewed `spec.md`, specification-review obligations, required system/integration and Go code ownership design context, technical-design-review obligations, and triggered validation or rollout obligations.
- For behavior changes and bug fixes, proof-first or test-first is the default.
- For proof-first tasks, name the expected fail-before signal unless a `Proof-first waiver:` is recorded.
- When `test-plan.md` exists, proof-first and test tasks must reference stable `TD-*` scenario IDs. Adding unreferenced scenario classes in `tasks.md` reopens test design unless the task records a narrow assertion-only rationale tied to an approved scenario.
- For docs, config, or mechanical changes where a failing test is not useful, record `Proof-first waiver: <reason>` on the relevant task or checkpoint, not only in chat.
- Generated or mirrored-source tasks must name the authoritative source, generator or sync command, expected generated paths, and drift proof. Direct edits to generated output require an explicit approved rationale.
- The planning handoff must name `task-review/readiness` as the next phase when the ledger is review-ready. If task-ledger review may need fan-out, the chat-only next-session prompt must include the exact `Subagent authorization:` line from `shared/subagents-and-handoff.md`; do not write the full prompt body into `tasks.md`.
