# Source-Of-Truth Extraction

## When To Load
Load this when the same package repeats stable normalization, mapping, validation, classification, label shaping, section reading, or error mapping logic across files, or when a diff moves such policy into a vague helper bucket.

This file is for local simplification review. If the owner is unclear or the policy crosses API, data, config, or package boundaries, escalate instead of inventing ownership.

## Review Lens
- Under-extraction is real when repeated stable policy is likely to drift and a same-package owner would reduce future edits.
- Over-extraction is real when the helper erases ownership, needs modes or callbacks, or forces callers to know hidden policy.
- Prefer same-package, policy-named helpers over global `util/common/shared` packages.
- Do not extract orchestration order just to remove lines. Request lifecycle, startup, transaction, and response phases often read better locally.

## Real Finding Examples
Finding example: repeated local policy needs one same-package owner.

```text
[medium] [go-language-simplifier-review] internal/infra/http/problems.go:66
Issue: Three handlers now repeat the same domain-error to problem-status classification in file-local switches.
Impact: Adding a new error class requires synchronized edits across handlers, so the apparent local clarity can drift into endpoint-specific behavior.
Suggested fix: Extract a same-package helper such as `classifyProblem` and keep response writing local to each handler.
Reference: references/source-of-truth-extraction.md
```

Finding example: extraction moved policy to a vague bucket.

```text
[high] [go-language-simplifier-review] internal/common/normalize.go:12
Issue: Email canonicalization moved from the users package into `internal/common` even though the validation and error contract are user-domain policy.
Impact: Other packages can reuse a narrower normalization rule without the user validation contract, creating two source-of-truth surfaces for the same input.
Suggested fix: Keep the helper in the users package with a policy name such as `canonicalEmail`, or create a narrower owner only if the approved design names one.
Reference: references/source-of-truth-extraction.md
```

## Non-Findings To Avoid
- Do not extract one-off logic without a stable second use.
- Do not flag intentionally local test setup that keeps tests independent and readable.
- Do not recommend global helpers as the default source-of-truth fix.
- Do not require extraction when repeated shape hides distinct statuses, errors, or side effects.

## Bad And Good Simplifications
Bad: a helper name hides which policy owns the normalization.

```go
package common

func Normalize(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
```

Good: keep policy in the package that owns its meaning and error contract.

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

Bad: extraction makes one helper serve distinct callers through mode flags.

```go
func classify(err error, transport string, exposeDetails bool) int {
	// many unrelated cases
}
```

Good: keep one owner per stable classification rule.

```go
func classifyProblem(err error) int {
	switch {
	case errors.Is(err, users.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, users.ErrConflict):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}
```

## Escalation Guidance
- Escalate to `go-design-review` when no package clearly owns the repeated policy.
- Escalate to `api-contract-designer-spec` when the source of truth is an external REST contract.
- Escalate to `go-data-architect-spec` or `go-db-cache-spec` when the repeated policy reflects schema, query, transaction, or cache truth.
- Escalate to `go-domain-invariant-review` when the repeated policy encodes business state or transition rules.

## Source Anchors
- [Organizing a Go module](https://go.dev/doc/modules/layout): packages can be split by ownership and internal packages can protect non-public implementation.
- [Package names](https://go.dev/blog/package-names): package names help users and maintainers know what belongs in the package.
- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments): avoid meaningless package names like `util`, `common`, `misc`, `api`, `types`, and `interfaces`.
- Repository pattern: `go-design-review/references/source-of-truth-seam-drift.md`.
