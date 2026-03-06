# Root-Cause Tracing For Go Services

## Overview
When a defect appears deep in the stack, fixing the crash point usually treats the symptom instead of the cause.
Trace backward until you find where the bad state was first created or first allowed through.

## Trace Procedure
1. Record the symptom location: `file:line`, panic or error message, failing test, and exact command.
2. Identify the immediate caller and the input values at that point.
3. Move one layer up and ask: who provided this value or state?
4. Repeat until you reach the first boundary where the invariant was already broken.
5. Fix at that boundary, then add downstream guardrails only if they prevent useful recurrence.

## Minimal Go Instrumentation Pattern
Use short-lived diagnostics while tracing.

```go
func debugBoundary(ctx context.Context, stage string, fields map[string]any) {
	fields["stage"] = stage
	fields["trace_id"] = traceIDFromContext(ctx) // optional helper
	log.Printf("debug boundary: %+v", fields)
}
```

Use near boundary transitions:

```go
debugBoundary(ctx, "http->app", map[string]any{
	"method": r.Method,
	"path":   r.URL.Path,
})

result, err := svc.Execute(ctx, cmd)

debugBoundary(ctx, "app->infra", map[string]any{
	"operation": "save_order",
	"has_error": err != nil,
})
```

## Error-Chain Tracing Pattern

```go
if err != nil {
	if errors.Is(err, context.DeadlineExceeded) {
		// timeout path
	}
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		// DB-specific origin
	}
}
```

## Test-Focused Backtracking
- run the narrowest failing test first
- inspect the first failing assertion or panic
- search upstream call paths
- if the issue is flaky, repeat the narrow test and capture timestamps or context IDs

## Stop Condition
Tracing is complete only when you can answer:
- which invariant failed first
- where that invariant should have been enforced
- why downstream layers did not stop the symptom earlier
