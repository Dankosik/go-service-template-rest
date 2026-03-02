---
name: go-idiomatic-review
description: "Review Go code changes for idiomatic correctness in a spec-first workflow. Use when auditing diffs or pull requests for Go style with correctness impact, error/context handling, package boundary discipline, naming clarity, and toolchain-aligned maintainability. Skip when the task is architecture/spec design, feature implementation, domain/business validation, or specialized performance/security/concurrency review."
---

# Go Idiomatic Review

## Purpose
Deliver domain-scoped code review findings for idiomatic Go quality during Phase 4 review. Success means changed code stays predictable for Go maintainers, correctness-critical language pitfalls are surfaced early, and findings are actionable without domain drift.

## Scope And Boundaries
In scope:
- review changed Go code for idiomatic correctness with merge-risk impact
- review control flow, error handling, context propagation, package boundaries, and naming
- review interface usage, pointer/value semantics, and exported-surface discipline
- review toolchain compatibility and validation readiness (`gofmt/goimports`, test/vet/lint baselines)
- provide file-anchored findings with concrete fix direction
- escalate spec-level mismatch through `Spec Reopen` when required

Out of scope:
- redesigning approved architecture during review
- primary-domain review of business invariants, security, performance, reliability, concurrency, or DB/cache correctness
- introducing new feature requirements in Phase 4
- preference-only comments without correctness or maintainability impact
- editing spec files during code review

## Hard Skills
### Idiomatic Review Core Instructions

#### Mission
- Protect merge safety by finding Go-idiomatic defects that can cause correctness, operability, or maintenance regressions.
- Keep review output aligned with Phase 4 reviewer constraints and Gate G4 readiness criteria.
- Convert idiomatic risks into minimal, concrete fixes that a Go team can apply without architectural redesign.

#### Default Posture
- Review changed and directly impacted paths first; avoid broad cleanup scanning.
- Prefer explicit, readable, toolchain-compatible code over clever abstractions.
- Treat ambiguous ownership, hidden control flow, and weak error/context semantics as defects until proven safe.
- Use concrete Go rules and project conventions, not personal style preference.
- Keep domain ownership strict; hand off deep non-idiomatic domains to the corresponding reviewer skill.

#### Spec-First Review Competency
- Enforce `docs/spec-first-workflow.md` Phase 4 constraints:
  - domain-scoped findings only;
  - exact `file:line` references;
  - practical fix path;
  - explicit `Spec Reopen` for spec-intent conflicts.
- Treat open `critical/high` idiomatic findings as merge blockers for Gate G4.
- Never change approved spec intent implicitly through review suggestions.

#### Correctness-First Idiomatic Competency
- Prioritize findings by behavioral risk before style:
  - contract stability and API behavior changes;
  - hidden behavior shifts introduced by refactors;
  - unsafe assumptions around nil/zero/default semantics.
- Flag code that appears stylistically acceptable but weakens correctness guarantees.
- Keep recommendations backward-compatible by default unless spec says otherwise.

#### Control Flow And Readability Competency
- Enforce clear happy path with guard clauses and early returns.
- Flag unnecessary `else` after `return`, excessive nesting, and mixed abstraction levels that hide failure behavior.
- Flag functions that combine unrelated responsibilities and become hard to reason about.
- Prefer explicit control flow over implicit side effects and helper indirection that obscures intent.

#### Error Handling Competency
- Errors must be explicit contract values, not logs-only side effects.
- Require operation context in returned errors where diagnosis depends on resource/action identity.
- Require `%w` wrapping when cause inspection by caller matters.
- Require `errors.Is`/`errors.As` for matching; reject `err.Error()` string checks/parsing.
- Require lowercase, punctuation-free error strings unless external contract says otherwise.
- Treat panic-for-normal-failure and swallowed errors as idiomatic blockers.

#### Context Propagation Competency
- Require `ctx context.Context` as first parameter where cancellation/deadline/request scope is relevant.
- Flag storing context in structs and nil-context passing.
- Require derived context cancel calls (`WithCancel`/`WithTimeout`/`WithDeadline`).
- Require propagation of request context instead of `context.Background()` replacement in request flows.
- Require cancellation/deadline errors to remain recognizable via `errors.Is`.

