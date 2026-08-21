# Operations And Proof

Load for health, readiness, drain, telemetry, deployment, or proof.

## Enable the capability

Choose the profile when initializing a repository:

```bash
make template-init \
  MODULE=github.com/acme/orders \
  CODEOWNER=@acme/platform \
  DATABASE=none \
  GRPC=enabled
```

Omitting `GRPC` is equivalent to `GRPC=none`: protobuf tooling, gRPC runtime
packages, examples, tests, dependencies, config, and CI profile checks are
removed. The selected value is recorded in `template.lock`, and repeating the
same initialization is byte-stable.

## Runtime behavior

Every RPC passes through the same interceptor chain, identical for unary and
streaming. The package doc on `internal/infra/grpc` is the single owner of the
order and of what each position buys a policy author — read it with
`go doc ./internal/infra/grpc`, and `builtinPolicies` in
`internal/infra/grpc/chain.go` is the executable copy. What the chain guarantees
a caller:

- `x-request-id` is validated or created and returned in response metadata. A
  single valid value is accepted; zero values, an invalid value, or two or more
  values all mint a fresh identifier, which is stricter than the HTTP listener's
  first-of-several header read.
- Panics never reach the caller and never disclose the panic value.
- Every business RPC runs under a deadline no later than its configured bound,
  so a caller that set none still cannot hold a handler indefinitely. The bound
  wraps the supplied policy interceptors and the handler alike.
- Admission is a non-blocking process-wide RPC semaphore; an RPC over the limit
  is shed as `RESOURCE_EXHAUSTED` rather than queued. Business RPCs and the
  standard health service hold separate budgets, so neither can starve the other.
- Service-supplied policy interceptors and generated handlers each answer
  through a sanitizing error boundary. A policy that means to choose its own
  status returns a plain `status.Error`; a handler classifies its domain error
  through `DomainErrors`. Anything else reaches the caller as `INTERNAL`, text
  included.
- Completion is logged once, outside error mapping, so the record carries the
  status the caller actually received.
- An RPC sanitized into `INTERNAL` also emits `grpc_unhandled_failure` at ERROR
  with `rpc.method` and `error_chain`. The chain is the unwrap chain rendered as
  Go types — `*fmt.wrapError -> *pgconn.PgError` — and never the error's message,
  which is the same text the boundary just refused to give the caller. Without
  that record an `INTERNAL` carries no detail, the access log carries only the
  code, and the failure is diagnosable solely by reproducing it. A dependency
  whose faults must survive this rendering publishes typed errors or sentinels
  rather than `errors.New`, which renders as `*errors.errorString` and names
  nothing.
- A step a service wants named in that chain wraps with `failure.Op` instead of
  `fmt.Errorf` — `create article -> store row -> *pgconn.PgError`. It is the only
  text the chain prints, because the name is a literal in the repository rather
  than anything the error carries; a handler with several writes that all fail
  the same way is what it exists for.

`grpc.health.v1.Health/Check` holds no admission slot at all, so a saturated
instance remains probeable. `Health/Watch` and any later health-service method
hold a slot in the health service's own budget, sized from
`MAX_CONNECTIONS` rather than `MAX_CONCURRENT_RPCS`. That separation is
load-bearing in both directions: grpc-go's client-side health checker keeps one
`Watch` open per subchannel for the connection's whole life, so counted as
business work every connected peer would permanently occupy a business slot, and
a shed `Watch` costs that caller the whole backend rather than one RPC — its
balancer stops selecting a peer whose watch failed with anything but
`UNIMPLEMENTED`. In the other direction, a peer opening watches cannot decide
what the service may serve. One watch per connection is the legitimate shape, so
the connection limit admits every well-behaved peer while still bounding a
hostile one.

A health budget under pressure is therefore a hostile-peer signal rather than a
capacity one. It is absent from the business active/shed instruments so those
keep describing business capacity alone; the no-label
`rpc.server.health.shed_requests` counter records health admission refusals.
Set `APP__GRPC__SERVER__ACCESS_LOG_HEALTH_CHECKS=true` only when per-RPC refusal
details are needed.

With the OIDC/JWT profile, health-service methods other than Check also require a
credential while Check remains public. Standard health methods are excluded
from routine access logs by default. The health service starts `NOT_SERVING`,
becomes `SERVING` only after the same dependency admission as HTTP readiness,
and returns to `NOT_SERVING` before drain. Its standard status semantics bypass
business-handler error mapping, so an unknown service remains `NOT_FOUND`.

