# Benchmarking

Use the narrowest benchmark that measures the accepted performance claim. This
repository provides one path for each normal proof level:

| Claim | Harness | Command |
| --- | --- | --- |
| Local CPU, allocations, or contention | Go `testing.B` | `make bench`, `make bench-compare`, `make bench-profile` |
| In-process handler, middleware, or composed router cost | `testing.B` plus `net/http/httptest` | the same Go benchmark commands |
| Real PostgreSQL query, repository, or pool cost | `testing.B` plus the existing Testcontainers integration seam | `make bench-db`, `make bench-db-compare` |
| Client-visible HTTP latency, error rate, and offered load | digest-pinned Grafana k6 in Docker | `make bench-http` |

Runtime telemetry is still required when production traffic, dependency state,
GC behavior, or capacity can materially change the conclusion. A benchmark is
controlled evidence, not a replacement for correctness tests or production
SLIs.

## Define the workload before measuring

Record the measured operation, representative fixture buckets, concurrency or
arrival model, warm/cold dependency state, machine or Docker testbed, and the
latency, throughput, allocation, or capacity budget. Keep materially different
input sizes, cache states, row counts, tenant shapes, and concurrency levels in
separate named cases; a combined average can hide a regression.

Use the same source revision, toolchain, power mode, dependency versions,
fixture shape, and benchmark parameters for baseline and candidate. Run on an
idle machine and do not keep rerunning until a desired statistical result
appears. Give each evidence set a stable workload identifier: a short name for
the fixture, state, and operation that can be compared without recording
secrets.

## Go and in-process HTTP benchmarks

Put `BenchmarkXxx` beside the package and code it measures. Use `B.Loop` for
new benchmarks:

```go
func BenchmarkParse(b *testing.B) {
	input := []byte(`{"id":"example"}`)
	b.ReportAllocs()

	for b.Loop() {
		if _, err := Parse(input); err != nil {
			b.Fatal(err)
		}
	}
}
```

Setup before `B.Loop` is excluded from timing. Keep per-operation construction
inside the measured interval when it belongs to the production operation.
Reset mutable state deliberately instead of allowing later iterations to
measure different data.

For HTTP, select one of two boundaries:

- benchmark a handler, middleware chain, or `internal/infra/http.NewRouter`
  with `httptest.NewRequest` and `httptest.NewRecorder` when process scheduling,
  sockets, TLS, and client behavior are outside the claim;
- use `httptest.NewServer` and its real `http.Client` when the claim includes
  loopback transport and response-body handling but does not require an
  independently deployed service.

Build a fresh request and recorder whenever the handler consumes or mutates
them. If request/recorder creation is test-harness setup rather than the
operation, bracket only `ServeHTTP` with `b.StartTimer` and `b.StopTimer`.
Otherwise leave creation timed. Always validate the response status and the
smallest correctness invariant outside the timed interval.

Use `RunParallel` only when concurrent callers are the accepted workload. It
does not prove service capacity, production worker bounds, or safe lifecycle
behavior.

### Capture and compare

The default run matches every Go benchmark, records allocations, repeats each
case 10 times, and saves raw output and environment metadata:

```bash
make bench

make bench \
  BENCH_PACKAGE=./internal/app/orders \
  BENCH_PATTERN=BenchmarkCalculateTotal \
  BENCH_WORKLOAD_ID=orders-100-lines \
  BENCH_COUNT=20 \
  BENCH_TIME=2s
```

Capture both sides on the same testbed:

```bash
make bench-baseline \
  BENCH_PACKAGE=./internal/app/orders \
  BENCH_PATTERN=BenchmarkCalculateTotal \
  BENCH_WORKLOAD_ID=orders-100-lines
# Apply the candidate change.
make bench \
  BENCH_PACKAGE=./internal/app/orders \
  BENCH_PATTERN=BenchmarkCalculateTotal \
  BENCH_WORKLOAD_ID=orders-100-lines
make bench-compare
```

`make bench-compare` uses the pinned Go `benchstat` tool and fails when the
recorded package, pattern, count, benchmark time, build tags, Go environment,
workload identity, dependency image, schema fingerprint, CPU identity/count, or
`GOMAXPROCS` differ. Artifacts live under
`.artifacts/bench/`:

- `baseline.txt` and `current.txt`: raw Go benchmark output;
- adjacent `.meta` files: revision, dirty state, settings, and Go environment;
- `comparison.txt`: the `benchstat` report.

`BENCH_COUNT=10` is a comparison-ready default, not an acceptance policy.
Statistical significance also does not establish that a delta is materially
important; judge it against the accepted budget.

The simple baseline-then-candidate sequence is sufficient for stable, material
deltas. For small or noisy decision-critical deltas, avoid run-order bias:
prepare two clean worktrees or prebuilt binaries, alternate one baseline and
one candidate batch on the same idle testbed, retain every raw batch, and give
the aggregated baseline and candidate inputs to `benchstat`. Use at least 10
samples per side and prefer 20 when the decision is sensitive; do not combine
samples from different machines or environments.

