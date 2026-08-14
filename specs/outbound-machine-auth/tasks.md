# Outbound machine authentication implementation ledger

status: implementation complete; verification incomplete

Completion: the portable outbound OAuth client-credentials capability satisfies R1-R10, every TD-001-TD-016 scenario and aggregate gate passes on the current tree and generated local outputs, and `OUTBOUND_AUTH=none|oauth2-client-credentials` outputs are deterministic and dependency-clean. Completion makes no provider, deployment, credential-rotation, capacity, criticality, long-stream continuity, or production-readiness claim.

Blocked stop: stop deterministic implementation and reopen Go Ownership/Test Design if forced fallback roots disappear, a supported runner cannot supply the required bindable non-loopback private address, or process-global trust/resolver use prevents test-binary isolation. Stop at the narrow owner named below when a deterministic gate cannot preserve an accepted R1-R10 behavior, source/placement decision, proof path, or cleanup boundary; an unavailable external provider or deployment input does not block this portable ledger and cannot be converted into a local claim.

Global constraints:

- Preserve R1-R10 as one portable capability: exactly one immutable dependency binding; environment-only client secret; explicit `client_secret_basic`; fixed token and resource HTTPS authorities; strict bounded token request/response; one process-local on-demand token; caller-independent provider budget; coalesced success/failure waves; closed sanitized failures and telemetry; optional readiness-neutral startup; and authenticated HTTP/gRPC consumption without feature-visible credentials, refresh, discovery, provider fallback, resource replay, or a second transport policy.
- Use the standard library plus the existing `internal/infra/httpclient`; add no direct OAuth module, SDK, token-source interface, factory, registry, persistent cache, background refresh loop, readiness probe, custom trust store, proxy path, or generated OAuth authority. This capability leaves `go.mod` and `go.sum` unchanged; a transitive OAuth module retained by an independently selected owner is not outbound-auth ownership.
- Keep `NewHTTPClient(*Client, *httpclient.Client)` exact. The shared `httpclient` attempt-authorization seam, its leaf proof, OAuth's sole production caller, the generated-client subprocess branch, and the composed TD-008/TD-009 proof are one indivisible T4 placement; no intermediate candidate may expose the seam without that caller and proof.
- TD-008/TD-009 cross the real `httpclient` retry, attempt authorization, generic OTel, fixed authority, private-address-at-dial, TLS hostname/chain, response bound, redirect, and response paths. Tests may change only process-local DNS and fallback-root inputs; they add no alternate Doer, dialer, TLS configuration, response path, retry path, exported test seam, `unsafe`, or `go:linkname` path.
- The generated-client parent remains `internal/infra/httpclient/generated_client_test.go`; its child uses only public constructors and environment-carried fixture addresses/CA path, installs its own resolver/fallback root, injects the authenticated Doer through `WithHTTPClient`, and owns its process cleanup. The compiled parent never imports OAuth.
- Register cleanup immediately for every listener, DNS socket, server, response body, connection, owner, idle pool, meter reader, goroutine gate, temporary CA file/tree, and child process. Release held work before `Client.Close`; let the owner close the token idle pool; close the resource pool; stop and join servers/DNS; restore `net.DefaultResolver`; and never run resolver-mutating tests in parallel. Each package test binary owns at most one fallback-root installation.
- Focused tests prove only portable behavior on their exercised current-tree path. Local TLS/address fixtures do not certify real DNS, CA installation, firewall, proxy, provider, credential issuance/rotation, deployment, quota, replica, latency, criticality, or stream continuity. Those claims require their named external owners and separately authorized real-path proof.
- No planned parallel wave is recorded: T4-R0 is a distinct singleton Local acceptance unit because it composes the same `httpclient` surfaces as S3 D2/T7/T9A; raw Worktree fan-in is forbidden. T3, T4-R0, and T5 otherwise remain serial where their OAuth harness, lifecycle, or cleanup ownership overlaps.

## Obligation reconciliation

| Test Design obligation | Ledger disposition and proof owner |
| --- | --- |
| TD-001 | T1 owns static config/runtime admission, the bootstrap-owned pure mapping/parity corpus, and secret custody; T3 reruns the exact command after runtime composition. |
| TD-002 | T1 owns the closed vocabulary/error table; T3 owns startup/close sanitization and reruns the exact cross-package command. |
| TD-003 | T1 owns `httpclient` public/private HTTPS policy and the token-client disclosure boundary. |
| TD-004 | T1 owns the exact Basic form transaction and strict bounded response table. |
| TD-005 | T2 owns on-demand cache/restart and operation-token state; T4 and T5 own their HTTP/gRPC attempt-margin deltas, with exact adapter commands attached there. |
| TD-006 | T2 owns the event-gated success/failure waves, caller cancellation, fail-fast window, recovery, and race proof. |
| TD-007 | T1 owns the closed provider response/fault classification table; T2 owns caller-independent provider lifetime and timeout. |
| TD-008 | T4 owns the concrete HTTP component and generated-client subprocess proof; T4-R0 continues it on Local after preserving S3's generic transport policy. It remains inseparable from TD-009 seam/caller placement. |
| TD-009 | T4 owns the `httpclient` leaf proof, OAuth sole caller, and real composed retry proof; T4-R0 re-establishes that one acceptance boundary on the current Local composition. |
| TD-010 | T5 owns real-TLS gRPC application RPC/stream creation, authority, cancellation, competing metadata, transport security, and downstream-status proof. |
| TD-011 | T5 owns the raw TLS/HTTP2 transparent-attempt falsifier against pinned grpc-go. |
| TD-012 | T5 owns health/control reconnect and long-lived-stream no-reauth proof. |
| TD-013 | T2 owns the private operation-local `provider_unavailable` receipt; T3 consumes it and owns local-only startup, unchanged health/probe inventory, optional degradation, invalid-config admission, and the exact composed rerun. |
| TD-014 | T2 owns credential-owner retirement/join; T3 owns ordered bootstrap close, partial-start cleanup, exactly-once error propagation, and the combined race command. |
| TD-015 | T6 owns the exhaustive reachable telemetry/disclosure matrix after T3-T5 make every signal path present. |
| TD-016 | T7 owns selector validation, retention/stripping, generated authority, module attribution, idempotence, docs, and all aggregate gates. |

