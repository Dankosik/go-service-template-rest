# Go Test Construction Patterns

## When To Load
Load this when writing or refactoring ordinary Go tests, helpers, subtests, fixtures, fuzz tests, benchmarks, or examples.

## Construction Rules
- Put tests in `*_test.go`; use same-package tests for unexported behavior and `_test` package tests for black-box exported behavior.
- Prefer standard `testing` structure unless the repository already standardizes a helper library.
- Use `t.Run` for named subcases and table-driven tests when shared setup improves clarity.
- Use `t.Helper()` in helpers that can fail the test.
- Use `t.Setenv`, `t.TempDir`, and `t.Cleanup` instead of global cleanup comments.
- Use `t.Parallel()` only when all state, environment, ports, temp files, mocks, and global hooks are isolated.
- For fuzz tests, call `f.Add` with small seed inputs and then one `f.Fuzz` target.
- For benchmarks, prefer `b.Loop()` on Go versions that support it and keep setup outside the measured loop.

## Good Example
```go
func TestParseLimit(t *testing.T) {
	testCases := []struct {
		name    string
		input   string
		want    int
		wantErr error
	}{
		{name: "empty uses default", input: "", want: 50},
		{name: "invalid rejects", input: "many", wantErr: ErrInvalidLimit},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseLimit(tc.input)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("ParseLimit(%q) error = %v, want %v", tc.input, err, tc.wantErr)
			}
			if got != tc.want {
				t.Fatalf("ParseLimit(%q) = %d, want %d", tc.input, got, tc.want)
			}
		})
	}
}
```

## Bad Example
```go
func TestParseLimit(t *testing.T) {
	got, _ := ParseLimit("many")
	if got == 0 {
		t.Fatal("bad")
	}
}
```

Why it is bad: it ignores the error contract, hides the input in the assertion, and accepts many wrong outcomes.

## Fuzz Example
```go
func FuzzParseLimit(f *testing.F) {
	for _, seed := range []string{"", "1", "100", "-1", "many"} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		got, err := ParseLimit(input)
		if err != nil {
			return
		}
		if got < 1 || got > 100 {
			t.Fatalf("ParseLimit(%q) = %d, want within [1,100]", input, got)
		}
	})
}
```

## Assertion Patterns
- Include inputs in failure messages when they explain the failure.
- Use `t.Fatal` or `t.Fatalf` when the rest of the test depends on the result.
- Use `t.Error` or `t.Errorf` when collecting independent assertion failures is helpful.
- Avoid comparing whole complex structs when only selected fields are contractually meaningful.
- Avoid `reflect.DeepEqual` on HTTP responses, errors, timestamps, or structs with unexported/unstable fields unless the exact whole value is the contract.

## Deterministic Coordination Patterns
- Use `t.TempDir` for filesystem tests; avoid shared fixture directories for mutable state.
- Use `t.Setenv` instead of manually saving/restoring environment variables.
- Use `t.Cleanup` for resource teardown and make cleanup failures visible with `t.Errorf`.
- For I/O behavior, prefer `testing/iotest` helpers such as `ErrReader`, `OneByteReader`, or `TimeoutReader` when they model the failure honestly.
- For filesystem abstractions, consider `testing/fstest.MapFS` and `fstest.TestFS` when the code accepts `fs.FS`.

## Repository-Local Cues
- `internal/config/config_test.go` uses `t.Setenv`, `t.TempDir`, sentinel errors, and focused configuration fail-path assertions.
- `internal/infra/http/router_test.go` favors explicit handler assertions over assertion frameworks.
- `test/postgres_integration_test.go` uses `t.Cleanup` for containers and pools.

## Exa Source Links
- [Go testing package](https://pkg.go.dev/testing)
- [Go fuzzing documentation](https://go.dev/doc/fuzz/)
- [testing/iotest](https://pkg.go.dev/testing/iotest)
- [testing/fstest](https://pkg.go.dev/testing/fstest)
- [testing/slogtest](https://pkg.go.dev/testing/slogtest)

