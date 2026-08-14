# Selected services authenticate fixed outbound dependencies with short-lived OAuth bearer tokens

status: ready

Problem: a generated service can use bounded HTTP and gRPC clients, but it has
no optional outbound credential owner. An adopter must otherwise acquire,
cache, attach, redact, and renew OAuth tokens in feature code or create a second
transport path that can bypass the template's destination, lifecycle, health,
and telemetry rules.

## Scope and non-goals

In scope:

- a template-init selector with `none` and `oauth2-client-credentials`
  profiles;
- confidential-client OAuth 2.0 client credentials with explicit
  `client_secret_basic` and opaque bearer access tokens;
- immutable configuration and environment-only client-secret delivery;
- fixed token-endpoint trust, bounded response validation, process-local token
  reuse and renewal, caller cancellation, provider-failure isolation, and
  sanitized observability;
- authenticated outbound HTTP and gRPC, including gRPC control streams and the
  boundary for long-lived streams;
- generated and concrete feature-client consumption without feature-visible
  credentials or token protocol;
- selected-profile retention and `none`-profile stripping.

Non-goals:

- user delegation, on-behalf-of identity, authorization-code flows, refresh
  tokens, token exchange, token introspection, access-token JWT parsing, or
  feature authorization policy;
- runtime authorization-server discovery, signed metadata processing, dynamic
  token-endpoint rotation, or metadata refresh;
- `private_key_jwt`, OAuth mTLS client authentication, certificate-bound access
  tokens, DPoP, workload identity, direct SPIFFE identity, cloud credential
  chains, or a service-mesh, sidecar, broker, HSM, or certificate control
  plane;
- provider registration, choosing scopes, resources, or audiences, issuing or
  rotating credentials, network provisioning, deployment manifests, or live
  provider validation;
- dynamic secret reload, a persistent or cross-process token cache, replica
  coordination, proactive refresh, a circuit breaker, or configurable cache
  and retry tuning;
- resource-operation retry, reconnect, resume, idempotency, or reconciliation
  policy;
- changing inbound `AUTHN`, inbound bearer verification, existing HTTP/gRPC
  public contracts, or the resource server's authorization decisions.

The excluded authentication families reopen this specification only when a
named provider, resource server, or deployment authority requires one for the
accepted dependency. A current authoritative source reopens Research only when
it introduces a materially distinct candidate family, invalidates the Basic
interoperability floor, or changes the reusable responsibility boundary.

## Behavior and contract delta

### R1 — Profile selection is explicit and fail-closed

**Actor and trigger.** A template adopter initializes a service and either
omits `OUTBOUND_AUTH` or supplies it once.

**Rule.** Omission selects `none`. An explicitly empty value or a value other
than `none` or `oauth2-client-credentials` fails initialization without
changing the checkout. The OAuth profile is valid only when at least one
resource-client profile capable of consuming it is also retained: bounded
outbound HTTP, native gRPC, or both. Token acquisition itself remains HTTP, so
selecting OAuth retains the bounded HTTP capability even when the only resource
client is gRPC.

**Selected outcome.** `oauth2-client-credentials` retains the complete outbound
credential capability, its operator guidance, its runtime dependency, and the
existing bounded transport capabilities it requires. The selected value is
recorded with the other template-profile authorities. Repeated initialization
with the same inputs is behaviorally idempotent.

**`none` outcome.** The initialized service contains no outbound OAuth config,
environment placeholder, operator guidance, profile marker, runtime artifact,
or OAuth dependency attributable only to this capability. Existing HTTP or
gRPC clients remain only when another selected profile owns them. Module
cleanup removes dependencies made unreachable by stripping.

**Falsifier.** The full selected/`none` cross-product with outbound HTTP and
gRPC either produces the retained/stripped outcomes above or fails before
mutation for an impossible consumer combination; no generated service retains
an unresolved marker or dormant OAuth dependency.

### R2 — Exactly one provisioned dependency is one confidential OAuth client boundary

**Actor and precondition.** A generated service has selected the OAuth profile,
and a deployment provisions exactly one named outbound dependency under a
client registration authorized to act as the service itself.

