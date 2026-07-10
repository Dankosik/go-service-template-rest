# Execution-Shape Routing Hardening

Mode: full orchestrated
Status: review_ready
Subagent gate: complete; five formal clarification lenses reconciled in `workflow-plans/specification.md`

## Context

The repository currently has eleven coupled routing defects (`B01-F01`-`B01-F11`) across execution authority, shape selection, workflow state, reclassification, adequacy, status reporting, enforcement, and generated mirror behavior. The accepted outcome is one deterministic replacement contract that preserves the light direct path, keeps bounded work lean, escalates protected work reliably, and can be resumed and regression-tested without interpreting contradictory prose.

The decision evidence is preserved in:

- `research/synthesis.md` for the complete finding map and cross-finding reconciliation;
- `research/shape-and-execution-authority.md` for `F01`, `F02`, and `F07`;
- `research/status-reclassification-and-resume.md` for `F03`, `F04`, `F05`, `F08`, and `F10`;
- `research/adequacy-enforcement-and-mirrors.md` for `F06`, `F09`, and `F11`;
- `research/pattern-fit.md` for workflow/state/status pattern comparison.

## Scope / Non-goals

In scope:

- one authoritative intake-to-shape-to-workflow-planning route;
- executable direct-path code-writing authority;
- shape precedence and protected-trigger behavior;
- capability authorization versus substantive agent-backed intent;
- typed artifact, phase, gate, session, waiver, and freshness state;
- atomic escalation, downgrade, refresh, and reopen behavior;
- research expectation, session collapse, and phase-control triggers;
- independent adequacy falsification and its canonical trigger;
- workflow-status behavior for durable and artifactless work;
- semantic guardrails, deterministic CI proof, behavior eval coverage, and mirror-availability semantics;
- removal or normalization of stale lower-precedence wording across canonical docs, skills, references, evals, agents, scripts, Make/CI, and generated mirrors.

Out of scope:

- Go service runtime behavior, REST/OpenAPI, persisted service data, deployment, or production rollout;
- rewriting completed historical task bundles under `specs/`;
- importing a runtime workflow/state-machine engine or adding a dependency;
- committing generated runtime mirrors that are intentionally local and ignored;
- claiming external agent-runtime discovery behavior that repository evidence cannot prove;
- changing unrelated workflow phases except where their shared routing/status interface must consume this contract.

## Constraints

- `AGENTS.md` remains the compact hard authority; detailed mechanics have exactly one lower-precedence owner and other surfaces link or consume them.
- The orchestrator owns classification, reclassification, reconciliation, and final decisions. Subagents and adequacy challengers remain read-only and advisory.
- Non-trivial ledger-backed code writing remains isolated-worker-only after task-review/readiness. The direct-path exception defined below does not weaken that rule.
- New workflow state uses canonical typed fields. Completed historical artifacts remain readable but are not rewritten into current compliance.
- Unknown or conflicting routing evidence fails closed; status helpers and challengers never repair or approve state.
- The target state is one coherent replacement, not a compatibility bridge followed by later hardening.
- `design/overview.md`, split design, technical design review, and rollout remain not expected unless specification review exposes a real architecture or dependency decision.

## Behavior / Contract Delta

ADDED:

- a deterministic shape decision table with explicit dominance and evidence;
- a typed agent-request intent that separates capability from substantive workflow intent;
- orthogonal workflow-state namespaces and a routing revision;
- one guarded reclassification transaction and stale-state rules;
- independent research, next-phase, same-session, and phase-file decisions;
- shape-falsifying adequacy behavior and one closed formal-challenge predicate;
- an explicit current-session envelope for observable artifactless direct work;
- deterministic routing fixtures in Make/CI and explicit mirror availability states.

MODIFIED:

- direct-path code writing changes from an impossible worker-only/no-ledger intersection to one narrow orchestrator-owned route;
- dedicated workflow planning changes from a competing classifier to a durable recording and challenge checkpoint;
- flat and compound status phrases become derived views of typed fields;
- workflow-status reports shape, evidence, adequacy, freshness, and the artifactless observation boundary;
- mirror checks prove deterministic generation even when mirrors are absent instead of equating absence with in-sync content.

REMOVED FROM NEW NORMATIVE OUTPUT:

- undefined `high-risk`, `complex workflow-control`, and bare `agent-backed` adequacy predicates;
- `lightweight local` as a newly emitted shape name;
- any implication that `research: not expected` permits same-session specification;
- any implication that mandatory specification review always requires `workflow-plans/specification-review.md`;
- regex/file-presence checks being described as semantic behavior proof.

UNCHANGED:

- Phase 0 precedes shape selection;
- direct path remains the routine no-bundle route;
- lean local remains the default for bounded non-trivial single-domain work;
- every protected trigger establishes a full-orchestrated floor;
- formal specification review and task-review/readiness remain mandatory for non-trivial artifact-backed work;
- subagents remain read-only, and the orchestrator remains final authority;
- mirrors remain generated and ignored unless an explicit consumer requirement changes that policy.

