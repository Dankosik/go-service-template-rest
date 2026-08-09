# Test design — safe gRPC clients and transport-neutral failures

status: ready

Accepted inputs:

- [`spec.md`](spec.md), SHA-256 `1bb0965f876b052abd8e5be2142acfc8734141d5506e719d9529e6a425860cac`;
- [`design/overview.md`](design/overview.md), SHA-256 `a304576e75220e630ec7a67b9ec9287020f45f485598034809dac8444a0d6667`;
- current implementation baseline `da89db83a78ca4a19fefe66d4105f69fb73b7ff0` with grpc-go `v1.82.1`.

The ready Specification owns behavior and the ready Technical Design owns file,
package, and proof placement. This plan adds no behavior, implementation seam,
generated contract, rollout step, or Planning order.

## Proof-boundary decisions

- R1 uses the existing raw HTTP/2 peer because the accepted observable is a
  PING frame on an idle transport. A dial-option assertion would false-pass if
  the option were zero-filled or ignored. grpc-go transport timers are outside
  `testing/synctest`, so one bounded real-time window is required. The default
  and enabled connections share that window; both explicitly disable health so
  no `Watch` stream makes the transport non-idle.
- R2 uses real loopback gRPC servers, grpc-go's standard health server/client,
  the manual resolver, and the production `grpcclient.New` composition. A mock
  balancer or asserted service-config string cannot prove backend eligibility,
  `UNIMPLEMENTED` fallback, in-flight preservation, or Watch cancellation.
  Tests that register manual resolver schemes use distinct scheme names, stay
  top-level sequential, and do not call `t.Parallel`; the package-global
  resolver registry is not a test-owned concurrent seam.
- R2's protected-profile composition strengthens the existing real TLS/OIDC
  boundary contract rather than adding a second server harness. Its server-side
  Watch observation proves grpc-go's internal stream received the dial
  credential; handler-visible principal and reserved-metadata absence prove the
  OIDC and sanitizer sides of the same path. A call-scoped or fake-interceptor
  test would false-pass while the automatic Watch remained unauthenticated.
- R3 uses focused leaf/package proof for classification rules, the real HTTP
  reference route for the HTTP projection, and a real `grpcx.NewServer` caller
  for the gRPC projection. A mapper-only test cannot prove either wire outcome.
- No Docker-backed integration or process e2e test is selected. Every changed
  boundary is observable in the owning package or reference composition; a
  broader level would duplicate the same oracle. `make test`, race, lint,
  generated-contract, and template-profile commands remain aggregate gates,
  not substitute scenarios.
- There is no residual-risk acceptance and no unavailable fixture, environment,
  or command. Test and fixture code are Implementation-owned.

## R1 — idle keepalive is opt-in