External proof boundaries are scope exits under `spec.md` Scope and non-goals: named-provider compatibility, deployed network/TLS, credential issuance/rotation, critical dependency policy, fleet capacity, and long-lived-stream continuity are not portable implementation tasks. Their provider/security, deployment/network, service/SLO, capacity, or concrete RPC owners may reopen the exact Specification/Design rule and add separately authorized proof; their absence is not completion evidence and does not weaken this ledger.

## Readiness review

The prior focused review at SHA-256
`8e6a2f410795dc25aca99d47f586288c208156150020f7c3a0aa00c4008775af` is
superseded: it incorrectly treated the original Worktree T4 Lead as a Local
continuation. Its proof and ownership findings remain historical only; it is not
movement evidence for T4-R0. Fresh review below must dry-run the distinct
fresh-Local acceptance unit and its carrier basis.

Focused independent Task Review / Readiness returned **PASS** on candidate
SHA-256 `08dd459c62f99705621f1804dce38dabe17cc688eb44c4976009668657ff914d`.
It found T4-R0 executable as one fresh singleton Local
`ACCEPTANCE_UNIT_LEAD` with the recorded `fresh-local/attempt-1` basis,
dispatch-time Local preflight, bounded writable/resources surface, exact proof,
and T7 release only through its new receipt. The original Worktree T4 Lead and
tree remain read-only predecessor/provenance, without Handoff, fan-in,
replacement, overwrite, or old-receipt reuse. Current Local dirt remains a
dispatch-time attribution gate. This receipt changes no task outcome,
dependency, proof, owner, or reopen condition.

Independent Task Review / Readiness returned `CONCERNS` on candidate SHA-256
`40934cbb96aa053815598ad4212e61fc0830a619e84bc7ccd2cb6f6ea680122c`.
The three prior blockers were falsified: T1's full config/HTTP regression gate
was executable, T2 accepted the private operation-token margin oracle before
T4/T5, and the focused-QA-approved TD-013 split needed no new seam. The one
bounded concern was dispositioned above and in T3's dependency/handoff: T2's
operation-local outage proof is an explicit consumed receipt for T3's composed
health/startup proof. Its observable is the exact T2/T3 command and its carrier
change reopens Test Design. This reconciliation-only disposition changes no
behavior, task boundary, proof policy, or reviewed execution path and requires
no fresh review under the shared convergence contract. The review proves
ledger readiness only, not implementation, provider compatibility, deployment,
or production readiness.

- [x] T1: Static configuration and one exact Basic grant fail closed before credential disclosure and expose only the accepted sanitized contract
  - Source: `spec.md` R2-R4/R7 and success criteria 2-3/8; `design/overview.md` Authority and configuration, Token-endpoint admission and transaction, Failure contract and telemetry, Go responsibility map, and inverse file map; `test-plan.md` TD-001-TD-004 plus the classification half of TD-007.
  - Owner/surface/resources: add `internal/infra/oauth2clientcredentials/doc.go`, `config.go`, `vocabulary.go`, `errors.go`, `provider.go`, `config_test.go`, `vocabulary_test.go`, `errors_test.go`, `provider_test.go`, and the initially required canary/provider fixtures in `harness_test.go`; add `internal/config/outbound_auth_config.go` and `_test.go`; change `internal/config/types.go`, `defaults.go`, `validate.go`, `load_environment_test.go`, `secret_policy_test.go`, and `snapshot_contract_test.go`; add the pure config-mapping/parity portion of `cmd/service/internal/bootstrap/startup_outbound_auth.go` and `_test.go`, the sole owner allowed to import both config representations, for immediate shared-corpus proof before runtime construction; change `internal/infra/httpclient/config.go`, `target_policy.go`, and reached tests, splitting current policy/fixture ownership into `config_test.go`, `target_policy_test.go`, `transport_test.go`, and `harness_test.go` as fixed by the inverse map. Preserve unrelated pre-existing `httpclient` behavior and tests; scripted/recording transports only, with no credential, live provider, proxy, custom root, or network mutation.
  - Depends on: none.
  - Proof: config/runtime admission and secret custody reject every zero/incomplete/contradictory/duplicate/over-bound/unknown source before I/O, preserve exact secret bytes only from `APP__OUTBOUND_AUTH__CLIENT_SECRET`, map valid values byte-for-byte, and never let config admit what runtime rejects or leak a canary. Run `go test -vet=off ./internal/config ./internal/infra/oauth2clientcredentials ./cmd/service/internal/bootstrap -run '^(TestOutboundAuthConfigContract|TestOutboundAuthSecretSourcePolicy|TestOutboundAuthConfigParity|TestFailureVocabularyAndPrecedence|TestAuthErrorsAreSanitized)$' -count=1`.
  - Proof: external/private HTTPS policy and the dedicated token client admit only the configured authority with the exact bounds and prevent any alternate scheme, suffix, address class, proxy, redirect, or authority from receiving Basic/token material. Run `go test -vet=off ./internal/infra/httpclient ./internal/infra/oauth2clientcredentials -run '^(TestPrivateHTTPSTargetPolicy|TestTokenEndpointHTTPPolicy|TestTokenEndpointAdmissionPreventsCredentialDisclosure)$' -count=1`.
  - Proof: the provider sends one exact Basic/form request, performs no retry, publishes only a valid exact-200 JSON Bearer with a usable bounded lifetime, classifies the complete non-200/fault partition, and returns no raw provider content. Run `go test -vet=off ./internal/infra/oauth2clientcredentials -run '^(TestProviderGrantRequestAndStrictResponse|TestProviderFailureClassificationIsClosed)$' -count=1`.
  - Proof: preserve every reached current config and bounded-HTTP behavior while adding and splitting owners: defaults/snapshot/source policy, invalid config, construction, fixed authority, pre/post-DNS policy, retry, propagation, redirect, response limits, TLS defaults, connection reuse, and generated-client composition all remain green. Run `go test -vet=off ./internal/config ./internal/infra/httpclient -count=1` before accepting T1.
  - Reopen if: a second dependency, principal, config/secret source, auth method, runtime discovery, proxy/plaintext route, provider request/response rule, failure class/precedence, or raw diagnostic owner is required — Specification; config/runtime mapping, target policy, strict-decoder ownership, or package/file placement changes — Technical Design/Go Ownership; a maintained library changes the portable responsibility boundary or Basic interoperability floor — Research/Specification.
  - Accepted: T1; evidence: all four recorded T1 proof commands passed on stable HEAD 25b2a31f78c91f24667f8d6b477cfff958b37aa5 and fresh independent implementation review PASS; candidate: current bounded diff

