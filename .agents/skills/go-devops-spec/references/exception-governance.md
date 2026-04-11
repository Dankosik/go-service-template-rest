# Exception Governance

## When To Load
Load this when a delivery spec allows temporary bypasses, informational-only downgrades, release-risk acceptance, vulnerability suppressions, branch-protection bypass, scan exceptions, migration rollback exceptions, or manual release overrides.

## Local Source Of Truth
- `scripts/dev/configure-branch-protection.sh` defaults to admin enforcement, strict checks, code-owner review, stale-review dismissal, conversation resolution, force-push denial, and deletion denial.
- `scripts/ci/required-guardrails-check.sh` fails when guardrail files or branch-protection contexts drift.
- `.github/workflows/ci.yml`, `.github/workflows/nightly.yml`, and `.github/workflows/cd.yml` are the evidence surfaces for gate results.
- `SECURITY.md`, `CONTRIBUTING.md`, `.github/CODEOWNERS`, and `.github/pull_request_template.md` are required governance artifacts under guardrails.

## Enforceable Policy Examples
- Every exception must name owner, approver, scope, affected gate, reason, compensating controls, expiry date or expiry event, follow-up issue, and reopen condition.
- No blocking gate may be downgraded to informational by editing workflow YAML alone; the policy change must update the spec/release plan and guardrail expectations in the same review.
- Branch-protection bypass is disallowed by default. If a ruleset or branch protection exception exists, it must be visible in exported GitHub settings or API evidence and tied to an audit event.
- Vulnerability or scan suppressions must include finding identifier, affected artifact, version or digest, severity, rationale, expiry, and rescan proof.
- Migration rollback exceptions must record why reversal cannot be rehearsed, whether the release is forward-only, and what restore or compensating proof gates publish.

## Non-Enforceable Anti-Patterns
- "Temporary waiver" with no expiry or owner.
- "Accept risk" without naming the exact gate and artifact affected.
- Bypass lists that accumulate actors without periodic review.
- Suppression comments in CI config without linked issue, expiry, or rescan condition.
- Allowing manual releases after failed CI because an operator says the change is safe.
- Leaving a broken required check unrequired instead of fixing, renaming, or formally replacing it.

## Evidence Artifacts
- Exception record in the task spec, release issue, or PR with owner, approver, expiry, compensating control, and follow-up.
- GitHub branch protection or ruleset export showing bypass actors and enforcement status.
- CI/CD run proving the compensating gate passed.
- Linked issue or scheduled follow-up for expiry review.
- Rescan, rerun, rollback rehearsal, or post-release verification log that closes the exception.

## Hand-Off Boundary
Do not accept product, security, data-loss, API-breaking, or distributed-consistency risk inside this delivery reference. Record delivery impact and require the owning specialist/spec to accept the underlying risk.

## Exa Source Links
- GitHub Docs: [About protected branches](https://docs.github.com/repositories/configuring-branches-and-merges-in-your-repository/managing-protected-branches/about-protected-branches)
- GitHub Docs: [Managing a branch protection rule](https://docs.github.com/en/github/administering-a-repository/enabling-required-status-checks)
- GitHub Docs: [About rulesets](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/about-rulesets)
- GitHub REST Docs: [REST API endpoints for rules](https://docs.github.com/en/rest/orgs/rules)
- GitHub Docs: [Script injections](https://docs.github.com/en/actions/concepts/security/script-injections)

