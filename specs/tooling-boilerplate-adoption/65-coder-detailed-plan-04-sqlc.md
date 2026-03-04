# 65 Coder Detailed Plan: Spec 04 (`sqlc`)

## Execution Context
Scope boundaries:
- Introduce `sqlc` generation baseline for Postgres access with migration-sourced schema.
- Implement first vertical slice using generated query code behind infra adapters.
- Add deterministic generation and drift checks.

Non-goals:
- ORM migration.
- Full big-bang replacement of existing DB access paths.
- Broad architectural refactor across unrelated modules.

Critical invariants:
- INV-S04-1: SQL remains explicit source-of-truth for query semantics.
- INV-S04-2: Generated DB code stays infra-local and not leaked to app/domain contracts.
- INV-S04-3: Schema source for generation is repository migrations (`env/migrations/*.up.sql`).
- INV-S04-4: Generation is reproducible and enforced by checks.

Forbidden changes:
- Direct sqlc package imports from app/domain layers.
- Raw SQL literals introduced in app/service layer.
- Hidden transaction semantics outside infra adapters.

## Execution Mode
- Mode: `batch`
- Checkpoint policy: checkpoint every `2-3` tasks.
- Coder autonomy: local implementation decomposition of adapters/mappers/tests remains coder-defined while preserving boundary and verification obligations.

## Task Graph
- S04-T01 -> S04-T02 -> S04-T03 -> S04-T04 -> S04-T05 -> S04-T06 -> S04-T07
- S04-T02 depends on S04-T01.
- S04-T03 depends on S04-T02.
- S04-T04 depends on S04-T03.
- S04-T05 depends on S04-T04.
- S04-T06 depends on S04-T05.
- S04-T07 depends on S04-T06.

## Task Cards

### Task ID
S04-T01

Objective:
- Bootstrap `sqlc` tooling and base generation config.

Spec Traceability:
- Decisions: S04-D1, S04-D5
- Invariants: INV-S04-3, INV-S04-4
- Test obligations: S04-TST-BOOTSTRAP

Change Surface:
- Tooling/go.mod + SQL generation configuration layer.

Task Sequence:
1. Add `sqlc` as tool directive.
2. Add baseline `sqlc.yaml` with PostgreSQL + `pgx/v5` settings.
3. Add minimal generation target command path.

Verification Commands:
- `go tool sqlc version`
- `make sqlc-generate`

Expected Evidence:
- `sqlc` command resolves via `go tool`.
- Generation command executes with baseline config.

Review Checklist:
- Config uses repository migration schema path.
- Tool pinning explicit.
- No runtime behavior impact yet.
- Scope boundaries preserved.

Ambiguity Triggers:
- If sqlc config requires unresolved naming/package conventions.

Change Reconciliation:
- Expected: tooling/config surfaces.

Progress Status:
- `todo`

### Task ID
S04-T02

Objective:
- Establish query source layout and first query set for `ping_history` vertical slice.

Spec Traceability:
- Decisions: S04-D2, S04-D3
- Invariants: INV-S04-1, INV-S04-3
- Test obligations: S04-TST-QUERY-SOURCE

Change Surface:
- Infra postgres query-source area.

Task Sequence:
1. Create queries directory structure.
2. Add named SQL queries for at least one write and one read path.
3. Ensure query semantics are explicit in SQL (ordering/limits as needed).

Verification Commands:
- `make sqlc-generate`

Expected Evidence:
- Generated methods created for named queries.
- Query SQL files are explicit and readable.

Review Checklist:
- No hidden SQL helpers in app layer.
- Query naming is stable.
- SQL semantics are visible.
- First vertical slice is minimal but complete.

Ambiguity Triggers:
- If migration schema and query expectations diverge.

Change Reconciliation:
- Expected: infra query-source + generated outputs.

Progress Status:
- `todo`

### Task ID
S04-T03

Objective:
- Add infra adapter wrappers around generated querier with domain-safe boundaries.

Spec Traceability:
- Decisions: S04-D4
- Invariants: INV-S04-2
- Test obligations: S04-TST-BOUNDARY

Change Surface:
- Infra adapter package and repository boundary code.

Task Sequence:
1. Introduce/extend infra repository wrapper interfaces.
2. Delegate DB operations to generated querier.
3. Keep sqlc-generated types confined to infra boundary with mapping where needed.

Verification Commands:
- `make test`

Expected Evidence:
- App/domain layers compile without importing generated sqlc package.
- Vertical-slice path works through adapter boundary.

Review Checklist:
- Boundary ownership respected.
- No leakage of generated types upward.
- Error handling/context wrapping preserved.
- Behavior remains compatible.

Ambiguity Triggers:
- If current app contract requires shape not aligned with generated query types.

Change Reconciliation:
- Expected: infra adapter/repository area.

Progress Status:
- `todo`

### Task ID
S04-T04

Objective:
- Define explicit transaction handling for multi-step paths and document boundaries.

Spec Traceability:
- Decisions: S04-D1
- Invariants: INV-S04-1
- Test obligations: S04-TST-TX

