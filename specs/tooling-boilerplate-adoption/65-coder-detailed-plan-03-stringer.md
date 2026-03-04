# 65 Coder Detailed Plan: Spec 03 (`stringer`)

## Execution Context
Scope boundaries:
- Establish `stringer`-based generation for eligible internal enums.
- Remove eligible handwritten `String()` implementations.
- Preserve external wire/storage text contract stability.

Non-goals:
- Converting all constants to enums.
- Changing externally visible text values used in API/storage contracts.
- Adding parser/codegen beyond `String()` generation.

Critical invariants:
- INV-S03-1: `stringer` applies only to eligible internal enums.
- INV-S03-2: External stable text contracts remain explicit and unchanged.
- INV-S03-3: Generated enum files are reproducible and drift-protected.

Forbidden changes:
- Using generated enum strings as external protocol truth by default.
- Silent behavior changes in logs/metrics labels without explicit review.

## Execution Mode
- Mode: `in-session`
- Checkpoint policy: checkpoint every `2` tasks.
- Coder autonomy: enum grouping, file split, and local refactoring remain coder-defined within contract-safety rules.

## Task Graph
- S03-T01 -> S03-T02 -> S03-T03 -> S03-T04 -> S03-T05 -> S03-T06
- S03-T03 depends on S03-T02.
- S03-T04 depends on S03-T03.
- S03-T05 depends on S03-T04.
- S03-T06 depends on S03-T05.

## Task Cards

### Task ID
S03-T01

Objective:
- Pin `stringer` as a tool directive and add callable generation command path.

Spec Traceability:
- Decisions: S03-D1, S03-D5
- Invariants: INV-S03-3
- Test obligations: S03-TST-BOOTSTRAP

Change Surface:
- Tooling/build generation layer.

Task Sequence:
1. Add `stringer` to tool directives.
2. Add generation target or include in consolidated generate flow.
3. Confirm command availability.

Verification Commands:
- `go tool stringer -help`

Expected Evidence:
- `stringer` resolves through `go tool`.
- Generation command baseline is in place.

Review Checklist:
- Tool pinning explicit.
- No runtime behavior impact.
- Command path documented.
- Scope boundaries intact.

Ambiguity Triggers:
- If `go tool stringer` resolution conflicts with current toolchain state.

Change Reconciliation:
- Expected: tooling/build layer.

Progress Status:
- `todo`

### Task ID
S03-T02

Objective:
- Inventory enum candidates and classify each as eligible/internal vs external-contract-bound.

Spec Traceability:
- Decisions: S03-D1, S03-D2
- Invariants: INV-S03-1, INV-S03-2
- Test obligations: S03-TST-CLASSIFICATION

Change Surface:
- Domain/type-definition layer.

Task Sequence:
1. Find enum-like integer types.
2. For each candidate, classify contract exposure.
3. Define migration set for this cycle.

Verification Commands:
- `rg "type .* (int|int32|int64|uint|uint32|uint64)" internal`

Expected Evidence:
- Candidate list with classification rationale.
- External-contract-bound enums excluded from direct `stringer` usage.

Review Checklist:
- Classification rationale explicit.
- No contract-risk candidate in migration set.
- Incremental scope respected.
- Recheckable inventory.

Ambiguity Triggers:
- If contract exposure of enum value is unclear.

Change Reconciliation:
- Expected: enum/type definition analysis surface.

Progress Status:
- `todo`

### Task ID
S03-T03

Objective:
- Add `//go:generate` directives for eligible enums and generate `_string.go` files.

Spec Traceability:
- Decisions: S03-D3, S03-D5
- Invariants: INV-S03-1, INV-S03-3
- Test obligations: S03-TST-GEN

Change Surface:
- Eligible enum source files + generated enum artifacts.

Task Sequence:
1. Add directives near enum definitions.
2. Run generation command.
3. Confirm generated files are stable on rerun.

Verification Commands:
- `make generate` (or dedicated enum generation target)
- `git diff --exit-code`

Expected Evidence:
- Generated files created for eligible enums.
- Rerun produces no new diff.

