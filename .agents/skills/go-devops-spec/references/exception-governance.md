# Exception Governance

## Behavior Change Thesis
When loaded for symptom "a gate, protection, scan, or rollout rule needs a waiver," this file makes the model choose a bounded exception record with owner, expiry, compensating proof, and reopen condition instead of likely mistake silently downgrading a blocking control to informational.

## When To Load
Load only when the prompt includes temporary bypasses, informational-only downgrades, release-risk acceptance, vulnerability suppressions, branch-protection bypass, scan exceptions, migration rollback exceptions, or manual release overrides.

## Role
This is a challenge and smell-triage reference, not primary design guidance. Prefer a narrower positive reference first when the task is mainly CI, branch protection, migrations, containers, or supply-chain trust; load this when an exception path itself is the decision pressure.

## Local Source Of Truth
- `scripts/dev/configure-branch-protection.sh` defaults to admin enforcement, strict checks, code-owner review, stale-review dismissal, conversation resolution, force-push denial, and deletion denial.
- `scripts/ci/required-guardrails-check.sh` fails when guardrail files or branch-protection contexts drift.
- `.github/workflows/ci.yml`, `.github/workflows/nightly.yml`, and `.github/workflows/cd.yml` are the evidence surfaces for gate results.
- `SECURITY.md`, `CONTRIBUTING.md`, `.github/CODEOWNERS`, and `.github/pull_request_template.md` are required governance artifacts under guardrails.

## Decision Rubric
- Every exception must name owner, approver, scope, affected gate/artifact, reason, compensating controls, expiry date or event, follow-up issue, and reopen condition.
- No blocking gate may be downgraded by editing workflow YAML alone; the policy/spec and guardrail expectations must change in the same review.
- Branch-protection bypass is disallowed by default. If a ruleset or branch-protection exception exists, it needs exported settings/API evidence and an audit event.
- Vulnerability or scan suppressions must include finding identifier, affected artifact, version or digest, severity, rationale, expiry, and rescan proof.
- Migration rollback exceptions must record why reversal cannot be rehearsed, whether the release is forward-only, and what restore or compensating proof gates publish.

## Imitate
- "Accept HIGH Trivy finding CVE-... for image digest ... until 2026-05-01; owner ..., approver ..., compensating control ..., rescan command ..., reopen if fixed base image releases earlier." Copy the bounded scan-exception shape.
- "Emergency branch-protection bypass requires named actor, scope-limited branch/ruleset change, audit link, follow-up issue, and post-merge reconciliation gate." Copy the bypass accountability shape.
- "Forward-only migration exception records why down rehearsal is impossible and which restore drill or backup proof replaces it." Copy the proof-substitution pattern.

## Reject
- "Temporary waiver." This lacks owner, expiry, affected gate, and proof.
- "Accept risk for this release." This lacks the artifact and reopen condition.
- "Comment out the failing check until the release is done." This hides policy drift and bypasses guardrails.
- "Operator says it is safe." This is not delivery evidence.

## Agent Traps
- Do not accept product, security, data-loss, API-breaking, or distributed-consistency risk in this delivery reference. Record delivery impact and require the owning specialist/spec to accept the underlying risk.
- Do not let bypass lists accumulate without review/expiry.
- Do not confuse a compensating control with a promise to fix later; compensating proof must be observable before release.

## Validation Shape
Use exception records in the task spec, release issue, or PR; exported branch protection/ruleset settings; CI/CD run proving compensating gates; linked expiry follow-up; and rescan, rerun, rollback rehearsal, or post-release verification logs that close the exception.

## Hand-Off Boundary
Do not accept product, security, data-loss, API-breaking, or distributed-consistency risk here. Record delivery impact and require the owning specialist/spec to accept the underlying risk.
