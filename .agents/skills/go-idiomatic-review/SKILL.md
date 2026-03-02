---
name: go-idiomatic-review
description: "Review Go code changes for idiomatic correctness in a spec-first workflow. Use when auditing diffs or pull requests for Go style with correctness impact, error/context handling, package boundary discipline, naming clarity, and toolchain-aligned maintainability. Skip when the task is architecture/spec design, feature implementation, domain/business validation, or specialized performance/security/concurrency review."
---

# Go Idiomatic Review

## Purpose
Deliver domain-scoped, actionable code review findings for idiomatic Go quality during Phase 4 review. Success means the review surfaces high-impact Go correctness and maintainability issues without drifting into unrelated domains.

## Scope And Boundaries
In scope:
- review Go diffs and touched files for idiomatic control flow and readability
- review error handling and context propagation correctness
- review package and exported-surface discipline against project boundaries
- review naming, interface usage, pointer/value semantics, and zero-value usability
- report findings with explicit impact and concrete fix direction
- escalate spec mismatches through `Spec Reopen` instead of redesigning in review

Out of scope:
- redesigning approved architecture
- validating business invariants as the primary domain
- deep performance, concurrency, security, DB/cache, or reliability audits as primary ownership
- rewriting specs or introducing new feature requirements during review
- style-only comments without correctness or maintainability impact

## Working Rules
1. Confirm the task is code review and identify review scope: changed files, impacted packages, and relevant approved spec artifacts.
2. Load review context with the dynamic loading rules in this file and stop when idiomatic coverage is sufficient.
3. Inspect findings in this priority order:
   - correctness and API behavior implications
   - errors and context handling
   - package boundaries and exported API surface
   - naming and readability
   - toolchain hygiene (`gofmt/goimports`, test and vet readiness)
4. Record only actionable findings with severity and file reference; avoid preference-only comments.
5. When an issue belongs to another review domain, log the signal and handoff target instead of taking ownership.
6. If code conflicts with approved spec intent, create a `Spec Reopen` record in `reviews/<feature-id>/code-review-log.md`.
7. Do not edit spec files during code review.

## Finding Protocol
For each nontrivial issue, record `IDM-###` with:
1. Location: exact `file:line`.
2. Rule violated: concrete idiomatic Go expectation.
3. Impact: correctness, operability, or maintainability risk.
4. Suggested fix: minimal concrete change path.
5. Verification: commands or tests to prove the fix.
6. Handoff: target review skill if primary ownership is elsewhere.

## Output Expectations
- Findings-first output ordered by severity: `critical`, `high`, `medium`, `low`.
- Use workflow-compatible entry format in `reviews/<feature-id>/code-review-log.md`:
  - `[severity] [go-idiomatic-review] [file:line]`
  - `Issue:`
  - `Impact:`
  - `Suggested fix:`
  - `Spec reference:`
- Keep findings concrete and testable; each blocking issue must explain why it is risky in Go terms.
- If no findings exist, state this explicitly and note residual risks or testing gaps.
- Return output in this stable section order:
  - `Findings`: idiomatic findings grouped by severity.
  - `Handoffs`: cross-domain issues with target review skill and handoff reason.
  - `Validation commands`: minimal command set to verify proposed fixes.
- Keep each section present even when empty:
  - if empty, write `none` and one short reason.

## Context Intake (Dynamic Loading)
Rule: load the smallest sufficient set of docs. Never bulk-load folders by default.
Stop condition: stop loading when context is sufficient to classify each finding by severity, ownership, and fix path.

Always load:
- `docs/spec-first-workflow.md` (reviewer focus, findings format, readiness definitions)
- `docs/llm/go-instructions/70-go-review-checklist.md`
- `docs/llm/go-instructions/10-go-errors-and-context.md`
- `docs/llm/go-instructions/30-go-project-layout-and-modules.md`
- `docs/project-structure-and-module-organization.md`

Load by trigger:
- Concurrency-related changes (`goroutine`, `channel`, `mutex`, lifecycle/shutdown):
  - `docs/llm/go-instructions/20-go-concurrency.md`
- Test-quality and command expectations:
  - `docs/llm/go-instructions/40-go-testing-and-quality.md`
  - `docs/build-test-and-development-commands.md`
- Exported/public API surface changes:
  - `docs/llm/go-instructions/50-go-public-api-and-docs.md`
- Performance-sensitive hot-path changes:
  - `docs/llm/go-instructions/60-go-performance-and-profiling.md`
- Security-impacting idiomatic risks:
  - `docs/llm/security/10-secure-coding.md`

Conflict resolution:
- The more specific document is the decisive rule for that topic.
- If specificity is equal, prefer trigger-loaded documents over always-loaded documents.
- If conflict remains, preserve approved spec intent and log `Spec Reopen` with evidence.

Unknowns:
- If critical facts are missing, proceed with bounded assumptions marked as `[assumption]`.
- Promote unresolvable assumptions to explicit open review risks with required follow-up.

## Definition Of Done
- Review stays within idiomatic Go domain boundaries.
- All `critical/high` idiomatic findings are identified and actionable.
- Findings are mapped to exact file references and include concrete fix direction.
- Cross-domain issues are handed off explicitly to the correct review skill.
- No spec-level conflict is left implicit; `Spec Reopen` is raised when required.

## Anti-Patterns
- tie each comment to concrete impact on correctness, operability, or maintainability
- keep review ownership boundaries explicit and route cross-domain issues via handoff
- provide file-anchored fix guidance for every nontrivial finding
- preserve approved architecture and spec intent unless `Spec Reopen` is required
- align all recommendations with Go toolchain conventions and project rules
