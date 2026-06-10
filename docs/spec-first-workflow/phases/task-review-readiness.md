# Task Review And Readiness Phase

Detailed phase companion for `docs/spec-first-workflow.md`. Read this when reviewing a completed `tasks.md` draft and deciding whether implementation may start.

## Read When

- A draft or repaired `tasks.md` is ready for read-only task-ledger review.
- Implementation handoff requires task-ledger review and implementation-readiness status.
- Task-ledger review fan-out, accepted concerns, proof-obligation mapping, or reopen routing must be checked.

## Inputs

- Draft `tasks.md` from the planning phase.
- Reviewed `spec.md`, specification-review result, compact design or `design/` bundle, and technical-design-review result when present.
- Triggered `test-plan.md`, `rollout.md`, review phase, validation phase, subagent gate, design fan-out, and proof-obligation evidence.

## Outputs

- Task-ledger review/readiness status of `PASS`, `CONCERNS`, `FAIL`, or eligible `WAIVED`.
- Updated readiness handoff in the active workflow-control surface or short `tasks.md` reference when appropriate.
- Implementation handoff when readiness allows coding, or the smallest owning reopen target.

## Stop Rule

Keep task review read-only with respect to implementation and ledger authorship. Stop after task-ledger review/readiness is recorded. Do not start implementation unless a later implementation session is explicitly starting from an approved, reviewed ledger.

Do not repair `tasks.md` inline as the reviewer. If the ledger needs edits, record `FAIL` with the smallest owning reopen target, usually `planning`; the planning phase owns ledger changes. If review exposes a missing approved decision or stale upstream review, route to specification, specification review, system/integration design, Go code ownership design, technical design, or technical design review instead of filling the gap in task-review wording.

## Task-Ledger Review

Task-ledger review is a distinct post-planning, pre-implementation gate. It checks whether the completed `tasks.md` matches the approved artifact chain and whether implementation can proceed without hidden specification, design, ownership, rollout, or validation decisions.

`tasks.md` is a draft until this gate checks it against the approved artifact chain. This gate must run after the ledger is written or materially repaired and before implementation starts.

This file is the canonical home for task-ledger review and implementation-readiness approval. Planning docs and planning skills may describe how to prepare a ledger for this gate, but they must not assign the final readiness verdict.

The reviewer answers one question: can a context-blind implementation session execute this ledger from the recorded start point through its named proof without making an unapproved decision? The answer is `PASS`, `CONCERNS`, `FAIL`, or eligible `WAIVED`; it is not a rewritten task list.

Use the 90-second implementation start test before approval: could a fresh implementation agent set the Codex Goal, read only the listed artifacts, start the first executable task, and know the completion proof without chat history? If not, the result is `FAIL` to planning unless the missing context belongs to specification, specification review, system/integration design, Go code ownership design, technical design, or technical design review.

## Review Packet Preflight

Before judging individual tasks, identify the review packet and reject stale or incomplete inputs:

- exact `tasks.md` path, draft status, repaired section when this is a follow-up review, and the first unchecked or first executable task if implementation would resume from a checkpoint;
- prior task-review `FAIL` or `CONCERNS` findings when this is a follow-up review, including the repaired ledger row or section, rechecked lens, and new verdict basis;
- reviewed `spec.md` path and specification-review verdict, including accepted risks, proof obligations, finding IDs, and follow-up status after any prior `FAIL`;
- compact design context or `design/` bundle, including triggered system/integration and Go code ownership design decisions, source responsibility audit when code ownership was triggered, plus technical-design-review verdict or not-expected rationale when separate design depth was not triggered;
- consumed subagent gate, `Design fan-out`, `Ledger-review fan-out`, and local-only or scoped-down rationales that the ledger depends on;
- triggered `test-plan.md`, `rollout.md`, review phase, validation phase, generated or mirrored source obligations, and legacy cleanup obligations;
- source-anchor set: approved spec decisions, review findings, design seams, conditional artifact rows, and proof-obligation IDs that must trace into the ledger;
- reopened or blocked findings from earlier phases, if any, and the artifact that proves they were closed.

