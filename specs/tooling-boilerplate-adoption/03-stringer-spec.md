# Spec 03: Enum String Boilerplate Reduction via `stringer`

## Problem
Enum-like integer types eventually require `String()` methods for logs, metrics labels, and diagnostics.
Handwritten switch-based string methods are repetitive and drift-prone.

## Goals
1. Standardize enum string generation via `stringer`.
2. Eliminate handwritten `String()` boilerplate for eligible enums.
3. Keep external wire contracts stable by separating internal enum formatting from API representation.

## Non-Goals
- No conversion of all constants to enums.
- No replacement of explicit string constants used in API/storage contracts.
- No automatic parser generation in this phase.

## Decisions (Normative)
1. `stringer` is approved for internal enum types with underlying integer kinds.
2. For externally stable values (API payloads, persisted public text values), use explicit string constants/types, not `stringer` output.
3. `String()` methods for eligible enums must be generated, not handwritten.
4. Generated files (`*_string.go`) are versioned in git and protected by drift checks.
5. `stringer` is pinned via `go.mod` tool directives and executed through `go tool stringer`.

## Eligibility Rules
Use `stringer` when all conditions are true:
1. Type is internal-facing.
2. Type is integer-based enum.
3. Human-readable value is needed for logs/errors/debug.

Do not use `stringer` when any condition is true:
1. Text value is part of a stable external contract.
2. Backward-compatible textual value must stay fixed independently of Go constant names.

## Generation Conventions
Directive template:
```go
//go:generate go tool stringer -type=<TypeName>
```

Placement rules:
1. Directive is placed in the same file/package as enum type definitions.
2. One generated file per type group where practical.

## Implementation Plan

### WP-1: Tool Bootstrap
- Add `stringer` to `go.mod` tool directives.
- Add generation target to the project command set (or include in consolidated generate flow).

### WP-2: Initial Enum Migration
- Identify existing handwritten enum `String()` methods.
- Replace with generated output for eligible internal enums.

### WP-3: Contract Safety Guard
- Review each enum candidate against external contract usage.
- Keep explicit string contract types where stability is mandatory.

### WP-4: CI Drift Guard
- Ensure generated enum files are covered by generation + git-diff checks.

## Validation
Mandatory evidence after implementation:
1. Enum generation command succeeds.
2. `make test`
3. `make lint`
4. `make fmt-check`

Acceptance checks:
- No handwritten `String()` switch remains for eligible enums.
- External contract text values remain unchanged.

## Rollout Strategy
1. Incremental migration on touch of enum-heavy code areas.
2. Full migration is optional; only eligible enums are targeted.

## Risks and Mitigations
- Risk: accidental external text change due to renamed constants.
  - Mitigation: do not use `stringer` for external contract text.
- Risk: generated file drift.
  - Mitigation: enforce generate-and-diff checks in CI.

## Definition of Done
1. `stringer` is pinned and callable through `go tool`.
2. Eligibility rules are documented and followed.
3. Generated enum files are drift-protected.
4. Targeted internal enums no longer use handwritten `String()` boilerplate.
