# Branch Protection And PR Governance

## When To Load
Load this when specifying merge governance, branch protection, rulesets, required reviews, CODEOWNERS, status checks, admin bypass, conversation resolution, or merge queue readiness.

## Local Source Of Truth
- `scripts/dev/configure-branch-protection.sh` configures required status checks, strict up-to-date branches, admin enforcement, code-owner review, stale-review dismissal, conversation resolution, and force-push/deletion denial.
- `scripts/ci/required-guardrails-check.sh` verifies required guardrail files and keeps branch-protection contexts aligned with `.github/workflows/ci.yml`.
- `.github/CODEOWNERS`, `.github/pull_request_template.md`, `SECURITY.md`, `CONTRIBUTING.md`, and `.github/dependabot.yml` are repository governance guardrails.

## Enforceable Policy Examples
- Protected default branch requires pull requests, up-to-date status checks, at least one approving review, code-owner review, stale-review dismissal, resolved conversations, force-push denial, deletion denial, and admin enforcement.
- Required checks must include every merge-blocking CI job exposed by `.github/workflows/ci.yml`; the guardrails check must fail if the configured context list and CI job list drift.
- CODEOWNERS placeholders block branch-protection setup until replaced with real owners.
- If GitHub rulesets are introduced, record whether they layer with or replace classic branch protection, and require the most restrictive effective rule to be documented as the merge policy.
- Any bypass actor must be named in an exception record with scope, owner, expiry, and audit trail; default policy is no silent bypass.

## Non-Enforceable Anti-Patterns
- "Use branch protection" without naming which protections are required.
- Requiring status checks whose job names are duplicated across workflows or unstable across renames.
- Depending on admin discipline instead of enabling admin enforcement or no-bypass controls.
- Keeping CODEOWNERS as a placeholder while treating code-owner review as active.
- Allowing direct pushes to protected branches because release fixes are "urgent" without an exception record and follow-up reconciliation.

## Evidence Artifacts
- Output from `make guardrails-check`.
- Output from `make gh-protect BRANCH=main` or the equivalent GitHub API/ruleset configuration change.
- Screenshot or exported API response showing required status checks, strict mode, review settings, conversation resolution, and bypass settings.
- PR evidence: required checks pass on the merge SHA, code-owner approval exists when touched paths require it, and conversations are resolved.

## Hand-Off Boundary
Do not decide code ownership, API ownership, or security review ownership here. This file only turns settled ownership policy into merge enforcement.

## Exa Source Links
- GitHub Docs: [About protected branches](https://docs.github.com/repositories/configuring-branches-and-merges-in-your-repository/managing-protected-branches/about-protected-branches)
- GitHub Docs: [Managing a branch protection rule](https://docs.github.com/en/github/administering-a-repository/enabling-required-status-checks)
- GitHub Docs: [About rulesets](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/about-rulesets)
- GitHub REST Docs: [REST API endpoints for rules](https://docs.github.com/en/rest/orgs/rules)
- OpenSSF: [scorecard-action](https://github.com/ossf/scorecard-action)

