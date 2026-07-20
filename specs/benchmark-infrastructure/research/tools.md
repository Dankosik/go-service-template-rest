# Benchmark infrastructure tool research

status: ready

review: PASS for research content blob `c4465f2fc853a80b316a4110b0bf3098060e9645` (2026-07-20)

valid as of: 2026-07-20

## Boundary

This research identifies evidence-backed tools and constraints for benchmark
infrastructure in this Go REST template. It does not select the final CI policy,
define service performance budgets, expose runtime debug endpoints, or implement
targets, benchmarks, workflows, or dependencies.

The downstream owner is Specification for the desired template behavior and
performance policy, followed by Technical Design for tool placement and CI
execution.

## Current repository baseline

### Supported: the repository has measurement-adjacent infrastructure, but no benchmark harness

- The module targets Go `1.26.5` (`go.mod`), so new benchmarks can use
  `testing.B.Loop`.
- Developer tools are pinned in the `go.mod` `tool` block and invoked through
  `go tool`; `Makefile` is the repository command interface.
- Unit, race, coverage, fuzz-smoke, flake-smoke, integration, and Docker-backed
  test paths exist. GitHub Actions already uploads test and coverage artifacts.
- HTTP composition has an in-process `http.Handler` seam at
  `internal/infra/http.NewRouter`; PostgreSQL integration proof already uses the
  repository's test/integration and Testcontainers conventions.
- Prometheus request-duration metrics and OpenTelemetry instrumentation exist.
  These are runtime observability signals, not controlled benchmark evidence.
- No `BenchmarkXxx`, `testing.B`, benchmark Make target, `benchstat` pin,
  benchmark artifact convention, load-test scenario, performance workflow, or
  runtime `pprof` endpoint was found.

Absence boundary: CodeGraph source inventory plus repository-wide searches of
tracked Go, `Makefile`, `go.mod`, `.github/workflows`, `scripts`, `build`, and
developer documentation on 2026-07-20. Generated skill examples mention
benchmark commands, but they are method guidance rather than executable
repository infrastructure.

Implication: reuse the existing Go test, tool-pinning, router, artifact, and
Docker seams. Do not create a second test framework or generic benchmark service.

## Decision-changing questions

### Q1. What should own repeatable in-process code benchmarks?

Disposition: **supported — Go `testing.B` is the default harness**.

Facts:

- The standard `testing` package discovers `BenchmarkXxx(*testing.B)` through
  `go test -bench`.
- `B.Loop`, available since Go 1.24, automatically excludes setup and cleanup
  from timing and keeps loop values alive against compiler elimination.
- The standard harness supports sub-benchmarks, allocation metrics,
  throughput/custom metrics, controlled `GOMAXPROCS`, and parallel benchmarks.
- Kubernetes scheduler performance tests are a mature real implementation that
  continues to use `go test -bench` for scenario-shaped integration benchmarks.

Implications:

- Use `B.Loop` for new benchmarks, but exclude only setup that is outside the
  accepted operation. Per-operation input/state construction remains measured
  when it is part of that operation; otherwise mutable state must be reset
  without contaminating the measured work.
- Use named sub-benchmarks for materially different input sizes, cache states,
  warm/cold or mutable-state cases, and concurrency shapes; do not aggregate
  them into one number. Specification/Test Design owns the measured operation
  boundary.
- Use `-benchmem` or `B.ReportAllocs` when allocation or GC cost matters.
- Use `B.ReportMetric` only for domain-relevant units such as rows/op or B/op.
- Use `RunParallel` only when concurrent callers are the workload being
  measured; it does not prove production capacity or safe concurrency bounds.
- A third-party Go microbenchmark framework would duplicate the standard owner
  without closing a current evidence gap.

Sources:

