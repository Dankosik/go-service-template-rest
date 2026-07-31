# Goal

status: accepted

Completion: an initialized service chooses `AUTHN=none|oidc-jwt`; `none`
contains no authentication residue, while `oidc-jwt` supplies the ready
standards-backed fail-closed HTTP and optional native-TLS gRPC verifier through
`reqctx.Principal`, with every TD-01–TD-21 falsifier and generated-profile
contract passing.

Blocked stop: stop with the earliest failing claim and its exact command,
observable, and owning upstream artifact. A missing security behavior,
trust/lifecycle rule, package/source authority, proof oracle, or generated
profile input reopens Specification, Design, Test Design, or Planning
respectively; it is not chosen during implementation.

Global constraints: work only in the isolated task worktree; canonical
OpenAPI/config/profile sources precede generated output; use go-jose v4.1.4
only as the JOSE/JWK primitive; one concrete verifier owns both transports;
never add a runtime bypass, permissive fake identity, generic authentication
framework, authorization policy, endpoint/token disclosure, or unbounded
network/telemetry dimension. Profile blocks and files must be capability-scoped
so sibling work remains independently integrable. Serialize the generator,
race, lint, and security gates.

- [x] T1: The complete optional OIDC/JWT pack is implemented as one internally
  valid profile change: strict initial/runtime trust and identity work through
  the production HTTP and conditional gRPC entry points, while initialization
  physically removes the complete pack for `AUTHN=none`.
  - Source: `spec.md` AUTHN-01–AUTHN-23 and invariants;
    `design/overview.md` selected dependency, exact config/parser/cache/
    lifecycle/adapters/observability/ownership/profile decisions;
    `test-plan.md` TD-01–TD-21, NT-01, and NT-02.
  - Owner/surface/resources: canonical HTTP contract
    `api/openapi/service.yaml` and derived `internal/openapi/openapi.gen.go`;
    immutable runtime config under `internal/config/` plus `env/config/base.yaml`
    and `env/.env.example`; concrete new `internal/infra/oidcjwt/` package and
    capability testdata; marker-scoped consumers in `internal/infra/http/`,
    `internal/infra/grpc/`, `internal/infra/httpclient/`,
    `internal/problem/`, and `internal/reqctx/` only if the ready design's
    existing principal surface requires a test companion; composition under
    `cmd/service/internal/bootstrap/`; dependency authority `go.mod`/`go.sum`;
    initializer/generator owners `scripts/init-module.sh`,
    `scripts/ci/template-init-check.sh`, bounded companion change-scope logic,
    and `.github/workflows/ci.yml`; capability documentation
    `docs/authentication.md` plus marker-scoped integration text in `README.md`,
    `docs/repo-architecture.md`, `docs/configuration-source-policy.md`,
    `docs/build-test-and-development-commands.md`, and
    `docs/ci-cd-production-ready.md`. Bounded discovery may add or update only
    adjacent tests, generated config docs, and profile-marked companion
    surfaces already consumed by those owners. Exclusive resources are Go
    module/generated files and temporary initializer fixtures; there is no
    external identity provider, deployment, or shared persistent resource.
  - Depends on: none.
  - Proof: strict verifier/config/refresh/lifecycle/HTTP/gRPC/telemetry behavior
    from TD-01–TD-17 and profile-owned documentation example from TD-20:
    `go test -vet=off ./internal/config ./internal/infra/httpclient ./internal/infra/oidcjwt ./internal/infra/http ./internal/infra/grpc ./cmd/service/internal/bootstrap -count=1`;
    expected observable is real-RS256 valid identity at both handlers, exact
    finite fail-closed outcomes, atomic bounded refresh/readiness/lifecycle,
    fixed-cardinality metrics, and poison-marker absence across returned and
    captured surfaces.
  - Proof: strict parser no-panic branch from TD-02:
    `go test -vet=off ./internal/infra/oidcjwt -run '^$' -fuzz '^FuzzStrictToken$' -fuzztime=10s`;
    expected observable is no panic and no successful principal for generated
    malformed inputs.
  - Proof: canonical-before-generated HTTP contract from TD-11:
    `make openapi-generate && make openapi-check`; expected observable is clean
    generated drift, one fail-closed top-level Bearer requirement, explicit
    public health overrides, runtime conformance, lint, and schema validity.
  - Proof: optional gRPC contract ownership from TD-13–TD-14:
    `make proto-check`; expected observable is existing generated protobuf
    authority remains clean and the authn adapter composes without changing
    RPC schema.
  - Proof: structural profile contract from TD-18–TD-19:
    `TEMPLATE_INIT_PROFILE=authn bash ./scripts/ci/template-init-check.sh`;
    expected observable is pre-mutation rejection for empty/invalid values,
    byte-stable repeat initialization, `none` physical purity and dependency
    cleanup, and compiling OIDC HTTP-only/HTTP+gRPC services across both
    requested outbound choices with correct locks, generation, and tidy state.
  - Proof: shared-state safety from TD-21 after focused non-race scenarios pass:
    `go test -vet=off -race ./internal/infra/oidcjwt ./internal/infra/grpc ./cmd/service/internal/bootstrap -run 'Test(Refresh|Trust|VerifierLifecycle|AuthnHealth)' -count=1`;
    expected observable is all owned join events completing with no race.
  - Proof: changed-Go and dependency security:
    `make fmt-check && make lint && go mod verify && make go-security`;
    expected observable is clean formatting/lint, verified module sums, and no
    material current govulncheck/gosec finding in the changed boundary.
  - Proof: closeout evidence from NT-01–NT-02:
    `git diff --check`, `go version`, `go list -m -json github.com/go-jose/go-jose/v4`,
    `go list -deps ./cmd/service`, and final `git status --short`; expected
    observable is clean patch syntax, recorded tool/dependency identity,
    production graph excluding test private keys/replay/session machinery, and
    a clean task worktree after the local commit.
  - Reopen if: a current go-jose advisory/toolchain incompatibility appears;
    the existing bounded HTTP owner cannot express exact issuer/JWKS authority
    without weakening its policy; the target cannot provide trusted HTTP proxy
    CIDRs or native gRPC TLS; or a deterministic TD scenario cannot reach the
    accepted production seam. Reopen Research/Design for the first two,
    Specification/System Design for the transport boundary, and Test Design
    for the last.

