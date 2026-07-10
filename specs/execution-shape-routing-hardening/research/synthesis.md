# Research Synthesis: Execution-Shape Routing Hardening

## Research Boundary And Fan-In Result

This synthesis reconciles:

- `research/shape-and-execution-authority.md` (`R1`);
- `research/status-reclassification-and-resume.md` (`R2`);
- `research/adequacy-enforcement-and-mirrors.md` (`R3`);
- `research/pattern-fit.md` (`R4`).

All four required lanes completed. Every lane was read-only and advisory, used explicit `no-skill`, separated facts from candidate models, and returned exact evidence anchors. The orchestrator compared cross-lane assumptions and retained all `B01-F01`-`B01-F11` IDs.

Research fan-in is not blocked: there is enough anchored evidence and a coherent candidate family for the specification phase to make the required policy choices. Nothing in this note is an approved repository decision. `blocks_spec` below means the future `spec.md` must decide the item before specification approval; it does not mean the research phase needs another lane.

## Reconciled Candidate Target Model

The strongest evidence-backed candidate for specification to evaluate is a small composite, not an imported workflow framework:

1. `AGENTS.md` remains hard policy authority; the orchestrator classifies after accepted intake.
2. One canonical ordered decision table or monotonic shape floor makes direct/lean/full precedence and unknown evidence explicit.
3. Dedicated workflow planning is invoked only when durable routing is triggered; it records and independently challenges the classification rather than becoming a second classifier.
4. Eligible direct-path code writing has one explicit actor route; the least-disruptive candidate is a narrow orchestrator-inline exception while approved-ledger code remains worker-mandatory.
5. Capability authorization and substantive user-required agent-backed execution are separate typed intents.
6. Artifact expectation, artifact lifecycle, phase state, gate outcome, session state, waiver disposition, and subagent gate are separate typed dimensions.
7. Initial classification, escalation, downgrade, refresh, and reopen use one guarded reclassification transaction with an explicit routing revision/evidence pointer and stale-state disposition.
8. Research expectation, next phase, same-session collapse, and phase-file requirement are independent fields.
9. Adequacy independently falsifies shape against canonical trigger evidence and consumes, rather than invents, reclassification state.
10. Deterministic routing fixtures run through Make/CI; behavioral skill evals supplement them only when a real runner is proven.
11. Canonical files are always validated; ignored runtime mirrors have explicit availability states and are rendered to temporary directories for generation proof.

This is a research recommendation. Specification must select or reject every policy choice explicitly and may choose another coherent model if it closes the same evidence and proof obligations without reopening another finding.

## Complete Finding Map

### `B01-F01` — Direct-Path Code-Writing Authority

**Verified evidence**

- Direct path is normally artifactless and advertises `edit -> proof` (`AGENTS.md:68`, `AGENTS.md:89-99`, `AGENTS.md:185`).
- Direct-path `tasks.md` must not be created for ceremony (`docs/spec-first-workflow/shared/artifact-model.md:88-96`).
- Direct code is simultaneously required to use a worker, while workers require an approved, reviewed ledger (`docs/spec-first-workflow/shared/artifact-model.md:82-84`; `AGENTS.md:45-47`).
- `go-coder` still supports direct work without `tasks.md` (`.agents/skills/go-coder/SKILL.md:44-48`, `.agents/skills/go-coder/SKILL.md:76-82`).

**Viable repair models**

- Narrow orchestrator-inline direct-code exception; approved-ledger code remains worker-only.
- New minimal direct-worker brief and worker-eligibility exception.
- Remove code-writing from direct path and reclassify every patch at least lean.

**Research recommendation**: evaluate the narrow orchestrator exception first; it changes the fewest owners and matches existing direct semantics. Not approved.

**Rejected alternatives**: choose the writer ad hoc; use a ledger-gated Goal helper without a ledger; let workers/subagents decide policy.

**Specification destination**: `Execution Actors And Direct-Path Code-Writing Eligibility`.

**Required proof**: tiny eligible code case has exactly one writer; unavailable-writer behavior; new protected trigger stops direct work; multi-surface escalation; approved-ledger orchestrator patch rejection; no-code direct remains executable.

**Classification**: `blocks_spec`.

### `B01-F02` — Classification Versus Workflow-Planning Ownership

**Verified evidence**

