# Spec-First Workflow

Detailed runtime companion to [AGENTS.md](../AGENTS.md) for trigger-based spec-first work.

## 1. Authority And Purpose

`AGENTS.md` is authoritative for roles, invariants, execution-shape triggers, and hard boundaries. When the two documents diverge, follow `AGENTS.md` first and repair this companion. This document explains how to apply those rules in task-local artifacts without forcing the full bundle onto every bounded change.

The workflow keeps the same quality concerns:

- clear framing and scope cuts;
- final decision ownership by the orchestrator;
- read-only advisory subagents when independent evidence or challenge is needed;
- canonical `spec.md` when a task-local decision artifact exists;
- executable task handoff when implementation is non-trivial;
- fresh validation evidence before completion claims.

The simplification is about artifact depth, not decision quality. Use the smallest workflow shape that preserves correctness.

Default decision quality:

- Unless the user explicitly asks for prototype, quick, simple, temporary, or intentionally staged delivery, choose the production-ready architecture and system-design answer for the accepted scope.
- Default to a target-state plan: decide the final architecture now, and make the implementation ledger include the work needed to reach that state without remembered-later cleanup.
- Do not create an MVP-now/future-hardening split when the production-ready decision is knowable and in scope.
- Temporary bridges, compatibility shims, feature flags, canaries, or staged rollout are not default recommendations. Use them only when the user requests staging or a live external constraint makes a one-step target-state change unsafe or impossible.
- When staging is unavoidable, record the target state, exit criteria, removal/proof tasks, and owner in the owning artifact as part of the accepted scope. Do not leave the cleanup as a follow-up or future-hardening note.
- Scope cuts are allowed only as clear non-goals, constraints, or accepted risks. They are not a license to defer required architecture, ownership, contract, reliability, security, or validation decisions.

## 2. Execution Shapes

| Shape | Use When | Artifact Depth | Gate |
| --- | --- | --- | --- |
| `direct path` | Tiny, reversible, one surface, obvious validation, no protected-domain trigger. | Usually none; a short inline plan or chat note is enough. | Local first-read sanity check and fresh proof. |
| `lean local` | Bounded non-trivial single-domain work, stable ownership, limited research, and local reasoning can close the decision frontier. | `spec.md` plus `tasks.md` by default; optional preserved research, one `design/overview.md`, or `workflow-plan.md` only when triggered. | Inline `Risk Challenge`; mandatory technical design review checkpoint when separate design depth is triggered; post-ledger task review/readiness gate; no mandatory subagent. |
| `full orchestrated` | Cross-domain, ambiguous, hard-to-reverse, high-impact, long-running, user-requested agent-backed, or protected-domain work. | `workflow-plan.md`, triggered `workflow-plans/<phase>.md`, preserved research when useful, approved `spec.md`, triggered design bundle, mandatory technical design review record, `tasks.md`, optional companion artifacts. | Formal challenge/review lanes as triggered, mandatory technical design review when design depth is triggered, post-ledger task review/readiness gate, and strict phase boundaries. |

Use `lean local` for bounded non-trivial single-domain work. This changes the amount of workflow ceremony, not the expected production readiness of the chosen solution.

### Escalation Triggers

Escalate from direct or lean local to full orchestrated when the task touches:

- public API, generated contracts, SDK behavior, or compatibility promises;
- persisted data, migrations, backfills, cache semantics, retention, or deletion behavior;
- auth, authorization, tenant isolation, secrets, browser session, CORS/CSRF, or abuse risk;
- money, billing, quotas, credits, reserves, or entitlements;
- concurrency, background workers, retry policy, lifecycle, shutdown, or shared state;
- deployment, rollout, rollback, failback, mixed-version behavior, or migration order;
- multiple independent owners or ambiguous source-of-truth;
- unclear validation path;
- broad audits, user-requested subagents, or explicit strict phase boundaries.

If a trigger appears after work starts, record the reopen target and move to the fuller path instead of stretching the current shortcut.

### Default Session Boundary Policy

Phase arrows describe order, not a default license to collapse multiple phases into one chat session.

Default rule: one session owns one workflow phase, then stops. When the phase has a next phase or reopen target, update the relevant workflow state and end the final chat response with a copy-pastable `Recommended next-session prompt` derived from the recorded artifacts.

A broad user request such as "do the full workflow", "implement the PRD and architecture fully", or "create all necessary documents" advances the overall workflow, but it does not override the one-phase session boundary. Start with the next valid phase, finish that phase honestly, then stop with the next-session prompt.

This boundary rule applies to:

- workflow planning;
- research and fan-in;
- specification and clarification-gate reconciliation;
- technical design;
- technical design review and reconciliation;
- task planning, task-ledger review, and implementation-readiness handoff;
- post-code review or reconciliation phases;
- validation and closeout;
- targeted reopen phases.

