# Branch Protection And PR Governance

## Behavior Change Thesis
When loaded for symptom "the delivery spec needs merge governance or protected-branch policy," this file makes the model choose enforceable GitHub branch-protection settings and drift guards instead of likely mistake "use branch protection" with vague review/check language or silent admin bypass.

## When To Load
Load for protected branch setup, rulesets, required reviews, CODEOWNERS, required status checks, admin/bypass rules, conversation resolution, or merge queue readiness.

## Local Source Of Truth
- `scripts/dev/configure-branch-protection.sh` configures strict required checks, PR review, code-owner review, stale-review dismissal, conversation resolution, force-push/deletion denial, and admin enforcement.
- `scripts/ci/required-guardrails-check.sh` keeps required branch-protection contexts aligned with `.github/workflows/ci.yml`.
- `.github/CODEOWNERS`, `.github/pull_request_template.md`, `SECURITY.md`, `CONTRIBUTING.md`, and `.github/dependabot.yml` are repository governance guardrails.

## Decision Rubric
- Protected default branch policy should require PRs, up-to-date required checks, at least one approving review, code-owner review, stale-review dismissal, resolved conversations, force-push denial, deletion denial, and admin enforcement.
- Required check contexts must be stable and match `.github/workflows/ci.yml`; `make guardrails-check` is the drift alarm for configured contexts and guardrail files.
- Merge queue adoption requires required-check workflows to include the `merge_group` trigger before the ruleset/branch protection requires those checks for queued merges.
- CODEOWNERS placeholders block code-owner review claims; replacement with real owners is a prerequisite for claiming owner review enforcement.
- If GitHub rulesets are introduced, state whether they layer with or replace classic branch protection and document the most restrictive effective rule.
- Any bypass actor needs an exception record with owner, scope, expiry, compensating controls, and audit evidence. Default is no silent bypass.

## Imitate
- "Run `make guardrails-check`; then apply `make gh-protect BRANCH=main`; record exported GitHub settings proving strict checks, admin enforcement, stale-review dismissal, code-owner review, and resolved conversations." Copy the command-plus-export proof shape.
- "If `openapi-breaking` becomes a required compatibility gate, add it to GitHub required checks and the guardrails checker in the same change." Copy the two-surface drift-control rule.
- "If merge queue is enabled, add `merge_group` to required-check workflows before enforcing queued merges." Copy the trigger-policy coupling rule.
- "Rulesets may be adopted only with an effective-policy note that says which classic branch-protection controls remain active." Copy the anti-ambiguity rule.

## Reject
- "Require status checks." This is not enough unless the exact contexts are named and stable.
- "Admins can merge emergency fixes directly." This is silent bypass unless there is an exception record and reconciliation path.
- "CODEOWNERS review is required" while `.github/CODEOWNERS` still contains `@your-org/your-team`. This claims enforcement that the repo setup script rejects.

## Agent Traps
- Do not decide who owns code here; only turn settled ownership into enforcement.
- Do not require duplicated or unstable job names as status checks.
- Do not treat GitHub UI screenshots alone as durable policy if a script or guardrail can keep it repo-reviewable.

## Validation Shape
Use `make guardrails-check`, `make gh-protect BRANCH=main` or equivalent API/ruleset export, and PR evidence that required checks pass on the merge SHA with code-owner approval and resolved conversations when required.

## Hand-Off Boundary
Do not decide code ownership, API ownership, or security review ownership here. This file only turns settled ownership policy into merge enforcement.
