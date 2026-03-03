# 65 Coder Detailed Plan: Spec 01 (`go tool` + `tool` directives)

## Execution Context
Scope boundaries:
- Migrate Go-based developer tooling to `go.mod` tool directives + `go tool` execution style.
- Keep behavior of existing make targets and CI checks functionally equivalent.
- Update docs for the new tool management baseline.

Non-goals:
- Runtime/service behavior changes.
- API contract changes.
- Migration of Node-based OpenAPI lint command (`npx`) in this task set.

Critical invariants:
- INV-S01-1: `go.mod` is single source of truth for Go tool versions.
- INV-S01-2: No mixed command style for migrated tools (`go run ...@` is removed for migrated scope).
- INV-S01-3: Existing quality gates remain green with equivalent intent.

Forbidden changes:
- Replacing existing quality gates with weaker alternatives.
- Introducing floating tool versions.

## Execution Mode
- Mode: `batch`
- Checkpoint policy: checkpoint after each `2-3` tasks.
- Coder autonomy: implementation details of make target internals remain coder-defined as long as outcome/invariants/evidence stay satisfied.

## Task Graph
- S01-T01 -> S01-T02 -> S01-T03 -> S01-T04 -> S01-T05
- S01-T03 depends on S01-T02.
- S01-T04 depends on S01-T03.
- S01-T05 depends on S01-T04.

## Task Cards

### Task ID
S01-T01

Objective:
- Build closure map for all Go tool invocations currently executed via `go run ...@version` and map them to target `tool` directives.

Spec Traceability:
- Decisions: S01-D1, S01-D2, S01-D4
- Invariants: INV-S01-1, INV-S01-2
- Test obligations: S01-TST-BASELINE

Change Surface:
- Tooling layer: `go.mod` dependency governance.
- Build orchestration: make target command style inventory.

Task Sequence:
1. Enumerate current Go tool invocations from Makefile and `go:generate` directives.
2. Classify each into: migrate-now / defer (if outside scope).
3. Produce explicit migration list used by next tasks.

Verification Commands:
- `rg "go run .*@" Makefile internal/api/doc.go`

Expected Evidence:
- Explicit list of migrated tool commands and deferred commands.
- No unresolved command in mandatory scope.

Review Checklist:
- Complete inventory captured.
- No mandatory tool omitted.
- Scope boundaries respected.
- Deferred items explicitly justified.

Ambiguity Triggers:
- If command cannot be mapped to a stable `go tool` name.

Change Reconciliation:
- Expected surface only; actual touched modules/files recorded during execution.

Execution Evidence (2026-03-03):
- Verification command executed:
  - `rg "go run .*@" Makefile internal/api/doc.go`
- Inventory and closure map (source-constrained to Makefile + `go:generate` in `internal/api/doc.go`):
  - Migrate-now (mandatory scope):
    - `golang.org/x/tools/cmd/goimports` -> target directive: `tool golang.org/x/tools/cmd/goimports` -> target execution: `go tool goimports` (Makefile: `fmt`, `fmt-check`).
    - `github.com/golangci/golangci-lint/v2/cmd/golangci-lint` -> target directive: `tool github.com/golangci/golangci-lint/v2/cmd/golangci-lint` -> target execution: `go tool golangci-lint` (Makefile: `lint`).
    - `golang.org/x/vuln/cmd/govulncheck` -> target directive: `tool golang.org/x/vuln/cmd/govulncheck` -> target execution: `go tool govulncheck` (Makefile: `go-security`).
    - `github.com/securego/gosec/v2/cmd/gosec` -> target directive: `tool github.com/securego/gosec/v2/cmd/gosec` -> target execution: `go tool gosec` (Makefile: `go-security`).
    - `github.com/zricethezav/gitleaks/v8` -> target directive: `tool github.com/zricethezav/gitleaks/v8` -> target execution: `go tool gitleaks` (Makefile: `secrets-scan`).
    - `github.com/getkin/kin-openapi/cmd/validate` -> target directive: `tool github.com/getkin/kin-openapi/cmd/validate` -> target execution: `go tool validate` (Makefile: `openapi-validate`).
    - `github.com/oasdiff/oasdiff` -> target directive: `tool github.com/oasdiff/oasdiff` -> target execution: `go tool oasdiff` (Makefile: `openapi-breaking`).
    - `github.com/golang-migrate/migrate/v4/cmd/migrate` -> target directive: `tool github.com/golang-migrate/migrate/v4/cmd/migrate` -> target execution: `go tool migrate` (Makefile: `migration-validate`).
    - `github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen` -> target directive: `tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen` -> target execution: `go tool oapi-codegen` (`internal/api/doc.go` `go:generate`).
  - Defer (explicitly out of this detailed-plan scope):
    - `go.uber.org/mock/mockgen` -> deferred to Spec 02 (`65-coder-detailed-plan-02-mockgen.md`).
    - `golang.org/x/tools/cmd/stringer` -> deferred to Spec 03 (`65-coder-detailed-plan-03-stringer.md`).
    - `github.com/sqlc-dev/sqlc/cmd/sqlc` -> deferred to Spec 04 (`65-coder-detailed-plan-04-sqlc.md`).
