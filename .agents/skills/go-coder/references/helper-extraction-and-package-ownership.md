# Helper Extraction And Package Ownership

## Behavior Change Thesis
When loaded for helper extraction or package-boundary pressure, this file makes the model choose direct code or a seam-named same-package policy owner instead of generic utilities, provider-side interfaces, or exports added only for tests.

## When To Load
Load this when the implementation may extract helpers, introduce interfaces, move code across packages, export symbols, or centralize repeated policy.

## Decision Rubric
- Inline one-use logic when extraction only hides the main operation.
- Extract when repeated behavior is stable policy, not merely repeated syntax.
- Keep policy in the package that owns its meaning before creating `internal/`, `util`, `common`, `shared`, or cross-package helpers.
- Name helpers after policy or ownership, not mechanics.
- Define interfaces at consumer seams where substitution is needed; do not export provider-side interfaces as a default.
- Do not widen visibility for tests when behavior can be tested through the public path or a focused same-package test.

## Imitate
Keep domain normalization in the package that owns the rule.

```go
package users

func canonicalEmail(raw string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))
	if email == "" || !strings.Contains(email, "@") {
		return "", ErrInvalidEmail
	}
	return email, nil
}
```

Define the narrow interface where it is consumed.

```go
package handlers

type userFinder interface {
	Find(context.Context, users.UserID) (users.User, error)
}

type UserHandler struct {
	users userFinder
}
```

Prefer direct checks when the state transition becomes clearer.

```go
if input.Email == "" || input.Name == "" || input.Disabled {
	return User{}, ErrInvalidCreateInput
}
```

## Reject
Reject generic ownership erasure.

```go
package util

func Normalize(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
```

Reject exports whose only caller is a test.

```go
func NormalizeEmailForTest(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}
```

Reject mode-heavy helpers that merge policies only because they look similar.

```go
func normalize(raw string, allowEmpty bool, lower bool) (string, error) {
	// ...
}
```

## Agent Traps
- Extracting because a function looks long, even though the helper forces readers away from the state change.
- Creating `util`, `common`, `shared`, or `helpers` before identifying one owner and one contract.
- Moving local package policy into `internal/` too early; import scope is not ownership.
- Using booleans, callbacks, mode strings, or option bags to merge behaviors that should stay separate.
- Introducing interfaces at the provider side before a consumer has a substitution need.

## Validation Shape
- Search for near-duplicates before extracting; repeated stable policy can justify one same-package source of truth.
- Check imports after a move; a good boundary should not create cycles or broaden unrelated dependencies.
- Test through behavior when possible; use same-package tests only when the unexported policy is the unit under change.
- Run tests for the old call sites and for the package that now owns the helper.