## Decisions

### D1 — Direct-Path Code Writing Has A Narrow Orchestrator Exception (`B01-F01`)

- Selected: when every direct-path predicate is proven and no protected trigger is true or unresolved, the orchestrator may author the tiny code change in the active workspace and run the explicit fresh proof. No `tasks.md`, worker brief, or workflow bundle is created for ceremony.
- Boundary: this exception applies only before a task ledger exists and only while the work remains tiny, reversible, single-surface, and obviously verifiable. It never applies to ledger-backed implementation.
- Loss of eligibility: discovery of a second material surface, non-obvious proof, protected trigger, or ownership ambiguity stops further direct writes and invokes D6 before work continues. Existing direct edits are unapproved working state until the new route accepts or replaces them.
- Unchanged worker rule: after an approved `tasks.md` marks code-writing implementation ready, every code-writing bundle remains delegated to isolated CLI workers; the orchestrator does not author those patches.
- Evidence: `research/shape-and-execution-authority.md` and the current contradiction between `AGENTS.md:45-47` and `docs/spec-first-workflow/shared/artifact-model.md:82-96`.
- Rejected: a new artifactless direct-worker protocol adds unnecessary launch/resume state; removing code-writing direct path makes tiny safe work non-lean by construction; choosing a writer ad hoc preserves the defect.
- Downstream consequence: direct actor, eligibility loss, and ledger boundary require deterministic fixtures and workflow-status coverage.

### D2 — Shape Classification Has One Actor And One Detailed Owner (`B01-F02`)

- Hard policy owner: `AGENTS.md` owns the complete shape decision algorithm in D3, including trigger dominance, unknown-evidence behavior, direct/lean admission, fallback-to-full behavior, the direct execution boundary, and the smallest-correct-shape invariant.
- Detailed mechanics owner: `docs/spec-first-workflow/shared/artifact-model.md` owns typed state, artifact implications, transition recording, and resume/status mechanics. It references the `AGENTS.md` routing rule IDs and must not restate an independently editable shape table.
- Decision actor: the orchestrator classifies after an accepted Phase 0 brief or explicit clear-input rationale and before subagent calls or workflow artifact creation.
- Dedicated workflow-planning procedure: `.agents/skills/workflow-planning-session` is invoked only when durable cross-phase/multi-session routing is triggered. It consumes, records, and challenges the classification; it is not a second policy owner or independent classifier.
- Falsification: if workflow planning or adequacy disproves the recorded shape, the orchestrator applies D6 and records the new route.
- Evidence: `research/shape-and-execution-authority.md` and `research/synthesis.md`.
- Rejected: classifying independently in the router, artifact model, and skill; invoking dedicated workflow planning for every direct/lean task; choosing lean versus full only after subagent fan-out.

### D3 — Shape Selection Uses A Dominance Decision Table (`B01-F02`, `B01-F06`, `B01-F07`)

Evaluate in this order:

1. If intake is not accepted, shape is not selectable and Phase 0 remains active.
2. If any authoritative full-orchestrated trigger is `true` or remains approval-relevant `unknown`, the minimum shape is `full orchestrated`. A user-owned unknown that prevents an accepted brief blocks intake instead of being guessed.
3. Otherwise, `direct path` is legal only when all positive predicates are true: tiny, reversible, one material surface, obvious validation, and no need for durable research, subagents, or resume.
4. Otherwise, `lean local` is selected only for bounded non-trivial single-domain work with stable ownership and a clear proof path.
5. Any remaining cross-domain, hard-to-reverse, ambiguous-owner, unclear-proof, or unclassified case selects `full orchestrated`.

The selected shape records matched rule, evidence pointers, and reopen trigger. `full orchestrated` without a matching floor is challengeable because it violates the smallest-correct-shape invariant.

The five-step algorithm above is normative hard policy in `AGENTS.md`. Its rows receive stable rule IDs that lower-precedence docs, artifacts, adequacy, fixtures, and evals reference. The artifact model projects artifact/state consequences from the selected rule; it does not own another copy of the selection predicates.

- Rejected: unordered prose predicates; a `UNIQUE` rule set that pretends real trigger overlap cannot occur; choosing full merely because agents or skills are available.
- Pattern basis: ordered decision-table `FIRST` semantics plus a monotonic floor, as compared in `research/pattern-fit.md`.

### D4 — Agent Authorization Is Intent, Not Shape State (`B01-F07`)

Canonical field:

`agent_request: absent | capability_only | substantive`

