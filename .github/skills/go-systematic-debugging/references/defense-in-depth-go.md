# Defense-In-Depth After Root-Cause Fix

## Overview
After source-level fix, add focused safeguards so the same defect class cannot silently return through a different path.

## Layer Model For This Repository
1. Transport boundary (`internal/infra/http`): decode, size limits, required fields, semantic validation.
2. Application/use-case layer (`internal/app`): business preconditions and transition checks.
3. Infra adapters (`internal/infra/*`): persistence/network/cache safety constraints.
4. Diagnostics layer: bounded logs/metrics/traces for future forensics.

## Guardrail Checklist
- Is boundary input validated before expensive side effects?
- Is domain/use-case invariant checked where ownership belongs?
- Are infra operations context-bounded and fail-fast?
- Is diagnostics sufficient to localize future failures quickly?

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
Do not add unrelated guardrails that increase complexity without reducing incident class risk.

## Verification
After adding guardrails:
- reproduce old failing scenario (must fail safely or be rejected early)
- run regression path (must pass)
- run baseline quality checks for changed scope
