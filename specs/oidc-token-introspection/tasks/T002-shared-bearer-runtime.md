# T002 — Shared bearer runtime with unchanged JWT behavior

Outcome:
Move the bearer carrier, sanitized failure taxonomy, verification metric, principal publication, HTTP resolver, native gRPC policy, and stream-expiry mechanics into `internal/infra/bearerauthn`, with `internal/infra/oidcjwt` retaining only JWT/JWKS trust, while every current JWT and `AUTHN=none` observable remains unchanged.

Consumes:
- [`../spec.md`](../spec.md) sha256 `4a97de705be58331a2de8c7621cc3eaff6dc88175fd147a59a1217c927c91292`, unchanged handler/transport/profile contract.
- [`../design/system-integration-design.md`](../design/system-integration-design.md) sha256 `0c1c8b627cc79c6cac44bb727473e9a05bf37acba248b5ca9e87b47798759b53`, shared-shell mechanism and admission/lifecycle rules.
- [`../design/go-code-ownership-design.md`](../design/go-code-ownership-design.md) sha256 `41e754e723a27356752013abeabc51d0b269b0dfd9137c40401dcae233637523`, exact package, import, composition, generated/manual, cleanup, and proof owners.
- [`../test-plan.md`](../test-plan.md) sha256 `856eb3dccf985c8d00764901e96b7c5843a4a54a8e4882eefa2b5e563a86dfea`, rows TD-HTTP-01, TD-GRPC-01, TD-STR-01, TD-CAP-01 through TD-CAP-03, and TD-OBS-01; TD-REG-01 and TD-OWN-01 are partially established here and proved finally in T003.
- Independent PASS receipts in [`../design/go-ownership-review.md`](../design/go-ownership-review.md), [`../design/technical-design-review.md`](../design/technical-design-review.md), and [`../test-design-review.md`](../test-design-review.md).
- Accepted current authentication baseline at `HEAD` `8967a4ac06d4fce0515703b15ffa5db35e5378ae`; overlapping dirty `oidcjwt` and initializer edits are excluded.

Provides:
- Accepted consumer-owned `bearerauthn.Verifier` and shared runtime contract for T003.
- A green source template plus generated `oidc-jwt` and `none` outputs with no authentication behavior change.

Boundary:
Perform only the reviewed behavior-preserving ownership move, import rewiring, obsolete-owner deletion, and common marker/profile cleanup needed to keep the current JWT and none outputs valid. Do not add introspection policy, provider calls, configuration leaves, endpoint credentials, the third initializer choice, caching, retry, or new telemetry. Keep the JWT Discovery/JWKS engine and refresh lifecycle concrete in `oidcjwt`.

Mutable owners:
- `internal/infra/bearerauthn` shared runtime contract and transport proof
- `internal/infra/oidcjwt` transport-neutral JWT/JWKS engine and retained proof
- HTTP authentication error mapping and current native gRPC authentication seam
- bootstrap shared authentication runtime seam and current JWT constructor placement
- common/JWT authentication initializer markers and current JWT/none generated-output proof

Exclusive locks:
- shared bearer runtime and sanitized authentication failure contract
- bootstrap authentication composition seam
- current authentication initializer marker/removal contract

Accept when:
- Claim: The ownership move leaves bearer grammar/removal, errors, challenges/statuses, principal publication, exact public gRPC Check rule, stream expiry, transport admission ceilings, verification telemetry, JWT verification/JWKS lifecycle, and none-profile absence unchanged.
- Focused check: Run every exact `command_or_procedure` cell for TD-HTTP-01, TD-GRPC-01, TD-STR-01, TD-CAP-01, TD-CAP-02, TD-CAP-03, and TD-OBS-01, plus the repository-native focused JWT/package tests, `make lint mod-tidy-check`, and `make template-init-check` for the current JWT/none outputs.
- Observable: Every named row reaches its fixed oracle with a non-zero test count; existing JWT tests pass after relocation; none output contains no authn runtime or JWT dependency; the import graph is acyclic; deleted `oidcjwt` transport/error owners and stale markers do not survive.

Reopen if:
Reopen System / Integration Design if JWT and introspection cannot share the selected transport semantics without changing behavior. Reopen Go Code / Ownership Design if the refreshed composition root, shared seam, generated/manual owner, package graph, or marker authority no longer matches the reviewed map. Reopen Planning if this behavior-preserving result cannot land green without T003.
