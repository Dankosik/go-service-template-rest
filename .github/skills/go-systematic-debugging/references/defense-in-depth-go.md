# Defense-In-Depth After Root-Cause Fix

## Overview
After a source-level fix, add focused safeguards so the same defect class cannot silently return through a different path.

## Layer Model
1. Transport boundary: decode, size limits, required fields, semantic validation
2. Application or use-case layer: business preconditions and transition checks
3. Infrastructure adapters: persistence, network, or cache safety constraints
4. Diagnostics layer: bounded logs, metrics, and traces for future forensics

## Guardrail Checklist
- Is boundary input validated before expensive side effects?
- Is the invariant checked where ownership belongs?
- Are infrastructure operations context-bounded and fail-fast?
- Is diagnostics sufficient to localize a recurrence quickly?

## Go Example: Boundary + Domain + Infra

```go
// Layer 1: transport validation
if req.AccountID == "" {
	return problem.BadRequest("account_id is required")
}

// Layer 2: domain/application invariant
if amount <= 0 {
	return ErrInvalidAmount
}

// Layer 3: infra safety
ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
defer cancel()
if err := repo.Save(ctx, entity); err != nil {
	return fmt.Errorf("save payment: %w", err)
}
```

## Anti-Overhardening Rule
Add only safeguards justified by the discovered failure mode.
Do not add unrelated guardrails that increase complexity without reducing recurrence risk.

## Verification
After adding guardrails:
- reproduce the old failing scenario and confirm it now fails safely or is rejected early
- run the regression path and confirm it still passes
- run the baseline quality checks needed for the changed scope