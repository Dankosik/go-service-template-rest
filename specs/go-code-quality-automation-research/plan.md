# Implementation Plan

## Strategy

Implement in one coherent tooling slice because the affected surfaces are small and coupled through local/CI command semantics.

## Ordered Phases

1. Workflow command plumbing:
   - Fix Docker guardrails zero-setup behavior.
   - Add `docker-check`.
   - Improve `check-full` fallback wording.
   - Remove race duplication from `test-report`.

2. Lint and Go cleanup:
   - Update `.golangci.yml`.
   - Fix `noctx` and `errchkjson` baseline findings.

3. Test gates:
   - Add nightly `-shuffle=on`.
   - Add one stable fuzz target.
   - Add bootstrap `goleak` coverage if clean.

4. Documentation:
   - Update PR template, contributor docs, and command reference.

5. Validation:
   - Run targeted checks, then `make check`.

## Implementation Readiness

Status: PASS.

Proof obligations:
- Lint config verifies and `make lint` passes.
- Fuzz smoke finds and runs at least one fuzz target.
- Targeted changed packages pass.
- `make check` passes.
- Docker guardrails no longer depends on host `go` for the Docker path, proven by a targeted command when Docker is reachable.