- Policy belongs to `AGENTS.md`, decisions to the orchestrator, and detailed implications to the artifact model (`AGENTS.md:17-23`, `AGENTS.md:113-123`; `docs/spec-first-workflow/shared/artifact-model.md:13-25`).
- The workflow-planning skill both expects direct/agent-backed knowledge as input and chooses shape later, while skipping direct and non-durable lean work (`.agents/skills/workflow-planning-session/SKILL.md:21-43`, `.agents/skills/workflow-planning-session/SKILL.md:118-127`).
- No scoped source contains a canonical invocation table.

**Viable repair models**

- Authoritative preclassification: orchestrator classifies; artifact model supplies algorithm; dedicated planning persists/falsifies only when triggered.
- Direct eligibility precheck followed by lean/full classification in workflow planning.

**Research recommendation**: authoritative preclassification; one classifier is easier to audit and avoids circular invocation. Not approved.

**Rejected alternatives**: invoke dedicated planning for every task; classify independently in multiple skills; call subagents before shape selection.

**Specification destination**: `Routing Authority And Invocation Table`.

**Required proof**: Phase 0 prerequisite; direct/lean/full routing cases; dedicated planning invoked exactly once; explicit skill invocation on tiny work stops at skip route; new trigger goes through the reclassification owner.

**Classification**: `blocks_spec`.

### `B01-F03` — Typed Status Namespaces

**Verified evidence**

- The shared vocabulary mixes `approved`, `draft`, `missing`, `blocked`, `waived`, `not expected`, and `conditional` (`docs/spec-first-workflow/shared/artifact-model.md:211-217`).
- Planning examples already compose expectation and existence as `missing, expected later` (`.agents/skills/workflow-planning-session/references/artifact-expectation-matrix.md:9-16`).
- Master and status helper handle phase, session, artifact, gate, review, and readiness separately without a legal-combination schema (`docs/spec-first-workflow/shared/artifact-model.md:176-189`; `.agents/skills/workflow-status/SKILL.md:120-140`).

**Viable repair models**

- Orthogonal canonical snapshot with typed axes and derived display labels.
- Compatibility-first composed rows with closed legacy mapping and fail-closed ambiguity.
- Event ledger plus projected snapshot.

**Research recommendation**: typed canonical axes with bounded read compatibility for completed historical artifacts; reject an event ledger. Not approved.

**Rejected alternatives**: one universal status enum; treat missing/not-expected/waived as synonyms; infer shape or validity from files.

**Specification destination**: `Typed Workflow State Schema`.

**Required proof**: legal/illegal combinations; deterministic legacy mappings; expected-but-absent versus not-expected/waived/conditional; namespace-aware status output; stale validity separate from lifecycle.

**Classification**: `blocks_spec`.

### `B01-F04` — Atomic Reclassification And Stale State

**Verified evidence**

- Current policy covers only upward escalation and updates only the current artifact (`docs/spec-first-workflow/shared/artifact-model.md:55-57`).
- Resume has authority order but no revision/freshness check (`docs/spec-first-workflow.md:66-72`; `docs/spec-first-workflow/shared/subagents-and-handoff.md:96-120`).
- Workflow control becomes historical after approved `tasks.md`; implementation may only record a blocker and reopen target (`docs/spec-first-workflow/shared/artifact-model.md:191-193`; `docs/spec-first-workflow/shared/subagents-and-handoff.md:164-168`).

**Viable repair models**

- Guarded transition block containing source/target, trigger evidence, routing revision, artifact delta, stale/superseded state, gate resets, session/next route, proof, and reopen owner.
- Same model with a content hash instead of explicit revision/evidence pointer.
- Append-only transition history plus projected snapshot.

**Research recommendation**: guarded transaction with explicit routing revision and evidence pointers; hashes and event sourcing are not yet justified. Not approved.

**Rejected alternatives**: partial master-only update; infer downgrade from missing artifacts; delete durable history; preserve prior approvals silently; let workflow-status repair conflicts.

**Specification destination**: `Reclassification Transaction And Resume Validity`.

**Required proof**: direct/lean escalation; evidence-backed downgrade; same-shape refresh; partial transition blocks resume; affected gates reset; artifactless direct creates a durable root before resume; post-ledger reopen follows the ledger-owned stop.