**Rule.** The minimum has one process-local credential owner bound to exactly
one authorization-server registration, client ID, client-authentication method,
token endpoint, requested scope set, resource or provider-specific audience
when present, and resource authority. Zero configured dependencies or more than
one is invalid for the selected profile. A second dependency reopens
Specification rather than introducing a registry, shared cache, or generic
multi-provider policy. The access token is opaque and never establishes user
delegation or a different acting principal.

The dependency identity is a bounded, deployment-controlled identifier used to
select the authenticated client and correlate safe telemetry. It is not a URL,
credential, tenant, client ID, scope, resource, or audience, and request input
cannot select or alter any credential binding.

**Configuration boundary.** Non-secret values enter the repository's typed,
immutable configuration snapshot and are validated before serving. The client
secret enters only through the `APP__...` environment-secret channel; it has no
non-empty YAML value, default, command-line form, or ambient provider-SDK
source. Selected-profile guidance may expose only an empty environment
placeholder; `none` exposes none. All values are fixed for the process lifetime.
Changing a client secret, registration, endpoint, scope, resource, or audience
requires a controlled restart unless a separate dynamic-credential capability
is later accepted.

**Minimum input envelope.** A provisioned dependency supplies its bounded
identity, exact client ID and non-empty client secret, explicit
`client_secret_basic`, one exact token endpoint, zero or more exact scopes, and
at most one standard OAuth `resource` or one provider-contract-named `audience`.
Omitting scopes is valid only when the provider registration fixes an exact
least-privilege default; it never asks the template to accept an unknown
provider default. The dependency also selects the repository-owned
public/private target policy and a finite provider-acquisition budget. The
minimum accepts no arbitrary token parameters, auth-style probing, fallback
provider, runtime discovery, token/cache tuning, or retry tuning.

**Falsifier.** Missing, contradictory, duplicate, empty, unbounded, or
unsupported configuration fails before any endpoint receives a credential or
resource call.

### R3 — The Basic secret reaches the one admitted token endpoint only

**Provisioning rule.** Runtime accepts one deployment-fixed token URL and
performs no authorization-server discovery. When a provider publishes RFC 8414
metadata, the provider or deployment owner uses it during provisioning to
verify the exact issuer, token endpoint, client-credentials grant, and token
endpoint authentication method before supplying the pinned URL. Declared
authentication methods must include `client_secret_basic`; an omitted method
list has RFC 8414's `client_secret_basic` default. A provider that requires
runtime discovery or dynamic endpoint rotation reopens Specification.

**Endpoint rule.** Before a token request can carry a secret, its endpoint is
an admitted, fixed HTTPS authority with no userinfo or fragment, normal
certificate-chain and hostname verification, the selected public/private
address policy before and after DNS resolution, finite connection and response
bounds, no ambient proxy, and no redirects. The repository's current plaintext
private-HTTP mode cannot carry a client secret; a private token endpoint needs
an explicit private-HTTPS policy with the same TLS and DNS guarantees.

The Basic credential is sent once, only in the authorization header of the
admitted token request. It is never sent in the form body, URL, resource
request, redirect, telemetry, or diagnostic output. Token
and resource endpoints remain separate trust authorities even when their
hostnames match.

**Failure.** A scheme, authority, authentication-method, certificate,
hostname, address-class, proxy, redirect, size, or deadline violation fails
closed before a credential can cross to a different authority. A private route
requires an explicit compatible target policy; it never weakens TLS or DNS
checks.

**Falsifier.** A redirect, ambient proxy, disallowed resolved address, plaintext
private endpoint, or alternate authority receives no Basic header, client
secret, or bearer token.

### R4 — Token requests and responses have one portable contract

**Request.** The credential owner sends one bounded TLS `POST` with
`application/x-www-form-urlencoded`, `grant_type=client_credentials`, optional
space-delimited scopes, and only the configured standard `resource` or named
provider `audience` when applicable. Parameters are unique. Client
authentication is explicitly `client_secret_basic`; no failed probe or second
authentication style is attempted. Automatic token-endpoint retry is disabled.

**Accepted response.** A response is accepted only when all of these are true:

- it has exact status `200 OK`, declares `application/json` with only optional
  media-type parameters, and contains a bounded JSON OAuth token response;
- `access_token` is present and matches RFC 6750's `b64token` grammar: one or
  more ASCII letters, digits, or `-._~+/`, followed only by optional `=`
  padding, with no whitespace or control characters;
- `token_type` is present and equals `Bearer` case-insensitively;
- `expires_in` is a positive integer no greater than 3,600 seconds and leaves
  more than the fixed 10-second early-expiry margin at receipt;
- when `scope` is present, its case-sensitive set exactly matches the requested
  set; omission means the requested set, or the registration's exact accepted
  default when no scope was requested, was granted as RFC 6749 specifies;
- no non-empty refresh token is issued.

Unknown additive fields are ignored. The client neither parses access-token
claims nor infers scope, resource, audience, or lifetime from them. The token
response and any extra fields are never persisted. Missing token type does not
default to Bearer, missing or non-positive expiry does not create a
non-expiring token, and a refresh token is rejected as an unsupported response
rather than acquiring an unowned long-lived credential lifecycle.

**Rejected response.** Any status other than `200 OK`, wrong media type,
oversized or malformed response, OAuth error, absent or unsupported required
field, invalid bearer syntax, scope mismatch, unusable or overlong lifetime, or
refresh token produces one sanitized semantic failure and publishes no token.
Raw headers, body, error description, error URI, and parser text do not cross
the credential boundary.

**Falsifier.** Every invalid case above yields no resource credential and no
raw provider content in any returned error or observable; unknown harmless
fields alone do not make an otherwise valid response fail.

### R5 — Tokens are reused in memory and renewed on demand

**Lifecycle.** One provisioned dependency keeps at most one accepted access
token in process memory. A token with more than 10 seconds of remaining lifetime
is reusable for that dependency. At or inside the early-expiry margin it is
unusable and is never attached, even if the server's timestamp has not yet
passed. An absent or unusable token causes one new client-credentials grant on
demand. Renewal never uses a refresh token.

There is no eager token request merely because the process started and no token
traffic while the dependency is idle. Restart discards all cached token state.
No token is written to a file, database, external cache, support bundle, or
diagnostic surface.

Without introspection or a provider push signal, the process cannot learn
revocation before the resource rejects the token or the token expires. The
local exposure window is therefore bounded by the accepted remaining lifetime,
which is never more than one hour.

One logical resource operation fixes one accepted token. A later internal
attempt for that same operation may reuse it only while it has more than 10
seconds remaining. If it reaches the margin first, the operation ends with a
local `token unusable` failure before resource I/O and without acquiring a new
token. A new caller operation may acquire normally.

**Failure and recovery.** Failed acquisition publishes no replacement and
never makes an expired or rejected token usable. A later acquisition allowed by
R6 can recover normally. A downstream `401`, `403`, gRPC `Unauthenticated`, or
gRPC `PermissionDenied` does not transparently refresh and replay the resource
operation.

**Falsifier.** Repeated calls before early expiry reuse one token; the first new
operation at the margin obtains a new grant before any resource call; an
internal attempt crossing the margin performs neither resource I/O nor token
acquisition; missing expiry never becomes an indefinitely reusable token;
restart retains no token.

### R6 — Concurrent callers share work without sharing cancellation

**Success wave.** For one dependency and token generation, concurrent callers
that arrive without a reusable token cause at most one token-endpoint request.
Callers still waiting when that request succeeds may use the same accepted
token.

**Caller cancellation.** Each HTTP request, gRPC RPC, and gRPC stream creation
can stop waiting promptly when its own context is canceled or expires. A
canceled waiter receives its caller-context outcome and no later token or
provider result. Canceling one waiter does not cancel useful acquisition for
other waiters or poison the accepted token.

**Provider ownership.** Token-endpoint work has its own finite acquisition
budget and is also bounded by process shutdown. The feature caller's deadline
limits how long that caller waits; it does not become the lifetime authority for
shared provider work. Shutdown prevents new acquisition, cancels bounded
in-flight provider work, and leaves no continuing activity.

