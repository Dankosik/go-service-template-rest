# R2: Status, Reclassification, And Resume

## Question And Scope

What typed state and transition model can unify artifact expectation/lifecycle, phase/gate/session state, reclassification, research-skip routing, phase-file triggers, and artifact-first resume?

Coverage: `B01-F03`, `B01-F04`, `B01-F05`, `B01-F08`, and `B01-F10`. This lane was read-only, advisory, and used explicit `no-skill`. Candidate models are not approved decisions.

## Confirmed Current-State Map

| Dimension | Current representation | Evidence |
| --- | --- | --- |
| Execution shape | `direct path`, `lean local`, `full orchestrated`; `lightweight local` is a read-time compatibility alias. | `docs/spec-first-workflow/shared/artifact-model.md:45-53`; `.agents/skills/workflow-planning-session/SKILL.md:79-90` |
| Artifact expectation/lifecycle | One flat vocabulary: `approved`, `draft`, `missing`, `blocked`, `waived`, `not expected`, `conditional`; examples compose `missing, expected later` and `conditional, trigger unknown`. | `docs/spec-first-workflow/shared/artifact-model.md:211-217`; `.agents/skills/workflow-planning-session/references/artifact-expectation-matrix.md:9-16` |
| Phase state | Master owns current phase and phase status, but no closed phase-state enum exists. | `docs/spec-first-workflow/shared/artifact-model.md:176-189` |
| Gate state | Subagent gates and readiness/review gates use overlapping but different values, including `complete`, `blocked`, `waived`, `not_expected`, `PASS`, `CONCERNS`, and `FAIL`. | `docs/spec-first-workflow/shared/subagents-and-handoff.md:29-44` |
| Session state | Phase status, boundary reached, readiness for next session, and next-session route are separate fields. | `.agents/skills/workflow-planning-session/SKILL.md:157-170` |
| Reclassification | Late trigger discovery only says to block/condition the current artifact and move to a fuller route. | `docs/spec-first-workflow/shared/artifact-model.md:55-57` |
| Resume | Approved ledger first; otherwise master, phase file, context bundle; when master is absent, infer minimally from downstream artifacts. | `docs/spec-first-workflow.md:66-74`; `docs/spec-first-workflow/shared/subagents-and-handoff.md:96-120` |
| Artifactless direct | Durable workflow files are intentionally absent; the route relies on bounded framing/rationale and fresh proof. | `docs/spec-first-workflow/shared/artifact-model.md:88-96` |

Confirmed implication: the repository already stores several dimensions in separate fields, but the value vocabularies are not typed and artifact expectation is still collapsed with artifact existence/lifecycle.

## Per-Finding Evidence

### `B01-F03`: Flat Status Vocabulary Is Ambiguous

- The canonical artifact model presents one flat list rather than separate expectation and lifecycle namespaces (`docs/spec-first-workflow/shared/artifact-model.md:211-217`).
- The planning reference's compound states prove that expectation and existence are distinct: `missing, expected later` and `conditional, trigger unknown` (`.agents/skills/workflow-planning-session/references/artifact-expectation-matrix.md:9-16`).
- The master owns phase, session, artifact, blocker, and gate state at once (`docs/spec-first-workflow/shared/artifact-model.md:176-189`).
- Gate outcomes use another partially overlapping vocabulary (`docs/spec-first-workflow/shared/subagents-and-handoff.md:29-44`).
- Workflow-status manually separates phase, routing/task, session, artifact, review, and readiness in its report, but does not define legal combinations (`.agents/skills/workflow-status/SKILL.md:120-140`).

Evidence-backed inference: a bare `blocked`, `complete`, or `waived` cannot be interpreted safely without its namespace, and impossible combinations cannot be checked deterministically.

Classification: `blocks_spec`.

### `B01-F04`: Reclassification Is Partial And Has No Stale-State Contract

