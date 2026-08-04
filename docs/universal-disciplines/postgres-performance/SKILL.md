---
name: postgres-performance
description: "PostgreSQL performance optimization and diagnosis through a tight evidence loop. Use when PostgreSQL performance or capacity is the task: slow queries or regressions; high CPU, I/O, locks, connections, WAL, replication lag, vacuum/bloat, or temp-file load; or requests to reduce database load or improve latency, throughput, or capacity through SQL, indexes, planner statistics, pooling, maintenance, configuration, or architecture."
---

# PostgreSQL Performance

Use a **tight evidence loop**:

`baseline -> bottleneck and source -> falsifiable cause -> smallest intervention -> comparable delta`

Choose only the unresolved links needed for the user's decision. Evidence that
places the constraint outside PostgreSQL is a valid result. Without a baseline,
return a bounded collection plan instead of settings.

Optimization lowers end-to-end workload cost while preserving correctness.
Account for writes, tail latency, replicas, and maintenance when they can change
the decision.

## Evidence bar

- A diagnosis states the decisive observations, then ranks the leading
  bottleneck and workload source against the strongest competing explanation.
- A recommendation connects the smallest intervention to a falsifiable cause
  and names its cost, failure signal, and rollback.
- A proven optimization has fresh comparable evidence for the target and
  protected invariants.

When evidence stops earlier, report the exact missing link and next bounded
probe. Keep correlation labelled as correlation until a causal probe supports it.

## Authority

- For answer, review, diagnose, or plan requests, preserve database state while
  inspecting available code, plans, telemetry, logs, and bounded read-only probes.
- For change, fix, or optimize requests, make in-scope local changes and run
  non-destructive validation. Production DDL/DML, configuration changes,
  restarts, session cancellation, maintenance, and load tests each require
  explicit authority.
- Keep an exact production authorization exact and verify it with fresh readback.

Ask only when missing information changes the authority boundary or makes the
next probe unsafe. Otherwise proceed, label gaps, and continue safe in-scope work
until the evidence bar or a concrete blocker is reached.

## Frame the decision

Pin the user-visible target, protected invariants, environment, workload, time
window, and one named repeatable baseline. Prefer an application SLO or business
operation over a generic goal such as lower CPU.

## Attribute the bottleneck and source

Read the relevant section of [references/diagnostics.md](references/diagnostics.md)
before choosing a live probe or interpreting supplied evidence. For a bounded
programmatic collection across several independent sources, read
[references/tool-routing.md](references/tool-routing.md).

Use time-aligned database, host, and application evidence. Compare deltas within
one statistics epoch. Rank work by total impact across the window while retaining
tail evidence, representative bind values, and selectivity. Normalized SQL can
hide parameter skew; averages can hide tail regressions.

## Test the cause

Test the leading explanation against its strongest competitor with the cheapest
safe discriminating probe. State the prediction before probing:

`If <cause>, then <bounded probe> changes <observable>; falsified by <result>.`

For a target query, inspect the real SQL, schema, indexes, statistics,
representative parameters, and plan. `EXPLAIN ANALYZE` executes the statement;
use it only when execution is safe and representative. Change one variable at a
time.

## Choose the smallest intervention

After the evidence supports a cause, read the matching branch in
[references/interventions.md](references/interventions.md). Verify version- or
provider-specific behavior against current primary documentation.

Stop at the first rung that meets the target:

1. remove unnecessary calls, rows, columns, round trips, or transaction time;
2. repair query shape, index fit, or planner statistics;
3. reduce contention and bound connections or concurrency;
4. tune maintenance or a narrowly evidenced resource setting;
5. change schema, partitioning, replication, caching, or hardware.

For the selected candidate, state causal fit, expected metric movement,
correctness constraints, write/storage/operational cost, rollout, failure signal,
and rollback. Prefer a session- or table-local experiment to a global setting.

## Prove the delta

Use the same data, parameters, concurrency, cache conditions, observation window,
and stopping rule before and after, or record every difference. Declare duration
or repetitions before the run and report the distribution rather than the best
sample.

Measure correctness, the target metric, and the relevant resource and operational
costs introduced by the change. Include p95 and p99 for latency or capacity
claims. A single `EXPLAIN ANALYZE` contributes execution evidence; a
representative application replay or custom `pgbench` script is required for a
capacity claim.

## Report

Lead with the verdict. Include only decision-changing evidence, the cause and
strongest competitor, the intervention or next probe, proof state, material
caveats, and authority gaps. Distinguish proposed, locally tested, applied, and
verified-live changes. Separate observation from inference and bound conclusions
to the measured window. For a narrow diagnosis, use `verdict -> decisive evidence
-> cause/competitor -> one probe or action -> gap`; expand only when the requested
decision needs it. Summarize raw artifacts by reference.
