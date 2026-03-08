**Implemented Test Scope**
- No repository edits were made; this is the exact test-addition plan for `POST /v1/payout-exports`.
- Add service-level tests in [service_test.go](/mnt/c/Users/danii/IdeaProjects/go-service-template-rest/internal/app/payoutexport/service_test.go) for idempotency replay/conflict, quota rejection, enqueue-once behavior, and operation state transitions.
- Add HTTP boundary tests in [payout_export_test.go](/mnt/c/Users/danii/IdeaProjects/go-service-template-rest/internal/infra/http/payout_export_test.go) for `202 Accepted` + `Location`, strict JSON rejection, missing `Idempotency-Key`, replay conflict, `429` + `Retry-After`, `413` split, and request-id diagnostics.
- Extend [openapi_contract_test.go](/mnt/c/Users/danii/IdeaProjects/go-service-template-rest/internal/infra/http/openapi_contract_test.go) to cover the start endpoint and the operation resource state surface.
- Add build-tagged integration tests in [payout_export_idempotency_integration_test.go](/mnt/c/Users/danii/IdeaProjects/go-service-template-rest/test/payout_export_idempotency_integration_test.go) to prove durable replay/conflict behavior across fresh app wiring and concurrent duplicate attempts.

**Scenario Coverage**
- `TestStartExportCreatesPendingOperationForNewKey`: new tenant/key/payload creates exactly one operation in `pending` and enqueues one export job.
- `TestStartExportReplaysSameOperationForSameKeySamePayload`: same tenant, same key, same payload returns the same operation id and does not enqueue a second job.
- `TestStartExportRejectsSameKeyDifferentPayload`: same tenant and key with a changed filter/date payload returns conflict and leaves the original operation intact.
- `TestStartExportConcurrentSameKeySamePayloadCreatesSingleOperation`: two overlapping starts with the same key/payload converge on one operation and one enqueue.
- `TestStartExportReturnsQuotaExceededWithRetryAfter`: quota exhaustion maps to a throttled result carrying retry guidance instead of generic validation failure.
- `TestOperationLifecyclePendingRunningSucceeded`: deterministic worker progression moves the operation `pending -> running -> succeeded`.
- `TestOperationLifecyclePendingRunningFailed`: deterministic worker progression moves the operation `pending -> running -> failed`.
- `TestPayoutExportStartReturns202AndLocation`: valid HTTP request with `Idempotency-Key` returns `202` and `Location: /v1/payout-exports/operations/{operation_id}`.
- `TestPayoutExportStartReplayReturnsSameLocationForSameKeySamePayload`: repeated HTTP start with identical key/payload resolves to the same operation resource.
- `TestPayoutExportStartRejectsMissingIdempotencyKeyWithRequestID`: missing required key fails explicitly and preserves the request id in visible diagnostics.
- `TestPayoutExportStartRejectsUnknownFields`: extra JSON members fail strict decoding.
- `TestPayoutExportStartRejectsTrailingGarbage`: valid JSON followed by extra tokens fails strict decoding.
- `TestPayoutExportStartOversizedBodyReturns413Not400`: request body overflow returns `413 Request Entity Too Large`, not a generic decode/validation `400`.
- `TestPayoutExportStartMapsQuotaExceededTo429WithRetryAfter`: throttled start returns `429 Too Many Requests` and a `Retry-After` header.
- `TestPayoutExportStartMapsSameKeyDifferentPayloadTo409`: same key with different payload at the HTTP boundary returns conflict and preserves request-id diagnostics.
- `TestOpenAPIRuntimeContractPayoutExportOperationStates`: operation resource exposes `pending`, `running`, `succeeded`, and `failed` states through the runtime contract.
- `TestPayoutExportIdempotencyPersistsAcrossFreshRequests`: integration test proves replay returns the same operation id after fresh app wiring.
- `TestPayoutExportConflictPersistsAcrossFreshRequests`: integration test proves same-key different-payload conflict is durable, not process-local.

**Key Test Files**
- [service_test.go](/mnt/c/Users/danii/IdeaProjects/go-service-template-rest/internal/app/payoutexport/service_test.go): business-level proof for idempotency, throttle handling, enqueue dedupe, and lifecycle without transport noise.
- [payout_export_test.go](/mnt/c/Users/danii/IdeaProjects/go-service-template-rest/internal/infra/http/payout_export_test.go): `httptest` boundary coverage for headers, status mapping, strict JSON behavior, size limits, and visible diagnostics.
- [openapi_contract_test.go](/mnt/c/Users/danii/IdeaProjects/go-service-template-rest/internal/infra/http/openapi_contract_test.go): runtime contract assertions for the start endpoint and operation polling surface.
- [payout_export_idempotency_integration_test.go](/mnt/c/Users/danii/IdeaProjects/go-service-template-rest/test/payout_export_idempotency_integration_test.go): integration proof that replay/conflict semantics survive fresh process wiring and concurrency.

**Validation Commands**
- `go test ./internal/app/payoutexport -run 'TestStartExport|TestOperationLifecycle'`
- `go test ./internal/infra/http -run 'TestPayoutExportStart|TestOpenAPIRuntimeContractPayoutExport'`
- `go test -race ./internal/app/payoutexport ./internal/infra/http`
- `go test -tags integration ./test -run 'TestPayoutExport(IdempotencyPersistsAcrossFreshRequests|ConflictPersistsAcrossFreshRequests|ConcurrentSameKeySamePayloadCreatesSingleOperation)'`
- `go test ./...`

**Observed Result**
- Ran `go test ./internal/infra/http` on the current unmodified tree: `ok github.com/example/go-service-template-rest/internal/infra/http (cached)`.
- Ran `go test ./...` on the current unmodified tree: existing packages passed, including `cmd/service/internal/bootstrap`, `internal/app/health`, `internal/app/ping`, `internal/config`, `internal/infra/http`, `internal/infra/postgres`, and `internal/infra/telemetry`.
- No post-change validation was run because this task explicitly forbade repository edits.

**Design Escalations**
- The prompt requires rejection for missing `Idempotency-Key`, but not the exact status. Lock `428 Precondition Required` versus `400 Bad Request` before writing the final HTTP assertions.
- Same-key same-payload retries are specified as equivalent, but not whether the replay response must be `202` again or another non-error replay form. The tests should at minimum lock same operation identity and same `Location`.
- "Visible diagnostics" for correlation need an explicit contract: echoed `X-Request-ID`, `request_id` in the problem body, or both.
- `Retry-After` is required on `429`, but its format and source are unspecified. Decide `delta-seconds` versus `HTTP-date` before making the header assertion exact.
- The operation states are named, but the terminal body contract for `succeeded` and `failed` is not. If result URL or structured failure details are required, that needs to be fixed before exhaustive contract tests.

**Residual Risks**
- Without the durable and concurrent idempotency tests, the endpoint can still double-enqueue after restart or under multi-instance contention.
- Without the strict JSON and `413` split tests, malformed and oversized requests can collapse into the same generic `400` path.
- Without explicit request-id assertions on failure responses, retry/debug correlation can disappear exactly on the paths operators need most.
- Without lifecycle contract coverage for the operation resource, the start endpoint can look correct while polling semantics drift or regress.
