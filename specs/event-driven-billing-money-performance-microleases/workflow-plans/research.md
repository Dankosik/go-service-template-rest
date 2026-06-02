# Research Phase Plan And Completion

Phase: research
Status: complete
Date: 2026-06-01
Owner: orchestrator
Parent workflow: `../workflow-plan.md`

## Scope

Research high-performance prepaid billing admission patterns for GonkaGate:
escrow, bounded counters, microleases, token buckets, Redis/in-memory spend
limiters, durable proxy checkpoint batches, async terminal usage events,
idempotent ingestion, and reconciliation.

This phase does not decide the final architecture. It preserves evidence for
the next specification phase.

## Mode

Research mode: fan-out plus local orchestrator synthesis.

Parallelism:

- L1-L4 were read-only subagent lanes run in parallel.
- L5 reliability was optional and closed after timeout; the orchestrator covered
  reliability controls locally from primary sources and other lane evidence.
- L6 local synthesis consumed all available lane results and external sources.

## Research Lanes

| Lane | Execution | Role | Skill | Owned question | Evidence target | Status |
| --- | --- | --- | --- | --- | --- | --- |
| L1 | subagent | performance-agent | no-skill | Which patterns reduce paid hot-path latency without removing bounded prepaid authority? | Escrow examples, Envoy/Redis token buckets, sharded counters, async metering throughput, checkpointing. | complete |
| L2 | subagent | distributed-agent | no-skill | What consistency model makes microleases safe under concurrency, replay, and partitions? | Escrow, bounded counters, leases/fencing, invariant confluence, Kafka/Redpanda EOS scope, outbox/inbox. | complete |
| L3 | subagent | domain-agent | no-skill | What business invariants and controls are required for bounded write-off and low-balance/risky modes? | Prepaid billing invariants, exposure caps, terminal lag gates, abuse mode, strict mode. | complete |
| L4 | subagent | data-agent | no-skill | Which storage surfaces may be authoritative versus cache/proof/projection? | Postgres ledger, Redis, in-memory buckets, Redpanda/Kafka, ClickHouse, proxy durable rows, dedupe stores. | complete |
| L5 | subagent | reliability-agent | no-skill | What reliability and fail-closed controls are required under crashes, Redis loss, broker lag, duplicates, and abuse? | Redis durability/replication, Kafka/Redpanda semantics, outbox/inbox, Stripe/Metronome/OpenMeter ingestion, Envoy local/global controls. | closed without report after timeout |
| L6 | local | orchestrator | research-session | Are the evidence and lanes sufficient to hand off to specification? | All lane reports, repo artifacts, external primary sources, preserved research notes. | complete |

## Source Coverage

Primary or official sources used:

- O'Neil escrow transaction paper.
- Bounded counter paper and coordination-avoidance paper.
- Redis rate limiter, persistence, and replication documentation.
- Envoy and Envoy Gateway local/global rate limiting documentation.
- Kafka producer and Kafka Streams exactly-once documentation.
- Redpanda transaction and idempotent producer documentation.
- Stripe Meter Events, Meter Event Stream, and rate-limit documentation.
- Metronome usage event and idempotency documentation.
- OpenMeter Kafka/ClickHouse ingestion documentation and architecture write-up.
- DynamoDB atomic counter and conditional write documentation.
- Firestore distributed counter documentation.
- AWS transactional outbox, Debezium outbox event router, and idempotent
  consumer pattern references.
- Repository docs and existing money architecture artifacts listed in the
  parent workflow context bundle.

## Fan-In Path

Fan-in artifacts:

- `../research/source-notes.md`
- `../research/pattern-catalog.md`
- `../research/architecture-options-matrix.md`
- `../research/risk-control-matrix.md`
- `../research/fan-in-synthesis.md`

Orchestrator synthesis rules:

- Subagent output is evidence, not authority.
- Research recommendations are not `spec.md` decisions.
- The next phase must reconcile evidence into accepted/rejected decisions and
  proof obligations.
- Missing live traffic, benchmark, or write-off budget data is an evidence
  limit, not a blocker for starting specification.

## Completion

Research questions handled:

1. Known patterns: handled in `pattern-catalog.md`.
2. High-RPS usage billing without per-request reserve: handled in
   `source-notes.md`, `architecture-options-matrix.md`, and
   `fan-in-synthesis.md`.
3. Redis/in-memory acceptability: handled in `source-notes.md`,
   `architecture-options-matrix.md`, and `risk-control-matrix.md`.
4. Business risk controls: handled in `risk-control-matrix.md`.
5. Architecture variants: handled in `architecture-options-matrix.md`.
6. Recommended next specification direction: handled in
   `fan-in-synthesis.md`.

Blockers:

- None for starting specification.

Evidence limits:

- No live traffic or production DB evidence.
- No benchmark data for active-lease admission, cold replenishment, Redis
  shared-bucket cost, or checkpoint cadence.
- No product-approved write-off/exposure budget.
- Optional reliability lane did not return a standalone report before timeout;
  local source research and other lanes covered the reliability controls needed
  for specification start.

## Stop Rule

Research phase complete. Stop before specification, technical design, planning,
tasks, implementation, validation, migrations, schemas, generated artifacts, or
tests.

Next action: specification.
