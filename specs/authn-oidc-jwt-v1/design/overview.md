# OIDC JWT Authentication Technical Design

status: ready

## Drivers and selected architecture

The target must satisfy the ready specification with one security authority,
one identity result, bounded network/concurrency work, no runtime bypass, and
complete initializer removal. The decisive drivers are:

| Driver | Consequence |
| --- | --- |
| Strict RFC 9068 access tokens, not ID tokens | The repository owns access-token header/claim policy and uses a JOSE primitive only for JWS/JWK cryptography. |
| One immutable issuer/audience and no caller-selected authority | Discovery runs once at startup; only its exact issuer and JWKS URI can establish trust. |
| Prompt rotation plus refresh-abuse resistance | One atomic cache and one coalesced refresh coordinator serve both transports. |
| Bounded outage trust | Last-known-good keys are usable for at most 15 minutes, then authn and readiness fail closed. |
| Existing HTTP/OpenAPI, gRPC policy, `reqctx.Principal`, health, telemetry, and egress owners | Concrete adapters compose into those seams; no generic authentication framework is added. |
| `AUTHN=none` purity | Every new source/config/contract/doc/test/dependency block is a removable profile owner; shared edits are marker-scoped. |
| No endpoint/token disclosure | Authn-specific egress instrumentation is suppressed; only bounded capability metrics and sanitized lifecycle events remain. |
| No unsupported performance claim | The normal path is local and bounded; proof establishes work-amplification limits, not a universal latency number. |

The selected target is one process-local concrete
`internal/infra/oidcjwt.Verifier`. Bootstrap establishes initial trust before
serving, then both HTTP and optional gRPC adapters call that same verifier.
The verifier owns immutable key snapshots, refresh coordination, authn metrics,
and finite errors. The repository's existing transport, health, egress, and
composition packages retain their current responsibilities.

No additional deployable, durable cache, introspection service, token store, or
runtime feature switch is introduced.

## Mechanism selection

### JOSE dependency

Select `github.com/go-jose/go-jose/v4` **v4.1.4** as the only new direct
dependency. It requires Go 1.24 (the repository uses Go 1.26), has no module
dependencies, accepts an exact JWS algorithm allowlist, exposes protected
headers, supplies RSA/JWK primitives, and includes the current published
security fix recorded in Research.

The capability uses:

- `jose.ParseSignedCompact` with only `jose.RS256`;
- `JSONWebKey`/RSA public-key conversion;
- `github.com/go-jose/go-jose/v4/json` for duplicate-rejecting JSON decode.

It does not use `go-jose/jwt` claim validation or any remote-key cache.
Repository code owns canonical compact/base64 checks, required claims and exact
NumericDate semantics, Discovery, key admission, refresh, and lifecycle.

| Substitute | Rejection |
| --- | --- |
| `golang-jwt/jwt/v5` | Strong claims options, but generic merged headers and no JWK primitive; strict duplicate JSON and the full cache remain local, so it does not reduce the custom boundary. |
| `lestrrat-go/jwx/v3` | Broad transitive surface and cache-owned goroutines/HTTP policy conflict with the exact repository lifecycle and key-miss policy. |
| Auth0 middleware | Carries transport middleware and older jwx, lacks strict RFC 9068 type policy and exact miss refresh, and duplicates repository HTTP/gRPC seams. |
| `go-oidc` | Its high-level verifier owns ID-token semantics and its remote key set has unbounded known-key staleness. |
| Custom RSA/JWS implementation | Standard-library signature calls alone do not justify reimplementing JOSE parsing and JWK conversion. |

Reopen dependency selection only for a new advisory, toolchain incompatibility,
or a maintained primitive that supplies the exact strict boundary with less
code without taking over repository egress, cache, or lifecycle policy.

### Outbound egress

Reuse `internal/infra/httpclient.ExternalHTTPS`; do not duplicate its public-IP,
post-DNS, no-proxy, no-redirect, authority, connection, header, body, timeout,
and idle-pool policy.

`AUTHN=oidc-jwt` therefore retains `internal/infra/httpclient` as an internal
prerequisite even when `OUTBOUND_HTTP=none`, as fixed by AUTHN-02A. That profile
still contains no service-facing general provider wiring or documentation.
Initializer retention is the union:

```text
keep internal/infra/httpclient =
    AUTHN == oidc-jwt OR OUTBOUND_HTTP == bounded
```

