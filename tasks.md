# Linter Hardening Task Ledger

## Context

This ledger captures the rollout plan for stricter `golangci-lint` coverage in this repository.
It is intentionally a planning artifact only. Do not treat this file as an implemented config change.

Goal: improve real Go code quality for an AI-native Go REST service template, especially code written or modified by LLMs. Prefer linters that catch bug classes, brittle tests, unsafe context usage, weak error boundaries, maintainability drift, or architecture drift before human review.

Current evidence from the read-only research pass:

- Pinned tool: `go tool golangci-lint --version` reported `v2.10.1` built with Go `1.26.2`.
- Current `.golangci.yml` uses `default: standard` and already enables many non-default checks, including `asasalint`, `asciicheck`, context, SQL, HTTP body, observability, error, nilness, and OpenAPI-adjacent safety checks.
- Current generated-code exclusions cover `internal/api/openapi.gen.go`, `internal/infra/postgres/sqlcgen/.*`, `.*_mock_test.go`, and `.*_string.go`.
- `make lint` passed with `0 issues`.
- `make guardrails-check` passed.
- `go tool golangci-lint run --enable-only=asasalint,asciicheck --timeout=3m` returned `0 issues`.
- `go tool golangci-lint run --enable-only=asasalint,asciicheck,fatcontext --timeout=3m` reported one `fatcontext` issue in `cmd/service/internal/bootstrap/startup_dependencies_additional_test.go`.
- `musttag`, `dupl`, `tagliatelle`, `tagalign`, `gomodguard`, `forbidigo`, `misspell`, `dupword`, and `prealloc` reported `0 issues` in targeted probes with the current config context.
- High-baseline or policy-heavy probes included `paralleltest` 233, `tparallel` 13, `err113` 82, `exhaustruct` 371, default `depguard` 123, default `revive` 92, `perfsprint` 42, `cyclop` 34, `funlen` 27, `wrapcheck` 14, `gocognit` 9, `containedctx` 2, `recvcheck` 1, and `nestif` 1.

Policy stance:

- Enable clean, low-noise, bug-oriented linters as CI-blocking once their small baseline is fixed.
- Start noisy or policy-heavy linters as nightly or informational until the baseline is intentionally cleaned or scoped.
- Do not enable default rule sets that mostly enforce taste, comments, or artificial package boundaries.
- Do not exclude tests wholesale. Scope or explain exceptions only where the test is intentionally serial, white-box, timing-sensitive, or checking context behavior.

## Phase 0: Reconfirm Baseline Before Each Rollout PR

- [x] T000 [Phase 0] Re-run the linter inventory and clean baseline checks before changing `.golangci.yml`. Depends on: none. Proof: `go tool golangci-lint linters --config .golangci.yml`, `go tool golangci-lint config verify`, `make lint`, `make guardrails-check`.
- [x] T001 [Phase 0] Confirm generated-code exclusions still cover OpenAPI, sqlc, mockgen, and stringer artifacts and do not hide hand-written code. Depends on: T000. Proof: inspect `.golangci.yml` exclusions and run `go tool golangci-lint run --timeout=3m`.
- [x] T002 [Phase 0] Keep `gosec` out of `.golangci.yml` unless the repo intentionally changes its security gate model. The existing `make gosec` path is cleaner because it excludes generated code and currently passes. Depends on: T000. Proof: `make gosec`.

## Phase 1: Low-Noise CI-Blocking Linters

Objective: enable high-signal linters with small or clean baselines.

Recommended linters: `musttag`, `canonicalheader`, `recvcheck`, `fatcontext`, `containedctx`, `dupl`.