**Classification**: `blocks_spec`.

### `B01-F05` — Research Skip Independent Of Session Collapse

**Verified evidence**

- Research is a concern, not always a phase (`docs/spec-first-workflow/phases/research.md:29-32`).
- Planning can record research not expected, but next routing defaults to research unless a same-session waiver exists (`.agents/skills/workflow-planning-session/SKILL.md:129-133`, `.agents/skills/workflow-planning-session/SKILL.md:157-170`).
- One-phase session boundaries remain independently mandatory (`docs/spec-first-workflow/shared/artifact-model.md:59-80`).

**Viable repair models**

- Separate `research expectation`, `next phase`, and `same-session collapse` fields.
- Encode research skip as a special transition but keep collapse as a separate guard.

**Research recommendation**: explicit independent fields; the ordinary skip route ends workflow planning and starts specification in the next session. Not approved.

**Rejected alternatives**: research not expected implies same-session specification; always schedule a research session; use waiver as a synonym for not expected.

**Specification destination**: `Concern Expectation And Phase Routing`.

**Required proof**: not-expected research routes next session to specification; separately eligible collapse remains distinguishable; contradictory expectation/next route blocks handoff.

**Classification**: `blocks_spec`.

### `B01-F06` — Adequacy Must Falsify Shape

**Verified evidence**

- Adequacy receives the selected shape and checks packet proportionality/consistency, but does not require evaluating all authoritative triggers or reclassification evidence (`.agents/skills/workflow-plan-adequacy-challenge/SKILL.md:44-76`).
- The challenger agent repeats that input boundary (`.codex/agents/challenger-agent.toml:29-40`).
- The authoritative matrix is in `AGENTS.md:61-83`.

**Viable repair models**

- Normalized trigger audit plus independent challenge evaluation.
- Formal challenge for every durable workflow-planning packet.
- Deterministic local matrix check for every classification plus trigger-scoped formal challenge.

**Research recommendation**: combine a deterministic local audit with independent formal falsification for full/protected, durable-planning, and challenged-route reclassification cases. Adequacy consumes `F04` state and cannot create its own transition model. Not approved.

**Rejected alternatives**: artifact proportionality proves shape; challenger chooses or approves shape; shape string without evidence; copied trigger matrices in every lower surface.

**Specification destination**: `Execution-Shape Evidence Contract` and `Adequacy Falsification Algorithm`.

**Required proof**: false direct/lean protected cases; unknown trigger evidence; unsubstantiated full; invalid downgrade; escalation without stale disposition; challenger advisory boundary.

**Classification**: `blocks_spec`.

### `B01-F07` — Capability Authorization Versus Substantive Agent-Backed Intent

**Verified evidence**

- Authority uses both `user-requested agent-backed` and broader `user-requested subagents` (`AGENTS.md:70`, `AGENTS.md:74-83`).
- The canonical platform-authorization line itself says the user requests and authorizes subagents (`docs/spec-first-workflow/shared/subagents-and-handoff.md:66-74`).
- Lower surfaces use undefined bare `agent-backed` (`.agents/skills/workflow-planning-session/SKILL.md:79-91`).

**Viable repair models**

- Two booleans: capability authorized and substantive user-required agent execution.
- Typed intent: `substantive | capability_only | absent`.

**Research recommendation**: typed intent; injected authorization defaults to capability-only unless separate task intent is substantive. Not approved.

**Rejected alternatives**: authorization line automatically triggers full; remove explicit-request wording without runtime proof; collapse missing authorization into local-only/waiver.

**Specification destination**: `Subagent Capability And Agent-Backed Intent`.

**Required proof**: canonical line only; permissive use statement; substantive agent-evidence requirement; missing authorization blocker; resume does not reinterpret injected text; independent strict-boundary/broad-audit triggers.

**Classification**: `blocks_spec`.

### `B01-F08` — Specification-Review Phase-File Trigger

**Verified evidence**

- Phase files are generally triggered only by durable local orchestration (`docs/spec-first-workflow/shared/artifact-model.md:195-209`).
- Mandatory specification review can be recorded in master, phase file when durable routing is needed, or lean `spec.md` (`docs/spec-first-workflow/shared/artifact-model.md:134-136`; `docs/spec-first-workflow/phases/specification-review.md:108-114`).
- Workflow planning instead expects a specification-review phase file whenever non-direct `spec.md` is expected (`.agents/skills/workflow-planning-session/SKILL.md:118-127`).