- [x] T2: One process credential owner reuses and renews on demand, coalesces caller-independent provider work, suppresses failure waves, and retires cleanly
  - Source: `spec.md` R5-R8 and success criteria 4/7; `design/overview.md` Token lifecycle, cancellation, failure isolation, Retirement, telemetry, performance, and shutdown; `test-plan.md` TD-005-TD-007, the OAuth owner half of TD-013, and the owner half of TD-014.
  - Owner/surface/resources: add `internal/infra/oauth2clientcredentials/client.go`, `telemetry.go`, `client_test.go`, `telemetry_test.go`, and `goleak_test.go`; extend only the design-owned shared fixtures in `harness_test.go`. The public surface is concrete `New(Config, metric.MeterProvider, *slog.Logger) (*Client, error)` and `(*Client).Close(context.Context) error`; clock/provider construction remains package-private. Mutable resources are one process context, one mutex/state machine, one provider request, one result channel, one token, deterministic clock, owned event gates, `testing/synctest`, and the package leak gate.
  - Depends on: T1 — accepted runtime config, token transaction, sanitized errors, target policy, and closed vocabulary needed to start.
  - Handoff: T1 produces one validated runtime Config/provider transaction and closed failure contract; T2 consumes them as the only acquisition and publication path.
  - Proof: construction/idle/restart, reuse, renewal at the ten-second margin, failed replacement, and the private operation token's no-renewal margin check match R5 without eager I/O or retained state. Run `go test -vet=off ./internal/infra/oauth2clientcredentials -run '^(TestTokenReuseRenewalAndRestart|TestOperationTokenCannotRenewAcrossExpiryMargin)$' -count=1`.
  - Proof: one success/failure wave preserves per-caller cancellation, fail-fast suppression, exact-boundary recovery, and caller-independent provider lifetime/timeout. Run `go test -vet=off ./internal/infra/oauth2clientcredentials -run '^(TestAcquisitionWavesPreserveCallerCancellation|TestProviderWorkOutlivesCallersWithinItsBudget)$' -count=1` and `go test -vet=off -race ./internal/infra/oauth2clientcredentials -run '^TestAcquisitionWavesPreserveCallerCancellation$' -count=10`.
  - Proof: Close retires admission, cancels and joins one provider wave, clears the token, closes the token idle pool once, returns only the closed timeout class, and leaves no goroutine; repeat Close is harmless. Run `go test -vet=off ./internal/infra/oauth2clientcredentials -run '^TestClientCloseRetiresAndJoinsAcquisition$' -count=1` and `go test -vet=off -race ./internal/infra/oauth2clientcredentials -run '^TestClientCloseRetiresAndJoinsAcquisition$' -count=10`.
  - Proof: one real private resolution can fail as exact `provider_unavailable` while changing only the operation error and accepted provider attempt/result signals; no adapter or exported token operation is involved. Run `go test -vet=off ./internal/infra/oauth2clientcredentials -run '^TestOutboundAuthProviderOutageIsOperationLocal$' -count=1`.
  - Reopen if: expiry/margin, cache, retry, failure-window, caller/provider cancellation, shutdown, revocation, rotation, or token-lifetime behavior changes — Specification; one result channel, one mutex, process-owned context, deterministic time, or owner-mediated cleanup cannot preserve the oracle — Technical Design; representative measurement shows the mutex or per-process grant ceiling violates an accepted budget — Technical Design/Performance.
  - Accepted: T2; evidence: all six recorded T2 proof commands passed on the current bounded diff over stable HEAD 25b2a31f78c91f24667f8d6b477cfff958b37aa5 and fresh independent implementation review PASS; candidate: current bounded diff

- [x] T3: Valid optional outbound auth is constructed without provider I/O and closes once in the existing startup, drain, background, dependency, and telemetry order
  - Source: `spec.md` R1/R2/R7/R8 and success criteria 1/7/9; `design/overview.md` Runtime policy parity, Startup and shutdown composition, Health/degradation, and bootstrap inverse map; `test-plan.md` TD-001/TD-002, the bootstrap half of TD-013, and bootstrap half of TD-014.
  - Owner/surface/resources: extend T1's `cmd/service/internal/bootstrap/startup_outbound_auth.go` and `_test.go` from the accepted pure mapping into local construction; change `cmd/service/internal/bootstrap/run.go` and `run_lifecycle_test.go`; extend only reached existing config/bootstrap tests. The stage constructs the token client/owner without I/O, registers immediate partial-start cleanup, adds no readiness probe, and joins one sanitized close result. Mutable resources are existing startup/shutdown contexts and the bootstrap lifecycle event recorder only.
  - Depends on: T2 — accepted concrete owner/New/Close, operation-local `provider_unavailable` receipt, and complete local lifecycle needed to start and prove.
  - Handoff: T2 produces one no-I/O constructor, idempotent bounded Close, and the accepted private operation-local outage receipt; T3 consumes them in the existing composition/shutdown order and exact TD-013 composed rerun.
  - Proof: exact config-to-runtime parity holds and startup/close errors retain only the closed class/text. Run `go test -vet=off ./internal/config ./internal/infra/oauth2clientcredentials ./cmd/service/internal/bootstrap -run '^(TestOutboundAuthConfigContract|TestOutboundAuthSecretSourcePolicy|TestOutboundAuthConfigParity|TestFailureVocabularyAndPrecedence|TestAuthErrorsAreSanitized|TestOutboundAuthStartupAndCloseErrorsAreSanitized)$' -count=1`.
  - Proof: compose T2's real operation-local outage with bootstrap's structural non-participation: valid selected config starts with zero provider I/O and no auth readiness participant; invalid config fails before listener mutation; repeated health reads call no auth/provider collaborator; liveness and the existing single drain transition remain repository-owned. Run `go test -vet=off ./internal/infra/oauth2clientcredentials ./cmd/service/internal/bootstrap -run '^(TestOutboundAuthProviderOutageIsOperationLocal|TestOutboundAuthStartupIsLocalOnly|TestOutboundAuthOutageDoesNotChangeHealth|TestOutboundAuthInvalidConfigFailsBeforeServing)$' -count=1`.
  - Proof: normal and partial-start paths close after admitted/background users and resource clients but before other dependencies/telemetry, propagate a close deadline exactly once, and preserve the current shutdown budget. Run `go test -vet=off ./internal/infra/oauth2clientcredentials ./cmd/service/internal/bootstrap -run '^(TestClientCloseRetiresAndJoinsAcquisition|TestOutboundAuthRuntimeCloseOrder|TestOutboundAuthCloseFailureIsJoinedOnce)$' -count=1` and `go test -vet=off -race ./internal/infra/oauth2clientcredentials ./cmd/service/internal/bootstrap -run '^(TestClientCloseRetiresAndJoinsAcquisition|TestOutboundAuthRuntimeCloseOrder)$' -count=10`.
  - Reopen if: the dependency becomes critical, startup/readiness must contact the provider, another dependency/config source is required, or shutdown semantics change — Specification; config parity, bootstrap stage, lifecycle order, no-probe composition, or exactly-once cleanup cannot hold — Technical Design/Go Ownership.
  - Accepted: T3; evidence: all four recorded T3 proof commands and the focused bootstrap regression gate passed on the current bounded diff; fresh independent implementation review PASS; candidate: current bounded diff

