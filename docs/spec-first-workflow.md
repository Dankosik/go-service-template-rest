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
| `lean local` | Bounded non-trivial single-domain work, stable ownership, limited research, and local reasoning can close the decision frontier. | `spec.md` plus `tasks.md` by default; optional preserved research, one `design/overview.md`, or `workflow-plan.md` only when triggered. | Inline `Risk Challenge`; no mandatory subagent. |
| `full orchestrated` | Cross-domain, ambiguous, hard-to-reverse, high-impact, long-running, user-requested agent-backed, or protected-domain work. | `workflow-plan.md`, triggered `workflow-plans/<phase>.md`, preserved research when useful, approved `spec.md`, triggered design bundle, `tasks.md`, optional companion artifacts. | Formal challenge/review lanes as triggered and strict phase boundaries. |

`lightweight local` is a compatibility alias for `lean local`. New artifacts should use `lean local`; older task bundles that say `lightweight local` remain valid.

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

Not expected by default:

- `workflow-plans/<phase>.md`;
- split `design/component-map.md`, `design/sequence.md`, and `design/ownership-map.md`;
- `test-plan.md`;
- `rollout.md`;
- review or validation phase files.

Lean local must not become an unstructured shortcut. It requires an inline `Risk Challenge`, executable tasks, and fresh proof.

### Full Orchestrated

Full orchestrated keeps the existing full workflow, but all heavier artifacts are trigger-scoped.

Typical layout:

```text
specs/<feature-id>/
  workflow-plan.md
  workflow-plans/
    workflow-planning.md        # only for a dedicated routing phase
    research.md                 # only for a dedicated research phase
    specification.md            # only when formal specification routing is needed
    technical-design.md         # only when dedicated design routing is needed
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
Historical task bundles that already use this full shape remain valid. Do not rewrite them during unrelated work.

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
- use read-only challenger output as questions for orchestrator reconciliation, not as authority;
- store final reconciled outcomes in `spec.md`;
- record gate status in `workflow-plan.md` and the active phase file when those files are used.

Formal clarification asks only approval-changing questions. Ordinary downstream design detail should be recorded as a constraint, proof obligation, follow-up, or `defer_to_design`, not as a reason to inflate `spec.md`.

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

## 10. Planning And Implementation Readiness

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

## 11. Coding, Review, Reconciliation, And Validation

Coding consumes the approved task handoff. It may create or edit code, tests, migrations, configs, generation inputs, and generated output required by the task ledger.

Post-code work may update only existing control or closeout surfaces:

- existing `workflow-plan.md`;
- existing `workflow-plans/review-phase-N.md` or `workflow-plans/validation-phase-N.md` when they were created before coding;
- existing `tasks.md` checkbox/progress state;
- `spec.md` `Validation` and `Outcome`.

Do not create new workflow/process artifacts after implementation starts. Reopen the earlier phase that owns the missing artifact instead.

Review is read-only and risk-driven. Review findings are advisory until the orchestrator reconciles them.

Validation uses fresh evidence. A closeout claim is valid only when the commands or manual proof actually cover that claim.

## 12. Subagents

Subagents are triggered by unresolved owning questions, not by phase ceremony.

Use `docs/subagent-contract.md` and `docs/subagent-brief-template.md` for reusable brief shape.

Every lane needs:

- goal and exact question;
- scope and constraints;
- expected output;
- evidence requirement;
- skill name or `no-skill`;
- read-only boundary.

A lane uses at most one skill. If the selected skill defines a stricter deliverable, follow it. Otherwise use the shared envelope from `docs/subagent-contract.md`.

## 13. Resume Order

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
