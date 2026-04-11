# Test-Level Selection

## Behavior Change Thesis
When loaded for symptom "the strategy needs a proof level choice", this file makes the model choose the smallest boundary that proves the risk instead of likely mistake "add integration or e2e coverage to look safer."

## When To Load
Load this when the task asks which level should prove a behavior, or when fuzz, race, contract, integration, or e2e evidence is being considered.

## Decision Rubric
- Choose unit when the proof target is deterministic local logic with no transport, DB, cache, clock, process, or goroutine boundary.
- Choose contract when the proof target is status, headers, request decoding, response shape, OpenAPI drift, idempotency semantics, or any client-visible boundary behavior.
- Choose integration when the proof target is a real source-of-truth boundary: SQL transaction behavior, migration compatibility, cache serialization/TTL/fallback, durable idempotency, or runtime dependency behavior.
- Choose e2e smoke only for minimal composed-runtime confidence after smaller tests own the exact correctness claims.
- Choose fuzz smoke only for deterministic input-heavy logic with cheap invariants and a seed/regression story.
- Choose race execution only when the risky concurrency path is exercised by a targeted scenario; `-race` is proof instrumentation, not a scenario by itself.
- When the choice is nontrivial, name one rejected level and the evidence gap it would leave.

## Imitate
| Risk | Select | Reject | Copy This |
| --- | --- | --- | --- |
| HTTP validation error shape changed | Contract | Unit-only | The selected level includes decoding, status, problem body, headers, and no partial side effect. |
| Transaction rollback on mid-step failure | Integration | Mock-only unit | The proof exercises durable state and absence of partial writes, not just branch flow. |
| Worker shutdown with shared state | Targeted component scenario under `-race` | Broad e2e smoke | The race run is paired with deterministic cancellation/drain rows that execute the lifecycle path. |
| Critical create flow wiring | E2E smoke plus lower-level contract/integration obligations | E2E as primary proof | Smoke proves wiring; smaller tests still prove validation, idempotency, and persistence semantics. |

## Reject
- "Use e2e for full confidence." This hides which invariant failed and usually skips the lower-level proof that would catch the actual regression.
- "Run `go test -race ./...` for concurrency coverage." Race detection only helps if the planned scenario executes the shared-state path.
- "Add fuzzing for robustness" on logic without a deterministic invariant or seed corpus expectation. That is test theater, not a strategy.
- "Unit test the API handler and call it contract coverage" when the public proof requires generated bindings, middleware, request decoding, or documented response semantics.

## Agent Traps
- Do not escalate level because a requirement is important; escalate only because the lower level cannot observe the needed boundary.
- Do not call e2e smoke a substitute for API, data, reliability, or security proof.
- Do not treat mocks as proof of DB, cache, transaction, migration, or runtime contract behavior.
- Do not name a command before selecting the proof level and risk surface it proves.

## Validation Shape
The strategy should state: risk -> selected level -> rejected level -> observable -> repository-supported command family. If any part is missing, the level choice is not yet implementation-ready.
