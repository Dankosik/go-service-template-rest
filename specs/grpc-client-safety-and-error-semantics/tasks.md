# gRPC clients use safe idle behavior, health-aware routing, and transport-neutral failures

status: done

Completion: T1 and T2 are accepted from the current bounded diff; default clients
omit idle keepalive, round-robin selection follows standard health without changing
application retries or stream-proof ownership, protected OIDC `Watch` composes
through the sanitized connection credential, the article collision is
`already_exists` on both composed transports, preserved compatibility checks pass,
and the repaired mixed-profile plus repository aggregate gates pass.

Blocked stop: Leave the affected unit unchecked and record the first failed
scenario, command, and observable. Reopen Specification for a caller-visible
behavior or identity conflict, Technical Design for a package/file, service-config,
production-seam, or profile-owner conflict, and Test Design only when its named
observable cannot be produced from the accepted owner.

Global constraints: Start from HEAD `da89db83a78ca4a19fefe66d4105f69fb73b7ff0`
and ready input SHA-256 values `1bb0965f876b052abd8e5be2142acfc8734141d5506e719d9529e6a425860cac`
(`spec.md`), `a304576e75220e630ec7a67b9ec9287020f45f485598034809dac8444a0d6667`
(`design/overview.md`), and `cdd1bc38cccaa4ca5aae2134200a67b4d806dec8275285e1bbdce018b6a3b2b9`
(`test-plan.md`). Preserve grpc-go v1.82.1 authority, resolver service-config
disablement, cancellation/deadline precedence, sanitization, no application
retry, and caller-owned `ClientConn.Close`. Do not change OpenAPI, Protobuf,
runtime configuration, persistence, or rollout surfaces. Run broad Go, race,
lint, and template-profile gates serially and preserve unrelated worktree changes.

