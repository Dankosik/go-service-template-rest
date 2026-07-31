# Goal

status: done

Completion: the optional bounded HTTP and native gRPC clients enforce the
accepted explicit correlation policy on every observable attempt, the
repository demonstrates trusted cross-service composition and cancellation,
and selected/unselected template profiles plus generated contracts remain
valid.

Blocked stop: stop without a readiness claim if the pinned grpc-go public API
cannot close a documented late-metadata path, if deterministic transparent
retry cannot expose both attempts, or if a profile fixture requires changing
the accepted policy/ownership. Record the failed observable and reopen the
narrow Design or Test Design owner.

Global constraints: add no runtime dependency; never propagate baggage;
zero-value propagation is fail-closed `none`; preserve caller contexts and
caller-owned HTTP/gRPC metadata; do not infer trust from transport or network
placement; preserve inbound behavior, bounded transport semantics, and
provider-owned generated authority; do not consume or modify the concurrent
Goose work in the primary checkout.

- [x] T1: the bounded HTTP client enforces the closed outbound policy on every
  explicit attempt without weakening its fixed-authority, retry, body,
  redirect, cleanup, or telemetry contracts
  - Source: `spec.md` COR-1 through COR-4; `design/overview.md` HTTP mechanism;
    `test-plan.md` TD-1, TD-2, and TD-9 HTTP seam.
  - Owner/surface/resources: `internal/infra/httpclient/` production and tests;
    HTTP responsibility wording in `docs/repo-architecture.md` and
    `docs/project-structure-and-module-organization.md`; generated HTTP
    composition guidance in `docs/first-production-feature.md`; minimal
    OpenAPI client testdata and temporary generated child directories under
    `internal/infra/httpclient/`, always removed by their creator; loopback
    listeners are test-owned; process-global OTel tracer-provider test state is
    non-parallel and restored by `telemetrytest.InstallSpanRecorder` cleanup.
  - Depends on: none.
  - Proof: policy table, stale mixed-case header and trailer removal,
    valid/invalid/missing request IDs, request header/trailer immutability,
    attempt-specific trace parentage, no
    correlation/header/raw-URL metric attributes, retry regression, and
    compile/runtime proof that the real pinned oapi-codegen `WithHTTPClient`
    seam accepts `*httpclient.Client`;
    `go test -vet=off ./internal/infra/httpclient -run
    'Test.*(Propagation|Retry|GeneratedClientComposition)' -count=1`; then
    `go test -vet=off ./internal/infra/httpclient -count=1`; expected observable
    is exact wire header/trailer fields per attempt, successful generated-client
    composition, plus unchanged existing safety/retry tests; documentation
    examples are root-reviewed against that compiled seam and `git diff
    --check`.
  - Reopen if: a hidden `net/http` resend must become a separately observable
    span; Reliability Design.

- [x] T2: the native gRPC client enforces the same policy across unary,
  streaming, credentials, resolvers, and transparent retries while preserving
  native target/authority selection and existing transport bounds
  - Source: `spec.md` COR-1 through COR-4; `design/overview.md` gRPC mechanism
    and resolver boundary; `test-plan.md` TD-3 through TD-6 and TD-8, including
    TD-5a/TD-5b/TD-5c, plus TD-9 gRPC seam.
  - Owner/surface/resources: `internal/infra/grpcclient/` production and tests;
    current `grpcclient.Options` consumers in
    `internal/infra/grpc/telemetry_test.go`,
    `examples/grpc-reference-service/cmd/benchmark-server/`, and
    `test/grpc_process_integration_test.go`; gRPC client guidance in
    `docs/grpc.md`; resolver selection/default mutations are isolated in fresh
    test-binary subprocesses, while proxy environment is test-scoped; loopback
    listeners are test-owned.
  - Depends on: none.
  - Proof: all-policy unary/stream and transparent-retry matrices, trusted
    per-RPC credentials with plaintext/TLS/error paths, pure resolver
    state/lifecycle/authority delegation, subprocess-isolated native scheme selection,
    live service-config/proxy/TLS suppression, limits, metrics privacy, and
    live cancellation; `go test -vet=off ./internal/infra/grpcclient
    ./internal/infra/grpc -run
    'Test.*(Propagation|PerRPCCredentials|Resolver|Proxy|TransparentRetry|Correlation|Telemetry|TLS)'
    -count=1`; `go test -vet=off -race ./internal/infra/grpcclient
    ./internal/infra/grpc -run 'Test.*Correlation' -count=1`; then
    `go test -vet=off ./internal/infra/grpcclient ./internal/infra/grpc
    ./examples/grpc-reference-service/... -count=1`;
    `REQUIRE_DOCKER=1 go test -count=1 -tags=integration ./test/...
    -run '^TestGRPCProcessLifecycle$'`; expected observable is
    exact per-attempt metadata/parentage, resolver callback counts and
    immutability, zero proxy/balancer use, deterministic cancellation
    completion, the real production process accepting the context-owned
    propagated request ID, and every existing client/server contract green.
  - Reopen if: the pinned public grpc-go API cannot conditionally preserve
    `AuthorityOverrider`, expose transparent attempts, or make the selected
    wrapped resolver final; gRPC Client Design and Security.

- [x] T3: repository composition proves trusted service-to-service calls and
  optional-profile integrity without inventing a downstream contract
  - Source: `spec.md` COR-4 and CON-1; `design/overview.md` Generated consumer
    composition and Go ownership; `test-plan.md` TD-7 and TD-10.
  - Owner/surface/resources: cross-adapter proof under the optional
    `internal/infra/httpclient/` package so it is removed with
    `OUTBOUND_HTTP=none`; profile/drift scripts remain unchanged unless
    current proof identifies an accepted owned companion; template-init
    temporary copies, private-address listeners, and the in-process DNS
    responder are creator-owned.
  - Depends on: T1 — output handoff — needed to start; T2 — output handoff —
    needed to start.
  - Handoff: T1 supplies the accepted bounded HTTP policy API and attempt
    behavior; T2 supplies the accepted gRPC connection policy API and
    metadata-ordering guarantees. T3 consumes those exact public seams in the
    composed HTTP proof and profile fixtures.
  - Proof: two repository HTTP service boundaries share trace/request ID and
    deterministically release downstream work on cancellation; optional
    profiles retain/remove complete capabilities; generated contracts do not
    drift;
    focused `go test -vet=off ./internal/infra/httpclient -run
    'Test.*ServiceToService.*Correlation' -count=1` and the same command under
    `-race`; `make openapi-check`; `make proto-check`;
    `TEMPLATE_INIT_PROFILE=minimal make template-init-check`;
    `TEMPLATE_INIT_PROFILE=postgres make template-init-check`;
    `TEMPLATE_INIT_PROFILE=grpc make template-init-check`; then `make check`.
    Expected observable is exact composed correlation/cancellation plus clean
    profile, generation, and repository gates. Run `make check-full` only when
    the final diff still triggers Docker-backed/security publication evidence;
    a skipped required branch is reported as a gap.
  - Reopen if: a deployed proxy/mesh rewrites the fields; Delivery.