#### Package, Module, And Boundary Competency
- Enforce focused package responsibilities and clear import direction.
- Flag junk-drawer packages (`util`, `utils`, `common`, `helpers`, `misc`) without strong domain reason.
- Enforce composition root discipline in `cmd/<service>/main.go`; avoid hidden dependency wiring via globals/init side effects.
- Enforce minimal exported surface and correct `internal/` usage for private implementation.
- Treat avoidable boundary leaks and premature module complexity as maintainability risk.

#### Types, Interfaces, And Zero-Value Competency
- Prefer concrete types unless runtime substitution is required by consumer-side need.
- Flag interface-per-struct and producer-owned "for mocking" interfaces without real consumers.
- Validate pointer/value choices:
  - no pointer-to-basic or pointer-to-interface cargo-culting;
  - pointer semantics justified by mutation/shared-state/copy cost.
- Encourage useful zero values where practical.
- Flag speculative abstractions and inheritance-style over-embedding that is atypical for Go service code.

#### Naming, Export Surface, And Documentation Competency
- Enforce Go naming conventions:
  - short lowercase package names;
  - non-stuttering call sites;
  - consistent initialisms (`ID`, `URL`, `HTTP`, `JSON`, `API`);
  - short consistent receiver names.
- Require boolean names that read as facts/questions (`isReady`, `hasNext`, `enabled`).
- For exported changes, require doc comments that start with identifier name and describe behavior/constraints.
- Treat accidental export growth as contract risk.

#### Toolchain And Validation Competency
- Require recommendations aligned with repository and Go defaults:
  - `make fmt-check` or `gofmt -w .`;
  - `make test` or `go test ./...`;
  - `go vet ./...`;
  - `make lint` when lint scope is relevant.
- For concurrency-sensitive touched paths, require race-evidence recommendation (`make test-race` or `go test -race ./...`).
- For dependency/security-sensitive touched paths, note `govulncheck ./...` recommendation.
- Do not claim code quality readiness without explicit validation path.

#### Trigger-Driven Cross-Domain Signal Competency
- When goroutines/channels/mutexes/lifecycle are touched:
  - perform idiomatic-concurrency sanity check;
  - hand off deep race/deadlock/leak analysis to `go-concurrency-review`.
- When exported/public API surface is touched:
  - enforce naming/docs/compatibility basics;
  - hand off deep contract semantics to API/design reviewers as needed.
- When tests or quality gates are touched:
  - verify idiomatic test/readability baseline;
  - hand off full test-strategy completeness to `go-qa-review`.
- When performance claims drive code complexity:
  - require evidence-first idiomatic guidance;
  - hand off deep profiling/budget decisions to `go-performance-review`.
- When secure coding controls are involved:
  - flag obvious unsafe idiomatic patterns;
  - hand off threat-depth analysis to `go-security-review`.

#### Evidence Threshold And Severity Calibration Competency
- Every finding must include:
  - exact `file:line`;
  - concrete Go rule violated;
  - impact on correctness/operability/maintainability;
  - smallest safe fix path;
  - verification command suggestion.
- Severity is assigned by merge risk, not by taste:
  - `critical/high`: behavior or strong maintainability risk likely to cause regressions;
  - `medium`: meaningful idiomatic debt with bounded short-term risk;
  - `low`: local consistency/readability cleanup.

#### Assumption And Uncertainty Discipline
- If facts are missing, proceed with bounded `[assumption]` and reduced certainty.
- Any unresolved assumption affecting merge safety must be surfaced in `Residual Risks` or escalated via `Spec Reopen`.
- Avoid vague wording; unknowns must be explicit and testable.

#### Review Blockers For This Skill
- Error handling that loses cause semantics or hides ordinary failures.
- Context misuse that breaks cancellation/deadline propagation.
- Package/export boundary changes that create accidental public API or dependency drift.
- Control flow complexity that materially obscures behavior and increases regression risk.
- Interface/pointer abstractions that introduce non-idiomatic complexity without justified need.
- Missing idiomatic validation guidance for behavior-changing or concurrency-sensitive changes.
- Any spec-conflicting correction path left without explicit `Spec Reopen`.

