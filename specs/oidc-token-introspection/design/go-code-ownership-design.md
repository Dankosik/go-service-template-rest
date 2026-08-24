# OIDC token introspection Go Code / Ownership Design

status: ready for independent Go Ownership and Technical Design Review

Consumes:

- [`../spec.md`](../spec.md)
- [`system-integration-design.md`](system-integration-design.md)
- current package documentation, callers, initializer controls, and generated
  source authority at `8967a4ac06d4fce0515703b15ffa5db35e5378ae`

This is an Ownership Map V1. It fixes responsibility, placement, dependency
direction, generated/manual authority, cleanup, and proof location. It does not
design function bodies or enter Test Design.

## Placement decision and reuse ladder

The first viable reuse rung is the current authentication transport contract:
move its bearer carrier, sanitized failure taxonomy, verification metric,
OpenAPI resolver, gRPC interceptors, principal publication, and stream-expiry
mechanics from `oidcjwt` into one narrow shared owner,
`internal/infra/bearerauthn`.

The concrete provider remains a separate adapter at
`internal/infra/oauthintrospection`. Its HTTP request, form, Basic credential,
JSON, numeric-time, and close mechanics use the Go standard library plus the
already-owned `internal/infra/httpclient`; no dependency is added. The accepted
client lacks transport-level response-header guards, so this design adds only a
caller-supplied `ResponseLimits` value and external/private constructors that
apply those guards while preserving existing constructors.

Strongest rejected source: exporting helpers from `oidcjwt` for the new adapter
to reuse. That keeps one provider importing another provider whose name and
lifecycle do not describe the shared responsibility. Upgrade condition: none;
new authentication mechanisms reopen Repository Architecture and Specification
rather than widening `bearerauthn` speculatively.

For the outbound client, the strongest rejected alternative is the unaccepted
dirty generic `httpclient.Config`/retry/instrumentation refactor. The selected
delta adds only the two header guards the current encapsulated transport cannot
otherwise enforce. If that concurrent candidate becomes accepted before
Implementation and supplies equivalent fixed-authority semantics, it replaces
the additive constructor names rather than creating a second API.

The only new interface is consumer-owned `bearerauthn.Verifier`, with the two
accepted trust engines as implementations in the source template and exactly
one in an initialized service. There is no factory, registry, runtime mode,
credential plugin, cache interface, or generic OAuth client.

## Responsibilities

