# gRPC performance evidence design

status: ready

## Drivers and selected architecture

The design must:

- measure the production `grpcx.NewServer` path without changing runtime
  defaults or adding benchmark-only seams to the production API;
- attribute fixed unary cost to native transport, OpenTelemetry, repository
  policy, structured logging, and deterministic handler work;
- exercise all four generated reference RPC cardinalities through that same
  production adapter;
- reuse the repository's pinned k6 image, benchmark metadata, artifact
  directory, profile commands, and comparison policy;
- keep the default local listener loopback-only and clean up every process and
  temporary artifact it owns;
- retain the reference-dependent k6 harness only when both native gRPC and the
  reference example are retained by module initialization;
- distinguish correctness smoke, local synthetic evidence, and deferred
  decision-grade capacity evidence.

The selected architecture has one benchmark platform with two complementary
paths:

1. Go `testing.B` benchmarks live beside `internal/infra/grpc` and measure
   allocation and fixed-layer cost over `bufconn`.
2. The existing benchmark runner starts an ephemeral production-composed
   reference server and runs the pinned k6 image against it over a real local
   TCP connection.

No production constructor, configuration key, listener, generated contract,
or runtime default changes.

## Decision slots

### D1: benchmark platform

Selected: extend `scripts/dev/benchmark.sh`, the existing Make targets, and the
existing pinned k6 image.

Rejected:

- a mandatory `ghz` path duplicates pinning, metadata, lifecycle, and reporting
  without closing a current k6 gap;
- a second standalone benchmark shell stack would duplicate source
  fingerprint and artifact rules;
- upstream gRPC-Go benchmarks omit this repository's policy and telemetry path.

Acceptance boundary: Go results use the current runner's metadata and
`benchstat` flow; gRPC load results land under `.artifacts/bench/grpc/` with
source, scenario, environment, mode, target, and unavailable-observable
metadata.

Reopen: k6 fails an accepted RPC cardinality, schema, load model, or required
metric that a focused maintained tool supplies.

### D2: production-composed benchmark server

Selected: add an ephemeral benchmark-only command under the reference service.
It:

- registers the existing Edition 2023/Opaque reference service;
- constructs the server through `grpcx.NewServer`;
- obtains the current production bounds from the pure canonical
  `config.DefaultGRPCServerConfig()` accessor also consumed by
  `defaultValues()`;
- applies the canonical `MaxConnections` with `netutil.LimitListener`, matching
  bootstrap's current listener mechanism;
- supplies an SDK meter provider, never-sampled SDK tracer provider, and
  admission instruments without an exporter;
- uses the production JSON logging handler with `io.Discard` as the sink so
  JSON encoding cost remains while filesystem or terminal I/O does not
  dominate;
- binds `127.0.0.1:0`, prints one machine-readable ready line, and handles
  interrupt/termination with bounded shutdown.

The command is compiled before measurement. Its compile/startup time is not
part of k6 latency. The runner parses the allocated port and owns termination,
wait, and artifact cleanup.

Rejected:

- the existing reference test server is bare `grpc.Server` and would measure
  the wrong path;
- importing benchmark fixtures into bootstrap or production packages would
  create a runtime/test ownership leak;
- binding `0.0.0.0` improves Docker convenience by weakening the local
  exposure invariant.

Acceptance boundary: all four methods are reachable from k6, standard policy
interceptors and the OTel stats handler run, reflection remains disabled, and
the process is gone after success or failure. A subprocess component test
builds and launches the real command, parses its actual ready record, asserts
the advertised listener is loopback, completes one generated unary call, sends
an OS termination signal, and observes bounded clean exit. A config unit test
proves that ordinary default loading and `DefaultGRPCServerConfig()` return the
same gRPC bounds.

### D3: Docker-to-loopback network path

Selected:

- Darwin uses Docker Desktop's `host.docker.internal:<allocated-port>`;
- Linux uses `--network host` and `127.0.0.1:<allocated-port>`;
- unsupported default platforms fail with a precise diagnostic;
- an explicit target/network override may point at an operator-owned local
  environment, but the runner records it and makes no capacity claim.

This avoids a wildcard host bind. It also keeps platform routing out of the k6
scenario.

Acceptance boundary: `grpc-inspect` needs no server; a real local run must
connect without changing server exposure.

Reopen: a supported Docker platform cannot route to the loopback listener.
Prefer an owned ephemeral container/network topology before permitting a
wildcard host bind.

### D4: Go microbenchmark attribution

Selected: add same-package `_test.go` benchmarks in `internal/infra/grpc`.
They use one benchmark-only unary `grpc.ServiceDesc` with generated
`wrapperspb.BytesValue` messages and construct these variants:

| Variant | Composition |
| --- | --- |
| `bare` | native `grpc.Server`, service only |
| `otel` | native server plus the same OTel gRPC stats handler shape |
| `policy` | correlation, admission, policy boundary, and error mapping using the actual package functions |
| `full_log_disabled` | `grpcx.NewServer` with successful INFO logging disabled |
| `full_json` | `grpcx.NewServer` with production JSON formatting to `io.Discard` |
| `full_handler_work` | full JSON path plus deterministic CPU work in the handler |