- Go, [`testing` benchmarks and `B.Loop`](https://pkg.go.dev/testing#hdr-Benchmarks).
- Go, [`go test` benchmark flags](https://pkg.go.dev/cmd/go#hdr-Testing_flags).
- Kubernetes,
  [`scheduler_perf` benchmark package](https://pkg.go.dev/k8s.io/kubernetes/test/integration/scheduler_perf).

### Q2. What should compare baseline and candidate results?

Disposition: **supported — `golang.org/x/perf/cmd/benchstat` is the default comparator**.

Facts:

- `benchstat` is maintained by the Go project in `golang.org/x/perf`.
- It consumes standard Go benchmark output, summarizes medians and confidence
  intervals, and performs A/B significance tests.
- Its documentation requires at least 10 samples, recommends about 20, warns
  against rerunning until significance appears, and recommends interleaving
  baseline and candidate runs on an idle, thermally stable machine.
- The `x/perf` repository remained active in July 2026. It is BSD-3-Clause, but
  it uses pseudo-versions rather than a stable tagged v1 release.

Implications:

- Pin an exact `x/perf` pseudo-version through the repository's existing
  `go.mod` tool convention; do not use `@latest` in CI.
- Preserve raw baseline and candidate outputs beside the `benchstat` report.
- Treat at least 10 and ideally 20 runs as `benchstat` maintainer guidance, not
  a universal acceptance target. Specification/Design must fix the count,
  material-change threshold, and comparison rule from the benchmark's observed
  variance and the performance budget before execution.
- Statistical significance alone does not establish that a delta is materially
  important. A shorter developer smoke run remains a different,
  non-decision claim.
- Do not replace statistical comparison with one raw `ns/op`, a hand-written
  percentage script, or a historical number from another machine.

Sources:

- Go project,
  [`benchstat` command documentation](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat).
- Go project,
  [`x/perf` repository and tool inventory](https://go.googlesource.com/perf/).

### Q3. Which tools explain a measured regression?

Disposition: **supported — native `pprof` and runtime trace are diagnostic complements, not benchmark gates**.

Facts:

- `go test` can emit CPU, allocation, block, mutex, and execution-trace
  artifacts directly.
- `pprof` attributes sampled CPU/allocation cost to code paths.
- runtime trace records goroutine scheduling, blocking, syscalls, GC, heap, and
  processor events and is interpreted with `go tool trace`.
- Go diagnostics guidance warns that profilers can perturb one another and
  recommends collecting one relevant class at a time.

Implications:

- Escalate from benchmark deltas to the profile matching the hypothesis: CPU
  for active compute, allocation/heap for memory pressure, block/mutex/trace
  for waiting, contention, or scheduler behavior.
- Do not run every profile on every benchmark or treat a profile as a
  pass/fail measurement.
- A production `/debug/pprof` surface is a separate observability/security
  decision. This research does not justify adding it to the public router.

Sources:

- Go, [`go test` profiling flags](https://pkg.go.dev/cmd/go#hdr-Testing_flags).
- Go, [`runtime/pprof`](https://pkg.go.dev/runtime/pprof).
- Go, [`runtime/trace`](https://pkg.go.dev/runtime/trace).
- Go, [Diagnostics](https://go.dev/doc/diagnostics).

### Q4. What should measure a real HTTP service under load?

Disposition: **bounded-uncertain — mature candidates exist, but selection
requires a representative endpoint workload and budget**.

Neutral decision slot: controlled HTTP workload generation with request
correctness checks, latency/error observables, repeatable load shape, and
machine-readable output.

#### Grafana k6

- Current evidence: actively maintained (`v2.1.0`, 2026-06-30), about 31k GitHub
  stars, 91 releases, AGPL-3.0.
- Strengths: scenarios as code; closed-VU and open arrival-rate executors;
  per-scenario tags; built-in HTTP metrics; percentile/error thresholds that
  produce a non-zero exit; JSON/CSV/custom summaries and external outputs;
  local, Kubernetes-distributed, and managed-cloud execution.
- Conditional fit: covers multi-step REST flows, explicit p95/p99/error
  budgets, and several traffic classes.
- Cost/limit: introduces a non-Go JavaScript scenario surface and a separately
  pinned binary/container. Distributed open-source execution has coordination
  and aggregation limits documented by k6. Tool use does not copy k6 code into
  the service, but the AGPL tool license still needs the repository's normal
  dependency/license disposition.

#### Vegeta

- Current evidence: mature and maintained (`v12.13.0`, 2025-10-31), about 25k
  stars, 71 releases, MIT.
- Strengths: small Go CLI/library, constant-rate HTTP attacks, explicit concern
  for coordinated omission, binary/JSON reports, histograms, plots, and simple
  Unix composition.
- Conditional fit: covers one endpoint or a generated request stream at a
  controlled arrival rate with a smaller scenario surface.
- Cost/limit: the reviewed interface is oriented around attack/report
  composition, not k6-style versioned multi-scenario threshold policy. Using it
  as a blocking CI gate would require repository-owned pass/fail and scenario
  scripting.

#### Fortio

- Current evidence: active (`v1.75.2`, 2026-06-11), 191 releases,
  Apache-2.0; originated as Istio's load tool.
- Strengths: fixed-QPS HTTP/gRPC load, percentile histograms, JSON results,
  Docker image, UI/server mode, and an embeddable Go library.
- Local fit: viable when HTTP and gRPC plus a built-in result UI are desired.
- Rejection for the current decision level: its extra server/UI/echo/proxy
  surface does not close a template need that k6 scenarios or Vegeta's smaller
  CLI leave open.

#### `hey`, `wrk`, and a custom generator

- `hey` and `wrk` remain useful ad hoc saturation probes.
- They are not the best template infrastructure owner because the reviewed
  surfaces prioritize request generation over versioned workload policy,
  correctness assertions, threshold semantics, and portable result artifacts.
- A custom `net/http` load generator would duplicate mature scheduling,
  reporting, and histogram work.

Decision implication:

- Do not add a generic load script to this template merely to hit `/health` or
  `/metrics`; it would produce a number without a representative product path.
- Keep k6 and Vegeta unranked until Specification defines the endpoint flow,
  arrival model, data shape, correctness checks, percentile/error budget, and
  execution environment. Design can then select k6 for scenario-rich needs or
  Vegeta for a fixed-rate single-flow need, using context-matched operational
  evidence. Keep only one.
- Current maintainer documentation and project activity establish capability
  and maturity, not production fit for this repository. No context-matched
  operational source was found because the local workload is not yet defined.

Sources:

- Grafana, [k6 scenarios and executors](https://grafana.com/docs/k6/latest/using-k6/scenarios/).
- Grafana, [k6 thresholds and exit behavior](https://grafana.com/docs/k6/latest/using-k6/thresholds/).
- Grafana, [k6 result output](https://grafana.com/docs/k6/latest/results-output/end-of-test/).
- Grafana, [`grafana/k6`](https://github.com/grafana/k6).
- Vegeta, [`tsenart/vegeta`](https://github.com/tsenart/vegeta).
- Fortio, [`fortio.org/fortio`](https://pkg.go.dev/fortio.org/fortio).
- `hey`, [`rakyll/hey`](https://github.com/rakyll/hey).
- `wrk`, [`wg/wrk`](https://github.com/wg/wrk).

### Q5. Should the template add continuous benchmark storage or PR gating now?

Disposition: **contradicted for the initial layer; bounded candidates exist for a later scale trigger**.

Facts:

- Standard GitHub-hosted jobs run on a new VM. The current repository uses
  `ubuntu-latest`, not a benchmark-specific stable runner.
- `github-action-benchmark` supports Go output, charts, historical storage, and
  percentage alerts, but its own documentation reports roughly ±10–20%
  amplitude on its GitHub-hosted examples and recommends stable/self-hosted
  runners when that is unacceptable. Its GitHub Pages write mode must not run
  unguarded on pull requests.
- CodSpeed offers managed regression reports, stable runners, history, and
  flamegraphs, but its Go integration is explicitly early-development,
  walltime-only, and currently supports only the required `-bench` flag.
- Bencher supports branch/testbed-aware history, several threshold models,
  SaaS or self-hosting, and a Go benchmark adapter. The Go adapter currently
  collects only a mean latency value without lower/upper confidence bounds.

Implications:

- Do not make historical results from `ubuntu-latest` a blocking performance
  oracle.
- If PR comparison is later required, first run baseline and candidate on the
  same machine/job, preferably interleaved, or use a controlled dedicated
  runner. Keep correctness tests blocking independently of noisy performance
  evidence.
- Start with raw artifacts plus `benchstat`. Add a history service only when
  there are stable owned benchmarks, a stable testbed, a threshold policy, and
  a named operator.
- Re-evaluate CodSpeed when its Go support covers the required flags and metrics.
  Re-evaluate Bencher when cross-commit history and operated testbeds justify a
  service. Use `github-action-benchmark` only for non-blocking visualization
  unless runner variance is independently bounded.

Sources:

- GitHub, [GitHub-hosted runner reference](https://docs.github.com/en/actions/reference/runners/github-hosted-runners).
- Go project, [`benchstat` measurement tips](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat).
- [`benchmark-action/github-action-benchmark`](https://github.com/benchmark-action/github-action-benchmark).
- CodSpeed, [Go integration and compatibility](https://codspeed.io/docs/benchmarks/go).
- Bencher, [Go benchmark adapter](https://bencher.dev/docs/explanation/adapters/#go-bench).
- Bencher, [continuous benchmark models](https://bencher.dev/docs/how-to/track-benchmarks/).

## Research-supported minimum for downstream design

The smallest candidate set that closes the current gap is:

1. Go `testing.B` benchmarks beside the owning package, using `B.Loop`.
2. One pinned `benchstat` tool for repeated baseline/candidate comparison.
3. Raw result and comparison artifacts with environment metadata.
4. On-demand native `pprof`/trace commands selected by the observed symptom.

Conditional additions:

- Add one HTTP load generator only with a real service workload and threshold:
  k6 for scenario-rich REST, or Vegeta for fixed-rate single-flow HTTP.
- Add a continuous benchmark service or dedicated runner only after stable
  benchmarks and an owned regression policy exist.

Skipped as unjustified now: a new Go benchmark framework, custom statistical
scripts, public runtime profiling endpoints, a generic template load scenario,
multiple HTTP generators, automatic PR blocking on `ubuntu-latest`, and a
benchmark database/dashboard.

## Evidence limits and reopen conditions

- No service-specific hot path, workload distribution, latency/throughput
  budget, database size, cache state, or concurrency target was provided.
  Specification owns those decisions.
- No benchmark was executed because the repository contains no benchmark target
  and this phase is research-only. Tool documentation does not prove local
  variance or a performance budget.
- GitHub runner variability was not measured for this repository. Reopen the
  environment question before any blocking CI threshold.
- External tool evidence is freshness-sensitive. Refresh versions, licenses,
  Go compatibility, and hosted-product limits immediately before pinning.
- Reopen load-tool selection when an accepted scenario needs browser execution,
  gRPC, distributed load, private-network execution, or a managed service.

## Stop rationale

The surviving tools occupy distinct slots: standard Go measurement,
statistical comparison, native diagnosis, optional HTTP scenario load, and
optional history. Current official sources, active project evidence, mature
real implementations, and the repository baseline agree on those boundaries.
Additional generators or dashboards would duplicate a covered slot without
changing the downstream decision.
