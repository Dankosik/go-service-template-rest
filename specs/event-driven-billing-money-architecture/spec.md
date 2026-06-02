# Event-Driven Billing Money Architecture Specification

Mode: full orchestrated
Status: approved after specification reopen
Date: 2026-06-01
Owner: orchestrator

## Context

This specification decides the production-ready usage-money architecture for
`billing-service` and `gonka-proxy`.

The first approved version selected a protected per-request billing HTTP
reserve before every paid external execution, with Redpanda carrying terminal
facts after reserve. Follow-up technical design review passed that packet, but
the user reopened specification before planning because per-request reserve is
too weak for the target performance envelope.

This reopened specification replaces the per-request reserve target with a
single uniform high-performance paid-request path:

- billing-service mints bounded account-scoped spending leases from PostgreSQL;
- each minted lease reserves USD funds in billing before proxy can spend it;
- `gonka-proxy` admits paid requests only by durably allocating per-request
  debit authorizations from an active billing-minted lease;
- terminal settlement remains asynchronous through durable proxy submission,
  Redpanda transport, billing inbox processing, and billing PostgreSQL ledger
  effects;
- billing-service remains the customer-money authority and caps charges by the
  billing-authorized lease and child debit authority.

The active workflow intentionally reopens the usage architecture question from
`specs/billing-money-core/`. The older `billing-money-core` artifacts remain
historical context for fixed-scale USD money, durable idempotency, holds,
ledger effects, and event-ingestion mechanics. They are not the active decision
authority for this workflow.

Research completed under
`specs/event-driven-billing-money-architecture/research/` found:

- `billing-service` currently publishes no business money OpenAPI surface.
- `billing-service` already has useful PostgreSQL money primitives for
  balances, idempotency, holds, terminal outcomes, ledger entries, and
  reconciliation.
- `gonka-proxy` still owns local balance reservation/deduction in current
  request paths, while also carrying a target TypeBox internal-money contract
  and older shared-balance bridge paths.
- Pure async fire-and-forget usage events cannot protect normal paid usage from
  overspend unless a prior billing-owned authorization already reserved the
  full exposure.
- Redpanda can carry terminal facts, replay, notifications, checkpoints, and
  derived facts, but it cannot be the no-negative customer-money boundary.
- Pricing, account scope, service auth, privacy, outage behavior, cutover, and
  validation obligations must be explicit before technical design.

## Scope / Non-Goals

In scope:

- paid usage spending lease issuance, replenishment, debit authorization,
  terminal settlement, write-off, reversal, compensation, and reconciliation
  architecture for `gonka-proxy` and `billing-service`;
- normal proxy paid-usage families that spend customer money, including
  completion-style inference and web-search-like paid operations when they are
  routed through the shared lease/debit/finalize/write-off model;
- target API and event contract direction for lease issuance, readback,
  terminal settlement, lease checkpoint/close, and billing facts;
- account-scope ownership and money-backed spend-limit evaluation ownership;
- Redpanda scope for current usage architecture;
- PostgreSQL correctness boundary for customer money;
- outage behavior before execution, after execution, during lease
  replenishment, and during broker, billing, worker, and PostgreSQL failures;
- privacy constraints for APIs, events, logs, traces, metrics, inbox/outbox
  rows, audit rows, proxy local durable lease rows, and reconciliation records;
- validation obligations, proof budgets, and later artifact triggers.

Out of scope:

- payment-provider sessions, payment webhooks, customer top-up runtime flows,
  payment presentation sync, payments-service evidence writeback, and Redpanda
  payment-evidence ingestion;
- public OpenAI-compatible `/v1*` route behavior;
- pricing catalog ownership, model routing, devshard execution, transfer-agent
  signing, and API-key lifecycle or policy configuration;
- a branchy model where some paid requests use leases and other paid requests
  use direct per-request reserve because they are "expensive", "risky", or
  otherwise special;
- pure cached balance checks, proxy-local mutable balance authority, Redis
  correctness state, process-local spending allowance, and direct per-request
  billing reserve fallback as target paid-admission paths;
- migration SQL, generated SQL, runtime adapters, workers, tests,
  implementation code, `tasks.md`, and edits to `design/`, `test-plan.md`, or
  `rollout.md` in this specification-reopen session.

## Constraints

- Billing-service is the authoritative USD customer-money source of truth.
- PostgreSQL transactions, unique constraints, durable idempotency, stored
  outcomes, spending leases, child debit authorization lineage, ledger entries,
  reconciliation rows, inbox rows, and outbox rows enforce correctness.
  Redpanda, Redis, proxy-local memory, and proxy-local balance tables do not.
- Normal paid execution must not start until the proxy has durably allocated a
  per-request debit authorization from an active billing-minted lease whose USD
  exposure is already reserved in billing PostgreSQL.
- Lease issuance and replenishment are billing money commands. They may be
  batched and amortized across many paid requests, but they must reserve funds
  in billing before proxy can spend the lease.
- Customer charge must never exceed the child debit authorization cap, and the
  aggregate customer charge for a lease must never exceed the lease's
  billing-reserved USD budget. Excess cost after an external effect may exist
  becomes explicit write-off, compensation, or reconciliation, not retroactive
  overcharge.
