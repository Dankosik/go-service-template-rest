# Test design — outbound OAuth client credentials

status: ready

Accepted authorities:

- [`spec.md`](spec.md), SHA-256 `8ed6157fdc283a0939535854150306fdc19fd51d75f52da7fbefeac3345f33fb`;
- [`design/overview.md`](design/overview.md), SHA-256 `687b6b551aec5b8ce6ece10e6cb49cf00c7a8eb36016bd16e22205a31287a597`.

[`research/synthesis.md`](research/synthesis.md), SHA-256
`7e7bc88805755d61c10a2bc8d5f2f98ee1f69788df6fd7cf0261bd2a9a7ee77a`,
is supporting evidence only. This plan fixes proof obligations, fixtures,
commands, cleanup, and claim limits for R1-R10 and the selected Go ownership.
It adds no behavior, production seam, provider compatibility, Planning order,
deployment action, or production-readiness claim.

## Proof-boundary decisions

- Config, strict parsing, failure vocabulary, cache state, and wave coordination
  use owning-package tests. Broader process tests cannot improve their oracle.
- Optional provider-outage locality composes two owners: the OAuth package
  drives one real private resolution to `provider_unavailable`, while bootstrap
  proves structurally that auth contributes no readiness participant and that
  health reads cannot call its fail-on-call collaborator. An exported token
  operation or an HTTP/gRPC adapter dependency is rejected: either would add a
  production seam or prove adapter behavior instead of R8 health ownership.
- Endpoint trust combines direct proof at `httpclient`'s pre-request and
  post-DNS policy owner with a recorded token transaction. A mock that only
  reports `Do` was called cannot prove where Basic credentials were allowed.
- HTTP retry attachment combines the real `httpclient` retry chain with the
  OAuth adapter through public concrete construction. TD-008/009 use the
  selected process-local fallback-CA, private-DNS, non-loopback TLS carrier;
  they replace no private dialer, TLS state, retry decision, authority check,
  response path, or exported production seam. The generated-client branch
  remains in the existing isolated generation/subprocess oracle. gRPC
  application, health, and stream attachment use real TLS grpc-go connections;
  the transparent-attempt case keeps the existing raw HTTP/2 peer because an
  ordinary gRPC server cannot force `REFUSED_STREAM` precisely.
- A package-private clock controls token expiry and failure-window boundaries.
  `testing/synctest` controls provider deadlines and shutdown budgets. Owned
  start/release/result channels control concurrency; elapsed time is only an
  outer hang diagnostic.
- Race and leak gates instrument the selected scenarios; neither is a scenario
  by itself. No fuzz target is selected because the bounded field grammar has a
  small exact invalid partition and no cheaper invariant than the table oracle.
- No Docker integration or process e2e suite is selected. The capability adds
  no datastore or external local dependency, and every portable behavior is
  observable at an owning package or bootstrap boundary. Live provider and
  deployed-path proof remain external below.
- There is no residual-risk acceptance. Every portable proof input is available
  from the accepted owners, including TD-008/009's process-local concrete HTTP
  carrier. Test and fixture code remain Implementation-owned.

## Deterministic fixture and cleanup contract

`internal/infra/oauth2clientcredentials/harness_test.go` owns the fixtures
shared by its client, provider, telemetry, HTTP, and gRPC tests: an atomic
movable clock; a scripted provider with request count, start/release gates, and
context observation; the one suite PKI; bounded TLS token and resource servers
on a bindable non-loopback private IPv4 address; a temporary private-DNS
resolver; a recording logger and manual OTel meter; and one canary corpus for
client ID, secret, token, endpoint, scope, resource, audience, headers, body,
parser text, and provider details. `goleak_test.go`'s package `TestMain`
installs that PKI once as the process fallback root before TLS proof. Values
that affect behavior come from the ready Specification and Design; hostnames,
token bytes, and dependency names are decision-neutral. No fixture is exported
or shared through a new cross-package test package.

Every acquired listener, DNS socket, server, response body, connection, owner,
idle pool, meter reader, goroutine gate, temporary CA file, child process, and
temporary generated checkout registers cleanup immediately. An OAuth test
releases held provider/resource work before `Client.Close`, lets that owner
close the token idle pool, closes the resource idle pool, shuts down servers
and DNS, restores `net.DefaultResolver`, and joins their result channels. The
generated-client parent owns its endpoints, DNS, CA file, child, and temporary
tree; the child owns response bodies, its OAuth owner, and its resource idle
pool. Generator cleanup remains the shell oracle's `trap`-owned temporary root.
Cleanup errors fail their owning test; no test relies on a host janitor or
retained credential. Process fallback roots are one-shot and intentionally die
with their OAuth or generated-child test binary.

## Proof obligations