- `capability_only`: the user permits agents, uses the repository's exact authorization line, or says agents may be used if helpful. This grants execution capability but has no shape effect by itself.
- `substantive`: the user requires agent-backed execution, named fan-out, independent lane evidence, or multi-agent participation as part of the accepted result. This is an authoritative full-orchestrated trigger.
- `absent`: no authorization or substantive request is present.
- If required lanes cannot run only because capability authorization is absent, the owning gate is `blocked: missing explicit subagent authorization`; it cannot become `local_only`, `scoped_down`, `waived`, or `not_expected`.
- Prompt-generated authorization text is interpreted as `capability_only` unless separate task intent proves `substantive`.
- Rejected: treating every authorization line as substantive; removing the exact line without runtime proof; storing agent intent as an artifact or phase status.

### D5 — Workflow State Uses Orthogonal Typed Namespaces (`B01-F03`)

Canonical fields and values:

| Namespace | Values | Meaning |
| --- | --- | --- |
| `execution_shape` | `direct_path`, `lean_local`, `full_orchestrated` | Selected process shape. |
| `artifact_expectation` | `expected`, `conditional`, `not_expected` | Whether the artifact should exist for this route. |
| `artifact_state` | `absent`, `draft`, `review_ready`, `approved`, `complete`, `blocked` | Current artifact lifecycle. |
| `record_validity` | `current`, `stale`, `superseded` | Whether an artifact, gate result, verdict, handoff, or envelope is valid for the referenced route. |
| `phase_state` | `not_started`, `active`, `complete`, `blocked`, `reopened` | Workflow-phase progress. |
| `procedural_gate_state` | `pending`, `complete`, `blocked`, `waived`, `not_expected` | Whether a procedural gate ran or has an eligible disposition. |
| `review_verdict` | `pending`, `PASS`, `CONCERNS`, `FAIL`, `WAIVED` | Independent review/readiness result. |
| `subagent_gate` | `complete`, `scoped_down`, `local_only`, `waived`, `not_expected`, `blocked` | Repository-owned lane/fan-in disposition. |
| `waiver_disposition` | `none`, `waived` | Separate scoped waiver with eligibility, rationale, evidence, and reopen trigger. |
| `session_boundary` | `open`, `reached` | Whether the owning phase may continue in the current session. |
| `handoff_readiness` | `not_ready`, `ready`, `blocked` | Whether the recorded next session can start. |
| `routing_scope` | `current_session`, `durable` | Whether the routing record may survive the current session. |
| `routing_revision` | positive integer within its routing scope | Freshness key. The identity is the pair `(routing_scope, routing_revision)`. |

Rules:

- Lifecycle/verdict and freshness compose. Examples include `artifact_state=approved + record_validity=stale`, `review_verdict=PASS + record_validity=stale`, and `procedural_gate_state=complete + record_validity=stale`.
- Every gate result, verdict, and handoff records the routing identity it observed. A revision mismatch makes the prior record stale; stale records remain historical evidence but cannot authorize readiness or execution.
- Display phrases are derived, not canonical state. `missing, expected later` maps to expectation `expected` plus state `absent`; `conditional, trigger unknown` maps to expectation `conditional` plus state `absent`.
- `waived` is never a synonym for absent or not expected.
- Ambiguous legacy phrases fail closed as `status unclear`; consumers do not guess.
- Completed historical bundles remain human-readable but are not assumed to be fully resumable. Typed inference is allowed only through the closed legacy mapping below; every other value is reported verbatim as `legacy_unmapped` and cannot authorize a gate, handoff, or implementation.
- New writers emit canonical shape names and typed components; `lightweight local` is a read-only alias for historical `lean local`.
- Rejected: one global status enum, compound free-text as authority, or deriving shape/approval from file presence.

Closed legacy read mapping after trimming Markdown backticks, surrounding whitespace, and one trailing period:

| Legacy field/value | Typed projection |
| --- | --- |
| Shape `direct path` or `direct_path` | `execution_shape=direct_path` |
| Shape `lean local`, `lean_local`, `lightweight local`, or `lightweight_local` | `execution_shape=lean_local`; lightweight forms are read-only aliases |
| Shape `full orchestrated` or `full_orchestrated` | `execution_shape=full_orchestrated` |
| Phase state `pending` or `not_started` | `phase_state=not_started` |
| Phase state `active`, `in_progress`, or `in progress` | `phase_state=active` |
| Phase state `complete`, `completed`, or `done` | `phase_state=complete` |
| Phase state `blocked` or `reopened` | matching canonical phase state |
| Artifact `approved`, `draft`, `blocked`, `complete`, or `completed` | `artifact_expectation=expected`, matching lifecycle (`completed` -> `complete`), `record_validity=current` |
| Artifact `missing`, `missing, expected later`, or `missing, expected next` | `artifact_expectation=expected`, `artifact_state=absent`, `record_validity=current` |
| Artifact `present, complete evidence` | `artifact_expectation=expected`, `artifact_state=complete`, `record_validity=current` |
| Artifact `conditional` or `conditional, trigger unknown` | `artifact_expectation=conditional`, `artifact_state=absent`, `record_validity=current` |
| Artifact `not expected` | `artifact_expectation=not_expected`, `artifact_state=absent`, `record_validity=current` |
| Artifact `waived` | `artifact_expectation=expected`, `artifact_state=absent`, `waiver_disposition=waived`, `record_validity=current` |
| Procedural gate `pending`, `complete`, `blocked`, `waived`, or `not_expected` | matching canonical gate state |
| Verdict `PASS`, `CONCERNS`, `FAIL`, or `WAIVED` | matching canonical verdict |
| `Session boundary reached: yes|no` | `session_boundary=reached|open` |
| `Ready for next session: yes|no` | `handoff_readiness=ready|not_ready` |

