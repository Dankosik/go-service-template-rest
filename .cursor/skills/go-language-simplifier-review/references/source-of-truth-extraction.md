# Source-Of-Truth Extraction

Behavior Change Thesis: When loaded for repeated same-package policy or vague helper buckets, this file makes the model choose a seam-named local owner instead of likely mistake of either tolerating drift-prone copies or extracting to global `common` code.

## When To Load
Load this when the same package repeats stable normalization, mapping, validation, classification, label shaping, section reading, or error mapping logic across files, or when a diff moves such policy into a vague helper bucket.

Use this for local simplification review only. If no package clearly owns the policy, escalate to design rather than inventing ownership.

## Decision Rubric
- Under-extraction is finding-worthy when repeated stable policy is likely to drift and a same-package owner would reduce future edits.
- Over-extraction is finding-worthy when the new helper erases ownership, needs modes or callbacks, or forces callers to know hidden policy.
- Prefer same-package, policy-named helpers over global `util`, `common`, `shared`, `helpers`, `types`, or `interfaces` packages.
- Do not extract orchestration order just to remove lines. Request lifecycle, startup, transaction, and response phases often read better locally.
- Keep response writing, side effects, and policy selection separate unless one local owner is already stable.

## Imitate
Finding shape to copy when repeated local policy needs one owner:

```text
[medium] [go-language-simplifier-review] internal/infra/http/problems.go:66
Issue: Three handlers now repeat the same domain-error to problem-status classification in file-local switches.
Impact: Adding a new error class requires synchronized edits across handlers, so the apparent local clarity can drift into endpoint-specific behavior.
Suggested fix: Extract a same-package helper such as `classifyProblem` and keep response writing local to each handler.
Reference: references/source-of-truth-extraction.md
```

Copy the move: separate the stable classification source of truth from caller-specific response work.

Finding shape to copy when extraction moves policy to a vague bucket:

```text
[high] [go-language-simplifier-review] internal/common/normalize.go:12
Issue: Email canonicalization moved from the users package into `internal/common` even though the validation and error contract are user-domain policy.
Impact: Other packages can reuse a narrower normalization rule without the user validation contract, creating two source-of-truth surfaces for the same input.
Suggested fix: Keep the helper in the users package with a policy name such as `canonicalEmail`, or create a narrower owner only if the approved design names one.
Reference: references/source-of-truth-extraction.md
```

Copy the move: name the lost owner and the specific contract that can be reused incorrectly.

## Reject
Reject this global extraction:

```go
package common

func Normalize(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
```

It hides who owns the normalization and which error contract travels with it.

Prefer a package-owned rule:

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

Reject mode-heavy source-of-truth helpers:

```go
func classify(err error, transport string, exposeDetails bool) int {
	// many unrelated cases
}
```

Prefer one owner per stable classification rule:

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

## Agent Traps
- Do not treat every duplicate shape as source-of-truth drift; distinct statuses, errors, side effects, or lifecycles can justify local repetition.
- Do not extract one-off logic without a stable second use.
- Do not flag intentionally local test setup that keeps tests independent and readable.
- Do not repair ownership drift by creating a broader ownership problem.

## Validation Shape
When recommending extraction, ask for proof that all former copies now call the same package-local owner and that caller-specific behavior remains local. When rejecting an extraction, ask for proof that the policy returns to its owner without changing public behavior.
