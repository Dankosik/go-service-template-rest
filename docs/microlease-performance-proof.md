# Microlease Performance Proof

This proof records the durable billing-issued microlease budget checks used by `specs/event-driven-billing-money-performance-microleases/tasks.md`.

Run billing authority-path proof:

```sh
rtk go test ./test -tags=integration -run '^TestBillingMicroleasePerformanceBudgets$' -count=1 -v
```

Run proxy allocator proof from `/Users/daniil/Projects/GonkaGate/gonka-proxy`:

```sh
rtk bun test src/__tests__/services/billing/microlease/durable-microlease-allocator.performance.test.ts
```

Budgets enforced by these tests:
- billing issue/replenish: p95 under 100 ms, p99 under 250 ms;
- proxy durable child allocation with and without memory precheck: p95 under 10 ms, p99 under 25 ms;
- cold replenishment without Redis or unbacked memory spend: p95 under 250 ms, p99 under 500 ms;
- first-token added latency for the proxy commit-before-execution path: p95 under 25 ms.

The billing integration proof also measures account contention, terminal ingestion, checkpoint cadence, close cadence, and stale reconciliation scan cost. Memory is used only as a cache/precheck in the proxy proof; every successful allocation still commits through the durable repository and returns a terminal obligation before execution can proceed.