- [x] T100 [Phase 1] Fix `canonicalheader` findings in `internal/infra/http/router_test.go` or document why the lowercase W3C `traceparent` spelling is intentionally required in those tests. Depends on: T000. Proof: `go tool golangci-lint run --enable-only=canonicalheader --timeout=3m`.
- [x] T101 [Phase 1] Fix `recvcheck` for `LoadReport` in `internal/config/config.go` by making receiver usage consistent without changing the public behavior. Depends on: T000. Proof: `go tool golangci-lint run --enable-only=recvcheck --timeout=3m` and `go test ./internal/config`.
- [x] T102 [Phase 1] Resolve the `fatcontext` finding in `cmd/service/internal/bootstrap/startup_dependencies_additional_test.go`. Prefer a test refactor that asserts deadline/cancellation without storing a context across the function literal boundary; use a narrow `nolint:fatcontext` only if the capture is the behavior under test and the explanation says why. Depends on: T000. Proof: `go tool golangci-lint run --enable-only=fatcontext --timeout=3m` and `go test ./cmd/service/internal/bootstrap`.
- [x] T103 [Phase 1] Resolve `containedctx` findings in `cmd/service/internal/bootstrap/startup_server.go` for `serveHTTPRuntimeArgs`. Prefer passing contexts explicitly or splitting the argument struct so runtime dependencies do not store `context.Context`. If the struct is intentionally a short-lived call argument, record that rationale and use the narrowest possible exception. Depends on: T000. Proof: `go tool golangci-lint run --enable-only=containedctx --timeout=3m` and `go test ./cmd/service/internal/bootstrap`.
- [x] T104 [Phase 1] Enable `musttag`, `canonicalheader`, `recvcheck`, `fatcontext`, `containedctx`, and `dupl` in `.golangci.yml` after T100 through T103 are clean or intentionally scoped. Depends on: T100, T101, T102, T103. Proof: `go tool golangci-lint run --enable-only=musttag,canonicalheader,recvcheck,fatcontext,containedctx,dupl --timeout=3m`.
- [x] T105 [Phase 1] Run full quick validation for the Phase 1 PR. Depends on: T104. Proof: `go tool golangci-lint config verify`, `make lint`, `make check`, `make guardrails-check`.

Exit criteria:

- Phase 1 linters are CI-blocking.
- Baseline is clean or every exception is narrow, path-specific, and explained.
- No generated-file exclusions were broadened to hide real code.

## Phase 2: Error Boundary Quality

Objective: catch weak error wrapping at package and dependency boundaries without forcing low-value wrapping inside local helpers.

Recommended linters: `wrapcheck`; optionally a configured `revive` subset for non-style rules.

- [x] T200 [Phase 2] Audit the current `wrapcheck` findings and classify each as a real boundary error, intentional passthrough, or context sentinel passthrough. Current classification: 14 findings; startup telemetry, HTTP server, dependency probe, and Postgres parse returns were real boundary wrapping issues, while `context.Err()` returns were sentinel cancellation paths that must preserve `errors.Is`. Depends on: Phase 1 complete. Proof: `go tool golangci-lint run --enable-only=wrapcheck --timeout=3m --max-issues-per-linter=0 --max-same-issues=0`.
- [x] T201 [Phase 2] Decide and record the repository rule for `context.Err()` and probe cancellation errors. Do not make implementation infer whether `context.Canceled` and `context.DeadlineExceeded` should be returned directly or wrapped at each boundary. Depends on: T200. Proof: short note in the PR description or a repo doc touched by the PR if the policy becomes durable.
- [x] T202 [Phase 2] Fix real `wrapcheck` boundary issues in startup bootstrap, dependency probes, HTTP server wrappers, and Postgres config parsing. Boundary errors now add operation context with `%w`; sentinel context errors still preserve `errors.Is`. Depends on: T200, T201. Proof: `go tool golangci-lint run --enable-only=wrapcheck --timeout=3m` plus targeted `go test` for changed packages.
- [x] T203 [Phase 2] Evaluate a configured `revive` subset only. Candidate rules: `context-as-argument`, `unused-parameter`, and other non-comment rules that catch correctness or maintainability issues. Do not enable default `revive` because the observed baseline is dominated by package and exported comment noise. Configured subset: `context-as-argument`, `unused-parameter`; initial six findings were fixed and the targeted run is clean. Depends on: T000. Proof: targeted `go tool golangci-lint run --enable-only=revive --timeout=3m` with the proposed config.
- [x] T204 [Phase 2] Enable `wrapcheck` once T202 is clean. Enable configured `revive` only if T203 proves a low-noise rule subset. Depends on: T202, optional T203. Proof: `make lint`, `make check`, `make guardrails-check`.

Exit criteria:

- External-package and interface-returned errors are wrapped or intentionally exempted.
- The repo has an explicit stance on direct `context.Err()` passthrough.
- No default `revive` style rules become CI-blocking by accident.

## Phase 3: Test Quality And Parallelism

Objective: make LLM-generated tests faster and less brittle without breaking tests that must stay sequential.

Candidate linters: `paralleltest`, `tparallel`. `testifylint` stays out unless the repo adopts `testify`.

