**Implemented Test Scope**
- No repository edits were made in this task; this is the exact test-addition plan for `POST /v1/payout-exports`.
- Add service-level tests in [service_test.go](/mnt/c/Users/danii/IdeaProjects/go-service-template-rest/internal/app/payoutexport/service_test.go) for idempotency semantics, quota rejection, enqueue-once behavior, and operation lifecycle transitions.
- Add HTTP boundary tests in [payout_export_test.go](/mnt/c/Users/danii/IdeaProjects/go-service-template-rest/internal/infra/http/payout_export_test.go) for `202`/`Location`, strict JSON failures, idempotency-key rejection/conflict, `429` + `Retry-After`, `413` split, and request-correlation diagnostics.
- Extend [openapi_contract_test.go](/mnt/c/Users/danii/IdeaProjects/go-service-template-rest/internal/infra/http/openapi_contract_test.go) to cover the start endpoint and `GET /v1/payout-exports/operations/{operation_id}` state exposure.
- Add a durable-store integration file [payout_export_idempotency_integration_test.go](/mnt/c/Users/danii/IdeaProjects/go-service-template-rest/test/payout_export_idempotency_integration_test.go) to prove replay/conflict behavior survives fresh app wiring.

**Scenario Coverage**
- `TestServiceStartExportCreatesPendingOperationForNewKey`: valid tenant, valid payload, and new `Idempotency-Key` create one operation in `pending` and enqueue exactly one export job.
- `TestServiceStartExportReusesOperationForSameKeySamePayload`: repeating the same tenant/key/payload returns the same operation id and does not enqueue a duplicate job.
- `TestServiceStartExportRejectsSameKeyDifferentPayload`: reusing the same key with a changed export filter or date range returns a conflict and leaves the original operation unchanged.
- `TestServiceStartExportReturnsQuotaExceededWithRetryAfter`: quota exhaustion returns a typed throttle result that carries retry guidance instead of being flattened into a generic validation error.
- `TestOperationLifecyclePendingRunningSucceededAndFailed`: with deterministic fake worker/repo coordination, the same operation moves through `pending -> running -> succeeded` and `pending -> running -> failed` without sleeps.
- `TestPayoutExportStartReturns202AndLocation`: HTTP `POST` with valid JSON and `Idempotency-Key` returns `202 Accepted` and a `Location` header matching `/v1/payout-exports/operations/{operation_id}`.
- `TestPayoutExportStartReplayReturnsSameOperationLocation`: the second HTTP call with the same key and same payload resolves to the same operation resource and does not create a new one.
- `TestPayoutExportStartRejectsMissingIdempotencyKeyAndPreservesRequestID`: missing required key returns the approved problem response and echoes the inbound `X-Request-ID` in visible diagnostics.
- `TestPayoutExportStartRejectsUnknownFields`: an extra JSON member fails strict decoding with `400`.
- `TestPayoutExportStartRejectsTrailingJSONGarbage`: a valid object followed by extra tokens fails with `400`.
- `TestPayoutExportStartOversizedBodyReturns413Not400`: body-limit overflow returns `413 Request Entity Too Large`, not a generic `400` decode/validation error.
- `TestPayoutExportStartReturns429WithRetryAfter`: quota exhaustion at the HTTP boundary maps to `429 Too Many Requests` and includes `Retry-After`.
- `TestPayoutExportStartConflictMapsDifferentPayloadReplayToProblem`: same key plus different payload over HTTP returns conflict and preserves request correlation id in the problem response.
- `TestOpenAPIRuntimeContractPayoutExportOperationStates`: the router exposes the operation resource with `pending`, `running`, `succeeded`, and `failed` status values and the approved status/header contract.
- `TestPayoutExportIdempotencyPersistsAcrossFreshRequests`: integration test posts the same tenant/key/payload against fresh app wiring and gets the same operation id both times.
- `TestPayoutExportConflictPersistsAcrossFreshRequests`: integration test posts the same tenant/key with a changed payload against fresh app wiring and gets a durable conflict instead of a second accepted job.

**Key Test Files**
- [service_test.go](/mnt/c/Users/danii/IdeaProjects/go-service-template-rest/internal/app/payoutexport/service_test.go): deterministic business-level proof for idempotency, quota, and lifecycle without transport noise.
- [payout_export_test.go](/mnt/c/Users/danii/IdeaProjects/go-service-template-rest/internal/infra/http/payout_export_test.go): request parsing, status-code split, `Location`, `Retry-After`, and `request_id` assertions with `httptest`.
- [openapi_contract_test.go](/mnt/c/Users/danii/IdeaProjects/go-service-template-rest/internal/infra/http/openapi_contract_test.go): runtime contract coverage for the new start endpoint and operation resource path.
- [payout_export_idempotency_integration_test.go](/mnt/c/Users/danii/IdeaProjects/go-service-template-rest/test/payout_export_idempotency_integration_test.go): build-tagged integration proof that replay/conflict behavior is durable, not process-local.

**Validation Commands**
- `go test ./internal/app/payoutexport -run 'TestServiceStartExport|TestOperationLifecycle'`
- `go test ./internal/infra/http -run 'TestPayoutExportStart|TestOpenAPIRuntimeContractPayoutExport'`
- `go test -race ./internal/app/payoutexport ./internal/infra/http`
- `go test -tags integration ./test -run 'TestPayoutExport(IdempotencyPersistsAcrossFreshRequests|ConflictPersistsAcrossFreshRequests)'`
- `go test ./...`

**Observed Result**
- Baseline only: `go test ./internal/infra/http` currently passes on the unmodified tree with `ok github.com/example/go-service-template-rest/internal/infra/http (cached)`.
- No post-change validation was run because this task explicitly forbade repository edits; the commands above are the post-implementation validation set.

**Design Escalations**
- The prompt requires rejection for a missing `Idempotency-Key` but does not pin the exact status code. If this API follows the repository’s existing idempotency pattern, make that `428 Precondition Required` before locking the HTTP assertion.
- Same-key same-payload is specified as “equivalent,” but not whether the second HTTP response must be `202` again or may be another non-error replay form. The tests should lock same operation identity and same `Location` until that replay status is explicit.
- `Retry-After` is required on `429`, but its format and derivation are not specified. Decide `delta-seconds` vs `HTTP-date` before making the header assertion exact.
- The operation states are named, but the terminal response body contract for `succeeded` and `failed` is not. If artifact URL or structured failure details are mandatory, that needs to be fixed before writing exhaustive body assertions.

**Residual Risks**
- Without the integration tests, idempotency can still appear correct in unit tests while duplicating exports after restart or across multiple instances.
- Without the `413` regression test, oversized bodies can accidentally collapse into the same generic `400` path as strict JSON decode failures.
- Without explicit `failed` operation coverage, the start endpoint may look correct while retry/recovery behavior regresses in the worker path.
- Without asserting `request_id` on rejected requests, request correlation can remain intact for happy-path traffic but disappear from the diagnostics users need during failures.