Add one authn-profile-marked `DisableInstrumentation` field to
`httpclient.Config`. When true, `otelhttp.WithFilter` rejects the request and
passes it directly to the already bounded base transport, creating neither
client span nor client metric and injecting no trace header. Existing outbound
profiles retain current instrumentation after `AUTHN=none` strips this field
and branch. OIDC supplies its own bounded signals without endpoint attributes.

## Immutable configuration

The OIDC profile adds this config-only shape:

```go
type Config struct {
    // existing fields...
    Authn AuthnConfig `koanf:"authn"`
}

type AuthnConfig struct {
    Issuer              string `koanf:"issuer"`
    Audience            string `koanf:"audience"`
    TrustedProxyCIDRs   string `koanf:"trusted_proxy_cidrs"`
}
```

`TrustedProxyCIDRs` is a canonical comma-separated string rather than a slice so
the repository's immutable snapshot remains comparable. Validation trims and
requires unique canonical `netip.Prefix` values; the HTTP adapter parses them
once at construction.

There is deliberately no `enabled`, algorithm, bypass, test-key, insecure
transport, or mutable refresh field. The fixed constants from AUTHN-03 live in
`oidcjwt`, and config validation enforces:

- absolute HTTPS issuer; no opaque URL, user info, query, fragment, or blank
  hostname; normalized scheme/host must equal the configured issuer form;
- non-empty audience;
- one or more valid unique trusted proxy CIDRs;
- when gRPC is compiled and enabled, `grpc.server.transport_security == "tls"`
  and the existing certificate/key validation succeeds.

Defaults for the three required values are empty. The source template and an
OIDC-initialized service fail config validation until explicitly configured;
there is no source-template execution escape. `AUTHN=none` removes the fields,
defaults, schema, validation, examples, and tests, so `APP__AUTHN__*` becomes an
unknown key.

Environment examples use:

```text
APP__AUTHN__ISSUER=https://issuer.example.com
APP__AUTHN__AUDIENCE=https://api.example.com
APP__AUTHN__TRUSTED_PROXY_CIDRS=10.0.0.0/8,2001:db8:1234::/48
```

These values are policy, not secrets. Tokens and private signing keys never
enter configuration.

## Trust bootstrap and key authority

`oidcjwt.New(ctx, config, meterProvider)` is synchronous:

1. Revalidate the immutable policy and construct an `ExternalHTTPS` discovery
   client pinned to the configured issuer authority. It has a 5-second request
   and response-header timeout, 32 KiB response-header bound, 1 MiB body bound,
   one connection/idle connection, no retry, and disabled instrumentation.
2. Derive the OIDC Discovery URL by appending
   `/.well-known/openid-configuration` to the issuer path. Fetch exactly once,
   require HTTP 200, strict JSON/EOF/no duplicate members, exact returned
   issuer equality, and one absolute HTTPS `jwks_uri`.
3. Close the discovery idle pool. Build a second independently pinned
   `ExternalHTTPS` client for the discovered JWKS authority. Cross-origin JWKS
   is allowed by OIDC but gets its own public-address and authority check.
4. Fetch the exact discovered URI under a fresh 5-second timeout, validate the
   complete candidate, and atomically install it. Any cancellation, status,
   body, JSON, key, or trust error closes both clients and aborts startup before
   listener binding.

Discovery is not refreshed in-process. Changing `jwks_uri` requires restart,
which keeps the network authority immutable and prevents request traffic from
amplifying Discovery. The configured issuer is authoritative; Discovery and
JWKS are semi-trusted evidence admitted only after these checks.

### Strict token representation

Before cryptography or refresh, verification:

1. requires at most 8 KiB and exactly three non-empty compact segments;
2. decodes every segment with `base64.RawURLEncoding.Strict`, rejects `=`, and
   requires byte-for-byte equality with its raw-url re-encoding;
3. decodes protected header and claims using go-jose's duplicate-rejecting JSON
   package and requires EOF;
4. rejects `crit`, `b64`, `jku`, `x5u`, embedded `jwk`, unprotected security
   parameters, non-`RS256`, non-access-token `typ`, and empty/non-string `kid`;
5. decodes local exact claim types. A custom NumericDate unmarshaller accepts
   only an integral base-10 `int64`; audience accepts only one non-empty string
   or a unique non-empty string array;
6. rejects every locally provable required-claim, issuer, audience, or time
   failure before a key-miss refresh.

Only after the selected RSA public key verifies the exact compact signing input
does the code publish `reqctx.Principal{Subject: sub}`. Unknown claims and raw
claim objects are discarded.

