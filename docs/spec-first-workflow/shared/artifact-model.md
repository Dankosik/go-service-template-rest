# Workflow Artifact Model

Detailed shared companion for `docs/spec-first-workflow.md`. Read this when choosing execution shape, artifact depth, task-local artifact ownership, status vocabulary, or layout rules.

## Read When

- Choosing `direct path`, `lean local`, or `full orchestrated`.
- Deciding which task-local artifacts are expected, conditional, waived, or not expected.
- Checking ownership for `workflow-plan.md`, `workflow-plans/<phase>.md`, `spec.md`, `design/`, `tasks.md`, `test-plan.md`, or `rollout.md`.

## Inputs

- `AGENTS.md` for hard invariants and trigger rules.
- The accepted Phase 0 intake brief or direct-path clarity rationale, plus current task-local artifacts and current phase state.
- The router in `docs/spec-first-workflow.md` for phase-specific follow-up reading.

## Outputs

- Execution-shape decision and rationale.
- Artifact-depth decision, expected artifact list, and trigger or waiver rationale.
- Workflow-control ownership decision when multi-session routing is real.

## Stop Rule

Use this shared file to choose shape and artifact ownership, then continue in the one phase file that owns the current work. Do not use this file as a substitute for phase-specific gates or implementation authority.

## Default Decision Quality Recording

`AGENTS.md` owns hard decision-quality policy: target-state delivery, dependency/OSS due diligence, Pattern Fit Diligence, focused file and package responsibility, provider-contract verification, and legacy cleanup. This shared artifact model owns where those rules are represented in task-local artifacts.

Code-level pattern fit is a coding and review concern. Record it in `tasks.md`, implementation proof, or review findings only when it affects local code placement, simplification, cleanup, or verification; do not treat it as a separate architecture Pattern Fit Diligence gate.

Maintainable implementation shape is part of decision quality. When focused file, package, or responsibility placement matters, record the owner, rejected owner locations, cleanup consequence, and test owner in the artifact that carries implementation context.

When any of those hard rules is relevant, the owning artifact records:

- the selected decision and rejected alternatives when real;
- the evidence source or bounded assumption that supports the decision;
- the downstream artifact that must carry the consequence, such as `spec.md`, `design/`, `tasks.md`, `test-plan.md`, or `rollout.md`;
- the proof obligation, freshness or negative-proof rule, and reopen target when proof fails;
- the trigger or waiver rationale when an artifact is `not expected`, `conditional`, or `waived`.

A material decision is not complete when its owning artifact leaves `TBD`, open alternatives, or implementation-time product choices, unowned cleanup, or unstated proof paths. In that case, keep the artifact `draft` or `blocked` and reopen the phase that owns the missing decision.

## Execution Shapes

| Shape | Use When | Artifact Depth | Gate |
| --- | --- | --- | --- |
| `direct path` | Tiny, reversible, one surface, obvious validation, no protected-domain trigger. | Usually none; a short inline plan or chat note is enough. | Local first-read sanity check and fresh proof. |
| `lean local` | Bounded non-trivial single-domain work, stable ownership, limited research, and enough clarity to keep artifact depth lean. | `spec.md` plus `tasks.md` by default; optional preserved research, one `design/overview.md`, or `workflow-plan.md` only when triggered. Compact system/integration and Go code ownership answers may live together when concise and uncontested. | Subagent gate decision; inline `Risk Challenge`; mandatory specification review before design or planning; mandatory technical design review checkpoint when separate design depth is triggered; post-ledger task review/readiness gate. |
| `full orchestrated` | Cross-domain, ambiguous, hard-to-reverse, high-impact, long-running, user-requested agent-backed, or protected-domain work. | `workflow-plan.md`, triggered `workflow-plans/<phase>.md`, preserved research when useful, reviewed `spec.md`, triggered system/integration design, triggered Go code ownership design, mandatory technical design review record, `tasks.md`, optional companion artifacts. | Planned read-only fan-out and fan-in as the default decision basis, mandatory specification review, ordered design checkpoints when separate design depth is triggered, mandatory technical design review, post-ledger task review/readiness gate, and strict phase boundaries. |