| responsibility | affected path | current evidence | semantic owner | exact package/file action | dependency/composition/generated boundary | cleanup | proof owner | reopen condition |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Shared bearer grammar, 32 KiB bound, credential removal, sanitized kinds, verification metric, principal publication, HTTP resolver, gRPC public method and stream expiry | HTTP OpenAPI validation and native gRPC policy chain | These responsibilities currently live in `internal/infra/oidcjwt/{errors,http,grpc,metrics,token}.go` and are behaviorally required by both profiles | `internal/infra/bearerauthn` | Add the package and move the existing behavior; expose only `Runtime`, its resolver/interceptors/close, the verifier result/contract, and failure inspection needed by transport renderers | Imports `reqctx`, OpenAPI validation, OTel metric, and gRPC only in transport files; imports neither config nor a concrete verifier | Removed as one `authn-bearer` pack when `AUTHN=none` | Beside the package; real HTTP/gRPC seams remain the proving boundary | Reopen System Design if the two profiles no longer share these semantics |
| JWT/JWKS trust engine | Bearer runtime -> JWT verification | `oidcjwt` currently owns provider discovery, JWKS refresh, parsing, and transport adapters | `internal/infra/oidcjwt` | Retain Discovery/JWKS/JWT policy and lifecycle; implement only the shared verifier contract; delete/move inbound transport ownership | Imports `bearerauthn`, `authntrust`, `httpclient`, JWT/JWKS libraries; no OpenAPI or gRPC imports after the move | Concrete close still cancels refresh and closes idle provider connections; entire package removed from introspection/none outputs | Existing verifier/provider tests remain here; moved transport tests do not | Reopen ownership if JWT verification requires a transport-specific input |
| RFC 7662 trust engine | Bearer runtime -> fixed provider -> strict response -> principal/expiry | No current package or caller; Specification fixes one synchronous provider edge | `internal/infra/oauthintrospection` | Add policy, verifier/client lifecycle, and strict response admission files; implement the shared verifier contract | Imports `bearerauthn`, `authntrust`, `httpclient`, `reqctx`, stdlib; no config, HTTP server, gRPC, feature, JWT, OAuth token-source, or bootstrap imports | Idempotent close releases idle connections; no goroutine/cache/queue to join | Package-local pure parser proof plus a real TLS fake provider in the same package | Reopen Specification for another endpoint auth/response contract; System Design for another provider edge |
| Fixed-authority response-header enforcement | Provider-selected header bounds -> encapsulated `net/http.Transport` | Accepted `httpclient` constructors pin destinations but expose no way to set `ResponseHeaderTimeout` or `MaxResponseHeaderBytes`; the dirty generic refactor is unaccepted and does not compile | `internal/infra/httpclient` enforces caller-supplied transport guards; `oauthintrospection` chooses them | Add `ResponseLimits`, `NewExternalHTTPSWithLimits`, and `NewPrivateHTTPSWithLimits` in `client.go`; keep existing constructors and behavior unchanged | The shared client owns transport construction only; it imports no authn/provider/config packages and gains no retry, telemetry, body parser, target enum, or operation budget | Existing `Client.CloseIdleConnections` remains the lifecycle owner | `internal/infra/httpclient/client_test.go` proves positive-value admission and real header timeout/size enforcement without provider semantics | Reopen System Design if an accepted concurrent client cannot provide equivalent guards without a parallel API |
| Pure introspection endpoint and target-class parity | Config load and adapter policy construction | `authntrust` is already the config/verifier shared leaf for issuer/JWKS/token-profile predicates | `internal/authntrust` | Add introspection endpoint and exact target-class predicates/constants; mark token-profile-only files as JWT-specific | Pure stdlib leaf; may not import config, infra, transports, or composition | Introspection file removed from JWT/none outputs; package removed only when no authn profile remains | `internal/authntrust/introspection_test.go` | Reopen ownership if a rule needs runtime DNS/address state instead of a pure string decision |
| Immutable introspection tuple and secret-source admission | Config load -> validated snapshot -> bootstrap | `internal/config/authn_config.go` owns type/default/validation; secret policy rejects non-empty secret-like YAML | `internal/config` | Extend source-template `AuthnConfig` with marked introspection leaves; retain JWT validator as source default; add an introspection-specific static replacement file under `scripts/profiles` | Config imports only `authntrust`; it never imports `oauthintrospection` or `httpclient` | Whole authn section removed for none; profile-specific leaves removed for the unselected concrete engine | `internal/config/authn_config_test.go` and snapshot contract tests | Reopen Specification if the tuple changes; ownership if config generation stops being static |
| Authentication composition and lifecycle | Config -> selected concrete verifier -> shared runtime -> routers -> close | `cmd/service/internal/bootstrap/startup_authn.go` and `run.go` own current construction/order/close | `cmd/service/internal/bootstrap` | Keep `authnRuntime` and stage seam in `startup_authn.go`; move the source JWT constructor to `startup_authn_profile.go`; introspection initialization statically replaces only that profile constructor; keep existing run order | Bootstrap may import config, telemetry, `bearerauthn`, and exactly one generated concrete verifier; concrete packages never import bootstrap | Existing deferred close covers partial startup; ordered close remains after transport drain and before dependency/telemetry cleanup | `authn_bootstrap_test.go` owns stage and cleanup placement; exact failure cases belong to later Test Design | Reopen System Design if composition becomes runtime-selectable or close ordering changes |
| HTTP failure rendering | Shared authentication error -> existing Problem response/challenge | `internal/infra/http/request_errors.go` currently imports `oidcjwt.KindOf` | `internal/infra/http` consumes `bearerauthn.KindOf`; `internal/problem` and `internal/failure` retain response vocabulary | Replace the concrete-provider import only; keep current status/details/challenge mapping | HTTP adapter may import shared bearer runtime errors, never a concrete trust engine | Leaves with common authn marker; none output retains no authn mapper | Existing `request_errors_test.go` and `authn_router_test.go` | Reopen Specification for a new caller-visible category |
| gRPC failure rendering and public method | Shared runtime interceptor -> status | Currently wholly in `oidcjwt/grpc.go`; outer gRPC adapter trusts policy statuses | `internal/infra/bearerauthn/grpc.go` | Move current exact status/public-check/stream wrapper behavior; outer `internal/infra/grpc` receives only marker/comment updates | Shared runtime may import grpc packages in `grpc.go`; concrete engines may not | gRPC file removed when `GRPC=none`; whole shared pack removed when authn none | Shared package gRPC tests and existing real TLS contract proof | Reopen Specification for exposure/status changes; ownership if gRPC policy seam changes |
| Authentication verification telemetry and disclosure boundary | Both transport paths -> one bounded counter | `oidcjwt/metrics.go` currently mixes shared verification and JWT-only refresh counters | Shared verification counter in `bearerauthn`; JWKS refresh counter/log remain in `oidcjwt` | Split the existing file ownership; introspection adds no new metric or per-request log/span | Meter provider is injected; no config values, response fields, or raw errors cross | Runtime close has no telemetry stage; process telemetry still flushes last | Shared metrics assertions; JWT refresh assertions remain in JWT package | Reopen Specification for new labels/signals; Observability owner for alert policy |
| Three-way initializer selection and physical absence | Source template -> generated checkout -> `template.lock` | `scripts/init-module.sh` currently supports none/JWT with one overloaded marker | Initializer and profile templates | Add `oidc-introspection`, common/JWT/introspection marker decisions, two introspection replacement templates, exact removals, dependency tidy, lock readback, and idempotence | `scripts/init-module.sh` is canonical writer; `scripts/ci/template-init-check.sh` proves outputs; generated files are not manual authority | `scripts/profiles` and all markers leave every initialized checkout | Initializer checker owns presence/absence/dependency/profile proof | Reopen Repository Architecture if static replacement cannot keep one generated owner |
| Neutral shared OpenAPI and docs | Selected authn profile -> generated contract/docs | Current scheme says `bearerFormat: JWT`; current doc is JWT-only | `api/openapi/service.yaml` for security; `docs/authentication.md` for operator contract; initializer for profile pruning | Make the common scheme token-format neutral; partition doc/config examples by common and concrete profile markers | OpenAPI remains source of generated bindings; docs never carry credential values | None removes the full authn material; selected output retains no unselected profile text | OpenAPI drift check and initializer absence checks | Reopen Specification if handler-visible security declarations change |