- [x] T1: Default clients omit idle keepalive, round robin follows standard health through protected Watch, and existing stream proofs remain isolated.
  - Source: [`spec.md` R1](spec.md#r1--idle-client-keepalive-is-opt-in) and [R2](spec.md#r2--round-robin-routing-consumes-standard-health); [`design/overview.md` D1](design/overview.md#d1--omit-keepalive-unless-both-existing-fields-opt-in), [D2](design/overview.md#d2--one-service-config-owns-address-selection-and-standard-health), and [the fixed file map](design/overview.md#go-ownership-and-file-map); [`test-plan.md` TD-R1-01..05](test-plan.md#r1--idle-keepalive-is-opt-in) and [TD-R2-01..10](test-plan.md#r2--round-robin-consumes-standard-health).
  - Owner/surface/resources: `internal/infra/grpcclient/{client.go,load_balancing.go,doc.go,propagation.go}`; `docs/{grpc.md,repo-architecture.md,project-structure-and-module-organization.md,first-production-feature.md}`; proof in `internal/infra/grpcclient/{client_test.go,keepalive_test.go,load_balancing_test.go,health_checking_test.go,propagation_test.go,transparent_retry_test.go}` with `keepalive_parity_test.go` removed; composed proof renames `internal/infra/oidcjwt/grpc_tls_test.go` to `grpc_tls_contract_test.go`, updates its pointer in `grpc_test.go`, and updates `scripts/{init-module.sh,ci/template-init-check.sh}`. `health_checking_test.go` alone owns test-local health backends/state; manual resolver schemes are distinct and top-level sequential; the raw keepalive peer owns one bounded 35-second window. Every client in `propagation_test.go` and the client in `transparent_retry_test.go` sets `HealthCheck=false`; their shared harnesses and propagation/transparent-retry oracles do not change. `grpcclient.Options` accepts but never creates one connection credential, and the existing sanitizer remains its only metadata-policy owner.
  - Depends on: none.
  - Proof: TD-R1-01..04 and TD-R2-01..08; `go test ./internal/infra/grpcclient -count=1`; the exact config, pre-I/O rejection, PING/HEADERS, backend-count, in-flight, fallback, callability, Watch-cancellation, propagation, and two-attempt retry oracles pass. TD-R2-09; `go test -vet=off ./internal/infra/oidcjwt -run '^TestGRPCAuthnBoundaryOverTLS$' -count=1`; the unauthenticated automatic-Watch control stays ineligible despite valid call-scoped auth, while the sanitized dial credential authenticates both the protected Watch and an application RPC. TD-R2-10; `make project-structure-check && TEMPLATE_INIT_PROFILE=authn make template-init-check`; enabled generated profiles contain only `grpc_tls_contract_test.go`, and `GRPC=none` profiles contain neither old nor new gRPC TLS test. TD-R1-05; `go test google.golang.org/grpc -run '^Test/ClientUpdatesParamsAfterGoAway$' -count=1`; the pinned dependency still owns adaptive interval recovery. TD-GATE-02; `go test -vet=off -count=5 -shuffle=on ./internal/infra/grpcclient -run '^TestRoundRobinHealth'`; `go test -vet=off -count=5 -shuffle=on ./internal/infra/grpcclient -run '^TestHealth'`; `go test -vet=off -count=5 -shuffle=on ./internal/infra/grpcclient -run '^TestClientConnCloseCancelsHealthWatch$'`; `go test -vet=off -count=5 -shuffle=on ./internal/infra/grpcclient -run '^TestPropagationPoliciesApplyToUnaryAndStreamingRPCs$'`; `go test -vet=off -count=5 -shuffle=on ./internal/infra/grpcclient -run '^TestTransparentRetryReappliesClosedPropagationPerAttempt$'`; every health and isolated-stream oracle is order-stable. TD-GATE-03; `go test -vet=off -race ./internal/infra/grpcclient ./internal/infra/grpc ./internal/infra/oidcjwt`; no client-health, keepalive, in-flight, close, server lifecycle, or protected-Watch race is reported.
  - Reopen if: `go.mod` moves from grpc-go v1.82.1, direct `pick_first` gains a health listener, a dependency needs a non-empty health service name or authentication that cannot use `PerRPCCredentials`, or the raw transport no longer exposes the accepted bounded PING/HEADERS oracle — Technical Design owns runtime/placement changes and Test Design owns oracle replacement.
  - Accepted: fixed candidate manifest SHA-256 `391fc75ec374be262b8d8e80cc8038d263e1cd6baca9a2a42834e4910f7eb387`; all mapped focused, shuffle, upstream, race, structure, and `authn` profile proofs passed. Independent Implementation Review returned `PASS` with no findings; unrelated OIDC, outbox, and workflow hunks were excluded.

- [x] T2: One transport-neutral classification produces `already_exists` over the composed HTTP and gRPC paths while every existing failure meaning and supported template profile survives.
  - Source: [`spec.md` R3](spec.md#r3--domain-failure-identity-is-transport-neutral); [`design/overview.md` D3-D6](design/overview.md#d3--internalfailure-owns-transport-neutral-classification) and [the fixed file map](design/overview.md#go-ownership-and-file-map); [`test-plan.md` TD-R3-01..10](test-plan.md#r3--domain-failure-identity-is-transport-neutral) and [aggregate gates](test-plan.md#carried-compatibility-and-aggregate-gates).
  - Owner/surface/resources: add `internal/failure/{failure.go,failure_test.go}` and `examples/reference-service/internal/article/{errors.go,errors_test.go}`; remove `internal/problem/mapping.go`; change `internal/problem/{problem.go,problem_test.go}`, `internal/infra/http/{router.go,domain_errors_test.go}`, `internal/infra/grpc/{config.go,status.go,doc.go,docs_test.go,error_details_test.go,interceptors_test.go}`, `cmd/service/internal/bootstrap/{startup_dependencies.go,startup_dependencies_test.go,startup_grpc.go,run.go}`, `examples/reference-service/internal/article/article.go`, `examples/reference-service/internal/httpapi/{problem.go,handler.go,router_test.go}`, `examples/reference-service/{reference.go,reference_test.go}`, `examples/grpc-reference-service/service.go`, `.golangci.yml`, `scripts/profiles/database-none/startup_dependencies.go.tmpl`, `scripts/{init-module.sh,ci/template-init-check.sh}`, and the four shared documentation files from T1; add `examples/reference-service/grpc_failure_mapping_contract_test.go`. The template-profile scripts own temporary checkouts and run serially.
  - Depends on: T1 — output handoff — needed to start.
  - Handoff: T1 supplies the accepted client/protected-health source and test state, renamed OIDC contract proof, updated shared profile scripts, and shared documentation; T2 consumes those exact files as its base, preserves T1 behavior, and runs the final aggregate proof over both units.
  - Proof: TD-R3-01..09 and TD-GATE-01; `go test ./internal/failure ./internal/problem ./internal/infra/http ./internal/infra/grpc ./internal/infra/grpcclient ./cmd/service/internal/bootstrap ./examples/reference-service/... ./examples/grpc-reference-service/... -count=1`; neutral vocabulary, HTTP-only `conflict`, exact HTTP 409 and gRPC `ALREADY_EXISTS`/`ErrorInfo`, existing producer meanings, precedence, sanitization, and both compositions pass. TD-R3-10 and TD-GATE-05; `make project-structure-check`; `make lint`; `make proto-check`; `make openapi-check`; `TEMPLATE_INIT_PROFILE=minimal make template-init-check`; `TEMPLATE_INIT_PROFILE=grpc make template-init-check`; lint enforces the leaf dependency boundary, generated contracts are unchanged, and both profile commands pass. Inside the existing `grpc` profile branch, a dedicated `DATABASE=none GRPC=none REFERENCE_EXAMPLE=keep` checkout must pass `go test ./...` and `go build ./cmd/service`, retain `examples/reference-service`, `internal/failure`, and the HTTP `already_exists` proof, and omit `grpc_failure_mapping_contract_test.go`, `examples/grpc-reference-service`, `internal/infra/grpc`, and `internal/infra/grpcclient`; the enabled fixture retains and runs the composed gRPC contract. TD-GATE-04; `make test`; the repository unit aggregate passes after the focused proof.
  - Reopen if: a shipped consumer requires the old collision identity, a real producer needs a new caller action, profile ownership changes, or production composition cannot carry one neutral mapper slice — Specification owns identity/behavior and Technical Design owns seams/profiles.
  - Accepted: fixed candidate manifest SHA-256 `1f6c29be6e7302a48675362ff180318006f0ab87449d64b587985a82937eded1`; all focused, structure, lint, generated-contract, minimal/gRPC-profile, mixed-profile, and repository-unit proofs passed. Independent Implementation Review returned `PASS` with no findings; unrelated T1/OIDC/outbox/workflow hunks were excluded.

Review disposition: Independent Task Review / Readiness of candidate SHA-256
`b2cf2c86ea0bf11642e518bf3a5956ce3c1eb7ce21a0a918dcdf1173e493908c`
returned `PASS` with no surviving findings. The reviewer dry-ran T1 through
acceptance, checked TD-R3-10 and later invalidating decisions, and confirmed the
current unrelated overlaps do not change the accepted seams.

Implementation evidence later reopened only T1's protected health-Watch seam.
The ready input hashes, T1 file map, composed proof, profile assertion, race
scope, and reopen condition now consume the reviewed Specification, Technical
Design, and Test Design repairs. T2 remains unchanged and blocked on T1.
Focused Task Review / Readiness at SHA-256
`b5a7b0bb336ddd53caea9bc04445775ecc831e0aa8e18512082c2ce7c8da31ba`
returned `PASS`: T1 is executable from the repaired inputs and T2 remains
blocked on its accepted handoff. This final edit changes only status and review
disposition.
