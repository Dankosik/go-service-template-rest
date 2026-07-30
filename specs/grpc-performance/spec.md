# Reproducible gRPC performance evidence

status: ready

## Scope and non-goals

The repository must provide a reproducible way to measure the performance of
its production-composed gRPC transport, attribute fixed overhead to the
transport, telemetry, repository policy, logging, and handler layers, and
exercise unary plus every streaming cardinality end to end.

The delivered capability covers:

- allocation-aware in-process Go benchmarks;
- local end-to-end gRPC load and smoke scenarios;
- profile capture for a named bottleneck hypothesis;
- evidence metadata and comparison rules compatible with
  [`docs/benchmarking.md`](../../docs/benchmarking.md);
- a current synthetic local baseline that proves the harness works.

The following are non-goals until representative derived-service evidence
reopens them:

- changing production gRPC limits, buffers, flow-control windows, worker
  dispatch, keepalive, compression, connection count, logging, telemetry, or
  sampling defaults;
- enabling PGO or committing a synthetic `default.pgo`;
- claiming production latency, throughput, capacity, cost, or improvement;
- adding `ghz`, Fortio, a custom codec, a custom buffer pool, `vtprotobuf`, or
  third-party compressors;
- running or purchasing remote infrastructure.

## Behavior and contract delta

### PERF-1: production-composed microbenchmarks

A developer can run repository-owned Go benchmarks that pass representative
generated messages through the same server composition used by the service,
including the standard interceptor order and OpenTelemetry stats handler.

The benchmark surface must distinguish at least:

- bare native gRPC transport;
- transport plus OpenTelemetry;
- repository correlation and admission policy;
- the complete interceptor chain with log output discarded;
- the complete interceptor chain with the production structured-log path;
- the complete chain with a deterministic synthetic handler.

Setup, listener construction, client connection establishment, and fixture
creation are outside the timed loop unless the benchmark explicitly names
connection setup as the operation under test. Each result reports time,
bytes/op, and allocations/op.

### PERF-2: four-cardinality end-to-end scenarios

A repository-owned load entrypoint can execute unary, server-streaming,
client-streaming, and bidirectional-streaming calls against a server composed
through the production gRPC adapter.

The default local scenario is a bounded smoke or synthetic sensitivity run. It
must not silently remove or raise production safety limits, require server
reflection, or expose a public listener.

### PERF-3: streaming observability

End-to-end streaming evidence includes custom metrics for:

- stream lifetime;
- per-message lag when the scenario defines a send/receive correlation;
- terminal success or failure;
- successful and failed stream counts;
- sent, received, and successfully processed message counts.

Built-in streaming counters alone do not satisfy this rule. A four-cardinality
smoke must fail when an RPC returns the wrong payload, count, terminal status,
or completion state even when transport-level metrics were emitted.

### PERF-4: evidence levels and required results

The gRPC harness exposes two local evidence levels with different result
contracts.

`smoke` is correctness evidence. It completes at least one successful call for
each RPC cardinality and validates the exact payload, message count, terminal
status, and stream completion. Its pass condition is zero correctness failures,
zero unexpected terminal failures, and a non-zero success denominator for
every cardinality. Smoke timings are diagnostic only and cannot support a
performance or capacity claim.

`synthetic` is local performance-sensitivity evidence. It runs an explicit
warmup followed by a fixed measured interval and emits:

- offered and achieved operation rate;
- successful unary-call, stream, sent-message, and received-message counts;
- unary duration plus stream-lifetime p50, p95, and p99;
- per-message-lag p50, p95, and p99 for streaming scenarios with correlated
  send/receive messages;
- terminal error, correctness-failure, timeout, and dropped-work counts/rates;
- client bytes sent and received;
- scenario duration, virtual-user/concurrency setting, connection count, and
  the workload identity required by PERF-5.

