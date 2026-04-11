# Workload Profile And Input Shape

## Behavior Change Thesis
When loaded for symptom "the proof depends on traffic mix, input shape, or cache/dependency state," this file makes the model choose representative workload buckets and labels instead of likely mistake median-only fixtures, hand-written happy-path payloads, or combined benchmark results that hide regressions.

## When To Load
Load when a performance contract needs traffic mix, payload size, row counts, cardinality, tenant skew, hot keys, cache warm/cold/cache-down state, retry shape, or benchmark fixture boundaries.

## Decision Rubric
- Prefer production-derived buckets when safe logs, traces, metrics, or customer-shape data exist.
- Use synthetic representative data only when the spec states why the size buckets, distributions, and concurrency are realistic enough.
- Use worst-case envelopes when the API or product contract allows large inputs that must remain within budget.
- Keep workload labels separate for dimensions that decide correctness: tenant size, cache state, dependency mode, payload size, and concurrency.
- Reject median-only proof for tail-sensitive paths unless median behavior is explicitly the contract.
- Treat idle-replica or off-peak profiles as weak evidence for peak-hour budgets.

## Imitate
- Account summary uses `small`, `typical`, `large`, and `hot_tenant` fixtures: 10, 250, and 5,000 transactions, plus one tenant owning 40% of rows. Copy the separate labels and the refusal to hide large-tenant behavior in an aggregate.
- Catalog read has `warm`, `cold`, and `cache_down` runs. Warm preloads the exact tenant, locale, and version dimensions. Copy the cache-state definition, not just the benchmark names.
- Queue worker capacity includes arrival rate, burst duration, average and p99 job payload size, retry rate, and poison-message rate. Copy backlog age as the acceptance signal when jobs/second alone is misleading.

## Reject
- A one-item slice benchmark for an endpoint with `page_size=200` and large tenants that routinely fill the page.
- A "representative" JSON payload with no optional fields, no large strings, no high-cardinality dimensions, and no rejected-but-validly-parsed negative cases.
- One benchmark that mixes tenants, cache states, and payload sizes, then reports only the combined result.
- A production profile from an idle replica used as proof for peak-hour performance.

## Agent Traps
- Copying fixtures from unit tests that were designed for correctness, not performance shape.
- Choosing small/typical/large names without numeric boundaries.
- Ignoring cache-down, retry, cold start, or dependency-degraded modes because they are "operational" rather than performance concerns.
- Reporting geomeans or aggregate deltas when the acceptance decision is per bucket.

## Validation Shape
Use workload-separated proof obligations:

```bash
go test -run='^$' -bench='BenchmarkAccountSummary/(small|typical|large|hot_tenant)$' -benchmem -count=20 ./internal/accounts > workload.txt
benchstat workload.txt
go test -run='^$' -bench='BenchmarkCatalog/(warm|cold|cache_down)$' -benchmem -count=20 ./internal/catalog > cache-shape.txt
benchstat cache-shape.txt
```

For production-derived profiles, require the sampling window, traffic class, replica selection, and sanitization/privacy assumption.