## PostgreSQL benchmarks

Put real-PostgreSQL benchmarks under `test/`, use `package integration_test`,
and start the file with `//go:build integration`. Reuse
`runPostgresContainer`, the digest-pinned PostgreSQL 17 Testcontainers
dependency, migration owner, and pool construction instead of adding a mock
database or a second container harness.

The benchmark must:

1. start PostgreSQL, apply migrations, create the pool, seed the declared
   fixture size, and run `ANALYZE` before the timed loop;
2. separate row-count, selectivity, warm/cold, and pool-concurrency cases by
   sub-benchmark name;
3. keep transaction creation, network round trips, row decoding, and commit in
   the timed interval when production owns them;
4. reset write fixtures outside the timed interval, or use a fresh transaction
   and rollback when rollback is not part of the claimed production cost;
5. enforce a per-operation or benchmark-level timeout and verify the smallest
   result invariant;
6. close rows, transactions, pools, contexts, and the container through
   `b.Cleanup`.

Run and compare:

```bash
make bench-db-baseline \
  BENCH_DB_PATTERN=BenchmarkOrderRepository \
  BENCH_DB_WORKLOAD_ID=orders-100k-warm
# Apply the candidate change.
make bench-db \
  BENCH_DB_PATTERN=BenchmarkOrderRepository \
  BENCH_DB_WORKLOAD_ID=orders-100k-warm
make bench-db-compare
```

The target sets `REQUIRE_DOCKER=1`, uses the `integration` build tag, fails if
Docker, `BENCH_DB_WORKLOAD_ID`, or a matching benchmark is unavailable, and
writes results under `.artifacts/bench/db/`. Metadata records the exact
PostgreSQL image digest, a fingerprint of `env/migrations`, and the named
fixture/workload. The infrastructure check keeps the Testcontainers and
Compose images identical and digest-pinned.

Use `EXPLAIN (ANALYZE, BUFFERS, WAL, FORMAT JSON)` only as a diagnostic plan
artifact for a representative query. `EXPLAIN ANALYZE` executes the statement,
adds measurement overhead, and does not replace the repeated client-visible
benchmark. Run write plans in a rolled-back transaction or disposable
database. Do not extrapolate a plan from toy data or stale statistics.

## External HTTP load benchmarks

The repository pins Grafana k6 by version and image digest and provides
`test/performance/http/single-flow.js`. The starter scenario:

- supports steady `constant-arrival-rate` and staged
  `ramping-arrival-rate` open-model executors;
- checks the expected response status;
- applies an explicit request timeout;
- fails on the configured latency percentile, error-rate, correctness, or
  dropped-iteration budget;
- discards response bodies after the HTTP layer has consumed them;
- writes the aggregate summary, console log, and run metadata under
  `.artifacts/bench/http/`.

The latency threshold uses the custom `flow_duration` trend around the complete
`http.request` call. Unlike k6 `http_req_duration`, this wall-clock boundary
also includes time spent blocked, connecting, and negotiating TLS. The summary
records the workload identifier, executor, stages, timeout, percentile, and
budget without copying header values or the request body.

Copy the ready input template, then replace every workload value. Keep secrets
and environment-specific values in the ignored `.env.bench` file:

```bash
cp test/performance/http/single-flow.env.example .env.bench
```

The example values prove runner wiring only; they are not template SLOs.

Start the service and its representative dependencies, seed the named fixture,
then run:

```bash
make bench-http
```

Use the steady executor for smoke, steady-load, and soak workloads by setting
`HTTP_BENCH_RATE` and `HTTP_BENCH_DURATION`. Use the ramping executor for
stress and spike shapes by supplying a starting rate and compact JSON stages;
the rate and duration fields from the example file are then ignored:

```bash
HTTP_BENCH_START_RATE=0 \
HTTP_BENCH_STAGES_JSON='[{"target":20,"duration":"1m"},{"target":50,"duration":"2m"},{"target":0,"duration":"30s"}]' \
make bench-http
```

Every stage target is iterations per second because `timeUnit` is fixed at
`1s`. A stage duration of `0` is allowed for an instantaneous spike transition.
Pre-allocate enough VUs for the highest target before measurement.

The Docker-host address in the example reaches a service running on the host.
For a service on a Compose or Docker network, set
`HTTP_BENCH_DOCKER_NETWORK=<network>` and use its container DNS name in the base
URL. Do not put credentials in the URL. Header values and request bodies are
not copied into run metadata, but they still belong only in ignored local or
secret-managed configuration.

For secret injection without a file, export the same variables and run
`make bench-http HTTP_BENCH_ENV_FILE=`. Direct environment values override
matching values from the environment file.

The aggregate summary is the default evidence because per-sample streaming can
consume disk and distort the load generator. Enable it only for diagnosis:

