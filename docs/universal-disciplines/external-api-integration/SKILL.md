---
name: external-api-integration
description: "boundary-first production integration with external HTTP providers. Use for design, build, diagnosis, or review of API clients, OAuth credentials, idempotency, deadlines/retries/rate limits, ambiguous outcomes, synchronization, webhooks, reconciliation, and provider migration. Own request identity through convergence and proof; hand public API, business/ledger, messaging, and PostgreSQL internals to their dedicated skills."
---

# External API Integration

The leading word is **boundary**. Treat the provider as an independently changing system that can accept an effect while losing the response, delay or duplicate callbacks, reorder events, throttle callers, and evolve its contract.

The boundary process is:

`provider contract -> request identity -> bounded attempt -> outcome classification -> callback/poll -> reconcile -> prove`

An HTTP client is only one part of this process. Keep the same operation identity and evidence across every stage.

## Scope and decision ledger

Match depth to the requested claim:

- For a whole side-effecting flow, production-readiness claim, provider migration, replay, or recovery redesign, trace the full boundary chain.
- For a scoped review, diagnosis, or correction, trace the affected stage and the dependencies that can change its verdict or safety.
- For a side-effect-free operation, start with the provider contract, bounded attempt, outcome, and focused proof; add another stage only when it changes the requested claim.

Respect explicit output and test limits. Expand only to close a safety-critical gap, and name that gap. Report only evidence that supports the requested decision.

Before drafting, reduce the requested decision to one ledger:

| Claim, caller, external effect, or observation/convergence path | Provider evidence or gap | Local invariant and owner | Durable local identity/checkpoint | Exact deadline and acceptance semantics | Ambiguity, recovery, and observable proof | Authority |
| --- | --- | --- | --- | --- | --- | --- | --- |

Keep documented or observed provider guarantees, local decisions, inferences, and unknown gaps in their own columns. Evidence is row-local: never inherit an idempotency, lookup, callback, or ordering guarantee from one endpoint or effect to a sibling; mark the sibling gap instead. A provider key or reference is bound to an operation; it is not the durable local operation identity. Give every side effect that separate local identity before I/O. Give callback, poll, and reconciliation paths their own rows when their evidence, gap, owner, or authority differs. Name the local decision/operation owner in every rendered row or mark an owner gap; a provider account is scope, not ownership.

A row is relevant when omitting it could change the verdict, make the correction unsafe, or leave a requested claim unproved. The ledger is a private preflight, not a default report template. Fold its relevant fields once into the requested format. Render it only when the user asks for a contract/ledger or several effects, callers, or observation paths must be compared; keep every column, shorten cells, and do not restate them in prose.

## Authority and ownership

- For review, diagnose, and design requests, inspect available contracts, code, fixtures, logs, and telemetry and report the result without changing repository or provider state.
- For build and fix requests, make local changes inside the requested scope and run safe, non-destructive tests.
- Treat real provider calls with side effects, credential or scope changes, webhook registration, production event replay, and destructive or authoritative synchronization as separate actions requiring explicit authorization. Keep an authorized action's environment, account, operation set, and limits exact, then verify it with a fresh readback.

Ask only when missing information changes authority, external-effect identity, or duplicate-effect risk and cannot be resolved from available evidence; otherwise proceed with labeled assumptions.

Keep secrets, tokens, authorization codes, signed payloads, and sensitive response bodies out of source, logs, metrics, test fixtures, and error messages. Provider-originated data remains untrusted until the provider's authentication contract has been checked.

This skill owns the unreliable external boundary, including recovery and proof. It does not own:

| Adjacent concern | Required handoff contract |
| --- | --- |
| Public API of this service | Caller deadline and cancellation, local operation identity, pending/final status semantics, and safe client retry behavior |
| Payment meaning or ledger invariants | Allowed state transitions, amount/currency invariants, source of truth, compensation, and terminality |
| Messaging implementation (`reliable-messaging`) | Message identity, ordering key, durable-acceptance point, delivery/ack semantics, and poison-message policy |
| PostgreSQL schema (`postgres-schema-design`) | Required grains, uniqueness, token-refresh coordination, checkpoints, retention, and audit evidence |
| PostgreSQL workload (`postgres-performance`) | Query shapes, expected rates, latency/capacity targets, and maintenance envelope |

