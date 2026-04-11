# Defense-In-Depth After Root-Cause Fix

## When To Load
Load this reference after the root cause is proven and the local fix is clear, but recurrence prevention is still a question.

Use it to add only the guardrails justified by the discovered failure mode.

## Commands
Verify the source-level fix before adding guardrails:

```bash
go test ./path/to/pkg -run '^TestName$' -count=1 -v
go test ./path/to/pkg -run '^TestName$' -race -count=1 -v
go test ./path/to/pkg/...
go build ./...
```

When guardrails add diagnostics, run the command that exercises the bad input or failure path:

```bash
go test ./path/to/pkg -run '^TestRejectsBadInput$' -count=1 -v
go test ./path/to/pkg -run '^TestPreservesContextCancellation$' -count=1 -v
```

## Evidence To Capture
- proven root cause and first broken invariant
- guardrail layer: transport, application, domain, infrastructure, or diagnostics
- why each guardrail blocks recurrence of this defect class
- RED/GREEN proof for the old failure and any new guardrail test
- explicit note when a plausible guardrail was rejected as over-hardening

## Layer Model
1. Transport boundary: decode, size limits, required fields, semantic validation.
2. Application or use-case layer: business preconditions and transition checks.
3. Infrastructure adapters: persistence, network, cache, context propagation, and resource ownership.
4. Diagnostics layer: bounded logs, metrics, traces, or error wrapping for future forensics.

## Go Example

```go
// Layer 1: transport validation.
if req.AccountID == "" {
	return problem.BadRequest("account_id is required")
}

// Layer 2: domain/application invariant.
if amount <= 0 {
	return ErrInvalidAmount
}

// Layer 3: infrastructure safety.
ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
defer cancel()
if err := repo.Save(ctx, entity); err != nil {
	return fmt.Errorf("save payment: %w", err)
}
```

## Bad Debugging Moves
- adding unrelated validation, retries, metrics, and refactors because the file is already open
- hiding the root cause behind a defensive nil check with no regression test
- adding diagnostics with secret leakage or high-cardinality fields
- changing API, timeout, retry, or data semantics without escalating the decision
- keeping broad guardrails that no longer connect to the proven failure

## Good Debugging Moves
- fix the earliest valid boundary first
- add one guardrail at the layer that owns the invariant
- keep diagnostics bounded and useful for recurrence triage
- reject over-hardening explicitly when it adds complexity without blocking the defect class
- verify the original failure and the new guardrail path separately

## Source Links
- [Go diagnostics](https://go.dev/doc/diagnostics)
- [context package](https://pkg.go.dev/context)
- [errors package](https://pkg.go.dev/errors)
- [testing package](https://pkg.go.dev/testing)