- [x] T4: HTTP generated and concrete clients attach one operation-fixed token through the real retry path, and the coupled shared seam has no other production caller
  - Source: `spec.md` R2/R5-R7/R9 and success criteria 4-5/8-9; `design/overview.md` HTTP attachment, Deterministic HTTP proof carrier, exact shared surface, and inverse file map; `test-plan.md` TD-005, TD-008, and TD-009.
  - Owner/surface/resources: add `internal/infra/httpclient/attempt_authorization.go` and `_test.go`; change `internal/infra/httpclient/client.go`, `retry.go`, `retry_test.go`, `generated_client_test.go`, and the reached split test/harness files; move `authn_policy_test.go` to `credential_provider_contract_test.go` with the union-owned credential-provider contract; add `internal/infra/oauth2clientcredentials/http.go` and `http_test.go`; extend only the single OAuth `harness_test.go`/`goleak_test.go` owner. This one task owns `AttemptAuthorizer`, `DoWithAuthorization`, the private nonretryable wrapper, OAuth's sole production call, `NewHTTPClient(*Client, *httpclient.Client)`, TD-008/TD-009 leaf/composed proof, and the marked generated subprocess branch. Mutable resources are one fallback-root installation per OAuth/child test binary, one temporary resolver/DNS socket, one bindable non-loopback private IPv4 address, controlled TLS token/resource servers, attempt gates, temporary CA/tree, and child process; resolver-mutating tests are serial.
  - Depends on: T2 — accepted credential owner and immutable operation token needed to start; T1 — accepted `PrivateHTTPS`, fixed token/resource authority, and response bounds needed to prove.
  - Handoff: T2 produces one operation-fixed token and margin check; T1 produces real bounded HTTPS authorities; T4 consumes both inside each existing HTTP retry attempt without reacquisition.
  - Proof: the attempt authorizer runs on a clone inside every real retry, preserves ordinary retry eligibility/count, becomes nonretryable on failure, and keeps the bearer outside generic OTel. Run `go test -vet=off ./internal/infra/httpclient -run '^TestAttemptAuthorizationPreservesRetryPolicy$' -count=1`.
  - Proof: through public concrete construction and the real authority/TLS/response path, constructor mismatch and caller Authorization fail before acquisition/I/O; cancellation stops only its wait; token failure reaches no resource; one bearer reaches only the fixed authority; 401/403 remain unchanged with one request/no replay; and the isolated generated child consumes the authenticated Doer. Run `go test -vet=off ./internal/infra/oauth2clientcredentials ./internal/infra/httpclient -run '^(TestHTTPClientResourceAuthorityIsFixed|TestHTTPClientAttachesOneOperationToken|TestHTTPClientRejectsCallerAuthorization|TestHTTPClientCallerCancellationStopsOnlyItsWait|TestHTTPClientPreservesDownstreamAuthResponses|TestGeneratedClientUsesAuthenticatedDoer)$' -count=1`.
  - Proof: permitted retries carry the same token and one provider grant; crossing the margin after attempt one returns exact `token_unusable` before a second resource attempt, retry, or acquisition. Run `go test -vet=off ./internal/infra/oauth2clientcredentials -run '^(TestOperationTokenCannotRenewAcrossExpiryMargin|TestHTTPRetryFixesOneTokenAndStopsAtMargin)$' -count=1`.
  - Reopen if: forced fallback roots disappear, a supported runner lacks the required private address, or process-global trust/resolver use prevents package-test isolation — Go Ownership/Test Design; fixed constructor, shared seam placement, generated subprocess ownership, or real transport composition cannot hold — Go Ownership/Technical Design; resource retry, header, cancellation, authority, or downstream semantics change — Specification.
  - Accepted: T4; evidence: all three recorded T4 proof commands, the touched-package regression, and the focused race gate passed on the current bounded diff; fresh independent implementation review PASS; candidate: current bounded diff

## T4 shared-surface serial continuation

The former formatting-only provenance blocker is invalidated by the reviewed
Go Ownership reconciliation at `design/overview.md` “T4 shared-HTTP provenance
reconciliation”: immutable tree `50e932807b9d730fc3ba67dc43ec9916f50ff4a0`
is the replacement T4 snapshot, but its Local delta is a substantive semantic
composition with S3 D2/T7/T9A, not a format-only continuation. T1
`c04257bbb08bd66fe4e11ec0b2a11ccc39abcd6b` and T2
`5b73e4ec258e246bebae267a18c57abfabc8677b` remain predecessor receipts. The
original Worktree T4 Lead `019ffa52-adaf-7fd0-8184-b939abd6f2bb`, its Worktree
candidate, and its immutable replacement snapshot remain predecessor/provenance
only until T4-R0 acceptance. T4-R0 cannot Handoff, consume that Worktree
candidate, replace or duplicate the original Lead, overwrite the snapshot, or
reuse the old T4 receipt.