The normal exception is implementation from an approved `tasks.md` that has passed the post-ledger task-review/readiness gate. Once implementation readiness is `PASS`, eligible `CONCERNS`, or eligible `WAIVED`, the implementation session may execute the approved ledger items and the proof named by the ledger without stopping between task IDs. After that point, workflow-control files are pre-code routing history unless the approved ledger explicitly names a separate review, validation, or reopen phase file as part of the work.

Direct path work has no durable phase boundary by default, so it may still complete inline with fresh proof.

## 3. Artifact Model By Shape

### Direct Path

Direct path may skip durable workflow artifacts. It still needs:

- a bounded local understanding of the requested change;
- no protected-domain trigger;
- an explicit proof command or manual proof before claiming completion.

Do not create `workflow-plan.md`, `workflow-plans/`, `spec.md`, or `tasks.md` just for ceremony.

### Lean Local

Lean local is the default for bounded non-trivial single-domain work.

Expected artifacts:

- `spec.md`: compact decision record.
- `tasks.md`: executable task ledger and implementation handoff after task review passes.

Conditional artifacts:

- `research/*.md`: when evidence must survive resume, audit, or later synthesis.
- `design/overview.md`: when compact design answers are too dense for `spec.md` but do not need split design files.
- `workflow-plan.md`: when multi-session state or reopen routing needs a durable control file.

If lean local uses a separate `design/overview.md`, run and record a technical design review checkpoint before writing or approving `tasks.md`. The checkpoint may be a local read-only orchestrator review when no formal lane trigger exists, but it must be distinct from the design-writing pass and must record `PASS`, `CONCERNS`, or `FAIL`.

Not expected by default:

- `workflow-plans/<phase>.md`;
- split `design/component-map.md`, `design/sequence.md`, and `design/ownership-map.md`;
- `test-plan.md`;
- `rollout.md`;
- review or validation phase files.

Lean local must not become an unstructured shortcut. It requires an inline `Risk Challenge`, executable tasks, and fresh proof.

### Full Orchestrated

Full orchestrated keeps the existing full workflow, but all heavier artifacts are trigger-scoped.

Separate technical design is the exception to optional review routing: once `design/overview.md` or split `design/` is triggered, technical design review is mandatory before planning. The review record may live in `workflow-plan.md`, the active phase file, or `workflow-plans/technical-design-review.md` when the review needs durable routing, lanes, blockers, or a session boundary.

Typical layout:

```text
specs/<feature-id>/
  workflow-plan.md
  workflow-plans/
    workflow-planning.md        # only for a dedicated routing phase
    research.md                 # only for a dedicated research phase
    specification.md            # only when formal specification routing is needed
    technical-design.md         # only when dedicated design routing is needed
    technical-design-review.md  # required when separate technical design is triggered and review routing needs durable state
    planning.md                 # only when dedicated planning routing is needed
    review-phase-N.md           # only when planning names a multi-session review checkpoint
    validation-phase-N.md       # only when planning names a multi-session validation checkpoint
  research/
    <topic>.md                  # only when evidence needs to persist
  spec.md
  design/
    overview.md                 # entrypoint when design is triggered
    component-map.md            # split only when useful
    sequence.md                 # split only when useful
    ownership-map.md            # split only when useful
    data-model.md               # conditional
    dependency-graph.md         # conditional
    contracts/                  # conditional design context, not runtime authority
  tasks.md
  test-plan.md                  # conditional
  rollout.md                    # conditional
```

Do not point agents at a specific task-local `specs/...` bundle as required precedent unless that directory exists in the current checkout.

## 4. Lean `spec.md`

Lean specs should answer the planning-critical questions without becoming a design bundle.

Recommended shape:

```markdown
# <Feature / Change>

Mode: lean local
Status: planned | implementing | verified

## Intent
What changes and why.

## Scope / Non-goals
In:
- ...

Out:
- ...

## Behavior / Contract Delta
ADDED:
- ...

MODIFIED:
- ...

REMOVED:
- ...

## Decisions
- D1: ...
- D2: ...

## Compact Design
Affected surfaces:
- `internal/...`

Ownership / source of truth:
- ...

Sequence / failure behavior:
- ...

## Risk Challenge
1. What irreversible or externally visible decision could be wrong?
   Answer: ...
2. What hidden invariant or owner could this break?
   Answer: ...
3. What fresh proof will make the completion claim trustworthy?
   Answer: ...
Gate: PASS | CONCERNS | FULL_REQUIRED

## Task Handoff
Use `tasks.md`.

## Validation
Forward-looking proof obligations.

## Outcome
Pending until fresh validation evidence exists.
```

Rules:

