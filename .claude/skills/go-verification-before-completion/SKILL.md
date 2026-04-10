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
- create new workflow/process/planning/design/temp artifacts to compensate for missing proof inputs

## Core Defaults
- Evidence first, wording second.
- Fresh run required for every positive claim in current scope.
- Use the smallest sufficient command set, but never weaker than the claim.
- If verification fails or was not run, say so explicitly.
- Verification is an artifact-consuming phase: consume existing `spec.md`, `plan.md`, optional `test-plan.md`, optional `rollout.md`, existing workflow-control artifacts, and fresh command output instead of authoring new process artifacts.
- If proof depends on a missing expected artifact or missing planning/design context, report the reopen target instead of inventing replacement artifacts during verification.

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
Use these defaults unless the scope requires something stricter:

| Claim | Minimum proof |
|---|---|
| Targeted fix works | the reproducible failing command now passes |
| Scoped package behavior is green | `go test ./path/to/pkg/...` |
| Repository tests pass | `make test` |
| Concurrency-safe for the changed path | `make test-race` or `go test -race ./...` |
| Lint clean | `make lint` |
| Build succeeds | `make build` |
| API contract/runtime checks green | `make openapi-check` |
| Migration safety checked | `make migration-validate` |
| Ready for handoff or review | scope-required tests plus the required quality checks for the changed surface |

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
Return verification work in this order:
- `Claim`
- `Scope`
- `Verification Commands`
- `Observed Result`
- `Conclusion`
- `Next Action`

`Conclusion` may be positive only when the evidence supports it. Otherwise state `not verified`.

## Escalate When
Escalate if:
- the required proving command is unclear
- the claim scope is broader than the available evidence
- delegated work has not been checked against current state
- failing commands block the claim and need remediation before any completion language