**Failure wave.** One failed acquisition result is shared, in sanitized form,
with callers waiting on that attempt. They do not become a serial queue of new
token requests. After the failed attempt completes, new callers fail fast for a
cooldown equal to at least the acquisition budget or one second. A syntactically
valid provider `Retry-After` on `429` or `503` may extend that base; bounded
positive jitter of at most 20 percent is added and the total is capped at one
hour. Then one later caller may start one recovery attempt. There is no
automatic retry inside either attempt. This process-local rule limits fast
failure and synchronized recovery without claiming fleet-wide throttling.

**Falsifier.** High concurrency yields one provider request on success and one
on failure; canceled followers return before the provider does; a canceled
caller does not cancel a token later consumed by a live caller; immediate
post-failure callers perform no provider I/O; the cooldown begins at failure
completion; a valid provider delay and bounded jitter are honored; a later
recovery wave again coalesces to one attempt.

### R7 — Failures and observability are semantic and non-disclosing

**Failure classes.** The outbound-auth boundary distinguishes at least:

| Class | Meaning and precedence |
| --- | --- |
| invalid configuration | Static input is absent, contradictory, or unsupported; no provider I/O occurred. |
| endpoint trust | Endpoint, TLS, DNS/address, proxy, redirect, or response-bound policy refused the endpoint. |
| caller canceled | The caller stopped waiting; its context outcome wins over a later provider result. |
| provider timeout | The independent token-acquisition budget expired. |
| provider unavailable | Transport or provider availability failed without a safe protocol rejection. |
| client rejected | The authorization server rejected the registered client or Basic credential. |
| grant rejected | Scope, resource, audience, grant, or authorization policy was rejected. |
| unsupported response | A success or error response could not satisfy R4 safely. |
| token unusable | The operation-fixed token entered the early-expiry margin before a later internal attempt; no resource I/O or token reacquisition occurred. |
| downstream unauthenticated | The resource server refused the bearer as unauthenticated; no transparent replay occurred. |
| downstream forbidden | The resource server authenticated but denied the operation; no transparent replay occurred. |

The first applicable local acquisition or attachment class is stable for
feature errors, logs, traces, metrics, and cached health. An upstream cause is
not wrapped into a surface that can render its raw text. HTTP status and OAuth
error codes may inform that internal classification, but provider bodies,
descriptions, URIs, headers, and endpoint identities never become
feature-visible details.

The two downstream classes are telemetry-only classifications. They do not
replace or generalize the concrete HTTP client's response semantics or gRPC's
`Unauthenticated` and `PermissionDenied` statuses. The concrete client remains
the owner of its safe downstream error surface; outbound auth records only the
bounded dependency identity and class, without raw resource-server details.

**Signals.** Operators can distinguish acquisition attempt/result/latency,
cache reuse versus acquisition, failure class, and the configured bounded
dependency identity. HTTP application rejections and gRPC application or
grpc-go control-RPC rejections are recorded once at their terminal result.
Metrics use only closed low-cardinality values. Traces or
access-controlled logs may correlate a dependency call using existing request
and trace identities, but never record client IDs, secrets, access or refresh
tokens, authorization metadata, issuer or endpoint URLs, scopes, resources,
audiences, raw provider content, or token claims. Silence is not the failure
signal: every failed wave remains observable under its sanitized class.

**Falsifier.** Seed each forbidden value into provider success and error
surfaces and assert it is absent from feature errors, readiness, logs, spans,
metrics, generated examples, and support-facing diagnostics while the stable
failure class remains present.

### R8 — Static validity affects startup; remote auth health does not affect liveness

**Startup.** Selecting the profile requires exactly one completely configured
dependency and a non-empty runtime secret. Zero, multiple, or incomplete
dependencies block serving without contacting the provider. A valid optional
dependency starts without token I/O; its first consumer activates the
on-demand path.

**Default criticality.** In the absence of a named service/SLO decision, an
outbound authenticated dependency is optional. Its provider state does not
participate in startup admission or readiness, and its failure affects only
operations that need that dependency. Naming the dependency critical reopens
this rule before Technical Design because it changes startup, degradation, and
readiness behavior.