Review Checklist:
- Directive placement local and clear.
- Generated code not manually edited.
- Naming and package placement consistent.
- Stability confirmed.

Ambiguity Triggers:
- If enum group layout causes conflicting generated symbol names.

Change Reconciliation:
- Expected: enum code + generated artifacts.

Progress Status:
- `todo`

### Task ID
S03-T04

Objective:
- Replace eligible handwritten `String()` methods with generated implementations.

Spec Traceability:
- Decisions: S03-D3
- Invariants: INV-S03-1
- Test obligations: S03-TST-REPLACEMENT

Change Surface:
- Internal enum helper methods and related call sites.

Task Sequence:
1. Remove eligible handwritten `String()` methods.
2. Ensure call sites compile against generated methods.
3. Keep behavior-compatible output where required internally.

Verification Commands:
- `make test`

Expected Evidence:
- No eligible handwritten `String()` switch remains.
- Tests compile and pass for impacted areas.

Review Checklist:
- Removed only eligible methods.
- No accidental external contract shift.
- Readability preserved.
- Internal behavior remains expected.

Ambiguity Triggers:
- If existing test expectations depend on old handwritten fallback string behavior.

Change Reconciliation:
- Expected: local enum implementation layer.

Progress Status:
- `todo`

### Task ID
S03-T05

Objective:
- Add drift protection and guidance for enum generation workflow.

Spec Traceability:
- Decisions: S03-D4
- Invariants: INV-S03-3
- Test obligations: S03-TST-DRIFT

Change Surface:
- CI/build checks and contributor documentation.

Task Sequence:
1. Ensure enum generation participates in drift checks.
2. Add concise contributor rule for enum changes.
3. Confirm guard behavior on stale generated artifacts.

Verification Commands:
- `make lint`
- `make fmt-check`

Expected Evidence:
- Guardrails documented and enforceable.
- Static checks remain green.

Review Checklist:
- Drift rule deterministic.
- Docs aligned with actual command path.
- No overreach into external contracts.
- Conventions explicit.

Ambiguity Triggers:
- If drift-check placement conflicts with existing generation orchestration.

Change Reconciliation:
- Expected: check/docs layer.

Progress Status:
- `todo`

### Task ID
S03-T06

Objective:
- Execute final validation suite and finalize evidence package.

Spec Traceability:
- Decisions: S03-D1..S03-D5
- Invariants: INV-S03-1..INV-S03-3
- Test obligations: S03-TST-FULL

Change Surface:
- Verification layer.

Task Sequence:
1. Run mandatory validation commands.
2. Confirm no contract regression.
3. Collect evidence for completion review.

Verification Commands:
- enum generation command
- `make test`
- `make lint`
- `make fmt-check`

Expected Evidence:
- Required checks pass.
- Eligibility and contract guard obligations satisfied.

Review Checklist:
- Evidence complete.
- No unresolved ambiguities.
- No regression in affected paths.
- Completion-ready state.

Ambiguity Triggers:
- If failures are unrelated pre-existing repository issues.

Change Reconciliation:
- Verification stage only.

Progress Status:
- `todo`

## Checkpoint Plan
- CP-S03-1 (after S03-T02):
  - Confirm candidate classification and contract-safety boundary.
  - Go/no-go: proceed only with approved eligible enum set.
- CP-S03-2 (after S03-T04):
  - Confirm generation and replacement correctness.
  - Go/no-go: proceed only if tests pass and no external contract drift is detected.
- CP-S03-3 (after S03-T06):
  - Confirm complete evidence and stable drift guard behavior.

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
- Contract-related ambiguities block progression until explicit owner decision is recorded.

## Coverage Matrix
- S03-OBL-1 (tool bootstrap and execution baseline) -> S03-T01
- S03-OBL-2 (eligibility and contract-safe classification) -> S03-T02
- S03-OBL-3 (generated `String()` adoption) -> S03-T03, S03-T04
- S03-OBL-4 (drift protection and guidance) -> S03-T05, S03-T06

## Execution Notes
- Prefer conservative classification when uncertain: treat unknown enum exposure as external-contract-bound until clarified.