If the packet cannot be named or contains stale, missing, or unresolved approval inputs, the task-review result is `FAIL` or blocked with the owning reopen target. Do not infer approval from surrounding prose, chat memory, a green command, or a draft ledger status.

## Fan-Out And Review Lenses

Before marking non-trivial `tasks.md` approved, use read-only task-ledger review fan-out by default. Each lane reviews the draft ledger against approved artifacts only; no lane edits `tasks.md`, solves missing decisions, or makes the final readiness decision.

Default lane candidates:

- coverage and traceability: every approved behavior, non-goal, accepted risk, proof obligation, source anchor, and cleanup surface has an executable task, checkpoint, companion artifact row, or explicit non-task rationale;
- ordering and checkpoints: dependencies, risk boundaries, generated or mirrored sync order, rollout or migration order, and checkpoint gates are sufficient before later tasks rely on earlier work;
- proof and QA: task-level proof, fail-before expectations, proof-first waivers, evidence fields, freshness, negative proof, final validation, and stale or too-narrow proof rules are specific enough for closeout;
- ownership and implementation handoff: Goal Contract, read-before-coding set, task-specific read set, owner package/file or placement rule, source responsibility audit, implementation quality bar, resume rule, and blocked-stop rule are usable by a new session;
- triggered specialist lenses: API, data, security, reliability, observability, performance, delivery, rollout, dependency/OSS, Pattern Fit, generated-source, mirrored-source, or legacy-cleanup lanes when those surfaces are part of the approved artifact chain.

A local-only or scoped-down review must explicitly list the default lane candidates considered, the evidence checked for each, why omitted lanes cannot change readiness, and the seam that would reopen fan-out. Missing explicit subagent authorization is not a valid `Ledger-review fan-out rationale:`. Without recorded task-ledger review fan-out status or a valid rationale, implementation readiness remains `FAIL` or blocked.

Each material lane finding should be synthesis-ready:

```text
Finding | Evidence anchor | Impact on implementation readiness | Classification | Owner/reopen target | Required disposition | Why not stronger/weaker
```

For `CONCERNS`, the disposition must name the accepted risk or proof obligation and the exact carrying task, checkpoint, `test-plan.md`, `rollout.md`, or explicit non-task rationale. For `FAIL`, the disposition must name the smallest earlier phase that can change the failing fact.

Task-ledger review must verify:

- specification review is `PASS` or `CONCERNS` with named accepted risks and proof obligations; a missing, `FAIL`, stale-after-repair, or unresolved specification-review gate blocks handoff;
- the ledger status is not treated as approved merely because it says `draft_review_ready`; a material repair after a prior task review requires a fresh or explicitly updated task-review verdict;
- every in-scope behavior, non-goal, constraint, and accepted decision from reviewed `spec.md` is represented in executable tasking, preserved constraints, or explicit non-task rationale;
- every accepted specification-review `CONCERNS` proof obligation is represented in executable tasking, design constraints, `test-plan.md`, `rollout.md`, or explicit non-task rationale;
- every approved dependency/OSS due-diligence decision is represented in executable dependency, integration, license/security, generation, or proof tasks where relevant; if due diligence is missing for custom infrastructure or a new dependency, reopen specification or technical design instead of letting implementation decide;
- every approved Pattern Fit decision is represented in executable tasking, design-preserving constraints, validation, or explicit non-task rationale; if pattern comparison is missing for an invented design shape, reopen research, specification, or technical design instead of asking implementation to choose a pattern;
- when separate technical design depth was triggered, design fan-out is `complete`, valid `scoped_down`, or eligible `local_only` for every triggered design checkpoint; a missing, `blocked`, or ineligible `local_only` authoring gate reopens the owning design checkpoint before `tasks.md` can be approved;
- when separate technical design depth was triggered, the implementation handoff includes `Technical-design-review consumed:` with `PASS` or `CONCERNS`, exact obligation mapping targets, and no unresolved hidden-design-work finding;
- when system/integration design was triggered, each planning-critical mechanism used by the ledger has selected or preserved behavior, source-of-truth owner, affected runtime or failure branch, code-carrying constraint, rejected live alternative closure rule, proof carrier, and reopen trigger; missing fields reopen `system-integration design`, not implementation;
- the Goal Contract names one durable objective, one successful completion condition, one completion-evidence set, and a separate blocked-stop condition; a ledger that treats "recorded blocker" as successful completion reopens planning;
- the Goal Contract covers every executable task, checkpoint, accepted proof obligation, and ledger-owned closeout surface through final validation, not merely "start implementation", "continue work", "finish the next slice", or "attempt the ledger";
- each task has one reviewable diff story, or the ledger records why coupled changes cannot be split safely;
- each material task carries enough fields for implementation to start without inventing ownership: intended surface, owner package/file, exact source anchor, dependency or checkpoint, proof, evidence fields, and reopen/stop condition;
- task coupling rationale is explicit when one task spans multiple surfaces, generated output plus source edits, data plus API changes, or cleanup plus behavior changes; otherwise omnibus tasks reopen planning;
- read context is selective enough to start work without flooding implementation: required start artifacts are listed under `Read before coding`, while task-specific design, contract, test-plan, rollout, or review artifacts are listed under `Read before relevant tasks`;
- checkpointed ledgers include a compact gate table or equivalent wording that states what proof/currentness must hold before later tasks rely on that checkpoint, what task range is blocked by the gate, and which owner reopens if the gate fails;
- later tasks cannot rely on checkpointed work until the gate evidence is current; vague gates such as "verify before continuing", "run tests as needed", or "investigate failures" are `FAIL` unless the ledger names the exact proof and reopen target;
- material tasks include `Source:` anchors or equivalent traceability to approved spec decisions, review findings, design sections, `test-plan.md`, or `rollout.md`;
- material task `Source:` anchors identify the exact decision, finding, seam, conditional artifact row, or proof obligation they preserve; generic artifact-only anchors are not traceable enough for approval;
- every accepted risk or proof obligation has a stable ID that appears in the relevant task, checkpoint, companion artifact row, and handoff; duplicate-only prose or unmapped summaries are not enough;
- behavior-change and bug-fix tasks include proof-first/test-first tasking, or an explicit task-level `Proof-first waiver:` with rationale;
- proof-first tasks name the expected fail-before signal unless the waiver explains why a fail-before proof is not useful for this docs, config, mechanical, or generated-output change;
- evidence fields are concrete enough for closeout: command/read, result, key output or evidence ref, changed proof files when relevant, residual blocker or narrower-claim state, and freshness expectation when stale output could mislead;
- final validation is not the only proof for behavior work when narrower task-level proof is feasible;
- accepted proof obligations map claim, evidence, freshness, negative proof when relevant, and carrying artifact into an executable task, checkpoint, `test-plan.md`, `rollout.md`, or explicit non-task rationale;
- skipped, unavailable, stale, failing, or too-narrow proof cannot satisfy a task checkbox, checkpoint, or completion claim;
- long-running ledgers include a resume rule that tells a context-blind implementation session where to continue and how much proof to rerun;
- the task-local implementation quality bar is present or explicitly not needed; broad code work must carry package/file responsibility, generated-source, dependency/custom-helper, lifecycle/observability, and smallest-proof expectations into implementation;
- file and package placement is narrow enough that implementation will not have to choose where a substantial code block belongs; when work touches a large or mixed-responsibility hand-written file, the ledger names the owning file, focused new seam file, package boundary, or approved rationale for keeping the code together;
- owner files are not optional for docs, config, generated-source, or mechanical work when the touched files are already knowable; for substantial hand-written code, the ledger may name an owner package plus a first-task placement-inspection rule only when the approved artifacts leave the exact file choice to local source inspection without changing design;
- unknown exact file is acceptable only when the ledger names the owner package or artifact surface plus a first-task placement-inspection rule; unknown owner package, package boundary, or generated/manual authority is `FAIL`;
- known in-scope legacy surfaces are represented as removal/refactor work, retained-surface rationale with owner/reason/proof/exit condition, or explicit not-applicable proof; missing cleanup coverage is a planning blocker, not implementation discretion;
- replacement ledgers include a per-surface cleanup audit table; generic prose is not enough when known old surfaces exist;
- `not_applicable` cleanup status is backed by bounded evidence rather than absence of memory;
- required compact design, `design/overview.md`, or split `design/` ownership, sequence, dependency, failure, and conditional-artifact rules are reflected in task order and proof expectations;
- technical-design-review `CONCERNS` are carried as named accepted risks and proof obligations, and any `FAIL`, unresolved `blocks_planning`, `reopens_design`, or `reopens_spec` finding blocks handoff;
- triggered `test-plan.md`, `rollout.md`, review phase, or validation phase obligations are either represented in the ledger or explicitly marked not expected with rationale before code starts;
- the ledger contains no open-question section, unresolved decision gate, `TBD`, hidden design work, or instruction for implementation to decide architecture, ownership, contract, sequencing, rollout, or validation policy;
- subagent gates consumed by planning are listed, no lane blocker or material severity conflict remains unresolved, and subagent-derived proof obligations are mapped into `tasks.md`, `test-plan.md`, or `rollout.md`;
- generated or mirrored outputs are task-owned only through their authoritative source plus generator/sync proof, unless an approved artifact explicitly authorizes direct edits;
- generated-source or mirrored-source drift proof is not hidden inside broad final validation when the ledger changes authoritative sources; the tasking must name source, generator or sync command, expected outputs, direct-edit rationale if any, and drift proof;
- the implementation handoff can be rendered from the ledger without inventing context: it has the Goal Contract, approved readiness status, read order, first executable task or checkpoint, accepted concerns or waiver, proof obligations, progress-update rule, and blocked-stop rule needed for a later Codex Goal prompt.