**Readiness and liveness.** A later accepted critical-dependency policy may
consume only cached, bounded, sanitized auth state through the repository's
existing readiness aggregation; a readiness handler never performs token
acquisition or resource I/O. Liveness remains process-only under every
provider outcome. Draining turns readiness off under the existing process rule
before transport shutdown; outbound auth adds no alternate health endpoint.

**Shutdown.** On-demand acquisition has no lifecycle after bounded in-flight
work and owned transports are stopped. Telemetry remains available long enough
to record cancellation and cleanup under the repository's existing shutdown
budget.

**Falsifier.** A valid selected service starts and remains live while the token
endpoint is unavailable; optional readiness performs no provider I/O and stays
governed by existing dependencies; invalid static config never serves; drain
and shutdown leave no provider work running after their owned stage.

### R9 — HTTP feature code receives an authenticated fixed dependency client

**Trigger.** A concrete or generated HTTP client begins one logical resource
operation for the provisioned dependency.

**Rule.** Before the admitted resource request is sent, the outbound-auth
boundary obtains one R5 token and attaches exactly one
`Authorization: Bearer <token>` header below feature code. A caller-supplied
authorization header is rejected before resource I/O rather than overwritten
or combined. Feature code can select the named dependency and invoke its API;
it cannot receive secret config, a token source, token values, refresh controls,
or raw OAuth errors.

The operation preserves the existing fixed resource authority, TLS, DNS/address
policy, proxy refusal, redirect refusal, request and response bounds, caller
context, correlation hygiene, and standard telemetry. Token acquisition uses a
separate admitted token authority. A resource retry already authorized by the
concrete dependency reuses the operation's one attached token only while it has
more than 10 seconds remaining and cannot change authority. At or inside the
margin, that retry ends with `token unusable` before resource I/O and without
token reacquisition. Outbound auth adds no retry.

**Failure.** Failure to obtain a token prevents resource I/O. Resource `401`
and `403` remain the concrete HTTP client's downstream response semantics under
R7; the generic boundary neither invalidates and replays the current operation
nor translates permission denial into provider unavailability.

**Falsifier.** A generated and a concrete client both reach only the configured
resource authority with one bearer header; token failure yields zero resource
requests; internal resource retry does not multiply token acquisition; supplied
authorization or a redirect cannot forward either credential.

### R10 — gRPC uses the same dependency credential on application and control streams

**Trigger.** A unary RPC or stream is created on the provisioned dependency's
existing gRPC client connection, including a client control stream such as the
standard health `Watch`.

**Rule.** Stream or RPC creation obtains one R5 token and supplies exactly one
lowercase `authorization` metadata value with the Bearer scheme. The credential
requires transport security and never replaces or weakens the connection's
transport credentials. Caller-supplied authorization metadata is rejected
before the RPC starts. The caller's context governs its wait under R6.

One concrete construction owns the raw connection credential, the application
wrapper, and terminal rejection observation. It rejects preconfigured competing
per-RPC credentials or observers, so callers cannot accidentally authenticate
application RPCs while leaving grpc-go control RPCs outside the same policy.

Application RPCs and client control streams on the same dependency use the same
credential binding only when registration, scopes, resource/audience,
client-authentication method, and failure semantics are identical. Similar
Bearer syntax alone is insufficient. Generated stubs and concrete clients
consume the authenticated connection without exposing OAuth behavior to
feature code.

Any transport-internal repeat of the same logical RPC reuses its operation-fixed
token only while it has more than 10 seconds remaining. At or inside the
margin, the repeat ends with `token unusable` before a new attempt and without
token reacquisition or replay.

**Long-lived streams.** The token authenticates stream creation. It is not
replaced in place after stream establishment. Whether a resource server lets an
established stream survive token expiry is that RPC's provider contract. If it
terminates the stream, reconnect/resume, replay, cursor, and idempotency belong
to the concrete client. A control stream that reconnects creates a new stream
and obtains a currently usable token under the same rules.

