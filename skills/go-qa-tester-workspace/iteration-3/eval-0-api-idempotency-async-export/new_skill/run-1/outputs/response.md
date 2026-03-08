**Implemented Test Scope**
- No repository files were edited. This is the exact test set I would add.
- Smallest honest split: service-level unit tests for idempotency and operation-state rules, HTTP runtime-contract tests for status/header/problem behavior, and DB-backed integration tests for durable deduplication plus async side-effect suppression.

**Scenario Coverage**
- `TestStartPayoutExportRequiresIdempotencyKey`: reject create before any operation row or enqueue side effect is created.
- `TestStartPayoutExportIdempotencyDecisions`: subtests `same_key_same_payload_reuses_operation`, `same_key_different_payload_conflict`, and `same_key_other_tenant_creates_new_operation`; assert tenant-scoped dedup and duplicate enqueue count stays `1`.
- `TestStartPayoutExportQuotaExceeded`: return typed quota exhaustion with retry delay metadata and create no operation or job.
- `TestPayoutExportOperationLifecycle`: subtests `pending_to_running_to_succeeded`, `pending_to_running_to_failed`, and `pending_to_succeeded_rejected`; assert only the approved lifecycle is legal.

- `TestOpenAPIRuntimeContractPayoutExportAccepted`: valid `POST` with `Idempotency-Key` returns `202`, sets `Location` to the operation resource, and echoes or generates `X-Request-ID`.
- `TestOpenAPIRuntimeContractPayoutExportReplayEquivalent`: same tenant, same key, same payload returns the same operation reference and no contract-visible duplicate side effect.
- `TestOpenAPIRuntimeContractPayoutExportMissingIdempotencyKey`: returns `428` with `application/problem+json` and matching `request_id`.
- `TestOpenAPIRuntimeContractPayoutExportConflict`: same key with different payload returns `409` with `application/problem+json` and matching `request_id`.
- `TestOpenAPIRuntimeContractPayoutExportQuotaExceeded`: returns `429`, includes `Retry-After`, preserves `X-Request-ID`, and exposes `request_id` in the problem body.
- `TestOpenAPIRuntimeContractPayoutExportRejectsStrictJSON`: subtests `unknown_field` and `trailing_json`; both return `400` and preserve correlation diagnostics.
- `TestOpenAPIRuntimeContractPayoutExportRejectsOversizedBody`: body over `RouterConfig.MaxBodyBytes` returns `413`, not `400/422`, with the problem envelope.
- `TestOpenAPIRuntimeContractPayoutExportOperationStatus`: subtests `pending`, `running`, `succeeded`, and `failed`; each polls the operation resource and asserts the expected status value, with structured failure detail on `failed`.

- `TestPayoutExportIntegrationIdempotentReplayCreatesSingleOperationAndSingleEnqueue`: real HTTP plus persistence path proves first and second POSTs share one durable operation and one queued export.
- `TestPayoutExportIntegrationConflictPreservesOriginalOperation`: conflicting replay returns `409`, leaves the original operation intact, and creates no second job.
- `TestPayoutExportIntegrationQuotaExceededCreatesNoOperation`: exhausted quota returns `429` with `Retry-After` and leaves no operation/idempotency residue.
- `TestPayoutExportIntegrationOperationLifecyclePolling`: worker-driven flow shows `pending -> running -> succeeded` and `pending -> running -> failed` through the polled operation resource.

**Key Test Files**
- Proposed `internal/app/payout_export/service_test.go`
- Proposed `internal/infra/http/payout_export_contract_test.go`
- Proposed `test/payout_export_integration_test.go`

**Validation Commands**
- `go test ./internal/app/payout_export -count=1`
- `go test ./internal/infra/http -run '^TestOpenAPIRuntimeContractPayoutExport' -count=1`
- `make openapi-check`
- `make test-race`
- `make test-integration`

**Observed Result**
- Executed baseline command: `go test ./internal/infra/http -run '^(TestOpenAPIRuntimeContractEndpoints|TestRouterRejectsRequestBodyTooLarge|TestRouterAddsRequestIDHeader)$' -count=1`
- Outcome: `ok github.com/example/go-service-template-rest/internal/infra/http 0.015s`
- No payout-export tests were added or executed because this task explicitly disallowed repository edits.

**Design Escalations**
- Current repo conventions in [service.yaml](/mnt/c/Users/danii/IdeaProjects/go-service-template-rest/api/openapi/service.yaml) and [router_test.go](/mnt/c/Users/danii/IdeaProjects/go-service-template-rest/internal/infra/http/router_test.go) use `/api/v1/...`, while the prompt says `/v1/payout-exports`. The final path prefix must be fixed before locking transport assertions.
- Same-key same-payload only says "equivalent". The tests above lock same operation reference and duplicate side-effect suppression; exact replay status/body after the operation has already finished still needs contract approval.
- `Retry-After` is required for `429`, but the prompt does not pin delta-seconds vs HTTP-date. I would assert presence and parseability unless the contract tightens it.
- The prompt does not define whether a quota-rejected request consumes an idempotency reservation. The integration test above assumes no reservation is created; that should be made explicit.
- Current problem responses already expose `request_id` in [problem.go](/mnt/c/Users/danii/IdeaProjects/go-service-template-rest/internal/infra/http/problem.go). If the operation resource itself must expose correlation identifiers, that needs explicit contract text.

**Residual Risks**
- The proposed package/file names assume the feature lands as `internal/app/payout_export`; adjust if the final service boundary or package naming differs.
- Async lifecycle tests will stay deterministic only if worker execution and timing are injectable; otherwise `pending` and `running` checks will become flaky.
- Durable dedup coverage assumes there is an inspectable persistence seam for operations and queued work; if the implementation hides both behind opaque infrastructure, the integration harness will need extra fixtures or repository-level probes.