| ID and source | Disposition and plausible wrong behavior | Controlled setup, trigger, and discriminating oracle | Boundary and executable command | Fixture/input and status | Proof owner, Planning constraint, and reopen owner |
| --- | --- | --- | --- | --- | --- |
| **TD-001** — R1/R2/R7, config and secret custody | New config/runtime/parity tables. Wrong behavior: zero, multiple, incomplete, contradictory, duplicate, empty, over-bound, or unsupported values reach construction; a secret is accepted from YAML/default/CLI/ambient state; config and runtime admit different values. | Enumerate the exact leaf/default/snapshot inventories and every Design bound, including environment-only `client_secret`. Load valid env canaries, non-empty file values, unknown leaves, resource/audience conflict, and hostile ambient OAuth/proxy variables. Invalid cases must return a key-only sanitized rule before transport construction; valid config must map byte-for-byte into runtime policy with provider calls zero. Run the shared corpus through config and runtime validation and require config never admits what runtime rejects. | Unit/composition: `go test -vet=off ./internal/config ./internal/infra/oauth2clientcredentials ./cmd/service/internal/bootstrap -run '^(TestOutboundAuthConfigContract|TestOutboundAuthSecretSourcePolicy|TestOutboundAuthConfigParity)$' -count=1` | Existing config source/snapshot harness plus synthetic bounded values; new test code only. | Config tests, OAuth `config_test.go`, and `startup_outbound_auth_test.go`; land before provider/adapters. Reopen Specification for a new source/value rule and Technical Design for a changed schema or mapping owner. |
| **TD-002** — R7, semantic errors and redaction | New vocabulary/error table. Wrong behavior: a class spelling or precedence drifts, a provider cause unwraps, safe caller cancellation is lost, or a forbidden value enters a returned/startup/close error. | Construct every eleven-class error with the canary corpus in underlying errors and fields. Require exact `FailureClassOf`, fixed safe text, no canary, and no raw `errors.Is/As` reachability. Only `caller_canceled` must preserve `context.Canceled` or `context.DeadlineExceeded`; downstream statuses remain unchanged and outside the auth error wrapper. | Unit: `go test -vet=off ./internal/infra/oauth2clientcredentials ./cmd/service/internal/bootstrap -run '^(TestFailureVocabularyAndPrecedence|TestAuthErrorsAreSanitized|TestOutboundAuthStartupAndCloseErrorsAreSanitized)$' -count=1` | Shared canary corpus and ordinary wrapped errors; no provider. | OAuth `vocabulary_test.go`/`errors_test.go` and bootstrap test; land with the failure owner. Reopen Specification for a new class/precedence and Technical Design if raw diagnostics gain an authorized owner. |
| **TD-003** — R3, endpoint admission before Basic disclosure | Strengthen `httpclient` target proof and add token-client composition proof. Wrong behavior: plaintext, userinfo/query/fragment, wrong suffix, private IP literal, public/private post-DNS mismatch, proxy, redirect, or per-request authority drift receives Basic credentials. | Table external/private HTTPS construction and direct pre/post-DNS address admission. Inspect the constructed token client for one connection, exact bounds, retry/instrumentation/propagation disabled, nil proxy, and redirect refusal. Pass an alternate-authority request through the actual authority transport with a recording inner transport; require `ErrTargetDenied`, inner calls zero, and no Basic/token canary. The admitted control records one exact endpoint request. | Unit/transport component: `go test -vet=off ./internal/infra/httpclient ./internal/infra/oauth2clientcredentials -run '^(TestPrivateHTTPSTargetPolicy|TestTokenEndpointHTTPPolicy|TestTokenEndpointAdmissionPreventsCredentialDisclosure)$' -count=1` | Existing `httpclient` authority/address patterns plus a same-package recording transport; no live DNS, proxy, or CA. | `httpclient/target_policy_test.go`, `config_test.go`, OAuth `provider_test.go`; land before token protocol proof. Reopen Specification for discovery/proxy/plaintext and Technical Design if target policy cannot express the selected route. |
| **TD-004** — R4/R7, exact grant and strict response | New real-transaction table. Wrong behavior: the secret enters URL/body, Basic escaping or form fields drift, a request retries, or an invalid/already-unusable response publishes a token/raw provider content. | Record one POST and independently decode Authorization and the form: exact escaped Basic client ID/secret; unique grant/scope/resource-or-audience fields; no other field; `Accept` and content type exact. For exact 200 responses partition media type including optional parameters, size, JSON object/trailing/duplicate/known-field types, token grammar/size, Bearer case, every `expires_in` spelling and bound, scope omission/exact/mismatch/duplicate, absent/empty/non-empty refresh token, and unknown additive fields. Every invalid 200 must yield `unsupported_response`, publish no token, make one request, and emit no canary. A gated body control captures the header-receipt clock, advances from just outside to inside the ten-second margin before parsing completes, and requires the same `unsupported_response`, one request, and no publication. Non-200 exact classes are TD-007. Only the valid controls publish one opaque token. | Package component: `go test -vet=off ./internal/infra/oauth2clientcredentials -run '^TestProviderGrantRequestAndStrictResponse$' -count=1` | Bounded TLS token server or scripted client with a header/body gate from the shared harness; Design-derived table values and canaries. | OAuth `provider_test.go`; land as one table with `provider.go`. Reopen Specification for another provider response/auth method and Technical Design if strict decoding cannot remain single-owner. |
| **TD-005** — R5, on-demand reuse and expiry | New deterministic state/adapter table. Wrong behavior: construction or idle time acquires, a reusable token is reacquired, a margin token is attached, an internal attempt reacquires, failed renewal serves an old token, or restart retains state. | On the movable clock require `New`/idle provider calls zero, first operation one, repeated operations before the margin still one, and the first new operation at the margin a second grant before resource I/O. Fix a token to one operation, cross the margin at an attempt gate, and require `token_unusable`, resource calls unchanged, and provider calls unchanged. Failed acquisition publishes no token; a separately constructed owner starts empty. | Unit/component: `go test -vet=off ./internal/infra/oauth2clientcredentials -run '^(TestTokenReuseRenewalAndRestart|TestOperationTokenCannotRenewAcrossExpiryMargin)$' -count=1` | Shared clock, scripted provider, and counted resource transport; no sleeps. | OAuth `client_test.go`, `http_test.go`, and `grpc_test.go` use one owner fixture but keep adapter assertions local. Land owner lifecycle before transport rows. Reopen Specification for another expiry/margin rule and Technical Design if the private clock seam cannot observe publication/attempt time. |
| **TD-006** — R6, success/failure waves and caller cancellation | New event-gated concurrency table plus race repetition. Wrong behavior: a wave makes multiple grants, followers serialize after failure, one canceled waiter cancels useful work or later consumes its result, fail-fast callers perform I/O, or recovery does not coalesce. | Hold the provider after it signals entry; start a fixed caller wave and require provider count one. Cancel leader and follower subsets and require their caller-context result before provider release while live callers receive the one accepted token. Repeat with one seeded failure: all waiting callers receive the same class, immediate post-failure callers return with provider count one, and the cooldown begins when the attempt finishes rather than when it starts. Advance exactly to the completion-relative boundary and require one recovery wave. Channels own entry/release/result; the movable clock owns the boundary. | Unit/race: `go test -vet=off ./internal/infra/oauth2clientcredentials -run '^(TestAcquisitionWavesPreserveCallerCancellation|TestProviderFailureCooldownStartsWhenTheAttemptFinishes)$' -count=1`; `go test -vet=off -race ./internal/infra/oauth2clientcredentials -run '^(TestAcquisitionWavesPreserveCallerCancellation|TestProviderFailureCooldownStartsWhenTheAttemptFinishes)$' -count=10` | Shared scripted provider/clock; caller count is a test input, not a capacity claim. | OAuth `client_test.go`; land with wave state. Reopen Specification for changed sharing/failure-window behavior and Technical Design if one result channel cannot preserve the oracle. |
| **TD-007** — R6/R7, provider lifetime, timeout, and failure classification | New time-controlled provider table. Wrong behavior: the initiating caller owns provider work, provider timeout is returned as caller cancellation, exact trust/client/grant/availability/response failures collapse, a fast failure retries, provider throttling is ignored, recovery synchronizes, or raw causes escape. | In `synctest`, start one provider request and cancel its initiating caller; require provider context live and a second caller able to consume success. Advance the independent acquisition timeout and require provider context canceled, one request, `provider_timeout`, and no token. The exact non-200/fault map is: TLS/target/proxy/address-policy failures and final 3xx redirects -> `endpoint_trust`; network, 408, 429, and 5xx -> `provider_unavailable`; 401/403 or OAuth `invalid_client` -> `client_rejected`; other recognized grant errors and other 4xx -> `grant_rejected`; non-200 2xx and any otherwise unrecognized surfaced status -> `unsupported_response`; invalid 200 remains `unsupported_response` in TD-004. A valid `Retry-After` on 429/503 extends the completion-relative base; injected maximum jitter proves the 20-percent bound and one-hour cap. Invalid or overflow headers are ignored. Every HTTP response case makes exactly one request; pre-request trust failures make zero credential-bearing requests. No case retries, publishes a token, or exposes a raw canary. | Unit/component: `go test -vet=off ./internal/infra/oauth2clientcredentials ./internal/infra/httpclient -run '^(TestProviderWorkOutlivesCallersWithinItsBudget|TestProviderFailureClassificationIsClosed|TestProviderFailureCooldownIsBoundedAndJittered|TestRetryHonorsRetryAfter)$' -count=1` | `synctest`, scripted responses, provider-context observation, movable clock/jitter, canary corpus. | OAuth `client_test.go` and `provider_test.go`, existing `httpclient` parser; land with timeout/classification. Reopen Specification for retry or class changes and Technical Design if provider context cannot stay process-owned. |
| **TD-008** — R2/R6/R9, HTTP operation attachment and downstream ownership | New mandatory concrete component and generated-client subprocess proof. Wrong behavior: resource-authority mismatch constructs, caller Authorization is overwritten, caller cancellation detaches, token failure reaches the resource, the header is duplicated/forwarded, feature code needs token API, or 401/403 is translated/replayed. | In the OAuth test binary, install the suite CA once as forced fallback roots and map fixed token/resource hostnames through temporary private DNS to one bindable non-loopback private IPv4 address. Construct the token owner, the resource `*httpclient.Client` with public `httpclient.New` and `PrivateHTTPS`, and the fixed `NewHTTPClient(*Client, *httpclient.Client)` adapter; replace no transport state. Constructor authority mismatch must return `invalid_configuration` before acquisition/I/O. The admitted TLS resource records exact authority and one bearer after the real target, TLS, response-bound, and response paths. Caller Authorization case variants fail before acquisition/I/O. During gated acquisition, caller cancellation/deadline returns its context result with resource count zero while a live waiter consumes the same provider success. Acquisition failure yields resource zero. Resource 401/403 remain unchanged with one request/no cache invalidation or replay; an alternate authority and a controlled cross-authority redirect receive no credential. Separately, the existing generated-client parent starts equivalent controlled endpoints/DNS, passes only fixture addresses and a temporary CA file to a generated child process, and the child installs its one fallback root/resolver, constructs both public clients, calls `NewHTTPClient`, injects the Doer through `WithHTTPClient`, and records one admitted request. | Concrete component plus generated subprocess: `go test -vet=off ./internal/infra/oauth2clientcredentials ./internal/infra/httpclient -run '^(TestHTTPClientResourceAuthorityIsFixed|TestHTTPClientAttachesOneOperationToken|TestHTTPClientRejectsCallerAuthorization|TestHTTPClientCallerCancellationStopsOnlyItsWait|TestHTTPClientPreservesDownstreamAuthResponses|TestGeneratedClientUsesAuthenticatedDoer)$' -count=1` | Available from ready Design: pinned Go 1.26.5 forced fallback roots; one harness-owned CA and hostname certificates; bindable non-loopback private IPv4 discovery; temporary private DNS; bounded token/resource servers; existing generated-client subprocess owner. OAuth and generated parent/child cleanup follows the contract above. | OAuth `harness_test.go`, `goleak_test.go`, and `http_test.go` own concrete assertions; existing `httpclient/generated_client_test.go` owns generated consumption. Preserve the public concrete constructor, real `httpclient` authority/TLS/response path, generated subprocess boundary, and cleanup owners. Reopen Go Ownership/Test Design if forced fallback roots disappear, a supported runner lacks the required private address, or process-global trust/resolver use prevents binary-local isolation; reopen Specification for changed R9 behavior. |
| **TD-009** — R5/R9, HTTP retries keep one operation token | New mandatory `httpclient` leaf proof plus composed OAuth proof. Wrong behavior: auth runs outside attempts, retries reacquire/change token, an expired fixed token reaches attempt two, auth failure is retried, or instrumentation observes the bearer. | `httpclient/attempt_authorization_test.go` independently drives the real retry chain with a synthetic authorizer and requires one invocation inside each attempt, unchanged ordinary retry eligibility/count, non-retryable authorizer failure, and no bearer canary in OTel. Through TD-008's same process-local CA/DNS/non-loopback TLS carrier and fixed concrete constructor, gate the real resource attempts and require one provider grant, the same bearer on every permitted attempt, and the existing retry count and response handling. Move the clock inside the margin after attempt one and before attempt two: exact `token_unusable`, one resource attempt, one provider request, no second resource I/O, retry, or reacquisition, and no bearer canary in generic HTTP telemetry. | Leaf and composed component: `go test -vet=off ./internal/infra/httpclient -run '^TestAttemptAuthorizationPreservesRetryPolicy$' -count=1`; `go test -vet=off ./internal/infra/oauth2clientcredentials -run '^TestHTTPRetryFixesOneTokenAndStopsAtMargin$' -count=1` | Available: existing `httpclient` retry harness plus TD-008's harness-owned PKI, private DNS, non-loopback TLS resource, movable clock, provider/resource counters, attempt gates, and cleanup owners. No alternate Doer, dialer, TLS config, retry path, or response path is substituted. | `httpclient/attempt_authorization_test.go` and `retry_test.go` own leaf parity; OAuth `http_test.go` owns fixed-token composition and remains the sole production-caller proof. Land the shared seam, leaf proof, OAuth caller, and composed proof together. Reopen Go Ownership/Test Design under TD-008's carrier conditions; reopen Specification only for changed resource retry semantics. |
| **TD-010** — R2/R6/R10, gRPC application RPC and stream creation | New real-TLS unary/stream table. Wrong behavior: wrong resource authority receives metadata, metadata is missing/duplicated, insecure transport sends it, construction permits split application/control auth, caller metadata/call credentials add a second source, caller cancellation detaches, acquisition failure starts the RPC, or downstream statuses are wrapped/replayed. | Construct only through `NewGRPCClient`; reject preconfigured `PerRPCCredentials` or terminal observer before dialing. Directly exercise admitted and mismatched HTTPS request URIs: admitted returns one bearer; mismatch returns exact `invalid_configuration`, no metadata, and no handler call. Run unary and each application stream cardinality through a TLS server and require one lowercase authorization value plus generated-client compatibility. Caller metadata case variants and every per-call credential option fail before acquisition/handler. During gated application-stream acquisition, cancellation/deadline returns its caller-context result with handler zero while a live operation consumes the same provider success. Acquisition failure leaves handler zero; insecure construction refuses the credential. `Unauthenticated` and `PermissionDenied` return unchanged with one handler call, one terminal metric, no new grant, and no replay. | TLS component: `go test -vet=off ./internal/infra/oauth2clientcredentials -run '^(TestGRPCResourceAuthorityIsFixed|TestGRPCApplicationCallsAttachOneToken|TestGRPCRejectsCompetingAuthorization|TestGRPCCallerCancellationStopsApplicationWait|TestGRPCRequiresTransportSecurity|TestGRPCPreservesDownstreamAuthStatus)$' -count=1` | Reuse repository grpc test descriptors/TLS material locally; counted provider and handler, tracked streams. | OAuth `grpc_test.go`; land with `grpc.go` and the optional `grpcclient` terminal observer. Reopen Specification for another metadata/status rule and Technical Design if connection-wide credentials cannot compose with the application wrapper. |
| **TD-011** — R5/R10, grpc-go transport attempts | New raw-peer attempt falsifier. Wrong behavior: a transparent attempt reacquires, changes token, starts inside the margin, returns grpc-go wrapper text, or forwards caller cancellation into shared provider work. | Extend the raw HTTP/2 peer pattern to record metadata and force one `REFUSED_STREAM`. With a reusable token require two attempts carrying the same bearer and one provider grant. Gate after attempt one, advance into the margin, and require no second HEADERS frame, no grant, and local `token_unusable`. Cancel a caller waiting for acquisition and require its context result while a live call later uses provider success. | Raw TLS/HTTP2 component: `go test -vet=off ./internal/infra/oauth2clientcredentials -run '^TestGRPCAttemptsUseOneOperationToken$' -count=1` | Package-local TLS raw peer derived from the existing grpcclient fixture, attempt gates, movable clock. | OAuth `grpc_test.go`; do not alter grpcclient's propagation peer. Reopen Technical Design if pinned grpc-go attempt or credential invocation changes; Specification if attempts may renew. |
| **TD-012** — R6/R8/R10, health control streams and long-lived streams | New control/application stream lifecycle table. Wrong behavior: Health/Watch bypasses auth or terminal rejection telemetry, control cancellation detaches, reconnect reuses an unusable token, an established stream mutates metadata/refreshes in place, or generic auth claims reconnect/resume. | A TLS standard health server records each Watch's initial metadata. Gate acquisition for one Watch, cancel its stream context, and require the caller-context result, zero handler entry, and provider work still consumable by a live waiter. Connect normally and require one bearer on the first Watch; terminate it, advance into the margin, reconnect, and require a new stream with a newly acquired usable bearer. Terminate a Watch with fixed `Unauthenticated` and require one closed gRPC rejection point. Establish an application stream, advance past expiry, exchange another message, then have the fixture terminate with fixed `Unauthenticated`; require provider/request-metadata counts unchanged and the exact status unchanged. The successful exchange is a no-reauth control, not a continuity claim. | TLS control/stream component: `go test -vet=off ./internal/infra/oauth2clientcredentials -run '^(TestGRPCControlStreamCancellationStopsOnlyItsWait|TestGRPCHealthWatchUsesConnectionCredentialOnReconnect|TestGRPCControlRejectionIsMeasured|TestLongLivedStreamDoesNotReauthenticateInPlace)$' -count=1` | Standard grpc health server, application stream descriptor, manual meter, metadata/provider counters, movable clock. | OAuth `grpc_test.go` and `telemetry_test.go`; land with control-stream branch. Reopen the concrete RPC owner before any continuity/reconnect/replay claim; Technical Design on grpc-go control-path drift. |
| **TD-013** — R8, startup, cached health, and optional degradation | New composed owner/health table. Wrong behavior: valid startup acquires, invalid config serves, provider outage changes readiness/liveness or registers a probe, readiness performs provider/resource I/O, or drain gets a second health rule. | In the OAuth package, use the package-private resolution path with a scripted provider and recording meter to drive one post-start operation to exact `provider_unavailable`; require only its closed operation error and provider attempt/result signals. In bootstrap, construct valid selected config without provider I/O, reject invalid/incomplete static config before listener mutation, and capture the current readiness/liveness/probe inventory with a fail-on-call auth collaborator. Require no auth readiness participant, unchanged liveness, zero provider/resource calls across repeated health reads, and the existing single drain transition. The two package observables compose because bootstrap has no provider-state input; no exported token operation or resource adapter is introduced for proof. | Owning-package plus bootstrap component: `go test -vet=off ./internal/infra/oauth2clientcredentials ./cmd/service/internal/bootstrap -run '^(TestOutboundAuthProviderOutageIsOperationLocal|TestOutboundAuthStartupIsLocalOnly|TestOutboundAuthOutageDoesNotChangeHealth|TestOutboundAuthInvalidConfigFailsBeforeServing)$' -count=1` | Existing OAuth scripted provider/private resolution and recording meter; existing bootstrap event/health fakes plus a fail-on-call auth collaborator. No live endpoint, HTTP/gRPC resource adapter, or new production/test seam. | OAuth `client_test.go`/`telemetry_test.go` own the real operation-local outage; `startup_outbound_auth_test.go` and existing lifecycle/health tests own startup and absence from health. Land the operation-local proof with the client owner before bootstrap composition; Planning may split the row only across those two dependency-ordered owners. Reopen R8 when the service/SLO owner declares criticality; Technical Design if the client cannot expose the private owning-package observable or bootstrap cannot omit a probe. |
| **TD-014** — R6/R8, retirement and ordered shutdown | New owner-close table, bootstrap lifecycle extension, race and leak gates. Wrong behavior: Close admits work, leaves provider I/O/goroutines, closes pools twice observably, loses a deadline, returns the wrong class, runs before admitted work/background users stop, or joins a close error zero/two times. | Hold one provider wave, call Close under `synctest`, require new acquisition refusal with `provider_unavailable`, provider-context cancellation, join before return, token reference cleared, idle pool closed once, and repeated Close no-op. Expire the shutdown context before the wave joins and require exact `provider_unavailable`, fixed safe text, and no raw cause. Extend the event recorder for drain/server/background/resource clients/auth owner/dependencies/telemetry; require Design order and exactly-once propagation of that same class for normal and partial-start paths. | Lifecycle/race: `go test -vet=off ./internal/infra/oauth2clientcredentials ./cmd/service/internal/bootstrap -run '^(TestClientCloseRetiresAndJoinsAcquisition|TestOutboundAuthRuntimeCloseOrder|TestOutboundAuthCloseFailureIsJoinedOnce)$' -count=1`; `go test -vet=off -race ./internal/infra/oauth2clientcredentials ./cmd/service/internal/bootstrap -run '^(TestClientCloseRetiresAndJoinsAcquisition|TestOutboundAuthRuntimeCloseOrder)$' -count=10` | `synctest`, provider gates, close counters, existing bootstrap event recorder; `goleak.VerifyTestMain` in the OAuth package. | OAuth `client_test.go`/`goleak_test.go`, bootstrap startup/run lifecycle tests; land with construction and every cleanup path. Reopen Specification for shutdown semantics and Technical Design if lifecycle order/owner changes. |
| **TD-015** — R7, bounded telemetry and disclosure | New exact signal matrix and degradation test. Wrong behavior: a required operation or grpc-go control RPC is silent, a logical RPC is counted twice, labels become input-driven, a forbidden value reaches metrics/logs/spans/readiness, provider requests inherit caller propagation, or telemetry failure changes auth. | Drive cache hit, acquisition success, each acquisition failure class, HTTP/gRPC application and control downstream rejection, and cancellation/retirement. Collect the four exact instruments and require only closed dependency/source/result/class/transport attributes, exact counts, and provider duration per attempt. Seed every canary through success/error surfaces and require absence from metric attributes, log records, returned/readiness errors, and spans; token requests must carry no trace/baggage/request ID and create no generic HTTP span. Failing instrument construction must preserve authentication and emit at most one closed degradation warning. | Component: `go test -vet=off ./internal/infra/oauth2clientcredentials -run '^(TestOutboundAuthTelemetryIsCompleteAndBounded|TestGRPCControlRejectionIsMeasured|TestOutboundAuthForbiddenValuesNeverReachSignals|TestOutboundAuthTelemetryFailureDegradesToNoop)$' -count=1` | Shared canary corpus, manual meter, recording tracer/logger, scripted outcomes. | OAuth `vocabulary_test.go` and `telemetry_test.go`; land after all reachable paths exist so the matrix is exhaustive. Reopen Specification for a new operator question and Technical Design if a required safe signal lacks an owner. |
| **TD-016** — R1, selected/none profile generation and stripping | Strengthen the independent template-init oracle. Wrong behavior: omission differs from explicit `none`; invalid selection mutates; selected output loses the token HTTP owner/chosen adapter; `none` retains config/secret/docs/code/tests/markers/direct imports; union-owned HTTP policy strips too early; lock/output disagree; repeat changes bytes. | Generate OAuth crossed with HTTP-only, gRPC-only, both, and invalid no-consumer, plus omitted and explicit-`none` controls for each transport. Require omitted and explicit `none` to produce byte-identical generated trees and identical `outbound_auth = "none"` lock state. Before/after snapshots prove explicit empty/unknown/impossible values fail pre-mutation and equal repeat is byte-identical. Independent inventories cover config/env/docs/package/adapters/shared seam/union option/lock/markers; selected examples contain only the empty environment secret placeholder and no usable endpoint/client/secret/token canary, while `none` contains none. Inbound OIDC controls prove union ownership. Each output runs tests/build, both tidy paths, unresolved-marker scan, and exact module rules: no direct OAuth requirement/import anywhere; HTTP-only outputs have no reachable OAuth module; a grpc-go-owned transitive module may remain only with identical attribution in selected and `none` gRPC controls. | Profile/structural: `TEMPLATE_INIT_PROFILE=outbound-auth make template-init-check`; `make project-structure-check`; `make mod-tidy-check` | Existing checkout-copy/snapshot/oracle helpers; no credential or network. Script `trap` removes every temporary checkout. | `scripts/init-module.sh` and `scripts/ci/template-init-check.sh`; one profile acceptance unit after package/adapters. Reopen Specification for selector semantics and Technical Design/Go Ownership for marker, inventory, or dependency ownership changes. |

