# 65 Coder Detailed Plan: Spec 02 (`mockgen`)

## Execution Context
Scope boundaries:
- Introduce deterministic mock generation baseline using `go.uber.org/mock/mockgen`.
- Migrate selected test seams from handwritten fakes to generated mocks.
- Add generation and drift controls.

Non-goals:
- Rewriting all existing tests.
- Replacing integration tests with unit mocks.
- Introducing DI framework/container behavior.

Critical invariants:
- INV-S02-1: Generated mocks are consumer-side and interface-narrow.
- INV-S02-2: Generated mocks are test-oriented by default (`*_mock_test.go`).
- INV-S02-3: Mock generation is reproducible and drift-detectable.

Forbidden changes:
- Broad provider interfaces generated as-is without seam reduction.
- Over-mocking that hides integration correctness.

## Execution Mode
- Mode: `in-session`
- Checkpoint policy: checkpoint every `2` tasks.
- Coder autonomy: internal test decomposition and expectation style are coder-defined within behavior-first constraints.

## Task Graph
- S02-T01 -> S02-T02 -> S02-T03 -> S02-T04 -> S02-T05 -> S02-T06
- S02-T03 depends on S02-T02.
- S02-T04 depends on S02-T03.
- S02-T05 depends on S02-T04.
- S02-T06 depends on S02-T05.

## Task Cards

### Task ID
S02-T01

Objective:
- Pin `mockgen` in tool directives and establish callable generation baseline.

Spec Traceability:
- Decisions: S02-D1, S02-D3
- Invariants: INV-S02-3
- Test obligations: S02-TST-BOOTSTRAP

Change Surface:
- Tooling and build command layer.

Task Sequence:
1. Add `mockgen` tool directive.
2. Ensure command execution path via `go tool mockgen` is available.
3. Add/prepare dedicated generation target (`make mocks-generate` or consolidated generate mode).

Verification Commands:
- `go tool mockgen -help`

Expected Evidence:
- `mockgen` resolves via `go tool`.
- Generation target exists and is callable.

Review Checklist:
- Version pinning explicit.
- No runtime impact introduced.
- Command naming consistent.
- Scope boundaries preserved.

Ambiguity Triggers:
- If tool directive conflicts with existing toolchain constraints.

Change Reconciliation:
- Expected: tooling/build layer only.

Progress Status:
- `todo`

### Task ID
S02-T02

Objective:
- Identify initial consumer-side seams and define interface slicing plan.

Spec Traceability:
- Decisions: S02-D2
- Invariants: INV-S02-1
- Test obligations: S02-TST-SEAMS

Change Surface:
- App/test boundary design (interface declarations and test seam ownership).

Task Sequence:
1. Inventory current/pending seams where mocks provide immediate value.
2. Slice interfaces to narrow behavior-focused units where needed.
3. Mark first adoption set for generated mocks.

Verification Commands:
- `rg "type .* interface" internal`

Expected Evidence:
- Explicit first adoption seam list.
- No broad interface forced into mock generation without justification.

Review Checklist:
- Consumer-side ownership respected.
- Interface size controlled.
- Adoption set is incremental.
- Integration-test coverage plan preserved.

Ambiguity Triggers:
- If seam ownership is unclear between app and infra layers.

Change Reconciliation:
- Expected: interface/test seam layer.

Progress Status:
- `todo`

### Task ID
S02-T03

Objective:
- Add `//go:generate` directives and generate first mock set.

Spec Traceability:
- Decisions: S02-D3, S02-D4
- Invariants: INV-S02-2, INV-S02-3
- Test obligations: S02-TST-GEN

Change Surface:
- Interface source files + generated test artifacts.

Task Sequence:
1. Add generation directives near target interfaces.
2. Generate mocks using source mode.
3. Ensure naming/path/package conventions are respected.

Verification Commands:
- `make mocks-generate`

Expected Evidence:
- Generated files present with expected naming convention.
- Generation rerun is stable (no unexpected diff after fresh run).

Review Checklist:
- Directive placement is local and clear.
- Generated files are test-oriented by default.
- No hand-edited generated file sections.
- Conventions matched.

Ambiguity Triggers:
- If cross-package test need requires non-test mock destination.

Change Reconciliation:
- Expected: test/mocks generation surfaces.