- Proxy local durable accounting may prove that the proxy allocated or observed
  a child debit, but it is not visible balance truth and cannot create customer
  money capacity beyond a billing-issued lease.
- If no active lease has enough remaining local authority, the proxy may request
  or wait for a billing replenishment command, but it must not fall back to a
  direct per-request reserve or stale cached balance admission. If replenishment
  cannot be durably accepted by billing in time, paid admission fails closed.
- No raw prompts, completions, SSE chunks, bearer tokens, API keys, DSNs,
  payment secrets, raw webhook bodies, dynamic proof URLs, or unbounded provider
  payloads may appear in APIs, events, logs, traces, metrics, inbox/outbox rows,
  proxy local lease/debit rows, audit rows, reconciliation records, research
  notes, or later workflow artifacts.
- Current evidence is static repository and sibling-repository source/contract
  evidence. No live deployment, production DB, or traffic evidence was used.

## Decisions

### D1. Target Architecture

Decision: use a billing-issued spending lease architecture with asynchronous
terminal settlement as the uniform target path for paid requests.

The target flow is:

1. `gonka-proxy` authenticates the caller, obtains account/policy context, and
   obtains immutable pricing snapshot evidence from `pricing-service`.
2. `gonka-proxy` ensures it has an active billing-issued spending lease for the
   account scope and proxy lease owner. If no lease has enough remaining local
   authority, proxy requests replenishment from billing before admitting more
   paid work.
3. Billing-service issues or replenishes a lease through a protected internal
   command and reserves the lease's USD budget in one short PostgreSQL
   transaction using account balance, durable idempotency, lease state, ledger
   effect, and stored outcome state.
4. `gonka-proxy` persists the lease grant and allocates each paid request by a
   durable local child debit authorization. Allocation must be atomic with
   reducing the local remaining lease authority and creating a terminal
   submission obligation before external execution.
5. `gonka-proxy` may execute the external paid operation only after the child
   debit authorization and terminal submission obligation are durable.
6. After execution succeeds, fails, aborts, times out, or becomes ambiguous,
   `gonka-proxy` records safe terminal evidence against the child debit and
   publishes terminal facts from its durable store.
7. Redpanda is the current target transport for terminal usage facts,
   lease checkpoint or close facts, ingestion rejection facts, reconciliation
   commands, and billing-emitted facts.
8. Billing-service consumes terminal and lease checkpoint/close events through
   a durable inbox, applies idempotent lease/debit settlement semantics, writes
   ledger/outcome/reconciliation state in PostgreSQL, and emits derived billing
   facts through a transactional outbox.
9. Billing-service reconciliation repairs stale leases, missing terminal facts,
   ambiguous debit state, proxy over-debit evidence, delayed or quarantined
   terminal events, and unreleased lease capacity from durable billing and proxy
   lineage.

Rationale:

- Per-request billing reserve has the simplest correctness model but adds a
  synchronous billing round trip and same-account reserve transaction to every
  tiny paid request.
- A billing-issued lease amortizes that synchronous reserve cost across many
  requests while preserving the core money invariant: all spendable lease
  capacity is reserved in billing before proxy can use it.
- The proxy hot path remains simple and uniform: allocate a durable child debit
  from a valid lease, then execute. It does not choose between reserve,
  allowance, cached balance, or request-risk branches.
- Redpanda moves terminal settlement and derived facts out of the proxy request
  path without becoming the money source of truth.

Rejected:

- per-request synchronous billing reserve as the target architecture, because
  it preserves money correctness but does not meet the desired high-performance
  paid-request path;
- fully async fire-and-forget usage events without prior billing lease
  authority, because they allow paid execution before funds are durably bounded;
- branchy admission where some paid requests use cached allowance and others
  use direct billing reserve;
- pure cached balance checks, proxy-local mutable balances, process-local
  reservations, and Redis-backed correctness state;
- Redpanda reserve command/outcome as the normal replenishment gate unless a
  later specification reopen proves it can beat protected HTTP without
  weakening fail-closed lease issuance.

### D2. Spending Lease Money Model

Decision: a spending lease is a billing-owned, account-scoped, bounded,
expiring, fenced authorization whose full USD exposure is reserved in
billing-service PostgreSQL at issuance.

Required lease invariants:

- Lease issuance or replenishment creates billing-owned reserved USD exposure
  before proxy can admit paid requests from that lease.
- Active lease budget subtracts from account available balance exactly like
  other reserved customer-money exposure.
- Each lease has a stable `spendingLeaseId`, account scope, lease owner,
  generation or fencing token, issued amount, remaining unsettled amount,
  expiry/cutoff timestamps, policy versions, pricing basis constraints, and
  stored outcome identity.
- Multiple leases for one account may exist only when billing has independently
  reserved each lease budget. Aggregate active lease exposure participates in
  the account reserved balance and cannot take available balance negative.
- Proxy child debit authorizations are strict children of one lease. Billing
  never charges a child debit above its child cap and never charges aggregate
  children above the lease's reserved cap.
