# Bootstrap Review Fixes Tasks

- [x] T001 [Phase 1] Add a same-package degraded dependency startup helper in `cmd/service/internal/bootstrap/startup_dependencies.go` that sets `startup_dependency_status` for the effective degraded mode and logs `startup_dependency_degraded`. Depends on: none. Proof: covered by T002/T003 and `go test ./cmd/service/internal/bootstrap`.
- [x] T002 [Phase 1] Update the Redis cache degraded path in `cmd/service/internal/bootstrap/startup_dependencies.go` to use the helper without changing its `feature_off` metric/log behavior. Depends on: T001. Proof: existing Redis bootstrap tests still pass.
- [x] T003 [Phase 1] Update the Mongo degraded path in `cmd/service/internal/bootstrap/startup_dependencies.go` to use the helper and set `startup_dependency_status{dep="mongo",mode="degraded_read_only_or_stale"} 1`. Depends on: T001. Proof: add metric assertion in `cmd/service/internal/bootstrap/startup_dependencies_additional_test.go`.
- [x] T004 [Phase 1] Simplify `dependencyInitFailure` in `cmd/service/internal/bootstrap/startup_dependencies.go` by removing the redundant context-error branch while preserving `config.ErrDependencyInit` wrapping. Depends on: none. Proof: existing dependency failure tests still pass.
- [x] T005 [Phase 1] Document `NETWORK_*` as an operational network-policy channel outside ordinary `APP__...` config in `docs/configuration-source-policy.md`, including source, precedence, fail-closed behavior, and example key families. Depends on: none. Proof: doc review plus no test impact.
- [x] T006 [Phase 1] Add commented `NETWORK_*` examples to `env/.env.example` without uncommented defaults. Depends on: T005. Proof: manual diff review confirms examples are commented.
- [x] T007 [Phase 1] Resolve `networkPolicyErrorLabels` ownership in `cmd/service/internal/bootstrap/network_policy_parsing.go` and related bootstrap code: preferred path is production logging use for policy class and reason class; fallback is moving the helper to test-only code. Depends on: T005. Proof: `rg networkPolicyErrorLabels cmd/service/internal/bootstrap` shows either production use plus tests or test-only definition, never production-defined/test-only use.
- [x] T008 [Phase 1] Run verification: `go test ./cmd/service/internal/bootstrap`, `go test ./internal/config ./cmd/service/internal/bootstrap`, `go vet ./cmd/service/internal/bootstrap`, and `gofmt -l` on touched Go files. Depends on: T001-T007. Proof: command outputs captured in final implementation response.

## Progress Notes

- 2026-04-12: T001-T008 completed. Verification passed after preserving `config.ErrDependencyInit` through the current bootstrap-local alias drift.
