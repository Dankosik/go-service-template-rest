# Error Contracts

## Behavior Change Thesis
When loaded for error-contract symptoms, this file makes the model decide which cause becomes caller-observable API instead of likely mistake "`wrapcheck` is satisfied, so the error is fine."

## When To Load
Load when a Go review touches returned errors, sentinels, `%w` or `%v`, `errors.Is`/`As`/`AsType`/`Join`, a failure that is logged rather than returned, or cancellation errors a caller branches on.

## Decision Rubric
- `wrapcheck` requires context on an error crossing a package edge; it does not judge which cause that publishes. `%w` makes the cause's identity part of this package's contract — choose it when callers must inspect that cause, and a package-owned sentinel or `%v` when the cause is diagnostic only.
- A sentinel the caller may not import cannot be part of your contract. Where an import boundary forbids the dependency, as depguard does for the database driver outside its adapter, the adapter translates the cause into a package-owned sentinel and wraps the rest for diagnostics.
- `errorlint` and `nilnil` already gate sentinel `==` comparison and `(nil, nil)`. The shape that survives them is log-and-swallow: `nilerr` flags a bare `return nil` in an error branch, and a log call before that return defeats it. Return the error; the caller owns where it is logged.
- Keep `context.Canceled` and `context.DeadlineExceeded` inspectable through wrapping wherever drain, retry, or status decisions read them — `internal/background` reports a canceled task as an ordinary stop only because that identity survived.
- Go 1.26 provides `errors.AsType[E error](err) (E, bool)`; prefer it when the target is an error type and the returned value reads better than an out-parameter.

## Imitate
`fmt.Errorf("load user: %w", pgx.ErrNoRows)` from an exported adapter method — the finding is not wrap-versus-not, it is that the driver's sentinel just became this package's API.

## Reject
- "Use `%w` because it is more idiomatic" — wrapping is an API decision, and the linter that demanded it does not know which cause is safe to expose.
- "Make this a custom error type" — a contextual opaque error is the safer contract when no caller branches on it.

## Agent Traps
- Name the caller policy a finding protects: what the caller must distinguish, retry, translate, or redact.
- Report a logged-and-swallowed failure by the success the caller wrongly observes, not by the missing log line.
- Leave transport status mapping, retry budgets, and domain meaning to their own lanes.

## Validation Shape
- Assert the exported contract with `errors.Is` or `errors.AsType` against the sentinel callers may name, never against the message string.
- Add a negative test proving an intentionally opaque cause stays uninspectable.
- Inject a failure on the swallowed path and assert no success-side effect occurs.