Define these handoffs when they are relevant; leave their internal design to the named owner.

## Conditional boundary references

- If the integration sends outbound HTTP, handles transport failures or rate limits, or synchronizes paginated data, read [references/http-resilience.md](references/http-resilience.md) before designing or changing its attempt loop.
- If the task includes OAuth/token lifecycle, scoped credential selection, or rotation, read [references/authentication.md](references/authentication.md). Load it for a pinned static credential only when authentication security is in scope.
- If the provider can push callbacks or webhooks, read [references/webhooks.md](references/webhooks.md) before designing or changing the receiver, replay controls, or asynchronous processing.
- If the result covers an external side effect, partial-progress/checkpoint policy, production readiness or migration, or executable failure proof, read [references/final-review.md](references/final-review.md) after drafting and apply its verifier pass before answering.

## 1. Pin the provider contract

Start from current official provider documentation, the configured SDK/version, captured redacted exchanges, and available sandbox evidence. Populate the boundary ledger for each operation with:

- environment, account/tenant, base URL, endpoint, method, API and SDK version;
- request and response schemas, required headers, size limits, and documented error codes;
- authentication, scopes/audience, credential lifecycle, and rotation mechanism;
- provider idempotency support, key scope, retention, parameter matching, and concurrent-request behavior;
- quota scope, rate signals, concurrency rules, and provider-directed delay;
- callback delivery, signature, event identity, ordering, retry, version, and retention guarantees;
- status lookup, listing, pagination/cursor, export, and sandbox capabilities;
- every unknown or inferred behavior, kept distinct from a documented guarantee.

First decide whether a synchronous integration is necessary. Preserve the caller's stated completion semantics: if named provider-attempt stages must fit its deadline, budget those stages inside it rather than moving them behind an early response. Use locally accepted `pending` plus asynchronous convergence only when the caller contract permits that handoff; reconciliation may outlive the request without redefining what the request promised.

Pin request and callback versions independently where the provider permits. Treat a sandbox as contract evidence for exercised behavior, not proof of production quotas, timing, routing, or downstream side effects.

Use tolerant reading for documented additive fields. Route an unknown required field, state, enum value, signature version, or semantic change to an explicit `incompatible` path with retained evidence and an alert; mapping it to a familiar state would hide contract breakage.

## 2. Establish request identity

Persist the ledger's local operation identity before the first attempt. Derive or associate the provider idempotency key with that operation and reuse it only for byte-equivalent or provider-equivalent intent. Persist the canonical intent/hash, provider account and environment, key scope/expiry, attempt records, and provider request/resource IDs needed for later lookup.

Prevent concurrent workers from creating distinct provider operations for the same local intent. A retry is another attempt of the same operation; a user-requested new effect receives a new identity.

When the provider lacks a documented idempotency guarantee for the side effect, keep automatic retries disabled once transmission might have begun. Require a provider lookup, reconciliation key, or explicit compensation path before choosing another write.

## 3. Execute a bounded attempt

Carry one monotonic end-to-end deadline from the caller or background job through queueing, authentication, DNS/connect/TLS, write, response headers/body, backoff, and parsing. Each attempt receives only the remaining budget, and cancellation reaches the transport and body reader. A reconciliation job may outlive the request, but it has its own deadline and ownership.

Choose retry eligibility from the pinned provider contract, request identity, failure phase, and ability to resolve ambiguity. Bound both attempts and elapsed retry time. Use exponential backoff with jitter to decorrelate callers, honor applicable provider delay signals without exceeding the operation deadline, and constrain concurrency at the provider's quota scope.

## 4. Classify the outcome

Normalize transport, HTTP, provider-code, authentication, schema, and local persistence results into states that drive different recovery:

- `succeeded`: authoritative evidence identifies the provider result;
- `permanent`: the contract says the same intent cannot succeed without a change;
- `retryable`: another attempt of the same identity is explicitly safe and remains inside budget;
- `ambiguous`: transmission might have produced the effect but no authoritative result is known;
- `incompatible`: the observed contract cannot be interpreted safely.