## Aggregate gates and claim limits

Focused scenario commands above are the behavioral oracles. After every
scenario passes, the implementation candidate also runs:

```bash
go test -vet=off ./internal/config ./internal/infra/httpclient ./internal/infra/grpcclient ./internal/infra/oauth2clientcredentials ./cmd/service/internal/bootstrap
go test -vet=off -race ./internal/infra/httpclient ./internal/infra/grpcclient ./internal/infra/oauth2clientcredentials ./cmd/service/internal/bootstrap
TEMPLATE_INIT_PROFILE=outbound-auth make template-init-check
make project-structure-check
make mod-tidy-check
make lint
make test
```

These commands prove only the current tree and generated local outputs they
exercise. `make test` is repository breadth, not a substitute for a scenario.
Race is mandatory for wave, attempt, stream, and shutdown state. Lint owns vet
and import-direction proof. `mod-tidy-check` and the profile oracle prove module
and generated-tree shape, not provider behavior. Existing inbound OIDC/JWT,
ordinary HTTP/gRPC, config, bootstrap, health, correlation, and telemetry tests
remain regression gates; they do not prove outbound OAuth semantics.

## External proof boundaries

| External claim | Required owner/input and later proof | Limit of this Test Design |
| --- | --- | --- |
| Named provider compatibility | Provider/security owner supplies registration, exact endpoint and metadata receipt, Basic/grant support, error/rate contract, scopes/resource/audience, and credentials; a separately authorized live conformance run observes the exact request and accepted/rejected behavior. | Local strict server proof establishes the portable client contract only. It cannot certify any provider. |
| Deployment and network path | Deployment/network owner supplies DNS, public/private route, system trust roots, egress/no-proxy compatibility, allowlists, and target runtime; deployed-path validation observes TLS authority and credential/resource delivery. | Local address-policy and TLS fixtures do not prove real DNS, CA installation, firewall, proxy, or deployment manifests. |
| Credential issuance and rotation | Deployment/security owner supplies issuance, custody, revocation, overlap, restart procedure, and recovery evidence. | Tests prove environment-only immutable custody and restart-discarded state, not rotation timing or operational success. |
| Dependency criticality | Service business/SLO owner either keeps the fixed optional default or reopens R8 before design. A later critical policy needs its own readiness/degradation proof. | TD-013 proves only the accepted optional behavior; it does not authorize or simulate a critical dependency. |
| Fleet capacity | Deployment/provider owners supply replica count, quotas, synchronized-start/expiry risk, latency, and representative workload measurement. | Caller-count race tests prove coordination correctness only. They establish no throughput, latency, quota, replica, mutex-cost, or fleet-capacity claim. |
| Long-lived stream continuity | Concrete gRPC API/resource owner supplies expiry enforcement, reconnect, resume, replay, cursor, and idempotency contract and proof. | TD-012 proves authentication at stream creation, new auth on reconnect, and no in-place mutation. It makes no continuity claim. |

