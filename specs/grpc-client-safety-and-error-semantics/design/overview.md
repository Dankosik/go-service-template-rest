# Technical design — safe gRPC clients and transport-neutral failures

status: ready

Realizes: [../spec.md](../spec.md) R1-R3 at ready SHA-256
`1bb0965f876b052abd8e5be2142acfc8734141d5506e719d9529e6a425860cac`.
R1 and R3 remain unchanged; this design consumes repaired R2 without restoring
the rejected claim that health-enabled `pick_first` must fail construction.

## Authority and retained design

This design replaces only the superseded client and classification parts of
[`grpc-transport-hardening`](../../grpc-transport-hardening/design/overview.md):

- S4 keeps its status-detail attachment and sanitization boundary, but
  `internal/failure` replaces the HTTP-owned `problem.Mapped` as the value
  crossing it;
- S6 keeps one client-owned default service config and the existing
  round-robin/pick-first choice, and adds standard health configuration to that
  same document;
- S7's universal idle-keepalive default and cross-half parity obligation are
  removed. The server still permits idle pings for clients that explicitly
  enable them;
- S8's `ErrorDomain`, its bootstrap source, and its `exhaustruct` guard remain
  unchanged.

All other server deadlines, admission, keepalive and rotation, interceptor
order, standard-health publication, readiness/drain/shutdown, metadata
sanitization, telemetry, streaming, TLS, resolver/proxy refusal, and generated
contract decisions remain in force.

Current pinned authority is grpc-go v1.82.1. Its client keepalive option states
that keepalive is disabled when the option is absent and clamps a supplied
interval below ten seconds; its client health path requires the health package,
a `healthCheckConfig`, and a balancer that requests health. The standard health
guide fixes `Watch`, `SERVING` eligibility, `NOT_SERVING` removal, and
`UNIMPLEMENTED` fallback. Sources:

- [`WithKeepaliveParams`](https://github.com/grpc/grpc-go/blob/v1.82.1/dialoptions.go#L571-L580)
  and [client parameter adaptation](https://github.com/grpc/grpc-go/blob/v1.82.1/clientconn.go#L1305-L1317);
- [grpc-go client health activation](https://github.com/grpc/grpc-go/blob/v1.82.1/clientconn.go#L1598-L1668)
  and [health stream behavior](https://github.com/grpc/grpc-go/blob/v1.82.1/health/client.go#L51-L118);
- [standard gRPC health checking](https://grpc.io/docs/guides/health-checking/).

## Design drivers

| Driver | Consequence |
| --- | --- |
| R1 default silence | The default must omit grpc-go's keepalive option; passing zero is not equivalent because grpc-go clamps it to a live interval. |
| R1 explicit opt-in | The existing interval and timeout remain the complete opt-in pair; no third flag or nested configuration type is needed. |
| Repaired R2 | Health is enabled by the client-owned service config for round robin. Direct `pick_first` remains accepted and connectivity-owned in grpc-go v1.82.1. |
| Protected OIDC `Watch` | One optional dial-level credential must authenticate grpc-go's internal health stream and application RPCs without creating another health or auth mechanism. |
| Resolver trust boundary | Health and address selection stay in the same client-owned default service config while resolver-supplied service config remains disabled. |
| Correlation trust boundary | Connection credentials pass through the existing reserved-key sanitizer before grpc-go puts their metadata on any control or application stream. |
| R3 one service identity | A mapper must be usable by feature, HTTP, and gRPC code without importing `net/http`, an RFC Problem catalog, or grpc-go. |
| R3 distinct caller action | The classified article collision is `already_exists`; HTTP-only fallback `conflict` remains available but is not a domain failure code. |
| Compatibility | Existing non-conflict codes, retry hints, safe details, `ErrorInfo.Domain`, precedence, and unknown-error sanitization keep their current behavior. |
| Minimal release surface | No runtime config, schema, generated source, neighbor, persistence, or deployed template client exists to migrate. |

## System decisions

### D1 — Omit keepalive unless both existing fields opt in

**Selected.** `grpcclient.DefaultConfig` leaves `KeepalivePingInterval` and
`KeepalivePingTimeout` at zero. Construction accepts exactly `(0, 0)` or two
positive values. A negative or partial pair fails before `grpc.NewClient`.
`New` appends `grpc.WithKeepaliveParams` only for the positive pair and keeps
`PermitWithoutStream: true` fixed, because the pair explicitly names idle
keepalive.

Omitting the option preserves grpc-go's disabled state. Passing a zero-valued
`keepalive.ClientParameters` is rejected because `WithKeepaliveParams` raises
its interval to grpc-go's ten-second floor and would silently keep the unsafe
behavior alive.

grpc-go continues to own the effective minimum and the later-connection
interval increase after `too_many_pings`. This adapter retains the configured
values and adds no retry or reconnect policy.

**Rejected.** A new `EnableKeepalive` flag or nested keepalive struct. The
existing pair is already an unambiguous opt-in and construction must reject a
partial pair anyway.

**Reopen.** A named dependency supplies a measured or documented intermediary
idle timeout that cannot be met through its own explicit pair.

### D2 — One service config owns address selection and standard health

**Selected.** Add `HealthCheck bool` beside `Config.LoadBalancing`.
`DefaultConfig` sets it to true. Setting it false is the explicit per-dependency
escape hatch.

`LoadBalancingPolicy.serviceConfig` remains the single renderer and emits both
the selected `loadBalancingConfig` and, when enabled,
`"healthCheckConfig":{"serviceName":""}`. `New` keeps
`grpc.WithDisableServiceConfig` and supplies this JSON through the existing
`grpc.WithDefaultServiceConfig`; a resolver still cannot add retries, replace
the balancer, or weaken metadata policy.

`grpcclient` blank-imports `google.golang.org/grpc/health` because that package's
initializer installs grpc-go's client health implementation. No local goroutine,
poller, probe RPC, balancer wrapper, or health state is added.

`grpcclient.Options` gains one optional
`PerRPCCredentials credentials.PerRPCCredentials`. When non-nil, `New` wraps it
with the existing `wrapPerRPCCredentials` owner from `propagation.go` and appends
one native `grpc.WithPerRPCCredentials` dial option. grpc-go then applies the
same credential source to its internal health `Watch` and to ordinary RPCs.
The wrapper removes `traceparent`, `tracestate`, `baggage`, and `x-request-id`
while preserving authentication metadata, credential errors, and
`RequireTransportSecurity`; grpc-go remains the owner of transport-security
validation and credential failure statuses. No provider, token cache, refresh,
or authentication policy moves into `grpcclient`.

For round robin, grpc-go v1.82.1 enables the health listener on each pick-first
child. A direct `pick_first` does not receive that private listener attribute,
so `HealthCheck: true` remains callable and uses connectivity state. The client
does not reject this repaired combination and does not claim direct
`pick_first` consumes health.

**Failure and recovery.** Each participating round-robin backend owns one
grpc-go `Watch` stream for the empty service name. `SERVING` makes its
subchannel eligible; any other published state removes it from new picks while
in-flight RPCs continue. Stream failures retry under grpc-go's health backoff
until transport or channel cancellation. `UNIMPLEMENTED` marks that transport
ready and disables its health check, falling back to connectivity. These are
transport-control retries, not application retries. `ClientConn.Close` owns
their cancellation. Under the OIDC/JWT profile, a missing, rejected, or
transport-ineligible connection credential prevents the protected `Watch` from
making the backend eligible; the shared client does not bypass authentication
or disable health to recover.

**Rejected.** `grpc.WithDisableHealthCheck` as a second policy switch. Omitting
`healthCheckConfig` already disables the feature through grpc-go's activation
contract. A custom probe or polling loop would duplicate the standard stream
and the balancer's eligibility owner.

**Rejected.** Keeping `Watch` public or relying on call-scoped credentials.
The first exposes long-lived unauthenticated work outside process admission;
the second cannot reach grpc-go's internal control stream. A credential bundle
or auth-specific adapter is unnecessary because the installed grpc-go
`PerRPCCredentials` seam and existing sanitizer already own the complete path.

**Reopen.** A measured fleet shows material watch cost, a concrete dependency
requires a non-empty service name, or its authentication mechanism cannot be
expressed as grpc-go `PerRPCCredentials`. Preserve health-aware drain or
explicitly choose orchestrator-only routing for that dependency.

### D3 — `internal/failure` owns transport-neutral classification

**Selected.** Add the leaf package `internal/failure` with one
`failure.go`. It owns:

- `Code`, the existing non-conflict service failure constants, and
  `CodeAlreadyExists`; it deliberately declares no generic `CodeConflict`;
- `Classification { Code, Detail, RetryAfter }`;
- `Mapper func(error) (Classification, bool)` and the current nil-skipping,
  first-match `Classify` behavior.

The package imports only the standard `time` package. A focused depguard rule
forbids `net/http`, grpc-go, repository runtime packages, and composition roots,
so the boundary cannot drift back into a transport while still compiling.

`internal/problem` remains the HTTP Problem catalog. It derives every shared
code constant's string value from `internal/failure`, retains the literal
HTTP-only `CodeConflict`, and adds `CodeAlreadyExists`. The two 409 definitions
share the current status, title, and type URI. `conflict` stays first so the
existing status-only `For(409)` fallback remains `conflict`; classified paths
use `ForCode` and therefore preserve `already_exists`.

Its package comment changes with that ownership: `problem` is an HTTP-only
catalog, not a leaf for features. It directs feature classification to
`internal/failure` and removes the current feature-import and leaf claims.

This is the smallest new boundary that can be imported by feature-owned
classification and both transports. Keeping `problem.Mapper` as an alias would
leave a second stale classification surface and is rejected.

**Rejected.** Special-case `article.ErrAlreadyExists` in gRPC or inspect error
text/type there. That makes the transport infer business meaning and leaves the
generic `conflict` ambiguity intact.

### D4 — HTTP projects a failure through its existing Problem catalog

`internal/infra/http` receives `[]failure.Mapper`, preserves deadline precedence,
and converts a successful classification's code to the HTTP-owned
`problem.Code`. The existing catalog lookup, Retry-After rounding, Problem body,
and unknown-error fallback remain the only HTTP projection.

The reference classifier moves from `internal/httpapi` to
`examples/reference-service/internal/article/errors.go` beside the feature's
sentinel identities. It maps `ErrAlreadyExists` to
`failure.CodeAlreadyExists`; the reference composition passes that same mapper
to any transport. `internal/httpapi/problem.go` retains only generated HTTP
Problem construction.

No OpenAPI change is required. The reference contract already declares 409 and
its `Problem.code` is a string; status, title, type URI, and safe detail remain
unchanged.

### D5 — gRPC projects the same failure without the HTTP catalog

`internal/infra/grpc` changes `Options.DomainErrors`, `errorRendering`,
`mapError`, `mappedStatus`, and `classifiedDetails` to `internal/failure`.
The path order stays:

`cancellation or deadline -> trusted service status -> failure.Classify -> gRPC projection -> sanitized INTERNAL`.

`mappedStatus` retains every non-conflict code mapping, adds
`CodeAlreadyExists -> codes.AlreadyExists`, and removes the domain
`conflict -> codes.Aborted` arm. It owns its safe fallback status message rather
than consulting the HTTP catalog. `ErrorInfo.Reason` remains the upper-snake-case
failure code; `ErrorInfo.Domain`, `RetryInfo`, handler-status rejection, health
pass-through, and absence of details on unclassified errors remain unchanged.

The bootstrap still constructs one mapper slice and supplies it to HTTP and
gRPC. Only the slice type changes. The PostgreSQL saturation mapper and the
gRPC cardinality example migrate to `failure` without changing their code,
detail, retry, or status outcomes.

### D6 — No rollout artifact or generated-contract change

The affected graph is one compiled process and any outbound client compiled
from this package. The template declares no outbound neighbor or deployed
client instance. There is no state, migration, runtime key, managed dependency,
irreversible step, protobuf, or OpenAPI shape change, so no `rollout.md`, proto
regeneration, OpenAPI regeneration, or new CI job is warranted.

Old and new clients interoperate with the existing server health service. An
old client retains its previous active keepalive; a new default client omits it
and consumes the server's already-published empty-service health. Explicitly
enabled new keepalive remains accepted by the unchanged server permission.

A real rolling deployment could temporarily expose both
`conflict`/`ABORTED` and `already_exists`/`ALREADY_EXISTS`. The only current
producer is the non-production reference example, so this is accepted. A
shipped consumer reopens R3 and requires a compatibility alias or coordinated
adoption that preserves the distinct caller action.

## Material flows

### Client construction and idle connection

`dependency composition -> grpcclient.DefaultConfig -> validate complete keepalive pair -> render client-owned selection/health service config -> grpc.NewClient (no I/O) -> caller-owned ClientConn`.

- `(0, 0)` installs no keepalive option and no idle ping policy.
- A positive pair installs grpc-go keepalive with idle pings explicitly enabled.
- Health configuration does not start I/O during construction. Resolution,
  connection, and `Watch` begin only when grpc-go connects.
- The caller still owns `Close`; each operation still owns its deadline and any
  application retry.

### Health-aware round-robin selection

`dependency credential -> existing reserved-key sanitizer -> grpc-go transport -> resolver addresses -> client-owned round_robin config -> one grpc-go subchannel per backend -> empty-service Health.Watch -> server auth/admission/token bound -> balancer eligibility -> new RPC pick`.

- `SERVING` admits the backend; `NOT_SERVING` removes it from new picks;
  `SERVING` restores it.
- A selected in-flight RPC is not canceled by a later health transition.
- `UNIMPLEMENTED` falls back to connectivity for that peer.
- Resolver service config remains disabled throughout; health never changes
  correlation metadata, retries, or application deadlines. The optional
  connection credential is shared by control and application RPCs, and its
  reserved correlation values are removed before either reaches the wire.

### Classified service failure

`feature/dependency error -> failure.Mapper at the feature or composition owner -> one failure.Classification -> HTTP or gRPC adapter`.

- HTTP: classification code -> `problem.ForCode` -> status/title/type plus safe
  detail and optional rounded `Retry-After`.
- gRPC: classification code -> local status switch plus exact `RetryInfo` and
  upper-snake-case `ErrorInfo.Reason`.
- The slug collision is one `already_exists` classification and two transport
  projections. Generic HTTP fallback `conflict` never crosses this mapper seam.
- Cancellation and deadline expiry answer before classification; no
  classification leaves unknown handler text reachable.

## Responsibility map

| # | Responsibility | Current evidence | Selected owner and action | Dependency / proof owner |
| --- | --- | --- | --- | --- |
| 1 | Optional client keepalive state and validation | `grpcclient.Config`, `DefaultConfig`, `New`, `validateConfig` | `internal/infra/grpcclient/client.go` — change | grpc-go option omitted or appended; `client_test.go`, `keepalive_test.go` |
| 2 | Address-selection and health service config | `LoadBalancingPolicy.serviceConfig` | `internal/infra/grpcclient/load_balancing.go` — change | one client-owned JSON document; `load_balancing_test.go`, new `health_checking_test.go` |
| 3 | grpc-go health implementation activation | none in production client package | `internal/infra/grpcclient/client.go` — change with blank health import | grpc-go owns streams/backoff; health proof stays in grpcclient |
| 3a | Connection credential for internal and application RPCs | call-scoped credentials already use `wrapPerRPCCredentials`, but `Options` has no dial seam | `internal/infra/grpcclient/client.go` — add one optional credential field and native dial option; `propagation.go` keeps sanitizer code and generalizes its option-neutral rationale | grpc-go owns invocation/security semantics; `grpc_tls_contract_test.go` proves TLS/OIDC composition and reserved-key removal |
| 4 | Client package/operator contract | stale keepalive and selection prose | `internal/infra/grpcclient/doc.go`, `docs/grpc.md` — change | dependency owners select health disable, keepalive opt-in, and any connection credential |
| 5 | Neutral failure vocabulary and registry | `internal/problem/mapping.go` currently imports the HTTP catalog package | `internal/failure/failure.go` — add; `internal/problem/mapping.go` — remove | stdlib-only leaf; `internal/failure/failure_test.go`, depguard |
| 6 | HTTP Problem vocabulary, including two 409 identities | `internal/problem/problem.go` and its unique-status assumption | `internal/problem/problem.go` — change | derives shared strings from failure; `problem_test.go` pins fallback and code lookup |
| 7 | HTTP failure projection | `RouterConfig.DomainErrors`, generated response error path, `RejectResponse` | `internal/infra/http/router.go` — change | failure -> problem; `domain_errors_test.go` |
| 8 | gRPC failure projection and details | `grpc/status.go`, `grpc/config.go` | same files — change | failure -> grpc-go; `error_details_test.go`, `interceptors_test.go`, `docs_test.go` |
| 9 | Production mapper composition | `runtimeDependencies.DomainErrors`, `newGRPCRuntime`, `run.go` | bootstrap files — change type only | one concrete slice still feeds both adapters; startup dependency proof |
| 10 | Reference article identities and mapper | sentinels in `article.go`, classifier in HTTP adapter | `article/errors.go` — add; remove those declarations from old owners | feature -> failure; article/HTTP/composed gRPC proof |
| 11 | Reference HTTP projection | `httpapi/problem.go`, `handler.go`, `reference.go` | same owners — change | no gRPC import; existing OpenAPI remains canonical |
| 12 | gRPC reference aggregation failure | `examples/grpc-reference-service/service.go` | same owner — change type only | preserves `RESOURCE_EXHAUSTED` and detail |
| 13 | Profile-safe neutral mapper seam | DATABASE=none template and minimal init fixture still declare `problem.Mapper` | profile template and template-init fixture — change | failure remains when GRPC=none; gRPC-only composed proof is removed with GRPC |
| 14 | Stable architecture and author guidance | repository docs name `problem.Mapper`, universal idle pings, and no client health | repository architecture/structure/gRPC/first-feature docs — change | no template-sync manifest change |

## Go ownership and file map

The selected import direction is:

```text
feature and bootstrap -> internal/failure
internal/problem      -> internal/failure (shared code values only)
internal/infra/http   -> internal/failure, internal/problem
internal/infra/grpc   -> internal/failure, internal/reqctx
internal/infra/grpcclient -> grpc-go (client health, balancing, keepalive)
```

No feature imports a transport, no transport imports a feature, and bootstrap
remains the only owner that knows both adapters. No interface, factory,
registration framework, generated source, or new runtime configuration is
added.

### Production and durable support files

| Path | Action | One present reason to exist; declarations and boundaries |
| --- | --- | --- |
| `internal/failure/failure.go` | add | The complete transport-neutral failure vocabulary, classification value, mapper type, and first-match registry. Imports only `time`; must not own transport projection or feature sentinels. |
| `internal/problem/mapping.go` | remove | Its classification responsibility moves intact to `internal/failure`; retaining aliases would create a second semantic seam. |
| `internal/problem/problem.go` | change | HTTP Problem definitions and lookup. Shared code values derive from failure; HTTP-only `conflict` and both 409 definitions stay here. Rewrite the package comment to name this HTTP-only ownership and direct features to `internal/failure`. |
| `internal/infra/http/router.go` | change | HTTP routing and generated response-error projection. All mapper-bearing surfaces use `failure.Mapper`; deadline precedence and Problem rendering stay local. |
| `internal/infra/grpc/config.go` | change | Server collaborators. `Options.DomainErrors` becomes `[]failure.Mapper`; no other option or lifecycle changes. |
| `internal/infra/grpc/status.go` | change | The two gRPC error boundaries and their local status/detail projection. Imports failure instead of problem. |
| `internal/infra/grpc/doc.go` | change | Package contract names neutral classification while preserving interceptor, trust, health, and sanitization rules. |
| `internal/infra/grpcclient/client.go` | change | Client connection construction, complete-pair keepalive validation, health activation, `Config.HealthCheck`, and the optional sanitized dial-level `Options.PerRPCCredentials`. It must not own credential generation, Watch state, or application retry. |
| `internal/infra/grpcclient/load_balancing.go` | change | One service-config renderer for address selection and optional standard health. It must not add a balancer or resolver authority. |
| `internal/infra/grpcclient/doc.go` | change | Public package contract for default health, explicit disable, opt-in idle keepalive, and connection credentials using the same reserved-key boundary as call-scoped credentials. |
| `internal/infra/grpcclient/propagation.go` | change | Keep the existing sanitizer and reserved-key owner unchanged; generalize only its `ireturn` rationale because both gRPC call and dial options now consume the credential interface. |
| `cmd/service/internal/bootstrap/startup_dependencies.go` | change | Dependency-owned mapper declarations migrate to failure; PostgreSQL semantics stay unchanged. |
| `cmd/service/internal/bootstrap/startup_grpc.go` | change | Composition crossing accepts the neutral mapper slice; all server options and error domain stay unchanged. |
| `cmd/service/internal/bootstrap/run.go` | change | Composition comments and mapper type ownership only; startup and lifecycle control flow do not change. |
| `scripts/profiles/database-none/startup_dependencies.go.tmpl` | change | The no-database profile exposes the same neutral empty mapper slice as the PostgreSQL profile. |
| `examples/reference-service/internal/article/errors.go` | add | Article sentinel identities and their one transport-neutral classifier; declarations move from `article.go` and `httpapi/problem.go`. |
| `examples/reference-service/internal/article/article.go` | change | Remove sentinel declarations now owned by `errors.go`; use-case behavior and effect order stay unchanged. |
| `examples/reference-service/internal/httpapi/problem.go` | change | Generated HTTP Problem construction only; remove domain classification and select definitions by code where identity matters. |
| `examples/reference-service/internal/httpapi/handler.go` | change | References the feature-owned classifier in comments and uses explicit HTTP problem codes; operation behavior stays unchanged. |
| `examples/reference-service/reference.go` | change | Composition supplies `article.ClassifyError` to HTTP. No gRPC runtime is added to the example. |
| `examples/grpc-reference-service/service.go` | change | Its aggregation failure classifier migrates to failure without changing the generated service or status outcome. |
| `.golangci.yml` | change | A focused rule keeps `internal/failure` free of HTTP, gRPC, repository runtime, and composition imports. |
| `docs/grpc.md` | change | Canonical client/failure guidance: health-aware default, opt-in keepalive, neutral codes, and `already_exists`. |
| `docs/repo-architecture.md` | change | Stable component/dependency ownership gains the neutral leaf and corrected outbound client flow. |
| `docs/project-structure-and-module-organization.md` | change | Placement algorithm and package table admit `internal/failure`; grpcclient ownership includes health eligibility and optional keepalive. |
| `docs/first-production-feature.md` | change | A dependency owner explicitly disables standard health only when needed and supplies keepalive only for a named idle-timeout requirement. |
| `scripts/init-module.sh` | change | GRPC=none removes the renamed OIDC gRPC TLS contract proof and the gRPC-only reference proof while retaining failure and HTTP `already_exists`. |
| `scripts/ci/template-init-check.sh` | change | Minimal fixture declares failure, and profile assertions cover the conditional composed gRPC proof. |

### Proof files and cleanup

These placements name proof ownership only. Test Design still owns the exact
scenario matrix, observables, and commands.

| Path | Action | One present reason to exist; owned proof |
| --- | --- | --- |
| `internal/failure/failure_test.go` | add | Neutral vocabulary plus nil-skipping and first-match classification. |
| `internal/problem/problem_test.go` | change | HTTP catalog code uniqueness, both 409 lookups, and retained status-only `conflict` fallback; remove the obsolete global status-uniqueness invariant. |
| `internal/infra/http/domain_errors_test.go` | change | HTTP classification projection, retry header, unknown fallback, and retained HTTP-only conflict behavior. |
| `internal/infra/grpc/docs_test.go` | change | The published gRPC table and implementation cover the neutral failure constants, not HTTP Problem codes. |
| `internal/infra/grpc/error_details_test.go` | change | gRPC code/reason/retry projection and cross-transport parity migrate to failure; add exact `already_exists` projection. |
| `internal/infra/grpc/interceptors_test.go` | change | Classified handler behavior uses failure while cancellation, deadline, trusted status, and sanitization expectations stay unchanged. |
| `internal/infra/grpcclient/client_test.go` | change | Default health/zero keepalive and complete-pair construction validation; no-I/O construction remains. |
| `internal/infra/grpcclient/keepalive_test.go` | change | Raw HTTP/2 observation owns default silence and explicit opt-in ping, with health disabled to avoid a Watch stream. |
| `internal/infra/grpcclient/keepalive_parity_test.go` | remove | The superseded universal client/server default invariant has no remaining owner. Server acceptance stays in server tests. |
| `internal/infra/grpcclient/load_balancing_test.go` | change | Round-robin/pick-first address selection, explicitly health-disabled callability, and one successful RPC with `DefaultConfig`, `HealthCheck: true`, and direct `pick_first`; no health transition logic is mixed into this file. |
| `internal/infra/grpcclient/health_checking_test.go` | add | Standard per-backend health eligibility and `UNIMPLEMENTED` fallback for the shared client. Test-local backends and state controls stay in this file unless another current proof needs them. |
| `internal/infra/grpcclient/propagation_test.go` | change | Its clients explicitly disable health so grpc-go's control `Watch` cannot occupy the harness stream observation before the test's own `Health.Watch`; propagation observables and coverage remain unchanged. |
| `internal/infra/grpcclient/transparent_retry_test.go` | change | Its raw HTTP/2 peer explicitly disables health so the two captured streams remain the application attempt and transparent retry, not health-control traffic. |
| `internal/infra/oidcjwt/grpc_test.go` | change | Update only its pointer to the renamed real TLS boundary contract; interceptor-level proof remains unchanged. |
| `internal/infra/oidcjwt/grpc_tls_test.go` | remove | Its existing executable TLS/OIDC boundary proof moves intact to the contract filename required by repository structure. |
| `internal/infra/oidcjwt/grpc_tls_contract_test.go` | add | The renamed real TLS/OIDC contract additionally proves that the automatic round-robin `Watch` and an application RPC receive one dial credential while credential-supplied reserved correlation metadata reaches neither handler. No production OIDC owner changes. |
| `cmd/service/internal/bootstrap/startup_dependencies_test.go` | change | PostgreSQL saturation still produces the same neutral retryable classification. |
| `examples/reference-service/internal/article/errors_test.go` | add | The article sentinels map to the closed failure identities, including `already_exists`. |
| `examples/reference-service/internal/httpapi/router_test.go` | change | The real reference HTTP route publishes 409/`already_exists`; its test-only response mirror consumes failure and selects Problem by code. |
| `examples/reference-service/reference_test.go` | change | Reference composition supplies the feature mapper; existing HTTP runtime proof remains otherwise unchanged. |
| `examples/reference-service/grpc_failure_mapping_contract_test.go` | add | With the gRPC profile retained, the actual article mapper crosses a real `grpcx.NewServer` boundary and a caller observes `ALREADY_EXISTS` plus `ErrorInfo`. It is removed when GRPC=none. |

The existing server lifecycle, health publication, keepalive acceptance,
reference protobuf, OpenAPI, bootstrap lifecycle, telemetry, resolver,
streaming, and race files are regression surfaces only. They are not rewritten
to restate this policy. The two client tests named above opt out only to keep
their current stream observables isolated; they do not become health-policy
proofs.

## Review disposition

The required Go Ownership panel reached three compatible `PASS` receipts.

- Responsibility and execution ownership first returned `CONCERNS` because the
  repaired health-enabled direct `pick_first` path lacked a proof owner. The
  file map now assigns one successful RPC to `load_balancing_test.go`; a fresh
  focused review returned `PASS`.
- Package and dependency architecture returned `PASS`. Later repairs did not
  change a package, import direction, visibility, generated/manual boundary, or
  composition edge, so that receipt remains current.
- File cohesion and naming first returned `CONCERNS` for the stale `problem`
  package comment and a boundary-test filename. The design now requires the
  HTTP-only comment and uses `grpc_failure_mapping_contract_test.go`; a fresh
  focused review returned `PASS`.

The broader Technical Design review then returned one bounded `CONCERNS`: the
new automatic health stream would consume the stream observables in the
propagation and transparent-retry proofs. The proof map now assigns explicit
test-local health disablement to every constructor in those two files and keeps
the shared harness unchanged. Fresh responsibility and file-cohesion reviews
both returned `PASS`; a fresh focused broader review also returned `PASS`.

The earlier reviewed semantic candidate was repository HEAD
`da89db83a78ca4a19fefe66d4105f69fb73b7ff0`, ready-spec SHA-256
`2f08a2f92e6c4254172718d22cf382d1a5fa7ac70d772d25881e1b0f3df8bc8f`, and
design SHA-256
`0453732a2826d4468f0100c17dca287220bf3311fd482be635dc1920dfbb4a9a`. This
historical receipt remains valid outside repaired R2.

Implementation later reopened only R2 after the OIDC/JWT owner protected
`Health/Watch`. The repaired design reuses grpc-go's dial-level
`PerRPCCredentials` and the existing sanitizer, and adds one composed TLS/OIDC
proof in the existing boundary test. R1, R3, packages, generated/manual
authority, runtime configuration, and release scope remain unchanged. Fresh Go
Ownership review of this repair returned compatible responsibility and package
`PASS` receipts. File cohesion first returned `CONCERNS`: the executable boundary
needed the repository's `_contract_test.go` name, and the sanitizer's `ireturn`
comment still named only a call option. The file map now requires the rename,
its two mechanical references, and the option-neutral comment; focused file
re-review returned `PASS`. All three current Go Ownership lanes therefore pass
on this repaired candidate. Broader Technical Design review at SHA-256
`5f2db172ca738d5d0426c52bf912851f2ac7f96fc4f3d95da1f7a62b9a300a41`
also returned `PASS`: the grpc-go credential/health path, OIDC fail-closed
boundary, failure/recovery lifecycle, composed proving surface, and unchanged
release/config scope have no unsupported edge. This final edit changes only
status and review disposition.