### JWKS admission and authority

Each candidate is built off to the side and installed with one
`atomic.Pointer[keySet]` store. The immutable snapshot contains `fetchedAt`,
generation, and a `map[kid]keySlot`; a slot retains the compatible count and
only exposes a key when exactly one compatible key exists.

Admission requires:

- strict JSON, EOF, no duplicate object members, and at most 100 total entries;
- every JWK is structurally valid; unrelated well-formed key algorithms/uses
  are ignored;
- a usable key is public RSA, modulus at least 2048 bits, canonical raw-url
  `n`/`e`, valid exponent, non-empty `kid`, `alg` absent or `RS256`, `use`
  absent or `sig`, and `key_ops` absent or permitting verification without a
  contradictory operation;
- at least one usable RS256 key.

Private/symmetric material is never retained. Any duplicate compatible `kid`
makes the complete candidate ambiguous and rejects it, even if other keys would
be usable. A candidate with any malformed JWK, excessive entries, ambiguity, or
no usable key is rejected as a whole. Rejection leaves the previous generation
and timestamp unchanged.

## Refresh, concurrency, and failure flow

`Verifier.Run(ctx, onTrustCurrent)` is registered once in the existing
background supervisor before serving. `onTrustCurrent` is bootstrap's optional
gRPC readiness sink; HTTP readiness calls the same local currentness check
directly. `Run` owns the timer, refresh goroutines, cancellation, and join.

The verifier has an explicit mutex-protected lifecycle:

```text
new -> running -> joined -> closed
  \--------------------------^
```

`New` completes initial trust but starts no goroutine. `Run` alone changes
`new` to `running`, installs its owned child context and `done` channel, and
returns only after every timer and refresh goroutine has joined; its deferred
transition sets `joined` and closes `done`. A second or post-close `Run` returns
a fixed lifecycle error. `Close` is idempotent: from `new` it changes directly
to `closed` and closes idle pools immediately; from `running` it cancels the
owned child context, waits for `done`, then closes idle pools; from `joined` it
closes idle pools and changes to `closed`. The state mutex is never held while
waiting. Thus deferred cleanup after `New` is safe even when later bootstrap
construction fails before supervisor registration, and the normal shutdown
path still joins `Run` before closing its client.

State ownership:

- `atomic.Pointer[keySet]` is the read-only request snapshot;
- one mutex protects the current `refreshCall`, global miss cooldown, and
  generation facts;
- every `refreshCall` has one close-only `done` channel and a fixed sanitized
  outcome;
- no token, subject, claim, or attacker `kid` is stored in shared state.

Scheduled refresh is due five minutes after each successful install and ignores
the key-miss cooldown. All triggers join an already active `refreshCall`. A
failed scheduled refresh arms exactly one recovery attempt for 30 seconds
later; each further failure repeats that fixed 30-second cadence until success
or cancellation. A success installs the snapshot and resets the next scheduled
attempt to five minutes after that install. These are lifecycle-owned timer
attempts, not HTTP-client retries: each attempt performs exactly one request,
and there is no nested retry or backoff.

For an unknown `kid` or compatible same-`kid` signature miss outside cooldown:

1. the first caller starts one lifecycle-owned JWKS fetch and starts the global
   30-second cooldown; concurrent callers join the same call;
2. each waiter selects its own request cancellation or `call.done`; waiter
   cancellation does not cancel the shared 5-second fetch;
3. success loads the new snapshot and retries key selection/signature exactly
   once;
4. failure retains the old snapshot. If it is still under 15 minutes, it
   remains authoritative and the missing/mismatching credential is invalid; if
   it is stale, the result is unavailable;
5. misses inside cooldown perform no network work and use the same currentness
   rule. A scheduled success may recover or advance the generation.

At `now >= fetchedAt + 15m`, the set is not current. No token can authenticate
until a successful atomic refresh; requests return unavailable and readiness
is false. Token expiry is checked independently on every call.

The maximum attacker-driven network amplification is one JWKS request per
30 seconds per process plus the scheduled request. The normal path performs no
network I/O, one bounded parse, O(1) key lookup, and at most one RSA
verification. These are complexity constraints, not latency/throughput claims;
focused counters and concurrency tests are sufficient until a real workload
establishes a performance budget.

