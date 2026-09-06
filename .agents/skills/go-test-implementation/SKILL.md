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

For a new integration scenario family, write one complete path through state
creation, the operation, independent observation, and cleanup before expanding
cases. Reuse that foundation while binding each case to its own inputs. An
existing valid scenario already supplies this foundation; do not rebuild it.
The active workflow decides when the scenario runs.

Load the [reference selector](references/index.md) only for a concrete problem
with the proving layer, fixture, control, or command. Keep the result in test
code and any existing task packet; add no proof-design artifact.

This method is complete when required tests and fixtures are written and their
execution commands are recorded. The active workflow and [Evidence
Contract](../../../docs/spec-first-workflow/shared/evidence-contract.md) own
validation timing and task completion. Missing test infrastructure is a
final-validation input, not missing permission to write the tests.
