---
name: go-coder
description: "Implement production-grade Go changes from approved requirements and task ledgers with review-clean defaults: explicit design, language-native and stdlib-first choices, mature-OSS-before-custom discipline when approved, seam-named same-package source-of-truth extraction when stable policy starts to spread, idiomatic control flow, preserved invariants, safe boundaries, and fresh verification evidence."
---

# Go Coder

## Trigger And Scope

Use this skill to implement approved Go features, fixes, refactors, integrations, regenerations, focused cleanup, and task-required tests. Translate an approved behavior and executable task boundary into the smallest complete production-grade diff.

Do not use it to choose unresolved architecture, API, data, security, reliability, rollout, dependency, or pattern semantics. Do not hand-edit generated output as the source of truth, widen scope through speculative cleanup, or repair missing workflow artifacts during coding.

## Approved Input And Write Boundary

- For ledger-backed work, require the current approved `tasks.md`, its source anchors and implementation obligations, and an eligible task-review/readiness verdict: `PASS`, eligible `CONCERNS` with named accepted risks and proof obligations, or eligible artifact-backed `WAIVED`.
- `FAIL`, stale or missing readiness, unnamed `CONCERNS`, a missing required ledger, contradictory obligations, or unresolved design blocks edits. Route to the recorded reopen target; this skill consumes readiness and never creates it.
- Direct-path edits are allowed only for the current orchestrator with a current `direct_state_envelope`, matched `SHAPE-DIRECT`, every `DIRECT-*` predicate true, and every `FULL-*` trigger false. Predicate loss stops edits.
- An isolated worker writes only its assigned ledger-bounded surfaces. It does not edit authoritative workflow state, mark final readiness, commit, push, merge, or claim integration completion. The orchestrator owns patch intake and fresh integration proof.
- Bind each edit to the active task, required generated drift, or cleanup made necessary by that task. Preserve unrelated dirty-worktree changes.

## Implementation Invariants

1. **Approved behavior is authoritative.** Preserve the approved source of truth and existing compatible semantics; stop when code would have to invent a product or protected-domain decision.
2. **The diff tells one complete story.** Choose the smallest complete change, include required replacement cleanup, and avoid speculative flexibility or adjacent refactors.
3. **Ownership stays focused.** Perform the `File Responsibility Check`: name the owner package/file and split-or-keep rationale before substantial edits. Prefer a focused same-package seam when stable mapping, normalization, validation, lifecycle, adapter, or test policy would otherwise spread; never hide ownership in vague `util`, `common`, or `shared` buckets.
4. **Go stays native and explicit.** Prefer the current Go language and standard library, then established repository patterns, then an approved mature OSS dependency. Do not introduce an unapproved runtime dependency, custom infrastructure, or material abstraction. Implement approved design patterns without inventing new pattern-shaped machinery.
5. **Boundaries remain safe.** Keep decoding, normalization, validation, state transitions, side-effect order, nil/empty behavior, aliasing, concurrency, retries, and partial failure deterministic and contract-preserving.
6. **Context, errors, and resources retain ownership.** Propagate caller context; preserve inspectable error identity; keep transport mapping at the boundary; make acquisition, cleanup, cursor terminal errors, transaction lifetime, cancellation, and goroutine shutdown explicit.
7. **Generated sources lead derived output.** Change OpenAPI, SQL, protobuf, mock, enum, or mirror owners first, regenerate or sync, and prove drift instead of treating derived files as authorities.
8. **Proof is smallest sufficient and fresh.** Add the narrow regression or invariant proof that would have failed before the change, run commands matched to the changed risk, and never convert skipped, cached-without-proof, stale, failing, or too-narrow evidence into a completion claim.

Cleanup required by the approved task is in scope. Remove or refactor replaced code, tests, fixtures, generated artifacts, configs, docs, scripts, examples, skills, agents, and mirrors. Retain an old surface only when the approved artifact chain names its owner, reason, proof of continued need, and exit condition; otherwise stop if removal would change protected behavior.

## Symptom-Driven Reference Selector

Use this entrypoint as a router. Name the behavior-change thesis before loading a reference: “For symptom X, this reference makes me choose Y instead of likely mistake Z.” Load at most one reference by default; load more only for independent pressures such as generated SQL plus transaction lifetime.

