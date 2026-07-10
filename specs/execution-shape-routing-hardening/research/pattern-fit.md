# Pattern Fit Diligence: Routing, State, And Status Models

## Scope And Decision Boundary

This note compares established workflow, state-machine, and status-model patterns that could inform `B01-F02`-`B01-F06`, `B01-F09`, and `B01-F10`. It is research evidence, not an approved policy. Exact field names, precedence, transitions, compatibility treatment, and enforcement belong to the future `spec.md` and its review gates.

## Repository Forces Confirmed

- The hard authority exposes three shapes and says to use the smallest shape that preserves correctness, while protected triggers force direct or lean work to full orchestrated (`AGENTS.md:61-83`).
- The artifact model currently records late trigger discovery by marking the current artifact `blocked` or `conditional` and moving to a fuller route, but does not define one atomic transition record or stale-state rule (`docs/spec-first-workflow/shared/artifact-model.md:55-57`).
- The master workflow plan must own shape, phase status, session state, next routing, artifact status, blockers, and active gates (`docs/spec-first-workflow/shared/artifact-model.md:174-189`). Those are related but not interchangeable state dimensions.
- The current shared vocabulary places lifecycle-like, expectation-like, waiver-like, and blocking terms in one flat list: `approved`, `draft`, `missing`, `blocked`, `waived`, `not expected`, and `conditional` (`docs/spec-first-workflow/shared/artifact-model.md:211-217`).
- Workflow-status already reads artifacts in authority order and treats the master as the pre-ledger owner of phase, session, blocker, artifact, and next-session state, but artifactless work has no equivalent durable source (`.agents/skills/workflow-status/SKILL.md:62-94`).
- The accepted task requires one closure for every `B01-F01`-`B01-F11`; in particular it requires separate status namespaces, atomic reclassification/stale-state disposition, independent research-skip and phase-file triggers, shape-aware adequacy, and artifactless-direct status behavior (`specs/execution-shape-routing-hardening/workflow-plan.md:58-74`).

## External Pattern Evidence

Sources were read on 2026-07-09. They are examples and pattern descriptions, not repository authority.

### Decision Table With An Explicit Hit Policy

