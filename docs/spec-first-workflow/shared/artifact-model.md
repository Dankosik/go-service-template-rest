# Workflow Artifact Model

Detailed shared companion for `docs/spec-first-workflow.md`. Read this after `AGENTS.md` classifies execution shape, when recording artifact depth, task-local ownership, typed state, reclassification, status, or layout rules.

## Read When

- Projecting artifact consequences from `direct_path`, `lean_local`, or `full_orchestrated`.
- Deciding which task-local artifacts are `expected`, `conditional`, or `not_expected`, plus any separate waiver disposition.
- Checking ownership for `workflow-plan.md`, `workflow-plans/<phase>.md`, `spec.md`, `design/`, `tasks.md`, `test-plan.md`, or `rollout.md`.

## Inputs

- `AGENTS.md` for hard invariants and trigger rules.
- The accepted Phase 0 intake brief or explicit clear-input rationale, plus the `AGENTS.md` trigger audit, current task-local artifacts, and current phase state.
- The router in `docs/spec-first-workflow.md` for phase-specific follow-up reading.

## Outputs

- Artifact-depth consequences for the `SHAPE-*` decision made under `AGENTS.md`.
- Typed workflow state, expected artifact list, and trigger or waiver rationale.
- Reclassification, resume, status-source, and workflow-control ownership records when routing state is real.

## Stop Rule

Use `AGENTS.md` to choose shape, then use this file to record artifact and state consequences before continuing in the one phase file that owns the current work. Do not use this file as a second classifier, a substitute for phase-specific gates, or implementation authority.

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

`AGENTS.md` is the only owner of the ordered classifier, positive predicates, protected triggers, agent-request interpretation, direct writer boundary, and adequacy predicate. This table projects artifact consequences from the matched `SHAPE-*` rule; it must not grow independent admission predicates.

<!-- workflow-rule-table:start -->
| Rule ID | `execution_shape` | Artifact depth | Gate and actor consequences |
| --- | --- | --- | --- |
| `ARTIFACT-DIRECT` | `direct_path` | Usually no workflow files; a short inline plan or chat note is enough. | Current orchestrator may write the tiny patch before any ledger exists while all `DIRECT-*` evidence remains current; first-read sanity check and fresh proof are required. |
| `ARTIFACT-LEAN` | `lean_local` | `spec.md` plus `tasks.md` by default; preserved research, `design/overview.md`, `test-plan.md`, or `workflow-plan.md` only when triggered. | Recorded subagent-gate decision, inline `Risk Challenge`, mandatory specification review, triggered design/test-design, task review/readiness, then ledger-backed worker execution when code writing is required. |
| `ARTIFACT-FULL` | `full_orchestrated` | `workflow-plan.md`, only triggered `workflow-plans/<phase>.md`, preserved research when useful, reviewed `spec.md`, triggered design/test-design, and `tasks.md`. | Planned read-only fan-out/fan-in, formal gates selected by canonical triggers, strict phase boundaries, task review/readiness, then ledger-backed worker execution when code writing is required. |
<!-- workflow-rule-table:end -->

`lean_local` remains the normal result for bounded non-trivial single-domain work only when `SHAPE-LEAN` actually matches. Workflow shape changes ceremony and durable state, not the expected production readiness, expert coverage, or evidence quality.

New routing records use this typed artifact-depth baseline; later trigger evidence may resolve `conditional` to `expected` or `not_expected` through a new routing revision.

<!-- workflow-rule-table:start -->
| Rule ID | Shape | `workflow-plan.md` | `workflow-plans/<phase>.md` | `spec.md` | `tasks.md` | Research/design/test/rollout companions |
| --- | --- | --- | --- | --- | --- | --- |
| `DEPTH-DIRECT` | `direct_path` | `not_expected` | `not_expected` | `not_expected` | `not_expected` | `not_expected`; proof remains inline/current-session |
| `DEPTH-LEAN` | `lean_local` | `conditional` on durable resume/routing | `conditional` on `ROUTING-PHASE-CONTROL` | `expected` | `expected` | `conditional` on their owning triggers |
| `DEPTH-FULL` | `full_orchestrated` | `expected` | `conditional` on `ROUTING-PHASE-CONTROL` | `expected` | `expected` | independently `expected`, `conditional`, or `not_expected` from real triggers; full shape does not make every companion mandatory |
<!-- workflow-rule-table:end -->

### Escalation Triggers

