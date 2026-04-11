# Errors And Contracts Review

## When To Load It
Load this reference when a Go review touches returned errors, sentinel or typed errors, wrapping with `%w` or `%v`, `errors.Is`, `errors.As`, `errors.Join`, `(nil, nil)` success ambiguity, panic-as-control-flow, logs replacing returns, or package API docs that promise error behavior.

## Exa Source Links
- [Working with Errors in Go 1.13](https://go.dev/blog/go1.13-errors)
- [errors package](https://pkg.go.dev/errors)
- [Go Code Review Comments: Handle Errors, Error Strings, In-Band Errors](https://go.dev/wiki/CodeReviewComments)
- [Errors are values](https://go.dev/blog/errors-are-values)
- [Effective Go: Errors](https://go.dev/doc/effective_go#errors), with the official caveat that Effective Go is not actively updated.

## Review Cues
- A caller could reasonably need to distinguish `not found`, `conflict`, `timeout`, `canceled`, or validation failure.
- Code wraps an implementation detail with `%w` and thereby exposes it as part of the API.
- Code uses `%v` where callers already rely on `errors.Is` or `errors.As`.
- Code compares `err == someErr` after wrapping can happen.
- Code parses `err.Error()` or hides failure behind logging.
- Code returns `(nil, nil)` for a missing value when the caller cannot tell whether work succeeded.

## Bad Review Examples
Bad review:

```text
Use %w because that is more idiomatic.
```

Why it is bad: it treats wrapping as style and may accidentally make an implementation detail part of the public contract.

Bad review:

```text
Don't return this error string; use a custom error type.
```

Why it is bad: it skips the caller contract. If callers only log the failure, an opaque error with context can be enough.

Bad review:

```text
errors.Is is always better than ==.
```

Why it is bad: direct equality can be correct at a local seam where wrapping cannot occur and the exact sentinel is the contract.

## Good Review Examples
Good finding:

```text
[high] [go-idiomatic-review] internal/store/user.go:87
Issue: This now returns fmt.Errorf("query user: %w", sql.ErrNoRows) from an exported repository method.
Impact: Exposing sql.ErrNoRows via %w makes the database package's sentinel part of this package's API; callers may start depending on errors.Is(err, sql.ErrNoRows), which blocks a later datastore change.
Suggested fix: Return the package-owned ErrUserNotFound sentinel, or wrap with %v if the SQL cause is only diagnostic text.
Reference: https://go.dev/blog/go1.13-errors
```

Good finding:

```text
[high] [go-idiomatic-review] internal/importer/parse.go:112
Issue: The parser logs the decode failure and returns nil, so callers observe success.
Impact: The import can continue with partial data and no inspectable failure, making retries and user-facing diagnostics impossible.
Suggested fix: Return the decode error with context, and let the caller decide whether and where to log it.
Reference: https://go.dev/wiki/CodeReviewComments
```

Good finding:

```text
[medium] [go-idiomatic-review] internal/http/client.go:51
Issue: The code checks strings.Contains(err.Error(), "deadline") instead of preserving and inspecting the wrapped error.
Impact: The check is brittle across stdlib wording and translated/wrapped errors; it can miss context.DeadlineExceeded after additional wrapping.
Suggested fix: Preserve the original error and check errors.Is(err, context.DeadlineExceeded) at the policy seam.
Reference: https://pkg.go.dev/errors
```

## Real Merge-Risk Impact
- Hidden success: swallowed errors can commit partial state or acknowledge failed work.
- Contract break: changing `%v` to `%w`, or vice versa, can silently add or remove caller-observable API.
- Diagnosability loss: string matching and double logging make failure classification and search unreliable.
- Compatibility lock-in: wrapping third-party or SQL errors can freeze implementation choices for exported packages.
- Panic risk: using panic for normal input or IO failure can take down a server path instead of returning a controlled error.

## Smallest Safe Correction
- Return the error instead of logging and continuing.
- Use `%w` only when caller inspection of the cause is intended; otherwise use `%v` or a package-owned error.
- Replace string matching with `errors.Is` or `errors.As` when wrapping can occur.
- Document exported error properties that callers may rely on.
- Return `(value, ok)` or `(value, error)` instead of an in-band ambiguous zero value.
- Preserve `context.Canceled` and `context.DeadlineExceeded` when cancellation semantics matter.

## Validation Ideas
- Add table tests that assert `errors.Is` or `errors.As` for the exported contract, not the exact full string.
- Add a negative test proving an implementation detail is not exposed if opacity is intentional.
- Test the hidden-success path by injecting an error and asserting no success-side effect occurs.
- Run `go test ./...` for local behavior and targeted package tests for error contracts.

## Handoffs
- Hand off transport status mapping to API or chi review lanes.
- Hand off retry and timeout policy to reliability review.
- Hand off business-specific error meaning to domain review.
- Hand off security-sensitive error disclosure to security review.