Descriptive suffixes such as historical `full orchestrated ...` shape prose or `complete for ...` phase prose are deliberately not parsed as typed authority. Workflow-status may display them but returns `legacy_unmapped` for readiness purposes. The bounded inventory that motivated the aliases is `specs/railway-auto-migrations/workflow-plan.md`, `specs/orchestrator-soul-md/workflow-plan.md`, and `specs/workflow-simplification/workflow-plan.md`; no broader historical rewrite is required.

### D6 — Reclassification Is One Guarded Revisioned Transaction (`B01-F04`)

Every initial classification, escalation, downgrade, same-shape refresh, or reopen records:

- transition kind;
- prior and target routing identity `(routing_scope, routing_revision)`, shape, and phase;
- trigger evidence and actor;
- artifact expectation/state delta;
- preserved, stale, superseded, or newly required artifacts;
- gate/verdict invalidations or preserved results with rationale;
- session boundary, next route, proof obligation, and reopen target.

Transition rules:

- Artifactless direct initial classification creates `routing_scope=current_session, routing_revision=1` in the orchestrator-owned direct envelope. Current-session changes increment that revision. Escalation materializes `routing_scope=durable, routing_revision=1` and records the exact current-session routing identity as its source before any further work.
- Any direct predicate from D3 becoming false or approval-relevant unknown causes immediate eligibility loss; the examples in D1 are not an exhaustive substitute for the decision table.
- Upward transitions are `direct -> lean/full` and `lean -> full` when new evidence establishes the higher floor.
- Downgrade requires explicit evidence that every trigger responsible for the prior higher floor is now false or no longer in accepted scope. Missing files, elapsed time, or perceived task simplicity are not evidence.
- A changed assumption sets every dependent artifact, gate result, verdict, adequacy result, and handoff to `record_validity=stale`. The prior lifecycle/verdict remains visible, but procedural gates reset to `pending` or `blocked`, handoff readiness becomes `blocked`, and no stale result authorizes execution. Unaffected evidence may survive only with an explicit dependency rationale.
- Master and active phase control must share the same revision before handoff is ready. Partial updates or conflicts set phase/handoff state to blocked.
- Existing durable artifacts are retained as historical `superseded` evidence on downgrade; they are not deleted to simulate artifactlessness.
- Artifactless direct escalation creates durable control before multi-session resume. The current-session direct envelope and any local edits become stale/unapproved inputs to the new route.
- After ledger approval, implementation reads `tasks.md` first, records the blocker and exact reopen target there, marks implementation readiness/result validity stale or blocked, and stops. A new owning-phase session performs reclassification in reopened durable control. Until fresh task review/readiness is recorded for that route, `tasks.md` remains the source of implementation state but cannot authorize execution; reopened workflow control owns phase routing. Historical pre-reopen control is not silently edited mid-implementation.
- Rejected: append-only event sourcing, content-hash coupling, master-only updates, silent approval survival, or workflow-status repair authority.

### D7 — Research, Next Phase, Collapse, And Phase Files Are Independent (`B01-F05`, `B01-F08`)

Routing records four separate decisions:

- `research_expectation: expected | conditional | not_expected`;
- `next_phase: <one phase or reopen target>`;
- `same_session_collapse: prohibited` for every distinct non-implementation phase;
- `phase_control: required | not_required` with trigger evidence.

Rules:

- `research_expectation=not_expected` normally ends workflow planning and routes the next session to specification. It does not authorize same-session specification.
- A distinct non-implementation phase never collapses into the next phase. Direct/lean work may omit a concern whose trigger is `not_expected`, or keep inline routing inside the phase that actually owns the work; that is concern omission, not phase collapse. The approved-ledger implementation exception is continuous execution inside implementation, not a non-implementation collapse.
- `workflow-plans/<phase>.md` is required only for durable local orchestration: multi-lane routing, fan-in, formal challenge, a multi-session stop rule, or a named review/validation checkpoint.
- A mandatory gate does not by itself require a phase-local file. In particular, specification review may be recorded in the master or lean spec when durable local orchestration is absent.
- This task requires `workflow-plans/specification-review.md` because the planned review is multi-lane and has a separate session boundary, not merely because `spec.md` exists.
- Rejected: always creating a specification-review phase file; treating research skip as a waiver; inferring gate existence from file existence.

