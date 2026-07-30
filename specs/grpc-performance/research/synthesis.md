# gRPC performance research

status: ready
valid_as_of: 2026-07-30
refresh_triggers:
- before changing the pinned Go, gRPC-Go, Protobuf, OpenTelemetry, or k6 versions;
- when a representative derived-service RPC schema or production workload is available;
- when a profile shows transport, codec, logging, telemetry, allocation, or scheduler cost dominating the hot path;
- before adopting an experimental gRPC-Go option, custom codec, custom buffer pool, or third-party compressor;
- before making a numeric latency, throughput, capacity, or cost claim.

## Accepted outcome and evidence boundary

Determine whether the newly integrated native gRPC path has a credible
high-performance foundation, discover the libraries and tools that could
increase its performance, eliminate attractive but poorly fitting tuning
options, and hand downstream design and test owners a measurement-first route
to the maximum performance justified by representative evidence.

This is a research artifact. It does not change runtime behavior, select final
configuration values, or establish a numeric service-level objective. A
performance claim requires a representative RPC schema, handler cost, payload
distribution, concurrency distribution, network path, client behavior, and
target environment. None of those workload authorities exists at template
level today.

The current gRPC integration is an uncommitted local candidate. This research
inspects that current tree without treating its implementation status as
accepted or published behavior.

Evidence covers:

- Go `1.26.5`, gRPC-Go `v1.82.1`, Go Protobuf `v1.36.11`,
  OpenTelemetry gRPC instrumentation `v0.69.0`, and the pinned k6
  `2.1.0` image in the current repository;
- the current server, client, interceptor, telemetry, configuration, runtime,
  diagnostics, reference-service, and benchmark infrastructure;
- current upstream gRPC, Go, Protobuf, OpenTelemetry, k6, and candidate-tool
  contracts available on the validity date;
- transport, serialization, middleware, runtime, network, and measurement
  candidate families.

It does not establish:

- that the current path is faster or slower than the HTTP path;
- an achieved requests-per-second value, latency percentile, allocation rate,
  CPU efficiency, memory footprint, or maximum connection count;
- that the teaching-only echo service represents a production handler;
- that a particular compressor, buffer size, worker count, connection-pool
  size, sampling policy, or PGO profile improves a derived service;
- public-edge, multi-region, or Railway gRPC performance.

## Conclusion

The current integration has a strong native baseline, but its performance is
not yet proven.

What is already structurally good:

- clients are designed around a long-lived shared `grpc.ClientConn`;
- generated calls already carry `grpc.StaticMethod()`;
- gRPC-Go's shared write buffer, dynamic HTTP/2 flow-control tuning, and
  default tiered buffer pool are active without custom code;
- Edition 2023 plus the Opaque Go API avoids locking message memory layout and
  can reduce allocations for suitable schemas;
- `GOMAXPROCS` and `GOMEMLIMIT` already follow container limits;
- message events are not enabled in OpenTelemetry;
- the private diagnostics surface can expose `pprof`;
- explicit connection, stream, RPC, header, and message bounds prevent an
  unbounded benchmark result from becoming an unsafe production default.

What remains unknown:

- the fixed cost of the full interceptor and telemetry pipeline;
- the cost of one JSON access log per business RPC;
- the saturation behavior of the `256` process admission limit and the `100`
  streams-per-connection limit;
- the effect of payload shape, streaming cardinality, TLS, RTT, bandwidth, and
  handler cost;
- whether CPU, allocation, scheduler, network, flow control, admission, or a
  downstream dependency is the first bottleneck.

No replacement RPC framework or third-party serialization library dominates
the current stack. The highest-value next capability is a layered benchmark
and profiling harness. The most credible later optimizations are PGO,
workload-specific access-log or telemetry policy, and additional client
connections only when measured saturation identifies those owners.

Static flow-control windows, custom buffer pools, blanket compression, and
experimental stream workers are not safe default optimizations. `vtprotobuf`
is not currently compatible with the template's Edition 2023 requirement and
must not be adopted into the base path.

## Open-item map