- Camunda's DMN documentation says a decision table hit policy determines how many rules may be satisfied and which satisfied rules appear in the result. `UNIQUE` treats multiple matching rules as a violation; `FIRST` allows overlap and returns the first matching rule.
- Its concrete examples use `UNIQUE` for season-to-dish selection, where only one season may exist, and `FIRST` for age-to-advertisement selection, where several rules can match but ordering chooses the result.
- Source: [Camunda 8 DMN hit policies](https://docs.camunda.io/docs/components/modeler/dmn/decision-table-hit-policy/).

Applicability: shape predicates can overlap, so an explicit hit policy is materially safer than relying on prose order. `FIRST` supports an ordered dominance rule; `UNIQUE` is useful as a validation invariant when predicates are intended to be mutually exclusive.

### Guarded Finite-State Transitions

- AWS Step Functions documents named states, a typed state kind, explicit `Next` transitions, `Choice` rules with multiple possible targets, and terminal `End`, `Succeed`, or `Fail` states.
- Its Lambda task example shows an actual state passing control through `Next`, while execution details preserve per-state timing, input, and output for inspection.
- Source: [AWS Step Functions workflow states](https://docs.aws.amazon.com/step-functions/latest/dg/workflow-states.html).

Applicability: reclassification and reopen can be expressed as a small guarded transition table with legal sources, evidence predicates, target shape/phase, invalidated state, and terminal or blocked outcomes. The repository does not need an AWS-like runtime engine to borrow this discipline.

### Separate Lifecycle From Outcome And Staleness

- GitHub Check Runs use a lifecycle-like `status` (`queued`, `in_progress`, `completed`, and restricted additional values) separately from a final `conclusion`; a conclusion is required when status is completed, and rerequest behavior resets suite status and clears its conclusion.
- Its examples show one in-progress check with a null conclusion and one completed check with a concrete conclusion.
- Source: [GitHub REST API: check runs](https://docs.github.com/en/rest/checks/runs?apiVersion=2022-11-28).
- Kubernetes API conventions describe typed conditions as a list keyed by condition type, with `True`, `False`, or `Unknown`, `reason`, `message`, `lastTransitionTime`, and `observedGeneration`; an older observed generation explicitly means the condition is stale relative to the desired object.
- The same document uses a Deployment `Available` condition as a real example and warns that one broad `phase` enum hampers evolution and forces consumers to infer properties. It also cautions that conditions are observations, not a comprehensive state machine.
- Source: [Kubernetes API conventions: typical status properties](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties).

Applicability: artifact expectation, artifact lifecycle, phase progress, gate outcome, session boundary, and waiver disposition should be modeled as separately named dimensions. A revision or evidence fingerprint can make stale approvals and stale routing observable. Condition-like proof records can supplement, but must not replace, deterministic transition authority.

### Append-Only Event History

- Temporal documents an append-only Event History used for durable recovery and debugging/audit, with every workflow event persisted and explicit history size/count limits.
- Its activity example records scheduled, started, completed, failed, timed-out, or canceled events and can reset a workflow from selected historical points.
- Source: [Temporal Events and Event History](https://docs.temporal.io/workflow-execution/event).

Applicability limit: this repository already has version-controlled workflow artifacts and needs a deterministic current control record, not a runtime recovery service. A new append-only event ledger would duplicate Git history, introduce replay/compaction semantics, and exceed the accepted scope.

## Candidate Repair Models

### P1: Ordered Shape Decision Table

Candidate mechanics:

1. Evaluate hard full-orchestrated floors, including protected triggers and substantive user-requested agent-backed work.
2. Evaluate every direct-path eligibility predicate, including all negative risk predicates.
3. Route the remaining bounded non-trivial case to lean local, or block classification when required evidence is unknown.
4. Record the matched rule/evidence and the rule that would reopen classification.

Two viable variants remain for specification:

- `FIRST`/dominance: overlapping rules are legal, but higher-risk rules have explicit precedence.
- `UNIQUE`/validation: normalize predicates so exactly one rule is valid; multiple matches fail the classifier and reopen routing.

Fit: strong for `B01-F02`, supports `B01-F06`, and gives `B01-F07` a precise distinction between capability authorization and a substantive execution request. A monotonic floor representation (`direct < lean < full`, choose the strongest triggered floor) is a compact implementation of the dominance variant.

Evidence limit: research does not choose the exact rule order or decide whether unknown evidence blocks or defaults to a safer shape.

### P2: Orthogonal Typed Workflow State

Candidate dimensions:

- `execution_shape`: selected shape plus rationale/evidence revision;
- `artifact_expectation`: expected, conditional, not expected, or eligible waiver disposition;
- `artifact_lifecycle`: absent, draft, review-ready, approved, stale, or blocked;
- `phase_state`: pending, active, complete, blocked, or reopened;
- `gate_result`: pending, pass, concerns, fail, waived, or not expected where the owning gate permits it;
- `session_state`: boundary open/reached and next-session readiness;
- `subagent_gate`: the repository-owned gate namespace and evidence pointer.

Fit: strong for `B01-F03` and `B01-F10`; it prevents `missing, expected later`, `conditional`, `waived`, `blocked`, and `approved` from pretending to be values of one enum. It also makes adequacy and mirror checks validate each dimension independently.

Evidence limit: these names are illustrative. Specification must reuse canonical repository terms where possible and define which combinations are legal rather than copy this list blindly.

### P3: Guarded Reclassification Transaction With Revision

Candidate transition record:

- source shape and source phase;
- trigger/evidence and current evidence revision or fingerprint;
- transition kind: initial classification, escalation, downgrade, same-shape refresh, or reopen;
- target shape, target phase, artifact expectation delta, and next-session route;
- artifacts/gates made stale, preserved, newly expected, or explicitly outside the route;
- owner, proof obligation, and failure/reopen target.

The transition is valid only if all affected state is updated as one logical change. Consumers reject an approval or adequacy result whose recorded evidence revision no longer matches current routing evidence.

Fit: strong for `B01-F04`, and supplies the transition substrate needed by `B01-F05`, `B01-F08`, and `B01-F10`. It uses the guarded-state discipline from Step Functions and the staleness signal from Kubernetes without introducing a runtime workflow engine.

Evidence limit: a content hash may be unnecessary or brittle for Markdown. Specification should compare a simple explicit routing revision, named evidence pointers, or another low-cost freshness carrier before choosing a hash.

### P4: Conditions As Advisory Proof Records

Candidate use: adequacy, authorization availability, mirror availability, and gate evidence can be represented as typed observations with state, reason, evidence pointer, and observed routing revision.

Fit: useful for `B01-F06`, `B01-F09`, and `B01-F11` because it keeps advisory facts visible without granting them transition authority.

Limit: Kubernetes explicitly warns that conditions are not state machines. Conditions alone cannot select shape, authorize execution, or perform reclassification; they must remain subordinate to P1/P3 and orchestrator authority.

## Rejected Alternatives

| Alternative | Why it does not fit this task |
| --- | --- |
| One universal `status` enum | Conflates expectation, existence/lifecycle, phase progress, gate outcome, session routing, and waiver authority; new values create compatibility ambiguity and consumers cannot validate combinations. |
| Unordered prose predicates | Does not define what happens when direct/lean/full descriptions overlap, cannot provide a falsifiable adequacy oracle, and makes reclassification dependent on reader intuition. |
| Conditions-only/open-world model | Good for observations but intentionally does not define comprehensive transitions; it would leave execution and reopen authority ambiguous. |
| Hierarchical/parallel statechart runtime | The accepted flow has one active phase plus orthogonal metadata, not concurrent runtime state regions requiring SCXML-style machinery. A transition table is easier to audit and mirror. |
| Temporal-style event sourcing | Durable replay and audit value are real, but a second append-only workflow history duplicates Git and requires event schema, replay, reset, versioning, and compaction policy outside scope. |
| Adopt DMN, Step Functions, Temporal, or another workflow dependency | The patterns are useful; their runtime/tooling is not. This task changes repository instructions and checks, and no new runtime dependency is currently justified. |

## Non-Authoritative Composite For Specification To Evaluate

The strongest candidate is a small composite rather than one imported framework:

1. one canonical ordered decision table or monotonic shape floor for classification;
2. orthogonal typed state dimensions for artifacts, phases, gates, sessions, and waivers;
3. one guarded reclassification transaction with an explicit freshness carrier;
4. condition-like advisory evidence for adequacy and mirror availability;
5. semantic fixtures generated from the same canonical predicates and transitions.

This composite remains a research recommendation. Specification must decide ownership, exact terms, precedence, legal combinations, transition effects, and compatibility behavior.

## Compatibility Consequences

- Existing historical bundles contain flat and compound strings; they must remain historical evidence, not be rewritten to simulate the new model (`specs/execution-shape-routing-hardening/workflow-plan.md:17-27`).
- Current skills, eval fixtures, guardrails, agent configs, and mirrors may parse or assert current phrases. A one-step target-state update needs an explicit inventory and negative searches for stale semantics.
- A compatibility reader or alias is justified only if a current in-scope consumer demonstrably needs it. The repository's no-temporary-bridge default means research cannot assume a migration shim.
- Artifactless direct work cannot expose durable typed state unless a lightweight non-file carrier is defined; the alternative is to make workflow-status explicitly exclude it and state the observation boundary. Specification must choose one coherent contract.

## Cross-Finding Interaction Check

- `F02` classification predicates must be the same predicates that `F06` adequacy falsifies and `F09` challenge triggers reference.
- `F03` typed dimensions must be the schema consumed by `F10` status reporting and the semantic checks in `F11`.
- `F04` reclassification must update the `F05` research route and the `F08` durable phase-file trigger independently; phase collapse cannot stand in for either.
- `F07` must not turn the mere availability/authorization of subagent tooling into a full-orchestrated trigger; any substantive user-requested agent-backed rule belongs in the P1 decision evidence.
- `F01` execution actor eligibility must remain a separate post-classification consequence. Shape selection alone cannot bypass the approved-ledger worker prerequisite.
- P4 observations cannot approve plans or execute transitions, preserving orchestrator authority and the advisory role of adequacy/subagent outputs.

No candidate is acceptable if it resolves one of these interactions by silently weakening another finding.

## Required Proof Carriers

- A table-driven shape corpus covering direct, lean, every protected trigger, multiple simultaneous triggers, capability-only authorization, substantive agent-backed requests, and unknown evidence.
- Transition cases for direct-to-full, lean-to-full, explicitly justified downgrade, same-shape evidence refresh, later reopen, research not expected, specification-review phase-file creation, and artifactless direct work.
- Invalid-combination cases for each typed state dimension and stale-revision cases for adequacy, approval, and resume.
- Workflow-status fixtures proving authority order, shape/adequacy reporting, conflict handling, and the chosen artifactless boundary.
- Semantic guardrails and evals that consume the canonical decision/transition terms rather than checking only file presence or headings.
- Mirror proof that distinguishes canonical source validity from optional runtime mirror availability.

## Missing Evidence And Handoff

- No external pattern proves the correct repository-specific precedence or vocabulary; those remain specification decisions.
- Research has not established that a content hash is better than an explicit routing revision or evidence pointer.
- Research has not found a need for a runtime workflow engine, state-machine library, or new dependency.
- This model can plausibly fit in a structured `spec.md` using decision, state-dimension, transition, and proof tables. `design/overview.md` should be triggered only if specification cannot keep those cross-surface mechanics compact and reviewable; this note does not create or approve that artifact.
- Recommended specification destinations: `Routing Authority And Shape Decision`, `Typed Workflow State`, `Reclassification And Resume`, `Adequacy And Proof Semantics`, `Compatibility And Mirrors`, and `Validation Obligations`.