## Files

### New shared bearer runtime

| path | responsibilities | one present reason to exist | declarations/visibility | call-path role | lifecycle/error ownership | allowed dependencies | forbidden responsibilities |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `internal/infra/bearerauthn/doc.go` | Package boundary | Make the shared/non-shared split reviewable | Package documentation only | None | Names ownership and exclusions | None | Executable behavior |
| `internal/infra/bearerauthn/runtime.go` | Verifier contract, verified result, runtime construction/delegated close, shared skew/token bounds | One composition point shared by HTTP and gRPC methods | Export only `Verifier`, `Result`, `Runtime`, constructor, and runtime methods bootstrap consumes | Concrete verifier -> runtime -> transport adapters | Delegates idempotent concrete close; unknown verifier failures fail closed | `context`, `time`, `reqctx`, metric provider | Provider/config/profile registry/cache policy |
| `internal/infra/bearerauthn/bearer.go` | Exact header/metadata bearer grammar and size admission | Carrier parsing is shared before either provider | Package-private parser | Resolver/interceptor -> verifier | Produces only shared missing/malformed/oversize errors | Stdlib | JWT or provider response parsing |
| `internal/infra/bearerauthn/errors.go` | Closed sanitized Kind/Error taxonomy and reason vocabulary | Both transports and both engines need one category identity | Export kinds, safe constructor/inspection only as required across packages | Engines/runtime -> transport renderers/metrics | Never wraps raw causes | Stdlib errors | HTTP bodies, gRPC status, provider text |
| `internal/infra/bearerauthn/metrics.go` | `authn.verifications` | Verification signal is common; JWKS refresh is not | Package-private recorder | Runtime records one terminal outcome | Bounded transport/result/reason only | OTel metric | Provider-specific labels/logs/traces |
| `internal/infra/bearerauthn/http.go` | OpenAPI Bearer scheme check, immediate header removal, resolver | HTTP seam is shared but distinct from gRPC mechanics | Export `Runtime.ResolveHTTP`; helpers private | OpenAPI validator -> verifier result -> `httpx.Authenticated` | Shared errors only | OpenAPI validation, `net/http`, `reqctx` | Routing, handler policy, provider calls |
| `internal/infra/bearerauthn/grpc.go` | Exact public Check rule, metadata removal, unary/stream interceptors, status mapping, stream expiry/I/O checks | Native gRPC has independent carrier/lifecycle mechanics | Export runtime interceptor constructors only | gRPC policy chain -> verifier -> principal/deadline -> handler | Maps shared/context failures; no raw cause | gRPC, `reqctx`, stdlib | Provider calls, feature status mapping |

