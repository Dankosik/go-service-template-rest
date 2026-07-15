---
name: go-test-review
description: "Use when a Go diff's tests and validation evidence need review for behavioral coverage, scenario traceability, assertion strength, determinism, and failure-path proof; Own test-quality defects; Skip when tests must be written, proof strategy must be designed, domain behavior is unresolved, or completion claims need final verification."
---

# Go Test Review

Load the [shared specialist contract](../specialist-contract.md) for common selection, scope, evidence, reference, return, and handoff mechanics; apply the domain-specific rules below.

## Target, Boundary, And Invariants

Review tests and reported validation against accepted behavior for proof that would fail on the changed regression. Own proof-quality defects, not test volume or the underlying domain policy; hand off only the semantic question a test review cannot decide.

1. Trace each material behavior, invariant, or contract change to named happy, failure, and relevant edge/abuse scenarios; test count, line coverage, compilation, and orphan tests are not behavioral proof.
2. Assertions must reject plausible false positives by checking stable observable outcomes, side effects, state transitions, emitted events, or error identity/shape, with diagnostics that localize the broken scenario.
3. Time, randomness, environment, process globals, external dependencies, parallelism, and goroutine scheduling are isolated or explicitly controlled; sleeps, repetition, race runs, and leak checks prove only the risks they actually exercise.
4. Negative proof covers each changed failure class, boundary, retry/conflict, async completion state, or accepted abuse path that could regress independently; do not invent threat, reliability, transaction, or domain semantics.
5. Fixtures, helpers, fuzz seeds, and goldens remain minimal, representative, owned, reviewable, and failure-legible; regeneration or shared mutable setup cannot silently bless the implementation under test.
6. Replacement work proves retired code, identifiers, fixtures, generated output, config, docs, and old test paths are absent, or records approved retained ownership, reason, proof, and exit condition.
7. Validation commands exercise the changed package and required level—unit, contract, integration, generated drift, race, fuzz, benchmark, or CI parity—and fresh evidence is distinguished from a proposed command.

## Symptom-Driven References

| Symptom | Load | Behavior change |
| --- | --- | --- |
| The review risks saying "add more tests" without naming the missing scenario. | `references/scenario-traceability-review.md` | Makes the model tie changed behavior to one named scenario and regression leakage instead of using test count or coverage as proof. |
| The test exists but can pass while the observable contract is wrong, or failures would not localize cause. | `references/assertion-strength-and-diagnostics.md` | Makes the model ask for stable got/want, side-effect, state, or error-shape assertions instead of library-preference or "assert more" findings. |
| Proof relies on timing luck, scheduler luck, shared state, parallel isolation, race runs, leak checks, or `testing/synctest` judgment. | `references/determinism-isolation-and-flake-risk.md` | Makes the model identify the uncontrolled source and deterministic proof shape instead of blanket sleep removal, `-race`, or `-count=100` advice. |
| The changed behavior includes failure classes, boundaries, malformed input, fuzz seeds, or abuse-path obligations. | `references/fail-edge-and-abuse-path-coverage.md` | Makes the model request the smallest representative negative case instead of exhaustive matrices, fuzz-everything advice, or threat-model ownership. |
| Reported validation does not exercise the changed risk surface at the right package, tag, contract, race, fuzz, or CI-parity level. | `references/validation-command-fit.md` | Makes the model map validation to the regression under discussion instead of accepting broad `go test ./...` or demanding full CI by default. |
| The QA gap depends on specialist semantics from domain, API, DB/cache, concurrency, security, reliability, performance, or design. | `references/cross-domain-test-gap-handoffs.md` | Makes the model separate the local executable proof gap from the specialist question instead of punting vaguely or over-owning another review lane. |

Translate a loaded example into the current diff's exact source, missing obligation, false-pass or regression-leakage path, and focused validation; do not paste generic examples.

## Findings And Escalation

Each finding adds the missing or weak obligation, how current proof can pass while behavior is wrong, fixture/cleanup impact when relevant, and the narrow validation that would falsify the regression. `critical` is missing critical coverage or systemic nondeterminism that invalidates the suite and blocks merge until resolved or explicitly handed off; `high` includes a significant scenario/assertion gap or a replaced path that can still execute, import, generate, or validate without approved retention proof. On a clean change return no findings; state only concrete residual evidence gaps.

Escalate changed proof strategy to `go-test-design`; absent domain/API/reliability/security/data policy to its specification owner; transaction, concurrency, threat, hot-path, or ownership semantics to `go-db-cache-review`, `go-concurrency-review`, `go-security-review`, `go-performance-review`, or `go-implementation-ownership-review`. Keep the local executable proof gap in this review without duplicating the specialist finding; stop if no accepted behavior exists to test.