**Viable repair models**

- One general durable-orchestration predicate for every phase-local file.
- A specification-review-specific predicate that duplicates the same factors.

**Research recommendation**: reuse the general predicate; mandatory gate and durable carrier remain independent. Not approved.

**Rejected alternatives**: phase file for every spec; infer gate absence from file absence; use phase-file creation to decide review content.

**Specification destination**: `Phase-Local Control Carrier Selection`.

**Required proof**: mandatory bounded review without phase file; multi-lane/multi-session review with phase file; conflict/duplicate owner rejection; adequacy checks carrier independently.

**Classification**: `blocks_spec`.

### `B01-F09` — Unified Adequacy Predicate And Terminology

**Verified evidence**

- Authority, adequacy skill, workflow-planning wrapper/step/reference, and challenger agent use different predicates (`AGENTS.md:195`; `.agents/skills/workflow-plan-adequacy-challenge/SKILL.md:3`; `.agents/skills/workflow-planning-session/SKILL.md:90`, `.agents/skills/workflow-planning-session/SKILL.md:147-151`; `.codex/agents/challenger-agent.toml:16-21`).
- `high-risk`, `complex workflow-control`, and bare `agent-backed` lack closed authoritative definitions; `lightweight local` should be read-only compatibility (`AGENTS.md:68-83`; `.agents/skills/workflow-planning-session/SKILL.md:83-85`).

**Viable repair models**

- One closed canonical predicate using shape/trigger/durable-control/reclassification facts.
- Challenge every durable-control packet.
- Keep synonyms through a canonical glossary mapping.

**Research recommendation**: one closed predicate and links from lower surfaces; eliminate undefined synonyms. Not approved.

**Rejected alternatives**: all non-trivial work; bare agent-backed; undefined complexity; new normative lightweight-local output; independent interpretations per skill.

**Specification destination**: `Canonical Adequacy Predicate And Terminology`.

**Required proof**: full always formal; ordinary lean local check; protected lean cannot skip; authorization-only does not trigger; substantive request does; downgrade requires trigger-clearing evidence; legacy alias read but not emitted.

**Classification**: `blocks_spec`.

### `B01-F10` — Workflow-Status Shape/Adequacy And Artifactless Direct Boundary

**Verified evidence**

- Current report omits shape, shape evidence, adequacy requirement/result, and adequacy evidence (`.agents/skills/workflow-status/SKILL.md:120-140`).
- Helper requires a task path and treats absent master conservatively (`.agents/skills/workflow-status/SKILL.md:48-78`).
- Direct is normally artifactless (`docs/spec-first-workflow/shared/artifact-model.md:88-96`).
- The direct eval supplies a path and explicit waiver; no truly artifactless or shape/adequacy case exists (`.agents/skills/workflow-status/evals/evals.json:38-47`).

**Viable repair models**

- Accept a named current-session direct evidence envelope; fail closed without it.
- Explicitly exclude all artifactless work and return unsupported.
- Create a durable direct control file, contradicting normal no-bundle routing.

**Research recommendation**: support an explicit non-durable current-session envelope and return `unsupported: no durable task state` otherwise. Never infer. Not approved.

**Rejected alternatives**: infer from task size/chat memory; let status create/repair control; require workflow files for every direct task.

**Specification destination**: `Workflow-Status Input And Report Contract`.

**Required proof**: envelope success; no-envelope unsupported; direct waiver; lean/full resume; shape/adequacy fields; stale/conflicting state; helper remains advisory.

**Classification**: `blocks_spec`; actual runtime/current-session carrier integration is `proof_only` after the contract is chosen.

### `B01-F11` — Semantic Guardrail/Eval/CI And Mirror Availability

**Verified evidence**