The fixture establishes one `bufconn` listener and one client connection
outside the timed loop. Each iteration performs one unary RPC, validates the
response, and includes Protobuf encode/decode, transport, selected server
layers, and handler response allocation. Payload sub-benchmarks use the
research sensitivity buckets that remain safe for local memory.

An SDK meter provider without export and a never-sampled SDK tracer provider
model the production no-export path. Admission uses real OTel instruments
rather than a no-op recorder in full variants.

Each fixture also exposes a deterministic untimed composition probe used by a
component test before benchmarking:

- the OTel variant proves a server RPC was observed by an in-memory metric
  reader;
- policy/full variants prove correlation metadata, admission recorder, and an
  injected policy counter on the actual server path;
- log-disabled proves a successful RPC writes no INFO access record;
- full JSON proves the successful RPC writes a parseable production JSON
  access record to a counting buffer;
- handler-work proves the deterministic response/checksum oracle.

Acceptance boundary: the component probe fails when a named layer is omitted;
`B.Loop`, `ReportAllocs`, `SetBytes`, `-benchmem`, and profile capture work;
fixture construction, composition probes, and cleanup are outside the timer.

### D5: k6 scenario and result contract

Selected: one schema-loaded script
`test/performance/grpc/all-cardinalities.js`. It imports the canonical
reference `.proto` directly and never requires reflection.

The schema and smoke oracle use the bounded one-based server-stream contract:
non-positive `count` produces no messages, `count` in `[1,1024]` produces
sequences `1..count`, and a larger count returns `ResourceExhausted` before
emitting a message. The reference service adds the same `1024` bound already
used by its client-stream aggregation. Schema comments are corrected before
regeneration so generated comments, runtime, and tests have one authority and
the `int32` loop cannot wrap at `MaxInt32`.

Modes:

- `smoke`: one VU completes unary, server-streaming, client-streaming, and bidi
  operations at least once; exact responses, counts, order, terminal success,
  and completion are thresholds;
- `synthetic`: an explicit warmup scenario precedes a fixed measured scenario;
  measured iterations rotate over all four operations and emit only
  measured-phase custom metrics.

Custom metrics:

- `grpc_bench_operation_successes`;
- `grpc_bench_unary_successes`;
- `grpc_bench_stream_successes`;
- `grpc_bench_messages_sent`;
- `grpc_bench_messages_received`;
- `grpc_bench_messages_processed`;
- `grpc_bench_correctness_failures`;
- `grpc_bench_terminal_failures`;
- `grpc_bench_unary_duration`;
- `grpc_bench_stream_duration`;
- `grpc_bench_message_lag` for correlated bidi messages.

The script uses async stream handlers and treats `end` without an earlier
`error` as terminal success. Every operation adds one explicit success/failure
sample tagged with `cardinality=unary|server_stream|client_stream|bidi`.
Success, stream, sent, received, and successfully processed message samples
carry the same cardinality tag where applicable. Smoke and synthetic thresholds
require a non-zero success denominator independently for all four cardinalities,
so an aggregate stream rate cannot hide an omitted method. Built-in
`dropped_iterations`, `data_sent`, and `data_received` complete the local result.

Default synthetic values are engineering sensitivity assumptions, not service
budgets: one VU, one shared connection per VU, a short warmup, and a bounded
measured duration. All are environment-overridable and recorded.

Acceptance boundary: k6 exits non-zero for a wrong payload/order/count,
unexpected terminal failure, missing per-cardinality success, a missing
processed-message sample, or dropped work. The summary contains non-empty
cardinality-tagged client-side rate, duration percentile, count, failure, and
byte metrics. Decision-grade server and generator observables are recorded as
unavailable.

### D6: profiling and optimization selection

Selected: reuse `make bench-profile` against the full Go benchmark variants
for CPU, memory, block, mutex, and trace evidence. Do not add automatic PGO,
compression, connection pools, worker pools, static windows, or buffer tuning.

Acceptance boundary: a profile identifies the scenario, source fingerprint,
package, benchmark pattern, and profile type. Any later optimization becomes a
new baseline/candidate task under `docs/benchmarking.md`.

## Material flows

### Go benchmark flow

1. Developer selects package, benchmark variant, payload bucket, count, and
   duration through the existing benchmark runner.
2. `go test` builds the same-package benchmark fixture.
3. The fixture constructs the chosen native or production server once, starts
   `bufconn`, and creates one client connection.
4. `B.Loop` invokes the unary method and validates the response.
5. Go emits timing/allocation samples; the runner writes source/environment
   metadata.
6. The existing comparison path consumes repeated baseline/current files with
   `benchstat`.
7. Cleanup closes client, server, listener, and SDK providers even on failure.

Failure is terminal when setup, RPC validation, cleanup, sample detection, or
metadata validation fails. No partial file becomes the current benchmark.

