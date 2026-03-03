# Claim To Proof Matrix (Go Service Template)

Use this matrix when choosing the smallest command set that still proves the claim.

## Common Claims

| Claim type | Minimum commands | Notes |
|---|---|---|
| Focused bugfix works | Reproducer command that failed before now passes | Prefer `go test -run '^TestName$' -count=1` for deterministic proof |
| Package behavior is green | `go test ./path/to/pkg/...` | Do not generalize to whole repo |
| Repository tests pass | `make test` | Broad claim requires broad command |
| Concurrency-safe for changed path | `make test-race` or `go test -race ./...` | Required for goroutine/channel/lock-sensitive changes |
| Lint clean | `make lint` | Lint does not prove build/test pass |
| Build succeeds | `make build` | Build does not prove behavior correctness |
| API contract/runtime checks green | `make openapi-check` | Use when handlers/spec/generated API are affected |
| Migration safety checked | `make migration-validate` | Required when migration-related behavior changed |

## Gate-Oriented Claims

| Claim | Typical proving set (adjust by scope) |
|---|---|
| "Ready for Gate G3" | scope-required tests + required quality checks + no unresolved Spec Clarification blockers |
| "Ready for Gate G4" | reviewer blockers resolved + required quality checks green + no open Spec Reopen blockers |

## Reporting Pattern

For each command include:
- command string,
- pass/fail,
- key summary (for example: `ok 42 packages`, `0 failures`, or first blocking error).

If any command fails:
- do not claim completion,
- report blocking command first,
- provide next remediation command.