- [x] T300 [Phase 3] Inventory tests reported by `paralleltest` and `tparallel` and classify them as safe parallel, intentionally serial, or needing refactor before parallelization. Depends on: Phase 1 complete. Proof: `go tool golangci-lint run --enable-only=paralleltest,tparallel --timeout=3m --max-issues-per-linter=0 --max-same-issues=0` initially reported 246 issues (`paralleltest` 233, `tparallel` 13); after the first rollout it reports the remaining non-blocking baseline of 146 issues (`paralleltest` 137, `tparallel` 9).
- [x] T301 [Phase 3] Add `t.Parallel()` to safe top-level tests and subtests, starting with pure unit tests that do not mutate environment, start listeners on shared ports, rely on global telemetry state, use goleak-sensitive scheduling, or assert wall-clock timing. Depends on: T300. Proof: added `t.Parallel()` to safe pure/local cases in `cmd/service/internal/bootstrap`, `internal/app`, `internal/config`, `internal/infra/http`, `internal/infra/postgres`, `internal/infra/telemetry`, and `internal/observability/otelconfig`; targeted package tests passed.
- [x] T302 [Phase 3] For intentionally serial tests, add narrow exclusions or `nolint` comments with concrete reasons such as env mutation, goleak scope, process signal behavior, global OpenTelemetry state, or timing-sensitive shutdown behavior. Depends on: T300. Proof: added narrow telemetry `nolint` comments for global OpenTelemetry state and `t.Setenv` incompatibility; broader remaining exceptions stay visible through the informational target instead of being hidden by broad exclusions.
- [x] T303 [Phase 3] Run flake-oriented validation after parallelization changes. Depends on: T301, T302. Proof: `make test`, `make test-race`, and `make test-flake-smoke`.
- [x] T304 [Phase 3] Add `paralleltest` and `tparallel` as nightly or informational first if the remaining exception surface is large. Promote them to CI-blocking only after the baseline is stable and the serial-test policy is accepted. Depends on: T303. Proof: added `make test-parallelism-check`, `make docker-test-parallelism-check`, and a nightly `continue-on-error` step; `make lint` stays clean because `paralleltest` and `tparallel` were not promoted to CI-blocking.

Exit criteria:

- Parallel-safe tests run in parallel.
- Serial tests have explicit, reviewable reasons.
- Flake/race checks pass after changes.

## Phase 4: Maintainability Drift Controls

Objective: catch LLM-driven function growth, nested branching, and copy-paste before it becomes hard to review.

Recommended linters: `gocognit`, `cyclop`, `nestif`, and `dupl`.
Avoid CI-blocking `funlen` and `maintidx` for now.

- [x] T400 [Phase 4] Keep `dupl` enabled from Phase 1 if it remains clean. If it starts reporting issues later, treat them as real copy-paste review signals unless a generated or table-fixture exception is justified. Depends on: T104. Proof: `go tool golangci-lint run --enable-only=dupl --timeout=3m` returned `0 issues`.
- [x] T401 [Phase 4] Tune `gocognit` and `cyclop` thresholds using current production-code findings. Avoid blindly accepting default `cyclop` threshold 10 if it mainly flags dense but stable tests. Decision: keep `gocognit` at 30, set `cyclop` to 20 for production code, and keep `nestif` at 5. Depends on: Phase 1 complete. Proof: `go tool golangci-lint run --enable-only=gocognit,cyclop,nestif --timeout=3m --max-issues-per-linter=0 --max-same-issues=0` returned `0 issues`.
- [x] T402 [Phase 4] Refactor production hotspots first, especially config loading/validation, startup dependency initialization, HTTP middleware validation, and Postgres DSN parsing. Config snapshot/validation and Mongo probe normalization were refactored; remaining startup, HTTP, Postgres, and telemetry complexity is below the tuned production threshold. Depends on: T401. Proof: `go test -count=1 ./internal/config ./cmd/service/internal/bootstrap ./internal/infra/http ./internal/infra/postgres ./internal/infra/telemetry` passed, and the tuned linter command returned `0 issues`.
- [x] T403 [Phase 4] Decide test policy for complexity linters. Policy: exclude `_test.go` only for `gocognit`, `cyclop`, and `nestif` for now, so table-heavy tests are not forced into artificial helper extraction while other test lint gates remain active. Depends on: T401. Proof: config review plus targeted linter run returned `0 issues`.
- [x] T404 [Phase 4] Enable tuned `gocognit`, `cyclop`, and `nestif` once the production baseline is clean and test policy is explicit. Depends on: T402, T403. Proof: `make lint`, `make check`, and `make guardrails-check` passed.

Exit criteria:

- Complexity gates protect production code from LLM bloat.
- Test fixtures are not refactored purely to satisfy arbitrary line or branch counts.
- `funlen` remains informational or avoided unless a future PR proves a narrow, useful config.