- [x] T4-R0: A fresh Local acceptance unit composes T4 attempt authorization and generated-client proof with accepted S3 generic transport policy without overwriting either provenance source
  - Source: `design/overview.md` “T4 shared-HTTP provenance reconciliation”, Go responsibility map/inverse file map, and exact shared surface; `specs/s3-compatible-object-storage/design/overview.md` D2/D7 and Go responsibility map; `specs/s3-compatible-object-storage/tasks.md` T2, T7, and T9A accepted receipts; `test-plan.md` TD-005, TD-008, and TD-009.
  - Owner/surface/resources: a fresh Local `ACCEPTANCE_UNIT_LEAD` exclusively owns this distinct unit on the current Local candidate at dispatch. The writable shared surface is `internal/infra/httpclient/client.go`, `generated_client_test.go`, `retry.go`, and `retry_test.go`, plus T4's coupled `attempt_authorization.go`, `attempt_authorization_test.go`, `internal/infra/oauth2clientcredentials/http.go`, `http_test.go`, and their already-design-owned harness companions when a required proof carrier demands it. Preserve S3's one-attempt HTTP/1, caller-selected immutable roots, request-deadline propagation, ordinary `Do`, fixed-authority admission, retry eligibility, and TLS verification; preserve T4's context-carried cloned per-attempt authorization, retry-to-instrumentation-to-authorizer-to-fixed-transport order, nonretryable wrapper classification, OAuth sole production caller, and OAuth-free generated parent/child containment. Mutable resources are the one Local candidate, OAuth/child fallback-root and resolver fixtures, controlled TLS/DNS servers, temporary CA/tree, and child process; no Worktree, Handoff, cherry-pick, raw diff fan-in, original-T4 replacement, or candidate/snapshot overwrite is permitted.
  - Dispatch basis: this review-cleared ledger revision / `T4-R0` / `fresh-local` / `attempt-1`; create one fresh Local `ACCEPTANCE_UNIT_LEAD` only after native project and current Local HEAD/status/attributed-dirt preflight. It receives the current Local candidate as its starting state, the original Worktree Lead identity and tree `50e932807b9d730fc3ba67dc43ec9916f50ff4a0` as read-only predecessor/provenance, and no Handoff input. Its unit receipt, not the original T4 receipt, releases T7.
  - Depends on: T1 and T2 accepted predecessor receipts — validated authority, provider transaction, and immutable operation token needed to start; S3 T2/T7/T9A accepted receipts — one-attempt, explicit-root, and deadline policy already present in Local and needed to preserve and prove; replacement tree `50e932807b9d730fc3ba67dc43ec9916f50ff4a0` — T4 provenance boundary needed to compare, not overwrite.
  - Handoff: T4-R0 produces one Local-only composed T4 continuation whose bounded diff identifies each T4 authorization/proof delta and preserves each S3 generic-transport invariant. T7 consumes its fresh T4-R0 accepted receipt before global generated-tree and aggregate closure; T6 remains an accepted historical telemetry receipt and is rerun only if T4-R0 evidence reaches a telemetry path.
  - Proof: preserve the generic one-attempt, immutable-root, deadline, authority, and bounds policy with `go test -vet=off ./internal/infra/httpclient ./internal/infra/s3 -run '^(TestOneAttemptTransportDoesNotReplayOrTransform|TestTransportUsesCallerRootCAsWithoutAmbientFallback|TestTransportRefusesAlternateAuthority|TestTransportBoundsControlAndObjectBodies)$' -count=1` and `go test -vet=off ./internal/infra/httpclient ./internal/infra/s3 -run '^(TestOneAttemptTransportUsesRequestDeadlineAndExplicitRoots|TestEffectiveDeadlineAndLifecycleOwnEveryPhase)$' -count=1`.
  - Proof: the authorization seam remains cloned per real retry, nonretryable on authorizer failure, and outside generic OTel; the fixed OAuth operation uses one token without replay and the generated child consumes only the authenticated Doer. Run `go test -vet=off ./internal/infra/httpclient -run '^TestAttemptAuthorizationPreservesRetryPolicy$' -count=1`, `go test -vet=off ./internal/infra/oauth2clientcredentials ./internal/infra/httpclient -run '^(TestHTTPClientResourceAuthorityIsFixed|TestHTTPClientAttachesOneOperationToken|TestHTTPClientRejectsCallerAuthorization|TestHTTPClientCallerCancellationStopsOnlyItsWait|TestHTTPClientPreservesDownstreamAuthResponses|TestGeneratedClientUsesAuthenticatedDoer)$' -count=1`, and `go test -vet=off ./internal/infra/oauth2clientcredentials -run '^(TestOperationTokenCannotRenewAcrossExpiryMargin|TestHTTPRetryFixesOneTokenAndStopsAtMargin)$' -count=1` before accepting T4-R0.
  - Reopen if: Local cannot preserve both the exact T4 order/proof containment and S3's one-attempt, immutable-root, deadline, ordinary-`Do`, authority, retry, or TLS invariants — Go Ownership and Test Design; resource authority, retry, header, cancellation, or downstream-response behavior changes — Specification; any required shared path falls outside this fixed surface — Go Ownership.
  - Accepted: T4-R0; evidence: all five recorded T4-R0 proof commands passed (12, 2, 2, 12, and 4 tests), focused `go test -vet=off ./internal/infra/httpclient ./internal/infra/oauth2clientcredentials ./internal/infra/s3` passed (397), focused race passed (17), and fresh independent implementation review PASS; candidate: current bounded diff

- [x] T5: gRPC application and control streams use the same transport-secure operation credential without replay or in-place reauthentication
  - Source: `spec.md` R2/R5-R8/R10 and success criteria 4/6-9; `design/overview.md` gRPC attachment, grpc-go versioned authorities, exact surface, and inverse file map; `test-plan.md` TD-005 and TD-010-TD-012.
  - Owner/surface/resources: add `internal/infra/oauth2clientcredentials/grpc.go` and `grpc_test.go`; extend only design-owned shared OAuth fixtures. `internal/infra/grpcclient` adds one optional terminal-result observer around its existing stats handler; OAuth's complete constructor exclusively installs that observer and `Options.PerRPCCredentials`, while generated clients receive only `grpc.ClientConnInterface`. Mutable resources are package-local real-TLS gRPC servers, standard health Watch, stream descriptors, raw TLS/HTTP2 peer, movable clock, provider/metadata counters, and owned attempt gates.
  - Depends on: T2 — accepted credential owner and immutable operation token needed to start.
  - Handoff: T2 produces one operation-fixed token plus caller-independent acquisition; T5 consumes it in one complete connection constructor, private credential/application wrapper, and terminal application/control observer.
  - Proof: admitted unary/all application stream cardinalities carry exactly one lowercase bearer; mismatched authority, competing metadata/per-call credentials, insecure transport, canceled acquisition, and provider failure reach no handler; downstream auth statuses remain unchanged and unreplayed. Run `go test -vet=off ./internal/infra/oauth2clientcredentials -run '^(TestGRPCResourceAuthorityIsFixed|TestGRPCApplicationCallsAttachOneToken|TestGRPCRejectsCompetingAuthorization|TestGRPCCallerCancellationStopsApplicationWait|TestGRPCRequiresTransportSecurity|TestGRPCPreservesDownstreamAuthStatus)$' -count=1`.
  - Proof: a forced transparent attempt reuses one bearer/grant; crossing the margin sends no second HEADERS and returns local `token_unusable`. Run `go test -vet=off ./internal/infra/oauth2clientcredentials -run '^TestGRPCAttemptsUseOneOperationToken$' -count=1`.
  - Proof: health Watch authenticates on each new stream, cancellation stops only its wait, reconnect acquires a then-usable token, and an established stream neither mutates metadata nor refreshes in place. Run `go test -vet=off ./internal/infra/oauth2clientcredentials -run '^(TestGRPCControlStreamCancellationStopsOnlyItsWait|TestGRPCHealthWatchUsesConnectionCredentialOnReconnect|TestLongLivedStreamDoesNotReauthenticateInPlace)$' -count=1`.
  - Reopen if: grpc-go's pinned attempt, credential, or control-stream call paths drift — Technical Design; another metadata/status rule, insecure delivery, transparent renewal, or generic reconnect/resume/replay is required — Specification; a concrete long-lived RPC needs continuity beyond stream creation — its concrete RPC owner before Specification/Design.
  - Accepted: T5; evidence: all three recorded T5 proof commands, the focused T5 race gate, and the full package regression passed on the current bounded diff; fresh independent implementation review PASS; candidate: current bounded diff

