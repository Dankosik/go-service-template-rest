# Go testing and quality instructions for LLMs

## Load policy
- Load: Optional
- Use when:
  - Writing, reviewing, or improving tests
  - Defining CI checks or local quality commands
  - Working with fuzzing, benchmarks, race detection, linting, or vulnerability scanning
- Do not load when: The task is only about a tiny code snippet and no testing or validation strategy is needed

## Testing principles

- Use the standard `testing` package by default.
- Test behavior and contracts, not incidental implementation details.
- Keep tests deterministic, isolated, and easy to run.
- Prefer simple fixtures over elaborate test frameworks.
- Make failure output specific and actionable.

## Unit test style

- Use table-driven tests when many cases share one structure.
- Name test cases clearly.
- Use subtests with `t.Run` for readability and selective execution.
- Use `t.Helper()` in helper functions.
- Use `t.Parallel()` only when the test is safe to run concurrently and isolation is clear.
- Prefer explicit expected values over clever assertion abstractions.

## Error and contract testing

- Check error behavior with `errors.Is` or `errors.As` when wrapping is involved.
- Test stable API behavior, not fragile error-message wording, unless the exact text is part of the contract.
- For public APIs, test the observable contract around nil, empty, timeout, cancellation, and edge cases where relevant.

## Benchmark guidance

- Use benchmarks for code on performance-sensitive paths.
- Benchmark representative workloads, not toy inputs only.
- Keep benchmark setup separate from the measured loop when possible.
- Use allocation reporting when it helps explain hot-path behavior.
- Do not treat microbenchmarks as substitutes for end-to-end profiling.

## Fuzzing guidance

- Use fuzz tests for parsers, decoders, protocol handling, serialization, and other input-heavy logic.
- Keep fuzz targets focused and deterministic.
- Minimize corpus noise and retain reproducing inputs for real bugs.

## Examples as documentation

- For public packages, add example functions when they clarify intended usage.
- Prefer examples that compile and reflect realistic usage.
- Keep examples short and idiomatic.

## Quality pipeline

Use a predictable, tool-driven quality pipeline. The typical baseline is:

- `gofmt -w .` or `go fmt ./...`
- `go test ./...`
- `go vet ./...`

For stronger coverage, add:

- `go test -race ./...`
- `staticcheck ./...`
- `golangci-lint run`
- `govulncheck ./...`

## Tool guidance

- `gofmt` is mandatory formatting.
- `goimports` is useful when import management should be automatic.
- `go vet` catches suspicious constructs; use it as a baseline.
- `staticcheck` is high-value and often worth running.
- `golangci-lint` is useful as a team entry point for a curated linter set.
- Choose linters intentionally. More linters is not automatically better.
- Do not rely on deprecated `golint`.

## Common anti-patterns to avoid

- Tests that depend on timing sleeps instead of proper synchronization
- Tests that compare wrapped errors by raw string
- Huge helper layers that hide what the test really does
- Benchmarks that include unrelated setup in the timed loop
- Enabling every linter without considering signal-to-noise ratio
- Skipping race detection for code that obviously uses concurrency
- Treating examples as prose instead of executable documentation

## What good output looks like

- Tests describe the intended behavior clearly.
- Quality commands match normal Go workflows.
- Static analysis complements tests instead of replacing them.
- Performance tests are separated from functional tests.
- The validation plan is realistic for CI and local development.

## Checklist

Before finalizing, verify that:
- Nontrivial logic has tests or a clear reason why not.
- Table-driven tests and subtests are used where they improve clarity.
- Error behavior is tested idiomatically.
- Concurrent code is a candidate for `go test -race`.
- The proposed lint and CI commands are focused and maintainable.
- Vulnerability scanning is included when dependency risk matters.