## Phase 5: Architecture And Dependency Policy

Objective: turn durable import and dependency rules into automated checks without using default deny-all behavior.

Candidate linters: configured `depguard`, `gomodguard`, configured `forbidigo`.

- [x] T500 [Phase 5] Design a project-specific `depguard` policy that complements, rather than replaces, `scripts/ci/required-guardrails-check.sh`. Do not enable default `depguard`; it reported 123 findings because no allow/deny policy was configured. Implemented as deny-only `lax` lists in `.golangci.yml`, not a default allowlist: app/domain stay transport/infra agnostic, driver and sqlc imports stay behind the Postgres adapter, chi stays in HTTP/generated surfaces, and alternate router/logging packages are denied. Depends on: Phase 1 complete. Proof: `go tool golangci-lint run --enable-only=depguard --timeout=3m --max-issues-per-linter=0 --max-same-issues=0` returned `0 issues`.
- [x] T501 [Phase 5] Cover import-boundary risks that LLMs are likely to introduce: app/domain importing infra adapters, direct driver imports outside infra, generated sqlc imports outside approved packages, and accidental use of alternate routers or logging libraries. Depends on: T500. Proof: targeted `depguard` run returned `0 issues`; `make guardrails-check` passed; a throwaway synthetic module caught app-to-api/infra/sqlc imports, chi outside the HTTP layer, gorilla/mux, pgx, and zerolog.
- [x] T502 [Phase 5] Add `gomodguard` only if the repo adopts explicit dependency policy, such as banning deprecated or superseded modules or requiring approved replacements. Decision: keep `gomodguard` out for now because the current tooling module graph includes several likely deny-list candidates as explicit indirect dependencies, so module-level blocking would be noisy until tooling dependencies are separated or a stricter direct-dependency policy exists. Depends on: T500. Proof: `go tool golangci-lint run --enable-only=gomodguard --timeout=3m --max-issues-per-linter=0 --max-same-issues=0` returned `0 issues` with no policy, and module graph inspection showed router/logging/driver candidates present through tooling dependencies.
- [x] T503 [Phase 5] Add configured `forbidigo` only with concrete forbidden APIs, for example `fmt.Print*` or `log.Print*` in runtime code. Implemented runtime-code rules for `fmt.Print*`, built-in `print`/`println`, and stdlib `log.Print*`/`Fatal*`/`Panic*`; tests are excluded, and the built-in `panic` function is not forbidden in this phase. Depends on: T500. Proof: `go tool golangci-lint run --enable-only=forbidigo --timeout=3m --max-issues-per-linter=0 --max-same-issues=0` returned `0 issues`; a throwaway synthetic module caught `fmt.Println` and `log.Print`.
- [x] T504 [Phase 5] Enable the configured architecture/dependency policy only after synthetic or real examples prove it catches the intended drift without blocking approved imports. Enabled `depguard` and `forbidigo`; `gomodguard` remains deferred by T502. Depends on: T501 and optional T502/T503. Proof: `go tool golangci-lint config verify`, targeted `depguard`, `forbidigo`, and `gomodguard` runs, the throwaway synthetic negative check, `make lint`, `make guardrails-check`, and `make check` passed.

Exit criteria:

- Architecture rules are explicit and repo-specific.
- No default deny-all `depguard` policy is enabled.
- New rules catch realistic LLM drift patterns.

## Phase 6: Struct And Tag Discipline

Objective: strengthen serialization and config boundary discipline without forcing every test fixture and zero-value struct literal to become noisy.

Candidate linters: configured `tagliatelle`; scoped `exhaustruct` only if practical.
Avoid global `tagalign`.

