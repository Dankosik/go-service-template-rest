---
name: go-verification-before-completion
description: "Use when a correctness, readiness, or completion claim needs fresh evidence; Own claim-to-command matching, result inspection, proof scope, and explicit gaps; Skip when code or tests must be changed, root cause is unknown, test strategy is unresolved, or implementation is still underway."
---

# Go Verification Before Completion

## Accepted Input And Boundary

Accept a concrete correctness, cleanup, readiness, or completion claim and its intended scope. Verification is evidence-only: inspect the current workspace and repository proof surfaces, but do not debug, repair, change tests, author process artifacts, or treat implementation reports and review findings as proof. If behavior, ownership, acceptance criteria, or the proving command remains unresolved after inspecting repository authorities, name that owner and stop.

## Method

1. Inventory each positive claim and its exact scope, including behavior, cleanup, generated drift, migration, race, build, lint, package, repository, or readiness dimensions.
2. Inspect the current workspace and repository-owned proof commands, then choose the smallest command set capable of proving every dimension.
3. Run the commands now. Inspect working scope, exit status, executed versus cached work, and key pass, fail, skip, or unavailable signals rather than trusting a summary line.
4. Reject stale reports, prior-session output, unexpected cache reuse, weakened commands, skips, exclusions, and focused proof that is narrower than the claim.
5. Classify each evidence item as passed, failed, unavailable, or skipped, then report `verified`, `partially verified`, or `not verified` without extrapolation.

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

## Proof, Return, And Stop

If a proving command is unclear, inspect `Makefile`, CI, and `docs/build-test-and-development-commands.md`; if it remains unclear, report the proof gap instead of guessing or forcing unrelated broad checks. Return a compact note with each claim and scope, commands actually run, observed pass/fail/skip/unavailable signals, and a proportional conclusion. Success means every positive statement has fresh evidence of equal scope. Otherwise return `partially verified` or `not verified`, the blocking signal, its owner, and the smallest next verification action.
