# Runtime And Implementation Sequence

## Implementation Order

1. OpenAPI test selector alignment.
   - Rename the security-decision guard test into the `TestOpenAPIRuntimeContract...` family.
   - Run or plan to run `make openapi-runtime-contract-check`.

2. Protected endpoint README guidance.
   - Update `internal/api/README.md` after the test-selector change so the contract proof and human recipe align.
   - If tightening Problem-response guard code is included, do it while already touching `internal/infra/http/openapi_contract_test.go`.

3. Ping history sample limit.
   - Add sample-local limit validation before SQL.
   - Add unit tests for over-limit behavior.
   - Run `go test ./internal/infra/postgres -count=1`.

4. Redis policy consolidation.
   - Add config-owned Redis mode/readiness API.
   - Replace validation and bootstrap call sites.
   - Remove redundant bootstrap normalizer if no longer used.
   - Add or update config/bootstrap tests.
   - Run `go test ./internal/config ./cmd/service/internal/bootstrap -count=1`.

5. Documentation placement/discoverability updates.
   - Apply planned docs tasks from `research/coverage-audit.md`.
   - Keep docs compact and link to canonical guides instead of duplicating full sections.

6. Final targeted proof.
   - Run all targeted commands from `plan.md`.
   - Review docs against `research/coverage-audit.md`.
   - Run `make openapi-check` and `make check` when local tooling is available and not blocked by environment.

## Failure Points

- If the OpenAPI runtime-contract target still misses the security-decision guard, the test rename or selector change is incomplete.
- If the README suggests handler-owned auth or broad root middleware, it weakens generated-route and public-route boundaries.
- If ping history only checks `limit <= 0`, the sample still teaches unsafe SQL limit behavior.
- If Redis store mode is still normalized independently in bootstrap and config, the policy remains split.

## Side Effects

- No runtime behavior changes for existing HTTP endpoints are expected.
- No schema or generated sqlc changes are expected.
- Redis bootstrap behavior should remain the same; only policy ownership should change.
