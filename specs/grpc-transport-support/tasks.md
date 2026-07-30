# Goal

status: done

Completion condition: the upstream template and both initialized profiles provide the
accepted optional native gRPC client/server capability; all four RPC
cardinalities, fail-closed security, shared lifecycle, generated-contract
workflow, documentation, and claim-matched local proof pass.

Blocked stop: if native gRPC cannot share the current admission/drain budget
without changing the accepted REST/diagnostics contract, record the failing
component proof and reopen `design.md` lifecycle ownership.

Global constraints: `.proto` is canonical before generated Go; no public
Railway ingress claim; no reflection, keepalive, global retry, `WaitForReady`,
or implicit plaintext; preserve unrelated work and the existing REST contract.

- [x] T1: Buf workflow, isolated all-cardinality reference, and capability profile ownership land atomically
  - Source: `spec.md` CAP-1/CAP-2/CAP-3/SRV-3; `design.md` Contract and tool ownership; TD-1, TD-2, TD-6
  - Owner/surface/resources: `buf*.yaml`, `scripts/run-buf.sh`, `scripts/proto.sh`, `scripts/ci/proto-check.sh`, `scripts/init-module.sh`, the minimum gRPC profile assertions in `scripts/ci/template-init-check.sh`, `Makefile`, `go.mod`, `go.sum`, `tools/go.mod`, `tools/go.sum`, `examples/grpc-reference-service/**`, generated example Go
  - Depends on: none
  - Proof: unformatted/undocumented/malformed/stale/breaking/invalid-base branches fail distinctly; format checking does not mutate source; absent-base is not applicable; omitted `GRPC` records `none` and removes owned surfaces; invalid `GRPC` mutates nothing; enabled records and retains the capability; repeated identical initialization is byte-stable; existing minimal/postgres choices remain green; all cardinalities pass; `bash ./scripts/ci/proto-check.sh`; `make proto-check`; `BASE_REF=HEAD make proto-breaking`; `make template-init-check`; `go test -vet=off ./examples/grpc-reference-service/...`
  - Reopen if: Buf cannot compile Edition 2023 Opaque source without a registry; Contract Tooling Design

- [x] T2: Shared request identity, gRPC server/client adapters, bounds, status, telemetry, and lifecycle pass focused proof
  - Source: `spec.md` SRV-1..6/CLI-1/SEC-1; `design.md` Runtime ownership; TD-3, TD-5, TD-8, TD-9, TD-10, TD-11
  - Owner/surface/resources: `internal/reqctx`, reached `internal/infra/http` request-ID/telemetry consumers and tests, `internal/infra/grpc`, `internal/infra/grpcclient`, `internal/infra/telemetry`, `internal/problem` as unchanged classification authority, `go.mod`, `go.sum`
  - Depends on: T1 — output handoff — generated reference service interfaces are needed to prove unary and streaming interceptors
  - Handoff: T1 provides generated all-cardinality client/server bindings accepted by Buf checks
  - Proof: fail-closed credentials, status provenance/disclosure, shared HTTP/gRPC correlation, application admission, native transport bounds, forced drain, and client ownership/defaults pass; `go test -vet=off ./internal/reqctx ./internal/infra/http ./internal/infra/grpc ./internal/infra/grpcclient ./internal/infra/telemetry`; `go test -vet=off -race ./internal/infra/grpc ./internal/infra/grpcclient`
  - Reopen if: native interceptor or graceful-stop contracts cannot implement the accepted status/lifecycle; Runtime Design

- [x] T3: The production composition root serves optional gRPC under the existing shared admission and drain contract
  - Source: `spec.md` SRV-1/SRV-2; `design.md` Composition and lifecycle; TD-4, TD-7, TD-12
  - Owner/surface/resources: `internal/config`, `cmd/service/internal/bootstrap`, `env/.env.example`, runtime-limit reporting, integration test
  - Depends on: T2 — output handoff — server adapter and health controller are needed to wire the production listener
  - Handoff: T2 provides a runtime-server-compatible adapter, serving-state controller, and explicit credential constructor
  - Proof: disabled default binds no port; enabled startup/health/drain/TCP behavior and preserved HTTP diagnostics pass; `go test -vet=off ./internal/config ./cmd/service/internal/bootstrap`; `go test -count=1 -tags=integration ./test/... -run GRPC`; `go test -vet=off -race ./cmd/service/internal/bootstrap`
  - Reopen if: platform grace budget cannot accommodate both application transports; Lifecycle Design

