---
name: go-verification-before-completion
description: "Verify correctness or readiness claims with fresh command evidence matched to claim scope before reporting success."
---

# Go Verification Before Completion

## Purpose
Prevent false-positive completion claims by requiring fresh verification evidence that matches the scope of the claim.

## Outcome-First Operating Rules
- Start by naming the skill-specific outcome, success criteria, constraints, available evidence, and stop rule.
- Treat workflow steps as decision rules, not a ritual checklist. Follow exact order only when this skill or the repository contract makes the sequence an invariant.
- Use the minimum context, references, tools, and validation loops that can change the deliverable; stop expanding when the quality bar is met.
- Before acting, resolve prerequisite discovery, lookup, or artifact reads that the outcome depends on; parallelize only independent evidence gathering and synthesize before the next decision.
- Prefer bounded assumptions and local evidence over broad questioning; ask only when a missing fact would change correctness, ownership, safety, or scope.
- When evidence is missing or conflicting, retry once with a targeted strategy or label the assumption, blocker, or reopen target instead of treating absence as proof.
- Finish only when the requested deliverable is complete in the required shape and verification or a clearly named blocker/residual risk is recorded.

## Scope
- verify statements such as "fixed", "tests pass", "lint clean", "build succeeds", or "ready for handoff"
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
References are compact rubrics and example banks, not exhaustive checklists or documentation dumps. Load at most one reference by default. Load more only when the claim clearly spans independent decision pressures, such as delegated work plus generated API drift plus a failed proof command.

Before loading, name the behavior-change thesis you need: "When loaded for symptom X, this file makes me choose Y instead of likely mistake Z." If no reference has a concrete thesis for the symptom, stay in `SKILL.md` and inspect live repo files such as `Makefile` or `docs/build-test-and-development-commands.md` as needed.

| Reference | Symptom | Behavior change |
|---|---|---|
| `references/claim-to-proof-mapping.md` | ambiguous "fixed", "green", "ready", test, lint, build, race, package, or repo claim | choose the narrowest sufficient proof for the exact claim instead of either over-running unrelated checks or generalizing a focused pass to repo readiness |
| `references/generated-api-and-migration-verification.md` | OpenAPI, generated API, mocks, stringer, sqlc, query, or migration surface changed | add drift or migration rehearsal proof instead of treating compile/tests as enough or accepting skipped migration output as validation |
| `references/delegated-work-verification.md` | another agent, tool, CI snippet, or prior session says work is done | rebind the delegated claim to current workspace evidence instead of treating a report or stale log as proof |
| `references/failure-and-gap-reporting.md` | proof failed, skipped, was missing, was cached unexpectedly, or is weaker than the requested claim | report "not verified" or "partially verified" with the blocking signal and next verification action instead of writing a positive closeout |

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
- "Fresh" means executed in the current iteration against the current workspace state.
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