Change Surface:
- Infra transactional workflow area.

Task Sequence:
1. Identify multi-step operations in first slice.
2. Keep transaction boundaries explicit in infra wrapper.
3. Ensure no hidden cross-layer transaction coupling.

Verification Commands:
- `make test`

Expected Evidence:
- Transaction boundaries are explicit and testable.
- No behavior drift in success/failure paths.

Review Checklist:
- Transaction ownership clear.
- Rollback paths handled.
- No app-layer transaction internals.
- Consistency assumptions explicit.

Ambiguity Triggers:
- If transaction boundary conflicts with current service invariants.

Change Reconciliation:
- Expected: infra transactional logic.

Progress Status:
- `todo`

### Task ID
S04-T05

Objective:
- Add unit tests for adapter behavior and error mapping around generated queries.

Spec Traceability:
- Decisions: S04-D4
- Invariants: INV-S04-2
- Test obligations: S04-TST-UNIT

Change Surface:
- Infra test layer (unit tests).

Task Sequence:
1. Add tests for happy/error flows in wrappers.
2. Verify mapping and wrapped error semantics.
3. Cover at least one transaction-related fail path where applicable.

Verification Commands:
- `make test`

Expected Evidence:
- Unit tests cover wrapper behavior and error mapping.
- Tests pass deterministically.

Review Checklist:
- Assertions are behavior-focused.
- Error classification clear.
- No flaky timing assumptions.
- Coverage aligned with obligations.

Ambiguity Triggers:
- If wrapper behavior contract is underspecified in existing code.

Change Reconciliation:
- Expected: infra unit tests.

Progress Status:
- `todo`

### Task ID
S04-T06

Objective:
- Add integration validation for first vertical slice using Postgres testcontainers.

Spec Traceability:
- Decisions: S04-D1..S04-D4
- Invariants: INV-S04-1..INV-S04-3
- Test obligations: S04-TST-INTEGRATION

Change Surface:
- Integration test suite and migration+query runtime interaction.

Task Sequence:
1. Add at least one read and one write integration test for sqlc path.
2. Ensure migrations + generated queries operate on consistent schema.
3. Validate behavior under real DB runtime.

Verification Commands:
- `make test-integration`

Expected Evidence:
- Integration tests pass for first sqlc vertical slice.
- Schema-query consistency confirmed in runtime.

Review Checklist:
- Deterministic setup/teardown.
- No environment-specific assumptions.
- Assertions reflect domain-relevant behavior.
- Failure messages actionable.

Ambiguity Triggers:
- If integration environment constraints block deterministic execution.

Change Reconciliation:
- Expected: integration test surface.

Progress Status:
- `todo`

### Task ID
S04-T07

Objective:
- Integrate `sqlc` checks into quality flow and finalize evidence package.

Spec Traceability:
- Decisions: S04-D5, S04-D6
- Invariants: INV-S04-4
- Test obligations: S04-TST-FULL

Change Surface:
- Build/CI check chain + docs for sqlc workflow.

Task Sequence:
1. Add `sqlc-check` command integration to quality chain.
2. Document regeneration rule for query/schema changes.
3. Run full mandatory validation suite.

Verification Commands:
- `make sqlc-generate`
- `make sqlc-check`
- `make test`
- `make test-integration`
- `make lint`

Expected Evidence:
- Drift detection works.
- Full required checks pass.
- Workflow is documented for contributors.

Review Checklist:
- CI/local parity maintained.
- No unrelated scope expansion.
- Evidence complete.
- Ready for handoff.

Ambiguity Triggers:
- If CI placement introduces unacceptable runtime/performance overhead and needs sequencing decision.

Change Reconciliation:
- Expected: CI/check/docs + verification stage.

Progress Status:
- `todo`

## Checkpoint Plan
- CP-S04-1 (after S04-T02):
  - Confirm tooling bootstrap + first query-source layout.
  - Go/no-go: proceed only if generation baseline works.
- CP-S04-2 (after S04-T04):
  - Confirm infra boundary safety + transaction explicitness.
  - Go/no-go: proceed only if boundaries are preserved and tests remain green.
- CP-S04-3 (after S04-T06):
  - Confirm unit+integration evidence for vertical slice.
  - Go/no-go: proceed only if integration checks pass.
- CP-S04-4 (after S04-T07):
  - Confirm drift guard and full validation closure.

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
- Tasks with boundary/contract ambiguity remain blocked until owner-approved resolution is recorded and coverage matrix remains closed.

## Coverage Matrix
- S04-OBL-1 (sqlc tooling baseline) -> S04-T01
- S04-OBL-2 (query source and schema discipline) -> S04-T02
- S04-OBL-3 (infra-only generated-code boundary) -> S04-T03
- S04-OBL-4 (explicit transaction behavior) -> S04-T04
- S04-OBL-5 (unit and integration evidence) -> S04-T05, S04-T06
- S04-OBL-6 (drift and CI enforcement) -> S04-T07

## Execution Notes
- Rollout should remain vertical-slice-first: one complete path to production-quality evidence before broad adoption.