- `Behavior / Contract Delta` describes added, modified, and removed behavior instead of restating the whole system.
- `Compact Design` answers affected surfaces, ownership/source-of-truth, and sequence/failure behavior. If those answers become dense or contested, split into design artifacts or escalate.
- `Risk Challenge` is the lean replacement for a formal challenge lane only when no escalation trigger is present.
- `FULL_REQUIRED` blocks lean coding and routes to full orchestrated work.
- `Outcome` stays pending until fresh evidence exists.

## 5. Lean `tasks.md`

Lean `tasks.md` is the main execution surface.

Recommended shape:

```markdown
# Tasks

## Goal Contract

Goal objective: Complete <feature/change> by executing this ledger from `T001` through final validation.
Stopping condition: all required tasks are checked, required proof passes or records a concrete blocker, and ledger-owned closeout evidence is current.
Read first: `spec.md`, plus `design/overview.md` when that artifact exists. Do not read `workflow-plan.md` for implementation unless no approved ledger exists yet.
Do not change: <non-goals and preserved constraints from `spec.md`>
Progress log: after each checkpoint, update the checkbox/evidence lines; if blocked, stop and record `Blocked:` with the missing input or failing proof.

## Implementation Handoff

Task ledger review: PASS | CONCERNS | FAIL | WAIVED
Implementation readiness: PASS | CONCERNS | FAIL | WAIVED
Consumes: `spec.md`, compact design or `design/`, technical-design-review result when present
Proof: <smallest sufficient proof command or manual proof>
Reopen target: <none | planning | specification | technical-design | technical-design-review>

## Tasks

- [ ] T001 Add failing proof for <behavior>
  Files: `internal/...`
  Proof: targeted proof fails for the expected reason before implementation.
  Evidence: Pending.

- [ ] T002 Implement scoped production behavior
  Files: `internal/...`
  Proof: targeted proof passes.
  Evidence: Pending.

- [ ] T003 Run validation and record outcome
  Proof: `go test ./...`, `rtk make check`, or the smallest relevant command.
  Evidence: Pending.
```

Rules:

- Use markdown checkboxes and stable task IDs.
- Name one objective and one stopping condition so the ledger can drive a long-running `/goal` without extra chat context.
- Treat non-trivial `tasks.md` as Goal-ready by default in this repository. That means the ledger should contain the Goal Contract fields a later handoff needs; it does not mean a Goal prompt is rendered before the ledger passes review/readiness.
- Keep the Goal contract derivative: it may summarize approved scope, constraints, proof, and stop rules, but must not introduce new decisions or weaken implementation readiness.
- Write the objective and stopping condition so a later implementation handoff can explicitly ask the next session to set a Codex Goal covering all executable ledger tasks.
- Point implementation at the files, docs, plans, or logs it must read first.
- Include checkpoint/progress rules when the ledger spans multiple tasks, sessions, or proof loops.
- Name dependencies when task order matters.
- Name exact files when known, or narrow artifact/package surfaces when exact file choice is not knowable yet.
- Include proof expectations and an evidence slot per task or checkpoint.
- Do not include unresolved open questions, `TBD` decisions, or pending decision gates in `tasks.md`. A ready ledger may carry accepted risks and proof obligations, but any implementation-blocking question must reopen specification, technical design, or technical design review.
- Treat a newly written `tasks.md` as a draft until the task-ledger review has compared it against `spec.md`, required design context, technical-design-review obligations, and triggered validation or rollout obligations.
- For behavior changes and bug fixes, proof-first or test-first is the default.
- For docs, config, or mechanical changes where a failing test is not useful, record the waiver as a proof obligation in `tasks.md`, not only in chat.

## 6. Workflow Control Artifacts

### `workflow-plan.md`

Use `workflow-plan.md` when cross-phase or multi-session state is real. It owns:

- execution shape and rationale;
- current phase and phase status;
- session boundary state;
- next-session routing;
- next-session context bundle;
- artifact status and trigger rationale;
- blockers, accepted assumptions, accepted risks, and reopen targets;
- active gate status such as clarification, adequacy, task-ledger review, or implementation readiness.

It does not own final decisions, technical design, executable tasks, raw research, or validation transcripts.

Once `tasks.md` is approved, `workflow-plan.md` no longer owns implementation progress, task completion, or closeout state. It may remain useful historical context, but agents must not update it during implementation or closeout. Pre-created review or validation phase files may be updated only when the approved ledger explicitly names them as required artifacts.

### `workflow-plans/<phase>.md`

Create a phase-local file only when the phase needs durable local orchestration: multi-lane routing, fan-in, formal challenge status, a multi-session stop rule, or named review/validation checkpoints.

It owns:

- local lanes or order/parallelism;
- phase-local completion marker;
- local stop rule;
- next action;
- local blockers;
- gate or handoff status for that phase.

It must not replace `spec.md`, `design/`, or `tasks.md`.

### Status Vocabulary

Use status words proportionally: `approved`, `draft`, `missing`, `blocked`, `waived`, `not expected`, or `conditional`.

