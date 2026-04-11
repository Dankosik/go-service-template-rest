# Errors And Contracts Review

## Behavior Change Thesis
When loaded for error-contract symptoms, this file makes the model choose the caller-observable failure contract and hidden-success risk instead of likely mistake "always use `%w`", "always make a custom error type", or "the log is enough."

## When To Load
Load when a Go review touches returned errors, sentinel or typed errors, wrapping with `%w` or `%v`, `errors.Is`, `errors.As`, `errors.AsType`, `errors.Join`, `(nil, nil)` ambiguity, panic-as-control-flow, logs replacing returns, or package API docs that promise error behavior.

## Decision Rubric
- Start from caller policy: what must callers distinguish, retry, ignore, log, translate, or redact?
- Treat swallowed errors and log-only failures as merge-risk when callers observe success or partial state.
- Use `%w` only when exposing the cause is part of the package contract; use `%v` or a package-owned error when the cause is diagnostic-only.
- Prefer `errors.Is` or `errors.As` when wrapping can occur and callers inspect a contract; direct `==` is fine at a seam where the exact sentinel is the contract and wrapping is impossible.
- On Go 1.26+, prefer `errors.AsType` when the target type itself implements `error` and its return form is clearer; keep `errors.As` for older Go versions or non-error interface targets.
- `errors.Join` changes the error tree. Check whether custom traversal misses joined causes or whether joining now exposes causes callers should not inspect.
- Reject string matching on `err.Error()` for policy unless the string is the only external protocol available and that fragility is documented.
- Treat `(nil, nil)` as a finding when absence and success are indistinguishable; prefer `(value, ok)` or `(value, error)` according to caller needs.
- Do not require typed errors for purely diagnostic failures that no caller should branch on.

## Imitate
```text
[high] [go-idiomatic-review] internal/store/user.go:87
Issue: This now returns fmt.Errorf("query user: %w", sql.ErrNoRows) from an exported repository method.
Impact: Exposing sql.ErrNoRows via %w makes the database package's sentinel part of this package's API; callers may start depending on errors.Is(err, sql.ErrNoRows), which blocks a later datastore change.
Suggested fix: Return the package-owned ErrUserNotFound sentinel, or wrap with %v if the SQL cause is only diagnostic text.
Reference: Go error wrapping contract
```

Copy the contract boundary: the finding is not "wrap vs do not wrap"; it is whether the cause becomes caller-observable API.

```text
[high] [go-idiomatic-review] internal/importer/parse.go:112
Issue: The parser logs the decode failure and returns nil, so callers observe success.
Impact: The import can continue with partial data and no inspectable failure, making retries and user-facing diagnostics impossible.
Suggested fix: Return the decode error with context, and let the caller decide whether and where to log it.
Reference: handle-errors review rule
```

Copy the hidden-success framing: the log is not a substitute for a returned failure at a policy seam.

```text
[medium] [go-idiomatic-review] internal/http/client.go:51
Issue: The code checks strings.Contains(err.Error(), "deadline") instead of preserving and inspecting the wrapped error.
Impact: The check is brittle across standard-library wording and additional wrapping; it can miss context.DeadlineExceeded.
Suggested fix: Preserve the original error and check errors.Is(err, context.DeadlineExceeded) at the policy seam.
Reference: errors.Is inspection contract
```

Copy the brittleness proof: explain what specific caller policy can misfire.

## Reject
```text
Use %w because that is more idiomatic.
```

Reject because wrapping is an API decision. It may accidentally expose an implementation detail.

```text
Make this a custom error type.
```

Reject unless callers need structured inspection. A contextual opaque error can be the safer contract.

```text
errors.Is is always better than ==.
```

Reject because direct sentinel equality can be correct at a local seam where wrapping cannot occur.

## Agent Traps
- Do not collapse every error issue into formatting style; prove caller-visible behavior.
- Do not ask for more logging when the bug is that the error is not returned.
- Do not expose SQL, HTTP, or third-party sentinels from exported packages unless that dependency is intentionally part of the contract.
- Do not claim `errors.Join` or `%w` behavior without checking the repository's Go version when version support is relevant.
- Do not take transport status mapping, retry policy, or domain meaning as this lane's final authority; hand off when needed.

## Validation Shape
- Add table tests that assert version-appropriate `errors.Is`, `errors.As`, or `errors.AsType` behavior for the exported contract, not the entire error string.
- Add a negative test proving an implementation detail is not exposed when opacity is intentional.
- Inject an error on the hidden-success path and assert no success-side effect occurs.
- Use focused package tests for local contracts; broader `go test ./...` is appropriate when exported behavior crosses packages.

## Handoffs
- Hand off transport status mapping to API or chi review lanes.
- Hand off retry and timeout policy to reliability review.
- Hand off business-specific error meaning to domain review.
- Hand off security-sensitive error disclosure to security review.