### D8 — Adequacy Independently Falsifies Shape Under One Closed Predicate (`B01-F06`, `B01-F09`)

A local deterministic trigger audit runs for every shape selection. Formal workflow-plan adequacy is required when any condition is true:

- selected shape is `full_orchestrated`;
- any authoritative full-orchestrated trigger is true or approval-relevant unknown;
- dedicated workflow-planning control is created or substantially repaired;
- a downgrade or reclassification could invalidate a prior formally challenged route.

Direct/lean work without durable workflow control uses the recorded local matrix check rather than a formal challenge.

Formal adequacy:

- receives the accepted brief, normalized trigger audit, selected/prior shape, routing revision, reclassification/stale-state record, and artifact expectations;
- independently evaluates the authoritative matrix and smallest-correct-shape rule;
- blocks false direct/lean, unresolved protected triggers, unsupported downgrade, inconsistent revisions, or undisposed stale state;
- may challenge unsubstantiated full routing;
- returns advisory findings only and never edits, approves, classifies, or reclassifies state.

Lower-precedence surfaces use this predicate by reference. They do not add `high-risk`, `complex workflow-control`, bare `agent-backed`, or other synonyms. Where prose needs a friendly phrase, it must resolve to the canonical predicate and named evidence.

Ownership is explicit: `AGENTS.md` owns the formal-adequacy predicate and advisory/no-approval invariant; the artifact model owns the trigger-audit, routing-revision, and stale-state record consumed by the gate; `workflow-plan-adequacy-challenge` and `challenger-agent` own procedure only.

### D9 — Workflow-Status Supports A Bounded Artifactless Envelope And Otherwise Fails Closed (`B01-F10`)

Workflow-status reads, in authority order:

1. `tasks.md` for implementation/closeout state and any recorded blocker/reopen pointer; it authorizes execution only when its readiness/verdict is current for the active durable route;
2. durable master and active phase control for pre-code routing;
3. approved phase artifacts according to the existing artifact-first chain;
4. an explicit current-session `direct_state_envelope` only when no durable task artifact exists.

The direct envelope is created and attested only by the current orchestrator after it re-evaluates D3 from the accepted brief and current source reads. It contains `provenance=orchestrator_current_session`, current-session routing identity, accepted framing or clarity rationale, direct trigger audit, selected actor, proof obligation/result, session state, and reopen seam. User-quoted, prior-session, or provenance-unknown envelopes are unsupported rather than trusted.

The envelope remains observable after `session_boundary=reached` through the current session's final status/closeout response. It expires when a new session begins or current-session provenance is no longer available and is never a later resume source.

Every report includes:

- execution shape, rationale/evidence, and routing revision when durable;
- adequacy required/result/evidence;
- phase, session, artifact, gate, review/readiness, and freshness/conflict state;
- implementation eligibility and exact next action.

If neither an identifiable task path nor an explicit current-session envelope exists, return `unsupported: no durable task state` and request the missing path/envelope. Never infer shape from task size, chat memory, or file absence. The helper remains read-only and cannot create, repair, approve, waive, or reclassify anything.

Cross-phase ownership is explicit: the artifact model owns status source precedence, the direct-envelope contract, report fields, and fail-closed semantics; `workflow-status` owns only the read-only procedure and presentation.

### D10 — Enforcement Separates Deterministic Semantics From Behavioral Evals (`B01-F11`)

- A repository-owned, network-free routing contract check parses stable, rule-ID-bearing normative tables from `AGENTS.md` and the artifact model, then evaluates independent table-driven input/expected-output fixtures. The documented tables remain authority; checker code and fixtures are proof-only consumers and cannot redefine a rule.
- The deterministic corpus covers shape precedence, every protected trigger class, capability versus substantive agent intent, typed-state combinations, reclassification, stale revisions, research skip, phase-file triggers, adequacy predicates, artifactless status, and mirror states.
- `required-guardrails-check.sh` continues to enforce required canonical files, owner links, read-only agent configuration, required eval cases, and forbidden stale terminology. Regex/file checks are not described as semantic behavior proof.
- Workflow-planning, workflow-plan-adequacy, and workflow-status eval manifests gain direct/lean/full, false-shape, escalation, downgrade, resume, authorization, phase-file, and artifactless cases.
- Behavioral evals are reported as test assets, not CI-executed proof, until a repository-owned credential-safe deterministic runner exists and CI invokes it.
- The executable target is `make workflow-routing-check`. `ci-local` invokes it directly. `scripts/dev/docker-tooling.sh` exposes the same command and its `ci` branch invokes it, so `docker-ci`, `check-full`, and `pr-check` inherit it. `.github/workflows/ci.yml` and the `release-preflight` job in `.github/workflows/cd.yml` invoke `make workflow-routing-check` explicitly. A missing direct or inherited invocation is a guardrail failure.
- Guardrails require every canonical routing/state/adequacy rule ID to have fixture coverage and every fixture to reference existing rule IDs. A normative change starts in `AGENTS.md` or the artifact model, never in the checker or expected outputs.
- No new machine-readable policy file or generated-doc authority is introduced. If implementation cannot parse the normative tables without creating such an owner, reopen specification/artifact depth instead of silently promoting fixtures to policy. On the selected model, `design/overview.md` remains not expected.