No external input above is required for deterministic portable implementation
completion. Its absence prohibits only the matching provider, deployed-path,
rotation, criticality, capacity, or stream-continuity claim. A named external
contract that conflicts with R1-R10 reopens the exact Specification or Design
owner before Planning may place such work.

## Current proof gaps and fail-before signals

The current tree contains no outbound-auth package, config section, transport
adapters, bootstrap stage, or profile selector. Missing future test files are
therefore proof gaps, not behavioral fail-before evidence. The nearest current
falsifiers are narrower:

- current `httpclient` proves fixed authority, public/private-HTTP address
  policy, redirect refusal, bounds, retry, correlation, bindable non-loopback
  private addressing, temporary private DNS, and generated-client subprocess
  composition but has no `PrivateHTTPS` class or per-attempt authorization seam;
- current `grpcclient` proves connection-wide credentials, control Watch,
  transparent retry, TLS, and propagation, but has no operation-fixed token
  wrapper or outbound credential owner;
- current OIDC/JWT tests demonstrate reusable deterministic patterns for
  provider/caller cancellation, redaction, metrics, and lifecycle, but their
  inbound behavior and readiness policy are not outbound proof;
- current config, bootstrap, and template-init tests are reusable owners, not
  evidence that R1-R10 already hold.

No live provider, deployment, network, credential-rotation, criticality,
capacity, or production evidence was inspected or produced.

