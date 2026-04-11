# Context And Lifetime Review

## Behavior Change Thesis
When loaded for context lifetime symptoms, this file makes the model review cancellation ownership and request lifetime instead of likely mistake "add context everywhere" or "never use `context.Background()`."

## When To Load
Load when a Go review touches `context.Context` parameters, request-scoped work, cancellation, deadlines, stored contexts, nil contexts, `context.Background`, `context.TODO`, `FooContext` additions, or cancellation-related errors.

## Decision Rubric
- Treat `ctx` as caller-owned on request-scoped work; replacing it with `Background` or `TODO` is a finding when it lets downstream work outlive the caller.
- Accept `Background` at process roots, startup wiring, tests, and work that intentionally is not tied to a caller request.
- Do not flag missing context on pure CPU/local helpers unless cancellation, deadlines, request values, or blocking I/O actually matter.
- Prefer `ctx context.Context` as the first parameter for operations that observe request cancellation or deadline policy.
- Avoid storing `context.Context` in long-lived structs; accept a stored context only for a clearly scoped object whose lifetime is the operation itself.
- Require `cancel()` when the current function owns a derived context with a timer, deadline, or cancellation resource.
- Preserve `context.Canceled` and `context.DeadlineExceeded` when callers make retry, logging, or status decisions from them.
- For stable exported APIs, prefer additive `FooContext(ctx, ...)` or an approved compatibility path over breaking an existing signature.

## Imitate
```text
[high] [go-idiomatic-review] internal/payments/client.go:74
Issue: Charge ignores the caller's ctx and creates the HTTP request with context.Background().
Impact: A canceled request can still call the downstream payment service and hold resources after the caller has given up.
Suggested fix: Use http.NewRequestWithContext(ctx, ...) and keep ctx flowing through the outbound call.
Reference: context lifetime contract
```

Copy the caller-lifetime framing: the bug is not the presence of `Background`, it is losing the caller's cancellation boundary.

```text
[high] [go-idiomatic-review] internal/worker/worker.go:29
Issue: Worker stores the constructor ctx and uses it for every Fetch call.
Impact: Callers cannot set per-call deadlines or cancel one operation without canceling the whole worker lifetime.
Suggested fix: Remove ctx from Worker and accept ctx on Fetch and Process methods that perform request-scoped work.
Reference: contexts-in-structs policy
```

Copy the per-call ownership distinction: stored context is risky because it hides which operation owns cancellation.

```text
[medium] [go-idiomatic-review] internal/cache/refresh.go:44
Issue: context.WithTimeout creates a timer but the returned cancel function is never called.
Impact: The timer and child context can live until timeout even when the operation finishes early.
Suggested fix: Call cancel with defer immediately after construction when this function owns the child context.
Reference: context cancellation contract
```

Copy the release-path proof: the finding names the leaked resource and the local ownership of the cancel function.

## Reject
```text
Add context everywhere.
```

Reject because context is for request-scoped cancellation, deadlines, and values; a finding must identify the affected blocking or request path.

```text
Never use context.Background().
```

Reject because `Background` is correct at process roots and intentionally detached work. The defect is replacing a caller-owned context inside a request flow.

```text
Move ctx into the struct so every method can use it.
```

Reject because it often expands the lifetime mistake. Prefer method parameters unless the struct itself is scoped to one operation.

## Agent Traps
- Do not raise a context finding only because a function lacks `ctx`; first prove the function performs work where cancellation or deadlines matter.
- Do not demand a breaking public signature change just to add context. Check whether an additive API can preserve compatibility.
- Do not flatten cancellation errors into generic wrapping guidance; inspectability may be the contract.
- Do not use this file for deep goroutine shutdown, retry budgets, or auth values in context; hand off those lanes.

## Validation Shape
- Cancel the parent context before or during the operation and assert the downstream call stops or returns an inspectable cancellation error.
- For derived deadlines, test the fast deterministic branch when possible; avoid slow real-time tests.
- For exported compatibility, compile or test both old and new entry points.
- `go test ./path/to/pkg` is enough when the behavior is local; use broader commands only when the call path crosses packages.

## Handoffs
- Hand off goroutine lifetime, channel cancellation, and worker shutdown depth to concurrency review.
- Hand off retry and deadline budgets to reliability review.
- Hand off request identity, auth values, and tenant data in context to security or API lanes.
- Hand off exported API evolution decisions to design/API review if the smallest fix changes public shape.