Successful business access logs are compatible and complete by default:
`APP__GRPC__SERVER__ACCESS_LOG_SUCCESS_SAMPLE_RATE=1`. For a measured
high-throughput workload, the rate may be lowered to a value in `[0,1]`.
Non-OK terminal statuses are still logged, and a positive
`APP__GRPC__SERVER__ACCESS_LOG_SLOW_THRESHOLD` retains successful calls at or
above that duration before sampling is considered. The default `0s` disables
the slow override because the template cannot invent a service latency SLO.
Sampling is deterministic for the validated request ID, so it adds no shared
random-number lock. When INFO logging is disabled, the interceptor bypasses
timing and attribute construction entirely.

On shutdown, HTTP readiness and gRPC health enter drain together. After the
configured propagation delay, HTTP and gRPC drain concurrently under the same
remaining application shutdown budget. gRPC first uses `GracefulStop`; expiry
starts transport `Stop` and returns at the caller's deadline without joining
either library stop call. grpc-go cannot kill a Go handler: a handler that
ignores its canceled RPC context may continue until the process exits, so every
handler must release feature work and dependencies on cancellation.

OpenTelemetry client/server `StatsHandler`s cover unary and streaming protocol
spans and metrics. Repository interceptors add only request identity, access
status, low-cardinality business active/shed instruments, and the no-label health
shed counter. Request messages, metadata values, peer-controlled names, and raw
error text are not metric labels. Server protocol telemetry is limited to
methods present in the registered service descriptors; unknown peer-supplied
method paths are omitted instead of becoming unbounded span names or
`rpc.method` series. Routine server health spans and duration samples are also
omitted by default because probe frequency is not business traffic. Set
`APP__GRPC__SERVER__TELEMETRY_HEALTH_CHECKS=true` for focused diagnosis; health
handling and client-side telemetry are unchanged either way.

The process-wide RPC limit and per-connection stream limit constrain different
owners. Keep one long-lived shared client connection first. Add another
connection only when representative evidence shows client-side queueing at the
per-connection stream ceiling while process admission, CPU, memory, network,
and dependencies still have headroom. Raising either limit without the
matching concurrent payload-memory measurement is not a performance
optimization.

A business stream holds its admission slot for its entire life, so
`MAX_CONCURRENT_RPCS` is the peak of concurrent unary RPCs *plus* concurrent
streams, not of unary RPCs alone. A service publishing long-lived subscriptions
therefore sizes it from expected subscribers first and gives those streams a
duration or idle policy of their own; the template ships neither, because a
stream's lifetime belongs to the feature that owns it. Standard health watches
are the exception and are already excluded, on the budget above.

Admission is a concurrency budget, not a rate. It bounds how much work runs at
once, so a method that returns quickly can be called as often as a caller likes
without ever filling it, and one caller can hold the whole budget. Telling
callers apart needs an identity only the service can choose, which is why no
limiter ships here — the same reason `internal/infra/http` ships its rate-limit
seam unwired.

A per-caller limit is a supplied policy, installed at
`grpcx.Options.UnaryPolicy` and `StreamPolicy` like any other cross-cutting rule.
Three things it must get right, all of which the HTTP side already records:
exempt `grpc.health.v1.Health/Check`, or a limited platform evicts the instance,
and hash the key when it is derived from a credential — both in
`internal/infra/http/middleware_ratelimit.go`; and bound the key map, because a
key derived from caller-controlled metadata is a memory leak with a limiter
attached, which `internal/infra/http/ratelimit_keyed.go` states beside the bucket
that does the bounding. `httpx.KeyedRateLimiter` is that bucket. A service needing one
identity limited across both transports promotes it to a shared leaf package
first rather than importing the HTTP adapter from a gRPC policy.

Reflection, a registry, grpc-gateway, Connect, and gRPC-Web are absent by
design. Add one only for a concrete contract, security, or measured reliability
requirement.

## RPC and connection lifetime

`APP__GRPC__SERVER__UNARY_TIMEOUT` caps how long one unary RPC may occupy a
handler, and `STREAM_TIMEOUT` does the same for a stream. Both derive the RPC
context's deadline, so a caller deadline that is already earlier still wins and
neither can extend one. `0s` disables the cap for that kind.

| Key | Default | Why |
| --- | --- | --- |
| `UNARY_TIMEOUT` | `8s` | The value `HTTP__REQUEST_TIMEOUT` carries, so one service answers on one budget over both transports. A separate key, so a deployment can still move the two apart. |
| `STREAM_TIMEOUT` | `0s` | A long-lived stream's duration policy belongs to the feature that owns the stream. |