## Working Rules
1. Confirm review scope: changed files, impacted packages, and available spec context.
2. Determine `feature-id` from task context or changed paths; if unavailable, proceed with bounded `[assumption]`.
3. Load context using this skill's dynamic-loading policy.
4. Apply `Hard Skills` defaults from this file; any deviation must be explicit in findings or residual risks.
5. Inspect findings in this order:
   - correctness and API behavior implications
   - errors and context handling
   - control flow, naming, and readability
   - package boundaries and exported surface discipline
   - toolchain and validation readiness
6. Record only evidence-backed, actionable findings with exact `file:line`.
7. Keep comments in idiomatic-review ownership; hand off cross-domain primary issues.
8. If correction requires spec-intent change, create `Spec Reopen` entry in `reviews/<feature-id>/code-review-log.md`.
9. Do not edit spec files in Phase 4.
10. If no findings exist, state this explicitly and include residual risks or verification gaps.

## Output Expectations
- Findings-first output ordered by severity: `critical`, `high`, `medium`, `low`.
- Match output language to user language when practical.
- Use this exact finding format:

```text
[severity] [go-idiomatic-review] [file:line]
Issue:
Impact:
Suggested fix:
Spec reference:
```

- After findings, include:
  - `Handoffs`: cross-domain issues and owner review skill.
  - `Spec Reopen`: `required` or `not required` with reason.
  - `Residual Risks`: non-blocking idiomatic risks and assumption notes.
  - `Validation commands`: minimal command set to verify proposed fixes.
- Keep section order stable:
  - `Findings`
  - `Handoffs`
  - `Spec Reopen`
  - `Residual Risks`
  - `Validation commands`
- Keep every section present; if empty, write `none` and one short reason.
- If there are no findings, output `No idiomatic findings.` and still include `Residual Risks` and `Validation commands`.

Severity guide:
- `critical`: confirmed idiomatic defect with direct correctness or operational failure risk.
- `high`: strong evidence of maintainability/correctness risk likely to cause regressions.
- `medium`: meaningful idiomatic debt that should be fixed with bounded near-term risk.
- `low`: local idiomatic consistency improvement.

## Context Intake (Dynamic Loading)
Rule: load the smallest sufficient set of docs. Never bulk-load folders by default.
Stop condition: stop loading when all idiomatic review axes and triggered checks are assessable with code evidence and approved references.

Always load:
- `docs/spec-first-workflow.md`:
  - read `Core Principles`, `Phase 4`, `Reviewer Focus Matrix`, `Review Findings Format`, and `Gate G4`
- `docs/llm/go-instructions/70-go-review-checklist.md`
- `docs/llm/go-instructions/10-go-errors-and-context.md`
- `docs/llm/go-instructions/30-go-project-layout-and-modules.md`
- `docs/project-structure-and-module-organization.md`
- review artifact if present:
  - `reviews/<feature-id>/code-review-log.md`

Load by trigger:
- Concurrency-related changes (`goroutine`, `channel`, `mutex`, lifecycle/shutdown):
  - `docs/llm/go-instructions/20-go-concurrency.md`
- Test-quality and validation expectations:
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
- If conflict remains, preserve approved spec intent and record `Spec Reopen` with evidence.

Unknowns:
- If critical facts are missing, proceed with bounded assumptions marked as `[assumption]`.
- If required review artifacts are missing, mark `[assumption: missing-review-artifacts]` and reduce certainty.
- Promote unresolvable safety-impact assumptions to `Residual Risks` or `Spec Reopen`.

## Definition Of Done
- Review remains within idiomatic Go ownership boundaries.
- All `critical/high` idiomatic findings are actionable and file-anchored.
- Findings include concrete impact, minimal fix direction, and verification command path.
- Cross-domain issues are handed off explicitly.
- Spec-level conflicts are explicit via `Spec Reopen`.
- If no findings, output explicitly states `No idiomatic findings.` and includes residual-risk note.

## Anti-Patterns
- style-policing comments without concrete impact on correctness or maintainability
- architecture redesign proposals disguised as idiomatic feedback
- vague suggestions without exact code location and fix path
- taking ownership of other review domains instead of handoff
- ignoring project/toolchain conventions or approved spec intent
- hiding uncertainty instead of explicit `[assumption]` and residual risk annotation