Proof files are owned beside that package as
`bearer_test.go`, `errors_test.go`, `http_test.go`, `grpc_test.go`, and
`grpc_tls_contract_test.go`. The existing OIDC HTTP/gRPC contract assertions
move to these owners; later Test Design selects their exact scenario matrix.

### New concrete introspection engine

| path | responsibilities | one present reason to exist | declarations/visibility | call-path role | lifecycle/error ownership | allowed dependencies | forbidden responsibilities |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `internal/infra/oauthintrospection/doc.go` | RFC 7662 package contract and privacy exclusions | The selector name is OIDC-symmetric but the protocol owner is OAuth introspection | Package documentation only | None | Names no-cache/no-retry/no-authorization scope | None | Executable behavior |
| `internal/infra/oauthintrospection/policy.go` | Immutable validated issuer/audience/endpoint/target/credential policy | Configuration parity must be rechecked at adapter construction without importing config | Export one complete `PolicyInput` and opaque `Policy` construction surface | Bootstrap config -> concrete verifier | Validation errors name fields/reasons, never values | `authntrust`, stdlib | I/O, caching, transport adapters |
| `internal/infra/oauthintrospection/verifier.go` | Fixed client construction with selected `ResponseLimits`, canonical request, five-second attempt, decoded-body bound, context classification, idempotent close | One file owns the entire stateless provider-call lifecycle | Export `Verifier`, `New`, shared verifier method, `Close`; request helpers private | Shared runtime -> one provider POST -> response admission | Converts every raw provider failure to shared unavailable/context error | `bearerauthn`, `httpclient`, stdlib | Retry, limiter, queue, cache, discovery, logs/spans |
| `internal/infra/oauthintrospection/response.go` | Strict bounded JSON member admission, NumericDate, issuer/audience/identity checks, principal/expiry | Parsing is a pure independent responsibility and the only non-trivial logic fork | Package-private document/member types and parser | Provider 200 body -> shared result | Returns only shared invalid/unavailable; discards raw fields | `encoding/json`, numeric/time stdlib, `bearerauthn`, `reqctx` | HTTP I/O, config, authorization, provider extensions |

Proof files are owned as `policy_test.go`, `response_test.go`,
`verifier_test.go`, `harness_test.go`, and `docs_test.go`. The TLS fixture stays
unexported in this package; no shared test utility is created.

### Existing Go files with semantic ownership changes