`AGENTS.md` owns the `FULL-*` trigger list. When a trigger appears after work starts, use the guarded reclassification transaction below; do not merely relabel one artifact or stretch the current shortcut.

### Macro-Phase Session Boundary Policy

Phase arrows describe dependency order. They do not make every checkpoint a user-started session.

Default rule: one user-started root session owns one macro phase and stops only before the next macro phase. The owning orchestrator runs every required independent review, repairs its own authoritative artifacts, and obtains a fresh re-review verdict before closing. Internal review or repair never emits a user handoff prompt.

Macro-phase boundaries are:

- intake/workflow planning;
- research and fan-in;
- specification, including clarification, specification review, repair, and fresh re-review;
- technical design, including system/integration design, Go code ownership design, technical design review, repair, and fresh re-review;
- test design, including independent QA review, repair, and fresh re-review;
- planning, including task-ledger review/readiness, repair, and fresh re-review;
- implementation/validation/closeout, including worker patching, post-code review, repair, fresh re-review, validation, and closeout;
- a targeted reopen of one of those owners when the required change cannot be completed inside the active macro phase.

Cross-macro-phase collapse remains prohibited unless the user explicitly changes that boundary. A broad request such as "do the full workflow" does not silently cross from specification into technical design or from planning into implementation. At a real boundary, update workflow state and render one copy-pastable `Recommended next-session prompt` from recorded artifacts.

Inside a macro phase, a review `FAIL` or actionable `CONCERNS` is not a session boundary. The root repairs every in-scope finding owned by the phase, invalidates the prior verdict, and launches a fresh read-only review thread or independent Codex process against the changed revision. Stop only on a current eligible verdict or an honest blocker that needs user policy, external authority, unavailable required evidence/tooling, or an earlier owner. Repeated review without changed evidence or repair is not progress; the shared subagent contract owns the stagnation stop.

Implementation from an approved `tasks.md` that has passed task-review/readiness remains one Goal session across approved ledger items and named proof. Code-writing ledgers use isolated Codex CLI worker bundles when execution mode is required; orchestrator-authored inline implementation is not allowed. Post-code review and validation are internal implementation checkpoints even when an approved ledger names durable review or validation carriers.

Direct path no-code work may complete with fresh proof. Direct-path code writing may be authored by the current orchestrator only before a task ledger exists and only while every `DIRECT-*` predicate and the `SHAPE-DIRECT` route remain current. Once an approved ledger exists, code-writing implementation remains isolated-worker-only.

## Artifact Model By Shape

### Direct Path

Direct path may skip durable workflow artifacts. It still needs:

- a bounded local understanding of the requested change from Phase 0 intake or an explicit clarity rationale;
- every `FULL-*` trigger is false with current evidence;
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

Non-trivial lean-local work must run and record a specification review gate after `spec.md` is written and before compact tasking, separate `design/overview.md`, or planning starts. The review should use at least one read-only specification-review lane unless the subagent gate has an eligible prototype-scoped waiver; that waiver changes lane execution only and never removes the mandatory review verdict. When independent product, domain, API/data, architecture, security, reliability, delivery, or validation questions exist, split them into narrow lanes. If no workflow-control artifact exists, record the verdict and named obligations in `spec.md`; if workflow-control exists, record or link the review there.

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

Separate technical design is the exception to optional review routing: once `design/overview.md` or split `design/` is triggered, technical design review is mandatory before planning. The review record may live in `workflow-plan.md`, the active phase file, or `workflow-plans/technical-design-review.md` only when `ROUTING-PHASE-CONTROL` is satisfied by durable local orchestration such as multi-lane routing, fan-in, formal challenge, a multi-session stop rule, or a named review/validation checkpoint.

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

- `execution_shape`, matched `SHAPE-*` rule, normalized trigger evidence, `agent_request`, and rationale recorded from `AGENTS.md`;
- `routing_scope`, `routing_revision`, and `record_validity` for the master route;
- accepted intake brief summary or pointer when the task did not stay in one chat;
- current phase and typed `phase_state`;
- typed `session_boundary` and `handoff_readiness`;
- next-session routing, meaning the next macro phase or owning reopen target and start point, never an internal review/repair checkpoint and never the full chat prompt;
- next-session context bundle, meaning artifact paths and one-line reasons needed to render the chat prompt, not a copy-paste prompt body;
- per-artifact `artifact_expectation`, `artifact_state`, `record_validity`, separate waiver disposition, and trigger rationale;
- blockers, accepted assumptions, accepted risks, and reopen targets;
- typed procedural gate and review-verdict state such as clarification, adequacy, task-ledger review, or implementation readiness;
- subagent gate audit when the phase is non-trivial, formally challenged, review-bound, or `agent_request=substantive`.