- The only explicit behavior covers upward escalation and updates only the current artifact before moving to a fuller route (`docs/spec-first-workflow/shared/artifact-model.md:55-57`).
- Shape guidance records escalation seams but not an atomic state change (`.agents/skills/workflow-planning-session/references/execution-shape-selection.md:9-14`, `.agents/skills/workflow-planning-session/references/execution-shape-selection.md:30-40`).
- Resume uses fixed artifact precedence with no routing revision, freshness marker, or reclassification certificate (`docs/spec-first-workflow.md:66-72`; `docs/spec-first-workflow/shared/subagents-and-handoff.md:96-120`).
- Once `tasks.md` is approved, workflow control becomes historical (`docs/spec-first-workflow/shared/artifact-model.md:191-193`).
- If implementation discovers an out-of-ledger blocker, it stops in the ledger-owned surface and returns a reopen target rather than repairing workflow control inline (`docs/spec-first-workflow/shared/subagents-and-handoff.md:164-168`).

No assigned source defines downgrade rules, stale/superseded disposition, the atomic update set, artifactless-direct escalation carrier, surviving approvals, or workflow-control reactivation after ledger approval.

Evidence-backed inference: resume can follow stale approval or a partially updated master/phase pair, while downgrade can erase required evidence by omission.

Classification: `blocks_spec`.

### `B01-F05`: Research Expectation Is Coupled To Session Collapse

- Research is a concern, not always a dedicated phase (`docs/spec-first-workflow/phases/research.md:29-32`).
- Workflow planning must record research mode or explicitly say later research is not expected (`.agents/skills/workflow-planning-session/SKILL.md:157-170`).
- Next-session routing nevertheless defaults to research unless an earlier waiver allows same-session collapse (`.agents/skills/workflow-planning-session/SKILL.md:129-133`).
- The one-phase session boundary is independent of phase arrows (`docs/spec-first-workflow/shared/artifact-model.md:59-80`).

Evidence-backed inference: the ordinary route `research not expected; stop workflow planning; next session starts with specification` is not explicit. Skipping research can therefore schedule a nonexistent phase or authorize an invalid collapse.

Classification: `blocks_spec`.

### `B01-F08`: Mandatory Review And Durable Phase File Are Conflated

- A phase-local file is generally created only for durable local orchestration such as lanes, fan-in, formal challenge, a multi-session stop, or a named checkpoint (`docs/spec-first-workflow/shared/artifact-model.md:195-209`).
- Specification-review results may live in the master, a dedicated phase file when durable routing is needed, or lean `spec.md` when no workflow control exists (`docs/spec-first-workflow/shared/artifact-model.md:134-136`; `docs/spec-first-workflow/phases/specification-review.md:108-114`).
- The workflow-planning skill instead says `workflow-plans/specification-review.md` is expected whenever `spec.md` is expected and work is not direct path (`.agents/skills/workflow-planning-session/SKILL.md:118-127`).
- The current task packet gives a task-specific durable reason: a distinct multi-lane review, not mere spec existence (`specs/execution-shape-routing-hardening/workflow-plan.md:88-90`).

Evidence-backed inference: bounded lean reviews can acquire unnecessary competing control files even though the review gate itself is mandatory.

Classification: `blocks_spec`.

### `B01-F10`: Workflow-Status Omits Shape/Adequacy And Cannot Observe Truly Artifactless Direct Work

- The report shape omits execution shape, shape evidence, adequacy requirement/result, and adequacy evidence (`.agents/skills/workflow-status/SKILL.md:120-140`).
- The helper requires a task-local path and stops when none is identifiable (`.agents/skills/workflow-status/SKILL.md:48-60`).
- With no master, it marks workflow control incomplete unless an explicit direct/lean rationale explains the absence (`.agents/skills/workflow-status/SKILL.md:62-78`).
- Direct work normally has no workflow-control bundle (`docs/spec-first-workflow/shared/artifact-model.md:88-96`).
- The direct eval supplies both a task path and a recorded waiver in the prompt; it does not cover truly artifactless work (`.agents/skills/workflow-status/evals/evals.json:38-47`).
- Existing evals cover missing path/artifact and readiness outcomes, but none requires shape or adequacy reporting (`.agents/skills/workflow-status/evals/evals.json:5-80`).

Evidence-backed inference: the helper cannot distinguish valid artifactless direct work from missing state unless the caller supplies a current-session record, and it cannot expose whether classification was adequacy-checked.