Cancellation is the protection, not the response. A handler that ignores its
context keeps its goroutine and its admission slot; this is the same accepted
limitation `internal/infra/http` records for its own request budget.

The liveness bounds are on by default and cannot end an RPC in progress: the
idle clock only runs while nothing is outstanding, and the ping bound closes
only when a ping goes unanswered, which means the peer is gone.

| Key | Default | Why |
| --- | --- | --- |
| `MAX_CONNECTION_IDLE` | `15m` | Reclaims a connection nobody is using. |
| `SERVER_PING_INTERVAL` | `1m` | Detects a vanished peer within roughly this plus the timeout, against gRPC's 2h default. |
| `SERVER_PING_TIMEOUT` | `20s` | gRPC's own default. |
| `MIN_CLIENT_PING_INTERVAL` | `10s` | The minimum a caller may ping. gRPC's own default rejects anything under five minutes, which would disconnect this repository's client half. |
| `PERMIT_PING_WITHOUT_STREAM` | `true` | Lets a caller keep an idle connection alive through a NAT or balancer idle timeout. |

Connection rotation is **off by default**, because it is the only bound here
that ends work in progress: at `MAX_CONNECTION_AGE` the connection is drained
with GOAWAY and force-closed once `MAX_CONNECTION_AGE_GRACE` expires, cutting
every RPC and stream still running. gRPC adds ±10% jitter to spread connection
storms, and reads a zero age as infinity.

Enable it behind an L4 balancer or any hop that pins a caller to one replica for
a connection's lifetime — without it, no existing caller discovers a new
replica. Accept in exchange that a stream outliving the age ends with
`UNAVAILABLE`: gRPC does not transparently resume a stream that has already
delivered a message, so the feature owning that stream handles it.

Startup refuses a negative value for any of these, an age with a non-positive
grace or a grace below the unary timeout, and a stream timeout at or above a
configured age. That last relation only refuses a budget that could never
decide; it does not promise the budget wins, because the two clocks start at
different moments — the stream's when the stream starts, rotation's when the
connection was accepted.

## Profile-guided optimization

Go PGO is available as an explicit build input, not a template default:

```bash
make bench-profile \
  BENCH_PACKAGE=./internal/infra/grpc \
  BENCH_PATTERN='^BenchmarkGRPCUnary/full_json/64B$' \
  BENCH_PROFILE=cpu \
  BENCH_WORKLOAD_ID=grpc-reference-unary

make build-pgo PGO_PROFILE=.artifacts/bench/profiles/cpu.pprof
make docker-build PGO_PROFILE=path/inside/build-context/service.pprof
```

The benchmark profile above proves only the build path. A release profile must
come from the same service binary under a representative production-shaped
workload. Retain its source revision, workload identity, Go version,
collection interval, and SHA-256 in the private delivery evidence. Refresh it
after a material compiler, workload, schema, or hot-path change. Do not commit
a synthetic template `default.pgo`: automatic discovery can silently apply a
stale profile to a later release. Every activated local or image build rejects
an absent or unreadable profile before compilation. Rebuild with
`PGO_PROFILE=off` for immediate rollback.

PGO does not replace correctness, race, contract, image, or benchmark proof.
Only equivalent repeated baseline/candidate evidence may claim an improvement.

<!-- profile:grpc-reference-benchmark:start -->
## Local smoke and synthetic benchmarks

For four-cardinality correctness and local synthetic sensitivity, use the
production-composed loopback server and digest-pinned Grafana k6 harness:

```bash
make bench-grpc-inspect
make bench-grpc-smoke
make bench-grpc
```

The harness builds
`examples/grpc-reference-service/cmd/benchmark-server` before measurement. The
command composes the generated reference service through the production
`grpcx.NewServer` adapter, applies the canonical production connection and
transport bounds, and listens only on an allocated `127.0.0.1` port. It does
not enable reflection or change the production service configuration.

The single canonical k6 scenario loads the Edition 2023 reference schema
directly and exercises unary, server-streaming, client-streaming, and
bidirectional-streaming RPCs. Inspect the schema and scenario on the exact
digest-pinned image before relying on a new host.

`bench-grpc-smoke` uses one VU and completes every cardinality once. Exact
payload, order, message count, one-based server-stream sequence, clean stream
termination, and a non-zero success denominator for each cardinality are
thresholds. Smoke timings are diagnostic wiring evidence only.

