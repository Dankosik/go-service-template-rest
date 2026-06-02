# Microlease Test Plan

Status: triggered, review-ready technical-design artifact
Trigger: validation spans money math, persisted state, protected HTTP,
proxy durability, event replay, privacy, performance, and rollout proof.

This is not an implementation task ledger. Planning must carry these proof
obligations into `tasks.md` after technical design review passes.

## Exit Criteria

Implementation readiness later requires proof that:

- no paid execution can start without billing-reserved parent capacity and
  durable proxy child debit lineage;
- active microlease exposure subtracts from visible available balance;
- child and aggregate parent caps are enforced under replay, conflict, outage,
  lag, and over-debit scenarios;
- memory and any future Redis use cannot mint or lose spend authority;
- direct per-request reserve fallback is disabled for migrated paid cohorts;
- APIs, events, logs, traces, metrics, inbox/outbox rows, audit rows, proxy
  local rows, and reconciliation records remain privacy-safe.

## Billing Money And Persistence Proof

Required:

- USD atom parser/formatter and range vectors for every new amount field;
- microlease reserve/release/charge/write-off/reversal ledger vectors;
- property tests for non-negative available balance and exposure conservation;
- idempotency replay/conflict tests for issue/replenish/readback/close;
- account balance row-lock concurrency tests for simultaneous issue/replenish;
- aggregate active exposure cap tests;
- expiry without close proof keeps exposure reserved;
- close proof releases only proven unallocated capacity;
- over-child and over-parent terminal evidence caps customer charge and opens
  reconciliation;
- migration validation for new tables, indexes, constraints, and rollback path;
- SQLC drift checks for query changes.

Planned command families:

- `make sqlc-check`;
- `make migration-validate` or `make docker-migration-validate`;
- targeted Postgres integration tests;
- `make check` / `make check-full` as appropriate for the changed surface.

## Protected HTTP Contract Proof

Required:

- OpenAPI schema validation for protected microlease routes;
- generated code drift proof;
- auth 401 and scope/account 403 behavior;
- idempotency and ambiguous-timeout readback behavior;
- body identifier placement and route-label safety;
- Problem/status mapping for conflicts, stale pricing, insufficient funds,
  admission throttle, manual review, and not ready;
- no raw sensitive payload in request/response schemas, Problems, logs, or
  metrics.

Planned command families:

- `make openapi-check`;
- targeted HTTP adapter and contract guardrail tests.

## Proxy Durable Allocator Proof

Cross-repo proof required before implementation readiness:

- durable grant persistence before spend;
- single-writer row-lock or compare-and-swap allocation per owner/fence;
- child debit and terminal obligation commit before external execution;
- duplicate child ID and changed fingerprint conflict behavior;
- stale fence denial;
- durable store outage fail-closed before execution;
- process restart rebuilds memory cache and does not reuse stale capacity;
- direct reserve fallback disabled for migrated paid cohorts;
- proxy local rows exclude raw prompts, completions, SSE chunks, bearer tokens,
  API keys, DSNs, raw event payloads, and dynamic proof URLs.

## Memory And Redis Proof

First target:

- prove no Redis dependency participates in paid admission;
- prove process memory is cache/precheck only;
- memory precheck success followed by durable allocation failure cannot execute;
- memory loss/restart rebuilds from durable proxy state.

If a later approved design introduces Redis:

- prove Redis is limiter/cache/backpressure only;
- prove rebuild from durable source;
- prove timeout/failover/split-brain degrades to strict/fail-closed;
- prove Redis state cannot create customer-money capacity.

## Event And Worker Proof

Required:

- event schema lint/generate/drift checks after proto introduction;
- proxy outbox writes terminal/checkpoint/close facts from durable source rows;
- billing inbox dedupes duplicate events and conflicts changed fingerprints;
- DB commit before offset commit;
- outbox retry for billing facts;
- out-of-order terminal/checkpoint/close behavior;
- broker outage and consumer lag behavior;
- poison/quarantine path with support-safe metadata;
- stale microlease/debit reconciliation within SLA;
- worker shutdown, bounded retry/backoff, and readiness probes.

## Privacy And Security Proof

Required:

- service auth and route scope tests;
- producer authenticity tests;
- no account IDs or raw request IDs in high-cardinality metrics labels;
- no raw prompts, completions, SSE chunks, bearer tokens, API keys, DSNs,
  payment secrets, raw provider payloads, raw event payloads, dynamic proof
  URLs, or sensitive request bodies in APIs, events, logs, traces, metrics,
  inbox/outbox, audit, proxy local rows, reconciliation, test fixtures, or
  workflow artifacts;
- secret scan and Go security scans when implementation changes code.

Planned command families:

- `make secret-scan` or repository equivalent when available;
- `make gosec` / `make govulncheck` or Docker equivalents;
- privacy-focused unit/integration assertions.

## Performance Proof

Benchmarks must measure:

- active microlease admission with memory precheck and durable child debit;
- active microlease admission without memory precheck;
- cold issue/replenishment latency;
- account contention under same-account concurrent replenishment;
- terminal ingestion throughput;
- checkpoint/close cadence;
- stale reconciliation scan cost;
- first-token impact in proxy path.

Initial budgets from `design/overview.md`:

- billing issue/replenish transaction p95 under 100 ms and p99 under 250 ms in
  local integration benchmark;
- proxy active durable child allocation p95 under 10 ms and p99 under 25 ms in
  local proxy benchmark;
- first-token added latency from active microlease admission p95 under 25 ms;
- cold replenishment added latency p95 under 250 ms and p99 under 500 ms;
- terminal reconciliation opens within 5 minutes of critical breach.

If proof shows the target cannot meet performance without memory-only or
Redis-only spend, reopen specification.

## Rollout Proof

Required:

- default-closed microlease controls;
- shadow/parity for balance/exposure and terminal facts;
- cohort gates;
- no dual writer;
- old proxy writer disablement;
- direct reserve fallback disablement;
- rollback/failback that fails paid admission closed or uses only already-minted
  valid microleases until cutoff/cap;
- operator-visible lag, stale, and reconciliation gates.

## Reopen Targets

Reopen technical design if a proof cannot be represented in tasks without
choosing new data, contract, sequence, worker, or rollout semantics.

Reopen specification if proof requires unbacked memory/Redis spend, direct
reserve fallback, weaker billing PostgreSQL authority, weaker proxy durable
lineage, broader payment/top-up/account/pricing/API-key authority, or weaker
privacy/outage policy.