It does not own final decisions, technical design, executable tasks, raw research, or validation transcripts.

Once `tasks.md` is approved, `workflow-plan.md` no longer owns implementation progress, task completion, or closeout state. It may remain useful historical context, but agents must not update it during implementation or closeout. Pre-created review or validation phase files may be updated only when the approved ledger explicitly names them as required artifacts.

### `workflow-plans/<phase>.md`

Create a phase-local file only when the phase needs durable local orchestration: multi-lane routing, fan-in, formal challenge status, a multi-session stop rule, or named review/validation checkpoints.

It owns:

- local lanes or order/parallelism;
- phase-local completion marker, `phase_state`, routing identity, and `record_validity`;
- local stop rule;
- next action;
- local blockers;
- typed gate/verdict state and handoff readiness for that phase;
- subagent lane plan, fan-in status, and local-only or scoped-down rationale when relevant.

It must not replace `spec.md`, `design/`, or `tasks.md`.

### Status Vocabulary

New workflow state uses orthogonal typed fields. Free-text display phrases are derived views, not authority.

<!-- workflow-rule-table:start -->
| Rule ID | Namespace | Canonical values | Meaning |
| --- | --- | --- | --- |
| `STATE-EXECUTION-SHAPE` | `execution_shape` | `direct_path`, `lean_local`, `full_orchestrated` | The `SHAPE-*` result selected under `AGENTS.md`. |
| `STATE-ARTIFACT-EXPECTATION` | `artifact_expectation` | `expected`, `conditional`, `not_expected` | Whether an artifact should exist for this route. |
| `STATE-ARTIFACT-LIFECYCLE` | `artifact_state` | `absent`, `draft`, `review_ready`, `approved`, `complete`, `blocked` | Artifact lifecycle independent of expectation and freshness. |
| `STATE-RECORD-VALIDITY` | `record_validity` | `current`, `stale`, `superseded` | Whether an artifact, gate, verdict, handoff, or envelope matches the active route. |
| `STATE-PHASE` | `phase_state` | `not_started`, `active`, `complete`, `blocked`, `reopened` | User-started macro-phase progress; internal checkpoint progress belongs in artifact, procedural-gate, review-verdict, and subagent-gate fields. |
| `STATE-PROCEDURAL-GATE` | `procedural_gate_state` | `pending`, `complete`, `blocked`, `waived`, `not_expected` | Whether a procedure ran or has an eligible disposition. |
| `STATE-REVIEW-VERDICT` | `review_verdict` | `pending`, `PASS`, `CONCERNS`, `FAIL`, `WAIVED` | Independent review/readiness result. |
| `STATE-RISK-CHALLENGE` | `risk_challenge_outcome` | `PASS`, `CONCERNS`, `RECLASSIFY_FULL` | Inline lean specification challenge result. It never occupies `procedural_gate_state` or `review_verdict`; `RECLASSIFY_FULL` blocks lean handoff for orchestrator-owned guarded reclassification. |
| `STATE-SUBAGENT-GATE` | `subagent_gate` | `complete`, `scoped_down`, `local_only`, `waived`, `not_expected`, `blocked` | Lane/fan-in disposition. |
| `STATE-WAIVER` | `waiver_disposition` | `none`, `waived` | A separate scoped waiver with eligibility, rationale, evidence, and reopen trigger. |
| `STATE-SESSION-BOUNDARY` | `session_boundary` | `open`, `reached` | Whether the owning phase may continue in the current session. |
| `STATE-HANDOFF` | `handoff_readiness` | `not_ready`, `ready`, `blocked` | Whether the recorded next session can start. |
| `STATE-ROUTING-SCOPE` | `routing_scope` | `current_session`, `durable` | Whether the routing record may survive the current session. |
| `STATE-ROUTING-REVISION` | `routing_revision` | positive integer within its routing scope | Freshness key; identity is `(routing_scope, routing_revision)`. |
<!-- workflow-rule-table:end -->

Composition rules:

<!-- workflow-rule-table:start -->
| Rule ID | Composition rule |
| --- | --- |
| `STATE-COMPOSE-FRESHNESS` | Lifecycle or verdict and freshness compose. `approved + stale`, `PASS + stale`, and `complete + stale` are valid history, but `stale` or `superseded` never authorizes readiness or execution. |
| `STATE-COMPOSE-EXPECTED` | Any non-`absent` artifact lifecycle requires `artifact_expectation=expected`. |
| `STATE-COMPOSE-NOT-EXPECTED` | `artifact_expectation=not_expected` requires `artifact_state=absent` and `waiver_disposition=none`. |
| `STATE-COMPOSE-CONDITIONAL` | `artifact_expectation=conditional` requires `artifact_state=absent` and `waiver_disposition=none` until the trigger is resolved. |
| `STATE-COMPOSE-WAIVER` | A waived artifact records `artifact_expectation=expected + artifact_state=absent + waiver_disposition=waived` plus eligibility, rationale, evidence, and reopen trigger. |
| `STATE-COMPOSE-REVISION` | Every procedural gate state, review verdict, adequacy result, and handoff records the routing identity it observed; a mismatch sets `record_validity=stale`. |
| `STATE-COMPOSE-DISPLAY` | `missing, expected later` displays `expected + absent`; `conditional, trigger unknown` displays `conditional + absent`. Display phrases are not stored authority. |
| `STATE-COMPOSE-FAIL-CLOSED` | New writers emit only canonical values. Ambiguous or conflicting values become `status unclear`; file presence never implies shape, approval, or readiness. |
<!-- workflow-rule-table:end -->

### Closed Legacy Read Mapping

Completed historical bundles remain readable but are not rewritten. A reader may normalize only by trimming Markdown backticks, surrounding whitespace, and one trailing period, then apply this closed mapping. Anything else is returned verbatim as `legacy_unmapped` and cannot authorize a gate, handoff, or implementation.

<!-- workflow-rule-table:start -->
| Rule ID | Legacy field/value | Typed projection |
| --- | --- | --- |
| `LEGACY-SHAPE-DIRECT` | Shape `direct path` or `direct_path` | `execution_shape=direct_path` |
| `LEGACY-SHAPE-LEAN` | Shape `lean local`, `lean_local`, `lightweight local`, or `lightweight_local` | `execution_shape=lean_local`; lightweight forms are read-only aliases only |
| `LEGACY-SHAPE-FULL` | Shape `full orchestrated` or `full_orchestrated` | `execution_shape=full_orchestrated` |
| `LEGACY-PHASE-NOT-STARTED` | Phase `pending` or `not_started` | `phase_state=not_started` |
| `LEGACY-PHASE-ACTIVE` | Phase `active`, `in_progress`, or `in progress` | `phase_state=active` |
| `LEGACY-PHASE-COMPLETE` | Phase `complete`, `completed`, or `done` | `phase_state=complete` |
| `LEGACY-PHASE-BLOCKED-REOPENED` | Phase `blocked` or `reopened` | Matching canonical `phase_state` |
| `LEGACY-ARTIFACT-LIFECYCLE` | Artifact `approved`, `draft`, `blocked`, `complete`, or `completed` | `artifact_expectation=expected`, matching lifecycle (`completed` -> `complete`), `record_validity=current` |
| `LEGACY-ARTIFACT-MISSING` | Artifact `missing`, `missing, expected later`, or `missing, expected next` | `artifact_expectation=expected`, `artifact_state=absent`, `record_validity=current` |
| `LEGACY-ARTIFACT-COMPLETE-EVIDENCE` | Artifact `present, complete evidence` | `artifact_expectation=expected`, `artifact_state=complete`, `record_validity=current` |
| `LEGACY-ARTIFACT-CONDITIONAL` | Artifact `conditional` or `conditional, trigger unknown` | `artifact_expectation=conditional`, `artifact_state=absent`, `record_validity=current` |
| `LEGACY-ARTIFACT-NOT-EXPECTED` | Artifact `not expected` | `artifact_expectation=not_expected`, `artifact_state=absent`, `record_validity=current` |
| `LEGACY-ARTIFACT-WAIVED` | Artifact `waived` | `artifact_expectation=expected`, `artifact_state=absent`, `waiver_disposition=waived`, `record_validity=current` |
| `LEGACY-GATE` | Gate `pending`, `complete`, `blocked`, `waived`, or `not_expected` | Matching `procedural_gate_state` |
| `LEGACY-VERDICT` | Verdict `PASS`, `CONCERNS`, `FAIL`, or `WAIVED` | Matching `review_verdict` |
| `LEGACY-SESSION` | `Session boundary reached: yes|no` | `session_boundary=reached|open` |
| `LEGACY-HANDOFF` | `Ready for next session: yes|no` | `handoff_readiness=ready|not_ready` |
| `LEGACY-UNMAPPED` | Any unlisted or descriptive value, including `full orchestrated ...` or `complete for ...` | `legacy_unmapped`; display only, never readiness authority |
<!-- workflow-rule-table:end -->

