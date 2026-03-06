# Claim To Proof Matrix (Go Service Template)

Use this matrix when choosing the smallest command set that still proves the claim.

## Common Claims

| Claim type | Minimum commands | Notes |
|---|---|---|
| Focused bugfix works | reproducer command that failed before now passes | Prefer a focused `go test -run '^TestName$' -count=1` when possible |
| Package behavior is green | `go test ./path/to/pkg/...` | Do not generalize to the whole repository |
| Repository tests pass | `make test` | Broad claim requires broad command |
| Concurrency-safe for changed path | `make test-race` or `go test -race ./...` | Required for goroutine, channel, or lock-sensitive changes |
| Lint clean | `make lint` | Lint does not prove build or behavior |
| Build succeeds | `make build` | Build does not prove runtime behavior correctness |
| API contract/runtime checks green | `make openapi-check` | Use when handlers, generated API code, or contract wiring changed |
| Migration safety checked | `make migration-validate` | Use when migration-related behavior changed |

## Readiness Claims

| Claim | Typical proving set (adjust by scope) |
|---|---|
| "Ready for scoped handoff" | scope-required tests plus the quality checks required for the changed surface |
| "Ready for review or merge" | blocking findings resolved plus required checks green for the changed surface |

## Reporting Pattern
For each command include:
- command string
- pass/fail
- key summary, such as `ok 42 packages`, `0 failures`, or the first blocking error

If any command fails:
- do not claim completion
- report the blocking command first
- provide the next remediation or verification step