- [x] T6: Every reachable auth path emits only the complete closed telemetry matrix and no forbidden credential or provider value
  - Source: `spec.md` R7 and success criteria 8-9; `design/overview.md` Failure contract and telemetry plus closed instrument table; `test-plan.md` TD-015 and aggregate claim limits.
  - Owner/surface/resources: finish `internal/infra/oauth2clientcredentials/vocabulary.go`, `telemetry.go`, `vocabulary_test.go`, `telemetry_test.go`, `errors_test.go`, and `doc_test.go`; change only reached T2-T5 files when an accepted signal is missing. Use the shared canary corpus, manual meter, recording tracer/logger, and failing instrument constructor; no exporter, provider, credential, or network resource.
  - Depends on: T3 — startup/retirement/error paths needed to prove; T4 — HTTP cache/acquisition/downstream paths needed to prove; T5 — gRPC application/control/downstream paths needed to prove.
  - Proof: cache/acquisition, every acquisition failure, caller cancellation, retirement, and HTTP/gRPC application or control downstream rejection emit exactly the four accepted instruments and closed attributes/counts; provider calls carry no propagation or generic HTTP span; every canary is absent from errors, readiness, logs, spans, and metrics; instrument failure preserves auth and emits at most one closed warning. Run `go test -vet=off ./internal/infra/oauth2clientcredentials -run '^(TestOutboundAuthTelemetryIsCompleteAndBounded|TestGRPCControlRejectionIsMeasured|TestOutboundAuthForbiddenValuesNeverReachSignals|TestOutboundAuthTelemetryFailureDegradesToNoop)$' -count=1`.
  - Proof: the package documentation/export inventory exposes only the exact Design surface and no token, source, provider interface, registry, clock, retry, transport, dial, TLS, or test hook. Run `go test -vet=off ./internal/infra/oauth2clientcredentials -run '^TestPackageContract$' -count=1`.
  - Reopen if: a new operator question, failure class/precedence, or disclosure rule is required — Specification; a required safe signal has no current owner or needs input-driven cardinality/raw diagnostics — Technical Design/Go Ownership.
  - Accepted: T6; evidence: both recorded T6 proof commands and the OAuth package regression passed on the current bounded diff; fresh independent implementation review PASS; candidate: current bounded diff

## T7 lint-recovery prerequisites

T7 remains blocked. The S3 Go Ownership review-cleared routing identifies no
outbound behavior, package, dependency, generated-source, or proof-policy
change: it assigns bounded lint repairs to accepted T2-T6 owners and one shared
bootstrap lifecycle sequence. These are singleton, serial prerequisites on the
one dirty current-tree candidate; no wave is valid while they and S3's own
T1-T9 repair chain contribute to the same whole-tree lint gate.

Canonical recovery order is S3 T1 through its reopened T8 lint handoff, T7-R0,
S3 T9, T7-R1 through T7-R5, then S3 T10. The S3 serial recovery lead remains
the receiving owner at every cross-ledger boundary: it dispatches the named
accepted-owner repair on the same candidate and may release the next item only
after its predecessor's focused proof is accepted. S3 T10 returns one accepted
whole-tree `make lint` and fresh implementation-review receipt to T7; it does
not accept T7 or rerun its generated-tree proof.

Focused independent Planning readiness review returned **PASS** on candidate
SHA-256 `d2636dc2658194b34af731aee9b773dde75b06e061cf9473b55eb5ffe826c4ad`:
the repaired cross-ledger order, receiving owner, T10 return condition, and
T7 blocker are executable from the ledger and S3's “T10 lint-repair routing”
without a new behavior, ownership, proof, or coordination choice. This
receipt-only record changes no planned unit or dependency.

- [x] T7-R0: The existing shared bootstrap lifecycle sequence is lint-clean while preserving both outbound-auth T3 and object-storage T8 close orders
  - Source: `specs/s3-compatible-object-storage/design/overview.md` “T10 lint-repair routing”; outbound `design/overview.md` “Startup and shutdown composition” and T3.
  - Owner/surface/resources: change only `cmd/service/internal/bootstrap/run.go`; the S3 T8 and outbound T3 lifecycle proof owners remain their existing test files. The mutable resource is the one dirty current-tree candidate.
  - Depends on: S3 T8 lint-repair handoff — the reopened object-storage construction/config/readiness/close constraints for the shared sequence — needed to start.
  - Handoff: S3's reopened T8 supplies its fixed lifecycle constraints; T7-R0 supplies one lint-clean shared `run.go` sequence and passing focused lifecycle proof to the S3 serial recovery lead, which next runs S3 T9 before releasing T7-R1.
  - External input/gate: S3's serial T1-T8 lint recovery reaches the joint `run.go` handoff; owner: S3 serial recovery lead; checkpoint: the handoff names the current candidate and the two unchanged lifecycle proof sets.
  - Proof: both existing profile lifecycle proof sets pass: outbound T3's recorded `go test -vet=off ./internal/infra/oauth2clientcredentials ./cmd/service/internal/bootstrap -run '^(TestClientCloseRetiresAndJoinsAcquisition|TestOutboundAuthRuntimeCloseOrder|TestOutboundAuthCloseFailureIsJoinedOnce)$' -count=1` and S3 T8's recorded focused lifecycle proof; no lifecycle/idempotency behavior changes.
  - Reopen if: a lint repair needs a different lifecycle sequence, package, or idempotency behavior — Go Ownership.
  - Accepted: T7-R0; evidence: S3 T8 lint-repair release revalidated by scoped `bash ./scripts/run-go-tool.sh golangci-lint run ./cmd/service/internal/bootstrap/...` PASS; outbound T3 lifecycle command PASS (9 tests/2 packages); S3 T8 lifecycle PASS (4) and close-order race x10 PASS (30); fresh independent implementation review PASS; candidate: current bounded diff; next owner: S3 serial recovery lead for T9.