- [x] T600 [Phase 6] Decide whether `tagliatelle` should enforce JSON, YAML, or koanf tag naming conventions. Decision: enable `tagliatelle` with explicit `snake` rules for `json`, `yaml`, and `koanf` tags, without `use-field-name`, because OpenAPI and config keys are contract-owned and already use snake_case such as `request_id`, `feature_flags`, and `otlp_traces_endpoint`. Depends on: Phase 1 complete. Proof: `go tool golangci-lint run --enable-only=tagliatelle --timeout=3m` returned `0 issues`.
- [x] T601 [Phase 6] Investigate whether `exhaustruct` can be scoped narrowly to boundary structs where missing fields are risky, such as config snapshots or API-facing DTOs. Decision: avoid global `exhaustruct` after the fresh global probe reported 376 issues; enable only `internal/config` snapshot structs and `internal/api.Problem`, exclude `_test.go`, and allow empty returns for error paths. Depends on: T600. Proof: `go tool golangci-lint run --enable-only=tagliatelle,exhaustruct --timeout=3m --max-issues-per-linter=0 --max-same-issues=0` returned `0 issues`, and `go test ./internal/config ./internal/infra/http` passed.
- [x] T602 [Phase 6] Avoid `tagalign` as a quality gate. Decision: keep `tagalign` out because it is formatting/style and `gofumpt` owns formatting discipline. Depends on: none. Proof: `go tool golangci-lint run --enable-only=tagalign --timeout=3m` returned `0 issues`; recorded as no-action here.
- [x] T603 [Phase 6] Enable `tagliatelle` and any scoped `exhaustruct` only if the scoped config remains low-noise and catches real boundary omissions. Enabled `tagliatelle` plus scoped `exhaustruct`; config snapshot builders now construct exhaustive boundary literals, and `api.Problem` initializes its optional request ID explicitly before conditional override. Depends on: T600, optional T601. Proof: `make lint`, `go test -count=1 ./internal/config ./internal/infra/http`, `make check`, and `make guardrails-check` passed.

Exit criteria:

- Tag rules protect serialization/config contracts.
- Struct exhaustiveness is used only where omitted fields are a real defect class.
- Style-only tag alignment does not become a CI gate.

## Deferred Or Avoid For Now

- [ ] T700 [Deferred] Revisit `err113` only if the repo adopts a durable sentinel/static error policy. Current baseline is 82 issues across tests and local parse/validation helpers, so enabling it now would create policy churn more than bug prevention. Proof when revisited: `go tool golangci-lint run --enable-only=err113 --timeout=3m --max-issues-per-linter=0 --max-same-issues=0`.
- [ ] T701 [Deferred] Avoid `testpackage` unless the repo intentionally abandons white-box testing of internals. Current baseline is 25 issues and conflicts with package-internal test patterns. Proof when revisited: `go tool golangci-lint run --enable-only=testpackage --timeout=3m`.
- [ ] T702 [Deferred] Avoid default `revive`; consider only configured non-style rules in Phase 2. Current default baseline is 92 issues, mostly comments and package naming. Proof when revisited: targeted `revive` run with a proposed rule subset.
- [ ] T703 [Deferred] Avoid global `exhaustruct`; consider only scoped Phase 6 use. Current baseline is 371 issues. Proof when revisited: targeted `exhaustruct` run with scoped settings.
- [ ] T704 [Deferred] Keep `perfsprint`, `intrange`, `prealloc`, `misspell`, and `dupword` out of the core quality gate unless a future PR shows they catch real defects in this repo. `perfsprint` reported 42 mostly mechanical suggestions, `intrange` 3, and the others were clean. Proof when revisited: targeted linter runs.
- [ ] T705 [Deferred] Keep `testifylint` out until the repo adopts `testify`. Current targeted run reported `0 issues` because there is no meaningful testify surface. Proof when revisited: targeted `testifylint` run after dependency adoption.
- [ ] T706 [Deferred] Avoid `maintidx` for now. It reported `0 issues`, but the signal is less actionable than `gocognit`, `cyclop`, `nestif`, and `dupl`. Proof when revisited: targeted `maintidx` run with proposed threshold.

## Cross-Phase Validation Commands

Use the smallest relevant proof for each PR, then run the broader checks before closeout:

```sh
go tool golangci-lint --version
go tool golangci-lint linters --config .golangci.yml
go tool golangci-lint config verify
go tool golangci-lint run --timeout=3m
make lint
make guardrails-check
make check
```

For stricter PRs that claim broader CI readiness, add:

```sh
make test-race
make test-flake-smoke
make gosec
make check-full
```

## Implementation Readiness

Status: CONCERNS.

Reason: the rollout is ready to continue phase-by-phase, but remaining later phases still need explicit policy decisions before becoming CI-blocking. The accepted risks are:

- `gomodguard` remains deferred until tooling dependencies are separated or the repo adopts a stricter direct-dependency policy.
- `tagliatelle` needs an explicit JSON, YAML, or koanf tag naming policy before it is useful as a quality gate.
- `exhaustruct` needs a narrow boundary-struct scope before it is useful as a quality gate.

Phase 1 through Phase 5 are now complete in this ledger. Next implementation session should start with Phase 6 (`T600` through `T603`) only after the struct/tag policy is intentionally designed rather than enabled from default linter behavior.