Rejected:

- regex-only semantic claims;
- requiring a live model or network in merge CI;
- calling eval JSON CI-covered without a runner;
- scanning completed historical bundles against the new-write schema.

### D11 — Canonical And Runtime-Mirror Availability Is Explicit (`B01-F11`)

Canonical sources remain tracked:

- repository policy/docs and `.agents/skills` for workflow instructions;
- `.codex/agents/*.toml` for agent configuration;
- scripts/evals/Make/CI for enforcement.

`canonical_available: true | false` is an independent prerequisite. Each configured mirror target is then classified separately as:

- `mirror_optional_absent`;
- `mirror_present_in_sync`;
- `mirror_present_stale`;
- `mirror_required_missing` only when a named consumer explicitly requires that mirror.

The checked-in target registries in `scripts/dev/sync-agents.sh` and `scripts/dev/sync-skills.sh` own consumer ID, target path, and `required | optional` for generation purposes; all current targets default to optional. Requiredness changes only through a reviewed change to those registries. The scripts do not own workflow policy.

Check behavior:

- canonical sources are validated directly; `canonical_available=false` always fails;
- mirror check always renders expected output into a temporary directory, including when the local mirror is absent;
- optional absence may pass only as `mirror_optional_absent` after successful render proof; it is never reported as in sync;
- present exact mirrors pass; present stale mirrors fail;
- a required consumer's missing mirror fails;
- strict/non-strict target-only-file behavior remains explicit;
- checks leave tracked and untracked repository state unchanged.

The aggregate passes only when canonical sources and temporary render succeed and every target is either present/in-sync or optional/absent. Any present/stale or required/missing target fails without hiding the other per-target results.

External runtime discovery remains a proof boundary: repository checks prove configured paths and deterministic rendering, not that every external runtime loaded the mirror. Generated mirrors are not committed by default.

## Authority And Consumer Matrix

| Concern | Canonical owner | Consumers |
| --- | --- | --- |
| Complete shape decision table, direct actor, agent intent boundary, formal-adequacy predicate, session invariant | `AGENTS.md` | router, artifact model, skills, agents, guardrails |
| Typed state, reclassification, artifact/phase-control mechanics, status precedence/envelope/report contract | `docs/spec-first-workflow/shared/artifact-model.md` | phase docs, session skills, status, adequacy |
| Loading and phase routing index | `docs/spec-first-workflow.md` | all sessions |
| Phase-local procedure and write/stop rules | one owning phase doc | matching session skill and workflow control |
| Subagent authorization and handoff mechanics | `docs/spec-first-workflow/shared/subagents-and-handoff.md` | phase docs, skills, prompts, subagent contract |
| Task-local final decisions | this `spec.md` until implemented into canonical owners | specification review, test design, planning |
| Adequacy procedure | `.agents/skills/workflow-plan-adequacy-challenge` and `.codex/agents/challenger-agent.toml` | formal workflow-plan challenge; policy remains upstream |
| Status procedure/presentation | `.agents/skills/workflow-status` | read-only status requests; policy remains in artifact model |
| Deterministic semantic proof | rule-ID fixtures/checker derived from canonical tables | Make, CI/CD, guardrails |
| Agent/skill mirror target registry and renderer | checked-in sync scripts; canonical content remains `.codex/agents` and `.agents/skills` | optional or explicitly required local runtimes |

No lower-precedence consumer may restate a conflicting trigger table, status schema, or transition model.

## Dependency / OSS Due Diligence

Applies: no.

Reason:

- The selected solution is a repository-native Markdown contract plus deterministic local fixtures/checks.
- No runtime library, workflow engine, state-machine package, database, service, or custom infrastructure is required.
- Existing shell/Make/CI and eval conventions can carry the proof surface.

Rejected options:

- DMN, AWS Step Functions, Temporal, SCXML/statechart, or another workflow dependency: their pattern disciplines are useful, but their runtime machinery and ownership cost do not fit a repository-instruction problem.
- A model-backed CI runner as mandatory semantic proof: no repository-owned runner is evidenced and deterministic fixtures cover merge-blocking semantics without credentials or network.

## Pattern Fit Diligence

Applies: yes; complete.

Selected composite:

- ordered decision table plus monotonic shape floor for classification;
- orthogonal typed snapshot for state;
- guarded revisioned transition for reclassification;
- condition-like advisory evidence for adequacy and mirror availability.

