# Go review checklist instructions for LLMs

## Load policy
- Load: Optional
- Use when:
  - Reviewing a pull request
  - Auditing code for idiomatic Go
  - Refactoring existing code to match best practices
  - Answering "what is wrong with this Go code?" or "how can this be more idiomatic?"
- Do not load when: The task is greenfield writing and no review or audit framing is requested

## Review posture

Review like an experienced Go engineer. Prioritize correctness, clarity, contract stability, tool compatibility, and maintainability over personal stylistic preference.

## Review order

Inspect in this order:

1. Correctness and API behavior
2. Error handling and context propagation
3. Concurrency safety and goroutine lifetime
4. Package boundaries and exported surface
5. Naming and readability
6. Tests and validation
7. Performance concerns that are justified by evidence
8. Dependency and toolchain hygiene

## What to check

### Formatting and imports
- Is the code naturally compatible with `gofmt`?
- Are imports minimal, correct, and organized as if by `goimports`?
- Are there unused imports or dead variables?

### Naming
- Are package names short, lowercase, and specific?
- Is there stutter in client code?
- Are initialisms capitalized idiomatically?
- Are receiver names short and consistent?

### Errors
- Are normal failures returned as errors instead of panics?
- Do errors include meaningful context?
- Is wrapping done with `%w` when cause inspection matters?
- Are `errors.Is` and `errors.As` used instead of string checks?
- Are error messages lowercase and punctuation-free?

### Context
- Is `ctx` the first parameter where needed?
- Is context propagated to blocking operations?
- Is context stored in a struct anywhere?
- Is every derived context canceled?

### Control flow
- Is the happy path easy to follow?
- Are there unnecessary `else` blocks after `return`?
- Are functions doing too many things at once?

### Interfaces and types
- Are interfaces small and justified?
- Are interfaces placed on the consumer side rather than added preemptively?
- Are pointers used only when semantically or structurally justified?
- Are zero values practical where they should be?

### Packages and modules
- Are package boundaries clear?
- Was a `util` or `common` package introduced without a strong reason?
- Should some code move into `internal/`?
- Is the exported API larger than necessary?
- Did any touched file become a mixed-responsibility "god file" that should be split into focused files within the same package?
- Are `go.mod` and `go.sum` being handled correctly?

### Concurrency
- Does every goroutine have a shutdown path?
- Are channel ownership and closure rules clear?
- Would `errgroup.WithContext` simplify error propagation and cancellation?
- Is shared state synchronized correctly?
- Should this code be checked with `go test -race`?

### Tests and quality
- Are there tests for nontrivial behavior?
- Would table-driven tests or subtests improve clarity?
- Are errors tested idiomatically?
- Are suggested quality commands aligned with Go norms?
- Should `staticcheck`, `golangci-lint`, or `govulncheck` be part of the recommendation?

### Performance
- Is there evidence for any performance-related change?
- Would a benchmark, profile, or trace be more appropriate than a speculative rewrite?

## Output style for review tasks

- Lead with the highest-impact issues first.
- Separate correctness issues from style improvements.
- Explain why a change is more idiomatic in Go terms.
- Prefer concrete rewrite suggestions over vague advice.
- Keep comments actionable and tool-aligned.

## Suggested validation commands

Depending on the scope of the review, recommend some or all of:
- `gofmt -w .`
- `go test ./...`
- `go test -race ./...`
- `go vet ./...`
- `staticcheck ./...`
- `golangci-lint run`
- `govulncheck ./...`

## What good output looks like

- The review focuses on real issues, not taste.
- Recommendations line up with how Go teams actually work.
- The most important fixes are obvious to the reader.
- Suggested changes are specific, idiomatic, and testable.
