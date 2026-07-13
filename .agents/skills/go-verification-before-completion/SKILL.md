---
name: go-verification-before-completion
description: "Verify correctness or readiness claims with fresh command evidence matched to claim scope before reporting success."
---

# Go Verification Before Completion

## Outcome

Bind every positive completion or readiness claim to fresh evidence from the current workspace, and narrow the conclusion when proof is missing, skipped, stale, cached unexpectedly, or out of scope.

## Method

1. State the exact claim and its scope.
2. Inspect the current workspace and the repository-owned proof commands.
3. Choose the smallest command set that directly proves the claim.
4. Run it now, inspect exit status plus key pass/fail/skip signals, and record the result.
5. Report `verified`, `partially verified`, or `not verified` without extrapolating beyond the evidence.

## Proof Rules

- A targeted fix replays the original failing signal or narrowest honest reproducer.
- Package, repository, race, lint, build, generated, migration, and readiness claims use the command owned by that surface.
- Replacement or cleanup claims add targeted negative proof for the exact retired identifiers, paths, routes, configs, commands, generated artifacts, fixtures, docs, skills, agents, or mirrors.
- Generated or mirrored changes require source-of-truth plus drift/sync proof; a hand-edited derived file is not evidence.
- Delegated reports, old CI logs, artifact status, and prior-session output are leads, not current proof.
- A skipped or cached action supports a positive claim only when its semantics genuinely prove the claim.
- Focused proof supports only a focused conclusion; broad readiness requires all checks triggered by the changed surfaces.

## Reference Selector

Load at most one reference by default; use more only for independent proof pressures.

| Symptom | Load | Decision it sharpens |
| --- | --- | --- |
| “Fixed”, “green”, “ready”, test, lint, build, race, package, or repo claim is ambiguous. | [claim-to-proof-mapping.md](references/claim-to-proof-mapping.md) | Select the narrowest sufficient proof without overclaiming. |
| OpenAPI, generated API, sqlc, query, or migration changed. | [generated-api-and-migration-verification.md](references/generated-api-and-migration-verification.md) | Add authoritative drift or migration proof. |
| An agent, tool, CI snippet, or prior session says work is done. | [delegated-work-verification.md](references/delegated-work-verification.md) | Rebind the claim to current workspace evidence. |
| Proof failed, skipped, is absent, or is weaker than the claim. | [failure-and-gap-reporting.md](references/failure-and-gap-reporting.md) | Report the gap and next proving action without optimistic wording. |

## Boundaries

Do not debug or repair the failure, author process artifacts, force unrelated repository-wide checks for a narrow claim, or treat review findings as verified implementation evidence. If the proving command is unclear, inspect `Makefile`, CI, and `docs/build-test-and-development-commands.md`; if it remains unclear, report that as the proof gap.

## Output

Return a compact note: claim and scope; commands actually run; observed pass/fail/skip signal; proportional conclusion; next action when not fully verified.

## Success And Stop

Success means each positive statement has fresh evidence of equal scope. Otherwise stop with `partially verified` or `not verified`, the blocking signal, and the smallest next verification action.
