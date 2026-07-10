# System / Integration Design Checkpoint

Internal checkpoint companion for the user-started technical-design macro phase. Run it first when separate design depth is triggered, before Go code ownership and technical design review in the same root session.

## Read When

- Specification review has passed or returned `CONCERNS` with named accepted risks and proof obligations.
- Separate technical design depth is triggered by API, external service, queue, database, cache, consistency, reliability, security, observability, rollout, dependency, or Pattern Fit decisions.
- Planning would otherwise need to choose source of truth, runtime sequence, failure behavior, data/cache shape, integration contract, validation handoff, or rollout shape.
- The next phase is recorded as `system-integration-design`, or the workflow is repairing system/integration design after review.

## Inputs

- Specification-review-approved `spec.md` and specification-review obligations.
- Current `workflow-plan.md` and `workflow-plans/system-integration-design.md` when phase-local routing exists.
- Preserved `research/*.md`, Dependency/OSS Due Diligence, and Pattern Fit evidence when the design depends on them.
- `docs/repo-architecture.md` when stable boundaries, ownership, dependency direction, runtime flow, package layout, generated sources, or infrastructure edges matter.
- Current provider contracts, generated specs, schema sources, queue/event definitions, cache contracts, config sources, or live contract endpoints when the approved scope touches them.

## Outputs

- A system/integration design record in `design/overview.md`, `design/system-integration.md`, or triggered split design artifacts.
- Trigger decisions for `design/contracts/`, `design/data-model.md`, `design/sequence.md`, `design/dependency-graph.md`, `design/pattern-fit.md`, `test-design`, and `rollout.md` when those surfaces are plausible.
- A contract-design checkpoint result when REST resources, OpenAPI or generated contracts, event payloads, client-visible status/error/idempotency/retry/async/freshness/compatibility semantics, or material internal interfaces are changed: `created`, `compact_sufficient`, `not_expected`, or `blocked` with evidence and readiness consequence.
- Recorded `Design fan-out (system/integration): complete | scoped_down | local_only | blocked` with candidate seams, lane table or rationale, fan-in result, and readiness consequence.
- Workflow-control updates that mark this internal checkpoint ready for `go-code-ownership-design`, or name the smallest reopen target when system/integration design is blocked.

## Stop Rule

The checkpoint stops authoring when complete or blocked, but it does not end the user session. When complete, the technical-design root continues directly into Go code ownership. Do not write `tasks.md`, approve implementation readiness, or start implementation.

## Ownership

System/integration design owns the behavior of the service as a system participant. It decides the mechanism that implementation must preserve before code placement is designed.

It owns:

- REST/API, event, queue, generated-contract, or external-call shape inside the reviewed spec boundary.
- Contract-design trigger decisions and the selected contract shape that implementation must preserve before OpenAPI, generated sources, handlers, clients, events, or internal interfaces are edited.
- Source-of-truth ownership, derived surfaces, generated versus handwritten authority, and explicit non-owners.
- Database, migration, cache, projection, retention, replay, consistency, and transaction-boundary decisions.
- Runtime sequence, sync/async boundaries, side effects, idempotency, retries, deadlines, cancellation, cleanup, recovery, and degraded-mode decisions.
- Security, tenant isolation, secret handling, abuse boundary, observability, metric cardinality, rollout, rollback, failback, and validation-handoff decisions when they affect correctness.
- Dependency/OSS and Pattern Fit consequences when they affect the system mechanism.

It does not own:

- Exact task IDs, implementation order, checkpoint placement, or proof execution sequence.
- Final owner file names when package/file placement can be chosen without changing the system mechanism.
- Local code style, helper extraction, or interface naming unless those choices change dependency direction, source of truth, runtime behavior, validation feasibility, or rollout safety.

If this phase discovers a missing approved behavior, invariant, external contract, source-of-truth policy, or proof feasibility decision, reopen specification, research, specification review, or a required user/specialist decision instead of deciding silently.

## Artifact Shape

Use the smallest durable artifact shape that lets Go code ownership design proceed without choosing system behavior:

- Keep the system design in `spec.md` `Compact Design` only when lean-local design is concise, uncontested, and explicitly sufficient for both code ownership and planning. Compact design must still satisfy the system mechanism closure bar below, or record why each closure field is not applicable with evidence.
- Use `design/overview.md` as the design entrypoint whenever separate design exists. It should name the reviewed spec, specification-review obligations consumed, system design status, artifact inventory, selected mechanism, rejected live alternatives, accepted risks, proof obligations, and next route.
- Use `design/system-integration.md` when the system mechanism needs more space than the overview but does not warrant multiple split files.
- Use split artifacts only when a dimension is planning-critical:
  - `design/contracts/` for API, event, generated, or material internal contract context.
  - `design/data-model.md` for persisted state, schema, migration, cache, projection, retention, replay, or consistency decisions.
  - `design/sequence.md` for runtime order, sync/async boundaries, side effects, failure points, cleanup, retry/recovery, and parallel versus sequential behavior.
  - `design/dependency-graph.md` for module/package dependency shape, generated-code flow, coupling risk, or source-of-truth ambiguity.
  - `design/pattern-fit.md` for dense Pattern Fit comparison.
  - `test-design` when validation shape needs a scenario matrix, proof levels, fail-before expectations, or quality gates before planning.
  - `rollout.md` only when rollout shape is too large or safety-critical for `tasks.md`.

The design artifact inventory must list each plausible conditional artifact or phase as `created`, `not expected`, `conditional`, or `blocked`. For `test-design`, use `triggered`, `not expected`, `conditional`, or `blocked` instead of creating `test-plan.md` inside system/integration design. `not expected` requires a concrete trigger test and evidence anchor showing the artifact or phase cannot change Go code ownership design, planning, proof, rollout, or review readiness. `conditional` is valid only when a later phase owns the trigger decision from execution detail; it cannot defer a knowable production-readiness decision. When the trigger is real but undecided, this phase is blocked.

## Triggered Contract Design Checkpoint

Contract design is not a universal phase. It is a mandatory checkpoint inside system/integration design when the approved scope changes any of these surfaces:

- REST resource model, URI shape, method semantics, request or response body, Problem Details/error profile, status mapping, pagination/filter/sort semantics, rate-limit behavior, auth-visible behavior, idempotency, preconditions, retries, async acknowledgement, operation resources, webhooks, freshness/consistency disclosure, compatibility, deprecation, or versioning.
- `api/openapi/service.yaml`, generated OpenAPI bindings, SDK-facing schema, or generated/manual route authority.
- Event payloads, queue messages, webhooks, callbacks, protobuf or other generated contracts.
- Material internal interfaces whose shape affects more than local package placement, test ownership, or private implementation detail.

Checkpoint results are quality gates, not labels:

- `created`: write `design/contracts/` because contract semantics are planning-critical or too dense for compact design.
- `compact_sufficient`: keep the contract in reviewed `spec.md` `Behavior / Contract Delta`, `Compact Design`, or `design/overview.md` because the changed contract is concise, uncontested, and explicit enough for Go code ownership, planning, OpenAPI/source updates, generated drift proof, and tests.
- `not_expected`: record the trigger test and evidence proving the approved scope does not change REST/API, event, generated, or material internal contract shape.
- `blocked`: stop and reopen specification, research, user/specialist decision, or system/integration design when resource ownership, client audience, security context, consistency, retry/idempotency, async behavior, compatibility, provider contract, or source-of-truth authority is unknown.

A `created` or `compact_sufficient` result is closed only when it states the caller or audience, selected resource or message shape, request/response/error/status semantics, retry/idempotency/concurrency rules, async/freshness/consistency rules when relevant, compatibility class, runtime source of truth, generated outputs, proof carrier, and reopen trigger.

For client-visible REST or OpenAPI behavior, use an API/contracts lane with `api-contract-designer-spec` only when there is a concrete independent contract question that can change the selected mechanism, test-design readiness, planning readiness, or implementation safety. Otherwise the root records the already-settled contract consequence locally. A valid lane question is specific, such as "Should this write be synchronous 201, asynchronous 202 with operation resource, or retry-safe replay through Idempotency-Key?", not "review the API."