### Guarded Reclassification Transaction

Every initial classification, escalation, downgrade, same-shape refresh, or reopen is one atomic routing transaction. It records transition kind; prior and target routing identity, shape, and phase; trigger evidence and actor; artifact expectation/state delta; preserved, stale, superseded, or newly required artifacts; gate/verdict invalidations or preserved results with rationale; session boundary; next route; proof obligation; and reopen target.

<!-- workflow-rule-table:start -->
| Rule ID | Transition rule |
| --- | --- |
| `TRANS-DIRECT-INITIAL` | Artifactless direct starts at `routing_scope=current_session, routing_revision=1` in the current orchestrator's direct envelope; current-session changes increment that revision. |
| `TRANS-DURABLE-INITIAL` | A route that starts durable begins at `routing_scope=durable, routing_revision=1`. |
| `TRANS-DIRECT-ELIGIBILITY-LOSS` | Any `DIRECT-*` predicate becoming false or approval-relevant unknown immediately stops direct eligibility and further direct writes before reclassification. |
| `TRANS-DIRECT-ESCALATE` | Direct to lean/full materializes `routing_scope=durable, routing_revision=1`, records the exact source current-session identity, and makes the prior envelope and local edits stale/unapproved inputs. |
| `TRANS-UPWARD` | `direct -> lean/full` and `lean -> full` occur as soon as a higher floor becomes true or approval-relevant unknown. |
| `TRANS-DOWNGRADE` | Downgrade requires evidence that every trigger responsible for the prior floor is false or outside accepted scope; missing files, elapsed time, or perceived simplicity are not evidence. |
| `TRANS-REFRESH` | Changed assumptions increment the revision and mark every dependent artifact, gate, verdict, adequacy result, and handoff stale; procedural gates reset to `pending` or `blocked`, handoff becomes `blocked`, and unaffected evidence survives only with an explicit dependency rationale. |
| `TRANS-DURABLE-REVISION` | Once durable state exists, refresh, escalation, downgrade, and reopen increment the durable revision; the route never returns to artifactless state by deleting control files. |
| `TRANS-PARTIAL-CONFLICT` | Master and active phase control must share the same routing identity before handoff; partial updates or conflicts block phase and handoff. |
| `TRANS-DURABLE-HISTORY` | Durable artifacts made unnecessary by downgrade remain `superseded` history; they are not deleted to simulate artifactlessness. |
| `TRANS-POST-LEDGER-REOPEN` | After ledger approval, record the blocker and reopen target in `tasks.md` first, stale/block implementation readiness, stop, then create a new owning-phase durable revision. `tasks.md` remains execution-state source but cannot authorize execution until fresh task review/readiness matches the new route. |
<!-- workflow-rule-table:end -->

No status helper, skill, challenger, or subagent may perform this transaction. The orchestrator is the only actor that classifies or reclassifies.

### Independent Routing Decisions

Research expectation, next phase, phase collapse, and phase-control depth are separate fields; one must not be inferred from another.

<!-- workflow-rule-table:start -->
| Rule ID | Routing rule |
| --- | --- |
| `ROUTING-RESEARCH` | Record `research_expectation=expected|conditional|not_expected` independently of shape and next phase. |
| `ROUTING-RESEARCH-SKIP` | `research_expectation=not_expected` normally ends workflow planning and routes a new session to specification; it never authorizes same-session specification. |
| `ROUTING-NEXT-PHASE` | Record exactly one next macro phase or owning reopen target. Internal review and repair checkpoints do not occupy `next_phase`. |
| `ROUTING-NO-COLLAPSE` | Record `same_session_collapse=internal_gates_required`: specification review, technical-design checkpoints/review, test-design review, task-review/readiness, and post-code review/validation continue inside their owning macro phase; crossing into the next macro phase remains prohibited. |
| `ROUTING-PHASE-CONTROL` | `phase_control=required` only for durable local orchestration: multi-lane routing, fan-in, formal challenge, multi-session stop rule, or named review/validation checkpoint; otherwise `not_required`. |
| `ROUTING-GATE-NOT-FILE` | A mandatory gate alone does not require `workflow-plans/<phase>.md`; record specification review in master or lean spec when durable review routing is absent. |
| `ROUTING-DEDICATED-PLANNING` | Dedicated workflow planning runs only for durable cross-phase or multi-session routing; it consumes, records, and challenges the existing `SHAPE-*` decision and never classifies independently. |
<!-- workflow-rule-table:end -->

