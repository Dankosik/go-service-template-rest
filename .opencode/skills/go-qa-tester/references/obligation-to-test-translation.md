# Obligation To Test Translation

## Behavior Change Thesis
When loaded for approved requirements, invariants, review findings, or bug notes, this file makes the model write named tests around observable obligations instead of likely mistake: branch-coverage tests, implementation-mirroring assertions, or guessed product semantics.

## When To Load
Load this when the hard part is turning an approved behavior statement or bug reproduction into a small set of Go test cases.

## Decision Rubric
- Extract obligations in the vocabulary of the approved artifact, not the current implementation.
- Give each obligation one scenario label only when it matters: `happy`, `fail`, `edge`, `abuse`, `duplicate`, `retry`, `concurrency`, `timeout`, `partial failure`, `migration`, or `degradation`.
- Name the observable proof before picking a package or helper: returned value, persisted state, emitted response, cache effect, async operation state, wrapped error, side-effect suppression, or bounded goroutine exit.
- Choose the smallest test that would fail if the obligation regressed.
- Record missing exactness as an escalation. Do not invent status codes, cache keys, tenant rules, retry policy, checkpoint strategy, or diagnostics fields to make a test look precise.

## Imitate
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

Copy the shape: the test proves duplicate semantics and single side effect without freezing a reservation, fingerprint, Redis key, or persistence model.

Bug obligation: "Malformed config duration returns the config-load category and includes the field name."

```go
func TestLoadConfigRejectsBadHTTPReadTimeout(t *testing.T) {
	t.Setenv("APP__HTTP__READ_TIMEOUT", "oops")

	_, err := LoadConfig(context.Background())
	if !errors.Is(err, ErrConfigLoad) {
		t.Fatalf("LoadConfig() error = %v, want ErrConfigLoad", err)
	}
	if !strings.Contains(err.Error(), "http.read_timeout") {
		t.Fatalf("LoadConfig() error = %q, want field context", err.Error())
	}
}
```

Copy the shape: stable category plus minimal diagnostic context, not an exact private formatting contract.

## Reject
```go
func TestStartExportReplayUsesRedisReservationV2(t *testing.T) {
	got := cacheKeyFor("tenant-a", "key-1")
	if got != "export:v2:tenant-a:key-1" {
		t.Fatalf("cache key = %q", got)
	}
}
```

Reject because it tests an internal tactic. It is valid only when the approved design already chose that exact key format as behavior.

```go
func TestStartExport(t *testing.T) {
	_, err := NewExportService(newRecordingExportStore()).Start(context.Background(), StartExportRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
}
```

Reject because it names no obligation, does not assert the error category, and could pass for the wrong rejection path.

## Agent Traps
- Treating a review finding such as "missing fail path" as permission to invent a full matrix.
- Testing helper functions only because they are easy to call, while the observable behavior remains unproved.
- Collapsing multiple obligations into one mega-test whose first failure hides the rest.
- Using names like `TestCreate` or `TestInvalidInput` when the scenario should expose condition and outcome.
- Mistaking "duplicate request" for a specific dedup storage strategy.

## Validation Shape
- Run the focused test name with `-count=1`.
- Add the package command when new helpers or fixtures interact with nearby tests.
- Add a broader command only when the obligation touches shared behavior or repository gates.