| ID and source | Disposition and carrier | Plausible wrong behavior | Controlled setup, trigger, and discriminating oracle | Boundary and executable command | Fixture/input and ownership |
| --- | --- | --- | --- | --- | --- |
| TD-R1-01 — R1 default and success criterion 1 | Strengthen `internal/infra/grpcclient/client_test.go::TestDefaultConfigIsFinite`. | `DefaultConfig` still ships `30s/10s`, health is not the default, or another transport bound drifts. | Build the fixed unreachable target and assert the existing finite bounds, round robin, `HealthCheck=true`, and keepalive `(0,0)`. The old defaults fail directly. | Unit: `go test ./internal/infra/grpcclient -run '^TestDefaultConfigIsFinite$' -count=1` | Existing decision-neutral config; grpcclient test owner. Land with D1/D2. Reopen Design only if the config owner or opt-in shape changes. |
| TD-R1-02 — no-I/O invariant | Existing proof remains sufficient in `client_test.go::TestNewPerformsNoNetworkIOAndReturnsOwnedConnection`. | Construction dials, starts resolution, or stops returning a caller-owned connection. | Construct against `dns:///unreachable.invalid:443`, require immediate success, then close. A blocking/eager dial fails instead of being hidden by a mock. | Component: `go test ./internal/infra/grpcclient -run '^TestNewPerformsNoNetworkIOAndReturnsOwnedConnection$' -count=1` | Existing unreachable target and grpc-go `NewClient`; grpcclient test owner. Reopen Design if construction deliberately becomes eager. |
| TD-R1-03 — complete-pair validation | Strengthen `client_test.go::TestNewRejectsMissingTrustAndBounds`. | A partial or negative pair reaches grpc-go, or `(0,0)` is rejected. | Retain existing invalid cases and add interval-only `(positive,0)`, timeout-only `(0,positive)`, negative interval, and negative timeout. Each must return no `ClientConn`; TD-R1-01/04 prove the two accepted states. | Unit: `go test ./internal/infra/grpcclient -run '^TestNewRejectsMissingTrustAndBounds$' -count=1` | Existing table; use decision-neutral `10s/1s` positives and `-1ns` negatives. grpcclient test owner. Reopen Specification if a third keepalive state is required. |
| TD-R1-04 — default silence and explicit idle ping | Replace `keepalive_test.go::TestClientPingsAConnectionWithNoActiveRPC` with `TestClientIdleKeepaliveIsOptIn`; remove `keepalive_parity_test.go` and its superseded test. | The default still appends zero/default keepalive and pings, the explicit pair is not wired, or health traffic makes the observation a false positive. | Start two raw peers and wait for both HTTP/2 handshakes. Both configs set `HealthCheck=false`; neither opens an RPC stream. The default peer must see no PING for 35s, past the old 30s policy; the explicit `10s/1s` peer must see a PING within the same window. Any application HEADERS frame fails isolation. | Raw loopback component: `go test ./internal/infra/grpcclient -run '^TestClientIdleKeepaliveIsOptIn$' -count=1` | Reuse the raw peer in `keepalive_test.go`; add handshake and unexpected-stream observables locally. grpcclient test owner. Planning must pair test replacement and parity-test deletion with D1. Reopen Design if grpc-go exposes a deterministic transport clock. |
| TD-R1-05 — adaptive interval after `too_many_pings` | Sufficient pinned dependency proof; do not duplicate grpc-go's state machine. | The adapter claims to own adaptation, or the pinned dependency no longer increases the later-connection interval. | Run grpc-go's own GOAWAY fixture; it sends `ENHANCE_YOUR_CALM/too_many_pings` and asserts the client interval changes from 10s to 20s. Repository proof only confirms D1 does not mutate/retry around it. | Dependency unit: `go test google.golang.org/grpc -run '^Test/ClientUpdatesParamsAfterGoAway$' -count=1` | grpc-go `v1.82.1` from `go.mod`; dependency proof owner. Reopen Technical Design on grpc-go version change or test removal. |

Fail-before discriminator: the current tree fails TD-R1-01 because it ships
`30s/10s`, fails the future zero-pair acceptance in TD-R1-03, and its current
raw-peer test proves the opposite of TD-R1-04. Missing replacement test code is
not itself behavioral evidence.

## R2 — round robin consumes standard health