- `waived` requires eligible direct-path, lean, or explicitly user-requested prototype rationale and scope.
- `not expected` requires trigger-based rationale when the artifact would otherwise be plausible.
- `conditional` means a later phase must decide the trigger because current evidence is insufficient; do not use it to postpone a knowable production-readiness decision.

## 7. Research

Research is a concern, not always a dedicated phase.

Use local research when the orchestrator can gather the needed evidence directly.

Use read-only fan-out when distinct unresolved questions need independent evidence or challenge.

Preserve `research/*.md` only when it materially helps later synthesis, auditability, or resume. A good research note includes:

- question or scope;
- findings with evidence and limits;
- source notes;
- conflicts, weak evidence, or assumptions;
- handoff implication.

Research notes support decisions but do not own them. Final decisions belong in `spec.md`.

## 8. Specification And Challenge Gates

`spec.md` is always the decision authority for task-local decisions.

For direct path, no `spec.md` is usually needed.

For lean local:

- write a compact `spec.md`;
- run the inline `Risk Challenge`;
- proceed only when the gate is `PASS` or `CONCERNS` with named proof obligations;
- escalate when the gate is `FULL_REQUIRED`.

For full orchestrated or protected-domain work:

- run formal `spec-clarification-challenge` before non-trivial `spec.md` approval unless an explicit eligible waiver exists;
- for broad or multi-domain full-orchestrated, protected-domain, high-impact, hard-to-reverse, cross-domain, or user-requested deep challenge work, use multi-challenger lens fan-out rather than one generic challenger by default;
- use read-only challenger output as questions for orchestrator reconciliation, not as authority;
- store final reconciled outcomes in `spec.md`;
- record gate status in `workflow-plan.md` and the active phase file when those files are used.

Formal clarification asks only approval-changing questions. Ordinary downstream design detail should be recorded as a constraint, proof obligation, follow-up, or `defer_to_design`, not as a reason to inflate `spec.md`. Do not classify architecture, ownership, contract, reliability, security, rollout, or validation choices this way when they are required to choose a production-ready solution for the accepted scope.

Default broad clarification lenses:

- scope and spec coherence;
- domain invariants and edge cases;
- architecture ownership and dependency boundaries;
- API, data, compatibility, and source-of-truth consequences;
- security, reliability, delivery, and validation proof.

Each lens is a separate read-only lane, usually `challenger-agent` with `spec-clarification-challenge`. Lanes may run in parallel when their questions are independent. Add extra lanes for real independent approval-risk domains, including when one default lens bundles domains that are independently approval-critical for the task. Use fewer lanes only with a recorded scoped-down rationale; a single lane is appropriate only for a narrow formal gate whose approval risk is concentrated in one question.

Before spawning, convert every lens into a concrete approval-critical question and lens-specific inspect-first list. Do not send five challengers the same generic "challenge this spec" prompt. If two lenses produce the same question, merge them or split the real underlying owner question before fan-out.

`Risk Challenge=CONCERNS` in lean local does not by itself trigger formal multi-challenger clarification. It requires named proof obligations and a check for unresolved scope, ownership, validation, or escalation gaps. Route to formal clarification only when those gaps cannot be honestly closed inline or another escalation trigger appears.

Multi-lane workflow-control records should use:

```text
Clarification challenge: complete | blocked | waived
Lanes: <agent + skill summary>
Lenses: <lens list>
Scoped-down rationale: <why fewer than the broad default, when applicable>
Resolution: <orchestrator-owned fan-in result>
```

`Lens` is metadata for coverage, not a replacement for `spec-clarification-challenge` classifications. Lane outputs use `blocks_spec_approval`, `blocks_specific_domain`, and `non_blocking_but_record`; shared handoffs use the classifications from `docs/subagent-contract.md`.

The orchestrator owns fan-in:

- deduplicate overlapping questions and findings;
- compare conflicting assumptions across lanes;
- classify each surviving point by strongest justified impact: approval blocker, domain reopen, record-only constraint, proof obligation, accepted risk, or no-action item;
- preserve a short fan-in table or equivalent status in the workflow-control file: lens, strongest finding, classification, action, and owner;
- treat lane-level missing input, unresolved blockers, and material blocker-severity conflicts as blocking the relevant approval area until answered, explicitly waived or accepted as risk, or routed to the owning phase;
- update `spec.md` only with final reconciled outcomes, not raw lane transcripts;
- reopen research, design, planning, or a specialist lane when a finding exposes a missing owner decision.

## 9. Design Depth

Design is content-triggered.

Lean local may keep design answers in `spec.md` `Compact Design` when affected surfaces, ownership, and sequence/failure behavior are concise and uncontested.

Use one `design/overview.md` when lean design context needs more room but still fits one artifact.