- A lease is not a bearer cash token. It is spend authority only for the
  authorized proxy lease owner, account scope, generation, and permitted use
  class.
- Expired or revoked leases cannot authorize new child debits. Already allocated
  child debits remain terminal settlement or reconciliation obligations.

Ledger semantics:

- Lease issuance moves USD from available to reserved through explicit ledger
  state such as `spending_lease_hold`.
- Child debit settlement finalizes charge, releases unused child capacity, and
  reduces lease reserved exposure without exceeding the lease cap.
- Lease close, expiry, cancellation, or reconciliation releases only capacity
  that billing can prove is unspent or intentionally written off.
- Write-off, reversal, compensation, and operator repair are explicit ledger
  effects. They are never silent balance edits.

### D3. Proxy Local Durable Accounting

Decision: proxy may perform local debit allocation only from a billing-minted
lease, and only through a durable non-authoritative lease ledger.

Proxy durable state must record:

- lease identity, generation/fence, account scope, proxy lease owner, issued
  amount, expiry/cutoff, and billing stored outcome reference;
- local remaining lease authority;
- one child debit authorization per paid request, with
  `debitAuthorizationId`, `usageOperationId`, child cap, operation fingerprint,
  pricing snapshot identity/fingerprint, sequence or monotonic allocation
  position, terminal deadline, and safe caller lineage;
- terminal submission obligation before external execution;
- terminal classification and safe terminal evidence after execution;
- publish/checkpoint state for Redpanda relay;
- recovery state for allocated-before-execution,
  executed-pending-terminal, terminal-lost, expired, redriven, closed, and
  reconciled cases.

Rules:

- Debit allocation must be atomic with reducing the local remaining lease
  authority and creating the terminal submission obligation.
- Process memory may cache lease state only as a performance optimization.
  Recovery after process restart must rebuild from durable proxy state.
- A stopped or restarted proxy must not reuse a stale lease generation or
  allocate a duplicate child debit with changed fingerprint.
- If the durable proxy lease ledger is unavailable, paid admission fails closed
  before external execution.
- Proxy durable rows are proof inputs and recovery anchors. They are not
  customer balance truth and cannot mint additional spend authority.

### D4. Multi-Proxy Concurrency And Fencing

Decision: every spendable lease is fenced to prevent concurrent proxy instances
from spending the same lease capacity.

Spec-level requirements:

- Billing issues each lease to a concrete proxy lease owner or allocator scope.
  Technical design must define the exact owner identity, such as proxy instance,
  proxy shard, or a shared proxy durable allocator.
- Only one durable allocator may spend a lease generation at a time. If several
  proxy processes share one lease, they must coordinate through one durable
  compare-and-swap or row-lock boundary, not process memory.
- Lease renewal or replacement produces a new generation/fence. Stale
  generations cannot allocate new child debits.
- Concurrent proxy instances may each hold separate leases for the same account
  only because billing reserved each lease budget independently.
- Billing settlement validates lease ID, generation/fence, child debit identity,
  account scope, producer authority, and terminal fingerprint before any money
  mutation.
- If billing receives evidence that child debit authorizations exceed the lease
  budget, duplicate a child ID with changed fingerprint, or reference a stale
  fence, it caps customer charge at valid authority and opens reconciliation or
  write-off for the excess.

### D5. API Contract Direction

Decision: `billing-service` OpenAPI is the target source of truth for protected
business money HTTP contracts.

Target protected HTTP capabilities:

- issue or replenish spending lease;
- read spending lease and operation state;
- close or cancel a lease when proxy has durable proof that unused capacity is
  safe to release;
- read primary balance state;
- authorize reconciliation or redrive commands for operator/admin repair.

Rules:

- New money endpoints must be added to `api/openapi/service.yaml` with real
  OpenAPI security and generated server/client surfaces.
- Lease issuance/replenishment commands require idempotency key, immutable
  fingerprint, account scope, requested lease amount, use class, proxy lease
  owner, pricing-policy constraints, expiry/cutoff request, and safe caller
  evidence.
- Same idempotency key plus same fingerprint returns the stored lease outcome.
- Same idempotency key plus changed fingerprint returns conflict and does not
  mutate money.
- A lease issuance timeout after possible acceptance is ambiguous; proxy must
  retry/read back the same command identity and must not mint a separate lease
  for the same requested capacity.
- Current proxy TypeBox internal-money billing contracts are source evidence for
  fields and migration shape, but they are not the target source of truth.
- Older proxy shared-balance bridge paths are compatibility-only. They cannot be
  the target contract and cannot remain a second long-lived contract owner.

Compatibility bridge policy:

- A bridge is allowed only when rollout proves one-step cutover is unsafe.
- The bridge must adapt old proxy call sites to the target lease/debit
  semantics. It must not preserve per-request reserve as a long-lived alternate
  paid-admission path.
- The bridge must be non-authoritative, point toward the target OpenAPI
  contract, and preserve target operation IDs, idempotency keys, fingerprints,
  account scope, stored outcomes, lease/debit lineage, and privacy constraints.