| path | responsibilities | one present reason to exist | declarations/visibility | call-path role | lifecycle/error ownership | allowed dependencies | forbidden responsibilities |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `internal/infra/oidcjwt/verifier.go` | JWT engine implementing shared verifier contract | Retain JWT/JWKS lifecycle after transport extraction | Existing `Verifier/New/Close`; verifier method conforms to `bearerauthn.Verifier` | Bearer runtime -> JWT parser/JWKS | Shared invalid/unavailable/context errors; JWT close unchanged | `bearerauthn`, JWT/JWKS libs | HTTP/gRPC adapters or shared metrics |
| `internal/infra/oidcjwt/token.go` | JWT claims and principal normalization only | Bearer carrier moves to common owner | Package-private claims/parser helpers | JWT verifier core | Shared errors | JWT and `reqctx` | Header/metadata parsing |
| `internal/infra/oidcjwt/metrics.go` | JWKS refresh failure counter only | Refresh is JWT-specific after verification metric moves | Package-private refresh recorder | JWKS refresh callback | Sanitized no-label signal | OTel metric | General verification outcomes |
| `internal/infra/oidcjwt/trust_envelope.go` | JWT algorithm/provider refresh bounds | Shared 32 KiB/skew move out; remaining constants are JWT-specific | Existing constants narrowed | JWT provider/parser | No lifecycle | Stdlib time | Introspection policy |
| `internal/infra/oidcjwt/doc.go` | JWT engine-only package contract | Package no longer owns transports | Documentation only | None | Documents delegated shell | None | Shared handler contract |
| `internal/infra/oidcjwt/provider.go` | Discovery/JWKS fixed-provider access | Constructor/error imports follow the extracted shared taxonomy and the accepted current client or equivalent refreshed surface | Existing private helpers | JWT startup/refresh | Sanitized shared unavailable at engine boundary | `authntrust`, `httpclient` | Introspection or shared transport logic |
| `internal/infra/httpclient/client.go` | Existing fixed-authority client plus caller-supplied response-header guards | Only this owner can set header guards on its encapsulated cloned transport without duplicating the DNS/authority/proxy boundary | Add exported `ResponseLimits` and two `WithLimits` constructors; keep current constructors/`Client` methods | Concrete provider construction -> existing safe `Do` | Validates positive limits; pool close unchanged | Stdlib and existing outbound trust/correlation leaves | Provider-selected values, response bodies, retries, telemetry, authn errors |
| `internal/infra/httpclient/client_test.go` | Transport-level response-header limit contract | The shared owner needs one real enforcement proof independent of introspection semantics | Test-only assertions; no exported fixture | Bounded constructor -> TLS response headers -> deterministic rejection/timeout | Test cleanup closes server/client | Stdlib test/TLS support | Provider request, authn response, retry policy |
| `internal/authntrust/doc.go` | Shared pure authn trust-leaf definition | Leaf now serves both concrete engines | Documentation only | Config/engines | None | None | Runtime state |
| `internal/authntrust/introspection.go` | Pure endpoint and target-class rules | One answer must be shared without config importing an adapter | Export narrow constants/predicates | Config and introspection policy construction | No errors with values | Stdlib URL/string | DNS, dialing, credentials |
| `internal/config/authn_config.go` | Source-template superset schema plus JWT-default validation and marked leaves | Section type/default/validation remain one config owner | Existing `AuthnConfig` surface; introspection leaves marked | Load -> immutable snapshot | Field-named validation only | `authntrust` | Adapter construction |
| `cmd/service/internal/bootstrap/startup_authn.go` | Consumer interface and startup stages | Shared composition seam should not be duplicated in profile templates | Existing unexported interface/stages only | Runtime wiring/tests | No concrete close implementation | Config/telemetry/transport types required by seam | Concrete provider construction |
| `cmd/service/internal/bootstrap/startup_authn_profile.go` | Source-template JWT concrete constructor | Exactly one small file is replaced for introspection output | Unexported `initAuthn` | Config -> JWT engine -> bearer runtime | Transfers ownership only after complete runtime construction | `bearerauthn`, `oidcjwt`, config, telemetry | Runtime selection/factory |
| `cmd/service/internal/bootstrap/run.go` | Common authn construction/use/close and common marker ownership | Existing lifecycle is canonical | Existing unexported wiring/stages | Process composition | Deferred and ordered close | Bootstrap owners | Provider mechanics |
| `internal/infra/http/request_errors.go` | HTTP mapping from shared authn kinds | Current mapper must stop importing a concrete engine | No new exports | Resolver error -> Problem response | Transport-owned sanitized response | `bearerauthn`, problem/failure | Provider inspection |

Delete `internal/infra/oidcjwt/errors.go`, `http.go`, and `grpc.go` after their
responsibilities move. Delete their provider-specific HTTP/gRPC test files only
after the equivalent assertions exist under `bearerauthn`; JWT verifier,
provider, policy, documentation, and harness tests remain in `oidcjwt`.

### Generated-control-only Go edits

Each path below changes only profile ownership/comments/import markers, not its
runtime decision. Its allowed dependencies and forbidden responsibilities stay
as documented by the current package.

| path | present reason |
| --- | --- |
| `internal/config/types.go` | Rename the common authn section marker from JWT-only to `authn-bearer`. |
| `internal/config/defaults.go` | Keep the selected authn defaults merge under the common marker. |
| `internal/config/validate.go` | Keep selected authn validation invocation under the common marker. |
| `internal/config/configtest/configtest.go` | Keep common authn fixture admission while profile-specific values are separately marked. |
| `internal/config/grpc_config.go` | Preserve current authn/gRPC cross-validation under the common marker. |
| `internal/config/http_config.go` | Preserve current positive HTTP admission requirement under the common marker. |
| `cmd/service/internal/bootstrap/startup_http.go` | Rename shared HTTP authn wiring markers. |
| `cmd/service/internal/bootstrap/startup_grpc.go` | Rename shared gRPC authn wiring markers. |
| `internal/failure/failure.go` | Retain authentication-neutral failure vocabulary under the common marker. |
| `internal/problem/problem.go` | Retain current authn Problem catalog entries under the common marker. |
| `internal/infra/grpc/health_method.go` | Keep the exact public Check ownership comment under the common marker. |
| `internal/infra/grpc/status.go` | Keep policy-status trust comments/coverage under the common marker. |

