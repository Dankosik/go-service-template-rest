# T003 — Complete strict uncached introspection profile

Outcome:
The source template and disposable generated outputs implement exactly one strict uncached `AUTHN=oidc-introspection` profile behind the accepted shared bearer runtime, preserve `AUTHN=none` and `AUTHN=oidc-jwt`, and satisfy every reviewed local Test Design obligation without claiming or performing provider/deployment enablement.

Consumes:
- T001 accepted fixed-authority response-header guard surface, including the fixed five-second timeout and 32 KiB maximum.
- T002 accepted `bearerauthn` verifier/runtime contract and green JWT/none baseline.
- [`../spec.md`](../spec.md) sha256 `4a97de705be58331a2de8c7621cc3eaff6dc88175fd147a59a1217c927c91292`.
- [`../research/synthesis.md`](../research/synthesis.md) sha256 `9568a46338b10b7b135a7f1917bc5998b82f6065ff0ab4ed0ef6667394994245`.
- [`../design/system-integration-design.md`](../design/system-integration-design.md) sha256 `0c1c8b627cc79c6cac44bb727473e9a05bf37acba248b5ca9e87b47798759b53`.
- [`../design/go-code-ownership-design.md`](../design/go-code-ownership-design.md) sha256 `41e754e723a27356752013abeabc51d0b269b0dfd9137c40401dcae233637523`.
- [`../design/go-ownership-review.md`](../design/go-ownership-review.md) sha256 `b35b7c512c2c4e5b23e7fb0c045ebca21412e1c3f6c05df2b231ddf9c9348a2e` and [`../design/technical-design-review.md`](../design/technical-design-review.md) sha256 `b89d5c1a6f40a2496b43ad81eb07612da556eccaa3e6277baa199077b4867965`.
- [`../test-plan.md`](../test-plan.md) sha256 `856eb3dccf985c8d00764901e96b7c5843a4a54a8e4882eefa2b5e563a86dfea` and [`../test-design-review.md`](../test-design-review.md) sha256 `3afc643049b874c5c8dedf0d323df4c6210ddc9c56fee6cb6cc10e1fe77c855e`.

Provides:
- Complete locally accepted source-template implementation of the strict RFC 7662 subset.
- Generated `none`, `oidc-jwt`, and `oidc-introspection` outputs with exact physical presence/absence, dependencies, neutral OpenAPI contract, documentation, and `template.lock` value.
- Local implementation evidence suitable for later adopter/deployment certification, without satisfying TD-EXT-01 or TD-EXT-02.

Boundary:
Add the reviewed pure trust predicates, immutable introspection configuration and environment-only secret admission, concrete `oauthintrospection` engine, strict response parser, bootstrap profile constructor and lifecycle wiring, three-way initializer/profile templates, neutral common OpenAPI wording, selected-profile docs/examples, depguard rules, and local/runtime-image proof. Use stdlib plus existing dependencies. Do not change `reqctx.Principal`, authorization, readiness ownership, transport admission ceilings, the accepted T001/T002 contracts, or any provider/deployment input. Do not add provider discovery, another credential method, cache, retry, limiter, queue, breaker, fallback, failover, runtime profile switch, or live action.

Mutable owners:
- `internal/authntrust` pure introspection endpoint and target-class rules
- `internal/infra/oauthintrospection` policy, verifier/client lifecycle, response admission, and package-local proof
- `internal/config` selected introspection tuple, secret-source admission, and profile-specific replacement
- bootstrap concrete authentication profile construction and lifecycle proof
- authentication initializer, profile templates, generated-output checker, runtime-image fixture, and exact lock generation
- common OpenAPI authentication source/generated contract, selected config examples, authentication/architecture documentation, and dependency-boundary enforcement

Exclusive locks:
- three-way authentication initializer/profile/removal and `template.lock` contract
- introspection configuration and secret-source contract
- bootstrap concrete authentication profile seam
- common OpenAPI Bearer security source and generated output

Accept when:
- Claim: Every local planned scenario and non-test falsifier in the fixed Test Design passes for one exact candidate; `none` and JWT remain unchanged; the introspection profile is strict, one-attempt, uncached, bounded, privacy-safe, statically selected, lifecycle-correct, and physically isolated; external rows remain unclaimed.
- Focused check: Run each exact `command_or_procedure` cell and fixed oracle for TD-CFG-01, TD-CFG-02, TD-REQ-01, TD-REQ-02, TD-RSP-01 through TD-RSP-05, TD-EDGE-01, TD-EDGE-02, TD-CACHE-01, TD-PAR-01, TD-SEC-01, TD-LIFE-01, TD-LIFE-02, TD-GEN-01 through TD-GEN-03, TD-REG-01, and TD-OWN-01. Run the previously accepted T001/T002 focused rows again wherever this candidate can affect their seam. A zero-match, cached, skipped, unavailable mandatory input, or dirty-tree substitute is not passing evidence.
- Integrated check: Run the eight commands in the fixed Test Design Validation ladder, in order, against the exact integrated candidate; Docker is mandatory for step 7 and must be refreshed before T003 starts.
- Observable: Every assigned scenario reaches its independent oracle with recorded non-zero test counts; every ladder command is terminal-success; generated outputs have the exact selected source, dependency, OpenAPI, documentation, marker, and lock inventory; runtime startup/readiness/provider-failure behavior is sanitized and correct; the final report records exact candidate identity and no external compatibility, capacity, revocation, egress, credential, alerting, or enablement claim.

Reopen if:
Use the exact `reopen_owner` in the fixed Test Design row that fails. Reopen Specification only for one of its explicit behavior/provider/cache/retry/readiness/budget conditions; System / Integration Design only for a lost fixed-edge, lifecycle, admission, or physical-absence mechanism; Go Code / Ownership Design only for a changed composition/import/generated owner; Test Design only for a missing or infeasible oracle. A constructor-name-only accepted T001 refresh stays inside Implementation. Missing TD-EXT-01 or TD-EXT-02 inputs blocks enablement, not this local task.