Evidence and real-use examples are preserved in `research/pattern-fit.md`: Camunda DMN hit policies, AWS Step Functions state transitions, GitHub Check Run status/conclusion separation, and Kubernetes conditions/observed generation.

Applicability:

- The composite gives deterministic overlap handling, separate state dimensions, explicit legal transitions, and visible staleness without importing a runtime engine.
- It fits repository ownership because the state is human-readable, version-controlled, and deterministically testable.
- It does not introduce Go runtime code; where a checker is implemented, straightforward repository-native code or shell remains preferable to framework-shaped abstractions.

Rejected patterns:

- one universal status enum: conflates independent dimensions;
- conditions-only/open-world state: advisory observations cannot authorize transitions;
- hierarchical/parallel statechart runtime: no concurrent runtime regions justify it;
- event sourcing: duplicates Git history and adds projection/replay/versioning machinery;
- external workflow engine: adds dependency and operational ownership without solving a runtime problem.

## Legacy Surface Disposition

This change replaces existing workflow-policy paths.

| Surface | Decision | Owner/reason | Proof obligation / exit condition |
| --- | --- | --- | --- |
| Direct-path worker-only wording that conflicts with no-ledger direct execution | `refactor_into_active_path` | `AGENTS.md` plus artifact model; establish D1 | Negative search plus direct/ledger actor fixtures |
| Competing shape-selection wording in router/artifact model/workflow-planning skill | `refactor_into_active_path` | D2 authority matrix | Owner-link checks and direct/lean/full routing fixtures |
| Flat/compound status examples | `refactor_into_active_path` | artifact model typed schema | Legal-combination fixtures and stale-term searches |
| `lightweight local` historical alias | `retain` for read compatibility only | artifact model/workflow-status; completed bundles are not rewritten | New-output negative search; exit only after historical-read compatibility is explicitly dropped |
| Undefined `high-risk`, `complex workflow-control`, and bare `agent-backed` terms when used as workflow-planning/adequacy trigger predicates | `remove` or replace with canonical predicate references | adequacy predicate owner | Targeted negative search in routing/adequacy surfaces; unrelated domain-risk prose is not in scope |
| Exact subagent authorization line | `retain` | shared handoff owner; platform capability bridge | Capability-only eval; exit only with proven platform-policy change |
| Workflow-planning, adequacy, and status skill procedures/references/evals | `refactor_into_active_path` | procedure owners consume D1-D11 | Required eval cases and behavior review |
| Challenger agent configuration | `refactor_into_active_path` | `.codex/agents` consumes adequacy predicate | Read-only config plus trigger/shape-falsification checks |
| Regex-only guardrails | `retain` for ownership/presence, supplement with semantic check | CI guardrail owner | Guardrail and deterministic checker both run |
| Agent/skill sync scripts and ignored mirrors | `refactor_into_active_path` | canonical-to-generated ownership remains | Absent/exact/stale/required-missing temporary-render fixtures |
| Completed historical bundles under `specs/` | `out_of_scope` | evidence only; no simulated rewrite | Diff scope and negative proof that implementation did not edit them |

Any additional stale lower-precedence wording found during implementation is in scope when it restates a replaced policy. It must be removed, normalized, or explicitly retained with the same owner/reason/proof/exit discipline.

## Formal Clarification Gate

Status: complete.

All five required read-only `spec-clarification-challenge` lenses ran and were reconciled. Initial approval-changing questions were answered from the preserved research and candidate decisions, the affected sections were repaired, and targeted rechecks found no surviving blocker. No targeted research, user decision, specialist reopen, or accepted risk remains.

| Lens | Strongest seam challenged | Final disposition | Decision / proof destination |
| --- | --- | --- | --- |
| Scope and spec coherence (`C1`) | Typed freshness, artifactless-direct transition identity, and session-collapse boundaries | `clear` after targeted recheck | D5-D7; validation obligations 3-5 |
| Domain invariants and edge cases (`C2`) | Direct eligibility loss, stale-state invalidation, post-ledger reopen, and same-session envelope lifetime | `clear` after targeted recheck | D3, D5, D6, D9; validation obligations 1, 4, 7 |
| Architecture ownership and dependency boundaries (`C3`) | Shape-policy ownership, adequacy/status policy versus procedure, and proof-only checker authority | `clear` after targeted recheck | D2, D8-D10; Authority And Consumer Matrix |
| Compatibility and source-of-truth consequences (`C4`) | Closed legacy mapping, direct-envelope provenance, and per-target mirror requiredness | `clear` after targeted recheck | D5, D9, D11; validation obligations 3, 7, 9 |
| Delivery and validation proof (`C5`) | Non-circular checker inputs, mirror aggregation, and exact Make/CI/CD carriers | `clear` after targeted recheck | D10-D11; validation obligations 8-10 |