Their existing tests change only where initializer-marker coverage or the
shared import path requires it. `internal/config/authn_config_test.go`,
`snapshot_contract_test.go`, `cmd/service/internal/bootstrap/authn_bootstrap_test.go`,
`internal/infra/http/request_errors_test.go`, and `authn_router_test.go` remain
the exact package-local proof locations.

## Import and composition graph

```text
internal/config -> internal/authntrust

cmd/service/internal/bootstrap
  -> internal/config
  -> internal/infra/bearerauthn
  -> exactly one of internal/infra/oidcjwt or internal/infra/oauthintrospection

internal/infra/oidcjwt
  -> internal/infra/bearerauthn
  -> internal/authntrust
  -> internal/infra/httpclient

internal/infra/oauthintrospection
  -> internal/infra/bearerauthn
  -> internal/authntrust
  -> internal/infra/httpclient

internal/infra/bearerauthn
  -> internal/reqctx
  -> OpenAPI validation and gRPC only in their owned adapter files

internal/infra/http -> internal/infra/bearerauthn (error inspection only)
```

No concrete engine imports config, bootstrap, HTTP server, native gRPC server,
features, or the other concrete engine. `bearerauthn` imports neither concrete
engine, so the graph is acyclic.

## Non-Go and generated/manual authority map

| path | owner action | generated/manual boundary |
| --- | --- | --- |
| `scripts/profiles/authn-oidc-introspection/authn_config.go.tmpl` | Add the complete introspection-only config type/default/validator replacement | Upstream static initializer input; removed from every generated checkout |
| `scripts/profiles/authn-oidc-introspection/startup_authn_profile.go.tmpl` | Add the introspection concrete constructor replacement | Upstream static initializer input; writes canonical bootstrap file, then is removed |
| `scripts/init-module.sh` | Accept three choices, apply common/concrete markers, copy replacements, remove unselected packages/files, write exact lock | Canonical generator writer |
| `scripts/ci/template-init-check.sh` | Prove three choices, gRPC combinations, dependency absence, marker absence, lock readback, repeated-init stability | Canonical generated-output proof; no production behavior |
| `scripts/ci/runtime-image-build.sh` and `Makefile` | Extend only the existing authn fixture selection where runtime-image coverage requires the new generated profile | Build/proof orchestration, not authn policy |
| `.golangci.yml` | Replace JWT transport exceptions with shared-shell/provider-engine dependency rules and profile-correct `PolicyInput` coverage | Mechanical enforcement of the decided graph |
| `api/openapi/service.yaml` | Keep one neutral Bearer scheme and public health overrides under common marker | OpenAPI source; regenerate `internal/openapi` rather than edit it |
| `env/config/local.yaml` and `env/.env.example` | Partition non-secret examples and environment-only client secret input by profile markers | Input examples only; never credential authority |
| `docs/authentication.md`, `README.md` | Document common bearer contract and selected concrete profile without retained unselected text | Manual operator authority |
| `docs/architecture/boundaries.md` | Record the shared bearer/concrete-engine split and that `httpclient` enforces caller-supplied header guards without choosing provider budgets or parsing bodies | Repository architecture authority |
| `go.mod`, `go.sum` | Add nothing for introspection; initializer `go mod tidy` removes JWT/JWKS modules from introspection output | Module graph is derived from retained Go source |

## Cleanup and phase boundary

The new common package replaces, rather than wraps, current duplicated owner
locations. Old `oidcjwt` transport/error declarations and stale marker names
must not survive. No compatibility alias is kept because all callers are
inside this repository and initializer output is generated atomically.

Go Ownership Review is triggered because the change moves responsibilities
across several package and generated-source boundaries. Test Design is the next
macro phase after a passing Technical Design Review; it alone selects the proof
matrix. Planning and implementation remain unauthorized.

Reopen System / Integration Design if this placement cannot preserve the
selected one-call mechanism or authority map. Reopen Specification only under
`../spec.md`'s explicit conditions. Reopen this ownership design if the current
composition root, package docs, depguard rules, or profile generator changes so
the import or generated/manual map above is no longer realizable.