Classification: `blocks_spec`; the chosen external/current-session observation mechanism is also a later `proof_only` item.

## Candidate State Models

### R2-M1: Orthogonal Canonical Snapshot

Illustrative dimensions:

- execution shape plus rationale and evidence revision;
- artifact expectation: expected, conditional, not expected;
- artifact lifecycle: absent, draft, review-ready, approved, blocked, stale, superseded;
- separate waiver record: target, eligibility, scope, rationale, evidence, reopen trigger;
- phase state: not started, active, complete, blocked, reopened;
- procedural subagent gate state using the repository-owned namespace;
- review/readiness result using `PASS`, `CONCERNS`, `FAIL`, or eligible `WAIVED`;
- session state: active, boundary reached, ready, blocked;
- reclassification record: source, target, trigger, evidence, affected state, next route.

Legacy displays become derived views; for example, `missing, expected later` means expectation `expected` plus lifecycle `absent`.

Benefit: strongest semantics and deterministic validation. Cost: every writer/reader/eval/guardrail must use the typed fields or an explicit legacy mapping.

### R2-M2: Compatibility-First Composed Rows

Keep human-readable output but require explicit columns:

`Expectation | Lifecycle | Validity | Waiver | Evidence/reopen trigger`

Historical single labels are interpreted only through a closed mapping; ambiguous combinations fail closed.

Benefit: lower presentation churn. Cost: weaker schema unless every consumer validates the components rather than the display label.

### R2-M3: Transition Ledger Plus Projected Snapshot

Record append-only routing events and derive a current snapshot.

Benefit: strongest audit trail. Cost: introduces event schema, projection, replay, and status/adequacy obligations that are disproportionate to Markdown-first workflow control. Pattern Fit evidence rejects a new event-sourced engine for this scope.

Research recommendation for specification to evaluate first: R2-M1 with bounded R2-M2 read compatibility for completed historical artifacts. R2-M3 is not justified. This is not approval.

## Candidate Reclassification Transaction

Any viable model should treat reclassification as one reconciled transition containing:

- source/target shape and phase;
- transition kind: initial, escalation, downgrade, same-shape refresh, or reopen;
- named trigger evidence and routing revision;
- artifact expectation delta;
- explicit preserved, stale, superseded, or newly required artifacts;
- reset/preserved gates and proof obligations;
- session boundary and next route;
- owner and failure/reopen target.

Legal-transition requirements:

- Initial selection follows accepted intake/direct rationale.
- Direct may escalate to lean/full; lean may escalate to full with named evidence.
- Downgrade requires evidence that every former trigger is absent or no longer material; missing files or perceived task size are not evidence.
- Affected approvals reset to pending/reopened instead of surviving silently.
- Artifactless direct work creates a durable transition root before multi-session resume.
- Existing durable artifacts are retained as historical/superseded on downgrade; they are not deleted to simulate artifactlessness.
- Partial transition or master/phase conflict blocks resume and status inference.
- Post-ledger discovery records a blocker/reopen target first; the explicit reopened owner performs reclassification.

Research recommendation: use an explicit routing revision plus named evidence pointers before considering a content hash. Markdown content hashes are likely brittle and have not been justified.

## Independent Research-Skip And Phase-File Model

Specification should evaluate separate fields rather than one coupled route:

- `Research concern: expected | not_expected | conditional`;
- `Next phase: research | specification | reopen`;
- `Same-session collapse: allowed | prohibited` with its own eligibility rationale;
- `Phase-local control required: yes | no` with a durable-orchestration trigger and carrier.

Mandatory specification review and a dedicated `workflow-plans/specification-review.md` must remain separate decisions. A phase file is a carrier selected by multi-lane/fan-in/formal-challenge/multi-session/named-checkpoint need, not proof that the gate exists.

## Workflow-Status Candidate Boundary

Two viable choices remain:

1. Support artifactless direct only from an explicit current-session evidence envelope containing accepted framing, shape/rationale, writer eligibility, proof state, adequacy disposition, and reopen seam.
2. Explicitly return `unsupported: no durable task state` when neither a task artifact nor that envelope exists; never infer shape from task size or chat history.

