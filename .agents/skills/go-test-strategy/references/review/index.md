# Reference Selector

| Symptom | Load | Behavior change |
| --- | --- | --- |
| The review risks reporting coverage, test count, or "assert more" rather than a leak, or a failure class went unproved. | [false-pass-and-missing-proof.md](false-pass-and-missing-proof.md) | Makes the model state the regression that would still pass, and choose between error identity and exact text from what would leak, instead of using coverage, volume, or library preference as the finding. |
| Proof rests on timing, scheduler, or shared-state luck, or on `testing/synctest` judgment. | [determinism-and-flake-risk.md](determinism-and-flake-risk.md) | Makes the model name the uncontrolled source and the deterministic shape that replaces it, instead of removing the sleep, adding `-race`, or raising `-count`. |
| The reported validation may not exercise the changed package, tag, generated contract, or race surface. | [../decision/validation-commands.md](../decision/validation-commands.md) | Makes the model test the command against the regression it must catch instead of accepting breadth as fitness. |

Translate a loaded criterion into the current diff's exact source, missing obligation, false-pass or regression-leakage path, and focused validation; do not restate it generically.