- Resolution summary:
  - Mandatory scope has explicit tool-directive target mapping for every discovered `go run ...@` command in `Makefile` and `internal/api/doc.go`.
  - No unresolved command remains in mandatory S01 scope.

Progress Status:
- `done`

### Task ID
S01-T02

Objective:
- Add required tool directives in `go.mod` and normalize module graph.

Spec Traceability:
- Decisions: S01-D1, S01-D3
- Invariants: INV-S01-1
- Test obligations: S01-TST-MOD

Change Surface:
- Module/toolchain governance area (`go.mod`, `go.sum`).

Task Sequence:
1. Add missing Go tool directives.
2. Run tidy/verify flow.
3. Confirm deterministic module graph and no unexpected drift.

Verification Commands:
- `make mod-check`

Expected Evidence:
- `go.mod` contains target tool directives.
- `make mod-check` passes.

Review Checklist:
- Versions pinned.
- No floating versions introduced.
- No unrelated dependency churn.
- Tool directives match inventory from S01-T01.

Ambiguity Triggers:
- If a tool module path conflicts with existing module requirements.

Change Reconciliation:
- Expected: module metadata area; deviations require rationale.

Execution Evidence (2026-03-03):
- Commands executed:
  - `go get -tool <tool>@<pinned-version>` for all S01 mandatory tools from inventory.
  - `go mod tidy`
  - `make mod-check` (pass)
- `go.mod` tool directives now include mandatory S01 scope:
  - `github.com/getkin/kin-openapi/cmd/validate`
  - `github.com/golang-migrate/migrate/v4/cmd/migrate`
  - `github.com/golangci/golangci-lint/v2/cmd/golangci-lint`
  - `github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen`
  - `github.com/oasdiff/oasdiff`
  - `github.com/securego/gosec/v2/cmd/gosec`
  - `github.com/zricethezav/gitleaks/v8`
  - `golang.org/x/tools/cmd/goimports`
  - `golang.org/x/vuln/cmd/govulncheck`
  - `gotest.tools/gotestsum` (already present, preserved)
- Pinned versions resolved in `go.mod` (no floating versions):
  - `github.com/oapi-codegen/oapi-codegen/v2 v2.6.0`
  - `golang.org/x/tools v0.42.0`
  - `github.com/golangci/golangci-lint/v2 v2.10.1`
  - `golang.org/x/vuln v1.1.4`
  - `github.com/securego/gosec/v2 v2.24.7`
  - `github.com/zricethezav/gitleaks/v8 v8.30.0`
  - `github.com/getkin/kin-openapi v0.133.0`
  - `github.com/oasdiff/oasdiff v1.11.10`
  - `github.com/golang-migrate/migrate/v4 v4.19.1`
  - `gotest.tools/gotestsum v1.13.0`
- Change reconciliation:
  - Actual touched files: `go.mod`, `go.sum`.
  - Module graph expansion is confined to tool dependency closure introduced by new tool directives.
  - Notable version lift (`go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp` -> `v0.61.0`) is enforced by Go MVS under the pinned tool graph and is re-applied by `go mod tidy` when attempting downgrade.