Split into design artifacts when the task needs durable, planning-critical context by dimension:

- `design/component-map.md`: affected packages, modules, generated surfaces, adapters, responsibility changes, stable surfaces, and intentional non-touches.
- `design/sequence.md`: runtime order, sync/async boundaries, side effects, failure points, retry/recovery behavior, and parallel versus sequential behavior.
- `design/ownership-map.md`: source-of-truth ownership, allowed dependency direction, generated-code authority, adapter responsibility, and explicit non-owners.

Conditional design artifacts:

- `design/data-model.md`: persisted state, schema, cache contract, projections, replay behavior, retention, or migration shape.
- `design/dependency-graph.md`: module/package dependency shape, generated-code dependency flow, coupling risk, or source-of-truth ambiguity.
- `design/contracts/`: API/event/generated/material internal interface design context. Runtime authorities like `api/openapi/service.yaml` still win.
- `test-plan.md`: validation obligations are too large or layered for `tasks.md`.
- `rollout.md`: migration sequencing, compatibility window, deploy order, rollback, or failback matters.

If a design trigger is real but the required decision is missing, reopen specification or technical design instead of burying it in `tasks.md`.

## 10. Technical Design Review

Technical design review is mandatory whenever separate design depth is triggered. It is the special pre-planning gate that tests whether the design bundle is coherent enough for executable planning.

This gate is not required for direct path work or for lean-local work whose design stays inside `spec.md` `Compact Design`; the inline `Risk Challenge` covers that smaller path. It is required when lean local creates a separate `design/overview.md`, and it is required for full-orchestrated triggered design.

Run technical design review after the design bundle and any triggered conditional artifacts are written, but before `tasks.md` or the task-ledger review/readiness handoff is approved.

If technical design review returns `FAIL`, the next action is a reopen of technical design or specification. After the repair, planning still waits for a follow-up technical design review verdict on the revised packet. The follow-up may be targeted to the failed findings and changed artifacts when the repair is narrow, but it must still check that adjacent design assumptions remain valid and record a new or explicitly updated gate status.

The review packet must be explicit enough that the reviewer does not rediscover phase state from scratch:

- approved `spec.md`;
- design entrypoint and triggered design artifacts, with status and trigger rationale;
- triggered `test-plan.md`, `rollout.md`, or explicit not-expected rationale when those surfaces matter;
- workflow-control paths that define the current phase, blockers, and expected review result;
- known assumptions, accepted trade-offs, non-goals, and reopen conditions.

The review must be read-only and risk-driven:

- inspect approved `spec.md`, the design bundle, triggered conditional artifacts, `docs/repo-architecture.md` when boundaries matter, and relevant specialist outputs;
- check source-of-truth ownership, dependency direction, runtime sequence, failure behavior, conditional artifact triggers, validation/rollout handoff, and accidental complexity;
- separate design defects from implementation preferences;
- identify any live fork where two plausible design options would materially change ownership, interfaces, data shape, async or sync semantics, operability, rollout, or validation, and verify the design has selected one with a rejection reason for the other;
- challenge the design from the first safe implementation slice: ask whether planning can create executable tasks without adding architecture, ownership, contract, sequencing, rollout, or validation policy;
- choose the strongest justified gate status, avoiding both over-blocking on proof-only concerns and under-blocking on missing ownership, contract, sequencing, rollout, or validation decisions;
- explain why the status is not stronger or weaker, especially for `CONCERNS` versus `FAIL`;
- when recommending `FAIL`, name the smallest reopen target, the decision or artifact that must change, and the concrete condition that a follow-up review should verify;
- return findings as advisory evidence for orchestrator reconciliation.

Technical design review is not a second design pass. If a finding requires a new decision, rewrite of the design bundle, or changed approval boundary, route it back to technical design or specification instead of solving it inside review.

For lean local with one `design/overview.md`, a local read-only orchestrator review is acceptable when no formal lane trigger exists, but the checkpoint and result must be recorded before `tasks.md`.

For full orchestrated triggered design, use at least one distinct read-only review lane unless a scoped-down local-review rationale is recorded. Add specialist lanes when independent API, data, security, reliability, observability, delivery, performance, or QA design risks are real. A design-integrator lane is the default fit when the hard part is coherence across specialist concerns.

Review gate status:

- `PASS`: planning may start from the reviewed design context.
- `CONCERNS`: planning may start only with named accepted design risks and proof obligations.
- `FAIL`: planning must not start; reopen technical design or specification. Repair alone is not enough to enter planning; the revised packet needs a follow-up review verdict of `PASS` or `CONCERNS`.

Gate decision discipline:

