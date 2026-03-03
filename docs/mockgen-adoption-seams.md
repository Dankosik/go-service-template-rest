# Mockgen Adoption Seams (Spec 02 / S02-T02)

## Objective
Define the initial consumer-side seams for incremental `mockgen` adoption and mark the first migration wave.

## Seam Inventory
- `internal/app/health.Probe`
  - Consumer: `health.Service`
  - Current manual doubles: `probeStub` in `internal/app/health/service_test.go`, `failingProbe` in `internal/infra/http/openapi_contract_test.go`
  - Decision: selected for first adoption wave.
- `internal/infra/http` strict-handler dependencies (`Ping`, `Ready`, metrics handler access)
  - Consumer: strict OpenAPI handler adapter
  - Current status: concrete dependencies are used directly in integration-oriented router tests
  - Decision: defer to a later wave to avoid broad test-surface migration in this increment.

## Interface Slicing Decisions
- Keep the first seam behavior-focused and narrow:
  - `Name() string`
  - `Check(context.Context) error`
- Own this seam in the consumer package (`internal/app/health`) instead of expanding provider-side interfaces.
- Preserve runtime behavior; this step only prepares test seam ownership for generated mocks.

## First Adoption Set
- Wave 1 target: `internal/app/health.Probe`
- Next task dependency: `S02-T03` adds `//go:generate` for this seam and generates the first mock set.
