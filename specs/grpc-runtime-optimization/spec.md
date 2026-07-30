# Bounded gRPC runtime optimization

status: ready

## Scope and non-goals

The template must make the fixed observability cost of native gRPC adjustable
without hiding failures, changing RPC semantics, or guessing a derived
service's capacity target. It must also provide an explicit production build
path for a representative Go PGO profile.

This change covers server access-log policy, routine server-side health
telemetry, and service-binary PGO activation. It does not change Protobuf
contracts, status mapping, admission, transport buffers, flow-control windows,
compression, keepalive, client connection ownership, or message layout.
Custom codecs, buffer pools, `vtprotobuf`, and experimental stream workers
remain out of scope until a profile identifies their owner.

## Behavior and contract delta

### OPT-1: bounded access-log work

When INFO logging is disabled, unary and streaming RPCs execute without
starting an access-log timer or constructing access-log attributes.

When INFO logging is enabled:

- health RPCs remain excluded unless `access_log_health_checks` is true;
- every non-OK terminal status is logged;
- a successful RPC at or above a positive `access_log_slow_threshold` is
  logged;
- every other successful RPC is selected deterministically from its validated
  request ID according to `access_log_success_sample_rate`;
- a missing request ID is logged rather than silently discarded.

The success sample rate is in `[0,1]`. `0` omits fast successful calls, `1`
records all of them, and the default remains `1` for compatibility. A zero
slow threshold disables the slow-success override; the default is zero because
the template has no service SLO from which to derive a threshold. Log records
keep the current method, status, duration, message, and correlation behavior.

### OPT-2: health telemetry noise is optional

Routine standard gRPC health calls are excluded from server-side
OpenTelemetry spans and RPC metrics by default. Business methods remain
instrumented, and unknown peer-supplied methods remain excluded.
`telemetry_health_checks=true` restores server health spans and metrics for
diagnosis. Client-side health telemetry is unchanged.

### OPT-3: representative PGO is an explicit build input

Local and production service builds use `-pgo=off` unless an operator supplies
an explicit profile path. A PGO build rejects an empty, missing, or unreadable
profile before claiming success. The template does not ship or synthesize a
`default.pgo`.

The profile must come from the same service binary and a representative
production-shaped workload, be retained as a revision/workload-labelled
private build input, and be refreshed when the workload, compiler, or hot
path materially changes. `PGO_PROFILE=off` is the rollback.

### OPT-4: deliberately unchanged behavior

RPC payloads, statuses, headers, health serving state, admission and shutdown
semantics, telemetry propagation, duration instruments for instrumented
methods, and generated contracts do not change. No numeric latency,
throughput, capacity, or PGO improvement is claimed without equivalent
baseline/candidate evidence.

## Invariants and edge cases

- Errors win over success sampling and are always logged while INFO is enabled.
- A positive slow threshold wins over success sampling.
- Health exclusion wins before log sampling.
- The sampling decision is stable for one request ID and performs no shared
  random-number or lock operation on the RPC path.
- NaN, infinity, out-of-range sample rates, and negative slow thresholds fail
  configuration validation.
- Disabling server health telemetry does not disable health RPC handling or
  client instrumentation.
- A PGO profile is never fetched from the network or generated implicitly by
  the build.

## Decisions, constraints, and authorities

- [`../grpc-performance/research/synthesis.md`](../grpc-performance/research/synthesis.md)
  owns the measured-candidate boundary and eliminated transport/library
  defaults.
- `internal/config` remains the runtime configuration authority;
  `internal/infra/grpc` owns RPC-edge policy; bootstrap only maps the validated
  snapshot.
- [`docs/benchmarking.md`](../../docs/benchmarking.md) owns performance claims
  and baseline/candidate equivalence.
- Official Go build/profile semantics and the pinned toolchain own PGO input
  validity.

## Success criteria and proof expectations

1. Focused tests falsify sampling precedence and both unary/streaming behavior,
   invalid config, health telemetry default/opt-in, and unchanged business
   telemetry.
2. Repeated production-composed microbenchmarks compare full logging,
   sampled-out success logging, and level-disabled logging with allocation
   evidence; correctness remains independently green.
3. `make build` proves the explicit non-PGO rollback, while `make build-pgo`
   accepts a readable representative profile and rejects missing or malformed
   inputs.
4. The production Dockerfile uses the same explicit PGO input contract and its
   ordinary build remains green with PGO off.
5. Environment and operator documentation state defaults, precedence,
   observability loss, activation, rollback, and stream/connection tuning
   boundaries.

## Risks, assumptions, and reopen conditions

The template assumes no organization-wide audit policy requires one successful
access record per RPC; the compatible default therefore remains full success
logging. Reopen OPT-1 if a derived service supplies an audit retention rule or
a measured SLO-based slow threshold.

Reopen transport or serialization design only when a representative CPU,
allocation, blocking, network, or flow-control profile identifies that owner.
Reopen PGO delivery when the build platform cannot securely provide a
repository-relative private profile input.