`design/contracts/` is design-only task context. Runtime authorities remain canonical, such as `api/openapi/service.yaml` for the REST wire contract, event/proto sources for non-HTTP contracts, and generated outputs derived from those sources. Technical design may approve the target contract and required source-of-truth updates, but implementation performs the canonical source edit, regeneration, and drift proof only from an approved `tasks.md`.

Use this closure row when a separate `design/contracts/` file is needed or compact design might otherwise be ambiguous. Omit a field only with a short non-applicable rationale:

```text
Contract surface | Trigger result | Selected shape or preserved contract | Runtime source of truth | Generated or derived outputs | Client/internal compatibility class | Retry/idempotency/concurrency rule | Async/freshness/consistency rule | Error/status model | Proof carrier | Reopen trigger
```

For REST contracts that need `design/contracts/`, include enough of the API-contract decision for planning to task OpenAPI and tests without inventing semantics:

- affected clients or callers, trust boundary, and resource ownership;
- endpoint/resource matrix with methods, statuses, and authoritative read/write semantics;
- request, response, error, validation, and limit shape at the boundary;
- retry, idempotency, precondition, and timeout-recovery behavior;
- async, operation-resource, webhook, freshness, and consistency behavior when applicable;
- compatibility class, coexistence/deprecation/removal rule, source-of-truth update, generated-output update, and proof obligation.

Invalid contract checkpoint outcomes:

- `design/contracts/: not expected` without checking changed REST/API, event, generated, and material internal interface surfaces;
- leaving resource model, status/error semantics, idempotency/retry, async/freshness, compatibility, or generated/manual authority to OpenAPI writing, handler implementation, SDK generation, or tests;
- treating `api/openapi/service.yaml` as a place to invent the design instead of the canonical runtime source updated from approved design and tasking;
- running an API/contracts lane that returns only style advice, route naming preference, or broad "looks fine" approval without evidence tied to a contract surface and reopen condition.

## Required Design Questions

Every completed system/integration design should answer these questions or mark them not applicable with evidence. Each applicable row must classify the answer as `selected mechanism`, `preserved existing mechanism`, or `not applicable`; preserved and not-applicable answers need an evidence pointer and reviewer falsification handle, not only "unchanged" prose.

```text
Question | Status | Decision or preserved boundary | Evidence or source of truth | Proof obligation and carrier | Reviewer falsification handle | Reopen condition
```

System mechanism closure is required for every planning-critical mechanism:

```text
Mechanism | Selected or preserved behavior | Source-of-truth owner | Runtime sequence or failure branch affected | Code-carrying constraint | Rejected live alternative and closure rule | Proof obligation and carrier artifact | Reopen trigger
```

Artifact inventory, workflow agreement, or conditional-artifact status alone is not enough to complete this checkpoint.

Required questions:

- What is the authoritative behavior or contract, and which surfaces are derived or non-authoritative?
- If a contract surface changes, what is the contract-design checkpoint result, selected shape, runtime source of truth, generated or derived outputs, compatibility class, proof carrier, and reopen trigger?
- Which services, adapters, queues, databases, caches, generated artifacts, config sources, and operators participate?
- What is the happy-path runtime sequence and every material failure branch?
- What are the timeout, cancellation, retry/no-retry, cleanup, idempotency, replay, and partial-work rules?
- What data, cache, migration, retention, consistency, or mixed-version behavior must be preserved?
- What security, tenant isolation, secret, abuse, observability, and cardinality boundaries are affected?
- What validation and rollout claims must later proof establish, which artifact carries each claim, and what pass/fail signal proves it?
- Which Dependency/OSS or Pattern Fit question could plausibly change the system mechanism, which option was selected, and which viable alternatives were rejected?

For runtime and failure rows, include ordered actors and steps, authoritative read/write points, side effects, deadline or cancellation boundary, retry/no-retry decision, cleanup or recovery result, and caller/operator-visible outcome for each material failure branch.

For validation and rollout rows, map each claim to `claim -> proof type or artifact -> owner phase -> pass/fail signal`. Rollout-sensitive changes require a selected rollout mechanism or explicit no-rollout-risk evidence.

Invalid shallow answers:

- `not affected`, `unchanged`, or `use existing behavior` without the preserved source-of-truth owner, evidence pointer, and reviewer falsification handle;
- `covered by tests` without the claim, proof type or artifact, owner phase, pass/fail signal, and reopen target;
- `use existing pattern` or `repo default` without naming the actual pattern or mechanism, source evidence, selected boundary, and rejected live alternative when one exists;
- `not expected` for `design/contracts/`, `design/data-model.md`, `design/sequence.md`, `design/dependency-graph.md`, `design/pattern-fit.md`, `test-design`, or `rollout.md` without a trigger test showing the artifact or phase cannot change Go code ownership design, planning, proof, rollout, or review readiness;
- `implementation decides`, `choose during coding`, or equivalent wording for source of truth, runtime sequence, failure policy, validation, rollout, or proof ownership.

## Fan-Out

Before marking the checkpoint complete, identify system seams that could change design correctness or later planning readiness. A candidate seam is not a domain label: it must name the concrete live fork or prove no fork exists, such as source A versus source B, sync versus async, fail-closed versus degraded, single deploy versus expand/backfill/contract, or custom code versus stdlib/repo-pattern/OSS. Use read-only lanes when independent domains can change the system mechanism.

Typical lanes:

- API/contracts and generated-source authority, using `api-contract-designer-spec` for client-visible REST/OpenAPI behavior unless a valid local-only rationale closes the contract lens.
- Data/source-of-truth, cache, migration, retention, or consistency.
- External service, queue, async lifecycle, idempotency, or distributed recovery.
- Security, tenant isolation, secrets, CORS/CSRF, abuse, or privacy.
- Reliability, deadlines, retries, shutdown, overload, fallback, or rollout.
- Observability, metrics, traces, logs, and diagnostic proof.
- Dependency/OSS or Pattern Fit whenever the design introduces a new dependency, custom infrastructure, meaningful abstraction, or non-trivial system mechanism, or when stdlib, repository patterns, mature OSS, or known design patterns could plausibly change the mechanism or planning handoff.

Record only candidate seams that expose a plausible independent design question. Do not enumerate every touched domain merely to justify omitting a lane.

Record the gate in this shape:

```text
Design fan-out (system/integration): complete | scoped_down | local_only | blocked
Candidate seams: <system seams considered>
Lane table: <lane id, lens/domain, live-fork question, artifact section it could change, planning blocker if unanswered, skill/no-skill, inspect-first target, read-only enforcement, status>
Collapsed seams: <duplicate or consequence-only seams folded into the integrated pass>
Escalation seams: <seams that require another lane, specification reopen, research, or user decision>
Fan-in outcome: <orchestrator reconciliation that changes or confirms the system design>
Reviewer falsification handle: <how review can verify omitted lanes and collapsed seams>
Readiness consequence: <ready for go-code-ownership-design | blocked | reopen specification/research/specification review>
```

`local_only` is valid when the root records that the current checkpoint has no concrete independent bounded question whose separate context would materially improve correctness or readiness. `full_orchestrated`, `FULL-*`, and domain count determine design depth, not automatic fan-out. If an actual required lane cannot run because read-only execution or authorization is unavailable, record `blocked`.

## Internal Handoff To Go Code Ownership Design

This checkpoint is ready to continue only when Go code ownership design can choose packages, files, interfaces, and tests without inventing system behavior. The handoff is root-internal and emits no user prompt.

The handoff must include:

- reviewed `spec.md` and specification-review result;
- system design artifact inventory and trigger decisions;
- contract-design checkpoint result, selected contracts, source-of-truth owners, generated or derived outputs, runtime sequence, failure behavior, data/cache shape, rollout or validation obligations, and accepted risks;
- rejected live alternatives with `closed for this scope` or `reopen only if <specific missing or contradictory evidence>` so Go code design does not reopen them casually;
- explicit reopen conditions if code ownership design finds that the selected package/file shape would change system behavior.

For every planning-critical mechanism that can affect package/file placement, proof ownership, rollout, or failure behavior, the handoff must include:

```text
Mechanism | Selected behavior | Source-of-truth owner | Required code-carrying constraint | Rejected live alternative | Rejection reason or closure rule | Proof obligation | Reopen trigger
```

If Go code ownership design would need to choose any system mechanism before placing code, the handoff is not complete.