Progress Status:
- `done`

### Task ID
S01-T03

Objective:
- Migrate command execution style from `go run ...@version` to `go tool ...` for in-scope tools.

Spec Traceability:
- Decisions: S01-D2, S01-D4
- Invariants: INV-S01-2
- Test obligations: S01-TST-CMD-EQUIV

Change Surface:
- Build/developer workflow layer (Makefile targets, `go:generate` directives).

Task Sequence:
1. Update Makefile tool invocations for in-scope commands.
2. Update OpenAPI generation directive to `go tool`.
3. Preserve command arguments and target semantics.

Verification Commands:
- `rg "go run .*@" Makefile internal/api/doc.go`
- `make openapi-check`

Expected Evidence:
- No migrated command remains on `go run ...@` style.
- OpenAPI generation/check flow still passes.

Review Checklist:
- Behavior parity preserved.
- No accidental command flag changes.
- `go tool` names resolve correctly.
- Drift checks still meaningful.

Ambiguity Triggers:
- If `go tool` invocation differs semantically from current `go run` path.

Change Reconciliation:
- Expected: build/codegen command surfaces; deviations justified.

Execution Evidence (2026-03-03):
- Updated in-scope command surfaces:
  - `Makefile`: migrated `fmt`, `fmt-check`, `lint`, `go-security`, `secrets-scan`, `openapi-validate`, `openapi-breaking`, `migration-validate` from `go run ...@` to `go tool ...`.
  - `internal/api/doc.go`: migrated `go:generate` to `go tool oapi-codegen`.
- Verification commands executed:
  - `rg "go run .*@" Makefile internal/api/doc.go` -> no matches.
  - `make openapi-check` -> pass (`go generate`, runtime contract test, Redocly lint, `go tool validate`).
- Argument/flag parity preserved:
  - Existing subcommands and flags retained (`oasdiff breaking --fail-on ERR`, `gitleaks git --no-banner --redact --exit-code 1`, migrate up/down sequence, etc.).

Progress Status:
- `done`

### Task ID
S01-T04

Objective:
- Clean up redundant version constants and align docs with new tool baseline.

Spec Traceability:
- Decisions: S01-D1, S01-D2
- Invariants: INV-S01-1
- Test obligations: S01-TST-DOC

Change Surface:
- Build config readability and developer docs.

Task Sequence:
1. Remove now-redundant version constants for migrated tools.
2. Update command docs to reflect `go tool` baseline.
3. Add short rule for adding new Go tools.

Verification Commands:
- `make fmt-check`
- `make lint`

Expected Evidence:
- No orphaned version constants for migrated tools.
- Docs reflect current command style.
- Static checks still pass.

Review Checklist:
- Docs match actual commands.
- No undocumented behavior changes.
- Keep non-goals intact.
- Readability improved.

Ambiguity Triggers:
- If doc guidance conflicts with existing onboarding flow.

Change Reconciliation:
- Expected: docs + build metadata; deviations noted.

Execution Evidence (2026-03-03):
- Build metadata cleanup:
  - Removed redundant migrated-tool version constants from `Makefile` (`KIN_OPENAPI_VALIDATE_VERSION`, `OASDIFF_VERSION`, `MIGRATE_VERSION`, `GOLANGCI_LINT_VERSION`, `GOVULNCHECK_VERSION`, `GOSEC_VERSION`, `GOIMPORTS_VERSION`, `GITLEAKS_VERSION`).
  - `REDOCLY_CLI_VERSION` kept (Node-based tool is non-goal for this spec).
- Docs alignment (`docs/build-test-and-development-commands.md`):
  - Updated command examples/behavior notes to `go tool` baseline for `fmt`, `fmt-check`, `lint`, `openapi-validate`, `openapi-breaking`, `go-security`, `secrets-scan`, `migration-validate`.
  - Added short rule section for onboarding new Go developer tools under `go tool` baseline.
