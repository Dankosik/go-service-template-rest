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

The simplification is about artifact depth. Use the smallest workflow shape that preserves correctness.

## 2. Execution Shapes

| Shape | Use When | Artifact Depth | Gate |
| --- | --- | --- | --- |
| `direct path` | Tiny, reversible, one surface, obvious validation, no protected-domain trigger. | Usually none; a short inline plan or chat note is enough. | Local first-read sanity check and fresh proof. |
| `lean local` | Bounded non-trivial single-domain work, stable ownership, limited research, and local reasoning can close the decision frontier. | `spec.md` plus `tasks.md` by default; optional preserved research, one `design/overview.md`, or `workflow-plan.md` only when triggered. | Inline `Risk Challenge`; mandatory technical design review checkpoint when separate design depth is triggered; no mandatory subagent. |
| `full orchestrated` | Cross-domain, ambiguous, hard-to-reverse, high-impact, long-running, user-requested agent-backed, or protected-domain work. | `workflow-plan.md`, triggered `workflow-plans/<phase>.md`, preserved research when useful, approved `spec.md`, triggered design bundle, mandatory technical design review record, `tasks.md`, optional companion artifacts. | Formal challenge/review lanes as triggered, mandatory technical design review when design depth is triggered, and strict phase boundaries. |

Use `lean local` for bounded non-trivial single-domain work.

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
- `tasks.md`: executable task ledger and implementation handoff.

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

Readiness: PASS | CONCERNS
Consumes: `spec.md`
Proof: <smallest sufficient proof command or manual proof>

- [ ] T001 Add failing proof for <behavior>
  Files: `internal/...`
  Proof: targeted proof fails for the expected reason before implementation.

- [ ] T002 Implement minimal behavior
  Files: `internal/...`
  Proof: targeted proof passes.

- [ ] T003 Run validation and record outcome
  Proof: `go test ./...`, `rtk make check`, or the smallest relevant command.