- Use `PASS` only when the reviewer has tried to falsify the design against source-of-truth ownership, sequence/failure behavior, validation, rollout, and artifact-trigger expectations and found no planning blocker.
- Use `CONCERNS` only when the design is coherent enough for planning and the remaining risk can be carried as a named accepted risk or proof obligation without asking implementation to choose a missing design decision.
- Use `FAIL` when planning would have to choose between live design options, repair ownership or dependency direction, define missing contract/data/rollout/failure semantics, or resolve a spec/design contradiction.
- Use `record_only` or no finding for cleaner-code preferences unless the issue changes planning safety or production-readiness proof.

Classify findings by strongest planning impact:

- `blocks_planning`: planning would invent or hide an important decision if it started now.
- `reopens_design`: the design bundle must change before review can pass.
- `reopens_spec`: the approved problem frame, invariant, scope, or contract must change.
- `accepted_risk_candidate`: the risk may be accepted only if the orchestrator names the reason and boundary.
- `proof_obligation`: planning may proceed only if the obligation is carried into `tasks.md`, `test-plan.md`, or `rollout.md`.
- `record_only`: useful context that does not affect planning entry.

Record the review result in the active workflow-control surface: `workflow-plan.md`, `workflow-plans/technical-design-review.md` when a dedicated review phase needs durable routing, or the lean-local artifact that owns the review checkpoint. The record must name the reviewed packet, reviewer or lane, scope, findings, orchestrator resolution, final gate status, and planning-input obligations. Follow-up review after `FAIL` must also name the prior failed review, the revised artifacts or decisions, which blockers were closed, any remaining accepted risks or proof obligations, and the new gate status. `CONCERNS` is valid only when every accepted risk and proof obligation is named for planning. Post-code discovery of a missing required technical design review reopens the earlier phase instead of creating a new review artifact after implementation starts.

## 11. Planning, Task Review, And Implementation Readiness

Planning turns approved decisions and required design context into `tasks.md`.

Direct path may use an inline plan.

Lean local and full orchestrated work use `tasks.md` for non-trivial implementation.

Planning must not invent missing design context. If exact tasking requires a missing decision, reopen the earlier concern.

`tasks.md` is a draft until the task-ledger review/readiness gate checks it against the approved artifact chain. This gate must run after the ledger is written or materially repaired and before implementation starts.

Task-ledger review must verify:

- every in-scope behavior, non-goal, constraint, and accepted decision from `spec.md` is represented in executable tasking, preserved constraints, or explicit non-task rationale;
- required compact design, `design/overview.md`, or split `design/` ownership, sequence, dependency, failure, and conditional-artifact rules are reflected in task order and proof expectations;
- technical-design-review `CONCERNS` are carried as named accepted risks and proof obligations, and any `FAIL`, unresolved `blocks_planning`, `reopens_design`, or `reopens_spec` finding blocks handoff;
- triggered `test-plan.md`, `rollout.md`, review phase, or validation phase obligations are either represented in the ledger or explicitly marked not expected with rationale before code starts;
- the ledger contains no open-question section, unresolved decision gate, `TBD`, hidden design work, or instruction for implementation to decide architecture, ownership, contract, sequencing, rollout, or validation policy.

If the review finds a blocker, use the smallest owning reopen target:

- `planning` for missing task coverage, wrong ordering, vague proof, missing evidence fields, or workflow-control handoff gaps that do not change approved decisions or design;
- `technical design review` when a required review verdict is missing, stale after repair, or has unresolved blocking findings;
- `technical design` when the ledger needs ownership, sequence, dependency, rollout, validation, or conditional-artifact context the design does not provide;
- `specification` when the missing or contradictory point changes accepted scope, behavior, invariant, public contract, non-goal, or approval boundary.

Task-ledger review and implementation readiness use the same status vocabulary:

- `PASS`: coding may start; no hidden architecture, ownership, contract, sequencing, rollout, or validation decision is needed for the next slice.
- `CONCERNS`: coding may start only with named accepted risks and explicit proof obligations; these concerns must be closed as decisions, not open questions.
- `FAIL`: coding must not start; route to the named earlier phase.
- `WAIVED`: allowed only for tiny direct-path or explicitly user-requested prototype scope with explicit rationale.

Readiness belongs in the planning handoff when planning artifacts exist. `workflow-plan.md` and `workflow-plans/planning.md` record the gate status when those artifacts are used; `tasks.md` may carry a short reference. Implementation may start only after task-ledger review produces `PASS`, eligible `CONCERNS`, or eligible `WAIVED`.

Planning also consumes the technical design review result whenever separate design depth was triggered. Missing review, blocking review, or repaired design after `FAIL` without a follow-up verdict is a planning-entry failure and a task-review blocker, not a detail to infer inside `tasks.md`. When the review result is `CONCERNS`, planning must copy the accepted design risks and proof obligations into the task-ledger review/readiness handoff and the relevant ledger or companion artifacts.

## 12. Coding, Review, Reconciliation, And Validation