| Item | Research method | Downstream owner | Disposition |
| --- | --- | --- | --- |
| Is the current integration already fast? | Current-state baseline + evidence gap | Performance test design | Native defaults are credible; no service-level benchmark exists, so numeric performance is unknown. |
| What is the first bottleneck? | Hot-path map + empirical probe design | Performance test implementation | Compare bare transport, codec, telemetry, repository policy, logging, and handler layers under the same workload. |
| Which load tool belongs in the template? | Solution discovery + repository fit | Performance design | Extend the existing pinned k6 harness with `k6/net/grpc`; use `ghz` only as a focused diagnostic complement. |
| Are gRPC-Go transport knobs useful? | Pinned source inspection | Go ownership | Defaults are intentionally tuned. Carry individual knobs only behind a profile-backed hypothesis. |
| Should clients pool connections? | External contract + saturation model | Client/reliability design | Preserve one shared connection by default; test a bounded pool only when stream limits or long-lived streams cause queueing. |
| Should the template enable compression? | Network/CPU trade-off analysis | API/performance design | No blanket default. Evaluate built-in gzip only for large compressible payloads on constrained links. |
| Should Protobuf be replaced or accelerated? | Candidate compatibility check | Contract/generation design | Keep official Opaque API. `vtprotobuf` is eliminated until it supports Protobuf Editions and the required Go API. |
| Can schema choices improve decode cost? | Official Opaque API evidence | API/performance design | Carry Opaque API and conditional lazy decoding for deep, partly consumed submessages; prove per schema. |
| Can Go compiler/runtime tuning help? | Runtime baseline + official PGO evidence | Delivery/performance design | PGO is a strong conditional candidate once a representative CPU profile and profile lifecycle exist. |
| Can observability be made cheaper? | Hot-path inspection + benchmark obligation | Observability design | Measure first. Preserve failure visibility; change logging, health filtering, attributes, or sampling only against an explicit observability budget. |
| What result counts as excellent? | Quantity provenance audit | Specification | No template authority supplies an SLO. A derived service must provide or derive latency, throughput, error, resource, and cost budgets. |

## Current baseline

### Runtime and dependency facts

| Surface | Current fact | Performance consequence |
| --- | --- | --- |
| Go runtime | Go `1.26.5`; no repository override of container-aware `GOMAXPROCS` | CPU scheduling should follow the container allocation without a second tuning owner. |
| Memory runtime | `automemlimit` sets `GOMEMLIMIT` from the cgroup at ratio `0.9` | The GC already has a container-aware memory boundary; arbitrary `GOGC` changes would be premature. |
| Server | Native `grpc.Server` on a separate listener | No HTTP gateway, JSON transcoding, or shared-handler adapter is in the native hot path. |
| Client | One long-lived `grpc.ClientConn` per target is the documented ownership model | Connection establishment and HTTP/2 setup are amortized over RPCs. |
| Protobuf | Edition 2023 with generated Opaque Go API | The generated representation can use presence bitmaps and future layout improvements. |
| Server transport | Default read/write buffers and dynamic flow control | gRPC-Go retains its adaptive defaults instead of freezing transport windows. |
| Buffering | Default gRPC-Go buffer pool and shared write buffers | A custom allocator or pooling layer would duplicate an already tuned owner. |
| Telemetry | `otelgrpc.NewServerHandler` and client handler; no message events | Every RPC has telemetry cost, but per-message event amplification is absent. |
| Tracing | never-sample without exporter; parent-based ratio `0.10` with exporter | Span machinery remains present; export and recording cost depends on configuration and sampling. |
| Diagnostics | Private `pprof` exists but is disabled by default | CPU, heap, mutex, block, goroutine, and execution-trace evidence can be enabled without a public endpoint. |
| Load generation | Repository pins k6 `2.1.0`, but only HTTP scenarios exist | Tooling can be extended without adding a second general load-test platform. |
| Reference service | Tests start a bare `grpc.NewServer()` | The example proves generated API behavior, not the production interceptor or telemetry cost. |
| Benchmarks | No `BenchmarkXxx` exists in the gRPC packages | There is no local allocation or fixed-overhead baseline. |

### Server hot path

A successful unary business RPC currently passes through:

1. gRPC-Go HTTP/2 transport, header processing, and Protobuf decode;
2. OpenTelemetry server stats handling;
3. correlation metadata validation or request-ID creation;
4. access logging setup and one completion log;
5. panic recovery;
6. process-wide admission with a non-blocking weighted semaphore;
7. two admission/load metric updates;
8. a policy error boundary;
9. zero or more service policy interceptors;
10. the application handler;
11. repository error mapping;
12. Protobuf encode and gRPC-Go response transport.

Streaming RPCs use the analogous stream interceptor chain. One long-lived
stream occupies one admission slot for its lifetime, while each connection is
bounded by the configured concurrent-stream limit.