| ID and source | Disposition and carrier | Plausible wrong behavior | Controlled setup, trigger, and discriminating oracle | Boundary and executable command | Fixture/input and ownership |
| --- | --- | --- | --- | --- | --- |
| TD-R2-01 — `SERVING -> NOT_SERVING -> SERVING` eligibility | New `internal/infra/grpcclient/health_checking_test.go::TestRoundRobinHealthControlsEligibility`. | The health package is not activated, the empty service is not watched, `NOT_SERVING` stays pickable, or later `SERVING` never restores the backend. | Resolve two test-local backends, each with a standard health server and counted application method. Under a 5s outer context, call until both `SERVING` backends have served. Set A `NOT_SERVING`; require eight consecutive B calls as the convergence fence, reset counters, then require eight validation calls with `A=0,B=8`. Set A `SERVING`; call until A serves again within the same bound. No sleep or elapsed duration is an oracle. | Loopback component: `go test ./internal/infra/grpcclient -run '^TestRoundRobinHealthControlsEligibility$' -count=1` | New file owns its backends, manual resolver, counters, and bounded call-loop controls. Planning must land it with D2. Reopen Specification for a non-empty service name or different eligibility rule. |
| TD-R2-02 — in-flight RPC preservation | New `health_checking_test.go::TestRoundRobinHealthTransitionDoesNotCancelInflightRPC`. | Publishing `NOT_SERVING` cancels an RPC already picked on that backend, or new work continues reaching it. | Under a 5s outer context, start calls until a channel confirms a blocking method entered A; B-directed attempts return normally. Publish A `NOT_SERVING`, require eight consecutive B probes, and assert the blocked call's result channel is still empty. Release it and require success within 2s; eight further probes must produce `A=0,B=8`. Channels own entry/release; timeouts are outer diagnostics. | Loopback lifecycle component: `go test ./internal/infra/grpcclient -run '^TestRoundRobinHealthTransitionDoesNotCancelInflightRPC$' -count=1` | Same test-local backend shape, no shared harness change. grpcclient test owner. Reopen Specification if health must terminate in-flight work. |
| TD-R2-03 — unsupported-peer compatibility | New `health_checking_test.go::TestHealthUnimplementedFallsBackToConnectivity`. | A peer without standard health is permanently excluded or retried by application policy. | Register the application service but no health service, use default health-enabled round robin, and require one application RPC to succeed within 2s with handler count exactly one. A generic unknown-service handler is forbidden because it could accidentally answer `Watch`; the real server must return `UNIMPLEMENTED`. | Loopback component: `go test ./internal/infra/grpcclient -run '^TestHealthUnimplementedFallsBackToConnectivity$' -count=1` | Test-local ordinary grpc.Server and counted application method. grpcclient test owner. Reopen Specification if unsupported peers become invalid. |
| TD-R2-04 — repaired `pick_first` and explicit disable | Strengthen `load_balancing_test.go`; retain `TestLoadBalancingPolicyDecidesHowManyBackendsAreReached` with health explicitly disabled and add `TestHealthConfigurationCallability`. | Address-selection proof is captured by health traffic; `HealthCheck=false` is ignored; or repaired direct `pick_first` is rejected/held unhealthy when health is enabled. | Existing two-address count test sets `HealthCheck=false`. The new table uses one application backend publishing `NOT_SERVING`; each 2s call must succeed exactly once for round robin with health disabled, `pick_first` with health disabled, and direct `pick_first` with `HealthCheck=true`. The `NOT_SERVING` control makes each callability result discriminate. | Component: `go test ./internal/infra/grpcclient -run '^TestLoadBalancingPolicyDecidesHowManyBackendsAreReached$' -count=1 && go test ./internal/infra/grpcclient -run '^TestHealthConfigurationCallability$' -count=1` | `load_balancing_test.go` owns address selection/callability only; no health transitions move into it. Reopen repaired R2 if grpc-go later gives direct `pick_first` a health listener. |
| TD-R2-05 — health Watch lifecycle | New `health_checking_test.go::TestClientConnCloseCancelsHealthWatch`. | `ClientConn.Close` leaves the control stream or goroutine alive. | A test health server signals when its empty-service `Watch` starts. Close the client connection and require the server-side stream context to finish within 2s; the timeout is only the liveness failure signal. Run under race in TD-GATE-03. | Loopback lifecycle component: `go test ./internal/infra/grpcclient -run '^TestClientConnCloseCancelsHealthWatch$' -count=1` | Test-local Watch observer in the new file. grpcclient test owner. Reopen Design if lifecycle ownership moves away from `ClientConn`. |
| TD-R2-06 — client-owned service config and no application retry | Existing `resolver_live_test.go::TestResolverServiceConfigProxyTLSAreClosedByClientPolicy` remains sufficient. | A resolver replaces round robin/health, installs a retry policy, or changes the transport trust boundary. | The effective resolver supplies a custom balancer and retry policy while a native control proves the proxy route is live. The shared client must bypass the proxy, report `DisableServiceConfig=true`, build/pick the resolver balancer zero times, make exactly one committed call, and retain TLS authority checks. | Live loopback contract: `go test ./internal/infra/grpcclient -run '^TestResolverServiceConfigProxyTLSAreClosedByClientPolicy$' -count=1` | Existing child-process fixture; grpcclient resolver proof owner. Reopen Design for resolver-supplied service config, proxies, or application retry. |
| TD-R2-07 — propagation stream-observable isolation | Strengthen every `grpcclient.New` construction in `propagation_test.go` with a local config whose `HealthCheck=false`; do not change `startMetadataCaptureServer`. | The automatic control `Watch` occupies `streamMetadata` before the test's application `Health.Watch`, so the test passes/fails against the wrong stream. | Existing unary/stream calls and metadata oracles remain unchanged. Explicit disable ensures the only observed stream is the application Watch driven by the test. | Component regression: `go test ./internal/infra/grpcclient -run '^TestPropagation' -count=1 && go test ./internal/infra/grpcclient -run '^TestPerRPC' -count=1` | Existing propagation fixtures; proof-isolation change only. Planning must place it in the D2 acceptance unit. Reopen Technical Design if another current proof needs the shared harness to observe control streams. |
| TD-R2-08 — transparent-retry stream-observable isolation | Strengthen `transparent_retry_test.go::TestTransparentRetryReappliesClosedPropagationPerAttempt` with `HealthCheck=false`; do not change the raw peer or its two-attempt oracle. | The control `Watch` becomes captured stream 1, so captured stream 2 is the first application attempt and the transparent retry is never proved. | Preserve the forced `REFUSED_STREAM`, exactly two application attempt headers, distinct attempt spans, and propagation assertions. Health disable makes both captured streams application attempts. | Raw loopback regression: `go test ./internal/infra/grpcclient -run '^TestTransparentRetryReappliesClosedPropagationPerAttempt$' -count=1` | Existing raw peer; proof-isolation change only. Planning must place it in the D2 acceptance unit. Reopen Technical Design only if this proof intentionally starts observing health-control traffic. |
| TD-R2-09 — protected automatic Watch composition | Rename and strengthen `internal/infra/oidcjwt/grpc_tls_contract_test.go::TestGRPCAuthnBoundaryOverTLS`. | Only application calls receive the connection credential, a call-scoped credential incorrectly rescues an unauthenticated automatic Watch, or the dial credential bypasses reserved-correlation sanitization. | Reuse the real TLS server, OIDC verifier, standard empty-service health server, existing signed token, and application counters. First build a default `grpcclient` without dial credentials, call `Connect`, and use a test interceptor outside OIDC to require one unauthenticated automatic Watch attempt. Invoke the application unary with valid call-scoped auth; require failure and zero new application calls, proving the ineligible backend cannot be rescued per call. Then build the default client with a dial credential returning the bearer token plus forged reserved values, call `Connect`, and require the protected Watch handler to observe a verified principal and none of those values; invoke the unary without call-scoped metadata and require exactly one new authenticated application call. The existing raw grpc-go connection retains the direct missing-credential auth control. | Composed TLS/OIDC contract: `go test -vet=off ./internal/infra/oidcjwt -run '^TestGRPCAuthnBoundaryOverTLS$' -count=1` | Existing TLS certificates, verifier harness, signed-token builder, service descriptor, and counters; add only local dial/call credentials plus unauthenticated-attempt and authenticated-handler observations. OIDC gRPC boundary test owner. Reopen Technical Design if grpc-go no longer applies dial credentials to internal Watch or the composed proof needs production OIDC changes. |
| TD-R2-10 — profile-safe contract-test rename | Update the `GRPC=none` removal path for `grpc_tls_contract_test.go`, the pointer in `grpc_test.go`, and the existing profile assertions. | The old filename is still removed while the renamed gRPC-only test survives a generated `GRPC=none` repository, or enabled variants retain the old non-contract filename. | Run the repository structure check, then the authn template profile matrix with explicit path oracles: each generated OIDC/gRPC-enabled checkout contains `grpc_tls_contract_test.go` and not `grpc_tls_test.go`; every `GRPC=none` checkout contains neither. Retained enabled variants also compile and execute the contract. | Structural/profile: `make project-structure-check && TEMPLATE_INIT_PROFILE=authn make template-init-check` | Existing init script and authn profile fixtures; extend their path assertions without a new fixture. Script/profile proof owner. Reopen Go Ownership if the boundary proof moves package or the profile removal owner changes. |

