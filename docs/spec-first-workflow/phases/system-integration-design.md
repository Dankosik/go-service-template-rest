# System / Integration Design

Trace each material flow from actor or trigger through caller-visible completion or durable finality, then choose the smallest coherent target-state runtime mechanism that satisfies the accepted behavior and invariants. Close decisions where they occur along the flow so implementation does not invent system behavior.

## Read When

- Implementation would otherwise decide contracts, source of truth, data flow, failure behavior, cross-service interaction, or rollout.
- The change introduces or materially affects public APIs/events, persisted data, caches, security boundaries, concurrency/lifecycle, or deployment behavior.
- An existing design needs repair after review or new evidence.

## Inputs

- Ready spec and dispositioned accepted risks/downstream proof obligations.
- `docs/repo-architecture.md` when repository boundaries matter.
- Current provider contracts, OpenAPI/event/schema sources, generated-source owners, and relevant runtime code.
- Decision-relevant current-state evidence for invariant/write/process ownership, runtime and data flow, workload and critical path, operators, external dependencies, and mixed-version constraints.
- Research that can change the mechanism.

## Method

Reconstruct every material flow, then trace actor/trigger -> ingress/producer -> ordered runtime hops -> storage/broker -> affected consumers and side effects -> caller-visible completion or durable finality. Activate only the triggered architecture, contract, exact-byte, release-graph, and fan-out branches below.

## Go Runtime Closure

For a material flow entering or changing Go runtime code, classify only the
triggered pressures: mechanism- or proof-changing context budgets; goroutine
owner, stop, unblock, and join; error identity across layers; body, rows, file,
transaction, timer, or shutdown lifetime; mutable publication; and canonical or
generated authority. Apply the matching Go methods to close those pressures.
Exact package and file placement remains owned by Go Code / Ownership Design.

## Outputs