The leading local hypothesis is that fixed policy, JSON access logging, and
telemetry will matter before native gRPC transport tuning for small, cheap
unary handlers. This is an inference from the number and nature of fixed
operations, not a measured conclusion.

### Current finite bounds

| Bound | Current default | Meaning | Not proven |
| --- | ---: | --- | --- |
| Connections | `4096` | Listener-level process connection cap | Safe or efficient resident-memory cost at the limit |
| Concurrent RPCs | `256` | Process-wide admission for unary and streaming RPCs | Optimal throughput or queueing point |
| Concurrent streams | `100` | HTTP/2 streams accepted per connection | Best client connection count for each workload |
| Request headers | `16 KiB` | Maximum received header-list size | Typical metadata cost |
| Receive message | `4 MiB` | Maximum decoded inbound message | Safe concurrent worst-case heap |
| Send message | `4 MiB` | Maximum outbound message | Safe concurrent serialization and buffering cost |

These values are safety constraints, not capacity targets. In particular,
`256 × 4 MiB` is not a valid memory forecast because actual wire, decoded,
application, telemetry, and buffering lifetimes differ, but it demonstrates
why message-size and concurrency limits cannot be raised independently.

## Performance model

For one RPC, reason about latency as:

`T_total = T_queue + T_network + T_http2 + T_decode + T_fixed_policy + T_handler + T_encode + T_gc`

For one process, sustainable throughput is bounded by the earliest exhausted
resource or policy:

`QPS <= min(CPU, memory/GC, network, connection streams, process admission, handler dependency, downstream capacity)`

This model prevents a faster codec from being credited for an admission queue,
or a larger HTTP/2 window from being credited for a CPU-bound handler.

The benchmark must therefore report at least:

- achieved RPC rate and offered RPC rate;
- p50, p95, p99, and maximum latency;
- status-code/error rate, timeouts, and load-generator dropped work;
- process CPU and CPU-seconds per fixed number of successful RPCs;
- allocations/op and bytes/op for in-process microbenchmarks;
- resident memory, heap profile, GC CPU, GC cycles, and pause distribution;
- bytes sent/received and connection/stream counts;
- admission rejections and active-RPC utilization;
- load-generator CPU headroom and network headroom.

## Upstream contract synthesis

### Channels, streams, and connection saturation

Official gRPC guidance says to reuse channels and stubs. Streaming avoids
repeated RPC setup for a genuinely long-lived logical flow, but a stream
cannot be load balanced after it starts and can reduce scalability or recovery
clarity. When active streams reach the connection's concurrent-stream limit,
additional calls queue; an additional channel/connection can relieve that
specific condition.

Local implication:

- keep one shared connection as the baseline;
- do not convert independent unary calls into streams solely for benchmark
  numbers;
- test more than one connection only when client-side queueing, long-lived
  streams, or the `100`-stream cap is visible;
- include connection setup, TLS, memory, load-balancing, and server connection
  cost when evaluating a pool.