- Verification commands executed:
  - `make fmt-check` -> pass.
  - `make lint` -> fail due to pre-existing `contextcheck` finding in `cmd/service/main.go` (outside S01 tooling/doc scope; unchanged file).
  - `rg "..._VERSION"` for removed migrated-tool constants in `Makefile` -> no matches.

Progress Status:
- `done`

### Task ID
S01-T05

Objective:
- Execute full validation set and collect evidence pack.

Spec Traceability:
- Decisions: S01-D5
- Invariants: INV-S01-3
- Test obligations: S01-TST-FULL

Change Surface:
- Verification layer only.

Task Sequence:
1. Run mandatory command suite.
2. Record pass/fail evidence.
3. Confirm no regressions in existing quality gates.

Verification Commands:
- `make mod-check`
- `make fmt-check`
- `make lint`
- `make openapi-check`
- `make test`

Expected Evidence:
- All required checks pass.
- No unresolved drift remains.

Review Checklist:
- Evidence complete.
- Failures triaged or fixed.
- No skipped mandatory checks.
- Ready for handoff.

Ambiguity Triggers:
- If any check fails due to non-tooling unrelated pre-existing issues.

Change Reconciliation:
- Verification-only stage; no extra change surface expected.

Execution Evidence (2026-03-03):
- Mandatory suite executed (no skipped checks):
  - `make mod-check` -> pass.
  - `make fmt-check` -> pass.
  - `make lint` -> fail (`contextcheck`) in `cmd/service/main.go:200` (pre-existing, out of tooling-boilerplate scope).
  - `make openapi-check` -> pass.
  - `make test` -> pass.
- Command status summary:
  - `mod-check=0`, `fmt-check=0`, `lint=2`, `openapi-check=0`, `test=0`.
- Drift posture:
  - No unresolved OpenAPI drift in validation path (`openapi-check` passed).
  - Module graph integrity checks passed (`go mod tidy -diff`, `go mod verify`, `git diff --exit-code -- go.mod go.sum`).

Blocked Task Record:
- `request_id`: `S01-T05-BLK-001`
- `blocked_task_id`: `S01-T05`
- `ambiguity_type`: `test`
- `conflicting_sources`: `S01-T05 Expected Evidence (all required checks pass)` vs pre-existing repo lint failure in `cmd/service/main.go:200`.
- `decision_impact`: cannot claim full Spec 01 validation closure or CP-S01-3 go/no-go readiness while lint is red.
- `proposed_options`:
  - fix/waive pre-existing lint finding in this branch, then rerun full suite;
  - accept explicit temporary exception in signoff and continue with documented risk.
- `owner`: `implementation owner + repo maintainer`
- `resume_condition`: `make lint` passes (or explicit signed exception recorded).

Progress Status:
- `blocked`

## Checkpoint Plan
- CP-S01-1 (after S01-T02):
  - Confirm inventory-to-tool-directive mapping closure.
  - Go/no-go: proceed only if `make mod-check` passes.
- CP-S01-2 (after S01-T04):
  - Confirm command-style migration and doc alignment.
  - Go/no-go: proceed only if openapi and static checks pass.
- CP-S01-3 (after S01-T05):
  - Confirm full validation evidence package.
  - Go/no-go: ready for implementation completion signoff.

## Clarification Contract
Required fields for blocked tasks:
- `request_id`
- `blocked_task_id`
- `ambiguity_type` (`contract`, `invariant`, `security`, `reliability`, `test`, `other`)
- `conflicting_sources`
- `decision_impact`
- `proposed_options`
- `owner`
- `resume_condition`

Resolution policy:
- Blocked task stays `blocked` until `resume_condition` is satisfied and conflict resolution is recorded.

## Coverage Matrix
- S01-OBL-1 (`go.mod` as tool source of truth) -> S01-T01, S01-T02
- S01-OBL-2 (`go tool` command standardization) -> S01-T03
- S01-OBL-3 (drift/reproducibility preservation) -> S01-T03, S01-T05
- S01-OBL-4 (developer guidance alignment) -> S01-T04

## Execution Notes
- If pre-existing repository issues fail mandatory checks, isolate and document them separately from scope changes before closure.
