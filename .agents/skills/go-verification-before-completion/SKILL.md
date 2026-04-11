---
name: go-verification-before-completion
description: "Verify correctness or readiness claims with fresh command evidence matched to claim scope before reporting success."
---

# Go Verification Before Completion

## Purpose
Prevent false-positive completion claims by requiring fresh verification evidence that matches the scope of the claim.

## Scope
- verify statements such as “fixed”, “tests pass”, “lint clean”, “build succeeds”, or “ready for handoff”
- map each claim to the smallest command set that honestly proves it
- run commands, inspect results, and report factual outcomes
- block optimistic completion language when proof is missing or weaker than the claim

## Boundaries
Do not:
- turn this into root-cause investigation or broad debugging
- treat design or code review findings as verified just because an agent reported them
- force full-repository verification for every narrow claim when smaller proof is sufficient
- soften missing or failing proof with optimistic wording
- create or repair workflow, research, specification, design, planning, or temp artifacts to compensate for missing proof inputs

## Specialist Stance
- Evidence first, wording second.
- Fresh run required for every positive claim in current scope.
- Use the smallest sufficient command set, but never weaker than the claim.
- If verification fails or was not run, say so explicitly.
- Consume existing task artifacts and fresh command output when they exist; do not author new process artifacts from this skill.
- If proof depends on missing expected context, report the proof gap and the smallest unblock action instead of inventing replacement context.
- If command output shows cached or skipped work, keep the conclusion narrower than an executed green run unless the cache or skip semantics are sufficient for the claim.

## Lazy References
Load only the reference needed for the claim shape:

| Reference | Load when |
|---|---|
| `references/claim-to-proof-mapping.md` | choosing the proof set for ambiguous completion, readiness, test, lint, build, or handoff claims |
| `references/focused-vs-repository-wide-verification.md` | deciding whether focused package proof is enough or a repository-wide claim needs broader checks |
| `references/go-test-build-race-and-lint-evidence.md` | matching Go test, build, race detector, vet, and lint claims to command evidence |
| `references/generated-api-and-migration-verification.md` | generated API, OpenAPI, sqlc, migration, or contract drift changed |
| `references/delegated-work-verification.md` | another agent, tool, CI snippet, or prior session claims work is done |
| `references/failure-and-gap-reporting.md` | any required proof failed, was skipped, was not run, or is weaker than the requested claim |

## Expertise

### Verification Gate Function
Before any success or readiness claim:
1. identify the exact claim
2. bind it to explicit scope
3. choose commands that directly prove that scope
4. run them now
5. inspect exit status and key pass/fail signals
6. report evidence or report the gap

### Claim-To-Proof Mapping
Use these defaults unless the claim scope requires something stricter:
- targeted fix: rerun the exact failing command or the narrowest reproducer that covers the fixed path
- scoped package behavior: run the relevant `go test` package pattern, with `-run` and `-count=1` when a specific test or uncached execution matters
- repository test claim: run the repository test target or an explicit repository-wide `go test` pattern
- race-safety claim: run race-detector coverage for the changed concurrent path
- lint, build, generated API, and migration claims: run the repository target that owns that proof
- readiness claim: combine the checks required by the changed surface; never use one green check as proof for unrelated surfaces

### Freshness And Scope
- “Fresh” means executed in the current iteration against the current workspace state.
- Focused verification is valid only for a focused claim.
- Broad claims require broad proof.
- Do not extrapolate from targeted checks to repository-wide success.

### Delegation And Trust
- An agent or subagent report is not proof by itself.
- Validate delegated work against current workspace state and fresh command output before claiming success.

### Failure And Gap Reporting
When proof fails or is missing:
- state the failing or missing command explicitly
- include the key error signal or missing proof gap
- avoid success language
- give the next concrete verification action

## Verification Quality Bar
A correct verification report:
- states the claim explicitly
- binds it to the right scope
- lists commands actually executed
- reports pass/fail honestly
- keeps conclusion wording proportional to proof strength

## Deliverable Shape
Return a compact verification note with:
- claim and scope
- commands actually executed
- observed pass/fail signal
- conclusion proportional to the evidence
- next action when not verified

The conclusion may be positive only when the evidence supports it. Otherwise state `not verified`.

## Escalate When
Escalate if:
- the required proving command is unclear
- the claim scope is broader than the available evidence
- delegated work has not been checked against current state
- failing commands block the claim and need remediation before any completion language
