# Go Test Construction Patterns

## Behavior Change Thesis
When loaded for ordinary Go test construction, this file makes the model keep scenario intent, fixtures, helpers, and assertions visible instead of likely mistake: helper-heavy tests, table-driven theater, opaque assertion frameworks, or unsafe `t.Parallel()` use.

## When To Load
Load this when writing or refactoring ordinary Go tests, subtests, fixtures, helpers, fuzz seeds, benchmarks, or examples and no narrower API, data, concurrency, or error reference owns the decision.

## Decision Rubric
- Put tests in `*_test.go`; use same-package tests for unexported behavior and `_test` package tests for black-box exported behavior.
- Prefer standard `testing` assertions unless nearby tests already standardize another helper style.
- Use table-driven tests when case differences stay readable and shared setup reduces noise; otherwise write separate tests.
- Use `t.Run` to name subcases that can fail independently.
- Use `t.Helper()` in helpers that can fail the test, and keep helpers thin enough that the behavior remains visible at the call site.
- Use `t.Setenv`, `t.TempDir`, and `t.Cleanup` for local resource control. Do not hand-roll global cleanup when testing already gives a scoped primitive.
- Do not use process-wide helpers like `t.Setenv` or `t.Chdir` in tests that call `t.Parallel()` or have parallel ancestors.
- Use `t.Parallel()` only when temp files, ports, mocks, package globals, and shared fakes are isolated.
- For fuzz tests, add small regression seeds with `f.Add` and one `f.Fuzz` target.
- For Go 1.24+ benchmarks, prefer `for b.Loop()` in new code unless nearby style or compatibility requires `b.N`; do not mix the two loop styles.

## Imitate
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

Copy the shape: case names are behavior statements, assertions include input, and the error category is explicit.

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

Copy the shape: fuzzing asserts an invariant and includes named regression seeds.

## Reject
```go
func TestParseLimit(t *testing.T) {
	got, _ := ParseLimit("many")
	if got == 0 {
		t.Fatal("bad")
	}
}
```

Reject because it ignores the error contract, hides the expected result, and accepts many wrong outcomes.

```go
func TestEverything(t *testing.T) {
	h := newHugeHarness(t)
	h.RunAll()
}
```

Reject because the review cannot see which scenario failed, which behavior was asserted, or whether helper logic mirrors implementation.

## Agent Traps
- Turning every pair of cases into a table even when a named standalone test would be clearer.
- Comparing whole complex structs when only selected fields are contractually meaningful.
- Using `reflect.DeepEqual` on errors, HTTP responses, timestamps, or structs with unstable fields.
- Calling `t.Parallel()` inside tests that share environment variables, package-level fakes, ports, or mutable fixtures.
- Letting cleanup errors disappear. Use `t.Cleanup` and report cleanup failures with `t.Errorf` when they matter.

## Validation Shape
- Focused test name first after construction changes.
- Package-level command when helpers, fixtures, or table shared setup can affect neighboring tests.
- Broader repository command only when test helpers are shared outside the package or generated/testdata assets changed.
