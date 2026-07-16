# Reference Selector

| Symptom | Load | Behavior change |
| --- | --- | --- |
| The review risks saying "add more tests" without naming the missing scenario. | [scenario-traceability-review.md](scenario-traceability-review.md) | Makes the model tie changed behavior to one named scenario and regression leakage instead of using test count or coverage as proof. |
| The test exists but can pass while the observable contract is wrong, or failures would not localize cause. | [assertion-strength-and-diagnostics.md](assertion-strength-and-diagnostics.md) | Makes the model ask for stable got/want, side-effect, state, or error-shape assertions instead of library-preference or "assert more" findings. |
| Proof relies on timing luck, scheduler luck, shared state, parallel isolation, race runs, leak checks, or `testing/synctest` judgment. | [determinism-isolation-and-flake-risk.md](determinism-isolation-and-flake-risk.md) | Makes the model identify the uncontrolled source and deterministic proof shape instead of blanket sleep removal, `-race`, or `-count=100` advice. |
| The changed behavior includes failure classes, boundaries, malformed input, fuzz seeds, or abuse-path obligations. | [fail-edge-and-abuse-path-coverage.md](fail-edge-and-abuse-path-coverage.md) | Makes the model request the smallest representative negative case instead of exhaustive matrices, fuzz-everything advice, or threat-model ownership. |
| Reported validation does not exercise the changed risk surface at the right package, tag, contract, race, fuzz, or CI-parity level. | [validation-command-fit.md](validation-command-fit.md) | Makes the model map validation to the regression under discussion instead of accepting broad `go test ./...` or demanding full CI by default. |
| The QA gap depends on specialist semantics from domain, API, DB/cache, concurrency, security, reliability, performance, or design. | [cross-domain-test-gap-handoffs.md](cross-domain-test-gap-handoffs.md) | Makes the model separate the local executable proof gap from the specialist question instead of punting vaguely or over-owning another review lane. |

Translate a loaded example into the current diff's exact source, missing obligation, false-pass or regression-leakage path, and focused validation; do not paste generic examples.