Fail-before discriminator: the partially implemented tree now proves protected
`Health/Watch` but exposes no dial-level credential from `grpcclient.Options`,
so the composed automatic Watch cannot authenticate. The old contract-test
filename is still wired into profile removal. Leaving either isolation edit out
still makes the control `Watch` consume the existing stream oracle; that
distinct false-pass/failure is why TD-R2-07 and TD-R2-08 remain mandatory.

## R3 — domain failure identity is transport-neutral

| ID and source | Disposition and carrier | Plausible wrong behavior | Controlled setup, trigger, and discriminating oracle | Boundary and executable command | Fixture/input and ownership |
| --- | --- | --- | --- | --- | --- |
| TD-R3-01 — neutral vocabulary and registry | Add `internal/failure/failure_test.go::TestCodesAreStableAndTransportNeutral` and `TestClassifySkipsNilAndUsesFirstMatch`. | A shared code string drifts, generic `conflict` becomes a domain identity, mapper order changes, or nil entries break a profile slice. | Assert exactly `bad_request`, `unauthorized`, `forbidden`, `not_found`, `method_not_allowed`, `already_exists`, `request_entity_too_large`, profile-owned `request_header_fields_too_large`, `unprocessable_content`, `too_many_requests`, `internal_error`, `service_unavailable`, and `gateway_timeout`, with no domain `conflict`. Run nil, non-match, first-match, and later-match controls and compare the whole `Classification`. The package test imports no transport. | Leaf unit: `go test ./internal/failure -count=1` | New leaf test; accepted values come from R3. Failure package test owner. Reopen Specification for a real new caller action. |
| TD-R3-02 — HTTP catalog with two 409 identities | Strengthen `internal/problem/problem_test.go`; remove only the obsolete global status-uniqueness assertion. | Adding `already_exists` makes status-only lookup nondeterministic, changes the HTTP fallback, or `ForCode` collapses the two identities. | Require unique codes; exact `ForCode(conflict)` and `ForCode(already_exists)` definitions both return status 409, title `conflict`, and type `https://www.rfc-editor.org/rfc/rfc9110#section-15.5.10`; exact `For(409)` remains `conflict`; every published code resolves; unknown status/code still refuse; `All` remains a copy. | HTTP catalog unit: `go test ./internal/problem -count=1` | Existing catalog test owner. Planning pairs this with D3/D4; no generated source. Reopen Specification if the HTTP-only fallback is removed or aliased. |
| TD-R3-03 — feature-owned article classification | Add `examples/reference-service/internal/article/errors_test.go::TestClassifyErrorUsesStableFailureIdentity`; retain `TestServiceCreatePreservesAlreadyExistsIdentity`. | The feature still maps its collision to generic `conflict`, leaks transport ownership, or loses wrapped sentinel matching. | Table wrapped `ErrNotFound`, `ErrAlreadyExists`, `ErrInvalid`, and unknown. Require exact neutral code and safe detail; the collision must be `already_exists`. Existing service proof keeps the sentinel through the use case. | Feature unit: `go test ./examples/reference-service/internal/article -count=1` | Existing sentinels move to `article/errors.go`; new test beside them. Reopen Specification for a shipped compatibility owner or new caller action. |
| TD-R3-04 — generic HTTP projection and fallback | Migrate/strengthen `internal/infra/http/domain_errors_test.go` to `failure`; retain first-match, retry rounding, deadline precedence, and unknown fallback, and add exact `already_exists` projection. | The neutral mapper cannot drive HTTP, retry hints drift, deadline loses precedence, unknown text escapes, or classified `already_exists` falls back to `conflict`/500. | Extend the existing response table with `already_exists -> 409`; require the exact code/status/type/detail and no Retry-After. Retain wrapped not-found, two retry delays, broad first-match control, deadline, unknown, nil mapper, and no-mapper fallback. | HTTP adapter: `go test ./internal/infra/http -count=1` | Existing response recorder and Problem decoder; HTTP adapter test owner. Reopen Specification if HTTP precedence or fallback changes. |
| TD-R3-05 — real HTTP slug-collision projection and composition | Rename/strengthen `httpapi/router_test.go::TestRouterCreateArticleRejectsDuplicateSlugWithAlreadyExists` and add `reference_test.go::TestReferenceServiceMapsAlreadyExistsOverHTTP`. | The route remains `conflict`, selects the status-only fallback after classification, changes status/type/title/detail, or the reference root fails to install the feature mapper. | Seed slug `clear-owners`, POST it again through both the generated route fixture and `NewHandler`. Require status 409, code `already_exists`, title `conflict`, type `https://www.rfc-editor.org/rfc/rfc9110#section-15.5.10`, detail `an article with this slug already exists`, and no wrapped internal text. | HTTP component/composition: `go test ./examples/reference-service/internal/httpapi ./examples/reference-service -count=1` | Existing memory repository, generated router, and root handler. HTTP reference/root test owners. Reopen Specification for a real consumer requiring an alias. |
| TD-R3-06 — real gRPC slug-collision projection | Add `examples/reference-service/grpc_failure_mapping_contract_test.go::TestArticleAlreadyExistsMapsThroughGRPCTransport`. | gRPC still derives `ABORTED` from the HTTP catalog, special-cases a transport error, loses the feature mapper, or omits machine identity. | Register a test unary method on real `grpcx.NewServer` whose handler returns wrapped `article.ErrAlreadyExists`; pass `article.ClassifyError` and error domain `reference-service.test`. The caller must receive `ALREADY_EXISTS`, message `an article with this slug already exists`, exactly one `ErrorInfo` with `Reason=ALREADY_EXISTS` and `Domain=reference-service.test`, no `RetryInfo`, and no wrapped internal text. | Composed gRPC contract: `go test ./examples/reference-service -run '^TestArticleAlreadyExistsMapsThroughGRPCTransport$' -count=1` | Use repository `grpctest` descriptors/bufconn and the actual feature mapper; new file is gRPC-profile-owned and removed by `GRPC=none`. Reopen Technical Design if production composition cannot carry the neutral mapper. |
| TD-R3-07 — non-conflict mappings, structured details, and docs | Migrate/strengthen `internal/infra/grpc/docs_test.go` and `error_details_test.go` to `failure`; add `already_exists`, remove domain `conflict`, retain every other expected code/reason/retry result. | A non-conflict status drifts, docs describe a different table, reason is not upper snake case, retry hints change, or an unclassified error gains details/text. | The docs test reads neutral constant identifiers and compares every published mapping to `mappedStatus`. Existing real-server detail tests retain exact RetryInfo, ErrorInfo domain/reason separation, empty-domain behavior, and sanitized unknowns. | gRPC adapter: `go test ./internal/infra/grpc -count=1` | Existing tests and `docs/grpc.md`; gRPC adapter/docs proof owners. Reopen Specification for a new failure code and Design if detail ownership changes. |
| TD-R3-08 — precedence, status trust, and health pass-through | Strengthen `interceptors_test.go::TestMapErrorAppliesEachBoundarysTrustRule` so cancellation/deadline cases also have a mapper that would classify them; retain server health and handler-status tests. | Classification overrides cancellation/deadline, handler status escapes sanitization, policy status is rejected, or standard health status is remapped. | For canceled/deadline errors install a broad matching mapper and still require CANCELED/DEADLINE_EXCEEDED with no classified detail. Retain exact trusted/untrusted status table, unknown-service health status, and raw handler-status rejection. | gRPC boundary: `go test ./internal/infra/grpc -run '^TestMapErrorAppliesEachBoundarysTrustRule$' -count=1 && go test ./internal/infra/grpc -run '^TestServerHealthPreservesStandardUnknownServiceStatus$' -count=1 && go test ./internal/infra/grpc -run '^TestServerSanitizesUntrustedHandlerStatus$' -count=1` | Existing interceptor/server harnesses; gRPC adapter test owner. Reopen Specification only if precedence or trust changes. |
| TD-R3-09 — existing producers retain meaning | Migrate `startup_dependencies_test.go::TestPostgresDomainErrorsMapSaturationToRetryableUnavailable` and the gRPC reference stream classifier/tests to `failure` without changing their oracles. | The type move changes PostgreSQL saturation, retry delay, `RESOURCE_EXHAUSTED`, safe detail, or bounded streaming behavior. | Wrapped `postgres.ErrSaturated` remains service-unavailable with positive retry and `ErrConnect` remains unclassified. Reference over-limit streams still return `RESOURCE_EXHAUSTED` through real `grpcx.NewServer`. | Producer regression: `go test ./cmd/service/internal/bootstrap ./examples/grpc-reference-service -count=1` | Existing producer fixtures; bootstrap and reference-example proof owners. Reopen Specification if producer caller meaning changes. |
| TD-R3-10 — dependency direction and profile survival | Add the focused `internal/failure` depguard rule; update profile/template assertions so neutral failure and HTTP proof survive `GRPC=none`, while the composed gRPC contract file is absent, and `GRPC=enabled` retains/runs it. | The neutral package imports a transport, minimal generation removes shared classification/HTTP behavior, a retained REST example keeps the gRPC-only contract after the runtime is removed, or enabled generation loses the contract. | Lint must reject forbidden imports. The gRPC profile check adds a dedicated checkout initialized with `DATABASE=none GRPC=none REFERENCE_EXAMPLE=keep`: `go test ./...` and `go build ./cmd/service` must pass; `examples/reference-service`, `internal/failure`, and its HTTP `already_exists` proof must remain; `examples/reference-service/grpc_failure_mapping_contract_test.go`, `examples/grpc-reference-service`, `internal/infra/grpc`, and `internal/infra/grpcclient` must be absent. The existing minimal and `GRPC=enabled REFERENCE_EXAMPLE=keep` checkouts retain their current compile/build and presence/absence controls. | Structural/profile gates: `make lint`; `TEMPLATE_INIT_PROFILE=minimal make template-init-check`; `TEMPLATE_INIT_PROFILE=grpc make template-init-check` | `.golangci.yml`, `scripts/init-module.sh`, `scripts/profiles/database-none/startup_dependencies.go.tmpl`, and `scripts/ci/template-init-check.sh`; accepted design owners. The dedicated mixed fixture belongs inside the existing `grpc` profile branch, so the named command executes it. Reopen Technical Design if profile ownership changes. |