| Symptom or decision pressure | Load | Behavior change |
| --- | --- | --- |
| A helper, dependency, or older idiom may duplicate current Go or stdlib behavior. | [stdlib-first-modern-go.md](references/stdlib-first-modern-go.md) | Prefer native facilities or explicit local policy over wrappers and dependency-by-habit. |
| Helpers, interfaces, package moves, exports-for-tests, large files, or repeated same-package policy are in play. | [helper-extraction-and-package-ownership.md](references/helper-extraction-and-package-ownership.md) | Keep direct code or one seam-named owner instead of generic packages, provider-owned interfaces, or test-only exports. |
| HTTP or message decoding, size limits, unknown fields, trailing input, normalization, validation order, or immutable fields are changing. | [boundary-decoding-and-validation.md](references/boundary-decoding-and-validation.md) | Establish one bounded strict boundary before side effects instead of scattering defensive checks or partially accepting input. |
| Wrapped errors, cancellation, domain-to-transport mapping, repository translation, or log-and-return behavior is changing. | [errors-context-and-boundary-mapping.md](references/errors-context-and-boundary-mapping.md) | Preserve context and semantic error identity at the owning boundary. |
| Rows, bodies, files, locks, timers, contexts, transactions, cursor errors, commit order, or post-commit cache effects are changing. | [resource-lifetime-io-and-transactions.md](references/resource-lifetime-io-and-transactions.md) | Keep lifetime, terminal errors, transaction scope, and authoritative-before-derived ordering explicit. |
| Slices, maps, `[]byte`, snapshots, cache entries, pointer identity, nil/empty shape, or mutex-bearing values cross an ownership boundary. | [mutable-state-aliasing.md](references/mutable-state-aliasing.md) | Clone or preserve identity at the handoff instead of leaking aliases or copying synchronization state. |
| Goroutines, channels, fan-out, worker pools, shutdown, timers, tickers, or request-scoped async work changes. | [concurrency-and-background-work.md](references/concurrency-and-background-work.md) | Make lifecycle, cancellation, bounds, ordering, and race proof visible. |
| Tests, fuzzing, benchmarks, clocks, randomness, failure messages, cleanup proof, or final verification shape changes. | [testing-verification-patterns.md](references/testing-verification-patterns.md) | Prove the changed behavior at the smallest deterministic layer and match verification to risk. |
| OpenAPI, sqlc, protobuf, mocks, generated files, configs, or mirror drift changes. | [generated-source-of-truth-and-drift.md](references/generated-source-of-truth-and-drift.md) | Edit the owning source, regenerate, and verify drift rather than patching derived output. |

## Required Evidence

Before editing, record the task/source anchor, owner package/file, split-or-keep rationale, expected behavior, forbidden regression, required cleanup, and proof command. For protected-domain or `test-plan.md` work, bind every referenced scenario to its invariant, owning path, forbidden side effect, and proof; missing or contradictory mapping reopens planning or test design.

Before handoff:

- inspect the diff for scope, ownership, generated-source order, obsolete surfaces, context/error/resource safety, and unapproved decisions;
- run the narrow failing reproduction or targeted tests first, then only the broader generation, race, integration, or repository checks required by the changed surface;
- update existing ledger progress only when the active task explicitly authorizes it and fresh proof covers the task; otherwise leave it unchecked with `Blocked:` or a narrower claim;
- report commands and observed results, changed behavior, preserved invariants, cleanup disposition, generated drift, and residual risks.

## Success, Stop, And Escalation

Success means the assigned behavior is implemented in the owning surfaces, required cleanup and generated outputs are aligned, focused proof is green, and the report stays proportional to that evidence.

Stop without edits, or stop at the first newly discovered decision boundary, when readiness is ineligible; required inputs conflict; a new dependency, pattern, public contract, data model, security rule, reliability policy, rollout shape, or owner is needed; generated authority is unknown; or safe proof cannot be named. A blocked report names the exact artifact or decision, smallest unblock action, and reopen owner. It does not use `implemented`, `fixed`, `ready`, or empty implementation headings.

Escalate architecture or ownership to `go-architect-spec` or `go-design-spec`; API or routing semantics to `api-contract-designer-spec` or `go-chi-spec`; data/cache to `go-data-architect-spec` or `go-db-cache-spec`; invariants to `go-domain-invariant-spec`; reliability or distributed consistency to their spec skills; security, observability, performance, or test strategy to the matching specialist.