Expected upstream failures do not terminate `Run`; they update fixed signals
and continue the fixed 30-second scheduled recovery cadence. An unexpected
panic inside verification or refresh is recovered at the capability boundary
and converted to sanitized unavailable/failure without formatting the panic
value. Supervisor cancellation stops timers, cancels in-flight fetches, joins
them, and returns within its existing five-second shutdown budget.

Currentness has its own timer reset by each successful install. It fires at the
exact 15-minute boundary independently of scheduled/failing fetches and invokes
the transition callback once; a later successful install invokes recovery once.
The callback never performs network work.

## HTTP contract realization

Canonical `api/openapi/service.yaml` gains a marker-owned
`http`/`bearer`/`JWT` security scheme and makes it the sole top-level
requirement. Both health operations explicitly set `security: []`; no anonymous
OR alternative exists. Reusable 400/401/431/503 problem responses document the
auth boundary. `internal/openapi/openapi.gen.go` remains generated authority.

`oidcjwt.HTTPAuthenticator()` returns the existing
`openapi3filter.AuthenticationFunc`. For the configured scheme it:

1. verifies the immediate `RemoteAddr` is in the parsed trusted-proxy set and
   exactly one `X-Forwarded-Proto` value is `https`;
2. reads exactly one Authorization value, deletes all Authorization fields from
   the request before any further work, and parses the exact Bearer grammar;
3. applies the 8 KiB limit, verifies, and writes the principal through
   `reqctx.SetPrincipal`.

The authn profile adds a marker-scoped call in `httpx.RejectRequest` to one
concrete `writeOIDCJWTAuthenticationFailure` helper. It unwraps only
`oidcjwt.Kind` and maps the fixed 400/401/431/503 problems, challenges, and
`Retry-After`; it never formats the cause. Add the marker-owned generic catalog
entry `request_header_fields_too_large` for 431. When `AUTHN=none`, the helper,
call/import, and catalog entry are removed; the repository's pre-existing
fail-closed `Authenticate` seam remains unchanged.

The adapter records exactly one verification outcome and never invokes a
handler on failure. A synthetic secured OpenAPI operation in tests proves
end-to-end principal arrival because the shipped production contract contains
only public health operations.

## gRPC parity and lifecycle

When compiled, `oidcjwt.UnaryInterceptor` and `StreamInterceptor` are appended
through existing `grpcRuntimeBindings` policy slices. Existing admission stays
outside authentication to bound expensive work; authentication remains before
domain mapping and every application handler.

Only `/grpc.health.v1.Health/` bypasses verification. The adapter removes
`authorization` from copied incoming metadata even for health. Other methods:

- require exactly one metadata value and the same Bearer grammar/limit;
- verify once before handler entry;
- attach the principal using `reqctx.ContextWithPrincipal`;
- wrap a stream exactly once with the immutable enriched context;
- map missing/malformed/invalid to `Unauthenticated`, oversize to
  `ResourceExhausted`, and stale trust to `Unavailable`, with fixed text.

Config rejects enabled plaintext gRPC before certificate loading. No reflection
or other standard-service bypass is added.

Standard gRPC health must track runtime trust, not only startup/drain. Add
authn-marker-scoped admitted/trust-current state to `grpcx.Server` and a
`SetAuthnReady(bool)` method. Effective health is:

```text
SERVING = startup admitted AND authn trust current AND not draining
```

`MarkServing`, `SetAuthnReady`, and `StartDrain` update this state under the
existing health mutex; drain remains terminal. The refresh callback flips trust
at the exact stale/recovery boundary. With `AUTHN=none`, the state/method is
removed and current gRPC lifecycle is byte-for-byte retained.

Immediately after constructing gRPC, bootstrap calls
`SetAuthnReady(verifier.CheckReady() == nil)` before startup admission can call
`MarkServing`. Initial successful trust therefore reaches `SERVING` on
admission without waiting for a refresh transition; subsequent callback values
own only stale/recovery changes.

HTTP `/health/ready` composes `Verifier.CheckReady` directly with startup
admission so generic readiness hysteresis cannot extend stale trust. Liveness
never depends on issuer availability.

## Error and observability contract

`oidcjwt.Error` exposes only a finite `Kind`:

```text
missing | malformed | oversize | invalid | unavailable | untrusted_transport
```

`Error()` is fixed text; it has no public `Unwrap`. Raw parser, signature,
network, status, JSON, key, endpoint, or panic detail is discarded at the
classification boundary. Startup errors name only a fixed stage and class.

The capability owns OTel scope `service.authn` and exactly:

- `authn.verifications` `Int64Counter`, unit `{verification}`, attributes
  `authn.transport=http|grpc` and `authn.result=success|failure`; failures also
  carry exactly one fixed
  `authn.reason=missing|malformed|oversize|invalid|unavailable`, while success
  omits the reason attribute. The internal `untrusted_transport` kind records
  as `malformed`, matching its public invalid-request class;
- `authn.jwks.refreshes` `Int64Counter`, unit `{refresh}`, attributes
  `authn.refresh.trigger=startup|scheduled|key_miss` and
  `authn.result=success|failure`;
- `authn.jwks.age` `Float64ObservableGauge`, unit `s`, no attributes, omitted
  until initial trust exists.

Instrument creation failure installs no-op instruments and emits one sanitized
fixed-name warning; it cannot change authentication. The callback reads only
the atomic install timestamp. No metric attribute comes from a token, claim,
subject, `kid`, issuer, audience, endpoint, error string, or response.

Logs are lifecycle-only: refresh and trust-current transitions with fixed
trigger/result and optional numeric age. There is no per-rejection auth log.
Existing request/RPC traces and access logs supply correlation. OIDC egress
instrumentation is disabled, no custom verification or refresh span is added,
and no Authorization/claims/baggage are propagated.

Leakage tests seed one poison marker into token segments, subject, `kid`, mock
body, parser/network errors, and injected panic, then inspect returned
problems/statuses plus captured logs, spans, and metrics.

## Composition, shutdown, and rollout

Bootstrap order:

1. load and validate config; initialize logging/telemetry and current runtime
   dependencies;
2. synchronously build OIDC initial trust and register its deferred idempotent
   cleanup;
3. create the supervisor and health service, with direct OIDC readiness;
4. build the HTTP router from `HTTPAuthenticator`;
5. when enabled, build native-TLS gRPC with both auth interceptors and register
   `SetAuthnReady` as the trust callback, then publish the verifier's current
   initial readiness once before admission;
6. register `Verifier.Run` with the supervisor before listener bind/admission;
7. serve and drain normally;
8. after HTTP/gRPC drain, cancel/join supervisor, close OIDC idle pools, close
   other dependencies, then flush telemetry.

Partial startup closes any client already acquired. `Close` and all readiness
transitions are idempotent. No key cache survives restart.

An OIDC profile rollout intentionally fails startup for invalid policy or
unavailable initial trust. Rollout proof must canary initial trust plus HTTP
readiness and gRPC health when enabled. Reverting first-time authentication to
an older anonymous build is not a safe rollback; rollback must target a
previous authenticated build or keep an external ingress auth boundary. This
task performs no deployment and claims no target-environment proof.

## Go code and ownership

| Responsibility | Owner and exact placement | Dependency / cleanup / proof |
| --- | --- | --- |
| Immutable authn config | `internal/config/types.go`, `defaults.go`, `validate.go`; new `authn_config_test.go`; `env/config/base.yaml`, `env/.env.example` | Marker-owned `AuthnConfig`; nested authn+gRPC validation; all removed for none. |
| JOSE/access-token verification | new `internal/infra/oidcjwt/verifier.go`, `token.go`, `strictjson.go`, `errors.go` | Imports go-jose and `reqctx`; no exported interface; focused real-signature, fuzz/no-panic, redaction tests. |
| Discovery/JWKS admission | new `internal/infra/oidcjwt/provider.go`, `keyset.go` | Imports concrete `httpclient`; unexported test-only client/clock seams in the consumer package; deterministic fake transport tests. |
| Refresh/currentness | new `internal/infra/oidcjwt/refresh.go` | Atomic snapshot plus one mutex/call; verifier implements local readiness; race/liveness/cancellation/shutdown tests. |
| Capability telemetry | new `internal/infra/oidcjwt/metrics.go` | Uses injected OTel meter provider; no generic telemetry interface; in-memory metric/log/span leakage proof. |
| HTTP adapter | new `internal/infra/oidcjwt/http.go`; marker call in `internal/infra/http/router.go`; new `internal/infra/http/authentication_oidcjwt.go` only for problem writing if needed | Existing OpenAPI authentication and `reqctx` path; adapter/contract integration tests. Removed for none. |
| gRPC adapter | new `internal/infra/oidcjwt/grpc.go`; marker state/methods in `internal/infra/grpc/server.go`; bootstrap policy wiring | File and shared blocks removed for `GRPC=none` or `AUTHN=none`; bufconn/TLS unary+stream/health proof. |
| HTTP contract | canonical marker blocks in `api/openapi/service.yaml`; generated `internal/openapi/openapi.gen.go` | Edit canonical first, regenerate, prove drift/default security/health overrides; remove blocks and regenerate for none. |
| 431 problem | marker entry in `internal/problem/problem.go` and tests | Capability-scoped shared edit; removed for none. |
| Egress suppression | marker field/branch/tests in `internal/infra/httpclient/client.go` | Existing profiles retain default telemetry; authn requests have no endpoint-bearing telemetry. |
| Composition/lifecycle | new `cmd/service/internal/bootstrap/startup_authn.go` and tests; marker calls in `run.go` and `startup_grpc.go` | Bootstrap knows concrete adapters; no feature/domain import inversion; partial-start and ordered-close proof. |
| Initializer/profile | `scripts/init-module.sh`, `scripts/ci/template-init-check.sh`, `template.lock` writer | Validate before mutation; union-retain httpclient; strip nested authn/gRPC blocks; regenerate then tidy. |
| Documentation | new `docs/authentication.md`; capability blocks in `README.md`, initializer-derived README text, architecture/config/build docs | OIDC guide and env examples retained only for OIDC; physically absent for none. |
| Dependency authority | `go.mod`, `go.sum` | Add exact go-jose v4.1.4; `go mod tidy` removes it for none; vulnerability/module graph proof before closeout. |