- `rollout.md` must record bridge owner, placement, enablement gates, exit
  criteria, removal/proof tasks, and rollback/failback rules.
- Exit criteria are: proxy uses target billing-service OpenAPI for lease
  issuance/replenishment/readback, proxy-local money writes are disabled for
  migrated cohorts, no dual writer remains, no direct reserve fallback remains,
  old bridge routes are removed or disabled by default, and validation proves
  parity with no divergent balance mutation.

### D6. Money Command Identity And Readback Semantics

Decision: every money-affecting command, lease event, child debit terminal
event, and repair command uses stable business identity, durable idempotency,
immutable fingerprints, and replay-stable readback.

Canonical identities:

- `accountScopeKey`: caller-visible billing account scope key.
- `spendingLeaseId`: billing settlement identity for one lease grant.
- `spendingLeaseGeneration` or `leaseFence`: fencing token for spend authority.
- `proxyLeaseOwnerId`: authorized proxy allocator owner for a lease.
- `debitAuthorizationId`: proxy durable child debit identity under one lease.
- `usageOperationId`: billing settlement identity for one paid usage attempt.
- `clientUsageRequestId`: proxy/client lineage for the usage attempt; not
  settlement truth.
- `request_id`: trace/correlation only; never settlement identity.
- `idempotencyKey`: command replay key scoped by account and operation kind.
- `operationFingerprint`: canonical hash over semantic input, pricing snapshot,
  policy versions, lease/debit identity, and intended operation.
- `terminalFingerprint`: canonical hash over terminal evidence and terminal
  classification.
- `settlementEffectId`: stable external readback identity for committed money
  effects.

Command and event semantics:

- Same idempotency key plus same fingerprint returns the stored outcome.
- Same idempotency key plus changed fingerprint returns conflict and does not
  mutate money.
- Same child debit ID plus same operation/terminal fingerprint replays stored
  child or terminal outcome.
- Same child debit ID plus changed fingerprint is conflict and cannot create a
  second money mutation.
- A terminal event settles against the original lease and child debit authority,
  even if the lease expired or a newer lease was minted later.
- `not_ready` is allowed only for readback or async terminal outcomes that are
  durably accepted but not yet settled. It is not an acceptable paid-admission
  outcome after proxy has no valid local lease capacity.
- Deadline expiration before billing accepts a lease command is no paid
  admission unless another valid lease already exists. Deadline expiration after
  possible acceptance is ambiguous and follows same-identity retry/readback.

Technical design must map these business outcomes to HTTP status codes and
Problem responses without changing the semantics above.

### D7. Account Scope And Spend-Limit Ownership

Decision: billing owns canonical account scope and money-backed spend
aggregates. API-key-service and identity-service remain identity, lifecycle,
and policy authorities.

Day-one account scope:

- The active account key is an opaque billing account scope formatted
  `user:<identity_user_id>` for user-backed customer accounts.
- Organization-ready account scope may be represented as `org:<organization_id>`
  only after an organization authority is contracted. The current usage
  architecture must not require organization behavior to ship user-backed
  billing.
- Billing stores and resolves the canonical account row for the account scope.

Spend-limit split:

- API-key-service owns API-key lifecycle, scopes, and policy configuration.
- `spend_limit_check_required` means the caller must obtain final
  spend/account checks from billing before paid execution.
- Billing owns money-backed leased, reserved, finalized, written-off, and
  pending spend aggregates used for final account spend admission and readback.
- If policy configuration later moves into billing, that requires an explicit
  contract/specification reopen.

### D8. Pricing Snapshot Ownership

Decision: proxy obtains immutable pricing snapshot evidence from
`pricing-service` before lease issuance or child debit allocation; billing
verifies that evidence and fails closed when it is missing, stale, mismatched,
unsupported, or not USD-compatible.

Rules:

- Pricing-service remains the pricing source of truth.
- Proxy is responsible for obtaining lease issuance/replenishment pricing
  evidence and child debit pricing evidence before paid execution.
- Billing lease issuance receives pricing snapshot identity, fingerprint,
  timestamp, expiry, reserve ceiling, policy versions, and USD customer-money
  amount or enough verified evidence to derive exact USD atoms without another
  service call inside the money transaction.
- Child debit terminal settlement carries the original pricing snapshot
  identity/fingerprint and child debit cap.
- Billing must not call pricing-service while holding a database transaction.
- Billing should not add pricing-service as a lease hot-path runtime dependency
  unless technical design records a stronger reason and the workflow reopens
  this decision.
- The current `GNK/USD` versus `GNK/USDT` selector drift is a contract blocker
  for implementation, not permission to accept non-USD customer-money input.
  Contract design must repair or verify the current pricing-service selector so
  billing receives USD-compatible immutable snapshot evidence before planning
  approves implementation.

Reopen condition:

- Reopen specification if pricing-service cannot provide or attest a
  USD-compatible immutable snapshot for customer-money lease decisions.

### D9. Redpanda Scope

Decision: Redpanda is current-scope transport for terminal usage facts, lease
checkpoint/close facts, authorized repair commands, ingestion rejection facts,
and billing-emitted facts. It is not the money authority and it is not
current-scope transport for payment evidence or top-up evidence.

