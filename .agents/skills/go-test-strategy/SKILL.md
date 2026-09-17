---
name: go-test-strategy
description: "Use while implementing Go tests when the observable failure, deterministic control, or smallest proving layer is non-obvious."
metadata:
  invocation: model
  kind: method
---

# Go Test Strategy

Choose a test that rejects plausible wrong behavior at the smallest layer that
can observe it. This is an optional method while implementing code or tests,
not a separate phase, approval, or mandatory preliminary test plan.

Ground the assertion in accepted product behavior, not the implementation's
current output. Reuse existing test patterns and controls. Preserve fixture
cardinality, entity ownership, absent values, and event ordering. Prefer
assertions that survive behavior-preserving refactors; test internal calls only
when they carry an accepted contract or resource bound.

For the concrete choice in front of you, identify the wrong behavior, its
observable consequence, and deterministic inputs or fault control. Unit tests
are sufficient when they observe that consequence. Exercise available real
in-process boundaries where a mock would miss the wrong behavior. An explicit
real-database or provider verification claim needs that actual boundary, but
this method cannot make such a claim mandatory for local completion. Follow the
[Evidence Contract](../../../docs/spec-first-workflow/shared/evidence-contract.md#required-and-optional-proof)
before selecting external infrastructure. Do not build a test environment,
runner, or scenario matrix merely because a mock has limits. Small fixtures and existing
test doubles remain normal test authoring; keep optional unobserved scope clear.

Load one [decision reference](references/decision/index.md) only when its
pressure changes the test being written. A final or explicitly requested
review may use the [review references](references/review/index.md); they do not
trigger review during ledger implementation. If a bounded consultation is
delegated, use the [shared specialist contract](../../contracts/specialist-contract.md).

Done when the testing choice can be implemented in the current task. Repair
fixtures and assertions locally; reopen a product owner only when expected
behavior is actually undefined. The active workflow owns validation timing and
task completion.