- Guardrails are mostly required-file and regex checks and omit semantic routing assertions (`scripts/ci/required-guardrails-check.sh:4-49`, `scripts/ci/required-guardrails-check.sh:173-218`, `scripts/ci/required-guardrails-check.sh:253-254`).
- Planning/status evals lack false-shape, reclassification, and shape/adequacy cases; adequacy has no eval manifest (`.agents/skills/workflow-planning-session/evals/evals.json:5-37`; `.agents/skills/workflow-status/evals/evals.json:5-81`).
- CI and Make run guardrails/mirror checks but no skill-eval runner (`.github/workflows/ci.yml:36-46`; `.github/workflows/cd.yml:138-148`; `Makefile:374-380`, `Makefile:450-463`).
- Canonical agents/skills are tracked; all runtime mirrors are ignored and may be absent while checks pass (`scripts/dev/sync-agents.sh:7-22`, `scripts/dev/sync-agents.sh:119-124`; `scripts/dev/sync-skills.sh:7-35`, `scripts/dev/sync-skills.sh:86-92`; `.gitignore:9-14`).

**Viable repair models**

- Deterministic routing-contract fixtures in Make/CI plus behavioral eval manifests and regex ownership/negative-term checks.
- Normalized machine-readable control-block validator after `F03/F04` schema selection.
- Hermetic model-eval CI only if a bounded runner is proven.
- Mirror states: canonical available, optional absent, present in sync, present stale, required missing; temporary render proves generation.

**Research recommendation**: deterministic fixtures as the merge-blocking baseline; optional normalized validator; behavioral evals must not be called CI-covered without a real runner; keep mirrors generated/ignored but prove temporary rendering and report absence honestly. Not approved.

**Rejected alternatives**: regex-only semantic claims; eval JSON without runner; committed mirrors by default; absence reported as in-sync; new-schema checks over completed historical bundles.

**Specification destination**: `Semantic Enforcement Matrix`, `Fail-Before Scenario Catalog`, `Canonical And Runtime-Mirror Availability`, and `CI/Make Proof Obligations`.

**Required proof**: all routing/status/adequacy cases above; required canonical files; forbidden-term drift; eval manifest coverage; Make/CI invocation; absent/exact/stale/required-missing mirrors; temporary render; clean git state after checks.

**Classification**: contract/enforcement model `blocks_spec`; actual external runtime discovery for Claude/Gemini/GitHub/Cursor/OpenCode is `proof_only` unless scope explicitly expands.

## Cross-Finding Closure Audit

| Candidate closure | Must consume or preserve | Reopen prevented |
| --- | --- | --- |
| `F01` chooses direct writer | `F02` classifier, `F04` escalation, `F10` observation, `F11` proof | Prevents a legal writer fix from bypassing shape/readiness or becoming invisible. |
| `F02` centralizes classification | `F06` independent falsification, `F09` one predicate | Prevents a single owner from becoming self-certifying. |
| `F03` types state | `F04` transition effects, `F10` report, `F11` validator | Prevents new ambiguous compound strings and consumer drift. |
| `F04` makes transitions atomic | `F01` writer stop, `F05` next route, `F08` phase file, `F06` stale evidence | Prevents escalation from leaving old approval or routing live. |
| `F05` separates research/collapse | `F04` next-route transaction, session boundary | Prevents research skip from bypassing specification boundary. |
| `F06` falsifies shape | `F02` canonical evidence and `F04` stale-state record | Prevents adequacy from defining a second classifier/transition model. |
| `F07` types authorization intent | `F02` decision table, `F09` predicate | Prevents authorization plumbing from self-escalating every route. |
| `F08` uses durable-control trigger | `F03` artifact expectation and `F05` session routing | Prevents mandatory review from forcing duplicate control files. |
| `F09` closes predicate | `F07` substantive intent and authoritative matrix | Prevents undefined synonyms from reopening authorization/shape drift. |
| `F10` reports state | `F03/F04/F06` as read-only inputs | Prevents status from becoming a repair or approval authority. |
| `F11` enforces semantics | All canonical decisions plus historical non-rewrite boundary | Prevents guardrails/mirrors from becoming a competing policy source. |

Fan-in conclusion: the composite candidate has no known internal closure that necessarily reopens another finding, provided specification adopts these consumption boundaries. The main dangerous combinations are explicitly rejected:

- direct-worker exception without typed/resumable state (`F01` reopens `F03/F10`);
- adequacy keyed to bare `agent-backed` (`F06` reopens `F07/F09`);
- phase-file-for-every-spec (`F08` reopens artifact ownership/ceremony);
- event-sourced transition engine (`F04` unnecessarily expands `F06/F09/F11`);
- committed mirrors as enforcement (`F11` contradicts current generated-local ownership);
- one universal status enum (`F03` makes `F04/F10/F11` ambiguous).

