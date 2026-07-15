# Reference Selector

References sharpen a triggered proof decision after accepted behavior is reconstructed; they neither define scope nor act as checklists.

| Symptom | Load | Decision it sharpens |
| --- | --- | --- |
| Proof level is unclear or broad e2e is proposed “for safety.” | [test-level-selection.md](test-level-selection.md) | Choose the smallest sufficient complementary boundaries. |
| The matrix is generic or happy-path-only. | [scenario-matrix-patterns.md](scenario-matrix-patterns.md) | Produce compact scenarios with observable outcomes. |
| Invariants or acceptance criteria lack traceable proof. | [invariant-and-acceptance-traceability.md](invariant-and-acceptance-traceability.md) | Map each claim to scenario and reopen trigger. |
| Timeout, retry, poison, backpressure, shutdown, or recovery matters. | [reliability-fail-path-test-obligations.md](reliability-fail-path-test-obligations.md) | Define deterministic failure triggers and lifecycle observables. |
| REST/OpenAPI, auth, validation, idempotency, or async `202` changed. | [api-contract-and-boundary-tests.md](api-contract-and-boundary-tests.md) | Choose boundary-observable contract proof. |
| SQL, cache, tenant storage, migration, outbox/inbox, replay, or reconciliation matters. | [data-cache-security-distributed-test-obligations.md](data-cache-security-distributed-test-obligations.md) | Select stateful and message observables rather than mocks. |
| Local/CI commands or proof limits are unclear. | [quality-gates-and-execution.md](quality-gates-and-execution.md) | Bind obligations to executable repository checks. |