```bash
make bench-http HTTP_BENCH_RAW_SAMPLES=1
```

That opt-in writes compressed `samples.json.gz`; it never silently enables
unbounded plain JSON output.

The generic scenario supports one request with an optional
`HTTP_BENCH_BODY` and JSON headers. When correctness depends on login, setup,
multiple requests, extracted identifiers, or weighted traffic classes, add a
feature-owned k6 scenario under `test/performance/http/` and select it with
`HTTP_BENCH_SCRIPT=<path>`. Keep scenario setup outside the measured default
function, tag each flow, and give each decision-relevant flow its own threshold.

Do not use `/health` or `/metrics` as a proxy for business-path capacity. Do not
add `sleep` to an arrival-rate scenario; the executor owns request arrival.
Pre-allocate enough VUs before the measured run; the starter deliberately does
not allocate more VUs during measurement because that work can skew load
generator results. Treat dropped iterations as a failed offered-load claim,
then distinguish insufficient VU allocation from service saturation before
changing the service.

For decision-grade high load, run k6 on a separate stable host and monitor its
CPU, memory, swap, and network along with the service. Keep roughly 20% load
generator CPU idle and reject a run with swap use or saturated generator
network; otherwise the generator, not the service, may be the bottleneck.

## Diagnose a measured delta

Collect one symptom-matched Go diagnostic from one concrete package:

```bash
make bench-profile \
  BENCH_PACKAGE=./internal/app/orders \
  BENCH_PATTERN=BenchmarkCalculateTotal \
  BENCH_PROFILE=cpu
```

`BENCH_PROFILE` accepts `cpu`, `memory`, `block`, `mutex`, or `trace`. Profiles
and traces explain a measured delta; they are not pass/fail gates:

- CPU attributes active compute time;
- memory `alloc_space` exposes allocation churn, while `inuse_space` exposes
  retained live heap;
- block exposes time waiting on channels, synchronization, and blocking
  operations;
- mutex exposes lock contention;
- trace exposes scheduler, goroutine, GC, network, and syscall timing.

The profile command prints the matching inspection command, including both
memory views. Use `-benchmem` results for bytes and allocations per operation;
use production telemetry when the claim depends on real GC or heap behavior.
For database work, pair client profiles with pool statistics and a
representative query plan when the bottleneck may be server-side.

Do not expose `/debug/pprof` merely for this workflow. A runtime profiling
endpoint needs its own access-control and operational decision.

## Completion and CI policy

A performance claim closes only when:

- independent correctness proof is green;
- the benchmark boundary and workload match the claim;
- baseline and candidate evidence came from equivalent environments;
- the result satisfies the accepted material-change or absolute budget;
- any service-level claim also has the required saturation and runtime evidence.

These targets are explicit evidence commands, not default CI gates. Shared
GitHub-hosted runners are not a stable blocking regression oracle. Add
persistent history or blocking automation only after the benchmark is stable,
a dedicated comparable testbed exists, and a named owner has approved the
threshold and response to failure.

When benchmark infrastructure itself changes, run:

```bash
make benchmark-infra-check
```

This fail-closed smoke executes the Go capture and pinned `benchstat` comparison
path, verifies PostgreSQL image provenance, then has pinned k6 inspect both the
steady and ramping scenario configurations. It proves wiring and validation
only; its deliberately tiny stdlib samples are not performance evidence and it
does not need a running service.

## References

- Go [`testing` benchmarks and `B.Loop`](https://pkg.go.dev/testing#hdr-Benchmarks)
- Go [`benchstat`](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat)
- Grafana k6 [constant arrival rate](https://grafana.com/docs/k6/latest/using-k6/scenarios/executors/constant-arrival-rate/)
- Grafana k6 [ramping arrival rate](https://grafana.com/docs/k6/latest/using-k6/scenarios/executors/ramping-arrival-rate/)
- Grafana k6 [arrival-rate VU allocation](https://grafana.com/docs/k6/latest/using-k6/scenarios/concepts/arrival-rate-vu-allocation/)
- Grafana k6 [built-in HTTP metrics](https://grafana.com/docs/k6/latest/using-k6/metrics/reference/)
- Grafana k6 [result outputs](https://grafana.com/docs/k6/latest/results-output/)
- Grafana k6 [large-test generator sizing](https://grafana.com/docs/k6/latest/testing-guides/running-large-tests/)
- Grafana k6 [thresholds](https://grafana.com/docs/k6/latest/using-k6/thresholds/)
- Grafana k6 [dropped iterations](https://grafana.com/docs/k6/latest/using-k6/scenarios/concepts/dropped-iterations/)
- Testcontainers for Go [PostgreSQL module](https://golang.testcontainers.org/modules/postgres/)
- PostgreSQL 17 [`ANALYZE`](https://www.postgresql.org/docs/17/sql-analyze.html) and [`EXPLAIN`](https://www.postgresql.org/docs/17/using-explain.html)
