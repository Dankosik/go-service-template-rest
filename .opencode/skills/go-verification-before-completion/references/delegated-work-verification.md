# Delegated Work Verification

## Behavior Change Thesis
When loaded for delegated, prior-session, tool, or CI claims, this file makes the model rebind the claim to current workspace and same-commit evidence instead of treating another report, stale log, or uninspected diff as proof.

## When To Load
Load this when another agent, worker, prior session, CI snippet, pasted log, or tool report says work is done, tests passed, findings were fixed, or code is ready for handoff.

## Decision Rubric
- Treat delegated reports as leads, not proof.
- Identify the delegated claim, the files/surfaces it covers, and whether later local edits could invalidate it.
- Inspect current workspace state enough to know what changed, but do not treat diff inspection as behavioral proof.
- Rerun the claim-scoped command now, or verify current CI output for the same commit, same command, and same scope.
- If the delegated output includes command logs, check timestamp or commit applicability before using them.
- After you edit anything covered by the delegated proof, the proof is stale until rerun.

## Imitate
| Claim | Choose | Copy this behavior |
|---|---|---|
| "The worker fixed it" | inspect current changed paths, then rerun the focused reproducer | Use the worker summary to find the proof target, not as proof. |
| "The reviewer finding is resolved" | verify the changed code addresses the finding and run the relevant command now | Pair semantic reconciliation with command evidence. |
| "Delegated tests passed" | rerun the named command or verify current CI for the same commit and scope | Check that the evidence still applies to the workspace. |
| "Ready to hand off delegated work" | understand current `git status --short`, changed-surface scope, and fresh claim-scoped commands | Include workspace drift in the trust decision. |

## Reject
| Plausible bad conclusion | Why it fails |
|---|---|
| "Fixed" because a worker final answer says fixed | A final answer is not executable evidence. |
| "Tests pass" because a pasted log passed before your later edits | Later edits invalidate the old proof for the touched scope. |
| "No findings remain" because a reviewer found none in a read-only pass | Review output can reduce risk, but it does not prove runtime behavior. |
| "Ready" while local uninspected modifications overlap the delegated scope | The delegated proof may not describe the current workspace. |

## Agent Traps
- Do not ask a second agent to "verify" and then treat the agent answer as proof; the proof is the command or same-commit CI evidence it points to.
- Do not collapse "diff inspected" into "behavior verified".
- Do not claim delegated CI proof unless the command, commit, and scope match the current claim.
- In a dirty worktree, separate user or unrelated changes from the selected scope before claiming what was verified.

## Validation Shape
Report what the delegated source claimed, what current workspace scope you checked, which command or same-commit CI evidence proves it, and what remains unverified.