- [x] T4: CI and operator/developer documentation expose exactly the selected capability
  - Source: `spec.md` CAP-1/DEP-1; `design.md` Security, rollout, and compatibility boundaries; TD-1, TD-2, TD-12
  - Owner/surface/resources: remaining full-profile assertions in `scripts/ci/template-init-check.sh`, `.github/workflows/ci.yml`, `.golangci.yml`, `README.md`, command/config/architecture/Railway/first-feature docs, `template-owned.paths`
  - Depends on: T3 — state gate — the retained and removed runtime surfaces must be final before profile purity can be proved
  - Proof: `GRPC=none` removes owned surfaces and `enabled` retains them idempotently while existing profiles stay green; `TEMPLATE_INIT_PROFILE=grpc make template-init-check`; `make project-structure-check`; `make proto-check`; `git diff --check`
  - Reopen if: a derived profile retains a direct gRPC owner it cannot execute or removes stable aggregate behavior; Capability Ownership Design

- [x] T5: The complete bounded change is reviewed and verified at repository scope
  - Source: all accepted criteria and TD-1..TD-12
  - Owner/surface/resources: bounded working-tree diff; broad Go and CI gates are non-concurrent host resources
  - Depends on: T4 — proof gate — profile and documentation closure is required before aggregate verification
  - Proof: every acceptance claim has current equal-scope evidence; `make check`; `make proto-check`; `make template-init-check`; `make mod-tidy-check`; `git diff --check`; run broader gates only when their changed-surface trigger remains applicable
  - Reopen if: a proof failure changes behavior, ownership, or proof policy; the narrow upstream owner named by the failing scenario

- [x] T6: Independent adversarial review findings are repaired without widening the capability
  - Source: post-implementation review of SRV-2/SRV-3/SRV-4/SRV-5/SRV-6, CLI-1, CAP-1/CAP-2/CAP-3, TD-1/TD-2/TD-5/TD-6/TD-7/TD-8/TD-9/TD-10/TD-11
  - Owner/surface/resources: `grpcx` policy/status/lifecycle boundaries, bootstrap registration seams, bounded reference streams, protobuf schema policy/self-tests, profile initialization, OTel/limit/TLS/process tests, and aligned operator/design/test documentation
  - Depends on: T5 — review input — the complete implementation and its original proof surface must exist before adversarial gaps can be repaired
  - Proof: direct service-owned policy statuses and standard health semantics survive while raw handler/policy errors are sanitized; forced shutdown returns within the caller budget even for a non-cooperative handler; client-stream aggregation is count/byte bounded; descriptor-backed server telemetry and secret canaries prove bounded unary/stream signals, final sanitized status, health suppression, and unknown-method filtering; live calls prove transport/client message and received-metadata bounds plus TLS hostname verification; valid and independently mismatched TLS material plus overflow rejection prove bootstrap loading and narrowing; new proto3 and unaccepted Edition 2024 schemas fail while only a same-path baseline-retained proto3 schema passes; reference-dependent benchmark commands are retained only for `GRPC=enabled REFERENCE_EXAMPLE=keep` and are absent for both removal profiles; enabled-with-default-example-removal is idempotent; `make proto-check`; `BASE_REF=HEAD make proto-breaking`; `bash ./scripts/ci/proto-check.sh`; focused and race Go tests; production-process gRPC integration; `make mod-tidy-check`; `make check`; `make template-init-check`; `make delivery-quality`; `make go-security`; `git diff --check`; independent Task Acceptance PASS for T6 and global completion
  - Reopen if: a derived service needs a different policy provenance contract, a platform requires hard termination of non-cooperative in-process Go handlers, or the external ingress/client-language matrix changes; reopen only the corresponding Runtime, Lifecycle, Deployment, or Contract Tooling decision
  - Acceptance: PASS — reviewer: critical-reviewer-agent; evidence: focused/race/process tests, protobuf policy/drift, profile purity, full template initialization, `make check`, delivery-quality, security, and whitespace gates passed; candidate: current bounded 73-path gRPC diff (fingerprint `8a68d7f2b7c1716d9bf04c5bea38911921eca86512af0bdfb3a713d42f8c3773`).