A retained local synthetic baseline is valid only when every required metric
has samples, every cardinality has a non-zero success denominator, correctness
and terminal-error thresholds pass, no offered work is silently omitted, and
the runner records which resource/headroom observables are unavailable.

The local baseline does not prove sustainable capacity. A decision-grade
capacity or optimization claim additionally requires server CPU, resident and
heap memory, GC, admission utilization/rejection, connection/stream counts,
network utilization, and load-generator CPU/network headroom on an equivalent
testbed. Those observables remain explicitly deferred until a representative
service workload and qualified testbed exist.

### PERF-5: workload identity

Every result identifies:

- source revision and generated-contract identity;
- Go toolchain, architecture, and operating system;
- scenario and RPC cardinality;
- payload shape and approximate encoded size;
- offered-load, concurrency, connection, TLS, and telemetry modes;
- server and load-generator resource limits when applicable;
- warmup and measured duration;
- success denominator and any dropped or rejected work.

The template may ship synthetic powers-of-scale to reveal sensitivity across
small and large payloads or low and high concurrency. Synthetic points are
labelled assumptions and never become a workload forecast, service budget, or
capacity target.

### PERF-6: causal comparisons

A baseline/candidate comparison changes one intended performance variable and
keeps other material inputs equivalent. When the candidate necessarily changes
source, generated code, dependency version, schema, or PGO input, the evidence
records both identities instead of requiring a shared commit or contract.

Repeated Go benchmark samples are consumable by `benchstat`. A result with
errors, an unknown success denominator, a saturated load generator, or
unrecorded material input drift cannot support an improvement claim.

### PERF-7: profile-driven follow-up

The repository documents and supports collecting the profile that matches the
current hypothesis:

- CPU for compute or serialization cost;
- heap or allocations for memory/GC cost;
- mutex or block for contention;
- runtime trace for scheduler, goroutine, or lifecycle behavior.

Profiles are tied to the exact scenario and build that produced them. A
synthetic benchmark profile may guide investigation but cannot become a
production PGO profile.

### PERF-8: unchanged runtime behavior

Ordinary service startup, generated API behavior, health, admission, telemetry,
logging, error mapping, shutdown, client construction, and production
configuration remain unchanged. Benchmark-only composition and fixtures cannot
be imported into production packages.

The reference-dependent end-to-end harness is retained in an initialized
service only when `GRPC=enabled` and `REFERENCE_EXAMPLE=keep`. If either owner
is removed, initialization removes the benchmark command, k6 scenario,
dedicated lifecycle check, Make entrypoints, shared runner branches, and
operator documentation together. The in-package `grpcx` microbenchmarks remain
owned by the gRPC runtime and do not depend on the reference example.

The isolated reference service receives one contract correction required for a
finite performance oracle: `ServerStream.count > 1024` returns
`ResourceExhausted` before emitting a response. This does not change the
template service's production API.

## Invariants and edge cases

- The production interceptor order is authoritative; the benchmark does not
  maintain an independent imitation of that order.
- The reference server-stream sequence is one-based. A non-positive `count`
  emits no responses; `count` in `[1,1024]` emits exactly `count` responses,
  beginning with `sequence=1` and ending with `sequence=count`; a larger count
  returns `ResourceExhausted` before the first response. The canonical schema,
  runtime, generated comments, and tests must agree.
- The teaching echo service is not evidence for the production pipeline unless
  the scenario explicitly composes the production adapter.
- Long-lived streams retain the current process admission semantics; the
  harness observes saturation rather than bypassing it.
- Large-message scenarios lower concurrency before approaching the current
  message limit so the harness does not create an avoidable memory hazard.
- Cancellation, terminal status, early stream close, slow readers/writers, and
  server drain remain observable failures rather than successful completions.
- Telemetry-disabled or log-discard variants are attribution controls. They do
  not authorize corresponding production defaults.
- Benchmark output and profiles are disposable evidence unless an accepted
  comparison explicitly retains them. Generated or raw result files are not
  committed by default.