## Reconciliation and acceptance unit

T1 is the single reconciliation disposition for every implementation-changing
obligation. Splitting verifier, adapters, canonical/generated contracts, or
initializer removal would leave at least one generated profile either
retaining dormant authentication or claiming OIDC without a working production
path, so no smaller task satisfies the valid split-boundary rule.

NT-01 is a proved no-implementation disposition: replay storage and
sender-constrained tokens are explicitly excluded and must remain absent and
documented. NT-02 is completion evidence attached to T1 rather than a later
proof-only task.

Acceptance unit A1 is T1. It runs sequentially in this worktree; there is no
planned parallel wave and no external gate.

## Acceptance evidence

Implementation review: independent claim-scoped re-review `PASS`. The reviewer
confirmed that the production bootstrap failure path performs no router build,
listener start, server start, or admission; every typed HTTP authentication
failure maps to its finite public response without reaching the handler;
TD-07/TD-08 exercise production refresh and rotation behavior; and TD-12/TD-15
exercise readiness/currentness. No known material finding remains in the
reviewed boundary.

Current deterministic proof:

- focused owner packages:
  `go test -vet=off ./internal/config ./internal/infra/httpclient ./internal/infra/oidcjwt ./internal/infra/http ./internal/infra/grpc ./cmd/service/internal/bootstrap -count=1`
  passed;
- strict-token fuzzing:
  `go test -vet=off ./internal/infra/oidcjwt -run '^$' -fuzz '^FuzzStrictToken$' -fuzztime=10s`
  passed without panic or false acceptance;
- the complete template contract:
  `TEMPLATE_INIT_PROFILE=authn bash ./scripts/ci/template-init-check.sh`
  passed all twelve generated fixtures, invalid-value pre-mutation rejection,
  repeat-byte stability, `AUTHN=none` purity, and OIDC HTTP-only/HTTP+gRPC
  compilation across both outbound profiles;
- the full repository behavior:
  `go test -vet=off ./... -count=1` passed;
- focused race proof:
  `go test -vet=off -race ./internal/infra/oidcjwt ./internal/infra/grpc ./cmd/service/internal/bootstrap -run 'Test(Refresh|Trust|VerifierLifecycle|AuthnHealth|AuthnReadiness)' -count=1`
  passed;
- `make fmt-check`, `make lint`, `make mod-tidy-check`, `make openapi-check`,
  `make proto-check`, `go mod verify`, `make go-security`,
  `make secret-scan`, and `make secret-scan-check` passed;
- an independent production build contained none of the deterministic private
  key sentinels; the selected primitive was recorded as
  `github.com/go-jose/go-jose/v4 v4.1.4` under Go 1.26.5.

The final local commit and post-commit clean-status check are closeout carrier
evidence. They do not widen the accepted behavior or the reviewed boundary.
