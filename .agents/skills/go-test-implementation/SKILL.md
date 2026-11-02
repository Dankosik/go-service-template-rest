---
name: go-test-implementation
description: "Use for test-only changes or non-routine fixtures and harnesses, selecting cases and assertions from accepted behavior during implementation."
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

Before adding a case, identify the material wrong behavior existing coverage
would miss, or the explicit proof requirement it satisfies. Extend an existing
test when that supplies the missing observation. Avoid duplicating the same
contract without a distinct failure mode; retain separate cases or proving
layers when they can fail independently, including success and denial paths.
This judgment needs no separate inventory or approval.

## Test Shape

Every new or changed test, routine or not, takes this shape:

- **Caller's view.** Drive the entry point a real caller uses — the router, the
  gRPC client, the exported function, ideally from a `_test` package — and
  assert what that client, operator, or feature author observes: status and
  body, error identity, durable row, emitted signal. Test an unexported helper
  only when the exported path cannot reach the case; never keep an export,
  package variable, or hook alive only for a test.
- **Real before fake.** Use real in-process collaborators first: stdlib types,
  `httptest`, the chi router, `bufconn`. When the engine owns the claim, use
  the existing Testcontainers fixtures — `pgtest` and the shared NATS harness.
  A fake stands in only for an out-of-process dependency on a Go-owned claim,
  and it returns data; it never implements the behavior the test asserts.
  Calls recorded on a fake are not evidence when a client-visible or durable
  effect exists.
- **No logic.** Expected values are literals or fixture data written from the
  accepted behavior. No loop, branch, or production helper computes an
  expected value; a table row that needs a different assertion shape becomes
  its own test.
- **Regression first.** For a bug, write the test at the owner boundary, watch
  it fail on the pre-fix code for the bug's reason, then fix. A regression test
  never seen failing proves nothing.
- **Probe for gaps.** After testing a changed predicate or branch, mutate it by
  hand — flip a comparison, drop a condition, return early — and rerun the
  focused test; a green run names a missing case. Revert the mutation. An
  invariant over generated input gets a fuzz target; see
  [proving-layer-and-oracle.md](references/proving-layer-and-oracle.md).

## Scenarios And Completion

For a new integration scenario family, write one complete path through state
creation, the operation, independent observation, and cleanup before expanding
cases. Reuse that foundation while binding each case to its own inputs. An
existing valid scenario already supplies this foundation; do not rebuild it.
The active workflow decides when the scenario runs.

Load the [reference selector](references/index.md) only for a concrete problem
with the proving layer, fixture, control, or command. Keep the result in test
code and any existing task packet; add no proof-design artifact.

This method is complete when required tests and fixtures are written and their
execution commands are recorded. The active workflow owns validation timing and
task completion. Missing test infrastructure is a final-validation input, not
missing permission to write the tests.
