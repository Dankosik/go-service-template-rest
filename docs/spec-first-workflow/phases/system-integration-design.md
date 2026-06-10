# System / Integration Design Phase

Detailed phase companion for `docs/spec-first-workflow.md`. Read this when separate technical design depth is triggered and the next design checkpoint must decide how the service behaves as part of the system before Go code ownership is designed.

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
- Trigger decisions for `design/contracts/`, `design/data-model.md`, `design/sequence.md`, `design/dependency-graph.md`, `design/pattern-fit.md`, `test-plan.md`, and `rollout.md` when those surfaces are plausible.
- Recorded `Design fan-out (system/integration): complete | scoped_down | local_only | blocked` with candidate seams, lane table or rationale, fan-in result, and readiness consequence.
- Workflow-control updates that route next to `go-code-ownership-design`, or to the smallest reopen target when system/integration design is blocked.

## Stop Rule

Stop after the system/integration design checkpoint is complete or blocked. Do not write `tasks.md`, approve implementation readiness, start implementation, or silently continue into Go code ownership unless the active workflow explicitly authorizes same-session phase collapse.

## Ownership

System/integration design owns the behavior of the service as a system participant. It decides the mechanism that implementation must preserve before code placement is designed.

It owns:

- REST/API, event, queue, generated-contract, or external-call shape inside the reviewed spec boundary.
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
  - `test-plan.md` or `rollout.md` only when validation or rollout shape is too large or safety-critical for `tasks.md`.

The design artifact inventory must list each plausible conditional artifact as `created`, `not expected`, `conditional`, or `blocked`. `not expected` requires a concrete trigger test and evidence anchor showing the artifact cannot change Go code ownership design, planning, proof, rollout, or review readiness. `conditional` is valid only when a later phase owns the trigger decision from execution detail; it cannot defer a knowable production-readiness decision. When the trigger is real but undecided, this phase is blocked.

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
- `not expected` for `design/contracts/`, `design/data-model.md`, `design/sequence.md`, `design/dependency-graph.md`, `design/pattern-fit.md`, `test-plan.md`, or `rollout.md` without a trigger test showing the artifact cannot change Go code ownership design, planning, proof, rollout, or review readiness;
- `implementation decides`, `choose during coding`, or equivalent wording for source of truth, runtime sequence, failure policy, validation, rollout, or proof ownership.

## Fan-Out

Before marking the checkpoint complete, identify system seams that could change design correctness or later planning readiness. A candidate seam is not a domain label: it must name the concrete live fork or prove no fork exists, such as source A versus source B, sync versus async, fail-closed versus degraded, single deploy versus expand/backfill/contract, or custom code versus stdlib/repo-pattern/OSS. Use read-only lanes when independent domains can change the system mechanism.

Typical lanes:

- API/contracts and generated-source authority.
- Data/source-of-truth, cache, migration, retention, or consistency.
- External service, queue, async lifecycle, idempotency, or distributed recovery.
- Security, tenant isolation, secrets, CORS/CSRF, abuse, or privacy.
- Reliability, deadlines, retries, shutdown, overload, fallback, or rollout.
- Observability, metrics, traces, logs, and diagnostic proof.
- Dependency/OSS or Pattern Fit whenever the design introduces a new dependency, custom infrastructure, meaningful abstraction, or non-trivial system mechanism, or when stdlib, repository patterns, mature OSS, or known design patterns could plausibly change the mechanism or planning handoff.

For every typical lane whose domain is touched by the approved scope, either run a lane or list it under `Collapsed seams` with the tested live-fork or domain-owned question, evidence checked, why it cannot change system design or planning readiness, and the seam that would reopen fan-out.

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

`local_only` is valid only with candidate-lane analysis proving no omitted lane can change system correctness or planning readiness. For full-orchestrated, protected-domain, high-impact, or user-requested agent-backed design, `local_only` is not eligible. In those shapes, `scoped_down` must still run at least one read-only specialist lane for a remaining planning-critical live fork or domain-owned decision unless read-only execution or authorization is unavailable; when no required lane can run, record `blocked`, not `scoped_down`.

## Handoff To Go Code Ownership Design

This checkpoint is ready to hand off only when Go code ownership design can choose packages, files, interfaces, and tests without inventing system behavior.

The handoff must include:

- reviewed `spec.md` and specification-review result;
- system design artifact inventory and trigger decisions;
- selected contracts, source-of-truth owners, runtime sequence, failure behavior, data/cache shape, rollout or validation obligations, and accepted risks;
- rejected live alternatives with `closed for this scope` or `reopen only if <specific missing or contradictory evidence>` so Go code design does not reopen them casually;
- explicit reopen conditions if code ownership design finds that the selected package/file shape would change system behavior.

For every planning-critical mechanism that can affect package/file placement, proof ownership, rollout, or failure behavior, the handoff must include:

```text
Mechanism | Selected behavior | Source-of-truth owner | Required code-carrying constraint | Rejected live alternative | Rejection reason or closure rule | Proof obligation | Reopen trigger
```

If Go code ownership design would need to choose any system mechanism before placing code, the handoff is not complete.