## Falsification Checks

Use these quick checks to catch weak ledgers before implementation:

| Check | Fails When | Usual Reopen Target |
| --- | --- | --- |
| Traceability | a material task, checkpoint, cleanup row, generated-source row, mirrored-source row, or proof row points only to a generic artifact path, chat summary, or no source. | `planning`, unless the approved source does not exist, then earlier owner |
| Goal Contract | objective or completion condition is vague, partial, slice-limited, not evidence-bound, or conflates blocked stop with successful completion. | `planning` |
| Owner file/package | implementation must choose a substantial owner file, package boundary, generated/manual authority, same-package seam, source responsibility audit, or rejected owner location without an approved placement rule. | `planning` or `go-code-ownership design` when ownership is missing upstream |
| Hidden design work | a task says or implies "implementation decides" for architecture, owner, sequence, rollout, validation, source responsibility audit, source of truth, dependency, Pattern Fit, generated authority, cleanup, or test ownership. | `system-integration design`, `go-code-ownership design`, or `specification` |
| Checkpoint gate | later tasks rely on a checkpoint without exact gate proof, freshness/currentness rule, blocked task range, and reopen target. | `planning` |
| Proof quality | required proof is skipped, unavailable, stale, too narrow for the task claim, final-validation-only when task proof is feasible, or missing negative proof for retired surfaces. | `planning` |
| Review concern mapping | accepted `CONCERNS` from specification review or technical design review are copied as prose but not mapped to carrying artifact, required evidence, freshness, and reopen target. | `planning` |
| Legacy cleanup | replacement work lacks a per-surface status of removed, refactored, retained with owner/reason/proof/exit, or not_applicable with bounded evidence. | `planning` |
| Generated or mirrored drift | authoritative source, generated or mirrored outputs, generator/sync command, direct-edit rationale, or drift proof is missing. | `planning`, `system-integration design`, or `go-code-ownership design` when authority is undecided |
| Handoff | a new implementation session cannot derive the required Codex Goal prompt and execution brief from `tasks.md` without chat history or reviewer invention. | `planning` |