Fail-before discriminator: the current tree has no neutral package, the article
classifier returns `conflict`, the HTTP route publishes `conflict`, and gRPC
projects it to `ABORTED`. TD-R3-03 through TD-R3-06 therefore distinguish the
accepted correction at the feature and both composed transport boundaries.

## Carried compatibility and aggregate gates

These rows dispose the deliberately preserved behavior named by the ready spec.
They are not substitutes for the R1-R3 scenarios above.

| ID and preserved surface | Existing proof and exact command | Wrong behavior rejected / observable | Planning placement and reopen owner |
| --- | --- | --- | --- |
| TD-COMPAT-01 — propagation, metadata, transparent retry, resolver trust | `go test ./internal/infra/grpcclient -count=1` | Reserved metadata crosses, caller metadata mutates, resolver policy wins, or transparent retry is replaced/loses per-attempt propagation. Existing wire metadata, call-count, and span oracles discriminate. | Run after TD-R2-07/08; reopen Technical Design for a trust-boundary or retry-policy change. |
| TD-COMPAT-02 — server deadline, admission, health, keepalive, shutdown, telemetry | `go test ./internal/infra/grpc -count=1` | Client work accidentally changes server admission/readiness/drain, server keepalive, shutdown, deadline, or telemetry. Existing lifecycle and wire observables remain the authority. | Regression gate after R1/R2; reopen the superseded transport-hardening owner on failure. |
| TD-COMPAT-03 — TLS, generated streaming, bounded aggregation | `go test ./internal/infra/grpcclient ./cmd/service/internal/bootstrap ./examples/grpc-reference-service -count=1` | TLS authority weakens, a cardinality stops composing, aggregation becomes unbounded, or cancellation leaks work. Exact hostname/status/message/lifecycle oracles already exist. | Regression gate after R3 type migration; reopen Technical Design only for a changed transport/generated boundary. |
| TD-GATE-01 — focused functional closure | `go test ./internal/failure ./internal/problem ./internal/infra/http ./internal/infra/grpc ./internal/infra/grpcclient ./cmd/service/internal/bootstrap ./examples/reference-service/... ./examples/grpc-reference-service/... -count=1` | One owning package or reference composition fails outside a selected test regex. | Run after focused scenarios; Implementation owns repair, upstream owner reopens only for a contract/design conflict. |
| TD-GATE-02 — order-sensitive health/isolation stability | `go test -vet=off -count=5 -shuffle=on ./internal/infra/grpcclient -run '^TestRoundRobinHealth' && go test -vet=off -count=5 -shuffle=on ./internal/infra/grpcclient -run '^TestHealth' && go test -vet=off -count=5 -shuffle=on ./internal/infra/grpcclient -run '^TestClientConnCloseCancelsHealthWatch$' && go test -vet=off -count=5 -shuffle=on ./internal/infra/grpcclient -run '^TestPropagationPoliciesApplyToUnaryAndStreamingRPCs$' && go test -vet=off -count=5 -shuffle=on ./internal/infra/grpcclient -run '^TestTransparentRetryReappliesClosedPropagationPerAttempt$'` | Health convergence or stream ownership depends on test order, scheduling luck, or an unowned sleep. | Run after all R2 tests; keep the 35s keepalive case out because repetition adds no scheduling coverage to its unavoidable real-time window. |
| TD-GATE-03 — client/server lifecycle race | `go test -vet=off -race ./internal/infra/grpcclient ./internal/infra/grpc` | Health Watch state, connection close, in-flight transition, keepalive peer, or server lifecycle races. | Mandatory for success criterion 7; Implementation owns race repair, concurrency policy reopens only if behavior must change. |
| TD-GATE-04 — repository unit aggregate | `make test` | A non-selected unit regression survives the focused set. | Run after focused proof; this is breadth, not a scenario oracle. |
| TD-GATE-05 — structure, lint, generated contracts, profiles | `make project-structure-check`; `make lint`; `make proto-check`; `make openapi-check`; `TEMPLATE_INIT_PROFILE=minimal make template-init-check`; `TEMPLATE_INIT_PROFILE=grpc make template-init-check` | File placement/import direction drifts, docs/mapping lint fails, generated authority changes, or profile ownership is wrong. OpenAPI/Protobuf commands prove unchanged canonical/generated surfaces only. | Run after code/docs/profile edits; no regeneration is authorized unless these checks expose actual drift owned by an upstream decision. |

