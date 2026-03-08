**Implemented Test Scope**
- No repository files were edited. This is the exact test set I would add.
- Smallest proving split: service-level unit tests for idempotency and lifecycle decisions, HTTP runtime-contract tests for transport semantics, and integration tests for real dedup persistence plus async side effects.

**Scenario Coverage**
- `TestStartExportRequiresIdempotencyKey`: service call without `Idempotency-Key` returns a typed precondition error that the handler maps to `428`, and creates no operation or enqueue side effect.
- `TestStartExportIdempotencyDecisions`: subtests `same_key_same_payload_returns_same_operation`, `same_key_different_payload_conflict`, and `same_key_other_tenant_allowed`; assert tenant-scoped dedup and duplicate enqueue count stays `1`.
- `TestStartExportQuotaExceeded`: returns a typed quota signal with retry delay metadata and does not advance lifecycle or enqueue work.
- `TestOperationStatusLifecycle`: subtests `pending`, `running`, `succeeded`, and `failed`; assert legal transitions and status read-model contents.

- `TestOpenAPIRuntimeContractPayoutExportAccepted`: `POST /v1/payout-exports` with valid JSON plus `Idempotency-Key` returns `202`, `Location` points to `/v1/payout-exports/operations/{operation_id}`, and `X-Request-ID` is echoed or generated.
- `TestOpenAPIRuntimeContractPayoutExportIdempotentRetryEquivalent`: same tenant, same key, same payload repeated over HTTP returns the same `Location` and no contract-visible divergence from the first `202`.
- `TestOpenAPIRuntimeContractPayoutExportRejectsMissingIdempotencyKey`: returns `428` with `application/problem+json` and preserved `request_id`.
- `TestOpenAPIRuntimeContractPayoutExportRejectsBadJSON`: subtests `unknown_field` and `trailing_garbage`; both return `400` problem responses with preserved `request_id`.
- `TestOpenAPIRuntimeContractPayoutExportRejectsOversizedBody`: body over configured limit returns `413`, not `400/422`, with problem envelope and preserved `request_id`.
- `TestOpenAPIRuntimeContractPayoutExportIdempotencyConflict`: same key with different payload returns `409` problem response with preserved `request_id`.
- `TestOpenAPIRuntimeContractPayoutExportQuotaExceeded`: returns `429`, includes parseable `Retry-After`, and preserves `X-Request-ID` plus problem `request_id`.
- `TestOpenAPIRuntimeContractPayoutExportOperationStatus`: `GET /v1/payout-exports/operations/{operation_id}` subtests `pending`, `running`, `succeeded`, and `failed`; each returns `200` with the expected status value, and `failed` includes structured error detail.

- `TestPayoutExportsIntegrationIdempotentRetryReusesOperation`: real HTTP and storage path proves first and second POSTs share one operation row and one background job.
- `TestPayoutExportsIntegrationIdempotencyConflictKeepsOriginalOperation`: conflicting retry returns `409`, preserves the original payload fingerprint and operation, and does not create a second job.
- `TestPayoutExportsIntegrationIdempotencyKeyScopedPerTenant`: same key and payload in different tenants produces separate operations.
- `TestPayoutExportsIntegrationQuotaExceededDoesNotCreateOperation`: exhausted quota returns `429` with `Retry-After` and leaves no operation or job residue.
- `TestPayoutExportsIntegrationOperationLifecycle`: worker-driven flow moves `pending -> running -> succeeded` and `pending -> running -> failed`, and the polled operation resource reflects stored state after each step.

**Key Test Files**
- Proposed [service_test.go](/mnt/c/Users/danii/IdeaProjects/go-service-template-rest/internal/app/payout_export/service_test.go)
- Proposed [payout_export_contract_test.go](/mnt/c/Users/danii/IdeaProjects/go-service-template-rest/internal/infra/http/payout_export_contract_test.go)
- Proposed [payout_exports_integration_test.go](/mnt/c/Users/danii/IdeaProjects/go-service-template-rest/test/payout_exports_integration_test.go)

**Validation Commands**
- `go test ./internal/app/payout_export -count=1`
- `go test ./internal/infra/http -run '^TestOpenAPIRuntimeContractPayoutExport' -count=1`
- `make openapi-check`
- `make test-race`
- `make test-integration`
- `make ci-local`

**Observed Result**
- `make openapi-runtime-contract-check` passes on the current baseline repo: `ok github.com/example/go-service-template-rest/internal/infra/http 0.016s`.
- No payout-export tests were added or executed because this task explicitly disallowed repository edits.

**Design Escalations**
- The current contract source [service.yaml](/mnt/c/Users/danii/IdeaProjects/go-service-template-rest/api/openapi/service.yaml) uses `/api/v1/...`, while the prompt uses `/v1/payout-exports`. The final prefix must be fixed before writing path-asserting tests.
- Same-key same-payload only says "equivalent". The tests above pin `202` plus identical `Location`; if replayed responses must also preserve body shape or `Retry-After`, the contract needs to say so.
- `429` requires `Retry-After`, but the prompt does not define delta-seconds vs HTTP-date or how the value is derived. I would assert presence and parseability unless the contract tightens it.
- The prompt does not say whether a quota-rejected request consumes the idempotency reservation. I would treat "no operation/no reservation created" as the safer behavior and lock it only after that rule is approved.
- "Visible diagnostics" is fully testable today as echoed `X-Request-ID` plus problem-body `request_id` because that is the current repo pattern in [problem.go](/mnt/c/Users/danii/IdeaProjects/go-service-template-rest/internal/infra/http/problem.go); if the operation resource itself must expose correlation data, that needs explicit contract text.

**Residual Risks**
- The proposed integration file assumes a real persistence seam for operations, idempotency fingerprints, and queued work; if the implementation hides those behind opaque infrastructure, the harness will need extra fixtures.
- Async lifecycle coverage will stay deterministic only if worker execution and time are injectable; otherwise `pending/running` assertions can become flaky.
- No fuzz target is proposed because strict table-driven contract tests are the smallest honest proof here; if the decoder path becomes custom rather than generated or standard, add a focused fuzz test later.