When the shared [persistence trigger](../shared/artifact-model.md#when-to-persist) applies, use `design/overview.md` or one focused file. Split contracts, data, sequence, or rollout only when that creates a useful review/ownership boundary.

For each material mechanism decision, record the selected mechanism, accepted decision drivers, supporting current evidence, bounded assumptions, a measurable acceptance boundary, material consequences for failure, operations, and rollout, required proof, and the reopen condition. When existing behavior changes, distinguish observed current state from the selected target state and name what is retained, replaced, or removed. Use the questions below as coverage checks; omit unaffected domains.

- What target-state component/runtime topology, system-level dependency direction, and invariant/write/process ownership are required, and which surfaces are derived?
- What accepted caller-completion/finality and consistency semantics, workload, and critical path determine request-path versus background work, sync/async interactions, capacity, latency/throughput/resource budgets, backpressure, and load shedding?
- What is the happy-path sequence and each material failure/partial-work branch?
- What are timeout, cancellation, retry/no-retry, idempotency, cleanup, recovery, and degraded-mode rules?
- What contract, schema, cache, consistency, retention, or mixed-version behavior changes?
- What security, tenant, secret, abuse, observability, and cardinality boundaries matter?
- For any transition, what target authority, mixed-version checkpoints, reconciliation, exit/removal proof, rollback limit, and owner bound it?
- When a real live fork exists, which surviving viable simpler or established alternatives were rejected, and why? Do not manufacture options for completeness.

## Architecture Rule

When one of those system decisions is live, apply `go-system-architecture` to the affected boundary and satisfy its completion criterion. Otherwise keep this phase compact.

## Contract Rule

When a caller-visible API, generated contract, event, or material shared interface changes, decide the semantics before implementation:

- audience/owner and trust boundary;
- request/message, response/result, error/status, validation, and limits;
- retry, idempotency, concurrency, async, freshness, and compatibility behavior where relevant;
- canonical source and generated outputs;
- proof and migration/deprecation consequences.

When canonicalization, hashing, signing, or verification depends on a data shape, close it at byte level: exact schema, field order, requiredness/nullability, bounds, exact bytes covered by canonicalization, digest, or signature, and at least one deterministic non-secret golden vector. When keyed signing or verification applies, also define public trust-material lookup and rotation. A metamodel or prose field list is insufficient. Environment-owned keys or trust data follow the router's [implementation-input closure](../../spec-first-workflow.md#implementation-input-closure); do not persist production secrets or private keys in repository artifacts.

Design against an external platform or service requires current official contract evidence. When integration shape or operational fit is non-obvious, also consume credible real implementations or engineering writeups for proven patterns and failure modes. Do not infer current external behavior from model memory. `design/contracts/` is optional context; it never replaces the runtime OpenAPI/event/proto source.

## Interaction And Data Flow

Interaction design is complete when a fresh reviewer can trace every material flow from its actor or trigger to caller-visible completion or durable finality without inventing ownership, contract, data authority, or failure/recovery behavior. A flow is material when its ordering, ownership, contract, authority, failure behavior, or finality can change implementation, rollout, or proof. Show current and target flows only when their difference matters.

For each material flow, record only decisions implementation must not invent:

- Path and owners: actor or trigger, ingress or producer, ordered caller/callee, storage, and broker hops, every affected consumer and side effect, and the caller-visible completion or durable finality boundary.
- Contract and data authority: canonical request/response contract or event schema, broker destination and any material routing/partition/ordering key, trust or tenant context, transformations and persistence, source of truth, identifiers, units, and absence semantics.
- Failure and recovery: deadlines and cancellation, returned or terminal failures, acknowledgement or offset-commit boundary, retry/no-retry, idempotency and correlation, and any DLQ, replay, reconciliation, or degraded-mode behavior.

When a durable design artifact is required, add a Mermaid diagram only when compact text is insufficient for a reviewer to validate ordering, ownership, fan-out, recovery, or transformation. Use `sequenceDiagram` for temporal request/event ordering and `flowchart` for topology or data transformation. The diagram is a review aid; it must agree with the normative text and canonical contracts and must not become another source of truth.

## System Release Closure

When the accepted outcome spans multiple deployables, repositories, or managed dependencies, derive the smallest affected deployment graph and its integrated release proof from the documented material flows. Close that graph as one system rather than treating each repository as independently complete. Keep the result inline unless the shared persistence trigger requires a durable design artifact.

- Inventory only the APIs, workers, jobs, producers, consumers, data stores, brokers, and other managed dependencies whose state or interaction can change the accepted outcome. For each affected node or edge, name its owner, canonical contract or configuration source, target environment, placement or region, and required network path.
- For every changed API, event, shared schema, or material configuration, identify affected producers and consumers plus the mixed-version window. Each affected owner must be updated, proven compatible from current evidence, or recorded as an external blocker; a producer-only green build is not contract closure.
- Close the required deployment inputs, dependency order, rollback boundary, and integrated proof. Include only triggered infrastructure such as migrations, topics, schemas, consumer groups, access policy, service discovery, environment-variable presence, capacity, and latency budgets. Proof must exercise the material service call or message path through its required durable effect and target-environment readiness or post-cutover signals; provider deployment status or one component's health is insufficient.

An unverified cross-region or otherwise remote latency-sensitive path is a blocker unless current evidence shows that the accepted end-to-end and per-hop budgets hold. Do not assume either co-location or multi-region safety from platform defaults. If a required node, edge, configuration, or proof is outside current authority, assign its owner and earliest checkpoint and narrow the completion claim instead of declaring the system ready, deployed, or complete.

## Fan-Out And Review

At phase entry, identify the materially affected domains: architecture/topology, domain behavior, contract, data, security, reliability/distributed flow, observability/performance, and delivery/rollout. Apply each matching method locally by default. A separate lane is triggered only for one concrete bounded independent question that can change a material design decision or its required evidence, under the shared [Delegation Decision](../shared/subagents-and-handoff.md#delegation-decision). Do not run unaffected lenses, and do not turn the number of affected domains into a required lane count.

Parallelize only concrete bounded questions that can be answered independently and can change a material design decision or its required evidence; keep dependent decisions sequential and synthesize all results before selecting the mechanism.

For structured or orchestrated work, run [Technical Design Review](technical-design-review.md) after the system and Go-ownership decisions are complete. The owning root handles repair and fresh re-review in the same root session. Direct work uses independent design review only when the user or risk requires it.

## Stop Rule

Continue to Go ownership only when every material flow is traceable in order from actor/trigger through owners, contracts and data authority, runtime sequence, and failure/recovery to caller-visible completion or durable finality, and package/file placement will not reopen a material system decision. Continue to test design or planning only after Go ownership is complete and the required technical-design review has returned `PASS`. Reopen the narrowest upstream evidence or decision owner when a missing fact, unset decision, or untraceable material flow can change accepted behavior, ownership, mechanism choice, or proof feasibility; when that owner is external, report the blocker and narrow the completion claim.
