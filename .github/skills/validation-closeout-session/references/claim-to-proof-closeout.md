# Claim-To-Proof Closeout

## Behavior Change Thesis
When loaded for an explicit closeout claim with uncertain proof scope, this file makes the model choose evidence proportional to the claim instead of treating a narrow green command or review summary as broad completion proof.

## When To Load
Load this after readiness is established and before running commands, when the claim is known but the smallest honest fresh proof set is still unclear.

## Decision Rubric
- Bind each command to a named claim and changed surface.
- Use scoped commands for scoped claims; reserve task-wide or repository-wide wording for proof that covers the approved task-wide or repository-wide obligations.
- Include generated-contract, migration, cache, integration, or rollout checks only when the approved artifacts make those surfaces part of the claim.
- Agent reports, old CI, and previous command output may guide command selection; they cannot prove a positive closeout claim.
- If the only available proof is narrower than the user-facing claim, narrow the claim or reopen earlier work.

## Imitate

```markdown
Claim: `phase complete` for the tenant export API handler changes only.
Scope: handler validation, generated OpenAPI drift, and package tests listed in Phase 1 proof obligations.
Verification commands:
- `go test ./internal/httpapi/export -count=1`
- `make openapi-check`
Conclusion: verified for Phase 1 only if both commands pass. Do not call the repository fully green from this focused proof.
```

Copy the scoped conclusion: it names what the commands prove and what they do not prove.

```markdown
Claim: `ready for handoff` for a task that changed API, SQL migrations, and cache invalidation.
Scope: all changed surfaces in the approved plan.
Verification commands:
- scoped package tests for changed API and cache packages
- repository-owned OpenAPI drift check
- repository-owned migration validation command
Conclusion: handoff-ready only if every required surface passes fresh verification.
```

Copy the multi-surface proof shape only when the approved plan actually spans those surfaces.

## Reject

```markdown
Claim: task done.
Proof: `go test ./internal/httpapi/export -run TestCreateExport -count=1` passed.
Conclusion: the whole repository is done and ready.
```

Fails because a single focused test cannot support task-wide completion wording.

```markdown
Claim: ready for handoff.
Proof: the review agent said the code looked safe.
Conclusion: verified.
```

Fails because delegated review is not fresh workspace proof.

```markdown
Claim: all tests pass.
Proof: did not run tests because the diff is small.
Conclusion: tests pass by inspection.
```

Fails because inspection cannot be reported as a command result.

## Agent Traps
- Letting broad words like `done`, `ready`, or `complete` survive after choosing a narrow proof set.
- Running the cheapest familiar command instead of the command tied to the artifact's proof obligation.
- Saying "all tests pass" when only a subset ran.
