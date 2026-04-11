# Root-Cause Tracing For Go Services

## Behavior Change Thesis
When loaded for deterministic symptoms with upstream bad state, this file makes the model backtrack to the first broken invariant instead of patching the crash site or adding a defensive nil guard.

## When To Load
Load when the symptom is deterministic or mostly deterministic, but the failing line may not be where the bad state was created: panics, typed-nil surprises, bad payload shape, incorrect state transitions, unexpected context errors, and integration regressions.

## Decision Rubric
- Preserve the first stack trace, first failed assertion, or first error-chain mismatch before editing.
- Inspect the immediate input at the symptom line, then walk one caller or boundary upstream at a time.
- Stop when you find where the value first became invalid or where validation first allowed it through.
- Fix at the earliest boundary that owns the invariant; add downstream guardrails only when they prevent recurrence or improve the error contract.
- Escalate instead of silently changing API, data, retry, timeout, security, or architecture semantics under defect pressure.

## Imitate

```text
Symptom: panic in handler response mapping.
Immediate bad value: *Order is nil inside an interface typed as Result.
Upstream boundary: repository returned (*Order)(nil), nil error for missing row.
First broken invariant: repository contract promises either non-nil *Order or ErrNotFound.
Fix scope: repository missing-row mapping plus regression test at repository boundary.
```

Copy the backtracking: the fix goes to the contract boundary that created bad state, not the mapper that crashed later.

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

Copy the typed-error preservation: do not stringify errors before the policy-owning caller can inspect them.

## Reject

```go
if result == nil {
	return nil
}
```

This may convert a contract violation into silent success while the upstream invalid value continues to leak.

```go
ctx := context.Background()
```

Using a new root context to make cancellation disappear destroys the caller's budget and hides the real boundary failure.

## Agent Traps
- Treating the panic site as the root cause because it has the loudest stack frame.
- Adding broad debug logs that include request bodies, SQL payloads, tokens, or PII.
- Forgetting typed-nil interface cases where `err != nil` or `value != nil` checks lie about the concrete value.
- Changing API or persistence behavior without naming the owning boundary and escalation.
- Keeping temporary boundary logging after the regression test proves the fix.

## Validation Shape
Record the narrow reproducer, the first broken invariant, the boundary where it first failed, the rejected hypotheses, and the exact command that proves the boundary-level regression now passes.
