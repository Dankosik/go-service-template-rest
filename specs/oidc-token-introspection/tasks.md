# Goal
status: done
Completion: T001 through T003 are accepted on one integrated candidate; every command in the reviewed Test Design validation ladder is terminal-success for that exact candidate; the required integrated Implementation Review passes; and no live-provider or deployment-enablement claim is made.
Global constraints: Preserve the reviewed Specification, System / Integration Design, Go Code / Ownership Design, and Test Design, including the fixed five-second provider and response-header timeouts. Treat all pre-existing dirty checkout bytes, especially the non-compiling `internal/infra/httpclient` refactor, as unaccepted and preserve them outside every candidate. Refresh each accepted current owner immediately before editing; an equivalent accepted `httpclient` surface may replace constructor names only when it preserves every fixed semantic guard. Add no dependency, cache, retry, limiter, queue, fallback, runtime authn switch, provider value, credential, network, deployment, or external action. TD-EXT-01 and TD-EXT-02 remain separate enablement gates below.

## Tasks

- [x] T001: The accepted fixed-authority HTTP client exposes and proves only the response-header guards required by introspection while every existing constructor and caller remains unchanged.
  - Depends on: none
  - Provides: accepted positive `ResponseLimits` semantics and equivalent external/private limited constructors enforcing the fixed five-second header timeout and 32 KiB header maximum
  - Packet: tasks/T001-fixed-authority-response-limits.md
Accepted: T001; evidence: TD-HDR-01 PASS 1; TD-HDR-02 PASS 1; TD-HDR-03 PASS 3; package PASS 15; review PASS; candidate: 4f9547ebabc736381c794bc8db5069f550ba3cc0
- [x] T002: The current JWT profile uses one shared bearer-authentication runtime without changing JWT, none-profile, HTTP, gRPC, telemetry, admission, or lifecycle behavior.
  - Depends on: none
  - Provides: accepted `internal/infra/bearerauthn` verifier/runtime contract and behavior-preserving JWT/none generated baseline
  - Packet: tasks/T002-shared-bearer-runtime.md
Accepted: T002; evidence: TD-HTTP-01 27; TD-GRPC-01 9; TD-STR-01 5; TD-CAP-01 1; TD-CAP-02 1; TD-CAP-03 1; TD-OBS-01 17; JWT/package 92; lint/mod-tidy-check 0 issues; template-init-check none+oidc-jwt passed; review PASS; candidate: tree be4f8bba708ff9dd1ceda992bc3c0acfa3f20323 landed as ce0092b5d8f47abd5fbd64230207ccf6217bfd72
- [x] T003: The source template and generated outputs provide the complete strict uncached `AUTHN=oidc-introspection` profile while preserving `none` and `oidc-jwt` and satisfying the full reviewed local proof matrix.
  - Depends on: T001 accepted response-header guard surface; T002 accepted shared bearer runtime and JWT/none baseline
  - Provides: locally accepted OIDC token-introspection implementation and three-way initializer output, without provider or deployment enablement
  - Packet: tasks/T003-oidc-introspection-profile.md
Accepted: T003; evidence: focused CFG/REQ/RSP/EDGE/CACHE/PAR/HTTP/GRPC/LIFE/SEC/GEN/REG/OWN held; ladder package 707, check 1361, race 216, shuffle 580, openapi-check, lint 0, template-init-check terminal-success, runtime-image 36c90b19a37d; TD-EXT-01/02 unclaimed; review PASS; candidate: tree 7656a42fff49998e0a948a6b7b423b2da8bc3ff5 landed as 10f72d71cd6182e3de56824d63cde646a45ed278
Accepted: integrated T001-T003; evidence: validation ladder terminal-success on tree 7656a42fff49998e0a948a6b7b423b2da8bc3ff5; integrated Implementation Review PASS; no live-provider or deployment-enablement claim; candidate: 10f72d71cd6182e3de56824d63cde646a45ed278

## No-implementation dispositions

- TD-EXT-01 remains an adopter/deployment-owned live compatibility gate before enablement. No provider, registration, endpoint, credential, token, network, or deployment authority is available, and no repository task can manufacture that evidence.
- TD-EXT-02 remains a provider/deployment/business-owned capacity, revocation, edge-abuse, and alerting gate before enablement. No entitlement, objective, propagation bound, edge policy, alert threshold, or deployment is available, and no local task may infer one.