Current consumed event scope:

- usage execution completed terminal fact with lease/debit lineage;
- usage execution failed, aborted, timed out, or write-off-required terminal
  fact with lease/debit lineage;
- lease checkpoint or close fact from the proxy durable lease ledger;
- reconciliation or redrive command only when it references existing durable
  billing state and is authorized as an operator/admin repair path.

Current emitted event scope:

- committed ledger effect facts;
- billing operation outcome facts;
- lease issuance, lease close, and debit settlement outcome facts;
- reconciliation-required facts;
- rejected, conflict, or quarantined ingestion facts.

Out of current scope:

- payment evidence normalized events;
- top-up evidence application;
- payment reversals/refunds;
- bearer spend tokens not backed by billing Postgres lease state.

Rules:

- Every processable event must carry event identity, schema version, event
  fingerprint, producer authority, operation identity, account scope when
  account-scoped, lease/debit lineage when relevant, and safe correlation
  references.
- Billing consumes critical events through a durable event inbox before offset
  commit.
- Billing emits facts through a transactional outbox written with the source
  PostgreSQL state change.
- Broker-level exactly-once semantics are not a correctness guarantee.

### D10. PostgreSQL Correctness Boundary

Decision: billing-service PostgreSQL is the only customer-money correctness
boundary.

Required correctness state:

- account rows and balance rows;
- spending lease rows and child debit authorization lineage;
- durable holds/reserved lease exposure;
- append-only ledger effects;
- durable idempotency records and stored outcomes;
- usage terminal outcomes;
- lease checkpoint/close outcomes;
- reconciliation cases;
- event inbox rows for consumed Redpanda events;
- billing outbox rows for emitted Redpanda facts;
- support-safe audit rows.

Invariants:

- `available_usd_atoms = settled_usd_atoms - reserved_usd_atoms`.
- Available, reserved, settled, and pending customer-money components remain
  non-negative where applicable.
- Lease issuance reserves USD before proxy can locally admit paid usage.
- Child debit authorization subtracts only from the proxy's local durable lease
  capacity; customer balance authority remains billing PostgreSQL.
- Finalize releases unused child capacity and commits a charge no larger than
  the child debit cap and no larger in aggregate than the lease cap.
- Write-off/reversal/compensation is explicit ledger state, not a silent
  balance edit.
- Event offset commit happens only after PostgreSQL stores a durable inbox
  outcome.
- Outbox publish retry cannot create another money effect.

Technical design may choose concrete table deltas from the historical data
model and event-ingestion context, but it must preserve these correctness
rules.

### D11. Terminal Outcome, Lease Close, And Edge Semantics

Decision: each child debit has one terminal money path. Duplicate, late,
racing, or conflicting terminal facts are replayed, rejected, written off, or
routed to reconciliation without a second money mutation.

Rules:

- Final charge is capped by the child debit authorization and the parent lease
  cap.
- If realized cost exceeds the child debit after an external effect may exist,
  billing charges at most the child debit and records the excess as write-off,
  compensation, or reconciliation under the same usage operation.
- If proxy over-debits a lease due to bug, replay, or stale fence, billing caps
  aggregate customer charge at the billing-issued lease budget and opens
  reconciliation for invalid or excess child debits.
- A valid terminal completion event finalizes the child debit once.
- A valid terminal failure/abort/write-off event releases or writes off the
  child debit once.
- Same terminal kind plus same fingerprint returns the stored terminal outcome.
- Same operation plus changed terminal fingerprint is a conflict.
- Completion after committed write-off, or write-off after committed
  completion, opens conflict/reconciliation and does not create a second money
  mutation.
- Missing or invalid parent lease/debit lineage opens ambiguous-terminal
  reconciliation and does not charge customer money beyond verified authority.
- Expired leases do not silently release capacity that may correspond to
  executed work. Unused lease capacity releases only through valid close proof,
  expiry reconciliation, or operator/admin repair.
- Web-search-like paid operations must map into the same
  lease/debit/finalize/write-off/reversal semantics. If an existing web-search
  path needs a separate refund/hold model that cannot preserve these rules,
  specification must reopen before technical design approval.

### D12. Outage, Expiry, And Backpressure Behavior

Decision: paid admission fails closed when no active lease can authorize the
request or when required terminal durability cannot protect customer money.

Behavior matrix:

| Situation | Required behavior |
| --- | --- |
| No active lease capacity before execution | Proxy requests/awaits lease replenishment or fails paid admission closed; no direct per-request reserve fallback. |
| Billing lease issuance unavailable | Existing active lease capacity may be spent until its debit cutoff and cap; no new capacity is minted; exhausted/expired leases fail paid admission closed. |
| PostgreSQL unavailable before lease commit | Lease does not commit; proxy may retry same identity inside deadline or fail closed. |
| Lease issuance accepted but caller times out | Outcome is ambiguous; proxy retries same identity or reads lease state. No duplicate lease identity for the same request. |
| Lease business rejection | No new lease capacity; proxy may use other active valid lease capacity for the same account only if already minted and policy-valid, otherwise fail closed. |
| Proxy durable lease/debit store unavailable | Proxy must fail paid admission closed before external execution. |
| Proxy allocates debit then crashes before execution | Recovery either executes only with durable terminal obligation intact or closes/reconciles the child debit without customer charge. |
| Proxy crashes after execution before publish | Durable terminal row survives restart and retries publish; billing lease exposure remains reserved until settlement or reconciliation. |
| Redpanda unavailable after execution | Proxy records terminal fact durably and retries publish; billing lease exposure remains reserved until settlement or reconciliation. |
| Billing terminal consumer unavailable or lagging | PostgreSQL truth is unchanged; terminal facts remain in broker/proxy outbox, lease exposure stays reserved, and lag/stale-lease alerts and reconciliation apply. |
| Terminal event consumed but processing cannot complete before durable outcome | Do not commit offset; retry by Redpanda redelivery. |
| Terminal event has durable retry outcome and offset committed | Billing inbox retry worker owns recovery from PostgreSQL. |
| Billing outbox relay unavailable | Ledger/outcome truth remains committed; outbox rows retry publishing without changing money. |
| Lease cutoff or expiry reached | No new child debits; open child debits settle or reconcile; unspent capacity releases only after valid close proof or reconciliation. |
| Stale lease/debit exceeds accepted age budget | Open or update reconciliation; fail or backpressure lease replenishment and new paid admission when lag threatens release/write-off obligations. |

Initial budgets for design and validation:

- billing lease issuance database transaction p95 under 100 ms and p99 under
  250 ms in the planned local/integration benchmark workload;
- amortized proxy added latency for paid admission from an active lease p95
  under 10 ms and p99 under 25 ms, excluding external execution and cold
  replenishment;
- cold lease replenishment added latency p95 under 250 ms and p99 under 500 ms
  in the planned benchmark workload;
- terminal event processing database transaction p95 under 100 ms and p99 under
  250 ms excluding intentional same-account contention;
- Redpanda critical terminal event lag warning at oldest unprocessed event age
  above 60 seconds for 5 minutes, critical at above 5 minutes for 5 minutes;
- stale lease/debit reconciliation eligibility no later than 5 minutes after
  lease expiry, child terminal deadline, or configured terminal-lag breach;
- hot replay retention for critical usage terminal and lease checkpoint topics
  at least 14 days;
- emitted billing facts retention at least 30 days or copied to a downstream
  owned store.

Technical design may tighten these budgets. Loosening them requires benchmark
evidence and specification or technical-design review reconciliation.

### D13. Security And Privacy

Decision: money APIs and events require authenticated service identity,
authorized route scopes, producer authenticity, account binding, fenced lease
authority, and privacy-safe payload boundaries.

API security:

- New business endpoints must not inherit the current public `security: []`
  system endpoint posture.
- Protected money routes require real OpenAPI security, authentication
  middleware, service principal identity, route scopes, and 401/403 Problem
  responses.
- Billing must reject caller/account mismatches, unknown account scope,
  unsupported caller principal, missing scope, stale policy evidence, stale
  lease fence, and unauthenticated money commands.

Event security:

- Critical events must come from authorized producer identities, using broker
  ACLs and/or envelope authentication defined in technical design.
- Redrive/replay and reconciliation commands require operator/admin authority
  and durable audit.
- Producer event identity and fingerprints are required for processable events.
  Poison events without valid identity may be quarantined by broker receipt
  coordinates only for safe rejection/readback; they cannot mutate money.
- Lease/debit event payloads are proofs of lineage and terminal state, not
  bearer spend tokens. Logs or support exports must not expose material that can
  be replayed as spend authority.

Privacy:

- APIs/events carry only minimum money operation fields, safe identifiers,
  fingerprints, amounts, account scope, lease/debit lineage, policy versions,
  pricing snapshot identity, terminal classification, and safe evidence
  references.
- Logs and traces use route templates, safe operation IDs, safe failure classes,
  and trace IDs. They must not include request/response bodies or raw money
  payload dumps.
- Metrics use low-cardinality labels only. No account ID, API key, request ID,
  event ID, inference ID, lease ID, debit authorization ID, payment evidence ID,
  or raw provider identifier labels.
- `verifyUrl` or similar evidence locators are SSRF-sensitive. Billing must not
  dereference dynamic URLs unless a later security design restricts them to
  fixed provider allowlists. Prefer provider proof references and fingerprints.
- DLQ/quarantine and reconciliation records store safe error class, event
  receipt identity, and fingerprints, not raw event payload.

### D14. Rollout And Cutover

Decision: rollout must converge on billing-service as the single money writer
and the lease path as the single paid-admission path for migrated cohorts.

Required rollout gates:

- import or map existing proxy balances into explicit billing USD ledger state;
- run shadow readback and parity checks before enabling billing writer cohorts;
- deploy billing lease issuance/readback/close contracts before proxy uses
  leases for paid admission;
- deploy proxy durable lease/debit/terminal store before any cohort can spend
  from billing-issued leases;