Use `lean local` for bounded non-trivial single-domain work. This changes the amount of workflow ceremony, not the expected production readiness, expert coverage, or evidence quality of the chosen solution. `AGENTS.md` remains the hard authority for classification and escalation; this table explains the artifact implications.

### Escalation Triggers

`AGENTS.md` owns the escalation trigger list. When a trigger appears after work starts, this artifact model owns the recording behavior: mark the current artifact `blocked` or `conditional`, name the fuller shape or reopen target, and move to the fuller path instead of stretching the current shortcut.

### Default Session Boundary Policy

Phase arrows describe order, not a default license to collapse multiple phases into one chat session.

Default rule: one session owns one workflow phase, then stops. When the phase has a next phase or reopen target, update the relevant workflow state and end the final chat response with a copy-pastable `Recommended next-session prompt` derived from the recorded artifacts. The ready-to-paste prompt is rendered in chat only; workflow files keep the state needed to render it, not the full prompt text.

A broad user request such as "do the full workflow", "implement the PRD and architecture fully", or "create all necessary documents" advances the overall workflow, but it does not override the one-phase session boundary. Start with Phase 0 intake when the task is not yet an accepted brief; otherwise start with the next valid phase, finish that phase honestly, then stop with the next-session prompt.

This boundary rule applies to:

- Phase 0 intake when user answers or durable routing are needed;
- workflow planning;
- research and fan-in;
- specification and clarification-gate reconciliation;
- specification review and reconciliation;
- system/integration design;
- Go code ownership design;
- technical design review and reconciliation;
- task planning, task-ledger review, and implementation-readiness handoff;
- post-code review or reconciliation phases;
- validation and closeout;
- targeted reopen phases.

The normal exception is implementation from an approved `tasks.md` that has passed the post-ledger task-review/readiness gate. Once implementation readiness is `PASS`, eligible `CONCERNS`, or eligible `WAIVED`, the implementation Goal session may orchestrate the approved ledger items and the proof named by the ledger without stopping between task IDs. Non-trivial code-writing ledgers should use isolated Codex CLI worker bundles when execution mode is eligible or required; inline work is an explicit ledger choice or narrow fallback repair. After that point, workflow-control files are pre-code routing history unless the approved ledger explicitly names a separate review, validation, or reopen phase file as part of the work.

Direct path work has no durable phase boundary by default, so it may still complete inline with fresh proof.

## Artifact Model By Shape

### Direct Path

Direct path may skip durable workflow artifacts. It still needs:

- a bounded local understanding of the requested change from Phase 0 intake or an explicit clarity rationale;
- no protected-domain trigger;
- an explicit proof command or manual proof before claiming completion.

Do not create `workflow-plan.md`, `workflow-plans/`, `spec.md`, or `tasks.md` just for ceremony.

### Lean Local

Lean local is the default for bounded non-trivial single-domain work.

Expected artifacts:

- `spec.md`: compact decision record, review-ready after specification and downstream-ready only after specification review passes.
- `tasks.md`: executable task ledger and implementation handoff after task review passes.

Conditional artifacts:

- `research/*.md`: when evidence must survive resume, audit, or later synthesis.
- `design/overview.md`: when compact design answers are too dense for `spec.md` but do not need split design files.
- `design/system-integration.md`: when service behavior, contracts, external calls, queues, data/cache/source-of-truth, runtime sequence, failure behavior, validation, or rollout need a dedicated design artifact.
- `design/contracts/`: when changed REST resources, event payloads, generated contracts, client-visible status/error/idempotency/retry/async/freshness/compatibility semantics, or material internal interfaces need contract context that planning must preserve. It is design context, not runtime authority.
- `design/go-code-ownership.md`: when package/file ownership, focused responsibilities, dependency direction, local abstractions, cleanup/removal, or test ownership need a dedicated design artifact.
- `workflow-plan.md`: when multi-session state or reopen routing needs a durable control file.

Non-trivial lean-local work must run and record a specification review gate after `spec.md` is written and before compact tasking, separate `design/overview.md`, or planning starts. The review should use at least one read-only specification-review lane unless the task has an explicit direct/prototype waiver; when independent product, domain, API/data, architecture, security, reliability, delivery, or validation questions exist, split them into narrow lanes. If no workflow-control artifact exists, record the verdict and named obligations in `spec.md`; if workflow-control exists, record or link the review there.

