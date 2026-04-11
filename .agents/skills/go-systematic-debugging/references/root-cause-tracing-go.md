# Root-Cause Tracing For Go Services

## When To Load
Load this reference when the symptom is deterministic or mostly deterministic but the failing line may not be where the bad state was created: panics, typed-nil surprises, bad payload shape, incorrect state transitions, unexpected `context` errors, and integration regressions.

Use it to walk backward from symptom to first broken invariant before choosing a fix.

## Commands
Keep the first command narrow and replayable:

```bash
go test ./path/to/pkg -run '^TestName$' -count=1 -v
go test ./path/to/pkg -run '^TestName$' -count=1 -failfast -json
go test ./path/to/pkg -run '^TestName$' -race -count=1 -v
go test ./path/to/pkg -run '^TestName$' -count=1 -timeout=30s -v
go build ./path/to/pkg
```

When request or fixture shape matters, save the smallest input that still fails and make the test consume that input directly.

## Evidence To Capture
- exact reproducer command and first failing signal
- first stack trace, first failed assertion, or first error-chain mismatch
- concrete input payload or fixture that crosses the boundary
- current invariant and where it should have been enforced
- caller and callee state at each boundary: transport, app, domain, persistence, cache, queue, and external dependency
- accepted and rejected hypotheses, with the command or observation that rejected each one

## Boundary Backtracking Pattern
1. Record the symptom location: `file:line`, panic or error message, failing test, and exact command.
2. Inspect the immediate inputs at that location.
3. Move one caller or boundary up and ask who created or accepted that value.
4. Repeat until the value first became invalid or first bypassed validation.
5. Fix at the earliest valid ownership boundary.
6. Add downstream guardrails only when they prevent useful recurrence or improve the error contract.

## Minimal Temporary Instrumentation
Use temporary diagnostics only when the existing test or logs cannot show the boundary.

```go
func debugBoundary(stage string, fields map[string]any) {
	fields["stage"] = stage
	log.Printf("debug boundary: %+v", fields)
}
```

Keep fields low-cardinality and secret-free. Remove the helper once the fix is proven unless the log has clear ongoing operational value.

## Error And Context Checks
Preserve typed error semantics while tracing:

```go
if err != nil {
	if errors.Is(err, context.Canceled) {
		// caller canceled the work
	}
	if errors.Is(err, context.DeadlineExceeded) {
		// owned budget expired
	}
	var target *SomeTypedError
	if errors.As(err, &target) {
		// origin-specific failure
	}
}
```

Do not stringify typed errors before the caller that owns the decision has a chance to inspect them.

## Bad Debugging Moves
- patching the panic site before finding who produced the bad value
- adding a nil guard that converts a contract violation into silent success
- replacing `ctx` with `context.Background()` to make an error disappear
- logging broad request or DB payloads that may contain secrets
- changing API, data, retry, or timeout behavior under the banner of a local bug fix

## Good Debugging Moves
- name the invariant in one sentence before editing
- keep the reproducer stable while moving up the call chain
- prove the first boundary where the invariant was already broken
- keep short-lived diagnostics easy to remove
- escalate when the fix belongs to API, data, reliability, security, or architecture policy

## Source Links
- [Go diagnostics](https://go.dev/doc/diagnostics)
- [context package](https://pkg.go.dev/context)
- [Go blog: context](https://go.dev/blog/context)
- [errors package](https://pkg.go.dev/errors)
- [testing package](https://pkg.go.dev/testing)
