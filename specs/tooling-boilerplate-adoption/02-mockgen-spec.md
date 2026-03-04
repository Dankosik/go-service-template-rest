# Spec 02: Mock Generation Standard via `mockgen`

## Problem
As the service grows, interface-based tests will need more doubles.
Manual fake implementations create repetitive code and high maintenance cost after interface changes.

## Goals
1. Standardize mock generation with one supported tool: `go.uber.org/mock/mockgen`.
2. Minimize handwritten mocks.
3. Keep test doubles local to consumer boundaries.
4. Ensure reproducible generation and CI drift protection.

## Non-Goals
- No replacement of integration tests with unit mocks.
- No generation for every interface by default.
- No runtime DI container adoption in this spec.

## Decisions (Normative)
1. `mockgen` is the only approved mock generator in this repository.
2. Generate mocks only for consumer-side narrow interfaces (not broad provider interfaces).
3. Default generation mode: source mode (`-source=...`) for deterministic behavior.
4. Generated files must be test-only (`*_mock_test.go`) unless there is an explicit cross-package test need.
5. Generated mocks are versioned in git; CI must detect drift.

## Interface Policy
1. Define small interfaces in the package that consumes a dependency.
2. One interface should represent one behavioral seam.
3. If an interface grows and mock churn increases, split the interface before generating new mocks.

## Generation Conventions
File naming:
- `zz_<interface>_mock_test.go`

Directive location:
- Keep `//go:generate` next to the interface definition file.

Command template:
```go
//go:generate go tool mockgen -source=<file>.go -destination=zz_<file>_mock_test.go -package=<pkg>
```

## Implementation Plan

### WP-1: Tool Bootstrap
- Add `mockgen` to `go.mod` tool directives.
- Add `make mocks-generate` target (or equivalent consolidated generate target).

### WP-2: First Adoptions
- Start with app/service seams where handwritten fakes are already present or expected soon.
- Replace handwritten mocks in those tests with generated mocks.

### WP-3: CI Drift Guard
- Add drift check after generation (clean git tree expectation for generated mocks).
- Add review checklist item: "interface changed -> regenerate mocks".

### WP-4: Test Style Alignment
- Keep behavior-focused tests; do not overfit tests to call-order details unless required by contract.
- Prefer simple expectation setup and explicit assertions.

## Validation
Mandatory evidence after implementation:
1. `make mocks-generate`
2. `make test`
3. `make test-race`
4. `make lint`

Acceptance check:
- No new manual fake structs for covered interfaces in changed areas.

## Rollout Strategy
1. Incremental adoption is required.
2. Do not refactor all legacy tests in one PR.
3. Apply on touch: when changing a test boundary, migrate that boundary to generated mocks.

## Risks and Mitigations
- Risk: over-mocking and fragile tests.
  - Mitigation: keep integration tests as primary confidence for infra boundaries.
- Risk: interface bloat increases generated noise.
  - Mitigation: enforce narrow consumer-side interfaces.

## Definition of Done
1. `mockgen` is pinned and callable through `go tool`.
2. Mock generation conventions are documented and applied.
3. CI can detect stale generated mocks.
4. New/updated unit tests at selected seams use generated mocks, not handwritten doubles.