- [x] T7-R1: The outbound credential owner and wave proof are lint-clean without changing T2 behavior
  - Source: `specs/s3-compatible-object-storage/design/overview.md` “T10 lint-repair routing”; T2.
  - Owner/surface/resources: change only `internal/infra/oauth2clientcredentials/client.go` and `client_test.go`; mutable resource: the serial current-tree lint candidate.
  - Depends on: S3 T9 — accepted serial S3 lint-repair receipt after T7-R0 — needed to start.
  - Handoff: S3 T9 returns its accepted focused proof to the S3 serial recovery lead, which releases T7-R1 as the next outbound repair.
  - Proof: every recorded T2 focused owner/wave/lifecycle command passes unchanged; the repair leaves cache, cancellation, failure-window, and close observables unchanged.
  - Reopen if: the repair needs a new owner, changes token/wave/lifecycle semantics, or weakens lint policy — Go Ownership or Specification as applicable.
  - Accepted: T7-R1; evidence: package lint returned no diagnostics in `client.go` or `client_test.go`; all six unchanged T2 owner/wave/lifecycle commands passed (2 tests, 4 tests, race x10, 3 tests, race x10, 1 test); fresh independent implementation review PASS; candidate: current bounded diff

- [x] T7-R2: Outbound bootstrap mapping and close proof are lint-clean without changing T3 behavior
  - Source: `specs/s3-compatible-object-storage/design/overview.md` “T10 lint-repair routing”; T3.
  - Owner/surface/resources: change only `cmd/service/internal/bootstrap/startup_outbound_auth.go` and `startup_outbound_auth_test.go`; `run.go` remains exclusively T7-R0. Mutable resource: the serial current-tree lint candidate.
  - Depends on: T7-R1 — accepted preceding lint-repair receipt — needed to start.
  - Handoff: T7-R2 returns its accepted focused proof to the S3 serial recovery lead, which releases T7-R3.
  - Proof: all four recorded T3 focused commands pass unchanged, including the composed startup/health and ordered-close/race checks.
  - Reopen if: the repair needs a shared lifecycle change outside T7-R0, a new bootstrap owner, or different startup/readiness/close behavior — Go Ownership or Specification as applicable.
  - Accepted: T7-R2; evidence: scoped bootstrap lint reported no diagnostics in `startup_outbound_auth.go` or `startup_outbound_auth_test.go`; all four unchanged T3 focused commands passed (132 tests/3 packages, 4 tests/2 packages, 9 tests/2 packages, race 60 tests/2 packages); candidate: current bounded diff

- [x] T7-R3: The sole outbound HTTP adapter proof is lint-clean without changing T4 behavior
  - Source: `specs/s3-compatible-object-storage/design/overview.md` “T10 lint-repair routing”; T4.
  - Owner/surface/resources: change only `internal/infra/oauth2clientcredentials/http_test.go`; mutable resource: the serial current-tree lint candidate.
  - Depends on: T7-R2 — accepted preceding lint-repair receipt — needed to start.
  - Handoff: T7-R3 returns its accepted focused proof to the S3 serial recovery lead, which releases T7-R4.
  - Proof: all three recorded T4 focused HTTP/attempt/generated-client commands pass unchanged.
  - Reopen if: the repair reaches production HTTP code, the shared `httpclient` seam, fixture ownership, or any HTTP behavior/proof policy — Go Ownership/Test Design.
  - Accepted: T7-R3; evidence: scoped package lint returned no diagnostics and `http_test.go` is gofmt-clean; the three unchanged T4 proof commands passed (2 HTTP attempt tests, 12 OAuth/HTTP adapter and generated-client tests, 4 token-margin/retry tests); candidate: current bounded diff; next owner: S3 serial recovery lead for current ledger routing.

- [x] T7-R4: The outbound gRPC credential and stream proof are lint-clean without changing T5 behavior
  - Source: `specs/s3-compatible-object-storage/design/overview.md` “T10 lint-repair routing”; T5.
  - Owner/surface/resources: change only `internal/infra/oauth2clientcredentials/grpc.go` and `grpc_test.go`; mutable resource: the serial current-tree lint candidate.
  - Depends on: T7-R3 — accepted preceding lint-repair receipt — needed to start.
  - Handoff: T7-R4 returns its accepted focused proof to the S3 serial recovery lead, which releases T7-R5.
  - Proof: all three recorded T5 focused gRPC commands pass unchanged.
  - Reopen if: the repair changes credential, stream, TLS, retry, or downstream-status behavior, or reaches another package — Go Ownership/Technical Design or Specification as applicable.
  - Accepted: T7-R4; evidence: scoped package lint reported no diagnostics in `grpc.go` or `grpc_test.go`; all three recorded T5 focused gRPC commands passed (8 application/authority/status tests, 3 transparent-attempt subcases, 4 control-stream tests); candidate: current bounded diff

- [x] T7-R5: The outbound telemetry/redaction proof is lint-clean without changing T6 behavior
  - Source: `specs/s3-compatible-object-storage/design/overview.md` “T10 lint-repair routing”; T6.
  - Owner/surface/resources: change only `internal/infra/oauth2clientcredentials/telemetry_test.go`; mutable resource: the serial current-tree lint candidate.
  - Depends on: T7-R4 — accepted preceding lint-repair receipt — needed to start.
  - Handoff: T7-R5 returns its accepted focused proof to the S3 serial recovery lead, which releases S3 T10.
  - Proof: both recorded T6 telemetry/package-contract commands pass unchanged; closed labels, canary absence, and no-op degradation remain exact.
  - Reopen if: the repair changes signal, disclosure, or telemetry-failure behavior, or reaches another owner — Go Ownership or Specification as applicable.
  - Accepted: T7-R5; evidence: scoped package lint returned no diagnostics; T6 telemetry/redaction proof passed (3 tests) and package-contract proof passed (1 test); fresh independent implementation review PASS; candidate: current bounded diff

