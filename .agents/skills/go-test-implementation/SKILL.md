---
name: go-test-implementation
description: "Executable Go falsifiers. Use for test-only changes or non-routine fixtures and harnesses, selecting cases and assertions from accepted behavior during implementation."
metadata:
  invocation: model
  kind: method
---

# Go Test Implementation

Write tests that reject wrong behavior through an observable independent of
the implementation being tested. Routine tests alongside production code stay
with `go-coder`; this method handles test-only work or non-routine controls.

Choose cases, fixtures, assertions, and commands while writing the tests from
accepted product behavior and existing repository patterns. No pre-approved
scenario, oracle, matrix, or separate test plan is required. Use the smallest
deterministic layer that observes the claim. A source-string assertion proves
behavior only when the exact text is itself the accepted output contract.

Load the [reference selector](references/index.md) only for a concrete problem
with the proving layer, fixture, control, or command. Keep the result in test
code and its normal task packet; add no proof-design artifact.

Return Implemented when the required tests and fixtures are written, recording
commands for final validation. Execute nothing during ledger implementation.
The [Evidence Contract](../../../docs/spec-first-workflow/shared/evidence-contract.md)
owns final execution: the command must run the intended tests and establish
their observables before a passing behavior claim. Missing test infrastructure
is a final-validation input, not missing permission to write the tests.
