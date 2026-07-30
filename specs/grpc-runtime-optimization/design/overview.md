# gRPC runtime optimization design

status: ready

## Drivers and selected architecture

The fixed path must remain allocation-conscious, failure-visible, bounded in
cardinality, reversible by configuration/build input, and free of a new runtime
dependency. The selected design extends the existing owners:

1. `grpcx` receives one immutable access-log policy and applies it in the
   existing unary/stream interceptors.
2. The existing descriptor-backed OTel filter additionally excludes standard
   health methods unless explicitly enabled.
3. Make and the production Dockerfile pass one explicit `-pgo` value to the
   service build; no profile is committed by the template.

Adding a logging library, a metrics replacement, a connection pool, or a
transport tuning layer loses because each duplicates an existing owner without
a profile-backed bottleneck. Removing duration telemetry loses the primary RPC
latency SLI; the pinned `otelgrpc` version exposes duration as its material
server histogram, so no metric View is added.

## D1: access-log policy

`internal/infra/grpc.Config` carries `AccessLogSuccessSampleRate` and
`AccessLogSlowThreshold` beside the existing health flag. `NewServer` builds a
value policy once and passes it to both interceptors.

Each interceptor first checks health exclusion and `slog.Logger.Enabled`.
Only an eligible INFO path starts a timer. After the inner chain returns,
non-OK and slow calls log before sampling is considered. Remaining successful
calls use an allocation-free FNV-1a pass over the request ID; boundary rates
short-circuit and an absent ID fails open to logging. Emission uses
`Logger.LogAttrs` with typed attributes.

This preserves one completion record shape and avoids global RNG state,
contention, and per-call sampler allocation. Random sampling is rejected
because repeat attempts with the same correlation ID could disagree and the
hot path would gain a mutable shared owner.

Acceptance boundary: exact precedence tests pass for unary and stream
interceptors, direct construction rejects invalid values, and the benchmark
distinguishes full, sampled-out, and level-disabled success paths.

## D2: server health telemetry filter

The existing descriptor-backed `otelgrpc.WithFilter` remains the sole server
instrumentation admission point. It first requires a registered method, then
rejects the standard health prefix unless `TelemetryHealthChecks` is true.

This removes periodic probe spans and duration samples without changing health
handling or business RPC telemetry. A separate meter provider or a second
stats handler is rejected because it duplicates instrumentation and can
double-count business calls.

Acceptance boundary: a real health RPC produces no server span/metric by
default, opt-in restores both, and a registered business RPC plus unknown
method retain their current behavior.

## D3: PGO build lifecycle

`PGO_PROFILE` defaults to `off` in Make and `ARG PGO_PROFILE=off` in the build
stage. Ordinary local and image builds pass the value to `go build -pgo`.
`make build-pgo` requires a non-`off` file and validates it with
`go tool pprof -raw` before building.

An activated Docker profile must be inside the build context and passed by
repository-relative path. Derived delivery owns staging the private profile
and recording its source revision, workload, Go version, collection interval,
and hash. The template neither downloads it nor puts it in a public image
layer. `off` is the immediate rollback.

Automatic `default.pgo` discovery is rejected because an accidentally copied
or synthetic profile would silently change a release. A bundled generic
profile is rejected because the reference workload is not representative of a
derived service.

Acceptance boundary: ordinary builds report no profile dependency; missing and
malformed explicit profiles fail; a readable profile reaches `go build`; the
Dockerfile check and production image build remain valid.

## Go ownership and cleanup

- `internal/config/{types.go,defaults.go,validate.go}` owns keys, compatible
  defaults, and validation; `internal/config/grpc_config_test.go` owns the
  snapshot proof.
- `internal/infra/grpc/{config.go,server.go,interceptors.go}` owns runtime
  policy. Existing package tests and `performance_test.go` own behavior and
  cost proof. No exported sampler type is introduced.
- `cmd/service/internal/bootstrap/startup_grpc.go` maps the validated snapshot
  only.
- `Makefile` and `build/docker/Dockerfile` own local and image build flags;
  `docs/grpc.md`, `docs/build-test-and-development-commands.md`, and
  `env/.env.example` own operator guidance.
- No generated source, public contract, feature package, client package, or
  dependency changes.

The dependency direction remains config/bootstrap -> `grpcx`; `grpcx` imports
only existing leaf/runtime packages and the standard library.

## Material flows

Successful RPC: correlation creates/accepts request ID -> access-log policy
checks level/health -> inner policy and handler complete -> error/slow/sample
precedence decides emission -> typed completion record or no record -> caller
receives the unchanged response.

Health telemetry: gRPC stats tag -> registered-method lookup -> health opt-in
check -> existing OTel handler records or discards server signals -> health
handler and response remain independent.

PGO build: operator collects representative CPU profile -> private delivery
input is validated -> explicit path reaches `go build -pgo` -> ordinary
artifact/image validation runs -> rollback rebuild passes `off`.

## Reopen conditions

Reopen D1 if profiling shows the request-ID hash or timer is material, or an
audit owner requires every success. Reopen D2 if health duration becomes an
accepted SLI. Reopen D3 if the delivery platform cannot supply a private
build-context profile or multi-binary profiles require distinct build stages.
