# Required Validation Scenarios

Load before required runtime/integration proof or an accepted runner/fixture
scenario. [Implementation](../implementation.md) owns the validation boundary.

When runtime or integration proof is already required, prioritize the smallest
existing scenario that can expose incompatibility at the changed boundary
before broader scenario expansion. Select from the actual change: startup
configuration, migration/runtime compatibility, serialized values, or retained
background processing. Use representative inputs for the suspected failure.

This ordering adds no mandatory scenarios or infrastructure. Reuse sufficient
current evidence under the Evidence Contract.

If the task explicitly includes a runner or infrastructure fixture, execute its
smallest complete scenario
through setup, the intended behavior, observation, and cleanup before expanding
the run. Reuse a still-valid result from coding when the Evidence Contract
permits it. Bind each scenario to its own inputs.
Preserve safe root-cause diagnostics and give cleanup a bounded
lifetime independent of a failed or cancelled scenario. Reuse the prepared
environment while its state and inputs remain valid; reset only what a failed
run invalidated.