The marked Markdown tables in `AGENTS.md` and this artifact model are normative. Deterministic checker code and fixtures are proof-only consumers: they may parse rule IDs and evaluate examples, but must not introduce a machine-readable policy owner or redefine a rule.

`make workflow-routing-check` is the network-free merge/release proof for these tables. Skill `evals/evals.json` manifests are behavioral coverage assets for human or future runner use; they are not CI-executed evidence unless a repository-owned deterministic, credential-safe runner is added and invoked explicitly.

### Workflow Status Source Precedence

Status reporting is read-only and uses the first current, non-conflicting source below. It never creates, repairs, approves, waives, classifies, or reclassifies state.

<!-- workflow-rule-table:start -->
| Rule ID | Precedence | Source and boundary |
| --- | ---: | --- |
| `STATUS-TASKS` | 1 | `tasks.md` owns implementation/closeout state and blocker/reopen pointers after ledger approval; it authorizes execution only when its verdict/readiness is current for the active durable route. |
| `STATUS-DURABLE-CONTROL` | 2 | Durable master and active phase control own pre-code routing when their routing identities agree. |
| `STATUS-PHASE-ARTIFACTS` | 3 | Approved phase artifacts supply their own current gate/decision state according to the artifact-first chain. |
| `STATUS-DIRECT-ENVELOPE` | 4 | An explicit current-session `direct_state_envelope` is eligible only when no durable task state exists. |
| `STATUS-UNSUPPORTED` | 5 | Without an identifiable task path or eligible direct envelope, return `unsupported: no durable task state`; never infer from task size, chat memory, recency, or file absence. |
<!-- workflow-rule-table:end -->

The `direct_state_envelope` is created and attested only by the current orchestrator after re-evaluating `AGENTS.md` from the accepted brief and current source reads. It contains `provenance=orchestrator_current_session`, current-session routing identity, accepted framing or clarity rationale, full/direct/lean trigger audit, selected actor, proof obligation/result, session state, and reopen seam. User-quoted, prior-session, provenance-unknown, or revision-mismatched envelopes are unsupported. A valid envelope remains observable through the current session's final status/closeout response after `session_boundary=reached`, then expires; it is never a resume source.

Every report includes execution shape and matched rule/evidence; routing identity; adequacy required/result/evidence; phase, session, artifact, gate, verdict/readiness, freshness/conflict state; implementation eligibility; allowed writes; and exact next action.

### Canonical And Mirror Availability

Canonical source availability is independent of mirror state. Sync scripts own checked-in consumer/target/requiredness registries for generation only; they do not own workflow policy. Every current target is `optional` until a reviewed registry change says otherwise.

<!-- workflow-rule-table:start -->
| Rule ID | Availability state | Result |
| --- | --- | --- |
| `MIRROR-CANONICAL-AVAILABLE` | `canonical_available=true|false` | Canonical absence always fails before mirror conclusions. |
| `MIRROR-RENDER-FAILED` | `mirror_render_failed` | Temporary rendering failed; no target state may be inferred and the aggregate fails. |
| `MIRROR-COMPARE-FAILED` | `mirror_compare_failed` | Target comparison could not complete; do not relabel the operational error as in sync or stale. |
| `MIRROR-OPTIONAL-ABSENT` | `mirror_optional_absent` | Pass only after successful temporary render; never report absent as in sync. |
| `MIRROR-PRESENT-IN-SYNC` | `mirror_present_in_sync` | Present target matches rendered source-managed content under the selected comparison policy; strict mode additionally forbids target-only files. |
| `MIRROR-PRESENT-STALE` | `mirror_present_stale` | Present target differs; fail. |
| `MIRROR-REQUIRED-MISSING` | `mirror_required_missing` | Named required consumer target is absent; fail. |
<!-- workflow-rule-table:end -->

Check mode always renders expected output into a temporary directory before inspecting target presence, reports every target independently, applies explicit strict/non-strict target-only-file behavior, treats render and comparison failures as closed operational states, aggregates without hiding sibling failures, and leaves tracked and untracked repository state unchanged. The aggregate passes only when canonical sources and temporary render succeed and each target is either present/in-sync or optional/absent. Repository checks prove configured targets and deterministic rendering, not that every external runtime loaded a mirror; generated mirrors remain uncommitted by default.