- [x] T7: `OUTBOUND_AUTH=none|oauth2-client-credentials` generation is deterministic, profile-clean, documented, and passes every portable scenario and aggregate gate
  - Source: `spec.md` R1 and R7-R10, profile success criterion 1, regression criterion 9, and external claim limits; `design/overview.md` Profile and generated authority, Non-Go file map and cleanup, dependency ownership, and reopen conditions; `test-plan.md` TD-016, aggregate gates, external proof boundaries, and bidirectional closure.
  - Owner/surface/resources: change `scripts/init-module.sh` and `scripts/ci/template-init-check.sh`; add `docs/outbound-machine-authentication.md`; change `README.md`, `docs/repo-architecture.md`, `docs/project-structure-and-module-organization.md`, `docs/configuration-source-policy.md`, `env/config/local.yaml`, and `env/.env.example`; apply the exact core/HTTP/gRPC/credential-provider markers and generated inventory to T1-T6 files. `scripts/init-module.sh` remains handwritten authority, `template.lock` and initialized trees remain generated, `template-owned.paths` remains unchanged, and no OpenAPI/protobuf/SQLC/migration/container/manifest generation is triggered. Mutable resources are only trap-owned temporary generated checkouts; no credential or network.
  - Depends on: T6 — every portable runtime/adapter/signal path and exported surface must be accepted before retention/stripping and aggregate closure can be proved; T4-R0 — accepted Local shared-surface continuation required to prove the current HTTP path; T7-R5 — accepted serial outbound lint-repair receipts — needed to prove; S3 T10 — accepted whole-tree `make lint` and fresh implementation-review receipt after S3 T1-T9 and T7-R0 through T7-R5 — needed to prove.
  - Handoff: T6 produces the complete deterministic capability; T4-R0 produces the accepted Local HTTP shared-surface continuation; T7-R0 through T7-R5 produce the bounded outbound lint-repair receipts; S3 T10 consumes those receipts with its own T1-T9 repairs and returns the sole whole-tree lint/review gate that T7 consumes before rerunning its generated-tree and remaining aggregate proof.
  - Proof: omission and explicit `none` are byte-identical; explicit empty/unknown/no-consumer values fail before mutation; HTTP-only, gRPC-only, and both selected outputs retain the core/token HTTP owner and only selected adapters; `none` strips all auth config/env/docs/code/tests/markers; union-owned credential HTTP policy remains for OIDC or outbound OAuth only; lock, repeat, module attribution, unresolved-marker, build, tests, and both tidy paths match TD-016. Run `TEMPLATE_INIT_PROFILE=outbound-auth make template-init-check`, `make project-structure-check`, and `make mod-tidy-check`.
  - Proof: on the same current candidate run `go test -vet=off ./internal/config ./internal/infra/httpclient ./internal/infra/grpcclient ./internal/infra/oauth2clientcredentials ./cmd/service/internal/bootstrap`, `go test -vet=off -race ./internal/infra/httpclient ./internal/infra/grpcclient ./internal/infra/oauth2clientcredentials ./cmd/service/internal/bootstrap`, `TEMPLATE_INIT_PROFILE=outbound-auth make template-init-check`, `make project-structure-check`, `make mod-tidy-check`, `make lint`, and `make test`; every focused TD-001-TD-016 command remains the behavioral oracle and must already pass.
  - Reopen if: selector/consumer/cardinality/retention semantics change — Specification; marker, inventory, generated/manual authority, package/file ownership, or dependency attribution changes — Go Ownership; stdlib/grpc-go/httpclient behavior or another source dependency changes a selected mechanism — Technical Design; a named provider/deployment contract conflicts with portable behavior — the exact Specification/Design owner, with Research reopened only for a materially distinct candidate family, invalidated Basic interoperability floor, or changed reusable responsibility boundary.
  - Accepted: T7; evidence: released HTTP T5 prerequisite at SHA-256 `322a204d26d7abdf9a8eeedfaf1c110d7309eaa72b542fabdc57f6f2c7d0b6c4` revalidated; focused and race package aggregates, `TEMPLATE_INIT_PROFILE=outbound-auth make template-init-check`, `make project-structure-check`, `make mod-tidy-check`, `make lint` (0 issues), `make test` (2688 tests, 4 declared skips), and `git diff --check` passed; fresh independent implementation review PASS, including an independent outbound-profile aggregate; candidate: current bounded T7 diff.

- [ ] T8: The immutable PR #135 capability incorporates the reference-audit corrections without widening its provider or transport policy
  - Source: the read-only audit of immutable commit `282b15e007f95ab0feaec530308570185ad58d0e`, updated R6/R7/R10, and TD-006/TD-007/TD-010/TD-012/TD-015/TD-016.
  - Owner/surface/resources: correct the process-local failure cooldown, reuse the existing HTTP `Retry-After` parser, replace the separately miswirable gRPC credential/wrapper exports with one complete constructor and one optional `grpcclient` terminal observer, add the outbound profile to PR CI, and synchronize operator/design/test artifacts. Add no discovery, provider fallback, token persistence, background refresh, resource replay, second transport policy, credential, or live endpoint.
  - Proof: focused provider, cooldown, gRPC construction/control telemetry, package-contract, HTTP retry, race, profile-generation, project-structure, tidy, lint, test, and diff checks pass from the isolated correction worktree based on exact `282b15e0`.
  - External boundary: provider registration and conformance, scopes/resource/audience, asymmetric or sender-constrained alternatives, credential issuance/rotation, DNS/TLS/egress, quotas/fleet capacity, deployment, criticality, and long-stream continuity remain unclaimed.
  - Reopen if: a provider requires discovery, another authentication method, incompatible exact-scope semantics, cross-process throttling, a different gRPC control policy, or any external proof above.
  - Validation: focused regression passed 353 tests; focused race passed 28; cooldown race repetition passed 20; full touched-owner race passed 547; outbound profile generation, project structure, module tidy, Go lint, deep lint, repository tests, delivery quality, and `git diff --check` passed.
  - Acceptance: implementation complete; verification incomplete until one fresh independent read-only implementation review returns PASS for this fixed correction candidate.
