**Implemented Test Scope**
- No repository files were edited in this task. The exact test split I would implement is: unit tests for idempotency and operation-state invariants, HTTP runtime-contract tests for boundary/status/header/error behavior, and one DB-backed integration slice for durable deduplication and operation polling.

**Scenario Coverage**
- `TestStartExportRequiresIdempotencyKey`: creation is rejected before any operation record or export enqueue happens.
- `TestStartExportSameKeySamePayloadReturnsExistingOperation`: same tenant, same `Idempotency-Key`, same payload returns the same operation id and suppresses duplicate side effects.
- `TestStartExportSameKeyDifferentPayloadConflicts`: same tenant, same `Idempotency-Key`, different payload returns conflict and leaves the first operation unchanged.
- `TestOperationTransitionPendingRunningSucceeded`: legal success path is `pending -> running -> succeeded`.
- `TestOperationTransitionPendingRunningFailed`: legal failure path is `pending -> running -> failed`.
- `TestOperationTransitionRejectsSkippingRunning`: direct `pending -> succeeded` and `pending -> failed` are rejected.
- `TestOpenAPIRuntimeContractPayoutExportAccepted`: `POST /api/v1/payout-exports` with `Idempotency-Key` returns `202 Accepted`, echoes `X-Request-ID`, and sets `Location` to `/api/v1/payout-exports/operations/{operation_id}`.
- `TestOpenAPIRuntimeContractPayoutExportMissingIdempotencyKey`: returns `428 Precondition Required` with `application/problem+json` and matching `request_id`.
- `TestOpenAPIRuntimeContractPayoutExportSameKeySamePayloadReplay`: second identical POST returns an equivalent response with the same operation reference.
- `TestOpenAPIRuntimeContractPayoutExportSameKeyDifferentPayloadConflict`: returns `409 Conflict` with `application/problem+json` and matching `request_id`.
- `TestOpenAPIRuntimeContractPayoutExportQuotaExceeded`: returns `429 Too Many Requests`, includes `Retry-After`, and does not create an operation.
- `TestOpenAPIRuntimeContractPayoutExportRejectsUnknownFields`: well-formed JSON plus an unknown field returns `400 Bad Request`.
- `TestOpenAPIRuntimeContractPayoutExportRejectsTrailingJSON`: valid document followed by extra JSON tokens returns `400 Bad Request`.
- `TestOpenAPIRuntimeContractPayoutExportRejectsOversizedBody`: body over `RouterConfig.MaxBodyBytes` returns `413` with the body-limit problem response, not a generic validation error.
- `TestPayoutExportIntegration_IdempotentReplayPersistsSingleOperation`: durable store contains one operation/idempotency record and one queued export after create + replay.
- `TestPayoutExportIntegration_OperationLifecycleSucceeded`: create, worker advance, and GET on the operation resource show `pending`, then `running`, then `succeeded`.
- `TestPayoutExportIntegration_OperationLifecycleFailed`: create, worker failure, and GET on the operation resource show `pending`, then `running`, then `failed`.

**Key Test Files**
- `internal/app/payout_export/service_test.go`: service-level idempotency and operation-transition invariants.
- `internal/infra/http/payout_export_contract_test.go`: `TestOpenAPIRuntimeContract*` HTTP boundary checks so `make openapi-check` exercises them.
- `test/payout_export_integration_test.go`: `//go:build integration` durable idempotency and operation-resource lifecycle coverage.

**Validation Commands**
- `go test ./internal/app/payout_export ./internal/infra/http -count=1`
- `go test -race ./internal/app/payout_export ./internal/infra/http -count=1`
- `make openapi-check`
- `REQUIRE_DOCKER=1 make test-integration`

**Observed Result**
- No repository files were changed.
- No validation commands were run in this no-edit task, so there is no fresh execution evidence yet.

**Design Escalations**
- The repository currently serves versioned API routes under `/api/v1/*`, while the task statement uses `/v1/*`. The HTTP tests above assume `/api/v1/payout-exports` unless the prefix is intentionally changing.
- “Equivalent” replay behavior for same-key/same-payload is underspecified for terminal operations. The tests above assume replay returns the same logical operation reference and suppresses duplicate side effects; if terminal replay should switch to `200 OK`, that needs to be made explicit.
- “Visible diagnostics” for correlation is underspecified. The tests above assume the required surface is echoed `X-Request-ID` plus `request_id` in problem responses; if the operation resource itself must expose correlation identifiers, add one more operation-resource assertion.

**Residual Risks**
- The exact file/package names assume the feature lands as `internal/app/payout_export` with a dedicated HTTP contract test file.
- If the async export flow uses background goroutines or an external queue worker, one additional leak/race-oriented test may be needed once the concrete orchestration shape exists.
