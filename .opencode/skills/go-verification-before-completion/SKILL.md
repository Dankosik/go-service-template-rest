---
name: go-verification-before-completion
description: "Require fresh command evidence before any completion/readiness claim in this Go spec-first workflow. Use when you are about to state that code is fixed, tests pass, checks are green, or a task/gate is ready. Skip when collecting context or drafting plans without correctness claims."
---

# Go Verification Before Completion

## Purpose
Enforce evidence-first completion signaling in this repository. Success means every positive correctness/readiness claim is backed by fresh command output for the same scope.

## Scope And Boundaries
In scope:
- verify claims such as "fixed", "tests pass", "lint clean", "ready for Gate G3/G4"
- map each claim to the smallest proving command set
- run commands, read output/exit status, and report factual outcome
- block optimistic completion language when evidence is missing
- surface verification gaps and next actions clearly

Out of scope:
- root-cause investigation process (use `go-systematic-debugging`)
- architecture/spec design decisions
- domain-specific code review findings (`*-review` skills)
- forcing full CI on every micro-claim when smaller proof is sufficient

## Hard Skills
### Verification Core Instructions

#### Mission
- Prevent false-positive completion claims.
- Keep status reporting objective, reproducible, and auditable.
- Align completion language with executed verification evidence.

#### Default Posture
- Evidence first, wording second.
- Fresh run required for every claim in current scope.
- Smallest sufficient command set, but never weaker than claim strength.
- If verification fails or is not run, report uncertainty explicitly.

#### Verification Gate Function Competency
Before any success/readiness claim:
1. Identify exact claim and affected scope.
2. Choose command(s) that directly prove the claim.
3. Run command(s) now (not from memory/old logs).
4. Read exit code and key failure counts.
5. Report result with evidence, or report failure/gap.

#### Claim-To-Proof Mapping Competency
Use this default mapping unless scope requires stricter checks:

| Claim | Minimum proof commands |
|---|---|
| "Targeted fix works" | reproducible failing command now passing (usually focused `go test -run ... -count=1`) |
| "Tests pass" | `make test` (or focused `go test` only if claim is explicitly scoped) |
| "Race-safe for changed concurrent path" | `make test-race` (or `go test -race ./...`) |
| "Lint clean" | `make lint` |
| "Build passes" | `make build` |
| "API contract/runtime checks pass" | `make openapi-check` |
| "Ready for Gate G3" | checks required by changed scope + explicit pass summary |
| "Ready for Gate G4" | reviewer-scope findings resolved + required gate checks for changed scope |

If local toolchain is unavailable, use Docker equivalents from `docs/build-test-and-development-commands.md`.

#### Freshness And Scope Competency
- "Fresh" means command executed in current iteration against current workspace state.
- Focused verification is allowed only when claim is explicitly focused.
- Broad readiness claims require broad verification coverage.
- Do not extrapolate from partial checks to repository-wide claims.

#### Spec-First Gate Alignment Competency
- Gate claims must reflect `docs/spec-first-workflow.md` semantics:
  - G3: implementation + required tests/checks + no unresolved clarification blockers.
  - G4: critical/high findings resolved and required quality gates green.
- Do not claim gate readiness from tests alone when other gate conditions are unmet.

#### Delegation And Trust Competency
- Agent/subagent report is not verification evidence by itself.
- Validate delegated work with commands and actual diff state before claiming completion.

#### Failure And Gap Reporting Competency
When proof fails or is missing:
- state exact failing command,
- include key error signal,
- avoid success language,
- provide the next concrete verification action.

#### Review Blockers For This Skill
- completion/readiness claim without fresh proving command
- repository-wide claim backed only by targeted check
- gate readiness claim missing required non-test conditions
- trusting prior runs/agent reports instead of current evidence
- ambiguous wording that implies success despite missing proof

## Working Rules
1. Capture the exact claim sentence before replying.
2. Bind claim to explicit scope (test, package, feature, gate, repo).
3. Select minimum proving command set for that scope.
4. Execute commands in current workspace.
5. Parse exit status and key pass/fail counters.
6. If all proofs pass, issue claim with command evidence.
7. If any proof fails, report factual status and blocking signal.
8. If proof not run, explicitly state "not verified" and next step.
9. Keep claim wording proportional to proof scope.
10. Do not merge completion with speculation.

## Output Expectations
Use this section order:

```text
Claim
Scope
Verification Commands
Observed Result
Conclusion
Next Action
```

Rules:
- `Verification Commands` must list commands actually executed.
- `Observed Result` must include pass/fail status per command.
- `Conclusion` may be positive only when evidence supports claim.
- If not verified, `Conclusion` must state `not verified`.

## Definition Of Done
- Every positive correctness/readiness statement is backed by fresh command evidence.
- Scope of claim matches scope of verification.
- Failed/missing checks are reported as blockers, not softened language.
- Next action is explicit when verification is incomplete.

## Anti-Patterns
- "should pass" / "looks good" without command output
- using old CI/local runs as proof for current state
- reporting "ready" after only partial checks
- treating lint pass as build/test pass
- treating changed files/diff as proof of correctness

## Context Intake (Dynamic Loading)
Rule: load the smallest sufficient set of docs. Never bulk-load folders by default.
Stop condition: stop loading when claim scope, proof commands, and gate constraints are unambiguous.

Always load:
- `docs/build-test-and-development-commands.md`
- `docs/spec-first-workflow.md` (gate semantics only when gate claim is present)

Load by trigger:
- bugfix/correctness claim from defect work:
  - `skills/go-systematic-debugging/SKILL.md`
- test-implementation readiness claim:
  - `skills/go-qa-tester/SKILL.md`
- production-code readiness claim:
  - `skills/go-coder/SKILL.md`
- reviewer sign-off/gate claim:
  - corresponding `skills/*-review/SKILL.md`

Companion reference:
- `references/claim-proof-matrix.md`

Conflict resolution:
- More specific gate/skill requirement overrides generic mapping.

Unknowns:
- If you cannot determine required checks, mark `[assumption]` and state the minimal safe next verification command.
