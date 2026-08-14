# Technical design — outbound OAuth client credentials

status: ready

Realizes [../spec.md](../spec.md) R1-R10 at ready SHA-256
`8ed6157fdc283a0939535854150306fdc19fd51d75f52da7fbefeac3345f33fb`.
Provider registration, privilege selection, secret issuance, network
provisioning, dependency criticality, fleet capacity, and live conformance stay
at the external checkpoints fixed by the Specification.

## Selected system

Keep one service process and the current configuration, bounded HTTP, bounded
gRPC, bootstrap, health, telemetry, and template-profile owners. Add one
process-local credential owner for one configured dependency and no other
deployable, store, queue, sidecar, persistent cache, background refresh loop,
readiness probe, or provider registry.

```text
generated or concrete dependency client
        |                         |
        | HTTP operation          | application gRPC operation or stream
        v                         v
authenticated HTTP Doer     authenticated grpc.ClientConnInterface
        |                         |
        +----- one operation-fixed token -----+
                                  |
                         connection PerRPCCredentials
                                  |
                   application attempts and Health/Watch
                                  |
                                  v
                    one process credential owner
                                  |
                  separate bounded HTTPS client, no retry
                                  |
                                  v
                       exact pinned token endpoint
```

The credential owner is the only authority for token acquisition, validation,
reuse, acquisition-wave coordination, failure suppression, retirement, and
sanitized auth telemetry. HTTP and gRPC adapters fix one returned token to one
logical resource operation. Feature code receives only the authenticated Doer
or `grpc.ClientConnInterface`.

The token endpoint and resource endpoint are separate fixed authorities and
separate client/connection pools even when their hostnames match. The token is
opaque. No component parses claims, discovers metadata, probes an auth style,
refreshes proactively, retries a token request, or converts a resource-server
authorization decision into an auth-provider decision.

## Implementation-source decision

The selected source is the Go standard library plus the existing
`internal/infra/httpclient`; no OAuth module is added. The new package owns only
the narrow client-credentials transaction accepted by R3-R7, not a reusable
OAuth framework:

- `net/url` owns deterministic form encoding;
- `net/http` owns the request and Basic header carrier;
- `encoding/json`, `mime`, and `strconv` supply parsing primitives;
- `context`, `sync`, and `time` supply the process-local acquisition lifecycle;
- `internal/infra/httpclient` retains authority, TLS, DNS/address, proxy,
  redirect, timeout, connection, header, and body bounds.

`golang.org/x/oauth2` v0.36.0 was evaluated because it was current on
2026-08-11 and its Go 1.25 minimum fits this repository's Go 1.26.5 directive.
It is rejected, not wrapped:

| Required edge | v0.36.0 result | Disposition |
| --- | --- | --- |
| Explicit Basic without probing | `AuthStyleInHeader` fits | Its request builder is internal and cannot be retained separately from the losing response path. |
| Exact `200` JSON, mandatory Bearer and expiry, no refresh token, strict token grammar | Disposable negative-path probes accepted `201`, wrong media type, absent token type or expiry, quoted expiry, a refresh token, and whitespace in the access token | Repository validation would have to run before and after the library parser, leaving two semantic owners. |
| Caller-cancelable coalescing and one failure wave | Eight concurrent immediate failures produced eight serialized requests; a canceled follower remained blocked | The repository owner would have to replace the cache and synchronization completely. |
| Redacted failures | `RetrieveError` rendered the seeded response body | Raw library errors cannot cross the required boundary. |
| HTTP and gRPC operation fixation | `oauth2.Transport` has no request-context token-source call and no gRPC/control-stream owner | Both attachments must remain repository-owned. |

