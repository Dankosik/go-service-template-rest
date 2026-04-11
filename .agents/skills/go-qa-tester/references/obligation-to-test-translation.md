# Obligation To Test Translation

## When To Load
Load this when the task gives approved requirements, invariants, review findings, bug reproduction notes, or a test plan and you need to turn them into named Go tests. Do not load this to invent missing product, API, data, or rollout semantics.

## Translation Steps
1. Extract obligations in the user's or approved artifact's vocabulary.
2. Classify each obligation as `happy`, `fail`, `edge`, `abuse`, `duplicate`, `retry`, `concurrency`, `timeout`, `partial failure`, `migration`, or `degradation` only when the behavior truly needs that scenario.
3. Identify the observable proof: return value, persisted state, emitted response, cache effect, async operation state, wrapped error, side-effect suppression, or bounded goroutine exit.
4. Name the smallest test that would fail if the obligation regressed.
5. Record unresolved exactness as an escalation, not as guessed assertions.

## Good Example
Approved obligation: "Starting the export twice with the same idempotency key and same payload returns the existing operation and creates no second export."

```go
func TestStartExportSameKeySamePayloadReturnsExistingOperation(t *testing.T) {
	store := newRecordingExportStore()
	svc := NewExportService(store)

	first, err := svc.Start(context.Background(), StartExportRequest{
		IdempotencyKey: "key-1",
		Filter:         "active",
	})
	if err != nil {
		t.Fatalf("first Start() error = %v, want nil", err)
	}

	second, err := svc.Start(context.Background(), StartExportRequest{
		IdempotencyKey: "key-1",
		Filter:         "active",
	})
	if err != nil {
		t.Fatalf("second Start() error = %v, want nil", err)
	}
	if second.ID != first.ID {
		t.Fatalf("second operation ID = %q, want existing %q", second.ID, first.ID)
	}
	if store.createCalls != 1 {
		t.Fatalf("create calls = %d, want 1", store.createCalls)
	}
}
```

Why it is good: it proves the approved duplicate behavior and the single side effect without freezing a storage reservation or fingerprint model.

## Bad Example
```go
func TestStartExportReplayUsesRedisReservationV2(t *testing.T) {
	got := cacheKeyFor("tenant-a", "key-1")
	if got != "export:v2:tenant-a:key-1" {
		t.Fatalf("cache key = %q", got)
	}
}
```

Why it is bad: it assumes an internal cache-key strategy. Use this only if the approved design already chose that exact key format.

## Assertion Patterns
- Prefer `if got != want { t.Fatalf("field = %v, want %v", got, want) }` for scalar outcomes.
- Prefer `errors.Is(err, ErrThing)` for sentinel categories and `errors.As(err, &target)` for structured causes.
- Assert side-effect suppression directly, such as one write, no publish, no extra row, or unchanged state.
- When exact transport status is not approved, assert through the service/domain layer and escalate the missing transport mapping.
- Keep test names behavior-first: `Test<Subject><Condition><Outcome>`.

## Deterministic Coordination Patterns
- Use a fake or recording dependency for side-effect counts instead of checking logs or sleeps.
- Use pre-seeded in-memory state for duplicate/retry cases.
- Use explicit handshakes such as `ready := make(chan struct{})` before canceling or triggering a race-sensitive path.
- Avoid "eventually" assertions unless the system is truly asynchronous and the poll has a bounded, diagnostic timeout.

## Repository-Local Cues
- `internal/infra/http/openapi_contract_test.go` names runtime contract scenarios and asserts status/body/header behavior with `httptest`.
- `internal/infra/postgres/ping_history_repository_test.go` tests query errors with `errors.Is` and context-specific error messages.
- `internal/config/config_test.go` uses many fail-path tests with explicit sentinel error checks.

## Exa Source Links
- [Go testing package](https://pkg.go.dev/testing)
- [Go errors package](https://pkg.go.dev/errors)
- [Go context package](https://pkg.go.dev/context)
- [Go race detector article](https://go.dev/doc/articles/race_detector.html)

