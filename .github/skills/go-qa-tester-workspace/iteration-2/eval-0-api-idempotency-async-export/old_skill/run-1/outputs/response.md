**Implemented Test Scope**
- No repository files were edited in this task.
- I would split the proof into three layers: unit tests in `internal/app/payoutexport/service_test.go`, HTTP boundary tests in `internal/infra/http/payout_export_contract_test.go` plus `internal/infra/http/openapi_contract_test.go`, and durable `//go:build integration` tests in `test/payout_export_integration_test.go`.
- Unit tests prove idempotency, quota, enqueue-once behavior, and legal operation-state transitions without transport noise.
- HTTP tests prove `POST /v1/payout-exports` status/header/body/error semantics, strict JSON behavior, body-size split, and correlation diagnostics.
- Integration tests prove replay/conflict behavior and operation polling survive fresh app wiring instead of only working in-process.

**Scenario Coverage**
- `[unit] TestStartExportCreatesPendingOperationForNewKey`: new tenant + new `Idempotency-Key` creates one `pending` operation and enqueues exactly one export job.
- `[unit] TestStartExportReusesOperationForSameKeySamePayload`: same tenant, same key, same payload returns the same operation id and does not enqueue a second job.
- `[unit] TestStartExportRejectsSameKeyDifferentPayload`: same tenant, same key, different payload returns conflict and leaves the first operation unchanged.
- `[unit] TestStartExportRejectsQuotaExceededWithRetryAfter`: quota exhaustion returns a throttle result with retry guidance and creates no operation.
- `[unit] TestOperationTransitionsPendingRunningSucceeded`: legal success path is `pending -> running -> succeeded`.
- `[unit] TestOperationTransitionsPendingRunningFailed`: legal failure path is `pending -> running -> failed`.
- `[http] TestPayoutExportAcceptedReturns202LocationAndRequestID`: valid `POST /v1/payout-exports` returns `202 Accepted`, `Location: /v1/payout-exports/operations/{operation_id}`, and echoes inbound `X-Request-ID`.
- `[http] TestPayoutExportReplayReturnsEquivalentOperationResponse`: second identical request returns the same logical operation reference and same `Location`.
- `[http] TestPayoutExportMissingIdempotencyKeyReturnsPreconditionFailure`: missing required key is rejected with `application/problem+json` and matching `request_id`.
- `[http] TestPayoutExportDifferentPayloadSameKeyReturnsConflict`: same key plus changed payload returns `409 Conflict` and preserves correlation diagnostics.
- `[http] TestPayoutExportQuotaExceededReturns429WithRetryAfter`: throttled request maps to `429 Too Many Requests` and includes `Retry-After`.
- `[http] TestPayoutExportRejectsUnknownFields`: strict JSON decoding rejects unknown fields.
- `[http] TestPayoutExportRejectsTrailingJSONGarbage`: valid JSON followed by extra tokens is rejected.
- `[http] TestPayoutExportOversizedBodyReturns413Not400`: oversized body returns `413 Request Entity Too Large`, not a generic decode/validation `400`.
- `[http] TestPayoutExportOperationResourceExposesLifecycleStates`: `GET /v1/payout-exports/operations/{operation_id}` exposes `pending`, `running`, `succeeded`, and `failed`.
- `[integration] TestPayoutExportIdempotencyPersistsAcrossFreshAppWiring`: create + replay through fresh app wiring still returns the same operation id.
- `[integration] TestPayoutExportConflictPersistsAcrossFreshAppWiring`: same key plus different payload through fresh app wiring still returns durable conflict.
- `[integration] TestPayoutExportOperationLifecyclePersistsSucceeded`: persisted operation moves through `pending -> running -> succeeded` when the worker completes.
- `[integration] TestPayoutExportOperationLifecyclePersistsFailed`: persisted operation moves through `pending -> running -> failed` when the worker fails.

**Key Test Files**
- `internal/app/payoutexport/service_test.go`
- `internal/infra/http/payout_export_contract_test.go`
- `internal/infra/http/openapi_contract_test.go`
- `test/payout_export_integration_test.go`

**Validation Commands**
- `go test ./internal/app/payoutexport -run 'Test(StartExport|OperationTransitions)' -count=1`
- `go test ./internal/infra/http -run 'TestPayoutExport|TestOpenAPIRuntimeContractPayoutExport' -count=1`
- `go test -race ./internal/app/payoutexport ./internal/infra/http -count=1`
- `go test -tags=integration ./test -run 'TestPayoutExport' -count=1`
- `make openapi-check`
- `go test ./... -count=1`

**Observed Result**
- Executed on the unmodified tree: `go test ./... -count=1` passed.
- Executed on the unmodified tree: `go test -race ./internal/infra/http -count=1` passed.
- No payout-export-specific tests were run because this was a no-edit task and those tests do not exist yet.

**Design Escalations**
- The prompt requires rejection for a missing `Idempotency-Key` but does not pin the exact status code; if this feature follows the repo's existing idempotency pattern, lock that test to `428 Precondition Required` before implementation.
- "Equivalent" replay behavior is underspecified for the second same-key same-payload `POST`; the tests above lock same operation identity and same `Location`, but final replay status/body semantics should be made explicit.
- `Retry-After` is required on `429`, but the contract does not say `delta-seconds` vs `HTTP-date`; that should be fixed before writing an exact header assertion.
- The repository's current OpenAPI/runtime paths use `/api/v1/*`, while the task statement uses `/v1/*`; the route prefix needs to be decided before writing the HTTP tests.

**Residual Risks**
- These tests would prove durable replay/conflict behavior, but a simultaneous multi-instance first-write race may still need a transaction/uniqueness-focused integration test once the storage model exists.
- If the async export path introduces goroutines or queue workers, one additional leak/cancellation test will likely be needed after the concrete worker orchestration is implemented.
