---
name: go-test-strategy
description: "Falsification design. Use while writing tests when a failure observable, deterministic control, or proving layer is non-obvious; resolve that choice inside the current implementation task."
metadata:
  invocation: model
  kind: method
---

# Go Test Strategy

Choose a test that rejects plausible wrong behavior at the smallest layer that
can observe it. This is an optional testing method within Implementation,
not a separate phase, approval, or mandatory preliminary test plan.

Ground the assertion in accepted product behavior, not the implementation's
current output. Reuse existing test patterns and controls. Preserve fixture
cardinality, entity ownership, absent values, and event ordering. Prefer
assertions that survive behavior-preserving refactors; test internal calls only
when they carry an accepted contract or resource bound.

For the concrete choice in front of you, identify the wrong behavior, its
observable consequence, and deterministic inputs or fault control. Unit tests
are sufficient when they observe that consequence. Use a real boundary when a
mock could pass while the claimed transaction, authorization, or recovery
behavior is absent. Do not create a scenario matrix or typed obligation record
before writing the test.

Load one [decision reference](references/decision/index.md) only when its
pressure changes the test being written. A final or explicitly requested
review may use the [review references](references/review/index.md); they do not
trigger review during ledger implementation. If a bounded consultation is
delegated, use the [shared specialist contract](../../contracts/specialist-contract.md).

Done when the testing choice can be implemented in the current task. Repair
fixtures and assertions locally; reopen a product owner only when expected
behavior is actually undefined. Under the [Evidence
Contract](../../../docs/spec-first-workflow/shared/evidence-contract.md), write
tests now and execute them only at final validation of the assembled ledger.
