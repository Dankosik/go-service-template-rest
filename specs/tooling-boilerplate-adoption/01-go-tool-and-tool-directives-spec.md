# Spec 01: Go Tooling Baseline via `go tool` + `tool` directives

## Problem
The repository already uses pinned versions in `Makefile`, but most developer tools are executed as `go run <module>@<version> ...`.
This duplicates version management across files and creates avoidable boilerplate in commands and CI maintenance.

Current baseline:
- `go` version is `1.26.0`.
- `go.mod` already contains one `tool` directive (`gotestsum`).
- OpenAPI generation and quality tooling are active and should stay reproducible.

## Goals
1. Make `go.mod` the single source of truth for Go-based developer tools.
2. Standardize command execution on `go tool <command>` everywhere possible.
3. Remove duplicated version strings from `Makefile` where tool directives are used.
4. Keep CI and local flows behavior-compatible.

## Non-Goals
- No runtime behavior changes.
- No changes to API contract semantics.
- No migration of Node-based tooling (`npx @redocly/cli`) in this phase.

## Decisions (Normative)
1. Use `tool` directives in `go.mod` for Go developer tools used by make targets and `go:generate`.
2. Use `go tool <name>` in `Makefile` and `go:generate` for tools defined in `go.mod`.
3. Keep versions pinned via module versions at add/update time; do not use floating versions.
4. Keep one command style only; no mixed `go run ...@version` and `go tool ...` for the same tool.
5. Add/keep explicit generated-artifact drift checks in CI for codegen tools.

## Tool Scope
Mandatory tool-directive candidates:
- `gotest.tools/gotestsum`
- `github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen`
- `golang.org/x/tools/cmd/goimports`
- `github.com/golangci/golangci-lint/v2/cmd/golangci-lint`
- `golang.org/x/vuln/cmd/govulncheck`
- `github.com/securego/gosec/v2/cmd/gosec`
- `github.com/zricethezav/gitleaks/v8`
- `github.com/getkin/kin-openapi/cmd/validate`
- `github.com/oasdiff/oasdiff`
- `github.com/golang-migrate/migrate/v4/cmd/migrate`

Deferred (separate specs):
- `go.uber.org/mock/mockgen`
- `golang.org/x/tools/cmd/stringer`
- `github.com/sqlc-dev/sqlc/cmd/sqlc`

## Implementation Plan

### WP-1: Tool Inventory and Pinning
- Add missing tool directives in `go.mod`.
- Run `go mod tidy` and validate deterministic module graph.
- Remove now-redundant `*_VERSION` constants from `Makefile` only for migrated tools.

### WP-2: Command Migration to `go tool`
- Replace `go run <module>@<version> ...` with `go tool <command> ...` in make targets.
- Update OpenAPI generation directive to use `go tool oapi-codegen`.
- Preserve command arguments and target behavior exactly.

### WP-3: Drift and Reproducibility Guards
- Keep/extend existing drift checks (for generated artifacts).
- Document one operational rule: if tool version changes in `go.mod`, regenerate artifacts and commit diff in the same PR.

### WP-4: Developer UX and Docs
- Update command documentation to reflect `go tool` usage.
- Add a short section in docs: "How to add a new Go developer tool".

## Validation
Mandatory evidence after implementation:
1. `make mod-check`
2. `make fmt-check`
3. `make lint`
4. `make openapi-check`
5. `make test`

Additional verification:
- `rg "go run .*@" Makefile internal/api/doc.go` must not return migrated tools.

## Rollout Strategy
1. Single PR migration is acceptable because behavior is command-level, not runtime-level.
2. If any command fails due to missing tool directive, treat as blocking regression and fix in the same PR.
3. No staggered rollout is required.

## Risks and Mitigations
- Risk: accidental command behavior drift.
  - Mitigation: preserve arguments exactly and keep existing check targets unchanged.
- Risk: hidden tools still executed via `go run ...@version`.
  - Mitigation: add grep-based acceptance check in review checklist.

## Definition of Done
1. All targeted Go tools are pinned via `tool` directives.
2. `Makefile` and `go:generate` use `go tool` for migrated tools.
3. Existing quality gates pass without functional regressions.
4. Documentation reflects the new tooling baseline.