Status code alone does not select a state. For side effects, a timeout, connection loss, truncated response, or some server errors can be ambiguous because the provider may have committed before the response was lost. Keep that state non-terminal until provider lookup, callback, or reconciliation resolves it. Preserve the provider request ID, error code, response metadata, attempt phase, and redacted evidence used for the decision.

A lookup `not found` proves absence only when the provider contract defines its visibility and completeness at that point. Otherwise retain `ambiguous`, name the observation gap, and do not turn absence of evidence into permission for a new write.

## 5. Observe callback or poll completion

Select the cheapest authoritative observation channel the provider actually offers. Use authenticated callbacks for prompt notification, backed by polling or listing when callbacks can be lost. If no callback exists, poll a status resource rather than repeating the original write, and apply its own deadline, cadence, quota, and terminal-state rules.

Join callback and poll results through the local operation identity plus documented provider identifiers. Treat timestamps and delivery order as evidence only to the extent the provider contract guarantees them.

## 6. Reconcile to convergence

Run reconciliation independently of the happy path. Select non-terminal, ambiguous, overdue, failed-callback, and drift candidates using a declared lookback and checkpoint. Query provider truth by a documented stable identifier, compare it with local state, and apply only allowed forward transitions or named repairs. Retain unresolved and incompatible cases for operator action.

Measure reconciliation lag, oldest ambiguity, checkpoint age, scanned/resolved/unresolved counts, and drift by state. These signals reveal lost callbacks and false success even when request latency looks healthy.

## 7. Prove and roll out the boundary

Build the smallest evidence stack that can falsify the design:

1. Contract tests pin requests, responses, error bodies, callback signatures, and versions using redacted fixtures or an official mock.
2. Fault-injection tests exercise each side effect's before-send, after-possible-send, persistence, cancellation, and recovery boundaries; conditional references define protocol-specific cases.
3. Sandbox tests verify the provider behaviors the sandbox can actually exercise and label production-only gaps.
4. End-to-end tests correlate one operation through provider result, callback/poll, reconciliation, and observable signals.

Observe end-to-end and provider latency distributions, attempt count, retry exhaustion, throttle delay, concurrency, quota headroom, ambiguous-outcome age/count, authentication refresh/rotation failures, callback verification/duplicate/lag, schema incompatibility, and reconciliation drift. Use bounded-cardinality dimensions; place operation and provider request IDs in traces or structured logs rather than metric labels. Canary stop signals mirror every in-scope signal class; a healthy quota or latency signal cannot hide authentication, duplicate-effect, ambiguity, incompatibility, or drift failure.

Roll out with pinned versions, sandbox evidence, a side-effect-free or shadow phase where possible, a tenant/operation canary, quota headroom, alerts, and a kill switch for new work. A provider migration assigns each operation to one writer, preserves identity across routing, drains and reconciles in-flight work and callbacks from both providers, and keeps rollback handlers alive. Rollback stops new routing; it does not erase unresolved external effects.

## Report

Lead with readiness and authority state. Render the relevant decision-ledger rows for interacting boundaries; for a narrow task, preserve their fields in the user's requested shape.

Use the report sections as a menu. Preserve the verdict, decision-changing evidence, material gaps, and next action; omit sections that do not apply to the requested depth.

For a narrow task, prefer `verdict -> affected contract -> cause or decision -> smallest correction -> focused proof -> authority/gap`. Use the full table and rollout sections only when several stages or external effects interact.

```markdown
## Verdict
[ready/not ready; designed, changed locally, sandbox-tested, or verified live]

## Boundary ledger
[relevant rows for effects and distinct observation/convergence paths]

## Outcome and recovery
[taxonomy, callback/poll path, checkpoint, drift handling]

## Proof
[executed tests and observed signals; distinguish static validation from behavioral evidence]

## Rollout and rollback
[canary, quotas, migration ownership, stop signals, unresolved in-flight work]

## Handoffs and gaps
[adjacent-owner contracts, missing provider evidence, separate authorizations]
```

Before finalizing, apply the routed verifier pass once. Lead with `not ready` for an exact remaining gap; omit irrelevant fields rather than widening the task.