**Failure.** Acquisition failure prevents the RPC or stream from starting.
gRPC `Unauthenticated` and `PermissionDenied` remain their concrete downstream
statuses and do not cause generic transparent replay. Other RPC deadline,
cancellation, load-balancing, health, message-bound, correlation, and telemetry
behavior is unchanged.

**Falsifier.** Unary, application-stream, and control-stream creation over real
transport security each carry one bearer value; cancellation stops only that
call's wait; no credential is sent over an insecure connection; an established
stream receives no in-place metadata mutation; a reconnect obtains a currently
usable token.

## Invariants and edge cases

- Missing or ambiguous identity, authority, client authentication, token type,
  expiry, privilege binding, or transport security denies rather than guesses.
- A client secret and an access token are credentials; a client ID, scope,
  resource, audience, issuer, and endpoint are treated as sensitive operational
  configuration even when they are not authentication secrets.
- The authorization server issues the token, the resource server decides
  whether it authorizes an operation, and this client treats the token as
  opaque. No local claim parsing can override either authority.
- One token never crosses dependency, registration, environment, scope,
  resource/audience, or resource-authority boundaries.
- Caller cancellation, provider timeout, provider rejection, malformed
  response, downstream unauthenticated, and downstream forbidden remain
  distinguishable and do not retry each other.
- No selected behavior introduces a second outbound destination policy,
  synchronous health I/O, resource-operation retry, or feature-visible token
  lifecycle.
- Existing inbound `AUTHN`, HTTP and gRPC correlation rules, transport bounds,
  readiness caching, drain order, and liveness meaning are deliberately
  unchanged.

## Decisions, constraints, and authorities

### Leading-hypothesis disposition

The portable `oauth2-client-credentials` hypothesis is **kept and narrowed**.
RFC 6749 requires Basic support for clients issued a password, and RFC 8414
defaults omitted token-endpoint auth metadata to `client_secret_basic`; this is
the smallest portable interoperability floor. RFC 9700 does not invalidate it,
but recommends asymmetric client authentication and sender-constrained access
tokens where the authorization server, resource server, and deployment can
support them. Therefore this profile claims interoperability for ordinary
short-lived bearer tokens, not strongest-available deployment security or
provider-specific compatibility.

Provider support for `private_key_jwt`, OAuth mTLS, certificate-bound tokens, or
DPoP is a mandatory checkpoint, not a dormant flag. Selecting one changes
credential custody, token-endpoint proof, resource transport, or both and
reopens Specification before design. Token exchange, workload identity, and
SPIFFE additionally change credential authority or principal semantics and
remain separate capabilities.

The Go-maintained OAuth client-credentials implementation remains a suitable
protocol primitive, not the behavioral owner. Its current construction-context
lifetime, serialized failed acquisition, auth-style probing, permissive missing
expiry/token type behavior, and raw retrieval errors do not satisfy R3-R7 on
their own. Technical Design may use the primitive only within a realization
that preserves every rule above; this specification selects no package, type,
composition order, or synchronization mechanism.

### Current authorities

- RFC 6749 owns the confidential-client grant, Basic interoperability floor,
  token request/response fields, and no-refresh-token posture.
- RFC 6750 owns Bearer header use and syntax; RFC 8414 owns provisioning
  metadata and authentication-method semantics; RFC 9700 owns the current
  asymmetric, sender-constraint, least-privilege, and audience-restriction
  guidance.
- The current bounded HTTP and gRPC client contracts own fixed destinations,
  TLS and address policy, response/call bounds, proxy and redirect behavior,
  correlation, telemetry, resource retry ownership, and gRPC per-RPC
  credential consumption.
- The current config contract owns one typed immutable snapshot,
  environment-only secrets, unknown-key rejection, and startup validation.
- The current bootstrap and health contracts own startup admission, cached
  readiness, unconditional liveness, drain, shutdown, cleanup, and final
  telemetry.
- The current template initializer and lock own selector validation,
  profile stripping, dependency cleanup, idempotence, and generated-profile
  provenance.
- [Research synthesis](research/synthesis.md) owns the evidence record and its
  current limitations; it does not override this behavioral contract.

### External-owner inputs and checkpoints