```

Rules:

- Use markdown checkboxes and stable task IDs.
- Name dependencies when task order matters.
- Name exact files when known, or narrow artifact/package surfaces when exact file choice is not knowable yet.
- Include proof expectations per task.
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
- active gate status such as clarification, adequacy, or implementation readiness.

It does not own final decisions, technical design, executable tasks, raw research, or validation transcripts.

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

- `waived` requires eligible direct-path, lean, or prototype rationale and scope.
- `not expected` requires trigger-based rationale when the artifact would otherwise be plausible.
- `conditional` means a later phase must decide the trigger.

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

Formal clarification asks only approval-changing questions. Ordinary downstream design detail should be recorded as a constraint, proof obligation, follow-up, or `defer_to_design`, not as a reason to inflate `spec.md`.

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

Run technical design review after the design bundle and any triggered conditional artifacts are written, but before `tasks.md` or implementation-readiness handoff is approved.

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
- return findings as advisory evidence for orchestrator reconciliation.

Technical design review is not a second design pass. If a finding requires a new decision, rewrite of the design bundle, or changed approval boundary, route it back to technical design or specification instead of solving it inside review.

For lean local with one `design/overview.md`, a local read-only orchestrator review is acceptable when no formal lane trigger exists, but the checkpoint and result must be recorded before `tasks.md`.

For full orchestrated triggered design, use at least one distinct read-only review lane unless a scoped-down local-review rationale is recorded. Add specialist lanes when independent API, data, security, reliability, observability, delivery, performance, or QA design risks are real. A design-integrator lane is the default fit when the hard part is coherence across specialist concerns.

Review gate status:

- `PASS`: planning may start from the reviewed design context.
- `CONCERNS`: planning may start only with named accepted design risks and proof obligations.
- `FAIL`: planning must not start; reopen technical design or specification.

Classify findings by strongest planning impact:

- `blocks_planning`: planning would invent or hide an important decision if it started now.
- `reopens_design`: the design bundle must change before review can pass.
- `reopens_spec`: the approved problem frame, invariant, scope, or contract must change.
- `accepted_risk_candidate`: the risk may be accepted only if the orchestrator names the reason and boundary.
- `proof_obligation`: planning may proceed only if the obligation is carried into `tasks.md`, `test-plan.md`, or `rollout.md`.
- `record_only`: useful context that does not affect planning entry.

Record the review result in the active workflow-control surface: `workflow-plan.md`, `workflow-plans/technical-design-review.md` when a dedicated review phase needs durable routing, or the lean-local artifact that owns the review checkpoint. The record must name the reviewed packet, reviewer or lane, scope, findings, orchestrator resolution, final gate status, and planning-input obligations. `CONCERNS` is valid only when every accepted risk and proof obligation is named for planning. Post-code discovery of a missing required technical design review reopens the earlier phase instead of creating a new review artifact after implementation starts.

## 11. Planning And Implementation Readiness

Planning turns approved decisions and required design context into `tasks.md`.

Direct path may use an inline plan.

Lean local and full orchestrated work use `tasks.md` for non-trivial implementation.

Planning must not invent missing design context. If exact tasking requires a missing decision, reopen the earlier concern.

Implementation readiness values:

- `PASS`: coding may start; no hidden architecture, ownership, contract, sequencing, rollout, or validation decision is needed for the next slice.
- `CONCERNS`: coding may start only with named accepted risks and explicit proof obligations.
- `FAIL`: coding must not start; route to the named earlier phase.
- `WAIVED`: allowed only for tiny direct-path or prototype scope with explicit rationale.

Readiness belongs in the planning handoff when planning artifacts exist. `tasks.md` may carry a short reference.

Planning also consumes the technical design review result whenever separate design depth was triggered. Missing or blocking review is a planning-entry failure, not a detail to infer inside `tasks.md`. When the review result is `CONCERNS`, planning must copy the accepted design risks and proof obligations into the implementation-readiness handoff and the relevant ledger or companion artifacts.

## 12. Coding, Review, Reconciliation, And Validation

Coding consumes the approved task handoff. It may create or edit code, tests, migrations, configs, generation inputs, and generated output required by the task ledger.

Post-code work may update only existing control or closeout surfaces:

- existing `workflow-plan.md`;
- existing `workflow-plans/technical-design-review.md` only to record a reopen or reconciled status when it existed before coding;
- existing `workflow-plans/review-phase-N.md` or `workflow-plans/validation-phase-N.md` when they were created before coding;
- existing `tasks.md` checkbox/progress state;
- `spec.md` `Validation` and `Outcome`.

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

If `workflow-plan.md` exists:

1. read `workflow-plan.md`;
2. read the current `workflow-plans/<phase>.md` if the task uses one;
3. read the files named in the `Next session context bundle`;
4. then read phase artifacts as needed.

If there is no `workflow-plan.md` because the task is direct or lean:

1. read `spec.md` when it exists;
2. read `tasks.md` when implementation or validation is next;
3. read optional `research/*.md` or `design/overview.md` only when named or needed.

Treat missing expected artifacts as incomplete unless an explicit waiver covers that exact artifact.

## 14. Final Chat Handoff

When a session reaches a boundary and a next session or reopen target exists, the final chat response must include a copy-pastable recommended next-session prompt derived from recorded artifacts.

Use no prompt when the workflow is honestly done.

The prompt is chat-only. It is not a workflow artifact and must not become a second source of truth.

## 15. Anti-Patterns

Avoid:

- treating full orchestrated as the default for every non-trivial task;
- using direct path for risky, ambiguous, public, data, security, money, reliability, concurrency, or rollout work;
- using lean local without `spec.md`, `tasks.md`, inline `Risk Challenge`, and proof;
- making `workflow-plans/<phase>.md` a second master plan, spec, design bundle, or task ledger;
- planning non-trivial implementation from `spec.md` alone when design context is triggered;
- approving non-trivial specs while formal challenge is required and unresolved;
- creating `test-plan.md`, `rollout.md`, split design files, or review/validation phase files for completeness;
- creating new process artifacts after coding starts;
- using subagents for broad ceremony rather than narrow unresolved questions;
- claiming done without fresh, scope-matched evidence.