Coding consumes the approved task handoff. It may create or edit code, tests, migrations, configs, generation inputs, and generated output required by the task ledger.

Implementation sessions may continue across the approved `tasks.md` items and the ledger's named proof checks. They must not use implementation momentum to create or approve missing specification, design, planning, review, or validation-phase artifacts.

Post-code work is ledger-driven. It may update only:

- existing `tasks.md` checkbox/progress state;
- `spec.md` `Validation` and `Outcome`.
- existing `workflow-plans/review-phase-N.md` or `workflow-plans/validation-phase-N.md` only when the approved `tasks.md` explicitly names that pre-created phase file as part of the post-code checkpoint.

Do not update `workflow-plan.md` or phase-control files merely because they exist. After `tasks.md` is approved, those files are not the implementation source of truth.

Do not create new workflow/process artifacts after implementation starts. Reopen the earlier phase that owns the missing artifact instead.

Review is read-only and risk-driven. Review findings are advisory until the orchestrator reconciles them.

Validation uses fresh evidence. A closeout claim is valid only when the commands or manual proof actually cover that claim.

## 13. Subagents

Subagents are triggered by unresolved owning questions, not by phase ceremony.

Read-only must be enforced by the actual execution choice. If a lane cannot reliably stay read-only, keep that question in the main orchestrator flow instead of delegating it.

Use `docs/subagent-contract.md` and `docs/subagent-brief-template.md` for reusable brief shape.

Every lane needs:

- goal and exact question;
- scope and constraints;
- lens or specialist domain when multiple challenge lanes share the same artifact;
- expected output;
- evidence requirement;
- skill name or `no-skill`;
- read-only boundary.

A lane uses at most one skill. If the selected skill defines a stricter deliverable, follow it. Otherwise use the shared envelope from `docs/subagent-contract.md`. Multi-lane challenge improves coverage only when lanes have distinct lenses and an explicit fan-in path.

## 14. Resume Order

Resume from artifacts, not chat memory.

If approved `tasks.md` exists and implementation, review, validation, or closeout is next:

1. read `tasks.md`;
2. read the artifacts named by `tasks.md`, usually `spec.md` and any required design, test-plan, or rollout context;
3. read `workflow-plans/<phase>.md` only when `tasks.md` explicitly names a pre-created review or validation phase file.

If there is no approved `tasks.md` and `workflow-plan.md` exists:

1. read `workflow-plan.md`;
2. read the current `workflow-plans/<phase>.md` if the task uses one;
3. read the files named in the `Next session context bundle`;
4. then read phase artifacts as needed.

If there is no approved `tasks.md` and no `workflow-plan.md` because the task is direct or lean:

1. read `spec.md` when it exists;
2. read `tasks.md` when implementation or validation is next;
3. read optional `research/*.md` or `design/overview.md` only when named or needed.

Treat missing expected artifacts as incomplete unless an explicit waiver covers that exact artifact.

## 14. Final Chat Handoff

When any non-implementation workflow phase reaches a boundary and a next session or reopen target exists, the final chat response must include a copy-pastable recommended next-session prompt derived from recorded artifacts. This is default behavior; the user does not need to ask the agent to stop or produce the handoff prompt.

Assume the next session is context-blind: it can read repository files, but it cannot see the current chat. The prompt should carry a short task-specific context capsule that explains the current state, why the named next step is next, and what the next session must not lose. It should not become a transcript, broad project summary, or second copy of the artifacts.

Select context by relevance:

- include the current workflow state, accepted objective, and the reason this exact next phase or reopen target is next;
- include exact artifact paths, task IDs, phase names, commands, blocker names, accepted decisions, accepted assumptions, accepted risks, and proof obligations that matter to the next phase;
- include one-line reasons for non-obvious files in the context bundle so the next session knows why to read them;
- omit generic repository rules already covered by `AGENTS.md`, long artifact excerpts, resolved debate history, unrelated prior-session details, and context the next phase can cheaply rediscover from the listed files;
- when uncertain, include a bounded assumption or reopen target instead of padding the prompt with unrelated context.

The recommended prompt should be operational, not just descriptive. Include:

- the exact next phase or reopen target;
- the artifact read order, task-local paths, and short reasons for any non-obvious context files;
- the immediate objective and expected output for that one phase;
- important blockers, accepted assumptions, accepted risks, and proof obligations from recorded state;
- a stop rule telling the next session to complete only that phase, update workflow state, and produce the following next-session prompt if another phase remains.

For implementation from approved `tasks.md` that has passed task-ledger review/readiness, compose the prompt with `.agents/skills/codex-goal-prompt-composer/SKILL.md`. The prompt must explicitly tell the next session to set a Codex Goal first, then execute all required tasks in the approved ledger from start to finish. It must not rely on a slash command being parsed from the handoff. It may tell the next session to execute the approved ledger and run its named proof without stopping between task IDs. It must still prohibit creating or approving missing pre-code workflow artifacts during implementation.