| Input | Owner | Checkpoint and effect |
| --- | --- | --- |
| Authorization server, exact pinned token endpoint, client registration, tenant/environment, allowed grant, auth methods, rate limits, and error contract | Provider and security owner | Required before any provider-specific Specification approval. Published metadata is checked during provisioning, not fetched at runtime. Live conformance is required before that provider can be called deployment-ready. Basic absence or a mandatory stronger method reopens Specification. |
| Resource server and least-privilege scopes, standard `resource`, or provider `audience` | Resource API owner and authorization-server registration | Required before an adopting service accepts runtime configuration or claims authorization compatibility. A user-delegated or multi-resource principal reopens the grant boundary. |
| Client-secret issuance, delivery, rotation, revocation, overlap, and recovery | Deployment/security owner | Required before deployment. This minimum rotates by controlled restart; a demonstrated shorter rotation or revocation requirement reopens dynamic credential ownership. |
| Access-token lifetime and revocation exposure | Provider, resource-server, and security owners | The portable maximum is one hour and tighter issuance remains valid. Without introspection or a provider push signal, revocation is observed only at resource rejection or expiry; a shorter required window reopens Specification. |
| Workload identity, private key, certificate, or sender-constrained token support | Platform, provider, and resource-server owners | Evaluated before accepting Basic for a named deployment. Selecting any of them reopens the affected credential and transport behavior before Technical Design. |
| Public/private network route, DNS, egress proxy, trust roots, and allowlists | Deployment/network owner | Required before Technical Design fixes the compatible target policy and before live validation. A private token endpoint requires private HTTPS; a mandatory proxy cannot silently bypass the fixed-authority guarantee. |
| Dependency criticality, degradation, startup, and readiness contribution | Service business/SLO owner | Default is optional. A critical declaration reopens R8 before Technical Design; implementation may not invent eager admission or readiness behavior. |
| Replica count, synchronized startup/expiry risk, provider quota, latency, and live capacity | Deployment/provider owners with measurement | Required before accepting jitter, proactive refresh, shared state, or a capacity claim. Their absence does not justify those mechanisms in the minimum. |
| Long-lived stream authorization, reconnect, resume, and replay contract | Concrete gRPC API/resource owner | Required before claiming continuity beyond stream creation. Generic outbound auth never implies transparent long-stream reauthentication. |

No provider, resource server, deployment platform, or live capacity evidence is
named in the current boundary. The specification is ready only as a portable
template contract and makes no provider-specific or production-readiness claim.

## Success criteria and proof expectations

1. Profile behavior is complete. Scope: every selected/`none`, HTTP, and gRPC
   combination. Pass: R1 retention, stripping, invalid-combination, dependency
   cleanup, lock, repeatability, build, and generated-authority outcomes all
   hold. Fail: any dormant artifact, unresolved marker, or OAuth-only
   dependency survives `none`.
2. Configuration and trust fail before disclosure. Scope: exactly one static
   dependency, its provisioning evidence and pinned endpoint, TLS, DNS/address
   policy, redirects, and proxies. Pass: valid inputs admit one authority and
   every zero, multiple, incomplete, or invalid path yields no
   credential-bearing request to an unadmitted authority.
3. Token protocol behavior matches R4. Scope: a controlled authorization-server
   boundary. Pass: the exact Basic request is observed; only exact `200 OK`
   JSON responses with valid Bearer syntax and a usable lifetime of at most one
   hour publish one opaque bearer; every rejected response publishes none and
   leaks no provider content.
4. Lifecycle and concurrency match R5-R6. Scope: the one dependency owner under
   deterministic expiry, an internal attempt crossing the margin, concurrent
   success, concurrent failure, caller cancellation, fail-fast window,
   recovery, and shutdown. Pass: acquisition, resource-attempt, and caller
   outcomes equal the rules, and no expired token or work survives its
   boundary.
5. HTTP consumption matches R9. Scope: one generated and one concrete client
   against a fixed resource authority, including an authorized internal retry,
   supplied authorization, redirect, `401`, and `403`. Pass: bearer attachment,
   request counts, authority, retry, and downstream outcomes match exactly.