## TD-008/TD-009 carrier feasibility

The ready Go design keeps the fixed
`NewHTTPClient(*Client, *httpclient.Client)` constructor and requires no private
resource-client seam. The OAuth test binary installs one harness-owned CA with
`x509.SetFallbackRoots` under forced fallback-root mode, resolves fixed
certificate hostnames through temporary private DNS to a bindable non-loopback
private IPv4 address, and constructs both concrete clients through public
production constructors. `httpclient.New` retains nil custom roots and owns the
real retry, generic OTel, fixed-authority, private-address-at-dial, TLS
hostname/chain, response-bound, redirect, and response paths. The fixture
therefore changes only process trust and DNS inputs; it bypasses none of the
claimed production path.

The pinned Go 1.26.5 `crypto/x509` contract permits one forced fallback pool
per process. The current repository's
`TestServiceToServiceHTTPCorrelationAndCancellation` proves the required
non-loopback private-address/private-DNS carrier, and
`TestGeneratedClientComposition` proves the existing in-module generation and
child-test owner. Both anchors passed together on this candidate with:

```bash
go test -vet=off ./internal/infra/httpclient -run '^(TestServiceToServiceHTTPCorrelationAndCancellation|TestGeneratedClientComposition)$' -count=1
```

That pass establishes fixture and owner feasibility only; it is not behavioral
evidence for unimplemented OAuth, `PrivateHTTPS`, attempt authorization, or
TD-008/TD-009. Implementation must add the exact row-owned tests and preserve
their process-local cleanup. Reopen Go Ownership/Test Design only if the pinned
toolchain removes forced fallback roots, a supported runner cannot supply the
already-required private address, or another process-global trust/resolver
consumer prevents package-test isolation. Such evidence never authorizes a
weaker production target policy or an alternate constructor.

