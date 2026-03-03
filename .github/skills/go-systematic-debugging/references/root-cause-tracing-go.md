# Root-Cause Tracing For Go Services

## Overview
When a defect appears deep in the stack, fixing the crash point usually treats only the symptom.
The goal is to trace backward until you find where the bad state was first created.

## Use When
- stack traces point to infra/runtime layer, but source input is unclear
- failure happens across multiple layers (HTTP -> app -> domain -> infra -> DB/cache)
- same symptom appears in different entry points

## Trace Procedure
1. Record symptom location (`file:line`, panic/error message, failing test).
2. Identify immediate caller and input values at that point.
3. Move one layer up and ask: who provided this value/state?
4. Repeat until the first boundary where invariant is broken.
5. Fix at that boundary, then add guardrails for downstream layers.

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
- run focused test first:
  - `go test ./... -run '^TestName$' -count=1 -v`
- inspect first failing assertion/panic
- search upstream call path with `rg` and `go test -run`
- if flake: run repeated single test and capture timestamps/context IDs

## Stop Condition
Tracing is complete only when you can answer:
- which invariant failed first,
- where that invariant should have been enforced,
- why downstream layers did not prevent symptom spread.