The gate establishes specification review-readiness only. Checker, CI, eval, and mirror behavior remain forward proof obligations and are not claimed as implemented.

## Open Questions / Assumptions

No unresolved product or policy decision requires user input.

- `[assumption]` The exact authorization sentence remains necessary for at least one supported agent runtime. The sentence is retained while its semantics become capability-only; a proven platform-policy change may reopen retention.
- `[assumption]` No repository-owned model-eval runner exists. Until one is proven, eval manifests are coverage assets and deterministic fixtures are the merge-blocking semantic proof.
- `[reopen_spec_if_false]` If an in-scope runtime requires committed mirrors or cannot consume canonical/generated paths under D11, reopen mirror ownership before planning.
- `[reopen_spec_if_false]` If specification review shows the decision/state/transition model cannot remain compact and unambiguous here, reopen artifact depth for `design/overview.md`; otherwise design remains not expected.

Accepted risks: none.

## Finding-To-Decision Traceability

| Finding | Owning decisions | Required downstream proof |
| --- | --- | --- |
| `B01-F01` | D1, D6, D9 | Direct actor, eligibility-loss, worker-boundary, and status cases |
| `B01-F02` | D2, D3 | Phase 0/direct/lean/full/invocation/falsification cases |
| `B01-F03` | D5, D6 | Typed legal combinations, legacy mapping, stale/conflict cases |
| `B01-F04` | D6 | Escalation, downgrade, refresh, partial update, post-ledger reopen cases |
| `B01-F05` | D7 | Research-not-expected and independent collapse cases |
| `B01-F06` | D3, D8 | False-shape, unknown-trigger, downgrade, advisory-boundary cases |
| `B01-F07` | D4, D8 | Capability-only, substantive, absent-authorization cases |
| `B01-F08` | D7 | Mandatory review with/without durable phase-control cases |
| `B01-F09` | D4, D8 | Canonical predicate and forbidden-synonym cases |
| `B01-F10` | D5, D6, D9 | Durable resume, envelope, unsupported, conflict, report-field cases |
| `B01-F11` | D10, D11 | Make/CI invocation, eval coverage, semantic fixtures, mirror states, clean-check cases |

## Validation

Forward-looking proof obligations:

1. **Shape and actor:** table-driven direct/lean/full cases cover every positive predicate and protected-trigger class; direct code has one actor; approved-ledger code rejects orchestrator patches.
2. **Authorization:** canonical authorization text and permissive agent use remain capability-only; substantive required fan-out establishes the full floor; missing required authorization blocks without silent downgrade.
3. **Typed state:** every legal and illegal expectation/state/validity/gate/session/waiver combination is exercised; the closed legacy table maps exact values and every other value returns `legacy_unmapped` without readiness authority.
4. **Reclassification:** current-session direct revision, direct-to-lean/full durable materialization, lean-to-full, evidence-backed downgrade, same-shape refresh, partial revision conflict, stale gate/verdict/handoff, artifactless durable-root creation, and post-ledger reopen are exercised.
5. **Routing independence:** research not expected routes a new specification session; collapse remains separate; specification review with no durable orchestration omits a phase file while multi-lane review requires it.
6. **Adequacy:** false direct/lean, unresolved trigger, unsubstantiated full, invalid downgrade, stale-state omission, and advisory-authority cases fail before the repair and pass after it.
7. **Workflow-status:** reports shape/adequacy/freshness for durable work, accepts only an orchestrator-attested current-session envelope, keeps it observable through same-session closeout, rejects unknown/prior-session provenance, returns unsupported without a valid source, and never repairs conflicts.
8. **Enforcement:** rule-ID coverage and independent fixtures prove the canonical tables; `workflow-routing-check` runs directly or through the named Make/docker/CI/CD carriers; guardrails fail when the checker, required fixtures/evals, canonical owners, rule links, or invocation is missing.
9. **Mirrors:** canonical unavailable, per-target optional-absent, exact, stale, required-missing, mixed-target aggregate, and strict/non-strict target-only cases render and compare through temporary directories without modifying repository state.
10. **Propagation:** targeted negative searches find no stale new-output terms or duplicated trigger/status/transition authority; canonical sync checks pass; completed historical bundles remain untouched.

Skipped, unavailable, stale, failing, cached-without-proof, or too-narrow evidence cannot satisfy these obligations. The triggered `test-plan.md` must turn them into scenario IDs before task planning.

## Handoff

- Next phase after successful clarification: specification review.
- Specification review must be distinct, read-only, multi-lane, and recorded in `workflow-plans/specification-review.md` because durable review routing is real for this task.
- Separate design and technical design review remain not expected unless the recorded reopen condition fires.
- Triggered test design follows specification review and must produce the scenario matrix before `tasks.md` planning.
- No implementation may begin until `tasks.md` is approved through task-review/readiness.

## Outcome

Pending until implementation and fresh validation prove every finding closure.