6. gRPC consumption matches R10. Scope: unary, application streaming, and a
   control stream over transport security, including canceled acquisition,
   stream expiry, reconnect, `Unauthenticated`, and `PermissionDenied`. Pass:
   metadata, call counts, stream lifetime, and outcomes match exactly.
7. Health and lifecycle remain repository-owned. Scope: valid optional config,
   invalid static config, provider outage, readiness reads, liveness, drain, and
   shutdown. Pass: R8 holds with no synchronous probe I/O or leaked provider
   work.
8. Disclosure proof covers every forbidden R7 value at error, log, trace,
   metric, readiness, example, and generated-profile sinks. Pass: only the
   closed dependency identity and semantic classes appear.
9. Existing inbound auth, bounded HTTP/gRPC transport, config, health,
   bootstrap, profile, correlation, telemetry, and generated-client contracts
   remain behaviorally unchanged outside the explicit delta.

These are proof boundaries, not a Test Design or implementation selection. No
provider compatibility, latency, throughput, quota, multi-replica, deployment,
or production claim is accepted without the matching external-owner evidence.

## Risks, assumptions, and reopen conditions

- **Bearer replay remains possible.** Safe boundary: short-lived,
  least-privilege, audience/resource-restricted tokens under TLS and strict
  disclosure controls, with an accepted lifetime of at most one hour. Without
  introspection or a provider push signal, revocation can remain unobserved for
  that remaining lifetime unless the resource rejects the token first. Reopen
  owner: Specification. Reopen when a named provider/resource supports or
  requires sender-constrained tokens, its risk owner rejects replayable bearer
  within that lifetime, or the required revocation window is shorter.
- **Basic is an interoperability floor, not the strongest method.** Safe
  boundary: a confidential client with a protected environment secret and a
  provider registration that explicitly accepts Basic. Reopen when metadata,
  registration, security policy, or deployment capability requires or supports
  an accepted asymmetric method for the named dependency.
- **The fixed 10-second early-expiry margin may not fit a provider.** Safe
  boundary: tokens arrive with more than 10 seconds of usable lifetime and
  clocks satisfy the provider contract. Reopen when provider evidence shows a
  shorter token lifetime or larger clock/network allowance is required; no
  adopter-facing tuning knob is added from conjecture.
- **Process-local coordination does not smooth replicas.** Safe boundary: no
  capacity or fleet claim and no evidence of provider pressure. Reopen when
  measured replica alignment or provider quotas falsify the one-attempt-per-
  process boundary.
- **Rotation requires restart.** Safe boundary: immutable environment-loaded
  config and an owner-controlled restart within the accepted revocation window.
  Reopen when the deployment owner supplies a stricter rotation/revocation
  window or proves restart overlap unavailable.
- **Optional dependency is a bounded assumption.** Safe boundary: operations
  not using the dependency remain useful and may serve while its auth provider
  fails. Reopen R8 when the service/SLO owner declares the dependency critical.
- **Long-lived stream continuity is unknown.** Safe boundary: authentication
  covers stream creation only. Reopen with the concrete RPC/resource contract
  before claiming survival, reconnect, or replay after token expiry.
- **Provider and deployment compatibility are unverified.** Safe boundary:
  portable template behavior only. Reopen the smallest Specification rule when
  a named provider or deployment contract conflicts; reopen Research only under
  the three candidate-boundary conditions in Scope.
- **The maintained primitive can drift.** Refresh its versioned cancellation,
  auth-style, response, expiry, caching, and error behavior before Technical
  Design selects a dependency and before implementation relies on it. Library
  drift that only changes realization stays in design; drift that changes a
  portable behavior or reusable responsibility reopens Specification or
  Research respectively.

## Specification review

The initial whole-artifact review found five blockers: runtime metadata
ambiguity, incomplete token-response bounds, retry behavior at the expiry
margin, mixed local/downstream error ownership, and dependency cardinality.
The current contract repairs all five. A focused independent re-review on
2026-08-11 found no surviving material finding and returned **PASS**. Its scope
was Specification behavior only; it provides no Technical Design, Test Design,
implementation, provider-conformance, deployment, or production evidence.