If a failed check requires editing `tasks.md`, return `FAIL` and route to planning repair. Do not solve the row inline in the review record. If a failed check proves the approved spec or design cannot support executable tasking, route to that earlier owner instead.

### Red-Flag Ledger Phrases

Treat vague implementation language as a falsification trigger, not a style nit. Phrases such as `as needed`, `if necessary`, `verify later`, `run relevant tests`, `ensure works`, `cleanup later`, `use appropriate file`, `choose appropriate file`, `place where it fits`, `refactor as needed`, `split if necessary`, `implementation decides`, or `TBD` are `FAIL` when they carry owner, source, proof, order, checkpoint, cleanup, generated or mirrored drift, rollout, or completion semantics. Route to planning repair unless the phrase exposes a missing upstream decision.

### Proof Claim Mapping

Proof must map to claims, not sit as a single broad command. When a ledger carries material proof, task review should be able to read this shape:

```text
Claim | Task/checkpoint | Command/read/manual proof | What output proves | Freshness or negative proof | Reopen target
```

If proof cannot be mapped this way, or if final validation is the first meaningful proof for behavior where task-level proof is feasible, the result is `FAIL` to planning.

If the review finds a blocker, use the smallest owning reopen target:

- `planning` for missing task coverage, wrong ordering, vague proof, missing evidence fields, or workflow-control handoff gaps that do not change approved decisions or design;
- `specification review` when a required review verdict is missing, stale after repair, or has unresolved blocking findings;
- `technical design review` when a required review verdict is missing, stale after repair, or has unresolved blocking findings;
- `system/integration design` when the ledger needs source-of-truth, sequence, dependency, rollout, validation, failure-behavior, or conditional-artifact context the design does not provide;
- `Go code ownership design` when the ledger needs source responsibility audit, owner package/file, rejected owner locations, responsibility boundary, cleanup, local abstraction, dependency direction, or test ownership context the design does not provide;
- `specification` when the missing or contradictory point changes accepted scope, behavior, invariant, public contract, non-goal, or approval boundary.

## Review Result Record

Record the task-review/readiness result in the active workflow-control surface when one exists, or as the approved readiness handoff/reference in `tasks.md` when the lean-local artifact owns the state. The record must name the reviewed ledger and approved artifact chain; it must not be a raw transcript of lane outputs.

Minimum result shape:

```text
Reviewed ledger: <path and status>
Reviewed artifact chain: <spec review status, design status/review status, companion artifacts>
Ledger-review fan-out: <complete | scoped_down | local_only | not_expected | blocked>
Review lens coverage:
| Lens | Trigger/source | Falsification check | Status | Evidence pointer | Finding/disposition | Reopen target |
| --- | --- | --- | --- | --- | --- | --- |
| coverage/traceability | <source anchors and accepted obligations> | <what would prove missing coverage> | <covered | not_applicable | concern | fail> | <artifact section> | <none | finding id and disposition> | <target or none> |
| ordering/checkpoints | <dependencies and checkpoint gates> | <what would make later tasks unsafe> | <covered | not_applicable | concern | fail> | <artifact section> | <none | finding id and disposition> | <target or none> |
| proof/QA | <proof and evidence obligations> | <what would make proof stale, skipped, or too narrow> | <covered | not_applicable | concern | fail> | <artifact section> | <none | finding id and disposition> | <target or none> |
| handoff/ownership | <Goal Contract, read set, owner files, resume rule> | <what would force implementation to invent context> | <covered | not_applicable | concern | fail> | <artifact section> | <none | finding id and disposition> | <target or none> |
| <triggered specialist lens> | <API/data/security/rollout/generated/cleanup/etc. trigger> | <lens-specific falsification check> | <covered | not_applicable | concern | fail> | <artifact section> | <none | finding id and disposition> | <target or none> |
Findings: <none | finding table with owner/reopen target and why not stronger/weaker>
Accepted concerns/proof obligations: <none | stable IDs with carrying artifact/task/checkpoint and evidence>
Proof coverage/currentness: <task/checkpoint/final claims, command/read/manual proof, command-source anchor or narrower-proof rationale, freshness rule, negative proof when relevant, reopen target>
Task ledger review: PASS | CONCERNS | FAIL | WAIVED
Implementation readiness: PASS | CONCERNS | FAIL | WAIVED
Readiness consequence: <implementation may start | implementation may start only with named obligations | implementation blocked>
Reopen target: <none | planning | specification | specification-review | system-integration-design | go-code-ownership-design | technical-design-review | user/specialist decision>
```

`PASS` requires `covered` or justified `not_applicable` status for coverage/traceability, ordering/checkpoints, proof/QA, handoff/ownership, and every triggered specialist lens. Unexamined plausible lenses are not `not_applicable`; omit a lens only through the recorded local-only or scoped-down rationale.

For `PASS`, summarize the falsification checks that found no blocker. For `CONCERNS`, name each accepted risk or proof obligation and where implementation must close it. For `FAIL`, name the first blocking fact, the smallest owner that can repair it, and whether a follow-up task review is required after repair. For `WAIVED`, name the direct-path or prototype basis, why no protected-domain or non-trivial ledger trigger remains, the skipped gate, the narrower implementation claim, proof still required, and the reopen trigger.

After any task-review `FAIL`, a repaired ledger never inherits the prior approval state. The follow-up review record must name:

```text
Prior failed finding | Repaired row/section | Rechecked lens | Closure status | New verdict
```

The follow-up may be narrow, but it must verify that the repair did not leave adjacent source, owner, proof, checkpoint, cleanup, generated-source, or handoff assumptions stale.

## Readiness Status

Task-ledger review and implementation readiness use the same status vocabulary:

- `PASS`: coding may start from the approved ledger. Use only when the whole ledger is implementation-ready from the approved start point through final validation, subject only to ledger-defined checkpoint gates; the review packet is current; fan-out status or rationale is eligible; every material task is traceable to approved artifacts; the Goal Contract is strong enough for a Codex Goal; and no hidden architecture, ownership, contract, sequencing, rollout, generated-source, cleanup, or validation decision remains.
- `CONCERNS`: coding may start only with named accepted risks and explicit proof obligations carried into concrete tasks, checkpoints, `test-plan.md`, `rollout.md`, or explicit non-task rationale. Use only when the ledger is otherwise executable and the remaining concern does not ask implementation to choose a missing decision.
- `FAIL`: coding must not start; route to the named earlier phase. Use when the ledger needs repair, an approval input is missing or stale, a material source/owner/proof/checkpoint/cleanup/generated-source/handoff field is vague or absent, or implementation would need to make an unapproved decision.
- `WAIVED`: allowed only for tiny direct-path or explicitly user-requested prototype scope with explicit rationale, no protected-domain or non-trivial ledger trigger remaining, a narrower implementation claim, skipped-gate disclosure, proof still required for that limited claim, and a reopen trigger. It is not available for non-trivial ledgers just because the reviewer is local-only or the change feels low risk.

Do not use `CONCERNS` for missing ledger structure, vague Goal Contract, missing owner package/file, absent proof path, unresolved checkpoint gate, hidden generated-source authority, missing, stale, or unowned generated/mirrored drift plan or proof, or unclassified legacy cleanup. Those are `FAIL` unless the task is eligible for `WAIVED` and the waiver names the exact rationale and narrower implementation claim.