## Review evidence

Independent whole-candidate QA review at SHA-256
`307aa50348a67ae715425b6d006c4195d9b6f0297ebe1960f8df11661a910d99`
returned `FAIL` with six blockers: infeasible HTTP component fixtures, missing
pre-publication expiry proof, adapter-path cancellation gaps, missing gRPC
resource-authority proof, ambiguous failure-class oracles, and no omitted-versus-
`none` profile control.

Root repair added exact falsifiers for the latter five and replaced the
infeasible TD-008/009 fixture claims with the mandatory upstream blocker above.
The first focused re-review confirmed those five repairs and found two wording
gaps: non-200 status coverage was not exhaustive, and the proof-boundary summary
still implied that every input was available. Root repair closed both. Terminal
focused QA review at SHA-256
`e98960b095cb5d193021ef9e4ca67310718061510989d9b836240ca2cd243c04`
returned `FAIL` solely because the fixed design still makes TD-008/009
infeasible. It confirmed that no other finding survives and that Planning
remains ineligible until Go Code / Ownership Design supplies the seam and a
focused QA review returns a movement-allowing verdict. No test, fixture,
production code, Planning artifact, deployment, or live validation was created
or performed.

Fresh independent whole-artifact QA review on the current pre-receipt candidate
at SHA-256
`91be0381330dbdcbf2985f327265b46d4354abe8c856c70000daf3c72ec78290`
returned `FAIL` on the same sole blocker. It independently confirmed the
TD-008/009 carrier infeasibility against the current `httpclient` private
dial/TLS ownership and found no other gap across R1-R10, preserved regressions,
failure, concurrency, lifecycle, disclosure, profile, command, cleanup, or
bidirectional closure. Planning remains ineligible. This receipt-only update
does not change the reviewed proof contract.