- disable proxy-local money writes for migrated cohorts before declaring billing
  authoritative;
- prove direct per-request reserve fallback is disabled for migrated cohorts;
- prove old shared-balance bridge paths are unused, disabled by default, or
  removed for migrated cohorts;
- define rollback/failback so it does not silently diverge dual balance writers
  or reactivate per-request reserve as an alternate target path.

Rollback rule:

- After a cohort uses billing leases as writer/admission authority, rollback may
  fail paid admission closed or continue using already-minted valid leases
  through their cutoff only if terminal durability and reconciliation remain
  healthy. It must not reactivate proxy-local balance mutation or direct
  per-request reserve against the same migrated money scope without
  reconciliation and explicit approval.

## Architecture Comparison

| Concern | Per-request billing reserve | Billing-issued spending lease target |
| --- | --- | --- |
| Paid hot-path latency | Every paid request waits for billing HTTP and a billing reserve transaction. | Most paid requests use a local durable child debit allocation from pre-reserved lease capacity. Billing is contacted for replenishment, readback, close, and repair. |
| No-overspend invariant | Strong: each request reserves before execution. | Strong if and only if billing reserves full lease capacity up front and child charges are capped by child and lease authority. |
| Proxy behavior | Simple but synchronous on every paid request. | Simple and uniform: allocate child debit from valid lease or fail/wait for replenishment. No risk-based branching. |
| Billing load | One reserve transaction per paid request plus terminal settlement. | One lease transaction per replenishment plus terminal settlement. Child requests do not create billing reserve transactions. |
| Failure exposure | Small per-request holds, but many synchronous dependencies. | Larger bounded reserved windows, requiring expiry, fencing, checkpoint, and stale-debit reconciliation. |
| Correctness owner | Billing PostgreSQL. | Billing PostgreSQL. Proxy local durable state is recovery proof only. |
| Main rejection reason | Too slow for many tiny paid requests. | Selected because it preserves billing-authorized funds while amortizing reserve overhead. |

## Formal Clarification Gate Reconciliation

Formal spec clarification challenge was required and rerun because this is a
full-orchestrated protected-domain specification reopen touching customer money,
persisted data, public/internal contracts, distributed event flow, outage
behavior, security/privacy, rollout, and validation.

Lanes used:

- scope and spec coherence;
- domain invariants and edge cases;
- architecture ownership and dependency boundaries;
- API, data, compatibility, and source-of-truth consequences;
- security, reliability, delivery, and validation proof.

Resolution:

| Finding area | Resolution |
| --- | --- |
| Scope boundary | This spec intentionally replaces the prior `D1/D2/D7` per-request reserve target. The prior design/TDR/test/rollout packet is superseded for planning. |
| Uniform path | Direct per-request reserve fallback, pure cached balance admission, and risk-based branchy paid admission are rejected for migrated cohorts. |
| Lease invariant | Lease issuance reserves the full USD lease budget in billing PostgreSQL; child debits are strict children whose aggregate customer charge is capped by the lease. |
| Fencing/concurrency | Every lease has owner and generation/fence semantics. Technical design must choose concrete allocator mechanics, but no two independent writers may spend the same lease generation without durable CAS or row-lock protection. |
| Crash recovery | Proxy must durably store lease grants, child debit allocations, and terminal obligations before execution; restart recovery cannot rely on memory. |
| Delayed terminals | Terminal events settle against the original lease and child debit authority, even after lease expiry or newer lease issuance. |
| API/data authority | Billing OpenAPI owns protected lease issuance/replenishment/readback/close; billing PostgreSQL owns lease/debit/ledger truth; Redpanda event schemas remain under `api/proto/events/v1/*.proto` in technical design. |
| Redpanda scope | Current scope expands from terminal facts only to terminal facts plus lease checkpoint/close facts and related billing facts. Redpanda remains transport, not money truth. |
| Outage policy | Existing active lease capacity may be spent only until cutoff/cap when billing issuance is unavailable; exhausted, expired, revoked, stale-fenced, or unbacked capacity fails closed. |
| Security/privacy | Lease/debit artifacts are not bearer tokens; producer authenticity, service auth, safe fields, route-template logging, and low-cardinality metrics remain mandatory. |
| Validation | Validation must cover lease issuance, local durable allocation, fencing, over-debit, crash recovery, lag, expiry, close, reconciliation, rollout, privacy, and performance. |

No unresolved clarification blocker remains for specification approval. A rerun
is required if technical design changes the uniform lease path, introduces a
direct per-request reserve fallback, moves pricing/account/API-key/payment
authority, weakens proxy terminal durability, weakens billing Postgres money
authority, or weakens outage/auth/privacy policy.

## Open Questions / Assumptions

- [defer_to_design] Exact route names, request/response schemas, Problem
  response codes, event envelope fields, topic names, lease/debit table shape,
  worker packaging, concrete proxy allocator identity, and compatibility adapter
  placement belong to technical design and contract design. They must preserve
  this spec's authority, idempotency, outage, auth, privacy, and validation
  decisions.