### gRPC smoke/synthetic flow

1. Developer runs the Make target with a mode and optional workload overrides.
2. The benchmark runner validates Docker, paths, mode, durations, counts, and
   artifact ownership.
3. For a real run, it builds the benchmark command, starts it on loopback,
   waits for the ready record, and resolves the Docker target.
4. The runner starts the digest-pinned k6 container with the repository mounted
   read-only and the artifact directory mounted writable.
5. k6 loads the canonical reference schema, opens one connection per VU, and
   executes the mode.
6. The script validates every RPC result and emits custom metrics; k6
   thresholds decide success.
7. The runner retains summary, log, server log, and metadata, then terminates
   and joins the server.

The final local evidence boundary is the successful k6 exit plus present
summary and metadata. It is not a production-capacity boundary.

## Go and file ownership

| Responsibility | Owner and placement | Dependency and authority | Proof owner |
| --- | --- | --- | --- |
| Layer-attribution fixture | `internal/infra/grpc/performance_test.go`, same `grpcx` package | May use unexported current interceptor owners; test-only dependency on OTel SDK and generated `wrapperspb`; no exported production seam | Focused benchmark run and profile |
| Canonical gRPC defaults | `internal/config/defaults.go` exports `DefaultGRPCServerConfig()` and `defaultValues()` consumes it | `internal/config` remains the only numeric default owner; benchmark code imports the pure accessor and applies `netutil.LimitListener` | Config default-equivalence unit test |
| Benchmark reference server | `examples/grpc-reference-service/cmd/benchmark-server/main.go` | Imports reference service/generated contract, `internal/config`, and `internal/infra/grpc`; no production package imports it | Command/unit lifecycle test or real runner smoke |
| Canonical RPC schema | Existing `examples/grpc-reference-service/api/proto/reference/v1/echo.proto` | Remains authoritative; generated Go remains derived; k6 loads the schema directly | `make proto-check`, k6 inspect |
| gRPC load scenario | `test/performance/grpc/all-cardinalities.js` | Depends only on k6 core modules and canonical schema | k6 inspect plus smoke/synthetic thresholds |
| Runner lifecycle and metadata | Existing `scripts/dev/benchmark.sh` | Reuses pinned image, fingerprint, artifact, Docker, and failure conventions | `benchmark-infra-check` and focused shell validation |
| Developer entrypoints | Existing `Makefile` | Adds `bench-grpc`, `bench-grpc-smoke`, and `bench-grpc-inspect`; no runtime configuration | Make dry/focused execution |
| Operator documentation | `docs/benchmarking.md` and `docs/build-test-and-development-commands.md` | Documents evidence level, variables, profiles, and claim boundary | `git diff --check` and command agreement |
| Initialized-profile ownership | composite `grpc-reference-benchmark` profile plus dedicated-path removal in `scripts/init-module.sh` | Shared commands/docs/runner branches survive only with `GRPC=enabled` and `REFERENCE_EXAMPLE=keep`; `grpcx` microbenchmarks remain runtime-owned | both generated profiles plus repeated-init residue checks in `template-init-check.sh` |

The dependency graph remains acyclic:

```text
benchmark command
  -> reference service + generated reference contract
  -> internal/config default accessor
  -> internal/infra/grpc

internal/infra/grpc performance_test
  -> grpc-go + OTel SDK test dependencies

scripts/dev/benchmark.sh
  -> compiled benchmark command + pinned k6 scenario
```

Production bootstrap, config, generated production contracts, and feature
packages do not depend on benchmark code.

## Cleanup, compatibility, and rollout

- Retain the existing HTTP and Go benchmark behavior unchanged.
- Add gRPC subcommands and Make targets; do not rename current targets or
  variables.
- Artifact deletion is limited to the owned
  `.artifacts/bench/grpc/<mode>/` files before a new run.
- Trap cleanup terminates only the PID started by the current runner and waits
  for it; it never scans or kills by process name or port.
- The generated reference code is not edited manually and no new schema is
  introduced. The `sequence` documentation fix is made in the canonical
  reference schema and regenerated.
- This is developer tooling only; no deployment, migration, compatibility
  window, or production rollout is triggered.

## Proof and reopen conditions

Technical design is satisfied when:

- the production-composed command and same-package benchmark require no
  benchmark-specific production API; the only new callable surface is the pure
  canonical config-default accessor consumed by runtime defaults too;
- k6 can parse the current Edition 2023 schema and inspect the scenario;
- focused Go benchmarks produce allocation samples;
- smoke proves every cardinality and cleanup;
- synthetic mode produces the required local result contract;
- existing benchmark infrastructure checks retain HTTP/Go coverage;
- changed Go, shell, JS, docs, and generated-authority gates pass.

If k6 cannot parse the current schema, reopen D5. The preferred next candidate
is an artifact mechanically derived from the canonical descriptor; a
hand-maintained proto3 mirror is not viable.

If Docker cannot reach loopback on a supported platform, reopen D3. Prefer an
ephemeral container/network topology over a wildcard bind.