Use this decision order:

1. If a required approval input is missing, stale, `FAIL`, unresolved, or contradicted by the ledger, the result is `FAIL`.
2. If implementation must choose or invent scope, behavior, owner, source of truth, architecture, sequence, rollout, validation, generated authority, dependency/OSS, Pattern Fit, cleanup disposition, proof feasibility, or checkpoint policy, the result is `FAIL`.
3. If `tasks.md` itself needs edits for coverage, ordering, owner-file/package, Goal Contract, evidence fields, proof mapping, generated/mirrored drift, cleanup audit, resume rule, or implementation handoff, the result is `FAIL` with `planning` as the owner unless the gap belongs to an earlier phase.
4. If the ledger is executable and all residual items are bounded accepted risks or proof obligations with concrete carrying rows and reopen targets, the result is `CONCERNS`.
5. If every readiness-critical check is covered or justified not applicable and no residual accepted risk is needed, the result is `PASS`.
6. Use `WAIVED` only when the work is eligible for direct-path or prototype waiver and the review record names the exact scope reduction and proof boundary.

CONCERNS carry-forward contract:

- every accepted concern, accepted risk, or proof obligation from specification review or technical design review must map to one concrete `tasks.md` task, checkpoint gate, `test-plan.md`, `rollout.md`, or explicit non-task rationale;
- the mapping must name the source review finding, claim or risk, carrying artifact, required evidence, freshness or negative-proof rule when relevant, and reopen target if the evidence fails;
- missing, vague, or duplicate-only mapping is task-ledger review `FAIL`, not an implementation detail to infer later.

Weak implementation handoffs are `FAIL` when a later implementation prompt would have to invent the Codex Goal objective, completion condition, first task/checkpoint, proof obligations, accepted concerns, progress-update rule, allowed closeout surfaces, or blocked-stop rule. A strong handoff points to the approved ledger and lets `shared/subagents-and-handoff.md` render the implementation prompt without adding new decisions.

Readiness belongs in the planning or task-review handoff when planning artifacts exist. `workflow-plan.md` and `workflow-plans/planning.md` record the gate status when those artifacts are used; `tasks.md` may carry a short reference. Implementation may start only after task-ledger review produces `PASS`, eligible `CONCERNS`, or eligible `WAIVED`.

Use this handoff shape when recording the readiness result:

```text
Task ledger review: PASS | CONCERNS | FAIL | WAIVED
Implementation readiness: PASS | CONCERNS | FAIL | WAIVED
Consumes: reviewed `spec.md`, specification-review result, compact design or `design/`, technical-design-review result when present
Technical-design-review consumed: <not expected with rationale | PASS | CONCERNS; obligations mapped to tasks.md/test-plan.md/rollout.md with finding IDs>
Design fan-out status: <complete | scoped_down | local_only | blocked | not expected with rationale>
Subagent gates consumed: <gate status and artifact/evidence pointer, or not expected with rationale>
Ledger-review fan-out: <complete | scoped_down | local_only | not_expected | blocked>
Ledger-review fan-out rationale: <required when local_only, scoped_down, or not_expected>
Proof coverage/currentness: <task/checkpoint/final claims, command/read/manual proof, command-source anchor or narrower-proof rationale, freshness rule, negative proof when relevant, reopen target>
Reopen target: <none | planning | specification | specification-review | system-integration-design | go-code-ownership-design | technical-design-review>
```

When readiness allows coding, render the chat-only implementation prompt using `codex-goal-prompt-composer` and `shared/subagents-and-handoff.md`. If the Goal Contract or readiness record cannot support that prompt without inventing the Codex Goal objective, full-ledger completion condition, accepted concerns, proof obligations, progress-update rule, allowed closeout surfaces, or blocked-stop rule, the result is `FAIL` to planning instead of a weak implementation handoff.