Performance, data/migration, money, and rollout lenses are excluded by inspected
current boundaries: the accepted design adds no persistence, schema, runtime
configuration, deployed neighbor, workload budget, or rollout transition. The
security lens contributes only preserved resolver/credential/metadata and
status-sanitization proof above; authentication and authorization policy do not
change.

## Review disposition

The shared review trigger applies because this Test Design controls
client-visible failure identity and connection-lifecycle proof.

1. Independent whole-candidate QA review at Test Design SHA-256
   `c4aefbf44e273aeb9a9c509ccd9113ebcdbb2c5debee20e482afaae94265406d`
   returned `FAIL`: the selected minimal profile removes all reference examples
   and the enabled gRPC profile retains the contract, so neither could detect
   the gRPC-only contract test surviving the supported
   `GRPC=none REFERENCE_EXAMPLE=keep` combination.
2. Root repair added that exact mixed profile, its compile/build procedure, and
   distinct retained/absent path oracles to TD-R3-10 without changing behavior,
   design, or another scenario.
3. Independent focused re-review at repaired SHA-256
   `3e79db7ccdbe0d6522f760c70f8f47e87c0ee979043d1b4eb9637c543fcfce01`
   returned `PASS`; no adjacent material gap or upstream reopen remained.
4. Implementation evidence later reopened R2 after the OIDC/JWT boundary made
   `Health/Watch` protected and the shared client lacked a credential seam for
   grpc-go's internal stream.