If lean local uses a separate `design/overview.md`, run and record a technical design review checkpoint before writing or approving `tasks.md`. The checkpoint should consume a read-only lane summary when an independent design review question exists. A local read-only orchestrator review is allowed only with a recorded local-only rationale, must be distinct from the design-writing pass, and must record `PASS`, `CONCERNS`, or `FAIL`.

Not expected by default:

- `workflow-plans/<phase>.md`;
- split `design/system-integration.md`, `design/go-code-ownership.md`, `design/component-map.md`, `design/sequence.md`, and `design/ownership-map.md`;
- `test-plan.md`;
- `rollout.md`;
- review or validation phase files.

Lean local must not become an unstructured shortcut or a local-only decision path. It requires a recorded subagent gate decision, inline `Risk Challenge`, executable tasks, and fresh proof.

### Full Orchestrated

Full orchestrated keeps the existing full workflow, but all heavier artifacts are trigger-scoped.

Separate technical design is the exception to optional review routing: once `design/overview.md` or split `design/` is triggered, technical design review is mandatory before planning. The review record may live in `workflow-plan.md`, the active phase file, or `workflow-plans/technical-design-review.md` when the review needs durable routing, lanes, blockers, or a session boundary.

Specification review is not optional for non-trivial work. It runs after the specification checkpoint and before technical design, compact lean tasking, or planning. The review record may live in `workflow-plan.md`, the active phase file, `workflow-plans/specification-review.md` when the review needs durable routing, or the lean-local `spec.md` when no workflow-control artifact is used.

Typical layout:

```text
specs/<feature-id>/
  workflow-plan.md
  workflow-plans/
    workflow-planning.md        # only for a dedicated routing phase
    research.md                 # only for a dedicated research phase
    specification.md            # only when formal specification routing is needed
    specification-review.md     # required when non-trivial spec review routing needs durable state
    system-integration-design.md # only when dedicated system/integration design routing is needed
    go-code-ownership-design.md # only when dedicated Go code ownership design routing is needed
    technical-design-review.md  # required when separate technical design is triggered and review routing needs durable state
    planning.md                 # only when dedicated planning routing is needed
    review-phase-N.md           # only when planning names a multi-session review checkpoint
    validation-phase-N.md       # only when planning names a multi-session validation checkpoint
  research/
    <topic>.md                  # only when evidence needs to persist
  spec.md
  design/
    overview.md                 # entrypoint when design is triggered
    system-integration.md       # triggered service/system behavior design context
    go-code-ownership.md        # triggered Go package/file responsibility design context
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

## Workflow Control Artifacts

### `workflow-plan.md`

Use `workflow-plan.md` when cross-phase or multi-session state is real. It owns:

- execution shape and rationale;
- accepted intake brief summary or pointer when the task did not stay in one chat;
- current phase and phase status;
- session boundary state;
- next-session routing, meaning the next phase or reopen target and start point, not the full chat prompt;
- next-session context bundle, meaning artifact paths and one-line reasons needed to render the chat prompt, not a copy-paste prompt body;
- artifact status and trigger rationale;
- blockers, accepted assumptions, accepted risks, and reopen targets;
- active gate status such as clarification, adequacy, task-ledger review, or implementation readiness.
- subagent gate audit when the phase is non-trivial, formally challenged, review-bound, or agent-backed.

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
- subagent lane plan, fan-in status, and local-only or scoped-down rationale when relevant.

It must not replace `spec.md`, `design/`, or `tasks.md`.

### Status Vocabulary

Use status words proportionally: `approved`, `draft`, `missing`, `blocked`, `waived`, `not expected`, or `conditional`.

- `waived` requires eligible direct-path, lean, or explicitly user-requested prototype rationale and scope.
- `not expected` requires trigger-based rationale when the artifact would otherwise be plausible.
- `conditional` means a later phase must decide the trigger because current evidence is insufficient; do not use it to postpone a knowable production-readiness decision.