`bench-grpc` runs an explicit warmup followed by a completion window equal to
the configured RPC timeout, then a fixed measured interval. The separation
prevents an in-flight warmup stream from overlapping measured load. Measured
iterations rotate across all four methods. Custom metrics retain operation and
stream successes, sent/received/processed messages, correctness and terminal
failures, unary duration, stream lifetime, and correlated bidi message lag.
The summary retains p50, p95, and p99 trend values, k6 byte and dropped-work
metrics, and independently tagged success denominators. Warmup does not enter
those custom measured-phase metrics, and every streaming operation uses the
recorded RPC timeout.

The bounded defaults are sensitivity assumptions, not service budgets. Override
them only as part of a named workload:

```bash
make bench-grpc \
  GRPC_BENCH_WORKLOAD_ID=reference-1k-8msg-4vu \
  GRPC_BENCH_PAYLOAD_BYTES=1024 \
  GRPC_BENCH_STREAM_MESSAGES=8 \
  GRPC_BENCH_VUS=4 \
  GRPC_BENCH_WARMUP_DURATION=5s \
  GRPC_BENCH_DURATION=30s \
  GRPC_BENCH_RPC_TIMEOUT=10s
```

`GRPC_BENCH_PAYLOAD_BYTES` is bounded at 1 MiB and
`GRPC_BENCH_STREAM_MESSAGES` at 1024 so the harness cannot silently raise the
reference or production safety limits. One client connection is shared per k6
VU. On macOS the container reaches the loopback listener through
`host.docker.internal`; on Linux the runner uses host networking. Any other
platform fails closed rather than widening the listener.

Artifacts are mode-scoped below `.artifacts/bench/grpc/`:

- `summary.json`: evidence level, workload inputs, unavailable resource
  observables, and the complete k6 metric summary;
- `k6.log` and `server.log`: client and benchmark-server diagnostics;
- `run.meta`: source/schema/scenario digests, pinned image, platform route,
  server Go toolchain/OS/architecture, payload/concurrency/duration settings,
  and Docker identity;
- `samples.json.gz`: optional per-sample output when
  `GRPC_BENCH_RAW_SAMPLES=1`.

The runner validates artifact ownership before deletion, starts and records one
server PID, and always terminates and joins only that process. Compilation and
startup are outside the measured k6 phase.

A successful local synthetic run proves harness sensitivity under its recorded
inputs. It does not prove production throughput, sustainable capacity, or an
optimization. Those claims additionally require an equivalent representative
service/testbed, repeated comparable baseline and candidate runs, server CPU
and memory/GC, admission utilization and rejection, connection/stream counts,
network utilization, and load-generator CPU/network headroom. The local summary
records those decision-grade observables as unavailable. The general
measurement and comparison contract remains in
[`docs/benchmarking.md`](../benchmarking.md).
<!-- profile:grpc-reference-benchmark:end -->

## Railway boundary

The repository does not claim that a native gRPC service is reachable through
a Railway public HTTP domain. Use Railway private networking with the service's
internal DNS and explicit port for service-to-service traffic. For public
native gRPC, revalidate end-to-end HTTP/2 trailers on the current platform or
use Railway TCP Proxy with application TLS and hostname verification.

The existing `railway.toml` exposes and health-checks the REST listener only.
Enabling the gRPC runtime does not publish its port or prove public reachability.

## Focused proof

```bash
go test -vet=off \
  ./internal/reqctx \
  ./internal/infra/http \
  ./internal/infra/grpc \
  ./internal/infra/grpcclient \
  ./internal/infra/telemetry \
  ./cmd/service/internal/bootstrap

go test -vet=off -race \
  ./internal/infra/grpc \
  ./internal/infra/grpcclient \
  ./cmd/service/internal/bootstrap

go test -vet=off ./examples/grpc-reference-service/...
# Upstream/PostgreSQL profile process proof:
go test -vet=off -count=1 -tags=integration ./test/... -run GRPC
make proto-check
```

The upstream/PostgreSQL-profile integration test starts the production binary,
waits for standard gRPC health and existing HTTP readiness, then proves
coordinated SIGTERM shutdown. `DATABASE=none` removes container-backed process
tests; its bootstrap lifecycle test remains in the ordinary Go suite.

## Upstream references

- [health](https://grpc.io/docs/guides/health-checking/) and
  [graceful shutdown](https://grpc.io/docs/guides/server-graceful-stop/)
- [Railway private networking](https://docs.railway.com/private-networking) and
  [TCP Proxy](https://docs.railway.com/networking/tcp-proxy)