These choices can be combined: support the named envelope and fail closed otherwise. The helper remains read-only and cannot repair or authorize the route.

## Rejected Alternatives

- One global enum for artifact, phase, gate, and session state.
- Treat `missing`, `not expected`, and `waived` as synonyms.
- Infer current shape from which files happen to exist.
- Let workflow-status resolve conflicts or become an authority.
- Treat research skip as same-session permission.
- Create a specification-review phase file for every non-direct `spec.md`.
- Delete prior artifacts on downgrade.
- Preserve approval after changed trigger evidence without explicit revalidation.
- Introduce an external workflow engine or database for this Markdown contract.

## Compatibility Consequences

- Completed historical bundles need not be rewritten; active/new consumers must use deterministic mappings or fail closed.
- `lightweight local` may remain a read-time alias but should not be emitted as new canonical state.
- A typed schema requires coordinated changes in master/phase templates, skills, status helper, adequacy, guardrails, evals, and sync/mirror consumers.
- Reclassification after ledger approval must not mutate historical workflow control during implementation; it needs an explicit reopen session.
- A current-session direct envelope is non-durable and cannot become a resume source after the session ends.

## Required Proof And Fail-Before Cases

- Truly artifactless direct: named envelope succeeds or the helper explicitly reports unsupported; no guessed state.
- Direct with explicit inline waiver: correct shape/waiver/adequacy disposition and no invented files.
- Lean without master: resume follows approved artifact chain and explicit shape evidence.
- Full with master/phase: shape, adequacy, phase, session, and next route are reported.
- Direct-to-full and lean-to-full: new expectations and affected gates update atomically.
- Full-to-lean/direct: trigger-falsification evidence and retained historical artifacts are required.
- Interrupted reclassification: status/resume blocks on partial or conflicting state.
- Post-ledger trigger: implementation stops in ledger-owned state and routes to an exact reopen owner.
- `research: not_expected` plus normal boundary routes the next session to specification.
- Research skip plus an independently eligible collapse remains distinguishable.
- Mandatory spec review without durable orchestration does not require a phase file.
- Multi-lane/multi-session spec review requires the phase file.
- Expected-but-absent, legitimate not-expected, conditional, and eligible waiver states remain distinct.
- Legacy compound statuses map deterministically or fail closed.
- Workflow-status reports shape and adequacy while remaining advisory.

## Cross-Finding Constraints

- `F03`/`F04` depend on `F02` for the authoritative transition writer and invocation boundary.
- Artifactless direct reclassification cannot silently decide `F01`'s writer.
- Adequacy evidence and stale-state behavior depend on `F06`/`F09`.
- Every schema, transition, and report rule needs `F11` enforcement.
- `F08` carrier selection can be checked by adequacy but is not owned by adequacy.
- `F10` consumes typed state and remains advisory.
- An append-only event model would unnecessarily expand `F06`, `F09`, and `F11`.

## Missing Evidence And Specification Handoff

- The prevalence of every legacy status spelling was not measured; completed bundles are outside rewrite scope.
- No durable carrier exists for current-session artifactless direct state.
- No assigned eval covers research skip, downgrade, stale resume, phase-file triggering, or shape/adequacy reporting.
- Post-ledger reactivation ownership is implicit rather than fully specified.
- Exact adequacy vocabulary depends on R3.

| Finding | Recommended specification destination | Reopen implication |
| --- | --- | --- |
| `B01-F03` | `Typed Workflow State Schema` | Unresolved namespace composition blocks spec completion; unknown legacy mapping may reopen research. |
| `B01-F04` | `Reclassification Transaction And Resume Validity` | New pre-code trigger reopens workflow planning; post-ledger trigger first stops implementation, then reopens the named owner. |
| `B01-F05` | `Concern Expectation And Phase Routing` | Contradictory research expectation/next phase/collapse state reopens workflow planning. |
| `B01-F08` | `Phase-Local Control Carrier Selection` | Wrong carrier selection reopens routing, not the reviewed spec content. |
| `B01-F10` | `Workflow-Status Input And Report Contract` | Missing/contradictory state remains unknown or blocked; the helper never repairs it. |
