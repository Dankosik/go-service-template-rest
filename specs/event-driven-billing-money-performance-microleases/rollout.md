# Microlease Rollout Design

Status: triggered, review-ready technical-design artifact
Trigger: rollout, mixed-version behavior, no-dual-writer gates, rollback, and
failback affect customer-money correctness.

This is choreography context, not an implementation task ledger.

## Rollout Principles

- Default closed: no paid cohort uses microleases until billing and proxy gates
  are explicitly enabled.
- Billing PostgreSQL remains the only customer-money authority throughout.
- Migrated paid cohorts have one paid-admission authority: microlease plus
  durable child debit. They must not keep direct reserve fallback as a hidden
  alternate path.
- Active microlease exposure is visible reserved balance immediately.
- Rollback must not create dual writes or unbacked spend.
- Redis is absent from the first target and cannot be introduced silently during
  rollout.

## Sequence

### R1. Expand

Deploy inert billing changes:

- protected HTTP microlease route definitions and handlers behind disabled
  controls;
- microlease tables, indexes, constraints, inbox/outbox, and admission controls;
- worker binary and config with no cohort enabled;
- event protobufs and adapters with producers/consumers disabled or shadow-only;
- telemetry and runbook surfaces.

Safety checks:

- migrations validate;
- OpenAPI/proto/sqlc generated artifacts are in sync;
- admission controls default to `fail_closed`;
- no migrated cohort can receive capacity.

### R2. Proxy Durable Preparedness

Deploy proxy durable grant/debit/terminal/checkpoint storage and bridge code in
shadow/no-spend mode.

Safety checks:

- child debit allocation can be simulated without external execution;
- terminal outbox and checkpoint rows are durable and privacy-safe;
- direct reserve fallback remains the actual production path until cohort gate,
  but migrated cohort gate is still off;
- shadow rows reconcile with billing readbacks without mutating money.

### R3. Shadow And Parity

Run shadow issue/readback calculations for selected internal accounts without
spend authority.

Safety checks:

- computed caps, safety floors, TTL/cutoff, and strict reasons match design;
- active exposure shadow does not mutate balances;
- pricing snapshot evidence is present and immutable enough for lineage;
- API-key attribution with `spend_limit_check_required` flows to final
  spend/account checks.

### R4. Limited Cohort Enablement

Enable microlease issuance for a tiny internal cohort with conservative caps:

- per-microlease cap at or below 1.00 USD;
- per-account active exposure cap at or below 2.00 USD;
- TTL 30 seconds;
- cutoff 25 seconds;
- terminal/reconciliation gates active;
- Redis disabled.

Safety checks:

- no direct reserve fallback for enabled cohort;
- no old proxy-local money writer for enabled cohort;
- balance visible available subtracts active microlease exposure;
- terminal facts settle or keep exposure reserved;
- close releases only proven unallocated capacity;
- stale exposure opens reconciliation within SLA.

### R5. Expand Cohorts Gradually

Increase cohorts only after the previous gate proves:

- performance budgets hold;
- terminal lag and stale counts stay under warning budgets;
- reconciliation case rate is understood;
- privacy checks pass;
- support readbacks explain every balance change;
- no direct reserve fallback or dual writer is observed for migrated cohorts.

### R6. Contract Legacy Paths

After stable migrated cohorts:

- remove or disable old direct reserve/finalize/write-off path for those cohorts;
- keep non-migrated cohort behavior separate and explicit;
- contract bridge code only after parity and support readbacks prove no hidden
  dependency remains.

## Rollback And Failback

Before a cohort is migrated:

- rollback can disable microlease controls and return to the old path because
  no production money authority moved.

After a cohort is migrated:

- rollback must not re-enable direct reserve fallback for in-flight paid work;
- paid admission may continue only with already-minted valid microleases until
  cap/cutoff/local health gates are exhausted;
- no new microlease capacity is issued while rollback state is uncertain;
- exhausted/expired/stale capacity fails paid admission closed;
- terminal settlement and reconciliation must continue until all active
  exposure is settled, released, or explicitly repaired;
- old proxy writer remains disabled for migrated cohorts unless specification
  reopens and accepts a new authority model.

## Operator Gates

Operators need visible safe controls for:

- global microlease enablement;
- cohort enablement;
- per-account or per-owner strict/fail-closed state;
- max microlease cap and active exposure cap;
- terminal lag warning/critical thresholds;
- stale child/debit thresholds;
- reconciliation backlog thresholds;
- emergency disable of new issue/replenish;
- readback of active exposure, unresolved child cap, and close/release state.

Operator notes and audit fields must be support-safe.

## Mixed-Version Compatibility

Mixed versions are allowed only while:

- billing can deny new capacity by default;
- proxy version advertises support for durable grant/debit/terminal/checkpoint
  semantics before cohort enablement;
- event schema versions are compatible and validated;
- old proxy path and new microlease path are not both writers for the same
  migrated account/cohort;
- billing readbacks can distinguish shadow, enabled, closing, and rollback
  states.

## Rollout Evidence

Required before broader rollout:

- migration validation evidence;
- generated drift evidence;
- protected HTTP auth/status evidence;
- event inbox/outbox replay evidence;
- proxy crash/restart evidence;
- no-direct-reserve-fallback evidence for migrated cohorts;
- no-dual-writer evidence;
- privacy evidence;
- performance benchmark evidence;
- support readback and reconciliation evidence.

## Reopen Targets

Reopen technical design if rollout needs a different migration sequence,
mixed-version compatibility model, rollback/failback behavior, Redis dependency,
or operator control surface.

Reopen specification if rollout requires direct per-request reserve fallback,
memory-only/Redis-only spend, weaker billing authority, weaker proxy durable
lineage, or broader service ownership.