Progress Status:
- `todo`

### Task ID
S02-T04

Objective:
- Migrate selected tests from manual fakes to generated mocks while preserving behavior assertions.

Spec Traceability:
- Decisions: S02-D1, S02-D2
- Invariants: INV-S02-1
- Test obligations: S02-TST-BEHAVIOR

Change Surface:
- Unit tests around selected seams.

Task Sequence:
1. Replace manual fake usage in selected tests.
2. Keep tests behavior-focused (avoid brittle over-specification).
3. Remove replaced manual fake code where safe.

Verification Commands:
- `make test`

Expected Evidence:
- Selected tests pass with generated mocks.
- Manual fake boilerplate reduced in covered seams.

Review Checklist:
- Assertions are outcome-focused.
- No unnecessary call-order coupling.
- Test readability remains acceptable.
- No regressions in covered behavior.

Ambiguity Triggers:
- If generated mocks make a test harder to understand than minimal handwritten fake.

Change Reconciliation:
- Expected: targeted unit-test files only.

Progress Status:
- `todo`

### Task ID
S02-T05

Objective:
- Add drift detection and review guardrails for generated mocks.

Spec Traceability:
- Decisions: S02-D5
- Invariants: INV-S02-3
- Test obligations: S02-TST-DRIFT

Change Surface:
- Build/CI check layer and contributor guidance docs.

Task Sequence:
1. Add mock generation drift check path.
2. Integrate check into existing quality flow where appropriate.
3. Add concise contributor rule: interface changes must regenerate mocks.

Verification Commands:
- `make mocks-generate`
- `git diff --exit-code`

Expected Evidence:
- Drift is detectable automatically.
- Guidance is documented.

Review Checklist:
- Check is deterministic.
- No false positives on clean rerun.
- Rule is visible to contributors.
- Scope remains incremental.

Ambiguity Triggers:
- If CI sequence placement conflicts with existing check order/performance budget.

Change Reconciliation:
- Expected: CI/check docs surfaces.

Progress Status:
- `todo`

### Task ID
S02-T06

Objective:
- Execute mandatory validation suite and finalize evidence.

Spec Traceability:
- Decisions: S02-D1..S02-D5
- Invariants: INV-S02-1..INV-S02-3
- Test obligations: S02-TST-FULL

Change Surface:
- Verification layer.

Task Sequence:
1. Run required test and lint commands.
2. Confirm race safety on updated tests.
3. Package evidence for handoff.

Verification Commands:
- `make mocks-generate`
- `make test`
- `make test-race`
- `make lint`

Expected Evidence:
- All required checks pass.
- No stale mock artifacts remain.

Review Checklist:
- Evidence complete.
- No unresolved blocker.
- No hidden manual fake reintroduction.
- Ready for completion.

Ambiguity Triggers:
- If race/lint failures are unrelated pre-existing issues.

Change Reconciliation:
- Verification-only stage.

Progress Status:
- `todo`

## Checkpoint Plan
- CP-S02-1 (after S02-T02):
  - Confirm seam selection and interface slicing quality.
  - Go/no-go: proceed only with explicit adoption seam set.
- CP-S02-2 (after S02-T04):
  - Confirm generation + migration of first test seams.
  - Go/no-go: proceed only if tests pass and behavior assertions remain robust.
- CP-S02-3 (after S02-T06):
  - Confirm full quality evidence and drift guard operation.

## Clarification Contract
Required fields:
- `request_id`
- `blocked_task_id`
- `ambiguity_type` (`contract`, `invariant`, `security`, `reliability`, `test`, `other`)
- `conflicting_sources`
- `decision_impact`
- `proposed_options`
- `owner`
- `resume_condition`

Resolution policy:
- Keep task blocked until resolution is documented and mapped back to traceability IDs.

## Coverage Matrix
- S02-OBL-1 (single approved mock generator) -> S02-T01, S02-T03
- S02-OBL-2 (consumer-side narrow interfaces) -> S02-T02, S02-T04
- S02-OBL-3 (test-focused generated artifacts) -> S02-T03
- S02-OBL-4 (drift visibility in CI) -> S02-T05, S02-T06

## Execution Notes
- Prefer applying migration on touched seams first; avoid broad mechanical rewrites in one run.