5. Root repair added TD-R2-09 for the exact TLS/OIDC automatic-Watch path and
   TD-R2-10 for the contract-test rename/profile boundary. R1, R3, and the
   earlier R2 scenarios remain unchanged.
6. Independent focused QA review returned `FAIL`: the positive composed path
   did not prove missing dial credentials keep the backend ineligible, and the
   rename command had no explicit path oracle. Root repair added the
   unauthenticated automatic-Watch/call-scoped-auth control and exact
   enabled/disabled filename assertions. Fresh focused re-review is pending
   while this artifact is `draft`.
7. Focused re-review found one remaining feasibility gap: the positive client
   was never told to connect before awaiting its automatic Watch. Root repair
   now calls `Connect` on both negative and positive control connections; a
   final focused re-review at SHA-256
   `d5adb3b8a4182285edc62ee8fbfc4f6456f84748b1330efa2df26d0c4f4cee2b`
   returned `PASS`. This final edit changes only status and review disposition.

Current command-feasibility evidence is intentionally narrower than future
implementation proof: focused existing client, gRPC, HTTP, catalog, and
reference commands passed 217 tests, and grpc-go's pinned adaptive-interval
command passed. New R1-R3 tests and profile fixtures do not exist until
Implementation and therefore have no passing claim here.

## Closure and Planning handoff

- Every R1-R3 claim, deliberate compatibility rule, and triggered lifecycle,
  retry, health, resolver-trust, sanitization, generated-authority, and profile
  boundary has one scenario or existing-proof disposition above.
- Each new/strengthened test stays in the proof file assigned by Technical
  Design. In particular, `propagation_test.go` and
  `transparent_retry_test.go` disable health locally; their harnesses are not
  changed and they do not become health-policy tests. The existing TLS/OIDC
  boundary contract alone owns protected automatic-Watch composition.
- Implementation must land each production change with its owned focused proof
  and cleanup: D1 removes `keepalive_parity_test.go`; D2 includes both isolation
  edits; D3-D5 include both composed transport outcomes; D6 includes the two
  profile gates. Planning may choose order only.
- Reopen Specification for a changed caller-visible outcome, failure identity,
  health eligibility rule, or keepalive state. Reopen Technical Design for a
  changed package/file owner, service-config authority, production seam, or
  profile boundary. Test implementation difficulty alone does not reopen either
  unless the named observable is impossible from the accepted owner.
