# System / Integration Design

Derive the design drivers, synthesize and select the smallest coherent target-state runtime mechanism that satisfies the accepted behavior and invariants, then trace each material flow from actor or trigger through caller-visible completion or durable finality. Close decisions where they occur along the flow so implementation does not invent system behavior.

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

## Architecture Synthesis And Selection

1. Derive explicit design drivers before selecting the target architecture:
   accepted behavior and invariants; evidence-bounded workload and critical
   path; hard constraints and authorities; triggered risks, failure/recovery
   obligations, and cross-cutting pressures required by [Outputs](#outputs);
   and rollout and proof boundaries. Give each material driver a consequence or
   threshold that can admit or reject a mechanism.
2. Treat the current architecture as evidence, not as the default target.
   Inspect it only until the relevant constraints, reusable capabilities,
   owners, and retained, replaced, or removed surface are known.
3. For each unresolved architecture decision slot, construct materially
   distinct viable target-state substitutes at the same decision level only
   when current evidence leaves a real fork. Consider only applicable live
   substitutes: delete or simplify current machinery; retain or reuse the
   current owner, pattern, dependency, or infrastructure; use a native platform
   capability; use already approved and operated infrastructure; or introduce a
   maintained dependency, managed capability, or custom mechanism. A viable
   candidate fixes the relevant boundaries, authority, ordering and finality,
   failure and recovery, and operational consequences; a pattern, product, or
   topology label alone does not. When the fork is expensive to reverse,
   construct the substitutes in independent generative lanes rather than in
   sequence, so that no candidate is authored against an already-preferred one.
4. Compare surviving substitutes against the same drivers and current evidence.
   Select one coherent architecture, record why it dominates and why each viable
   rejected substitute loses, and name the assumption or reopen condition that
   could reverse the choice. A substitute loses as a whole, not in every part:
   absorb any element of a rejected substitute that the selection can adopt
   without violating a driver, and record it as adopted rather than rejected.
   When evidence leaves one viable mechanism, record what collapses the fork
   without manufacturing alternatives.

### Selection Criterion

Selection is closed only when:

- every material driver, including each triggered cross-cutting pressure, is
  satisfied or enforced by a target decision;
- decision-relevant evidence supports the selected architecture's local
  boundary, ownership, platform, and operational fit, or bounds the exact proof
  gap and reopen checkpoint;
- every affected component, edge, store, contract, or custom mechanism in the
  target is necessary for at least one driver;
- every real fork is closed by same-level comparison against the same evidence,
  or by evidence that no fork remains;
- the combined decisions remain coherent across all material flows.

Trace the selected architecture through Interaction And Data Flow. If that
trace exposes an unresolved mechanism or boundary, return to selection.

## Interaction And Data Flow

Interaction design is complete when a fresh reviewer can trace every material flow from its actor or trigger to caller-visible completion or durable finality without inventing ownership, contract, data authority, failure/recovery behavior, or performance and capacity consequences. A flow is material when its ordering, ownership, contract, authority, failure behavior, finality, or critical-path work and resource demand can change implementation, rollout, or proof. Show current and target flows only when their difference matters.

Select only mechanisms that are behaviorally equivalent under the ready spec; if the trace exposes alternatives with materially different user- or operator-visible outcomes, reopen Specification instead of deciding that divergence here.

For each material flow, record only decisions implementation must not invent:

- Path, boundaries, and owners: target component responsibilities; actor or trigger; ingress or producer; ordered runtime, storage, and broker hops; every affected consumer and side effect; and the completion or finality boundary. At each crossing, name the initiating owner, receiving owner, responsibility transferred, contract, and state or effect produced.
- Contract, transformation, and authority: canonical request/response contract or event schema; routing, partition, or ordering key; trust or tenant context; transformations and persistence; source of truth; decision, write, process, and finality owner; identifiers, units, and absence semantics; and material consistency, visibility, freshness, synchronization or serialization, and concurrency rules.
- Work, scale, and critical path when scale-sensitive: evidence-bounded normal, maximum, and recovery workloads; dominant time and space complexity; retained and new hop multipliers; serial, genuinely parallel, and queued composition; structural ceiling; and later measured boundary. Apply the [Performance Rule](#performance-rule) for the decision method.
- Failure, recovery, and degradation: at each material failure point, state timeout and cancellation propagation, what may already be committed, returned or terminal failure, acknowledgement or offset-commit boundary, retry owner, budget, and exhaustion behavior, idempotency scope, key, and lifetime, correlation, replay, recovery or reconciliation, degraded behavior, and restoration or failback boundary.

When a durable design artifact is required, add a Mermaid diagram only when compact text is insufficient for a reviewer to validate ordering, ownership, fan-out, recovery, or transformation. Use `sequenceDiagram` for temporal request/event ordering and `flowchart` for topology or data transformation. The diagram is a review aid; it must agree with the normative text and canonical contracts and must not become another source of truth.

## Outputs

When the shared [persistence trigger](../shared/artifact-model.md#when-to-persist) applies, use `design/overview.md` or one focused file. Split contracts, data, sequence, or rollout only when that creates a useful review/ownership boundary.

For each material architecture decision slot, record:

- the applicable drivers and current evidence;
- the surviving substitutes or evidence that no real fork remains;
- the selected mechanism and why each viable rejected substitute loses;
- bounded assumptions, a measurable acceptance boundary, material failure,
  operational, and rollout consequences, required proof, and the reopen
  condition.

When behavior changes, distinguish observed current state from the target state
and name what is retained, replaced, or removed. The material-flow record is the
universal coverage check; add only branch-specific decisions whose trigger
matches.

For each security or trust, performance or capacity, observability or operability, rollout or rollback, compatibility, or mixed-version pressure triggered by the ready spec, current evidence, or any viable candidate, use it as a design driver during selection; for the selected architecture, record only the architecture-level decision, owner, operational or proof boundary, and reopen condition. Apply the matching specialist method by reference rather than restating it here.

## Go Runtime Closure

When a material flow enters or changes Go runtime code, apply [Go Change Surface](../../../AGENTS.md#go-change-surface) only to pressures that can change the system mechanism or required proof. Leave package/file placement to [Go Code / Ownership Design](go-code-ownership-design.md); changed-code conformance remains owned by [Implementation](implementation.md).

## Architecture Rule

When a material boundary crossing changes component/runtime authority, source of truth, sync/async interaction, consistency, failure ownership, or migration, apply `go-system-architecture` to that crossing and satisfy its completion criterion. Otherwise keep the flow in the universal kernel.

## Performance Rule

When a material flow is scale-sensitive—its work grows with input or data cardinality, traffic, round trips, serialization or copies, fan-out, retained memory, or contention—or has an accepted latency, throughput, capacity, or resource objective, apply the `go-performance` Decision branch before mechanism closure and satisfy its completion criterion. The disposition covers the complete material flow, including retained stages, rather than only the changed component. Without a numeric budget, `constraint_only` closes only with an evidence-bounded workload, end-to-end multiplier or ceiling, and structural falsifier that rejects the losing mechanism; otherwise reopen the budget owner.

## Contract Rule

When a caller-visible API, generated contract, event, or material shared
interface changes, realize the ready spec's closed semantics as a concrete
contract and runtime path:

- map every accepted request/message, response/result, error/status,
  validation, limit, retry, idempotency, concurrency, async, freshness, and
  compatibility rule to its exact representation and transport behavior;
- select the canonical source, generated outputs, transformation and wiring
  owners, and drift boundary;
- close representation- and rollout-level proof plus migration/deprecation
  sequencing.

An absent or ambiguous observable contract rule reopens Specification; design
does not choose among different caller- or operator-visible outcomes. Reuse
`go-api-contract` for client-visible REST only to realize a behaviorally
equivalent canonical representation and transport wiring after Specification
has applied it to semantics. Route durable event mechanism and replay to
`go-distributed`, and trust-material, signing, tenant, secret, or abuse
mechanism to `go-security`; keep this section's canonical-source, exact-byte,
and external-evidence rules authoritative across those handoffs.

When canonicalization, hashing, signing, or verification depends on a data shape, close it at byte level: exact schema, field order, requiredness/nullability, bounds, exact bytes covered by canonicalization, digest, or signature, and at least one deterministic non-secret golden vector. When keyed signing or verification applies, also define public trust-material lookup and rotation. A metamodel or prose field list is insufficient. Environment-owned keys or trust data follow the router's [implementation-input closure](../../spec-first-workflow.md#implementation-input-closure); do not persist production secrets or private keys in repository artifacts.

Design against an external platform or service requires current official contract evidence. When integration shape or operational fit is non-obvious, also consume credible real implementations or engineering writeups for proven patterns and failure modes. Do not infer current external behavior from model memory. `design/contracts/` is optional context; it never replaces the runtime OpenAPI/event/proto source.

## System Release Closure

Before treating release impact as untriggered, disposition it whenever a change
can affect a runtime deployable, worker or job, public or shared contract,
persisted state or migration, runtime configuration or trust material, or a
managed dependency. Record `no deployment impact` only from current evidence
that no deployed node, edge, state, or operator path can change. Otherwise
derive the smallest affected deployment graph and its integrated release proof
from the documented material flows. Close that graph as one system rather than
treating each repository as independently complete. Keep the result inline
unless the shared persistence trigger requires a durable design artifact.

- Inventory only the APIs, workers, jobs, producers, consumers, data stores,
  brokers, and other managed dependencies whose state or interaction can change
  the accepted outcome. For each affected node or edge, name its owner,
  relevant current-to-target state, canonical contract or configuration source,
  target environment, placement or region, and required network path. Include
  a legacy reader, writer, process, data shape, or alternate configuration form
  only when it can change compatibility, order, recovery, or proof.
- For every changed API, event, shared schema, or material configuration, identify affected producers and consumers plus the mixed-version window. Each affected owner must be updated, proven compatible from current evidence, or recorded as an external blocker; a producer-only green build is not contract closure.
- Close the required deployment inputs, dependency order, irreversible and
  rollback or roll-forward boundaries, and integrated proof. For every release
  gate, name its prerequisite, action, distinct success and safe failure
  signals, evidence-bounded duration or behavior-changing time horizon, recovery
  consequence, and proving surface. Include only triggered infrastructure such
  as migrations, topics, schemas, consumer groups, access policy, service
  discovery, environment-variable semantics, capacity, and latency budgets.
  Persist a non-trivial sequence under the compact [Rollout Record](#rollout-record).
  Proof must
  exercise the material service call or message path through its required
  durable effect and target-environment readiness or post-cutover signals;
  provider deployment status or one component's health is insufficient.

An unverified cross-region or otherwise remote latency-sensitive path is a blocker unless current evidence shows that the accepted end-to-end and per-hop budgets hold. Do not assume either co-location or multi-region safety from platform defaults. If a required node, edge, configuration, or proof is outside current authority, assign its owner and earliest checkpoint and narrow the completion claim instead of declaring the system ready, deployed, or complete.

### Rollout Record

When `rollout.md` is triggered, keep only the affected deployment graph and its
critical path:

```markdown
# Rollout
status: draft | ready | blocked | done
Affected graph: <affected nodes and edges; current -> target state>
Irreversible boundary: <last rollback-safe state and first new-only effect>

| Gate | Owner; node or edge | Prerequisite | Action | Success and distinct safe failure signal | Duration or behavior-changing horizon | Rollback or roll-forward | Proof |
| --- | --- | --- | --- | --- | --- | --- | --- |

Completion: <user-visible path, durable effect, and target-environment signals>
```

Each gate names its authoritative readback and proving surface. Before Planning,
cite the exact Test Design scenario, command, or bounded procedure plus its
environment or fixture. Use an evidence-based bound when available; otherwise
name the signal that ends the step and the missing estimate's owner. Reuse a
passing gate while its candidate, inputs, preconditions, and risk surface remain
unchanged; after failure, repeat only invalidated or still-uncovered gates.

## Fan-Out And Review

After the design drivers and current evidence identify affected domains and
decision slots, apply the shared [Delegation
Decision](../shared/subagents-and-handoff.md#delegation-decision) and the
matching methods under [Routing And Loading](../../../AGENTS.md#routing-and-loading). Route eligible
architecture, contract, data, security, reliability, delivery, observability,
performance, and Go-ownership questions to their specialist read-only lanes.
Keep dependent decisions sequential in the root, synthesize every material lane
result before final selection, and retain one coherent cross-domain
architecture. Load [Technical Design Review](technical-design-review.md) only
when the shared review trigger applies. Apply focused root self-review after
system and Go-ownership decisions are complete.

## Stop Rule

Continue to Go ownership only when architecture selection is closed under [Architecture Synthesis And Selection](#architecture-synthesis-and-selection); [Interaction And Data Flow](#interaction-and-data-flow) is complete for every material flow; every triggered cross-cutting pressure is closed at architecture level under [Outputs](#outputs); and package/file placement cannot reopen a material system decision. Continue to test design or planning after Go ownership and any triggered review have reached `PASS` or dispositioned `CONCERNS`. Reopen the narrowest upstream evidence or decision owner when a missing fact, unset decision, or untraceable material flow can change accepted behavior, ownership, mechanism choice, or proof feasibility; when that owner is external, report the blocker and narrow the completion claim.