The authoritative versioned sources are the
[`clientcredentials` implementation](https://go.googlesource.com/oauth2/+/refs/tags/v0.36.0/clientcredentials/clientcredentials.go),
the [token retrieval parser](https://go.googlesource.com/oauth2/+/refs/tags/v0.36.0/internal/token.go),
the [reuse source](https://go.googlesource.com/oauth2/+/refs/tags/v0.36.0/oauth2.go),
and the [transport](https://go.googlesource.com/oauth2/+/refs/tags/v0.36.0/transport.go).
Retaining the module only for hidden form construction would add more code than
the selected form POST and strict parser and would violate single ownership.

`go.mod` and `go.sum` therefore remain unchanged. grpc-go v1.83.0 already names
`golang.org/x/oauth2` v0.36.0 in its own module graph even though the current
main module imports none of its packages. Profile proof therefore asserts no
new direct requirement, import, or reachable OAuth package attributable to
outbound auth; it asserts complete module absence only in an initialized output
where no independently retained dependency, such as grpc-go, owns that module.
Reopen this source decision only if a maintained release exposes a strict
independently composable request/response primitive or a per-call-context owner
with cancelable coalescing and failure-wave suppression. A provider-specific
SDK is not a substitute unless a named provider reopens the portable boundary.

## Authority and configuration

`internal/config.OutboundAuth` is one flat section, not a slice, map, registry,
or named-provider collection. That representation makes zero configured
dependencies invalid in the selected profile and makes a second dependency
unrepresentable; unknown extra keys remain rejected by the current loader.

| Configuration key | Typed value and admission rule | Custody and consumer |
| --- | --- | --- |
| `outbound_auth.dependency` | 1-64 ASCII characters matching `[a-z][a-z0-9._-]*`; it is not a URL or credential | Safe identity used only for fixed client selection, closed logs, and bounded metric labels |
| `outbound_auth.client_id` | Exact valid UTF-8 string, 1-512 bytes, no control characters; never normalized or emitted | Basic username after OAuth form escaping |
| `outbound_auth.client_secret` | Exact valid UTF-8 string, 1-4096 bytes, non-blank, no control characters; never normalized or emitted | Only `APP__OUTBOUND_AUTH__CLIENT_SECRET` may be non-empty; held in the immutable snapshot and credential owner for the process lifetime |
| `outbound_auth.client_authentication` | Required exact value `client_secret_basic` | Selects the only request construction; no default, probe, fallback, or body credential |
| `outbound_auth.token_endpoint` | Absolute HTTPS URL, at most 2048 bytes, non-opaque, with a host and without userinfo, query, forced query, or fragment | Exact request URL; no discovery or endpoint rewriting |
| `outbound_auth.token_target_class` | Required `external_https` or `private_https` | Maps once in bootstrap to the existing `httpclient.TargetClass` owner |
| `outbound_auth.token_private_host_suffix` | Empty for external; required normalized DNS suffix of at most 253 bytes for private | Pre-DNS private-zone admission; not a custom trust-root or proxy control |
| `outbound_auth.scopes` | Space-separated set of at most 32 RFC 6749 scope tokens; each at most 256 bytes, joined value at most 4096 bytes; duplicates rejected and the snapshot stores sorted canonical order | Exact request and response-set comparison; empty means the registration's externally accepted exact default |
| `outbound_auth.resource` | Empty or one absolute URI at most 2048 bytes, without userinfo or fragment | Unique standard `resource` form field |
| `outbound_auth.audience` | Empty or one valid UTF-8 value at most 2048 bytes with no control characters | Unique provider-contract `audience` form field; mutually exclusive with `resource` |
| `outbound_auth.resource_authority` | Absolute HTTPS origin at most 2048 bytes, with no userinfo, path beyond `/`, query, or fragment | Immutable bearer disclosure boundary for both resource transports |
| `outbound_auth.acquisition_timeout` | Required duration in `[100ms,30s]` | One provider attempt and the minimum failure-cooldown input; not a retry or cache-tuning knob |

All section defaults are empty or zero so a selected profile cannot start from
an invented registration. `env/config/local.yaml` carries only empty non-secret
placeholders. `env/.env.example` carries exactly the empty client-secret
placeholder. The existing secret-source policy recognizes `client_secret` and
rejects a non-empty file value before snapshot construction. No CLI leaf,
provider SDK, ambient environment convention, file secret, default credential,
or dynamic reload path is added.

Configuration validation is pure and completes before transport construction
or listener mutation. It canonicalizes only dependency identity, target class,
private suffix, scopes, endpoint spelling, and resource origin; it never trims
or rewrites the client ID or secret. Error text names only the invalid key and a
closed rule. Snapshot, startup, and profile proof use seeded forbidden values to
ensure none can enter errors or observability.

The runtime `oauth2clientcredentials.Config` is the semantic authority for
these bounds. `internal/config` must restate them because its depguard boundary
forbids imports of runtime owners; it may reject more source spellings while
canonicalizing, but may not admit a value the runtime owner rejects. Bootstrap,
the lowest package allowed to import both, owns one shared-corpus parity check.

The client secret and accepted token are process-memory credentials. Close
drops the owner's references, but the design makes no false zeroization claim
for immutable Go strings or transport buffers. Rotation replaces the process.

## Token-endpoint admission and transaction

The credential owner constructs a dedicated `httpclient.Client` from the exact
token endpoint with these fixed bounds:

| Policy | Fixed value |
| --- | --- |
| target | configured `ExternalHTTPS` or `PrivateHTTPS` |
| request and response-header timeout | the acquisition timeout |
| maximum response headers | 32 KiB |
| maximum response body | 64 KiB |
| maximum access-token bytes | 8 KiB |
| maximum connections / idle connections | 1 / 1 |
| retry | disabled |
| correlation propagation | none |
| general outbound HTTP instrumentation | disabled |
| proxy / redirect | none / refused by `httpclient` |

`PrivateHTTPS` is one new target class in the existing target-policy owner. It
requires HTTPS, the explicit private DNS suffix before resolution, a private
resolved address at dial time, ordinary system certificate-chain and hostname
verification, and the same per-request authority check as every bounded
client. It does not accept private IP literals, plaintext, custom application
roots, or a proxy. Platform installation of a private CA into the system trust
store remains a deployment decision.

One admitted acquisition sends exactly one POST to the configured URL. The
body is `application/x-www-form-urlencoded` and contains unique
`grant_type=client_credentials`, optional canonical `scope`, and at most one of
`resource` or `audience`. The Authorization value is Basic over the RFC 6749
form-escaped client ID and secret. The body and URL contain neither credential.
`Accept: application/json` is the only additional protocol header. Provider
context is derived from the owner's lifetime and the acquisition deadline, not
from a caller or startup deadline.

The response is read once through the existing body bound. Acceptance requires
exact status `200`, a successfully parsed `application/json` media type, and
one top-level JSON object. The package's strict decoder rejects trailing data,
duplicate top-level names, wrong known-field types, and ambiguous numeric
spellings while discarding unknown additive fields within the total body cap.
`expires_in` is an unquoted decimal integer in `[1,3600]`; exponent, sign,
fraction, string, null, and overflow spellings fail. The receipt timestamp is
captured when headers arrive, and publication rechecks that expiry remains more
than ten seconds away.

`access_token` must be 1-8192 bytes and match RFC 6750 `b64token` exactly.
`token_type` must be present and equal `Bearer` case-insensitively. A present
scope is parsed with the same token grammar and must be a duplicate-free exact
case-sensitive set match. A present refresh token must be a string and may only
be empty. The owner retains only the opaque access token and absolute expiry;
the response, unknown fields, headers, and body are closed and discarded.

For non-200 responses the package may inspect only a bounded JSON `error` name
to select a closed class, never its description, URI, headers, or body text.
`invalid_client` or status 401/403 is `client_rejected`; other recognized OAuth
grant errors and other 4xx are `grant_rejected`; timeout/408/429/5xx and network
availability failures are provider classes; TLS verification, target, proxy,
redirect, or address-policy failures are `endpoint_trust`; an invalid 200
response is `unsupported_response`. Unknown parser or transport text is
discarded before the owner returns.

## Token lifecycle, cancellation, and failure isolation

The owner has five material states: empty, acquiring, reusable token,
failed-until-window-end, and retired. They are protected by one mutex. It holds
at most one token, one immutable acquisition result shared by current waiters,
one failure result and boundary, and one active provider cancellation. It stores
no waiter collection: each caller waits on the active result channel and its own
context.

### Cache hit and acquisition wave

1. A caller whose context is already done receives `caller_canceled` with the
   safe context identity and does not enter owner state.
2. Under the mutex, a token with more than ten seconds remaining is returned as
   one immutable operation token. At or inside the margin it is discarded.
3. A caller during an active acquisition captures the same result channel. A
   caller inside a prior failure window receives the prior sanitized failure
   immediately.
4. Otherwise one caller creates the wave, fixes the provider deadline,
   increments the one active acquisition, and starts one owner-lifetime
   provider operation.
5. Every leader and follower waits independently for the shared result or its
   caller context. Caller cancellation stops only that wait; it neither cancels
   provider work nor changes another caller's result.
6. Success publishes one token only after the final expiry check, clears prior
   failure state, closes the wave result, and records success. Failure publishes
   no token and computes a cooldown from completion: the maximum of one second,
   `acquisition_timeout`, and a valid `Retry-After` from `429` or `503`, plus
   positive random jitter up to 20 percent, capped at one hour. It stores one
   sanitized failure until that boundary, closes the same result, and records
   failure.
7. At or after that boundary one later caller may create one recovery wave. No
   loop or automatic retry exists.

The provider request remains useful even if its initiating caller stops
waiting; a later caller may consume its success. The maximum active provider
work is one request per process. During continuous fast failure, provider
attempts are bounded by one per completed cooldown per process. A provider that
does not honor context can delay owner close only until the existing process
shutdown budget; the selected `httpclient` path does honor cancellation.

The public constructor uses the process clock. A package-private construction
seam admits a deterministic clock and provider client only for package proof;
neither is runtime configuration or an exported extension point.

### Retirement

After inbound drain and supervised background join, close atomically retires
new acquisition admission, clears the cached credential, cancels the active
provider context, waits for that single wave under the caller's shutdown
context, and then closes the token client's idle pool. Close is idempotent.
It returns nil after the wave joins. If the shutdown context expires first, it
returns the closed sanitized `provider_unavailable` error; it never returns the
provider or transport cause. Partial startup registers the same close
immediately after construction. Telemetry remains alive until this stage
completes, including the canceled provider-attempt result.

## HTTP attachment

`oauth2clientcredentials.HTTPClient` is a concrete Doer around one existing
`httpclient.Client` and the credential owner. Construction requires the base
client's HTTPS origin to equal `resource_authority`. Caller request input cannot
select another owner, token endpoint, or resource authority.

The operation order is:

```text
reject caller Authorization
  -> obtain and fix one operation token
    -> existing resource retry
      -> check that fixed token still has >10s and attach one Bearer value
        -> existing correlation sanitizer and OTel attempt instrumentation
          -> existing fixed authority and response bounds
```

The current `httpclient` chain cannot perform the expiry check inside its retry.
It gains one narrow provider-auth seam: a concrete Do method accepting one
`AttemptAuthorizer func(*http.Request) error`. A package-private transport calls
that function on a cloned request immediately inside each retry attempt and
before the fixed-authority transport. An authorizer failure is marked
non-retryable inside `httpclient` while retaining the sanitized underlying error
identity. Ordinary `Client.Do` and unauthenticated clients are unchanged.

The OAuth HTTP adapter is the only production caller. Its callback captures the
already fixed operation token, rejects any Authorization value again, checks
the ten-second margin with the owner clock, and adds exactly one Bearer header.
It cannot reacquire. The callback sits inside OTel, so token material is absent
when generic HTTP instrumentation observes the request. Existing retry
eligibility, method/idempotency rules, delays, attempt cap, correlation,
redirect refusal, and response semantics remain owned by `httpclient`.

The wrapper returns resource responses unchanged. It records only 401 as
`downstream_unauthenticated` and 403 as `downstream_forbidden`; neither status
is retried by the current resource retry and neither invalidates the cached
token. A retry reaching the margin returns `token_unusable` before resource I/O
and without a second acquisition.

### Deterministic HTTP proof carrier

TD-008 and TD-009 need no production or exported test seam. Their deterministic
concrete/retry carrier stays inside the OAuth package test binary and changes
only that process's DNS and trust inputs before constructing the real clients;
the generated-client compatibility branch extends the existing isolated
generation/subprocess oracle:

1. `harness_test.go` solely owns `testPKI`, `newTestPKI`, and the
   package-private `suiteTestPKI` reference. `goleak_test.go`'s package
   `TestMain` calls that constructor once, assigns the one suite reference,
   sets `GODEBUG=x509usefallbackroots=1`, and installs that CA once with
   `x509.SetFallbackRoots`; it does not create a second CA or certificate
   source. The pinned Go 1.26.5 contract makes a client with nil `RootCAs` use
   that pool on macOS and other supported runners, so `httpclient.New` keeps
   normal certificate-chain and hostname verification.
2. `harness_test.go` also owns `privateTestAddress`,
   `startPrivateHTTPSTestServer`, and `installPrivateTestResolver`. They find
   the same bindable non-loopback private IPv4 class
   already required by the repository's HTTP correlation component proof,
   start bounded TLS token and resource servers on that address with distinct
   fixed hostnames signed by the test CA, and temporarily maps both names to
   that address through a package-local UDP DNS server installed as
   `net.DefaultResolver`.
3. `http_test.go` constructs the token owner through the already-selected
   package-private `newClient` clock/concrete-provider construction seam in
   `client.go`, constructs the resource client with public `httpclient.New`
   using `PrivateHTTPS`, its fixed hostname and port, the real retry policy,
   and ordinary nil TLS roots, then calls the fixed public
   `NewHTTPClient(*Client, *httpclient.Client)` constructor. `newClient`
   accepts only the deterministic clock and an already constructed concrete
   token `*httpclient.Client`; it is not an interface or alternate resource
   transport path, and public `New` remains the sole production caller.
4. Requests therefore cross the real attempt authorizer, retry, generic OTel,
   fixed-authority, private-address-at-dial, TLS hostname/chain, response-bound,
   redirect, and response paths. The fixture replaces no transport, dialer,
   TLS config, response path, or retry decision after construction.
5. Existing `internal/infra/httpclient/generated_client_test.go` owns the
   generated-client half of TD-008 under outer test
   `TestGeneratedClientUsesAuthenticatedDoer`. It extends its current pinned
   in-module generation/subprocess oracle: the parent test owns controlled TLS
   token/resource servers, DNS, and a temporary CA file, while the generated
   child test binary installs that CA as its one fallback root and points its
   own `net.DefaultResolver` at the parent DNS socket using environment-carried
   fixture addresses, then constructs the public OAuth owner and concrete
   resource client, calls `NewHTTPClient`, and injects the returned Doer through
   the generated `WithHTTPClient` option.
   This compile-and-run branch uses no OAuth private state and does not cover
   retry-margin timing; `http_test.go` remains the owner of the deterministic
   concrete/retry assertions. The outer test imports no OAuth package in its
   compiled `httpclient` source, so production and test package imports remain
   acyclic.

The OAuth package owns the deterministic concrete/retry carrier because it is
the narrowest package that can exercise its private clock/provider controls and
the public concrete composition at once. Test-only imports point from OAuth
tests to the standard library and the existing `httpclient` dependency;
production import direction does not change. The `httpclient` package retains
its separate `attempt_authorization_test.go` leaf proof and its existing
generated-client subprocess oracle; neither compiled test package imports
OAuth.

Each OAuth test releases held work and closes every response body, then calls
`Client.Close` once so the owner alone closes the token idle pool, closes the
resource client's idle pool, shuts down both TLS servers/listeners, closes the
DNS socket, restores `net.DefaultResolver`, and joins the server and DNS
goroutines. The generated-client parent owns the equivalent server, DNS,
temporary-directory/CA-file, and child-process cleanup; the child closes its
OAuth owner, resource idle pool, and response bodies. Tests using a replaced
process resolver do not run in parallel. `TestMain` restores the prior
`GODEBUG` value after the suite; fallback roots intentionally last for the
OAuth test process because Go permits setting them only once. The generated
child has its own one-shot root. `go test` builds and runs packages as separate
test binaries, so neither trust root alters another package test or production.

Rejected carriers remain smaller only by bypassing the claimed path:

- an `export_test.go` symbol in `httpclient` is not compiled into the imported
  package used by the OAuth test binary;
- a same-package `httpclient` test importing OAuth creates the forbidden
  `httpclient -> OAuth -> httpclient` cycle;
- a test-support package cannot reach either package's private state and is
  unnecessary when DNS and trust drive public construction;
- a build-tagged hook still creates a tagged exported API and a non-default
  test command;
- `unsafe`, `go:linkname`, default-transport mutation, or field reflection
  bypasses package ownership or loses to `httpclient.New` rebuilding its own
  dial/TLS chain.

Reopen this proof-carrier decision only if the supported Go toolchain removes
forced fallback roots, a supported runner cannot supply the non-loopback
private address already required by repository component proof, or the real
implementation introduces a process-global resolver or trust consumer that
cannot be isolated inside this package test binary. Such evidence reopens Go
Ownership/Test Design; it does not authorize weakening production admission.

## gRPC attachment

`oauth2clientcredentials.NewGRPCClient` is the one public construction for an
authenticated dependency connection. It creates one private
`credentials.PerRPCCredentials`, installs it connection-wide through
`grpcclient.Options`, wraps the raw `*grpc.ClientConn` as the standard
`grpc.ClientConnInterface` given to generated or concrete clients, and installs
one terminal result observer around the existing OTel stats handler. It rejects
preconfigured per-RPC credentials or a competing observer, so application and
grpc-go control paths cannot be wired under different auth policies.

`grpcclient.Options.ObserveRPC` is the only shared-client addition. Its zero
value changes nothing; when present, it receives one terminal result from
grpc-go's connection stats path after the existing OTel handler, without
request or response values. grpc-go v1.83.0 is already pinned. Its
`ClientConn.NewStream` carries the caller context through transparent attempts,
and transport creation invokes connection credentials for each attempt. Its
standard health checker deliberately bypasses application interceptors and
creates a non-retry control stream on the raw connection; the connection-wide
credential is therefore the one existing seam that covers both paths. The
versioned authorities are [`newAttemptLocked`](https://github.com/grpc/grpc-go/blob/v1.83.0/stream.go#L454-L508),
[connection credential invocation](https://github.com/grpc/grpc-go/blob/v1.83.0/internal/transport/http2_client.go#L664-L725),
and [client health stream construction](https://github.com/grpc/grpc-go/blob/v1.83.0/clientconn.go#L1594-L1662).

For an application `Invoke` or `NewStream`, the wrapper first rejects any
case-insensitive outgoing `authorization` metadata and any per-call
`PerRPCCredentials` option. Rejecting the whole call-credential option is the
only way to guarantee one authorization source without invoking an unknown
credential for inspection. It then obtains one operation token, places an
unexported immutable operation state in the call context, and delegates to the
raw connection. Generated clients see only `grpc.ClientConnInterface`.

On each transport attempt, `GetRequestMetadata` validates that grpc-go's request
URI has the configured HTTPS resource origin. For application contexts it reads
only the fixed operation state, checks the margin, and returns exactly one
lowercase `authorization` entry. It never reacquires. If a transparent attempt
crosses the margin, it marks that operation `token_unusable`; the application
wrapper returns the closed local error rather than grpc-go's credential-wrapper
text.

For a control-stream context, where no application operation state exists, the
credential obtains a current token using that control stream's context and the
same owner. grpc-go's `Health/Watch` creates a new non-retry stream for each
reconnect, so each new stream receives a then-usable token. Once any application
or control stream is established, the metadata is immutable; expiry does not
replace it in place.

The credential always returns `RequireTransportSecurity=true`. grpc-go rejects
it at connection construction when paired with insecure transport credentials.
The package also validates the request URI before returning metadata, so an
authority mismatch sends no bearer and reports `invalid_configuration`. The
same binding may serve application and control traffic only because the one
configured registration, scopes, resource/audience, method, authority, and
failure policy are identical; a second binding is outside R2.

The connection stats observer records only terminal `Unauthenticated` or
`PermissionDenied` for both application and grpc-go control RPCs. The
application wrapper remains responsible only for logical operation-token
fixation. Statuses return unchanged and never trigger a new token or transparent
replay. Reconnect, resume, replay, cursor, and idempotency remain concrete-client
policy.

## Failure contract and telemetry

The package owns one `FailureClass` enum and sanitized error type. The only
classes are the R7 set:

| Class | Returned/recorded boundary |
| --- | --- |
| `invalid_configuration` | Static section or HTTP/gRPC resource-binding mismatch |
| `endpoint_trust` | Token endpoint URL, TLS, authority, redirect, proxy, or address admission |
| `caller_canceled` | One HTTP, RPC, stream, or control caller stops waiting |
| `provider_timeout` | The independent acquisition deadline expires |
| `provider_unavailable` | Provider transport or availability failure, including owner retirement |
| `client_rejected` | Authorization server rejects client authentication |
| `grant_rejected` | Authorization server rejects the fixed grant or privilege request |
| `unsupported_response` | Status/media/size/JSON/token/type/expiry/scope/refresh contract failure |
| `token_unusable` | An operation-fixed token reaches the early-expiry margin before an internal attempt |
| `downstream_unauthenticated` | Resource HTTP 401 or gRPC `Unauthenticated`; telemetry only |
| `downstream_forbidden` | Resource HTTP 403 or gRPC `PermissionDenied`; telemetry only |

Returned auth errors contain only a fixed safe sentence and class. Only caller
cancellation unwraps the safe `context.Canceled` or `context.DeadlineExceeded`
identity. Provider transport, parser, header, body, OAuth description/URI,
endpoint, client ID, secret, scopes, resource, audience, and token details are
classified and discarded before return. Downstream results are never wrapped
or translated.

The token client disables general HTTP spans and remote propagation. The
package adds metrics only, under meter `service.outbound_auth`:

| Instrument | Closed attributes |
| --- | --- |
| `outbound.auth.token.resolutions` counter | dependency, source=`cache|acquisition`, result, failure class on failure |
| `outbound.auth.provider.attempts` counter | dependency, result, failure class on failure |
| `outbound.auth.provider.attempt.duration` histogram in seconds | same provider-attempt attributes |
| `outbound.auth.resource.rejections` counter | dependency, transport=`http|grpc`, result=`unauthenticated|forbidden`; gRPC includes application and control RPC terminals |

Attribute sets are prebuilt because every set is closed and the configured
dependency count is one. No identity other than the validated dependency name,
no URL, registration value, scope/resource/audience, provider status/body, token
age/value, or caller value is an attribute. Instrument construction degrades to
no-op and emits at most one closed `outbound_auth_metrics_degraded` warning;
telemetry failure never changes authentication. Bootstrap may emit one
`outbound_auth_configured` record with the dependency name and no other auth
configuration. It emits no per-request provider failure log.

## Health, degradation, performance, and shutdown

Static invalidity stops configuration before listeners or provider I/O. A
valid selected profile performs no eager acquisition. Provider state contributes
no startup admission, readiness probe, synchronous health I/O, or liveness
condition. Failure affects only operations using this dependency. grpc-go's
resource connection health remains its existing load-balancing control and does
not become service readiness.

The hot cache path performs one time read, one mutex critical section, and one
closed-label counter update. State is O(1), provider concurrency is one, and
there is no retained waiter queue. A healthy process performs approximately one
grant per accepted token lifetime; continuous failure performs at most one
grant per completed cooldown. Fleet provider load is those ceilings multiplied
by replicas; per-process jitter reduces synchronized recovery but does not
coordinate the fleet. No latency, throughput, quota, or fleet-capacity claim is
made. Reopen synchronization only if a representative benchmark shows the
single mutex materially consumes an accepted operation budget; reopen fleet
coordination only from measured synchronized-expiry or provider-quota pressure.

Shutdown order is fixed:

1. existing readiness drain stops new inbound work while admitted work retains
   auth capability;
2. existing HTTP/gRPC server drain completes;
3. supervised background work that may use dependency clients is canceled and
   joined;
4. adopting composition closes resource gRPC connections/control streams and
   HTTP idle pools;
5. the auth owner retires acquisition, cancels and joins provider work, clears
   its token, and closes the token HTTP idle pool;
6. other runtime dependencies close; telemetry flushes last.

The base template currently has no concrete outbound resource client, so its
new bootstrap stage owns only step 5. Ordered shutdown stores that close result
and includes it once in the existing terminal `errors.Join` beside serving and
background failures. The partial-start deferred safety net assigns a close
failure into `runWithRuntime`'s named return. A close-completed flag prevents the
defer from joining the same result twice. Thus a provider that fails to stop
within the shared shutdown budget is process-terminal and reaches the existing
process-exit record; it is never silently discarded. A future dependency client
is composed and closed in the same `startup_outbound_auth.go` stage rather than
moving auth or credentials into feature code.

## Profile and generated authority

`scripts/init-module.sh` remains the handwritten generator authority. It parses
`OUTBOUND_AUTH` before any mutation: omission selects `none`; explicit empty or
an unknown value fails; `oauth2-client-credentials` requires
`OUTBOUND_HTTP=bounded`, `GRPC=enabled`, or both. It records `outbound_auth` in
the generated `template.lock`.

Three marker groups keep the output minimal. The core group also owns the
source-only `OUTBOUND_AUTH=none` argument in
`scripts/ci/runtime-image-build.sh`: that script returns before fixture setup in
an initialized service, but the literal is still an outbound-auth environment
surface and must not survive a `none` output.

| Marker | Retained when | Contents |
| --- | --- | --- |
| `outbound-auth-oauth2-client-credentials` | OAuth selected | core package, config, bootstrap, telemetry, endpoint transport, docs, examples, lock input |
| `outbound-auth-http` | OAuth and bounded outbound HTTP selected | HTTP adapter, the `httpclient` per-attempt authorization seam, and the marked generated-client authenticated-Doer proof branch |
| `outbound-auth-grpc` | OAuth and gRPC selected | gRPC client/credential, optional terminal observer seam, and proof |
| `credential-provider-http` | OIDC or outbound OAuth selected | the bounded HTTP option and transport branch that omit general instrumentation for credential-adjacent provider requests |

OAuth selection always prevents deletion of `internal/infra/httpclient`, even
for gRPC-only output, because token acquisition uses it. `AUTHN=oidc-jwt`
continues to retain the same package independently. The existing
`DisableInstrumentation` option moves from its OIDC-only marker into the
union-owned `credential-provider-http` marker. It remains when either credential
pack has a present caller and is stripped when both are absent; an ordinary
bounded-HTTP-only output keeps no dormant credential option.

When `OUTBOUND_AUTH=none`, every core, HTTP, and gRPC auth marker is stripped,
the config/environment/guidance/bootstrap/package surfaces disappear, and no
direct OAuth requirement, import, or reachable OAuth package attributable to
this capability is present. A transitive module retained solely by another
selected owner is not an outbound-auth artifact. When a resource transport is
absent, only its adapter and companion seam disappear. The generator still runs
root and tools `go mod tidy`; no OpenAPI, protobuf, SQLC, migration, container,
or manifest generation is triggered.

`scripts/ci/template-init-check.sh` remains the independent retention/removal
oracle and covers selected/none crossed with HTTP-only, gRPC-only, both, and the
invalid no-consumer selection. It checks unresolved markers, config and secret
placeholder presence/absence, package/adapters, the union-owned credential HTTP
option, `template.lock`, build, repeat initialization, bounded-HTTP retention,
and the direct/reachable dependency rule. It does not consume the generator's
own inventory. Pull-request CI invokes that exact oracle with
`TEMPLATE_INIT_PROFILE=outbound-auth`; profile markers remove the job from
generated services that do not retain this capability. `template-owned.paths`
is unchanged because it owns upstream
synchronization, not initialized runtime profiles.

The initialized source tree and `template.lock` are generated. All Go, config,
generator, CI-oracle, and guidance sources in the template remain handwritten.
No generated client, OpenAPI document, protobuf, provider metadata document, or
SDK becomes OAuth authority.

## External checkpoints and reopen conditions

The portable design is closed without inventing these inputs. Each remains a
hard checkpoint before the narrower claim it owns:

| External owner input | Earliest checkpoint and effect |
| --- | --- |
| Registration, exact endpoint, grant, Basic support, rate/error contract | Before provider-specific configuration; a required stronger method or discovery reopens Specification |
| Least-privilege scopes and one resource or audience | Before adopter config or authorization-compatibility claim; another principal/resource reopens R2 |
| Secret issuance, delivery, overlap, rotation, revocation | Before deployment; restart remains the only rotation mechanism |
| Token lifetime and revocation exposure | Before accepting the one-hour bearer window; a tighter required window reopens R4/R5 |
| Workload identity, private key, mTLS, certificate binding, DPoP | Before accepting Basic for a named deployment; selection reopens the affected behavior |
| Public/private route, system trust roots, no-proxy compatibility, allowlists | Before provider-specific config; mandatory proxy or incompatible private TLS blocks that deployment |
| Dependency criticality | Before changing startup/readiness; default stays optional |
| Replica count, quota, provider latency/capacity and `Retry-After` contract | Before any proactive refresh, shared cache, broker, or fleet-capacity claim; local bounded jitter remains only a portable recovery floor |
| Long-lived stream expiry, reconnect, resume, replay | Before continuity claims beyond stream creation; concrete RPC owner supplies policy |

The three Research reopen conditions remain unchanged: a materially distinct
candidate family, invalidation of the Basic interoperability floor, or a change
to the reusable responsibility boundary. Library drift that only changes the
implementation-source choice reopens this design.

## Rejected same-level alternatives

| Alternative | Why it loses |
| --- | --- |
| `x/oauth2` wrapped by response-inspecting transports | It retains only hidden form construction while strict parsing, lifecycle, redaction, and both attachments are duplicated outside it |
| `oauth2.NewClient` or `oauth2.Transport` | Creates or owns a resource path outside current authority/retry/correlation/cleanup and cannot serve gRPC |
| Feature-level token/header helpers | Expose credentials and duplicate policy at every caller |
| One HTTP client for token and resource endpoints | Conflates authorities, disclosure rules, retry, telemetry, and pools |
| Disable resource retry | Changes current concrete-client behavior instead of composing the accepted fixed-token rule |
| New generic transport or auth interface | One grant and one configured dependency provide no second implementation or consumer |
| `grpcclient` auth interceptors | The standard `ClientConnInterface` wrapper fixes application operations; connection credentials cover control streams and the stats handler supplies one terminal observation point |
| Eager acquisition or readiness probing | Turns an optional provider into process admission and synchronous health I/O |
| Proactive refresh, circuit breaker, shared cache, broker, sidecar, persistent state | No accepted behavior or measurement earns them |
| Provider SDK, workload identity, private-key JWT, mTLS, DPoP, token exchange, SPIFFE | Changes provider/principal/credential/resource-transport authority and is outside the minimum |

## Go responsibility map

| Responsibility and path | Current evidence and selected owner | Exact action, boundary, and proof owner |
| --- | --- | --- |
| Immutable one-dependency schema and secret policy | `internal/config` owns section-local schema/default/validation and already rejects non-empty secret-like file keys | Add `internal/config/outbound_auth_config.go`; mark the field/calls in `types.go`, `defaults.go`, and `validate.go`. Config tests own bounds, environment-only secret, snapshot omission, and redaction. `internal/config` never imports runtime adapters. |
| Public/private HTTPS endpoint admission | `internal/infra/httpclient/target_policy.go` already owns pre-request and post-DNS target policy | Add profile-marked `PrivateHTTPS` there and update `config.go` wording. `target_policy_test.go` owns TLS/suffix/address refusal. No OAuth package reimplements DNS/TLS. |
| Token HTTP client and strict grant transaction | Existing bounded client owns transport but no OAuth transaction; `x/oauth2` is eliminated above | Add `internal/infra/oauth2clientcredentials/provider.go`; it depends inward on `httpclient` and stdlib only. `provider_test.go` owns exact wire/strict parser/redaction, with shared fixtures in `harness_test.go`. |
| Runtime policy and limits | Config cannot import runtime owners; provider/client need one immutable validated value | Add package `config.go` for adapter `Config`, constants, canonical authority/scope validation, and the config-to-runtime parity contract. Bootstrap is the only package importing both representations; `startup_outbound_auth_test.go` owns parity. |
| Closed failures | Provider, lifecycle, two adapters, and telemetry need one safe taxonomy | Add `FailureClass` and its constants to `vocabulary.go`; add private sanitized errors and exported class lookup to `errors.go`. No raw provider cause is retained; downstream statuses remain outside it. `vocabulary_test.go` and `errors_test.go` own spelling/bounds and text/class/disclosure respectively. |
| Cache, wave, caller/provider cancellation, failure window, close | No outbound owner exists; `oidcjwt` is lifecycle precedent but inbound policy is not reusable | Add package `client.go` with concrete `Client`, operation token, one wave, process context, and idempotent close. Public construction is concrete; only a package-private clock/provider seam exists. `client_test.go` owns state and `goleak_test.go` the package lifecycle gate. |
| Adapter metrics and safe warning | Adapter-owned closed signals belong beside the adapter, not telemetry SDK setup; repository naming requires one literal owner | Add `vocabulary.go` for every metric/log literal and `telemetry.go` for instruments, prebuilt attributes, recording, and no-op degradation. `vocabulary_test.go` owns the closed label space; `telemetry_test.go` owns construction, recording, degradation, and forbidden values. |
| HTTP logical-operation fixation and deterministic cross-package proof | `httpclient` owns retry and currently exposes no inner-attempt seam; OAuth package tests alone can combine private owner controls with public concrete HTTP construction, while the existing `httpclient` subprocess oracle owns generated-client consumption without a compile-time import cycle | Add profile-marked `httpclient/attempt_authorization.go` plus one `Client.DoWithAuthorization` entry in `client.go` and one nonretryable check in `retry.go`. Add OAuth `http.go` as the sole production caller. `httpclient/attempt_authorization_test.go` and `retry_test.go` own leaf parity. OAuth `harness_test.go`, `goleak_test.go`, and `http_test.go` own the process-local CA/DNS/TLS carrier and deterministic TD-008/TD-009 concrete/retry composition through public `httpclient.New` and `NewHTTPClient`. Existing `httpclient/generated_client_test.go` owns the parent-hosted controlled endpoints and generated child compile/run for `TestGeneratedClientUsesAuthenticatedDoer`; no production surface is added for proof. |
| gRPC application/control binding | `grpcclient.Options.PerRPCCredentials` covers the raw connection and grpc-go health; its stats handler sees terminal application/control results; generated clients accept `grpc.ClientConnInterface` | Add OAuth `grpc.go` with one private credential/application wrapper and exported complete connection constructor. Add the optional `grpcclient.Options.ObserveRPC` stats-handler callback; no second transport or interceptor policy. `grpc_test.go` owns construction, unary/stream/control, terminal rejection telemetry, and fixed-attempt behavior using pinned grpc-go v1.83.0. |
| Startup and shutdown composition | `cmd/service/internal/bootstrap/run.go` owns order, terminal error aggregation, and partial-start cleanup | Add `startup_outbound_auth.go`; mark wiring/construction/close in `run.go`. It maps config, builds token client/owner without I/O, registers close, joins ordered close failure once into the terminal error, joins partial-start close failure into the named return, and exposes the local owner for later concrete clients. It adds no readiness probe. `startup_outbound_auth_test.go` and `run_lifecycle_test.go` own order, single error propagation, and cleanup. |
| Profile generation and dependency cleanup | `scripts/init-module.sh`, `template.lock`, independent CI oracle, and the source-only runtime-image fixture already own these | Add selector, markers, lock field, retention predicate, README derivation, and oracle matrix; mark the fixture's fixed `OUTBOUND_AUTH=none` input as core so generated `none` outputs remove it. No `go.mod` edit; both tidy paths remain. Shell oracle owns generated-tree proof. |
| Operator and architecture guidance | Current docs own configuration, package boundaries, and adopter composition | Add `docs/outbound-machine-authentication.md`; update README, repository architecture, project structure, and configuration-source policy under profile markers. No rollout or deployment artifact is added. |

Import direction is acyclic:

```text
cmd/service/internal/bootstrap
        +--> internal/config (pure snapshot)
        +--> internal/infra/oauth2clientcredentials
        |          +--> internal/infra/httpclient --> net/http
        |          +--> internal/infra/grpcclient --> grpc-go
        |               (gRPC-marked file only)
        +--> resource concrete/generated clients (adopter-owned)
```

`httpclient` never imports the OAuth package. Feature and generated packages
never import config, OAuth errors, or token types. The new package does not
import bootstrap, health, feature code, generated clients, OIDC, or inbound
authentication.

### Exact cross-package surface

The new package exports only the surface with a present bootstrap, transport,
or feature-client consumer:

- `type Config struct` with exactly `DependencyName string`, `ClientID string`,
  `ClientSecret string`, `ClientAuthentication string`, `TokenEndpoint string`,
  `TokenTargetClass httpclient.TargetClass`, `TokenPrivateHostSuffix string`,
  `Scopes []string`, `Resource string`, `Audience string`,
  `ResourceAuthority string`, and `AcquisitionTimeout time.Duration`;
- `func New(Config, metric.MeterProvider, *slog.Logger) (*Client, error)`, which
  validates runtime policy, constructs the dedicated token HTTP client and
  private telemetry, and performs no provider I/O;
- `type Client struct` with private state and
  `func (*Client) Close(context.Context) error`; token resolution, operation
  token, clock, wave, provider client, and state transitions remain private;
- `type FailureClass string`; exported constants
  `FailureInvalidConfiguration`, `FailureEndpointTrust`,
  `FailureCallerCanceled`, `FailureProviderTimeout`,
  `FailureProviderUnavailable`, `FailureClientRejected`,
  `FailureGrantRejected`, `FailureUnsupportedResponse`,
  `FailureTokenUnusable`, `FailureDownstreamUnauthenticated`, and
  `FailureDownstreamForbidden`; and
  `func FailureClassOf(error) (FailureClass, bool)`; the concrete sanitized
  error and construction remain private;
- `type HTTPClient struct`,
  `func NewHTTPClient(*Client, *httpclient.Client) (*HTTPClient, error)`, and
  `func (*HTTPClient) Do(*http.Request) (*http.Response, error)`; no OAuth HTTP
  interface or token-source method is exported;
- `type GRPCClient struct`,
  `func NewGRPCClient(*Client, grpcclient.Config, grpcclient.Options) (*GRPCClient, error)`,
  the `Invoke` and `NewStream` methods required by `grpc.ClientConnInterface`,
  and `func (*GRPCClient) Close() error`; the credential, raw connection,
  application connection, stream wrappers, and terminal observer remain
  private.

The one shared HTTP transport seam exports
`type AttemptAuthorizer func(*http.Request) error` and
`func (*httpclient.Client) DoWithAuthorization(*http.Request,
AttemptAuthorizer) (*http.Response, error)`. The OAuth HTTP adapter is its only
production caller. All telemetry types and record methods, provider wire types,
strict decoder types, operation state, and test substitutions are private.
There is no exported token, token source, provider interface, factory,
registry, cache, refresh control, clock, retry control, test constructor,
transport hook, dialer hook, TLS hook, or root-pool hook.

## Inverse Go file map

Every listed file has one present reason to change. Files not listed do not
move or acquire OAuth responsibility.

| File action | Present reason, declarations, dependencies, and forbidden ownership |
| --- | --- |
| add `internal/infra/oauth2clientcredentials/doc.go` | Package contract for operator-facing acquisition, transport-adapter consumers, and lifecycle maintainers; names the deliberately absent generic token-source, registry, refresh, discovery, and provider extension seams. No behavior or generation directive. |
| add `internal/infra/oauth2clientcredentials/config.go` | Exported runtime `Config`, fixed bounds, canonical scope/resource authority, validation. May use stdlib and `httpclient.TargetClass`; no loading, I/O, lifecycle, or provider behavior. |
| add `internal/infra/oauth2clientcredentials/vocabulary.go` | Exported `FailureClass` and its eleven constants plus every closed metric attribute, attribute value, meter/instrument name, and operator event/field literal, including length/cardinality ceilings. No instrument construction or error behavior. |
| add `internal/infra/oauth2clientcredentials/errors.go` | Private sanitized error construction plus exported `FailureClassOf` lookup behavior used by provider/client/adapters; spellings come only from `vocabulary.go`. No HTTP/gRPC status translation or raw cause retention. |
| add `internal/infra/oauth2clientcredentials/client.go` | Exported concrete owner/New/Close, private token/wave state, acquisition admission, caller wait, failure boundary, retirement. No package comment, form/JSON, or carrier attachment. |
| add `internal/infra/oauth2clientcredentials/provider.go` | Exact Basic form POST, bounded response classification, strict token response, token endpoint client ownership. No cache, resource request, or retry. |
| add `internal/infra/oauth2clientcredentials/telemetry.go` | Private instrument construction/recording, prebuilt label sets, no-op degradation, one warning; every literal comes from `vocabulary.go`. No SDK/exporter setup or provider content. |
| add `internal/infra/oauth2clientcredentials/http.go` | Exported `HTTPClient`, constructor and Doer; resource-origin admission, operation fixation, attempt authorizer, 401/403 observation. Whole file is `outbound-auth-http`; no generic retry policy. |
| add `internal/infra/oauth2clientcredentials/grpc.go` | Exported `GRPC`, constructor, required credential methods and Wrap; private application connection/stream wrappers and fixed operation state. Whole file is `outbound-auth-grpc`; no connection construction, resolver, retry, or long-stream recovery. |
| add package sibling `config_test.go`, `vocabulary_test.go`, `errors_test.go`, `client_test.go`, `provider_test.go`, `telemetry_test.go`, `http_test.go`, `grpc_test.go`, `doc_test.go` | Each proves its same-named owner only; `doc_test.go` checks the package contract/exported seam inventory. `http_test.go` additionally owns deterministic TD-008/TD-009 concrete/retry construction through package-private `newClient`, public `httpclient.New`, public `NewHTTPClient`, and the real controlled TLS resource path. Generated-client proof stays with the existing `httpclient` oracle below. HTTP/gRPC files follow their transport markers. |
| add package sibling `harness_test.go` | Sole owner of `testPKI`, `newTestPKI`, `suiteTestPKI`, deterministic clock, hostname certificate issuance, bounded TLS token/resource servers on a bindable non-loopback private IPv4 address, temporary UDP DNS plus resolver restoration, seeded forbidden values, and event gates used by at least client/provider/HTTP/gRPC proof. It imports only stdlib/test dependencies and `httpclient` already allowed by the production package; production cannot import it. Every listener/socket/goroutine is test-owned and joined. |
| add package sibling `goleak_test.go` | One package `TestMain` calls harness-owned `newTestPKI` once, assigns `suiteTestPKI`, installs that same process-only fallback CA before any TLS proof, restores `GODEBUG` after the suite, and owns the package-wide acquisition/server/DNS shutdown leak gate. It creates no CA or certificate source and cannot restore the one-shot fallback pool, which is confined to this package's test binary. No feature assertions. |
| add `internal/config/outbound_auth_config.go` and `_test.go` | Raw section type, empty defaults, pure validation/canonicalization and its proof; whole file pair uses the core marker. |
| change `internal/config/types.go` | Add only the marked `OutboundAuth` section field. |
| change `internal/config/defaults.go` | Merge only the marked section defaults. |
| change `internal/config/validate.go` | Invoke only the marked section validator in existing order before runtime owners. |
| change `internal/config/load_environment_test.go`, `secret_policy_test.go`, `snapshot_contract_test.go` | Prove the environment-only secret, file rejection, known-key/snapshot redaction, and selected/stripped representation. No runtime behavior. |
| change `internal/infra/httpclient/config.go` | Move the credential-adjacent option to the union marker, generalize its wording, and update private-suffix wording; accept no provider auth. |
| change `internal/infra/httpclient/target_policy.go` | Add the marked private-HTTPS scheme/suffix/address branch at the existing policy authority. |
| add `internal/infra/httpclient/attempt_authorization.go` and `_test.go` | Whole `outbound-auth-http` pair owns the one function callback, cloned attempt mutation, and private nonretryable wrapper. No token acquisition or Bearer policy. |
| change `internal/infra/httpclient/client.go` | Insert the marked attempt transport between generic attempt instrumentation and fixed-authority/response-bound transport, and expose the single `DoWithAuthorization` entry that carries its callback only on that operation. Ordinary `Do` crosses the same transport as a no-op authorizer path. The file also has independent current S3 ownership for one-attempt HTTP/1, caller-selected immutable roots, and request-deadline propagation; those remain S3 D2/T7/T9A policy and cannot be moved into OAuth. |
| change `internal/infra/httpclient/retry.go` and `retry_test.go` | Treat the private attempt-authorization error as nonretryable. `attempt_authorization_test.go` owns per-attempt cloning, instrumentation, no-disclosure, and no-resource-on-authorizer-failure; `retry_test.go` retains the narrow classification parity check. Existing generic repeatability helpers remain `httpclient` retry policy, not OAuth behavior. |
| change `internal/infra/httpclient/generated_client_test.go` | Retain its sole present reason: pinned generated-client composition. Add an `outbound-auth-http`-marked extension to its parent-owned temporary in-module generation/subprocess oracle with controlled TLS token/resource servers, private DNS, temporary CA input, and outer `TestGeneratedClientUsesAuthenticatedDoer`; the parent passes only fixture addresses and paths through the child environment, and the generated child installs its own process fallback root and resolver, constructs public OAuth/resource clients, calls public `NewHTTPClient`, injects the authenticated Doer, executes one admitted request, and cleans up. One-file-only PKI/server/DNS fixture declarations stay in this file. The compiled parent does not import OAuth; no retry-margin or OAuth-private-state assertions move here. Stripping the marked branch preserves the existing bounded-client generated composition test. |
| add `internal/infra/httpclient/config_test.go` | Move existing configuration-admission proof out of the overloaded client test; own only config parsing and finite bounds, explicitly excluding target class, scheme, suffix, authority, and address policy. |
| add `internal/infra/httpclient/target_policy_test.go` | Move existing authority and dial-address policy proof from `client_test.go`; add only private-HTTPS scheme, suffix, per-request authority, and public/private pre/post-DNS address admission. |
| add `internal/infra/httpclient/transport_test.go` | Move existing response-limit transport/body proof from `client_test.go`; no target or client-chain assertions. |
| add `internal/infra/httpclient/harness_test.go` | Move `validExternalConfig`, `roundTripFunc`, and only the fixtures currently shared by client, retry, propagation, target, or transport test files. |
| move `internal/infra/httpclient/authn_policy_test.go` to `credential_provider_contract_test.go` | Put the OIDC/outbound-auth union marker on the executable credential-adjacent no-instrumentation boundary, rename its test `TestCredentialProviderDisablesGeneralInstrumentation`, and remove its timeout, size, connection, proxy, and redirect assertions. |
| change `internal/infra/httpclient/client_test.go` | Retain only client construction, complete chain including ordinary TLS defaults and proxy refusal, connection reuse, redirect refusal, and cross-transport integration proof after the required splits. |
| add `cmd/service/internal/bootstrap/startup_outbound_auth.go` and `_test.go` | Config-to-adapter mapping, token-client/owner construction without I/O, sanitized close result, and parity proof. No feature client or health policy. |
| change `cmd/service/internal/bootstrap/run.go` | Add marked runtime-wiring construction, immediate cleanup registration, close-completed guard, ordered close after background join/before dependency close, and exactly-once terminal error join on ordered or partial-start paths. |
| change `cmd/service/internal/bootstrap/run_lifecycle_test.go` | Prove partial-start and ordered close relative to drain/background/dependencies/telemetry plus exactly-once propagation of a close deadline. |

No test-only exported API, generic token source, factory, registry, generated
Go, or cross-package fixture package is admitted. `doc.go` exists because the
package has operator, transport-adapter, and lifecycle audiences plus several
deliberately absent seams; it does not compensate for an unclear file split.

The file map reopens only if implementation evidence shows that the strict
response parser and provider transaction change independently enough to need
separate owners, or grpc-go's pinned attempt/control behavior contradicts the
verified call paths. Ordinary private helpers and behaviorally equivalent Go
syntax remain Implementation choices.

### T4 shared-HTTP provenance reconciliation

Tree `50e932807b9d730fc3ba67dc43ec9916f50ff4a0` is the immutable replacement
T4 snapshot. Its four `httpclient` blobs are not a formatting baseline for the
current Local candidate: the executable delta is a composition of the accepted
owners below. This reconciliation preserves the snapshot as provenance without
reclassifying any current code as a formatting-only continuation or changing a
ledger receipt.

| Current Local path | Snapshot-to-Local disposition | Current owner and preserved boundary |
| --- | --- | --- |
| `internal/infra/httpclient/client.go` | T4 retains the context-carried `AttemptAuthorizer`, cloned per-attempt request, one public `DoWithAuthorization` entry, and its placement inside retry and generic OTel. S3 adds one-attempt HTTP/1, explicit immutable roots, and request-deadline propagation. | T4 owns only the authorization seam; S3 D2/T7/T9A owns the generic transport policy. Neither owner may change ordinary `Do`, fixed-authority admission, retry eligibility, or TLS verification to realize its part. |
| `internal/infra/httpclient/generated_client_test.go` | The parent/child TLS, DNS, temporary-CA, fallback-root, resolver, public-constructor, and cleanup branch is substantive T4 TD-008 proof, not formatting. | T4 owns this one existing generated-client oracle extension. The compiled parent remains OAuth-free; only the generated child imports OAuth, and no OAuth-private-state or retry-margin assertion enters this file. |
| `internal/infra/httpclient/retry.go` | The attempt-authorization wrapper remains a nonretryable `httpclient` failure; local inlining of the already-guarded generic idempotency helper is not an OAuth policy change. | T4 owns only the wrapper classification. Generic request repeatability remains with `httpclient`. |
| `internal/infra/httpclient/retry_test.go` | The one wrapper-classification assertion is T4 leaf parity; the richer per-attempt proof belongs in `attempt_authorization_test.go`. | T4 owns both narrow proof locations, with no duplicate production policy or test-only export. |

Reopen Go Ownership and Test Design if a future form cannot keep the exact
retry-to-instrumentation-to-authorizer-to-fixed-transport order, cloned
per-attempt mutation, parent/child generated-proof containment, or the two
leaf proof locations. Reopen Specification only if resource authority, retry,
header, cancellation, or downstream-response behavior changes. A future
provenance comparison may reuse this reconciliation only when these owners and
boundaries remain intact; it does not by itself reuse T4 proof or accept T4.

## Non-Go file map and cleanup

| Action | Owner and reason |
| --- | --- |
| add `docs/outbound-machine-authentication.md` | Manual operator/adopter contract, configuration, external checkpoints, failure classes, and HTTP/gRPC composition |
| change `README.md` | Selector example and link, under core marker |
| change `docs/repo-architecture.md` and `docs/project-structure-and-module-organization.md` | New adapter owner/import boundary and profile path |
| change `docs/configuration-source-policy.md` | Environment-only client-secret example and restart rotation boundary |
| change `env/config/local.yaml` | Empty non-secret section placeholders under core marker |
| change `env/.env.example` | One empty client-secret placeholder under core marker |
| change `scripts/init-module.sh` | Selector, pre-mutation validation, marker resolution, retention, lock, initialized README, tidy |
| change `scripts/ci/template-init-check.sh` | Independent cross-product, stripping, idempotence, module and build oracle |
| change `scripts/ci/runtime-image-build.sh` | Source-only runtime-image fixture's fixed `OUTBOUND_AUTH=none` input under the core marker; the generated-service early return remains unchanged |

There is no replaced OAuth code to delete. `x/oauth2`, discovery, refresh token,
provider SDK, proxy, shared cache, and feature-level attachment paths are not
added. Existing OIDC, resource HTTP/gRPC, health, inbound server, generated
contract, module, migration, container, and deployment files stay unchanged
except the explicitly marked shared HTTP policy/seam and documentation above.

## Design review

The original required complementary Go ownership panel remains historical
evidence for every unchanged runtime and public-surface decision. The
TD-008/TD-009 proof-carrier reopen then ran its own required focused panel.
Initial candidate SHA-256
`e3f98be022a95c7e6526fae4b18934ae7fda8693b8a40253fa3aca69414290c7`
received package/dependency PASS but execution-path and file-cohesion FAIL: CA
creation had two named owners, token-pool cleanup bypassed `Client.Close`, and
generated-client consumption had no exact file owner.

Root repair fixed one harness-owned PKI, one owner-mediated token-pool close,
and the marked `generated_client_test.go` subprocess branch. Fresh focused
re-review on candidate SHA-256
`563ed33fb22a869e8cfb84d38764d297cba92e10aee3c19283c99a225bf9889f`
returned:

- responsibility and execution-path ownership: PASS; both duplicate-owner
  findings were falsified and no normal, failure, lifecycle, cleanup, or proof
  path contradiction survived;
- package and dependency architecture: PASS; the parent test has no OAuth
  import, the generated child uses only public constructors and process fixture
  inputs, imports remain acyclic, and marker stripping preserves the existing
  bounded-client oracle;
- file cohesion and inverse file map: PASS; generated-client composition keeps
  one owner, one-file fixtures remain beside it, and the OAuth harness,
  `TestMain`, and `http_test.go` splits each retain one present reason.

The repaired panel results are compatible: one concrete package owns the
credential and deterministic concrete/retry carrier, the existing generated
composition file owns the generated child, one bootstrap path owns production
lifecycle, and every production or proof file has one non-overlapping reason
to change. Recording these receipts and changing `status` to `ready` do not
change the reviewed candidate.

Broader independent Technical Design review: PASS on candidate SHA-256
`1f0560d1470e07838401a2981f41832b043b5e90828a338e0ffa8d3b3f3508ff`.
No material counterexample survived the R1-R10, source-contract, cross-domain,
external-checkpoint, or phase-boundary review. The focused reopen changes no
system component, runtime mechanism, public Go surface, external checkpoint,
or R1-R10 behavior, so the shared trigger does not reopen that broader review.

Evidence reached the ready Specification and research, this design, current
repository authorities, and local module sources for `x/oauth2` v0.36.0 and
grpc-go v1.83.0. It did not reach a live provider, deployment, capacity,
Test Design, Planning, Implementation, or production validation. The design is
ready for Test Design to resume only TD-008 and TD-009, replace their obsolete
private-dial/TLS blocker with this carrier, refresh the design hash it consumes,
and run the focused QA review. Planning and Implementation remain ineligible
until that Test Design review reaches a movement-allowing verdict.