## Decisions, constraints, and authorities

- [`research/synthesis.md`](research/synthesis.md) owns current candidate
  evidence and reopen conditions.
- [`docs/benchmarking.md`](../../docs/benchmarking.md) owns benchmark levels,
  host qualification, variance, comparison, and claim policy.
- The current gRPC server, client, configuration, generated Protobuf contract,
  telemetry, and logging implementations remain runtime authorities.
- The existing digest-pinned k6 image remains the default end-to-end load
  platform. A second general load tool requires a concrete unmet capability.
- The performance decision is `proof_only`: build and run the evidence
  capability before selecting an optimization.

## Success criteria and proof expectations

The capability is complete when:

1. focused Go benchmark commands run the named layer variants with
   `-run='^$'`, `-benchmem`, and repeated-sample support;
2. an end-to-end local smoke exercises all four RPC cardinalities through the
   production-composed server and validates payload, count, status, and
   completion;
3. streaming scenarios emit the custom observables required by PERF-3;
4. a local synthetic run emits every PERF-4 client-side result, carries a
   non-zero success denominator for every cardinality, and explicitly marks
   the deferred decision-grade resource/headroom fields unavailable;
5. the repository's benchmark validation rejects missing benchmark samples,
   failed correctness checks, unexpected terminal failures, dropped or
   unaccounted work, and malformed evidence metadata;
6. CPU and memory profile commands can target the full production-composed
   benchmark without timing setup work;
7. existing gRPC correctness, generation, lint, and repository-integrity
   checks remain green for the changed surface;
8. the retained local result is labelled synthetic and reports no unsupported
   production or before/after claim;
9. generated profiles contain no dangling gRPC benchmark command when the
   reference example or native gRPC capability is removed.

## Delivery and completion split

The accepted outcome is delivered in two dependency-ordered phases because the
current implementation host has no Docker-compatible runtime or k6 binary.

Phase A is the current implementation completion. It owns:

- the canonical reference-contract correction and generated drift proof;
- the canonical gRPC-default accessor and equivalence proof;
- the Go layer-attribution benchmarks and profile entrypoints;
- the production-composed benchmark server, k6 scenario, runner, Make
  entrypoints, and documentation;
- focused Go, shell, generation, and repository-integrity proof that does not
  require executing k6.

Phase A may claim that the repository contains an implementation-ready gRPC
performance harness and that its Go benchmark path is locally proven. It may
not claim that the k6 script is executable, that all cardinalities pass through
the containerized client, that the synthetic result contract is satisfied, or
that the full capability is complete.

Phase B is the mandatory activation and acceptance task. On a host with the
repository-pinned k6 image and a supported Docker network path, it runs inspect,
four-cardinality smoke, and synthetic modes, then validates the retained
summary and metadata against PERF-2 through PERF-5. Only Phase B can close
success criteria 2 through 5 and authorize the full capability-complete claim.
Any Phase B failure reopens the implementation owner; an unavailable runtime
keeps Phase B blocked rather than weakening its oracle.

## Risks, assumptions, and reopen conditions

- Assumption: a synthetic reference schema is sufficient to validate the
  harness, not to characterize a derived service. Reopen workload design when
  an actual service supplies payload and traffic distributions.
- Assumption: local execution is sufficient for harness acceptance. Reopen
  capacity proof when a production-equivalent host, network route, and service
  budget exist.
- Reopen transport tuning only when a profile or saturation result identifies
  transport buffering, flow control, stream dispatch, or connection queueing
  as material.
- Reopen logging or telemetry policy only when full-path attribution shows a
  budget violation and observability/security owners can preserve required
  signals.
- Reopen compression only for a large, compressible payload on a constrained
  link with controlled peer compatibility.
- Reopen PGO only when a representative CPU profile has provenance, refresh,
  comparison, canary, and rollback owners.