## Pattern Fit Diligence

Pattern evidence is preserved in `research/pattern-fit.md`.

- DMN hit policies demonstrate explicit `FIRST` versus `UNIQUE` handling for overlapping decision-table rules; this fits shape precedence/falsification.
- AWS Step Functions demonstrates named typed states, guarded choice transitions, explicit next states, and terminal outcomes; this fits a small transition table without importing its runtime.
- GitHub Check Runs separate lifecycle status from final conclusion and clear outcome on rerequest; this fits typed dimensions and reopen semantics.
- Kubernetes conditions demonstrate typed observations and `observedGeneration` staleness while warning that conditions are not comprehensive state machines; this fits proof/adequacy observations subordinate to transition authority.
- Temporal Event History demonstrates real durability and audit value but also the schema/replay/limit machinery of event sourcing; this is disproportionate because Git already preserves repository history.

Pattern conclusion: use the pattern disciplines, not the external engines or dependencies. No new runtime dependency or custom workflow engine is justified by current evidence.

## Design-Depth Decision

`design/overview.md`: **not expected on current research evidence**.

Rationale:

- The accepted scope changes repository workflow policy, documentation, skills, evals, scripts, CI, and mirrors; it does not change service runtime, Go package ownership, or an external integration architecture.
- The candidate model can fit in one structured `spec.md` through an authority/invocation table, shape decision table, typed state table, legal transition table, consumer/owner matrix, and proof matrix.
- Creating a design artifact now would duplicate the canonical decision record rather than add a distinct mechanism owner.

Reopen trigger: if specification clarification shows the state/transition/consumer model cannot remain compact and reviewable in `spec.md`, or if a chosen enforcement mechanism introduces a real architecture/dependency decision, reopen artifact depth before planning. Separate system/integration and Go code ownership design remain not expected on current scope; technical design review is therefore not expected unless this trigger fires.

## Classification Register

| Classification | Items | Consequence |
| --- | --- | --- |
| `blocks_spec` | `F01`-`F11` normative choices, including F11's semantic/mirror contract | Specification must decide and trace each item before it can become review-ready. |
| `proof_only` | Runtime need for exact authorization syntax; artifactless-direct current-session carrier integration; external runtime mirror discovery; existence/CI viability of any model-eval runner | Carry named proof to spec/test design/tasks; absence does not require more research unless the chosen policy depends on it. |
| `accepted_risk` | None | Research accepted no unresolved risk as a substitute for a decision. |
| `needs_specialist` | None | No unresolved domain fact currently requires a new specialist lane; specification challenge/review lanes remain required by the full-orchestrated route. |

## Specification Handoff

Specification should consume this packet and produce review-ready decisions, not copy research recommendations as pre-approved conclusions. Recommended section destinations:

1. `Authority, Definitions, And Invocation Table` — `F02`, `F07`, `F09`.
2. `Execution Shapes And Direct Execution Actors` — `F01`, `F02`.
3. `Typed Workflow State` — `F03`, `F10`.
4. `Reclassification, Freshness, And Resume` — `F04`.
5. `Research, Session, And Phase-Control Routing` — `F05`, `F08`.
6. `Adequacy Falsification` — `F06`, `F09`.
7. `Canonical Sources, Consumers, And Mirrors` — `F11`.
8. `Compatibility And Historical Bundles` — all findings where legacy terms exist.
9. `Validation And Fail-Before Obligations` — `F01`-`F11`.

Specification must preserve a finding-to-decision table for every ID and record selected/rejected model, owner, compatibility consequence, proof carrier, and reopen target. Formal clarification remains required for this full-orchestrated task; specification review remains a distinct later gate.

## Remaining Evidence Limits

- External pattern sources do not determine repository-specific policy.
- Completed historical bundle vocabulary was not exhaustively counted because rewriting those bundles is an explicit non-goal.
- No repository-owned model-eval runner exists today.
- External runtime discovery of ignored mirrors is not provable from repository files.
- No final choice has been made between direct-inline, direct-worker, or no-code direct policy; between exact typed field names; or between the exact formal-adequacy trigger variants.

These are visible specification choices or downstream proof items, not reasons to keep research open.
