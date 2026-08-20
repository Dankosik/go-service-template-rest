# Technical design — gRPC transport hardening

status: ready
Gate: the Go Ownership panel's three-`PASS` requirement was **not** met and was
deliberately overridden after the fourth round. [Review
disposition](#review-disposition) records the evidence for that decision and the
residual risk it accepts. Planning and Implementation consume this file knowing
the override, not in ignorance of it.

Realizes: [../spec.md](../spec.md) rules R1–R9 at its repaired revision. Behavior
is fixed there; this file fixes mechanism, ownership, and placement only.

Revision: repaired after the Go Ownership panel rejected the first candidate
(FAIL/CONCERNS/FAIL) and after the spec's own repair moved R4 and R5. Every panel
finding is dispositioned in [Review disposition](#review-disposition).

## System decisions

Only the forks current evidence left open are recorded. Where evidence admitted
one mechanism, the collapsing evidence is named instead of a manufactured
alternative.

### S1 — A policy may replace the context the work below it observes

**Driver.** R3 and R4 need the handler to observe a deadline. The current policy
shape cannot express that: `aroundRPC` receives `ctx` but its `call func() error`
closes over the *original* context in both adapters
([chain.go:31](../../../internal/infra/grpc/chain.go:31),
[chain.go:48](../../../internal/infra/grpc/chain.go:48)). A policy can observe
and replace the result; it cannot enrich the context. This is why correlation is
the one policy still written per RPC kind.

**Go vocabulary for the time bound.** The spec calls it a budget; the Go surface
calls it a deadline — `deadlineAround`, `builtinDeadline`, `deadline_test.go` —
following `httpx.RequestTimeout`, the repository's existing name for the same
bound on the other transport, and matching the `UnaryTimeout` and `StreamTimeout`
fields this change adds. "Budget" is already taken
three times in this package: `Config`'s own field doc and the admission limiter
both call the concurrency semaphore the budget, and `Server.Shutdown` calls its
drain deadline the shutdown budget. A fourth sense beside `admission_test.go`
would cost every later reader a lookup.

**Selected.** Widen the shape to `call func(context.Context) error`, so a policy
passes down whatever context it decided on. Existing policies pass the context
they received. `asStreamInterceptor` wraps the stream only when the context
actually changed, reusing the existing `serverStreamWithContext`
([interceptors.go:75](../../../internal/infra/grpc/interceptors.go:75)) rather
than declaring a second wrapper. That type's comment is already policy-neutral —
it names no reader, only "a streaming handler" — so no edit is owed there, and
adding one would introduce the per-reader coupling the comment was written to
avoid.

**Rejected — a second policy shape** (`func(ctx) (context.Context, func())`
adapted separately): adds a second concept and a second ordered list to the chain
builder, and the order between the two lists becomes a new place to get wrong.

**Rejected — a per-RPC-kind budget pair** like correlation: two copies of one
timeout rule, which is precisely the drift
[doc.go:88](../../../internal/infra/grpc/doc.go:88) names as the problem this
design exists to prevent. Correlation's exemption rests on its two halves being
*genuinely different* (`grpc.SetHeader` versus `stream.SetHeader`); the budget's
two halves would be identical. That exemption's stated second clause — "must also
replace the context the handler observes" — stops being a differentiator under
S1 and is corrected in `doc.go`.

### S2 — The budget is the outermost policy that can bound work

**Driver.** Anything placed outside the budget is unbounded. Anything placed
inside it is covered.

**Selected position:** correlation → access log → **budget** → recovery →
admission → policy error boundary → supplied policy → handler error boundary.

**Why.** The access log must stay outside so it records the status the caller
actually receives, including `DEADLINE_EXCEEDED`. Everything below the budget —
recovery's own work, admission, a supplied authentication policy that may reach a
remote issuer, and the handler — is then inside the bound. The HTTP transport
already fixed the same adjacency:
`AccessLog → RequestTimeout → MaxInFlight → … → Recover`
([router.go:112](../../../internal/infra/http/router.go:112)). gRPC keeps
recovery outside admission for its own recorded reason, which the budget does not
disturb.

**Bounded honestly.** Two pieces of RPC work stay outside the chain and therefore
outside the budget: unary message decode, which runs before the chain and is
bounded by the receive-message limit, and response send, which runs after the
chain unwinds and is bounded by the stream and connection limits. The spec's
pre-decode-shedding non-goal already accepts the first; `doc.go` records both so
"outermost policy that can bound work" is not read as "bounds every byte".

### S3 — One builder, two configured values, one composition point

**Driver.** R3 and R4 are one policy with two values. The invariant that must
survive is that no policy can reach unary RPCs and miss streaming ones.

**Selected.** `builtinPolicies` gains a budget parameter and is called **once per
chain at its existing composition point**,
[server.go:70](../../../internal/infra/grpc/server.go:70). The list is still
produced by one function, so a policy cannot exist in one chain and not the
other, and the order is still decided in one place. The invariant moves from *one
list value* to *one list builder*.

**The failure this must not have.** Calling `builtinPolicies` once and passing
the result to both chains still compiles, still passes
`TestBenchmarkVariantsCoverEveryBuiltinPolicy`, and silently gives streams the
unary budget. **Two** comments assert the invariant being replaced, and both
change: [server.go:68](../../../internal/infra/grpc/server.go:68), and
[doc.go:43-46](../../../internal/infra/grpc/doc.go:43), whose clause "both chains
are built from it, so a policy cannot reach unary RPCs and miss streaming ones"
is the same assertion phrased as an instruction to the next policy author. The
first candidate claimed the doc sentence carried only an order claim and could
stay; it carries both, in the paragraph a reader consults before touching the
chain.

**Second consequence, and its missing proof.** With two calls, nothing structural
forces both chains to receive the same `*admissionLimiter`, which is what makes
the admission budget process-wide
([interceptors.go:196](../../../internal/infra/grpc/interceptors.go:196)). The
limiter is constructed once in `NewServer` and passed to both calls.

The first candidate named `TestAdmissionLimitIsSharedAcrossUnaryAndStreamingRPCs`
as the guarantee's proof. It is not:
[interceptors_test.go:268](../../../internal/infra/grpc/interceptors_test.go:268)
constructs the limiter itself and adapts that one value to both interceptor
types, so it proves the limiter is shareable and passes identically whether
`NewServer` shares one or builds two. The replacement proof is a new server-level
owner — see `admission_test.go` in the file map — which fills the process-wide
budget through one RPC kind on a server built by `NewServer` and observes the
other kind shed.

**Rejected — passing the budget separately to `unaryChain`/`streamChain`**, as
`handlerErrors` already is: that parameter exists because the handler boundary's
*position* differs from the shared list's. The budget's position is identical in
both chains; only its value differs. Threading it separately would put one
policy's order in two places.

### S4 — Details attach where a classified mapping is rendered

**Driver.** R1 and R2 apply to a classified domain error and to nothing else.

**Selected.** `mappedStatus` is the only site holding a `problem.Mapped`, so it
is the attachment site. A cancelled RPC, an expired RPC, an unclassified error,
and a sanitized handler status never reach it, which is exactly the R1/R2
absence rule — the boundary falls out of the existing control flow rather than
needing a guard. Confirmed against `mapError`'s current arms and against
`problem.Classify` returning false for a nil mapper slice, which is what makes
`policyErrorBoundary` detail-free without a special case.

**Reason rendering.** The spec fixes upper snake case. The rendering is a
`strings.ToUpper` of the code, and the invariant that every catalog code renders
to a conforming reason is checked over `problem.All()`.

**Failure policy.** `status.WithDetails` can fail only for an OK code, which
`mappedStatus` cannot produce. It is still checked, and a failure returns the
status without details: a detail that cannot be attached must not cost the caller
its status.

**Domain delivery.** `mappedStatus` is reached through `handlerErrorBoundary` →
`mapError`. Threading two independent parameters through both would make every
signature carry an unnamed pair, so the mapper slice and the error domain travel
as one unexported value type. `policyErrorBoundary` passes its zero value, which
is correct: it never classifies.

### S5 — Keepalive bounds are flat, and no new configuration type is introduced

**Driver.** `grpcx.Config` is documented as bounds already validated by the
runtime config owner. Keepalive durations are exactly that, so the `Config`
versus `Options` fork collapses. What did not collapse is the *shape*.

**Selected.** Seven unprefixed flat fields on both `grpcx.Config` and
`config.GRPCServerConfig`: `MaxConnectionIdle`, `MaxConnectionAge`,
`MaxConnectionAgeGrace`, `ServerPingInterval`, `ServerPingTimeout`,
`MinClientPingInterval`, `PermitPingWithoutStream`. koanf keys are the snake-case
forms under `grpc.server.`, matching the existing flat `max_concurrent_streams`
and `max_header_list_bytes`. **No `GRPCKeepaliveConfig` type is created**, so no
new `exhaustruct` entry is needed either — the existing `GRPCServerConfig` entry
already covers the new fields.

**Rejected — a nested `Keepalive` struct**, which the first candidate selected.
Three reflection guards compare `grpcx.Config` against `config.GRPCServerConfig`,
and nesting damages all three:

| Guard | Under nesting |
| --- | --- |
| `TestGRPCServerConfigFillsEveryTransportBound` ([startup_grpc_test.go](../../../cmd/service/internal/bootstrap/startup_grpc_test.go)) | A nested struct is non-zero once one sub-field is set, so the "no bound left behind" check silently weakens |
| `TestServerConfigMappingFillsEveryTransportBound` ([config_parity_test.go](../../../internal/infra/grpc/config_parity_test.go)) | Same shape, same weakening |
| `assertTransportMirrorsDefaults` ([benchmark main_test.go](../../../examples/grpc-reference-service/cmd/benchmark-server/main_test.go)) | Looks each `grpcx.Config` field up by name in `config.GRPCServerConfig` and compares with `reflect.Value.Equal`, which reports false for two differently-typed structs — the guard breaks outright |

Flat keeps all three working with no change to their logic. Nesting would have
required inventing cross-type nested comparison semantics inside the one guard a
service author copies. Better call-site reading is not worth that.

**Optional bounds do not break the guards.** The spec makes rotation off by
default, so `MaxConnectionAge` defaults to zero while its grace keeps the spec's
10s — the grace is a value the rule uses once an age is set, not a second
disable switch, and defaulting it to zero would make "set only the age" a
startup refusal under S5's own positivity rule. The two "no field is zero" guards
set their own fixtures rather than reading defaults, so they still prove the
mapping. `assertTransportMirrorsDefaults` compares against defaults, so it reads
zero on both sides for the two fields the spec ships disabled — `MaxConnectionAge`
and `StreamTimeout`. Those two mirrors are vacuous until a deployment sets them;
every other new field, the grace included, stays covered.

**Naming.** grpc-go's `ServerParameters.Time` and `.Timeout` become
`ServerPingInterval` and `ServerPingTimeout`, and `EnforcementPolicy.MinTime`
becomes `MinClientPingInterval`. The library's names are ambiguous in a struct
that already has several durations; these say which side pings and what is
bounded.

**Validation.** `grpcx.validateConfig` and `internal/config.validateGRPCConfig`
each own the full rule set, as they already do for every other bound:

- every keepalive duration is non-negative, so the age has exactly two meanings
  rather than three;
- the liveness bounds and the client-ping minimum are positive;
- when `MaxConnectionAge` is positive, its grace is positive and at least the
  unary budget;
- when both the stream budget and `MaxConnectionAge` are positive, the stream
  budget is strictly smaller;
- `MaxConnectionAge` zero disables rotation and imposes no further rule.

The conditional rules are why `config_parity_test.go`'s shared rule set gains
cases rather than one more bound: two owners must agree on a rule that is
conditional, which is exactly the drift that file exists to catch.

### S6 — Address selection travels as this client's own default service config

**Driver.** R7 must not reopen the resolver route the client's trust boundary
closed.

**Selected.** `grpc.WithDefaultServiceConfig` beside the existing
`grpc.WithDisableServiceConfig`. Its own documentation states that disabling
service config affects the resolver only and that a supplied default is still
used, so the two compose without weakening
[propagation.go:17](../../../internal/infra/grpcclient/propagation.go:17)'s
invariant. The policy is a typed value rendering to a fixed JSON constant, in the
shape `PropagationPolicy` already establishes: a small unsigned enum with a
`valid()` predicate refused at construction.

**Placement — `Config`, not `Options`.** The panel returned this as contested,
because the cited mirror `Propagation` lives on `Options`. They differ:
`Options` is the trust and observability boundary, and address selection is
neither. Selection is determined by the shape of the target — one address or
many — and the target lives on `Config`. Recorded so the asymmetry with
`Propagation` is a decision rather than an oversight.

**Zero value.** Round robin is the zero value, so a hand-built `Config` gets the
distributing default rather than the pinning one, mirroring `PropagationNone`
being the zero value for the same reason: the value a caller did not choose is
the safe one.

**Claims this touches, corrected from the first candidate.** That candidate said
three places refuse service config without qualifying whose, and that the
architecture file is template-owned. Both were wrong. What is actually true:

- [propagation.go:32](../../../internal/infra/grpcclient/propagation.go:32) is
  the one unqualified claim, and gains "resolver-supplied".
- [doc.go:23](../../../internal/infra/grpcclient/doc.go:23) says
  "server-supplied", which is the wrong qualifier rather than a missing one, and
  is corrected.
- [repo-architecture.md:47](../../../docs/repo-architecture.md:47) and
  [:246](../../../docs/repo-architecture.md:246) already say
  "resolver-provided", so no qualification is owed. Its real delta is the "New
  outbound gRPC dependency" seam, which tells a service author which decisions it
  makes per neighbor and must now also name the address-selection policy and the
  client's liveness defaults beside the propagation choice it already enumerates.
- That file is **repository-owned, not template-owned**:
  [template-owned-purity-check.sh:102](../../../scripts/ci/template-owned-purity-check.sh:102)
  lists it in a `repo_owned` array that fails the build if it ever enters the
  manifest, because owning it would overwrite real repository decisions on the
  next sync. Nothing about it propagates to derived checkouts.
- [docs/grpc.md:370](../../../docs/grpc.md:370) is the claim R7 actually
  falsifies: "rejects … resolver-supplied service configuration so neither can …
  add a hidden retry/balancer policy". The client now installs its own balancer
  policy, so that clause is replaced, not qualified.

### S7 — The client owns its keepalive contract, both halves

**Driver.** R6 is a constraint between the shipped server's minimum accepted ping
interval and the shipped client's ping interval. R8 is the client's live
behavior. Both are the client's keepalive contract.

**Selected owner.** `internal/infra/grpcclient`, because the client is the party
that violates R6's invariant, and because R8 is its own behavior. The import
direction works: `internal/infra/**` is excluded from the
`feature_packages_no_adapters` depguard rule, `internal/config` does not import
this package, and `internal/infra/grpc/config_parity_test.go` is the committed
precedent for an adapter test reading `internal/config`.

**Client shape.** Flat on `Config`, matching S5: `KeepalivePingInterval` and
`KeepalivePingTimeout`. `PermitWithoutStream` is fixed true rather than
configured, because R8 states pinging an idle connection as the behavior, not as
an option. Recorded because it is the one place the two halves are deliberately
asymmetric — the server configures its permission, the client does not configure
its need.

### S8 — `Options.ErrorDomain` is guarded by the linter, not by a crossing test

**Driver.** The panel found, from two lanes independently, that a configured
value on `Options` escapes every guard: `exhaustruct` does not include
`grpcx.Options`, and the bootstrap crossing test reflects over `grpcx.Config`.
R2's production half — that the composition root always supplies a domain — would
have had no proof, and its failure mode is the silent one R2's own absence rule
makes invisible.

**Selected.** Add `grpcx.Options` to the `exhaustruct` include list, inside the
existing `profile:grpc` markers. The pattern is written against the **import
path**, not the package name: `^github\.com/example/go-service-template-rest/internal/infra/grpc\.Options$`.
This is the first entry in that list whose last path element differs from its
package name, which is exactly the case an author copying a neighbouring entry
gets wrong. Anchor it — `grpcclient.Options` also exists. The repository already uses this exact mechanism
for this exact reason: `oidcjwt.PolicyInput` is in that list with the recorded
note that a configured trust value must reach the composition root or a
deployment runs with it unset.

**Cost, measured rather than assumed.** Two review lanes independently added the
entry to a scratchpad copy of `.golangci.yml` and ran
`golangci-lint --enable-only=exhaustruct`. There are **two** production literals,
not one:

```
examples/grpc-reference-service/cmd/benchmark-server/main.go:71:3:
  grpcx.Options is missing fields TransportCredentials, UnaryPolicy, StreamPolicy
```

`newGRPCRuntime` is clean, and `service_test.go`'s literal is correctly excluded
as a test. So the benchmark server's literal gains those three names plus
`ErrorDomain`, and its file-map row owns that edit. The first candidate's "cost is
nil today" was false.

**What the benchmark supplies for `ErrorDomain`.** A constant naming the
reference service, not the empty string. That command exists to measure the
production-composed path, and an empty domain silently removes `ErrorInfo` from
every error it measures.

**What the rule does and does not prove.** It proves the field is *named* at every
production literal, not that it is non-empty. `ErrorDomain: ""` satisfies it. The
non-empty half comes from `internal/config`'s own refusal of an empty
`observability.otel.service_name`, which is a different owner and is recorded
here so the two are not mistaken for one guarantee.

**Durable consequence.** Every future non-test `grpcx.Options` literal must name
every field, including the example a service author copies.

**Rejected — moving `ErrorDomain` to `Config`:** its origin is
`cfg.Observability.OTel.ServiceName`, which is outside the
`config.GRPCServerConfig` that `grpcServerConfig` takes. Carrying it there means
widening that function to the whole `config.Config`, which the other three
crossings cannot supply.

**Rejected — extending the bootstrap crossing test to `Options`:** it would prove
the same thing at one call site while the linter proves it at every one.

**Source-of-truth record.** The error domain is the service's own identity, taken
from `observability.otel.service_name` because that is the repository's only
configured service identity and is already reused outside observability
(`startup_diagnostics.go`'s build-info payload). The consequence a reader must
know: renaming it for telemetry reasons changes a value remote callers match on.
`docs/grpc.md` records that; introducing a second key for one identity is the
rejected alternative.

## Material flows

Only the crossings this change alters.

### Unary RPC, budget path

`grpc-go dispatch → correlation → access log (starts its clock) → budget (derives
a context whose deadline is no later than the configured value) → recovery →
admission (holds a slot) → policy error boundary → supplied policy → handler error
boundary → handler`.

- The context crossing is the new one: the budget hands its derived context down,
  and each policy below passes what it received. For a stream the derived context
  reaches the handler through the existing `grpc.ServerStream` wrapper.
- A caller deadline earlier than the budget wins, because `context.WithTimeout`
  never extends a parent deadline. The budget is a cap by construction.
- On expiry the handler returns its context error, the handler error boundary
  renders `DEADLINE_EXCEEDED`, admission releases the slot as the handler
  returns, and the access log records the status the caller receives.
- A handler that ignores cancellation keeps its goroutine and its slot. Accepted
  and identical to the HTTP transport's recorded limitation.
- Health RPCs are exempt, by the same `isHealthMethod` predicate that already
  exempts them from admission and routine logging.

### Classified domain error, detail path

`handler returns a domain error → handler error boundary → mapError (cancel and
deadline answer first, then the trust rule) → problem.Classify → mappedStatus →
status carrying RetryInfo when the delay is positive and ErrorInfo when a domain
is configured`.

- Authority: `problem.Mapper` remains the single classification owner shared with
  the HTTP transport. The gRPC rendering adds representation, never
  classification.
- Values crossing the boundary are the rendered `problem.Code` and the mapper's
  own delay. No handler text, dependency identity, or peer value becomes a
  detail.

### Connection lifetime

`listener accepts (bounded by MaxConnections) → grpc-go serves → close on an
unanswered ping, or on the idle bound with no RPC outstanding, or — only when
rotation is configured — GOAWAY at the jittered age followed by force-close after
the grace`.

- With the shipped defaults only the first two paths are live, and neither can
  end an RPC in progress: the idle clock runs only while nothing is outstanding,
  and the ping bound fires only when the peer stops answering.
- Rotation, when configured, ends every RPC on the connection at the grace
  boundary. The spec owns that consequence and its configuration relations.
- Drain is unchanged: `StartDrain` still publishes `NOT_SERVING`, `GracefulStop`
  still owns shutdown, and a transport that self-closes signals the same
  condition variable `GracefulStop` waits on, so self-closing helps shutdown
  rather than racing it.

## Responsibility map

| # | Responsibility | Current evidence | Owner and action | Dependency / surface |
| --- | --- | --- | --- | --- |
| 1 | Policy shape that may enrich the RPC context | `aroundRPC` and both adapters in `chain.go` | `internal/infra/grpc/chain.go` — **change** | `call func(context.Context) error`; stream adapter wraps only on change, reusing `serverStreamWithContext` |
| 2 | The RPC time budget policy | none | `internal/infra/grpc/interceptors.go` — **add** `deadlineAround(timeout time.Duration) aroundRPC` | unexported; reuses `isHealthMethod` |
| 3 | Budget position and per-chain value | `builtinPolicies` in `chain.go`; its call site and invariant comment at `server.go:68-70` | `internal/infra/grpc/chain.go` and `server.go` — **change** | `builtinPolicies(log, accessLogs, admission, deadline)`; two calls, one shared `*admissionLimiter`; new `builtinDeadline` name constant |
| 4 | Status detail rendering | `mappedStatus`, `mapError`, `handlerErrorBoundary` in `status.go` | `internal/infra/grpc/status.go` — **change** | unexported `errorRendering{mappers, domain}`; new imports `genproto/googleapis/rpc/errdetails` and `protobuf/types/known/durationpb` |
| 5 | Keepalive bounds and the budgets, with their conditional validation | `Config`, `validateConfig` in `config.go` | `internal/infra/grpc/config.go` — **change** | seven flat keepalive fields plus `UnaryTimeout` and `StreamTimeout`; no new type |
| 6 | Error domain collaborator | `Options` in `config.go` | `internal/infra/grpc/config.go` — **change** | `Options.ErrorDomain string`; empty omits `ErrorInfo` |
| 7 | Applying keepalive to the native server | `NewServer` server options in `server.go` | `internal/infra/grpc/server.go` — **change** | appends `grpc.KeepaliveParams`, `grpc.KeepaliveEnforcementPolicy`; a zero age is passed through, which grpc-go reads as infinity |
| 8 | Refusing a nil service registration | `registerServices` in `server.go` | `internal/infra/grpc/server.go` — **change** | returns `error`; `NewServer` propagates |
| 9 | Runtime configuration for the new bounds | `GRPCServerConfig` | `internal/config/{types,defaults,validate}.go` — **change** | nine flat fields, their koanf keys, defaults, and the conditional rule set |
| 10 | Linter enforcement that a configured value reaches the composition root | `exhaustruct` include list | `.golangci.yml` — **change** | add `grpcx.Options` inside the `profile:grpc` markers |
| 11 | Configuration-to-adapter crossing | `grpcServerConfig`, `newGRPCRuntime` | `cmd/service/internal/bootstrap/startup_grpc.go` — **change** | carries the new bounds; supplies `ErrorDomain` from `cfg.Observability.OTel.ServiceName` |
| 12 | Client address-selection policy | none | `internal/infra/grpcclient/load_balancing.go` — **add** | exported `LoadBalancingPolicy` with `LoadBalancingRoundRobin` as zero value and `LoadBalancingPickFirst`; unexported `valid()` and service-config rendering |
| 13 | Client connection liveness and policy wiring | `Config`, `DefaultConfig`, `New`, `validateConfig` in `client.go` | `internal/infra/grpcclient/client.go` — **change** | `LoadBalancing`, `KeepalivePingInterval`, `KeepalivePingTimeout`; `grpc.WithKeepaliveParams`, `grpc.WithDefaultServiceConfig` |
| 14 | Direct dependency declaration | `go.mod` indirect block | `go.mod` — **change** | `genproto/googleapis/rpc` moves to direct; already pinned in `go.sum`, so `make mod-tidy-check` is the forcing gate |
| 15 | Package and operator contracts | both `doc.go`, `docs/grpc.md`, `docs/repo-architecture.md`, `propagation.go`, `resolver_live_test.go` | **change** | interceptor order, widened policy shape, corrected correlation-exemption reason, detail contract, selection and liveness. Service-config claims split three ways: `propagation.go:32` gains "resolver-supplied"; `grpcclient/doc.go:23` and `resolver_live_test.go:2` say "server-supplied", a wrong qualifier that is corrected; `docs/grpc.md:370`'s "add a hidden retry/balancer policy" is replaced, because the client now installs one |

Cross-package surfaces added: `grpcx.Options.ErrorDomain`,
`grpcclient.LoadBalancingPolicy` and its two constants, and flat fields on two
existing config structs. No new type, no new interface: no consumer substitutes
an implementation, and `LoadRecorder` remains the only inversion this transport
needs.

**Import direction, production code:** unchanged, verified with `go list` rather
than by inspection. `internal/infra/grpc` imports exactly `internal/problem` and
`internal/reqctx` plus external libraries; `internal/infra/grpcclient` imports
exactly `internal/reqctx`.

**Test builds add two edges**, both permitted by every depguard rule, since
`feature_packages_no_adapters` excludes `**/internal/infra/**`:

- `grpcclient`'s test package to `internal/config`, for R6's two default owners.
  Precedented by `internal/infra/grpc/config_parity_test.go`.
- `grpcx`'s test package to `internal/infra/http`, for R1's cross-transport
  invariant. `retryAfterSeconds` is unexported, so a gRPC-side restatement of
  `math.Ceil` plus a floor of one would prove a copy of the rule rather than the
  rule. The edge is one exported function, not the whole router:
  `httpx.RejectResponse` ([router.go:488](../../../internal/infra/http/router.go:488))
  returns the same `handleGeneratedResponseError` that renders `Retry-After`, and
  its only shared vocabulary is `problem.Mapper`, which `grpcx` already imports.
  The narrowing is about API coupling, not build cost: Go links whole packages,
  so importing `internal/infra/http` at all already pulls chi, `internal/openapi`,
  telemetry, and the authn seam into that test binary. What `RejectResponse` buys
  is that the gRPC suite depends on one exported function and one shared type
  rather than on router construction and its preconditions.

The first candidate declared neither edge and claimed import direction was
unchanged outright.

## File map

**Test-file criterion, stated once so the three analogous splits below agree.**
One file per proved behavior. A unit-level and a server-level proof of the same
policy are two behaviors, because they answer different questions — which rule
broke, versus what a caller sees. A cross-owner agreement between two default
sets is its own behavior and gets its own file, following the committed
precedent `internal/infra/grpc/config_parity_test.go`. Enforcement provenance —
whether grpc-go or this package applies a bound — is not a grouping criterion;
the first candidate used it to fold keepalive into `limits_test.go`, whose own
question is whether a server refuses oversized input, which a ping interval is
not.

| Path | Action | One present reason to exist | Notes |
| --- | --- | --- | --- |
| `internal/infra/grpc/chain.go` | change | The policy shape and the one place chain order is decided | Both adapters; `builtinPolicies` gains a `deadline time.Duration` parameter — named for the deadline, not the budget, because it sits beside `admission` in this file |
| `internal/infra/grpc/correlation_service_test.go` | change | One request ID across a real client and server, on unary and streaming RPCs | Its header claims the package's single streaming service "because it is the only test that needs one", which the harness move falsifies. It keeps its own handler, which closes over its observation channels, and switches to the shared registration helper so the package has one way to register a stream |
| `internal/infra/grpcclient/resolver_live_test.go` | change | Whether `New`'s closures hold together over a live connection | Its header says "server-supplied service config" — the same wrong qualifier corrected in `doc.go`. Its assertions survive R7 unchanged: a resolver-selected balancer is still never built, and `BuildOptions.DisableServiceConfig` stays true |
| `internal/infra/grpc/server.go` | change | The lifecycle adapter and native server construction | Two `builtinPolicies` calls; two keepalive server options; `registerServices` returns an error; **the invariant comment at line 68 is replaced here**; and line 71's `handlerErrorBoundary(options.DomainErrors)` becomes the one production site that builds S4's `errorRendering` from both the mapper slice and `Options.ErrorDomain` — a construction that compiles with a zero domain and silently drops every `ErrorInfo` |
| `internal/infra/grpc/interceptors.go` | change | The server policy implementations and the predicates they share | Gains `deadlineAround`. Its admission comment at line 195 is replaced: it asserts "one limiter value backs both chains", which S3 turns from structural into a `NewServer` convention, and cites `TestAdmissionLimitIsSharedAcrossUnaryAndStreamingRPCs` as end-to-end proof, which S3 disqualifies |
| `internal/infra/grpc/status.go` | change | This transport's two error boundaries and the status they render | Gains `errorRendering` and detail attachment |
| `internal/infra/grpc/config.go` | change | The validated bounds and the composition-root collaborators | Nine flat fields, one `Options` field, the conditional rule set |
| `internal/infra/grpc/doc.go` | change | The package contract and interceptor order | Budget position and what it does not bound; widened policy shape; corrected correlation-exemption reason; detail contract; keepalive |
| `internal/infra/grpc/deadline_test.go` | add | The time budget observed through a real server, where a unit test cannot see admission release or the stream wrapper | Slot release, both RPC kinds, stream default off and configured on |
| `internal/infra/grpc/interceptors_test.go` | change | The interceptor policies as units, driven directly so a failure names the rule that broke | Gains `deadlineAround`'s unit cases including the health exemption; already imports `testing/synctest` for a time-driven policy. Three further edits: `TestAdmissionLimitIsSharedAcrossUnaryAndStreamingRPCs` is renamed to what it proves — one limiter value serves both interceptor types — because `admission_test.go` now owns the claim its current name makes; its direct `mapError` call and the `mappers` field of that test's case struct follow S4's `errorRendering` signature; and the header's closed list of owned policies widens to include the deadline |
| `internal/infra/grpc/admission_test.go` | add | The process-wide admission budget observed through a server built by `NewServer` | Fills the budget from one RPC kind, observes the other shed. This is the proof S3's two-call composition needs and that `interceptors_test.go`'s unit test cannot give |
| `internal/infra/grpc/keepalive_test.go` | add | Connection lifetime: liveness bounds that cannot cut live work, and rotation that can | R5 under shortened bounds, plus R4's clause that a live stream survives an idle interval and an answered ping |
| `internal/infra/grpc/error_details_test.go` | add | The client-visible detail contract | Presence and absence, reason shape over `problem.All()`, domain omission, no handler text, and the cross-transport retry invariant |
| `internal/infra/grpc/config_parity_test.go` | change | The two bound owners held to one answer | Four edits, the fourth being its stale three-crossing comment named in Cleanup. `serverConfigFromRuntime` carries the new bounds. `TestServerConfigMappingFillsEveryTransportBound`'s env corpus gains the nine new `APP__GRPC__SERVER__*` keys, or its no-zero-field assertion fails nine times. The conditional rules are the both-directions kind, so they join `TestAccessLogRulesMatchConfigValidation`'s corpus rather than the containment test — which means that test and the file header's "the two access-log rules" framing both widen beyond the access log |
| `internal/infra/grpc/server_test.go` | change | Construction and lifecycle proof | Gains the nil-registration case |
| `internal/infra/grpc/harness_test.go` | change | The shared server harness | Three edits, each with its declaration named so no choice is left open. (1) `testServerConfig()` supplies the liveness bounds and leaves rotation unset — required, because `config_parity_test.go` runs that fixture through `validateConfig`. (2) A third stand-up path returning the listener address alongside the server, so R5's vanished-peer case can dial a raw `net.Conn` and abandon it; neither `startTestServer` (bufconn) nor `serveOverTCP` (dials through `grpcclient.New`, which pings on its own under R8) can. Its header's "the two ways" enumeration widens. (3) `registerStreamTestService`, mirroring the existing `registerUnaryTestService`: a parameterized registration taking the handler, so `deadline_test.go`, `admission_test.go`, and `keepalive_test.go` each get a stream through a real server, and `correlation_service_test.go` keeps its own handler while losing the sole-caller pin. Its existing `testStreamFullMethod` constant is currently a bare string used only in `grpc.StreamServerInfo` literals; registering under it gives one constant two senses, so the registered method takes its own name |
| `internal/infra/grpc/docs_test.go` | change | The published mapping table held to the code | `mappedStatus` call site follows the signature |
| `internal/infra/grpc/performance_test.go` | change | The measured subset of the real chain | Its guard forces three edits, not one: `knownBuiltinPolicies` gains `builtinDeadline`, `policyVariantExcludes` decides whether the budget is measured, and both `builtinPolicies` call sites follow. Its `handlerErrorBoundary(nil)` call also follows S4's `errorRendering` signature |
| `internal/infra/grpcclient/load_balancing.go` | add | The address-selection policy and the exact service config it renders to | Named for the behavior it owns; it holds no `balancer.Builder`, unlike `resolver.go`, which is named for an extension point it genuinely wraps |
| `internal/infra/grpcclient/client.go` | change | Bounded, instrumented connection construction | Three `Config` fields, two dial options |
| `internal/infra/grpcclient/propagation.go` | change | The correlation trust boundary and its reserved keys | The service-config closure claim at line 32 gains "resolver-supplied" |
| `internal/infra/grpcclient/doc.go` | change | The package contract | Selection and liveness; the same qualification |
| `internal/infra/grpcclient/load_balancing_test.go` | add | Whether a client reaches every resolved backend | Two loopback servers under each policy. Stand-up mechanism, decided here because `New` hard-codes `grpc.WithResolvers` and exposes no dialer seam: register **an additional unique scheme** through `resolver.Register`, using grpc-go's `resolver/manual` builder rather than the package's `nopResolver`, whose header enumerates the two proofs that need it. Two constraints. It needs none of the child-process isolation `resolver_selection_test.go` uses — the harness records that isolation covers the resolver registry *and* `resolver.GetDefaultScheme()`, but scoped to that file's own mutation of the default; adding a scheme nobody else names relocates nothing. It does need registration from a **sequential top-level test**, because `resolver.Register` writes the unsynchronized registry map that `resolver.Get` reads on every `grpcclient.New`, and this package's parallel tests reach that read — a review lane reproduced the race |
| `internal/infra/grpcclient/keepalive_parity_test.go` | add | Two default owners held to one answer: the shipped client's ping interval against the shipped server's minimum accepted interval | R6. Its own file, following `internal/infra/grpc/config_parity_test.go` — a cross-owner agreement gets its own owner rather than joining either side's behavior tests |
| `internal/infra/grpcclient/keepalive_test.go` | add | This client's live keepalive behavior | R8: an idle connection is pinged, observed through a server whose enforcement policy rejects the ping |
| `internal/infra/grpcclient/client_test.go` | change | Construction and bound proof | `DefaultConfig` completeness; refusal of an invalid policy |
| `internal/config/types.go` | change | The configuration shape | Nine flat `GRPCServerConfig` fields |
| `internal/config/defaults.go` | change | The canonical defaults | New keys and values, each named in the spec's R3, R4, R5, R6, and R8 default blocks. Two default to zero because the spec ships them disabled — `max_connection_age` and `stream_timeout`. The grace defaults to 10s rather than zero, so that setting an age alone is a working configuration rather than a startup refusal |
| `internal/config/validate.go` | change | The configuration rules | The conditional keepalive and budget rule set |
| `internal/config/grpc_config_test.go` | change | The gRPC configuration rules' proof | New bounds and every conditional rule, including each refusal |
| `internal/config/snapshot_contract_test.go` | change | Every known config leaf key held to the typed shape | Two sentinel maps each gain nine `grpc.server.*` leaves; `reflect.DeepEqual` against a reflection-derived truth means this fails deterministically otherwise |
| `env/.env.example` | change | The operator-visible key inventory | Nine `APP__GRPC__SERVER__*` keys inside the existing `profile:grpc` block |
| `cmd/service/internal/bootstrap/startup_grpc.go` | change | The one crossing from configuration to transport bounds | New bounds and the error domain |
| `cmd/service/internal/bootstrap/startup_grpc_test.go` | change | Proof that the crossing leaves no bound behind | Fixture extended so every new field is non-zero at the source; the top-level reflection is correct as written under S5's flat shape |
| `examples/grpc-reference-service/cmd/benchmark-server/main.go` | change | The measured production-composed server | `settingsFromDefaults` carries the new bounds. Its `grpcx.Options` literal at line 71 additionally gains `TransportCredentials`, `UnaryPolicy`, `StreamPolicy`, and `ErrorDomain`, because S8's lint entry reaches it. Its "third place that crossing is written out" comment becomes the fourth |
| `examples/grpc-reference-service/cmd/benchmark-server/main_test.go` | change | Proof that the benchmark mirrors the canonical defaults | Comparison logic is unchanged under S5's flat shape. One edit is owed: its "written out twice more" comment is a fourth stale three-crossing claim. The guard is vacuous for the two disabled-by-default fields, `MaxConnectionAge` and `StreamTimeout`, so `settingsFromDefaults` dropping either is undetectable there until a deployment sets it |
| `examples/grpc-reference-service/service_test.go` | change | The reference service's own composition through the production adapter | **The fourth crossing.** Carries the new bounds. Its comment claims parity with one peer, not three, so the stale clause is "exactly one way to fill Config rather than two" — under a four-crossing set that count is wrong |
| `go.mod` | change | The module's declared dependencies | `genproto/googleapis/rpc` becomes direct |
| `.golangci.yml` | change | Repository lint policy | `grpcx.Options` joins the `exhaustruct` include list |
| `docs/grpc.md` | change | The service author and operator contract | New keys and defaults, detail contract, selection default, keepalive, the error-domain coupling. Two existing lists also widen under S2: the enumeration of what wraps a supplied policy, and the "what the chain guarantees a caller" list — both now have a deadline around the policy slot and the handler |
| `docs/first-production-feature.md` and `docs/project-structure-and-module-organization.md` | change | Two further carriers of the outbound-gRPC decision set | Each enumerates the per-neighbor propagation choice that `repo-architecture.md`'s seam does, so each gains the address-selection policy and liveness defaults. `first-production-feature.md`'s "a resolver-selected balancer cannot silently bypass its metadata policy" stays true and becomes misleading beside a client that installs its own balancer policy |
| `docs/repo-architecture.md` | change | The recorded extension seams | Its service-config clauses are already qualified and owe nothing. The delta is the "New outbound gRPC dependency" seam: it enumerates the decisions a service author makes per neighbor, and must now name the address-selection policy and the client's liveness defaults beside the propagation choice |

No file is removed. No compatibility path is retained: every changed signature is
unexported or internal to this module.

## Cleanup

The widened `aroundRPC` shape replaces the narrow one outright; every call site
changes in the same commit.

**Four** comments claim a three-crossing set and all four are corrected, because
the spec now fixes four:
[startup_grpc.go:89](../../../cmd/service/internal/bootstrap/startup_grpc.go:89),
[config_parity_test.go:45](../../../internal/infra/grpc/config_parity_test.go:45),
the benchmark server's `main.go:156`, and its `main_test.go:173`. The first
candidate named three and explicitly excused the fourth.

Both invariant comments named in S3 are replaced, not supplemented:
[server.go:68](../../../internal/infra/grpc/server.go:68) and
[doc.go:43-46](../../../internal/infra/grpc/doc.go:43).

Four comments assert an ownership or proof claim this change breaks, and each is
corrected with its file:
[interceptors.go:195](../../../internal/infra/grpc/interceptors.go:195), whose
"one limiter value backs both chains … proves it end to end" is false in both
halves after S3;
[harness_test.go:1](../../../internal/infra/grpc/harness_test.go:1) and
[correlation_service_test.go:5](../../../internal/infra/grpc/correlation_service_test.go:5),
which between them pin the package's single streaming service to one caller that
is about to have three more; and
[client.go:17](../../../internal/infra/grpcclient/client.go:17), which scopes
`Config` to "the fixed target and finite per-call transport bounds" while it
gains address selection and connection liveness, neither of which is per-call.

Four further comments go stale in files already in the map and are corrected with
them: [config.go:66](../../../internal/infra/grpc/config.go:66) and
[config_parity_test.go:10](../../../internal/infra/grpc/config_parity_test.go:10),
both of which scope the two owners' duplicated rules to the access log;
[chain.go:10](../../../internal/infra/grpc/chain.go:10), where `aroundRPC`
"observes or replaces the result" and now also the context; and
[docs/grpc.md:379](../../../docs/grpc.md:379), which says the server does not
apply the HTTP request timeout to streams — still true, and now sitting beside a
gRPC unary budget defaulting to the same value, which is what makes it
misleading.

`docs/grpc.md`'s statement that keepalive tuning is absent by design is replaced
by the configured contract; its statements about reflection, gateways, and
registries stay accurate and stay.

## Proof ownership

| Rule | Proof owner | Level |
| --- | --- | --- |
| R1, R2, and the cross-transport retry invariant | `internal/infra/grpc/error_details_test.go` | focused package |
| R3, R4 budget as units | `internal/infra/grpc/interceptors_test.go` | focused package, `synctest` |
| R3, R4 budget through a server | `internal/infra/grpc/deadline_test.go` | focused package, `-race` |
| R4's keepalive clause and R5 | `internal/infra/grpc/keepalive_test.go` | focused package, real transport, shortened bounds |
| The admission budget stays process-wide across S3's two calls | `internal/infra/grpc/admission_test.go` | focused package, through `NewServer` |
| R4/R5 configuration relations, refused at startup | `internal/config/grpc_config_test.go` and `internal/infra/grpc/config_parity_test.go` | focused package |
| R6 | `internal/infra/grpcclient/keepalive_parity_test.go` | constants |
| R8 | `internal/infra/grpcclient/keepalive_test.go` | focused package, ≥10s wall clock |
| R7 | `internal/infra/grpcclient/load_balancing_test.go` | focused package, two loopback servers |
| R9 | `internal/infra/grpc/server_test.go` | focused package |
| Crossing completeness, all four | `startup_grpc_test.go`, `config_parity_test.go`, benchmark `main_test.go`, `service_test.go` | focused package |
| Config key inventory | `internal/config/snapshot_contract_test.go` | focused package |
| `ErrorDomain` reaches the composition root | `exhaustruct` via `make lint` | lint gate |
| Unchanged behavior | the existing gRPC and grpcclient suites | focused package, `-race` |

Concurrency and lifecycle are triggered by R3, R4, and R5, so those packages run
under the race detector. R5's proof uses shortened bounds through a real
transport, as the spec's criteria now state: `testing/synctest` does not apply,
because the timers under test are inside grpc-go's transport goroutines rather
than in repository-owned code.

R8 has a floor the server side does not: grpc-go clamps a client ping interval
below 10s up to 10s, so observing a ping on an idle connection costs at least
that in wall clock and the shortening lever R5 relies on does not exist there.
That is the cost of proving R8 at all, recorded rather than designed around.

No benchmark is required; `performance_test.go` changes only to keep its guard
honest.

## What would invalidate this file map

- If `deadlineAround` grows a second responsibility — an idle bound distinct from a
  total bound — `interceptors.go` gains an independently changing reason and the
  budget moves to its own file.
- If keepalive grows enough fields that flat naming stops reading, the nesting
  fork reopens and the three reflection guards must be repaired together, as S5
  records.
- If a second policy needs context enrichment *and* per-kind mechanics,
  correlation's exemption stops being a single exception and the chain needs a
  recorded rule rather than a documented special case.
- If a service needs field-level violation details, `status.go` gains a second
  detail-shaping responsibility and the detail rendering moves out of it.

[Implementation](../../../docs/spec-first-workflow/phases/implementation.md#execute)
owns adapting the placement when that evidence appears in the real code.

## Review disposition

The Go Ownership panel rejected the first candidate: responsibility and
execution-path ownership **FAIL**, package and dependency architecture
**CONCERNS**, file cohesion and naming **FAIL**. Every finding is dispositioned
below. This revision is the fixed candidate and has **not yet been re-reviewed**;
all three lanes are owed it, since none returned a `PASS` receipt that survives.

**Reopened Specification and closed there:** the maximum-age contradiction with
R4, the unachievable retry-parity claim, the reason spelling, and the
three-versus-four crossing count. The spec is now `ready` after three independent
rounds.

**Repaired here** — this table is the round-1 log; where a later round reversed
one of its dispositions, the later section says so and the file map above is the
current answer.

| Finding | Disposition |
| --- | --- |
| Per-chain call site and invariant comment missing from both maps, Cleanup naming the wrong owner | Row 3 and the `server.go` file-map row now own the composition point; Cleanup points at `server.go:68` |
| The shared `*admissionLimiter` loses its structural guarantee under two calls | Recorded in S3 with its one end-to-end proof named |
| `Options.ErrorDomain` escapes every guard | S8: `grpcx.Options` joins `exhaustruct`, the mechanism the repository already uses for `oidcjwt.PolicyInput` |
| `snapshot_contract_test.go`, `env/.env.example`, benchmark `main_test.go`, `service_test.go`, `go.mod`, `propagation.go`, both `harness_test.go`, `docs/repo-architecture.md` omitted | All in the file map |
| Nesting damages three reflection guards | S5 selects flat and drops the new type and its lint entry |
| `KeepaliveConfig` named only three of seven fields | S5 names all nine new fields and their key forms |
| Import-direction claim false for the test build | Scoped to production, with the test edge named |
| `balancer.go` named for an extension point it does not implement | Renamed `load_balancing.go` |
| `deadline_test.go` duplicating `interceptors_test.go`'s reason | Split: units to the existing owner, server-level behavior to the new file, each with its recorded reason |
| `keepalive_test.go` duplicating `limits_test.go`'s reason | Server-side keepalive joins `limits_test.go`; no new file |
| `keepalive_parity_test.go` carrying two independent reasons | Renamed `keepalive_test.go`, whose one reason is the client's keepalive contract at both levels |
| `LoadBalancing` on `Config` versus `Options` | S6 records the reason and the deliberate asymmetry with `Propagation` |
| Whether zero keepalive values are accepted | S5's validation rules pin it |
| Server nests while client flattens; server configures `PermitWithoutStream` while client does not | Both resolved: flat on both, and S7 records why the permission is asymmetric |
| Error-domain coupling to `observability.otel.service_name` unrecorded | S8 records it, and `docs/grpc.md` carries it for operators |
| S2's absolute "outside the budget is unbounded" framing | Bounded honestly: decode and send are named as outside it |
| Correlation-exemption reason going half-stale under S1 | Named as a `doc.go` edit in row 15 |

### Second panel round

The repaired candidate went back to all three lanes: responsibility **FAIL**,
package architecture **CONCERNS**, file cohesion **FAIL**. Three blockers, each
found by more than one lane and two of them supported by an executed
`golangci-lint` probe rather than by reading:

| Finding | Disposition |
| --- | --- |
| S8's "cost is nil today" false — a second production `grpcx.Options` literal in the benchmark server breaks `make lint` | S8 carries the measured output; the benchmark row owns the four names it gains and the `ErrorDomain` value it supplies |
| S3 named `TestAdmissionLimitIsSharedAcrossUnaryAndStreamingRPCs` as the shared-limiter proof; that test builds the limiter itself and never reaches `NewServer` | S3 records the gap; `admission_test.go` is the new owner |
| S3 said `doc.go:43-46` carries only an order claim and stays; it carries the one-list-value invariant too | Both comments are in S3 and Cleanup |
| `docs/repo-architecture.md` called template-owned; it is repository-owned by `template-owned-purity-check.sh`, and its cited claims are already qualified | S6 corrected in both directions, with the real seam delta recorded |
| `grpcclient/doc.go:23` says "server-supplied", a wrong qualifier rather than a missing one | S6 |
| A fourth three-crossing comment, in the file the map said needed no rework | Cleanup names four; the benchmark `main_test.go` row owns its two edits |
| Three inconsistent criteria for three analogous test splits | One criterion stated once above the file map; `keepalive_test.go` and `keepalive_parity_test.go` follow it, reversing two earlier merges |
| R1's cross-transport proof implies an undeclared `grpcx` → `internal/infra/http` test edge | Both test edges declared, with the reason the edge is unavoidable |
| R4's keepalive clause had no proof owner | `keepalive_test.go` owns it |
| `grpcclient/harness_test.go` already takes `serverOptions ...grpc.ServerOption`, so its row was a no-op | Row dropped |
| `config_parity_test.go`'s two corpora; which one takes the conditional rules was unstated | Named, with the header-claim consequence |
| `performance_test.go` forces three edits, not one | Named |
| Four further stale comments not enumerated | In Cleanup |

### Third panel round

Verdicts: responsibility **FAIL**, package architecture **CONCERNS**, file
cohesion **FAIL**. Findings were markedly smaller and heavily overlapping — two
lanes independently raised the same top item — and no system decision moved.

| Finding | Disposition |
| --- | --- |
| S5 said `MaxConnectionAge` **and its grace** default to zero; the spec ships a 10s grace. With a zero grace, setting only an age would be refused at startup by S5's own positivity rule | S5, the `defaults.go` row, and the benchmark `main_test.go` row corrected to a 10s grace; the mirror guard is vacuous for the age alone, so that row owes one edit, not two |
| The `docs/repo-architecture.md` file-map row still prescribed the qualification S6 had just shown is not owed, and dropped the seam delta S6 named. Raised by two lanes | Row replaced with the seam delta |
| `interceptors.go:195` asserts "one limiter value backs both chains" and cites the test S3 disqualifies; absent from Cleanup. Raised by two lanes | In Cleanup and in that file's row |
| Three new test files each need a stream through a real server, while two comments pin the package's single streaming service to one caller | The registration moves to `harness_test.go`; both comments are in Cleanup |
| `load_balancing_test.go` had no stand-up mechanism: `New` hard-codes `WithResolvers` and exposes no dialer seam | Decided in that row — register an additional unique scheme, which needs none of the child-process isolation the default-scheme tests use |
| The declared `grpcx` → `internal/infra/http` edge was justified by "the only way is to drive the router"; `httpx.RejectResponse` is exported and reaches the same renderer | Edge narrowed to that one function, with the cost of the alternative recorded |
| No row named where `Options.ErrorDomain` is consumed | `server.go:71` named, with the silent-zero failure mode |
| `budget_test.go` named the fourth "budget" sense in a package that already has three | Renamed throughout: `deadlineAround`, `builtinDeadline`, `deadline_test.go`, aligning with `Config.UnaryTimeout` |
| `serverStreamWithContext`'s comment does not frame the type as correlation's; the prescribed edit was not owed | S1 corrected |
| `service_test.go`'s comment claims parity with one peer, not three | Row corrected |
| `config_parity_test.go`'s env corpus needed the nine new keys; `interceptors_test.go`'s and `harness_test.go`'s header enumerations go stale; `resolver_live_test.go` is a fifth carrier of the wrong qualifier; `client.go:17`'s scope is stale; `docs/grpc.md` anchor off by one | All in the rows or Cleanup |
| A paragraph claiming `TestDefaultGRPCServerConfigMatchesLoadedDefaults` carries `Options.ErrorDomain` onward — that test is unrelated to it | Deleted; S8 already states the split correctly |

### Fourth panel round

Verdicts: responsibility **FAIL**, package architecture **CONCERNS**, file
cohesion **FAIL**. **No system decision was challenged, and no finding concerned
a design choice.** Every item was either a claim about existing code or an
internal inconsistency — an S-section repaired without the rows it governs, or a
rename applied everywhere but one parameter.

Two would have broken implementation and are the round's justification:

- The `exhaustruct` include list matches **import paths**, so the entry must read
  `internal/infra/grpc\.Options`, not `grpcx.Options`. This would be the first
  entry in that list whose last path element differs from its package name, and
  the design had never recorded the pattern string. S8 now carries it.
- `resolver.Register` writes an unsynchronized registry map that `resolver.Get`
  reads on every `grpcclient.New`, and this package's parallel tests reach that
  read. A lane reproduced the race. The `load_balancing_test.go` row now requires
  registration from a sequential top-level test.

The rest, all applied: `StreamTimeout` is a second disabled-by-default field, so
S5 and two rows overstated what the mirror guard proves; row 3 still named the
chain parameter `budget` after the rename; `resolver_live_test.go` and
`correlation_service_test.go` were required to change by row 15 and Cleanup but
had no file-map rows; the `harness_test.go` row named two new capabilities
without naming their declarations; `interceptors_test.go` calls `mapError`
directly, a fourth compiler-forced edit; the `config_parity_test.go` row said
three edits where Cleanup requires four; the `internal/infra/http` edge was
justified by a build-cost claim that Go's whole-package linking falsifies; two
further docs carry the same outbound-gRPC seam enumeration; S6 kept an
off-by-one anchor its own row had already fixed; S1's naming appeal cited fields
this change adds; and R8's proof has a 10s wall-clock floor that was unrecorded.

One spec gap surfaced and was closed there: R8 had a falsifier in its rule block
but no success criterion. Criterion 6 now carries it.

**Closed upstream.** A negative `MaxConnectionAge` was refused by no owner:
grpc-go normalizes only zero, so a negative value arms the age timer immediately
and rotates every connection at once, while the rules branched on positive and
zero alone. R5 models two states and the type has three. Since no alternative
answer is defensible — a negative duration is refused — the spec's R5 and its
criterion 9 now require every keepalive duration to be non-negative, and S5's
rule list follows.