The import direction remains acyclic:

```text
bootstrap
  -> config
  -> infra/oidcjwt -> infra/httpclient, reqctx, go-jose, OTel, grpc/OpenAPI types
  -> infra/http    -> infra/oidcjwt (profile-only failure mapping)
  -> infra/grpc    (profile-only readiness state)
```

`oidcjwt` never imports bootstrap, `httpx`, `grpcx`, a feature package, or
generated handlers. Production constructors use concrete dependencies.
Unexported interfaces/functions exist only where `oidcjwt` tests substitute
HTTP, clock, and timer behavior; no consumer-facing authentication framework is
created.

## Profile and generation closure

Initializer validates presence-sensitively: unset `AUTHN` defaults to `none`,
explicit empty and unknown values fail before mutation. It writes
`authn = "none|oidc-jwt"` to `template.lock` and includes it in repeat-choice
validation/output.

- `AUTHN=none`: remove `internal/infra/oidcjwt`, authn bootstrap/config/HTTP/
  gRPC files and blocks, OpenAPI scheme/default/responses, docs/examples/tests;
  retain `httpclient` only if `OUTBOUND_HTTP=bounded`; regenerate and tidy away
  go-jose.
- `AUTHN=oidc-jwt,GRPC=none`: retain verifier/HTTP/config/docs and
  `httpclient`; remove `grpc.go`, gRPC state/wiring, all protobuf/gRPC surfaces.
- `AUTHN=oidc-jwt,GRPC=enabled`: retain one verifier plus both adapters; runtime
  config requires native gRPC TLS.

`scripts/profiles` remains generator-only and is removed from initialized
services. OpenAPI is regenerated only after all marker decisions, followed by
`go mod tidy` and tools tidy. Profile proof checks paths/symbols/config keys,
dependency presence, compilation, canonical drift, invalid-before-mutation,
and repeated whole-tree byte stability.

## Proof and reopen boundaries

Test Design must provide the complete deterministic matrix from the spec plus:

- current go-jose advisory and exact module-graph scan;
- strict JSON/base64/JWK fuzz/no-panic evidence;
- refresh generation/cooldown network-attempt oracles, race, goroutine/liveness,
  blocked-fetch shutdown, and exact 15-minute boundary;
- HTTP synthetic protected-operation identity/non-invocation and OpenAPI
  inherited-security proof;
- real TLS gRPC unary/stream/health/readiness parity;
- poison-marker inspection across all telemetry/error/panic carriers;
- profile matrices for both outbound choices and both gRPC choices.

Reopen System Design for multi-issuer trust, introspection/revocation,
sender-constrained tokens, an egress proxy that defeats `ExternalHTTPS`, native
HTTP TLS instead of a trusted proxy, or a target deployment unable to provide
the declared TLS boundaries. Reopen Specification for different algorithms,
limits, freshness/cooldown, public methods, identity shape, error statuses, or
safe anonymous rollback. Reopen Go Ownership only if implementation inspection
proves an exact file split cannot preserve the acyclic graph above.