The ready Go design at SHA-256
`687b6b551aec5b8ce6ece10e6cb49cf00c7a8eb36016bd16e22205a31287a597`
reopened only that carrier. Root Test Design repair replaced the obsolete
private-dial/TLS blocker in TD-008/TD-009 with its process-local fallback-CA,
private-DNS, non-loopback TLS carrier and preserved the fixed constructor, real
`httpclient` retry/authorization/authority/TLS/response path,
generated-client subprocess owner, cleanup ownership, and every R1-R10
disposition. That repair produced the fixed candidate reviewed below.

Fresh focused independent QA review at SHA-256
`17206bd5531bfd16dca75c7041242fe664e0efe5942e66eae105f635626815a4`
returned `PASS` with no surviving finding. Its boundary was TD-008, TD-009,
the proof-boundary summary, deterministic fixture and cleanup contract,
carrier feasibility, aggregate claim limits, handoff, and bidirectional
closure; unchanged R1-R10 rows were checked only for repair-introduced
contradictions. It confirmed the fixed concrete constructor, real
`httpclient` retry/authorization/authority/TLS/response path,
generated-client subprocess owner, cleanup and command fitness, and complete
R2/R5/R6/R7/R9 mapping. It independently reran the carrier-anchor command and
confirmed its stated non-OAuth limit. This receipt and the `ready` status are
receipt-only changes to the reviewed proof contract.