- [defer_to_design] Technical design must decide whether lease checkpoint/close
  is carried over protected HTTP, Redpanda, or both. Any chosen path must be
  idempotent, producer-authenticated, privacy-safe, and owned by billing
  PostgreSQL for money mutation.
- [defer_to_design] Technical design must choose exact lease expiry, debit
  cutoff, terminal deadline, close proof, and release semantics. It must not
  silently release capacity that could correspond to executed external work.
- [defer_to_design] Technical design must choose concrete allocator fencing for
  multiple proxy instances, such as per-instance leases, per-shard leases, or a
  shared durable proxy allocator. The selected mechanism must provide durable
  single-writer semantics for each lease generation.
- [reopen_spec_if_false] If pricing-service cannot provide or attest
  USD-compatible immutable pricing snapshot evidence, reopen specification.
- [reopen_spec_if_false] If an existing web-search paid path cannot map into the
  common lease/debit/finalize/write-off/reversal model, reopen specification
  before technical design approval.
- [reopen_spec_if_false] If the target performance envelope cannot be met with
  billing-issued leases and durable local debit allocation without moving
  balance authority to proxy or cache, reopen specification.
- [assumption] No live traffic or production database evidence was needed for
  reopened spec approval. Later design, test-plan, rollout, or validation may
  require live evidence before implementation or release claims.

## Task Breakdown / Handoff Link

Next phase: technical design.

The previous technical design packet, `test-plan.md`, `rollout.md`, and
follow-up technical design review PASS were valid for the superseded
per-request reserve architecture. They are historical context only until
technical design is updated for the lease architecture and receives a fresh
technical design review.

Expected technical-design output:

- repaired task-local `design/` bundle for the approved lease architecture;
- `design/contracts/` or equivalent contract design context for protected
  OpenAPI lease APIs and current-scope Redpanda event contracts;
- data-model delta for spending leases, child debit authorization lineage,
  event inbox/outbox, admission/backpressure state, and any required repair to
  existing money primitives;
- sequence/failure design for lease issuance/replenishment, local durable debit
  allocation, terminal event settlement, lease checkpoint/close,
  reconciliation, proxy durable submission, billing inbox retry, and outbox
  relay;
- ownership map for billing-service, gonka-proxy, pricing-service,
  api-key-service, identity-service, Redpanda, and PostgreSQL;
- updated `test-plan.md` and `rollout.md` or explicit phase routing if those
  are split from technical design;
- mandatory technical design review packet after design repair is complete.

Technical design must not create `tasks.md` or implementation code. Planning is
blocked until technical design and technical design review complete with
`PASS` or eligible `CONCERNS`.

## Validation

Forward-looking validation obligations:

- money math and PostgreSQL invariant tests for lease issuance, replenishment,
  child debit allocation lineage, finalize, write-off, reversal, compensation,
  ledger conservation, non-negative available balance, idempotency
  replay/conflict, lease close/release, and terminal uniqueness;
- API contract tests for protected lease money routes, auth/scope failures,
  account-scope mismatch, missing idempotency, changed fingerprint conflict,
  ambiguous timeout readback, stale pricing, unsupported currency, insufficient
  funds, stored rejection replay, lease exhaustion, lease expiry, and stale
  fence rejection;
- proxy durable lease/debit/terminal proof for local allocation concurrency,
  process restart, persistent store outage, Redpanda outage, billing outage,
  request teardown after external execution, duplicate child debit, changed
  child fingerprint, and no memory-only spend authority;
- event replay tests for duplicate terminal events, changed fingerprint,
  out-of-order terminal before known lease/debit, finalize/write-off race,
  missing lease/debit lineage, crash after DB commit before offset commit, inbox
  retry after committed offset, outbox duplicate publish, quarantine, redrive,
  broker outage, and lease checkpoint/close replay;
- reconciliation tests for stale leases, stale child debits, ambiguous terminal
  state, missing inference evidence, terminal conflict, invalid lease fence,
  proxy over-debit, unreleased capacity, and explicit write-off/compensation;
- privacy/security tests or assertions proving no raw prompts, completions, SSE
  chunks, tokens, DSNs, API keys, raw event payloads, payment secrets, dynamic
  proof URLs, or full provider payloads are logged, traced, stored, emitted, or
  exposed in DLQ;
- performance benchmarks for active-lease paid-admission latency, cold
  replenishment latency, lease issuance transaction latency, local durable debit
  allocation, same-account contention, terminal event processing, Redpanda lag
  handling, stale lease recovery, and proxy first-token impact;
- rollout validation for import/parity, shadow readback, bridge exit,
  proxy-local writer disablement, direct reserve fallback disablement,
  rollback/failback, and no dual writer for migrated cohorts.

`test-plan.md` must be updated before planning because these proof classes are
too broad for inline task notes. `rollout.md` must be updated before planning
because cutover, compatibility, rollback, and mixed-version behavior changed.

## Outcome

Specification reopened and approved on 2026-06-01 with billing-issued spending
leases as the new target architecture.

Implementation, migrations, generated SQL, runtime adapters, tests, task
planning, and technical-design repair have not started in this reopen session.
The workflow is ready for the technical design phase only.