Implementation goal handoff rules:

- use `codex-goal-prompt-composer` whenever the recommended next-session prompt sets a Codex Goal;
- apply that skill's Goal Line Quality Gate before returning the prompt;
- start the fenced prompt with `First, set a Codex Goal for this session:` followed by a short durable goal objective;
- the next paragraph must say `After the goal is set, execute every required task in <tasks.md path> from start to finish`;
- derive `<approved objective>` and `<verifiable stopping condition>` from the `tasks.md` Goal Contract and implementation-readiness handoff;
- scope the goal to all executable tasks in the approved ledger, from the recorded first task or checkpoint through final validation, not just the first task ID;
- keep the Codex Goal objective as a durable objective only; do not pack artifact lists, constraints, risks, commands, or detailed execution rules into it;
- put all execution details under `Implementation brief` so the durable goal stays stable while the working instructions remain readable;
- include working directory, artifact read order, task-local paths, accepted constraints, accepted risks, proof obligations, and named validation commands or manual proof;
- if readiness is `CONCERNS` or `WAIVED`, keep the Codex Goal objective focused on the approved objective and put the concern, waiver rationale, and required proof obligations in the implementation brief;
- tell the next session to update only ledger-owned progress/evidence and closeout surfaces permitted by `tasks.md`;
- if the `tasks.md` Goal Contract is missing or too vague to form a verifiable Codex Goal, do not invent a broad objective; reopen planning to repair the Goal Contract or mark the implementation prompt as blocked;
- include a blocked-stop rule: if an implementation-blocking decision, missing artifact, or failing proof cannot be resolved inside the approved ledger, stop, record the blocker in the allowed ledger/closeout surface, and return the exact reopen target instead of inventing new workflow artifacts.

Recommended implementation prompt shape:

```text
First, set a Codex Goal for this session:
Complete <approved objective> by executing every required task in `<task-local>/tasks.md` without stopping until <verifiable stopping condition>.

After the goal is set, execute every required task in `<task-local>/tasks.md` from start to finish. Start at <T001 or recorded checkpoint>, continue through the ledger's final validation/proof, and do not redefine success around a smaller slice.

Implementation brief:

Work in `<absolute repo path>`.
Read first:
- `<task-local>/tasks.md` because it is the approved implementation ledger and source of truth.
- `<task-local>/spec.md` because it is the canonical decision record.
- <additional task-local artifacts named by `tasks.md`, each with a one-line reason>.

Current state:
- Next phase: implementation.
- Task-ledger review: <PASS | CONCERNS | WAIVED>.
- Implementation readiness: <PASS | CONCERNS | WAIVED>.
- First executable task/checkpoint: <T001 or named checkpoint>.
- Accepted concerns or waiver: <none | named concern/waiver plus proof obligation>.

Execution rules:
- Execute all required tasks in `tasks.md` in dependency order through the ledger's named proof; do not stop between task IDs unless blocked.
- Preserve the accepted constraints, non-goals, risks, and proof obligations recorded in the listed artifacts.
- Do not create or approve missing pre-code workflow artifacts during implementation.
- Update existing `tasks.md` progress/evidence and any ledger-owned closeout surfaces exactly as allowed by `tasks.md`.
- If blocked by a missing decision, missing artifact, or unresolved failing proof outside the approved ledger, stop with the blocker, evidence, and exact reopen target.
```

Use no prompt when the workflow is honestly done.

The prompt is chat-only. It is not a workflow artifact and must not become a second source of truth.

Before returning the prompt, apply the start test: a new session with no chat history should know the single next phase, why it is next, what to read first, what constraints and proof obligations matter, and where to stop. Remove any sentence that does not help that session start or avoid a real mistake.

## 15. Anti-Patterns

Avoid:

- treating full orchestrated as the default for every non-trivial task;
- using direct path for risky, ambiguous, public, data, security, money, reliability, concurrency, or rollout work;
- using lean local without `spec.md`, `tasks.md`, inline `Risk Challenge`, and proof;
- making `workflow-plans/<phase>.md` a second master plan, spec, design bundle, or task ledger;
- planning non-trivial implementation from `spec.md` alone when design context is triggered;
- starting implementation from an unreviewed `tasks.md` or treating a draft ledger as approval;
- approving non-trivial specs while formal challenge is required and unresolved;
- splitting work into MVP plus future hardening when the production-ready decision is knowable and in scope;
- picking the fastest or simplest architecture by default instead of the best production-ready choice for the accepted scope;
- creating `test-plan.md`, `rollout.md`, split design files, or review/validation phase files for completeness;
- creating new process artifacts after coding starts;
- using subagents for broad ceremony rather than narrow unresolved questions;
- claiming done without fresh, scope-matched evidence.