Planning readiness then exposed one TD-013 placement gap: bootstrap could not
drive a provider-failing operation before either resource adapter existed.
Root Test Design repair split that composed claim at its accepted owners: the
OAuth package's private resolution path now proves an operation-local
`provider_unavailable`, while bootstrap proves that outbound auth supplies no
readiness participant or provider-state input and that repeated health reads
perform no auth I/O. The repair adds no public token operation, resource
adapter dependency, or production/test seam.

Fresh focused independent QA review on candidate SHA-256
`7dfc1798f67dd40b927ee0eea1389dd0b4411e260c0b11627499d945f4452558`
returned `PASS` with no surviving finding. It falsified the package-private
carrier, composed bootstrap oracle, adjacent TD-005-TD-007/TD-014-TD-015
ownership, cleanup, aggregate gates, and bidirectional closure. Existing
bootstrap/health baseline proof established fixture feasibility only, not
outbound OAuth behavior. This receipt and `ready` status are receipt-only
changes to the reviewed proof contract.

## Bidirectional closure

| Accepted surface | Final disposition |
| --- | --- |
| R1 profile validation, retention, stripping, repeatability, and dependency cleanup | TD-001 and TD-016 |
| R2 exactly one fixed dependency and immutable credential/config boundary | TD-001, TD-003, TD-008, TD-010 |
| R3 Basic disclosure only to one admitted token endpoint | TD-003 and TD-004 |
| R4 exact request and strict bounded response | TD-004 and TD-002 |
| R5 on-demand reuse, expiry margin, operation fixation, and no replay refresh | TD-005, TD-008, TD-009, TD-011, TD-012 |
| R6 coalescing, caller cancellation, provider budget, failure window, recovery, and retirement | TD-006, TD-007, TD-014 |
| R7 stable failures, disclosure controls, and bounded telemetry | TD-002, TD-004, TD-007, TD-008, TD-010, TD-015 |
| R8 startup, optional degradation, cached health, drain, and shutdown | TD-013 and TD-014 |
| R9 HTTP generated/concrete consumption, retry attachment, and downstream ownership | TD-008 and TD-009 |
| R10 gRPC application/control attempts, TLS, long streams, reconnect, and downstream ownership | TD-010, TD-011, TD-012 |
| Preserved inbound auth, bounded transports, config, health, bootstrap, profile, correlation, telemetry, and generated-client contracts | Owning-package aggregate commands and TD-008 through TD-016 where the selected change crosses them |
| External checkpoints and reopen conditions | External proof table; each matrix row names its exact Specification, Technical Design, Go Ownership, or concrete-client reopen owner |

Planning may choose task order and acceptance-unit placement only. It must
preserve every selected scenario, observable, fixture/input, command, cleanup
rule, external boundary, and reopen route without inventing proof policy.