Source: [gRPC performance best
practices](https://grpc.io/docs/guides/performance/).

### Buffers and flow control

Pinned gRPC-Go defaults to `32 KiB` read and write buffers, enables shared
write buffers, uses a tiered default `mem.BufferPool`, and dynamically grows
HTTP/2 flow-control windows using bandwidth-delay-product estimation.

The static window options explicitly disable dynamic flow control. Upstream
advises most users not to configure them unless memory pressure requires a
fixed window. A larger buffer can reduce syscalls or increase batching in one
workload while increasing per-connection memory in another. A zero buffer can
remove buffering but increase syscall pressure.

Local implication:

- preserve defaults in the base template;
- test `ReadBufferSize` or `WriteBufferSize` only after syscall, CPU, heap, or
  high-RTT evidence identifies buffering;
- do not expose static stream or connection windows as general performance
  configuration;
- do not add a custom buffer pool until allocation profiles show a stable size
  distribution that the current tiered pool handles poorly.

Sources:

- [gRPC-Go `ServerOption`
  documentation](https://pkg.go.dev/google.golang.org/grpc);
- [gRPC flow control](https://grpc.io/docs/guides/flow-control/);
- [gRPC-Go v1.82.1 source](https://github.com/grpc/grpc-go/tree/v1.82.1).

### Stream workers

`grpc.NumStreamWorkers` replaces the default goroutine-per-stream dispatch
with a reusable worker pool. Its documented motivation is avoiding repeated
goroutine stack growth, but it is experimental and defaults to zero.

Local implication:

- carry it only as a benchmark candidate for high-rate, short, cheap RPCs when
  CPU profiles show scheduler, `runtime.morestack`, or stack churn cost;
- verify latency under saturation and mixed long/short streams, not only peak
  throughput;
- do not turn an experimental option into a template-owned compatibility
  contract without an explicit benefit and rollback condition.

Source: [gRPC-Go package
documentation](https://pkg.go.dev/google.golang.org/grpc).

### Opaque Protobuf and lazy decoding

The official Opaque API can represent elementary-field presence with bitmaps
instead of pointers. Upstream benchmarks show workload-dependent allocation
and decode improvements; they are evidence of capability, not a forecast for
this template. Opaque also permits opt-in lazy decoding of annotated
submessages so unused nested branches need not be decoded.

Local implication:

- the template has already selected the strongest official base candidate;
- benchmark generated messages that resemble actual schemas, not the small
  echo message;
- consider `[lazy = true]` only for deep submessages that are frequently not
  accessed;
- verify first-access latency, allocations, total handler latency, and
  correctness for every lazy schema.

Source: [Go Protobuf: the new Opaque
API](https://go.dev/blog/protobuf-opaque).

### Compression

Compression trades CPU and allocation work for fewer network bytes. gRPC
supports per-message decisions and peer negotiation; a client can fail when
the peer does not support the chosen compressor.

Local implication:

- leave compression disabled by default;
- use built-in gzip as the first experiment for large, compressible messages
  on bandwidth-constrained or high-cost links;
- do not compress small messages, already compressed media, or CPU-bound
  traffic by policy;
- compare end-to-end tail latency, CPU/successful RPC, bytes transferred, and
  peer-language compatibility;
- do not add zstd, lz4, or snappy wrappers to the template without a
  controlled-peer use case that beats gzip and justifies another dependency.

Source: [gRPC compression](https://grpc.io/docs/guides/compression/).

### Keepalive

Keepalive can prevent an idle HTTP/2 connection from being removed by a proxy
and can reduce the first-call reconnect delay. It is primarily a connection
liveness and long-lived-stream concern, not a steady-state throughput knob.
Upstream warns against aggressive pings and requires client/server
coordination.

Local implication:

- do not enable keepalive for benchmark throughput;
- carry it to reliability design only when an observed proxy idle timeout,
  mobile/unreliable network, or long-lived stream requires it;
- include infrastructure policy and `GOAWAY too_many_pings` behavior in proof.

Source: [gRPC keepalive](https://grpc.io/docs/guides/keepalive/).

### PGO and runtime profiles

Go PGO uses a representative CPU profile at build time and can optimize
application, dependency, and standard-library hot paths. Official Go results
show material gains in some benchmark suites, but those values are not a
service forecast.

Local implication:

- PGO is a higher-confidence candidate than speculative transport knobs once
  a representative CPU profile exists;
- compare identical `pgo=off` and `pgo=profile` builds on the same testbed;
- version profile provenance, workload, age, collection rate, and rollback
  trigger;
- do not commit a synthetic echo profile as a universal `default.pgo`.

Sources:

- [Profile-guided optimization](https://go.dev/doc/pgo);
- [Go diagnostics](https://go.dev/doc/diagnostics).

### OpenTelemetry and access logging

The current OpenTelemetry gRPC handlers create RPC-level instrumentation but
do not emit per-message events. Sampling controls trace recording/export, not
all instrumentation work. The repository also emits one structured access log
for every non-health business RPC and records admission/load metrics.

There is no authoritative generic overhead percentage for this exact pipeline.

Local implication:

- benchmark the full production path, not only bare gRPC-Go;
- attribute cost by selectively replacing one layer at a time in a benchmark
  fixture;
- preserve errors, slow calls, correlation, and operational signals;
- consider health-method filtering, attribute reduction, trace sampling, log
  sampling, or error/slow-only access logs only if measured overhead violates
  the accepted observability budget;
- any reduced logging policy needs an observability/security decision, not a
  performance-only edit.

Sources:

- [OpenTelemetry gRPC instrumentation
  v0.69.0](https://pkg.go.dev/go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc@v0.69.0);
- [OpenTelemetry sampling
  specification](https://opentelemetry.io/docs/specs/otel/trace/sdk/).

## Candidate solution assessment

### Carry forward

| Candidate | Why it survives | Required discriminator |
| --- | --- | --- |
| Existing native gRPC-Go transport | Current upstream implementation, no translation layer, mature profiles and benchmarks | Full-path service benchmark |
| Existing Opaque Protobuf API | Official path with lower-allocation and future-layout potential | Representative message benchmarks |
| k6 `k6/net/grpc` | Already pinned and operated; supports unary and every streaming cardinality; can load `.proto` files without reflection | A four-cardinality smoke plus custom streaming metrics integrated with repository evidence format |
| Go `testing.B` + `benchstat` | Best fit for fixed overhead, allocations, and paired local comparisons | Layered benchmark fixture |
| `pprof` and Go execution trace | Already compatible with the private diagnostics model; identifies CPU, heap, blocking, locks, goroutines, GC, and scheduler cost | Representative steady-state profile |
| PGO | Optimizes observed application and dependency hot paths without a runtime fork | Representative profile and paired build proof |
| Conditional connection pool | Directly addresses measured per-connection stream saturation | Queueing or stream-cap evidence |
| Conditional built-in gzip | Can exchange CPU for network reduction without a third-party codec | Large compressible payload over constrained link |
| Conditional Opaque lazy fields | Can skip unused nested decode work without changing wire format | Deep, partly consumed message schema |

### Diagnostic complement, not base dependency

[`ghz`](https://github.com/bojand/ghz) is a focused gRPC load generator with
unary and streaming support, concurrency/RPS schedules, multiple connections,
compression, and machine-readable output. It is useful for:

- reproducing one method quickly;
- varying `--connections` to discriminate stream saturation;
- testing a compiled descriptor set when server reflection remains disabled;
- comparing results with k6 when the generator itself is suspect.

The repository already owns k6, remote benchmark orchestration, evidence
capture, and CPU-headroom policy. Adding `ghz` as a second mandatory harness
would duplicate lifecycle, pinning, reporting, and operator knowledge. Carry
it only as a version-pinned optional diagnostic until k6 fails a concrete
requirement.

k6's built-in gRPC streaming metrics are counters; they are not sufficient by
themselves for stream duration, per-message lag, terminal failure rate, or
successful-stream/message denominators. The harness must define custom
`Trend`, `Rate`, and `Counter` metrics and thresholds for those quantities,
then pass a four-cardinality smoke before k6 is accepted as sufficient for the
streaming matrix.

### Reference only

The [official gRPC-Go benchmark
suite](https://github.com/grpc/grpc-go/tree/v1.82.1/benchmark) is useful for
understanding transport microbenchmarks and upstream option comparisons. It
does not include this repository's interceptors, telemetry, schemas, handlers,
limits, TLS, or deployment path, so it cannot prove template performance.

### Eliminated or deferred

| Candidate | Disposition | Reopen condition |
| --- | --- | --- |
| `vtprotobuf` | Eliminated for the base path. Current tagged generator declares proto3 optional support but not Protobuf Editions support; its gRPC codec also expects VT methods on every message unless a mixed codec is maintained. | A maintained release supports the template's Edition 2023/Opaque contract and passes generation, codec, conformance, interop, and representative performance proof. |
| Fortio | Eliminated as the primary arbitrary-service harness. Its generic gRPC path is reflection-oriented while the template deliberately leaves reflection off; k6 and `ghz` fit descriptors/proto sources better. | It provides a unique required capability not available in the owned harness. |
| Custom gRPC codec | Deferred. It expands peer compatibility and generated-message ownership for an unmeasured codec bottleneck. | A codec profile dominates total cost and a compatible client/server fleet is controlled. |
| Third-party zstd/lz4/snappy gRPC compressors | Deferred as template defaults. They create peer-language, maintenance, security-update, and negotiation obligations. | A controlled service pair proves a material end-to-end win over no compression and gzip. |
| Custom `mem.BufferPool` | Deferred. gRPC-Go already supplies a tuned default and the replacement API is experimental. | Heap/allocation profiles show a stable, material mismatch and paired proof includes retained memory. |
| Static flow-control windows | Rejected as a performance default because they disable dynamic flow control. | A memory-constrained workload proves a fixed window improves its accepted budget without unacceptable throughput/tail loss. |
| Blanket larger read/write buffers | Deferred. Benefit depends on syscalls, payloads, RTT, and connection count; memory cost scales with connections. | Profiles identify transport syscall/buffering cost and a bounded value wins the representative matrix. |
| `grpc.NumStreamWorkers` | Deferred and experimental. It may reduce stack churn but changes scheduling and saturation behavior. | CPU/scheduler profiles identify the targeted cost and mixed-workload proof shows a stable win. |
| Aggressive keepalive | Rejected as a throughput optimization and unsafe without peer coordination. | Infrastructure idle-timeout or long-lived-stream evidence creates a reliability requirement. |
| Converting unary calls to streams | Rejected as a generic optimization. Long-lived streams lose per-call load balancing and complicate failure recovery. | The product operation is intrinsically one long-lived logical flow. |

## Recommended empirical program

Research does not choose the implementation, but the following probes are the
smallest set that can resolve the surviving decisions.

No performance probe was executed in this phase. The only ready-made echo
fixture starts a bare `grpc.Server`, so its result would exclude the production
interceptor and telemetry pipeline while appearing to represent it. Creating
the missing production-composed benchmark fixture is the first downstream
performance-test obligation.

### Probe 1: fixed-cost layer attribution

Add in-process unary and streaming benchmarks that can run the same generated
message and no-op handler through:

1. Protobuf marshal/unmarshal only;
2. bare gRPC-Go in-memory transport;
3. gRPC-Go plus OpenTelemetry;
4. gRPC-Go plus correlation and admission;
5. the complete repository interceptor chain with logs discarded;
6. the complete chain with the production JSON log path;
7. the complete chain plus a representative handler.

Report `ns/op`, `B/op`, and `allocs/op`; collect CPU and heap profiles for the
full path. Use the repository benchmark protocol and `benchstat` for paired
comparisons. A benchmark fixture must construct the production `grpcx.Server`,
because the current reference-service test starts a bare server.

Decision closed: whether the optimization owner is Protobuf, gRPC transport,
telemetry, logging/policy, or the handler.

### Probe 2: end-to-end capacity and tail latency

Extend the existing k6 harness with descriptor/proto-loaded gRPC scenarios.
Keep the load generator on a separate host for decision-grade capacity
results and retain at least the repository-required CPU headroom. Define
custom stream-lifetime, per-message-lag, terminal-status, successful-stream,
and successful-message metrics before using streaming results for percentile
or success-normalized claims.

Synthetic sensitivity matrix before a derived service supplies workload data:

| Dimension | Initial sensitivity points |
| --- | --- |
| Cardinality | unary, server stream, client stream, bidirectional stream |
| Payload | approximately 64 B, 1 KiB, 64 KiB, 1 MiB; add near-limit only with a memory-safe concurrency cap |
| Content | compressible and incompressible for compression experiments |
| Handler | no-op/echo, fixed CPU work, fixed downstream delay |
| Offered load | below saturation, knee, and controlled overload |
| Concurrency | `1`, near CPU parallelism, `64`, `256`; narrower values when large messages make them unsafe |
| Connections | `1`, then a bounded pool only around observed per-connection queueing |
| Network | local low RTT, representative service RTT/bandwidth, high-RTT sensitivity point |
| Security | plaintext private baseline and TLS where the deployment contract requires it |
| Telemetry | exporter absent, representative sampling/export, and full production mode |
| Build | PGO off and representative PGO profile when available |

The payload and concurrency values are engineering-selected powers-of-scale
that exercise qualitatively different transport and allocation regimes under
the current `4 MiB` message and `256` RPC safety caps. They are not a workload
forecast, capacity target, or acceptance boundary. Replace or weight them with
observed derived-service distributions before making a workload or capacity
claim.

Decision closed: sustainable load, saturation owner, tail behavior, capacity
limit, and whether connection, compression, telemetry, or PGO candidates
materially improve the accepted workload.

### Probe 3: long-lived stream lifecycle

Measure mixed short unary calls and long-lived streams while varying:

- stream count around the per-connection limit;
- stream message rate and backpressure;
- one versus multiple client connections;
- cancellation, slow readers/writers, and reconnect;
- drain and forced shutdown.

Report unary tail latency, stream lag, memory/stream, goroutines, admission
occupancy, connection count, and recovery behavior.

Decision closed: whether one admission budget and one shared connection remain
appropriate for the service's stream mix.

### Probe 4: serialization candidates

Use actual message shapes:

- many elementary fields with explicit presence;
- deep nested submessages where only top-level fields are read;
- repeated strings/bytes;
- large byte payloads;
- request-to-domain mapping representative of the service.

Compare ordinary Opaque decode with eligible lazy fields. Do not benchmark
`vtprotobuf` until its compatibility reopen condition is satisfied.

Decision closed: whether schema/API-level work can beat transport tuning.

## Evidence and benchmark controls

Every comparison should preserve:

- explicit baseline and candidate source identities;
- the same Go toolchain, target CPU architecture, container limits, kernel,
  TLS mode, network route, and load-generator image;
- the same commit and generated contract when they are not the intended
  candidate variable; schema, generated-code, dependency, or PGO comparisons
  record their distinct identities and keep every other input equivalent;
- an explicit warmup and steady-state interval;
- raw samples, benchmark command, environment fingerprint, CPU headroom, and
  profiles;
- correctness assertions independent of throughput;
- successful-RPC denominators so errors cannot make a candidate look faster;
- focused local evidence during iteration and isolated decision-grade evidence
  for final claims;
- one changed variable per causal comparison;
- rollback thresholds for experimental or service-specific tuning.

The repository's [`docs/benchmarking.md`](../../../docs/benchmarking.md)
remains the authority for benchmark level, workload equivalence, host
qualification, evidence retention, and completion policy.

## Downstream decision implications

### System/performance design

- Keep native gRPC-Go, the Opaque API, dynamic flow control, default buffers,
  and one shared client connection as the baseline candidate.
- Define the benchmark fixture as a first-class user of the production server
  composition rather than the bare teaching server.
- Keep performance settings narrow and profile-triggered; do not expose every
  upstream knob as environment configuration.
- Define how a service derives latency, throughput, resource, and cost budgets
  when no business SLO exists.
- Decide a profile provenance and refresh policy before enabling PGO.
- Route any lower-cost logging or telemetry mode through observability and
  security acceptance.

### Test design

- Separate microbenchmark, local integration, remote capacity, and deployment
  evidence.
- Cover all four RPC cardinalities and mixed workloads.
- Prove failure behavior at admission and stream limits, not only success below
  them.
- Compare bare and full-stack paths so an upstream library is not blamed for
  repository policy cost.
- Validate generator headroom; a saturated load generator invalidates the
  capacity result.
- Keep correctness, race/liveness, memory, and shutdown proof independent from
  performance acceptance.

### Implementation ownership

- Extend the existing benchmark platform before adding another general tool.
- Add `ghz` only if a focused connection or method diagnostic has a concrete
  owner and version pin.
- Do not implement the eliminated or deferred candidates without satisfying
  their reopen conditions.
- Preserve finite production limits during benchmarking; use explicit,
  isolated benchmark overrides rather than weakening defaults.

## Quantity and provenance ledger

| Quantity | Value | Provenance | Status |
| --- | ---: | --- | --- |
| Go version | `1.26.5` | Current `go.mod` and `go env` | Established |
| gRPC-Go version | `v1.82.1` | Current `go.mod` | Established |
| Protobuf version | `v1.36.11` | Current `go.mod` | Established |
| OTel gRPC version | `v0.69.0` | Current `go.mod` | Established |
| k6 image | `2.1.0`, digest-pinned | `scripts/dev/benchmark.sh` | Established |
| Server read/write buffer default | `32 KiB` each | Pinned gRPC-Go documentation/source | Upstream default |
| Connection cap | `4096` | Current template configuration | Safety default, not measured target |
| Process RPC cap | `256` | Current template configuration | Safety default, not measured target |
| Per-connection stream cap | `100` | Current template configuration | Safety default, not measured target |
| Header cap | `16 KiB` | Current template configuration | Safety default |
| Receive/send message cap | `4 MiB` each | Current template configuration | Safety default |
| Trace sampling with exporter | `0.10` | Current telemetry configuration | Observability default, not performance optimum |
| Synthetic payload sensitivity points | about `64 B`, `1 KiB`, `64 KiB`, `1 MiB` | Engineering-selected powers-of-scale below the current `4 MiB` safety cap | Assumption; replace/weight with service distribution |
| Synthetic concurrency sensitivity points | `1`, near CPU parallelism, `64`, `256` | Engineering-selected low/parallel/moderate/cap points | Assumption; not a capacity target |
| Achieved throughput | unknown | No representative benchmark | Open |
| Latency percentiles | unknown | No representative benchmark | Open |
| Allocation/CPU overhead | unknown | No gRPC benchmark or profile | Open |
| Maximum efficient connections | unknown | No connection-scaling probe | Open |
| Accepted SLO/cost budget | unknown | No derived-service authority | Downstream input |

## Counter-evidence and conflict handling

- Upstream benchmark numbers are not copied into a local capacity claim because
  they omit repository policy, telemetry, schemas, handlers, limits, and
  deployment conditions.
- The teaching echo service is not used as a production baseline because its
  test creates a bare `grpc.Server`.
- The common advice to enlarge HTTP/2 windows conflicts with current gRPC-Go's
  dynamic BDP estimator and explicit warning on static windows; the pinned
  implementation contract wins.
- The common advice to add object pooling conflicts with gRPC-Go's current
  default tiered buffer pool; a local allocation profile is required.
- The common advice to add more channels conflicts with official reuse
  guidance unless one connection is actually stream-saturated.
- The common claim that a faster generated codec is a drop-in improvement
  conflicts with the current Editions/Opaque contract and mixed-message codec
  requirements.
- Lower logging and trace volume can improve benchmarks but may violate
  observability or security acceptance; it is not credited without those
  independent requirements.

## Candidate-space saturation and stop rationale

The search covered every decision-changing family visible in the current hot
path:

- load generation, statistical comparison, and profiling;
- channel/connection topology and streaming shape;
- HTTP/2 flow control, buffers, worker dispatch, and keepalive;
- official Protobuf representation, lazy decoding, and third-party generated
  codecs;
- compression;
- interceptors, access logs, metrics, and tracing;
- Go runtime memory/CPU behavior and PGO;
- local, remote, TLS, high-RTT, large-message, overload, and mixed-stream
  workloads.

Additional libraries are unlikely to change the next decision. The unresolved
items are empirical: representative schemas, workloads, service budgets, and
profiles. Research therefore stops at a stable boundary and hands those proof
obligations to performance design and test design rather than inventing tuning
values.

## Sources

### Repository evidence

- `go.mod`
- `scripts/dev/benchmark.sh`
- `docs/benchmarking.md`
- `docs/grpc.md`
- `internal/config/grpc.go`
- `internal/infra/grpc/server.go`
- `internal/infra/grpc/interceptors.go`
- `internal/infra/grpcclient/client.go`
- `internal/infra/telemetry/metrics.go`
- `internal/infra/telemetry/traces.go`
- `internal/observability/logctx/logctx.go`
- `cmd/service/internal/bootstrap/startup_grpc.go`
- `examples/grpc-reference-service/service_test.go`

### Primary external evidence

- [gRPC performance best practices](https://grpc.io/docs/guides/performance/)
- [gRPC benchmarking](https://grpc.io/docs/guides/benchmarking/)
- [gRPC flow control](https://grpc.io/docs/guides/flow-control/)
- [gRPC compression](https://grpc.io/docs/guides/compression/)
- [gRPC keepalive](https://grpc.io/docs/guides/keepalive/)
- [gRPC-Go v1.82.1 source](https://github.com/grpc/grpc-go/tree/v1.82.1)
- [gRPC-Go package documentation](https://pkg.go.dev/google.golang.org/grpc)
- [gRPC-Go memory package](https://pkg.go.dev/google.golang.org/grpc/mem)
- [Go Protobuf Opaque API](https://go.dev/blog/protobuf-opaque)
- [Go profile-guided optimization](https://go.dev/doc/pgo)
- [Go diagnostics](https://go.dev/doc/diagnostics)
- [OpenTelemetry gRPC instrumentation v0.69.0](https://pkg.go.dev/go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc@v0.69.0)
- [OpenTelemetry trace sampling specification](https://opentelemetry.io/docs/specs/otel/trace/sdk/)
- [k6 gRPC documentation](https://grafana.com/docs/k6/latest/using-k6/protocols/grpc/)
- [k6 built-in and custom metrics](https://grafana.com/docs/k6/latest/using-k6/metrics/)
- [k6 large-test guidance](https://grafana.com/docs/k6/latest/testing-guides/running-large-tests/)

### Candidate implementation evidence

- [gRPC-Go benchmark suite](https://github.com/grpc/grpc-go/tree/v1.82.1/benchmark)
- [`ghz`](https://github.com/bojand/ghz)
- [`vtprotobuf`](https://github.com/planetscale/vtprotobuf)
- [Fortio](https://github.com/fortio/fortio)
