# Parallel Slice Execution

## Read When

Read only when an Acceptance-Unit Lead selects multiple slices, a non-initial
base, a shared mutable resource or proof gate, or a cross-checkout input. The
[Lead Execution](implementation-lead-execution.md) contract owns the unit and
acceptance.

## Execution Map

The map is trace state, not a repository artifact. Record:

- the accepted base tree;
- each slice's ID, outcome, base, writes, immutable inputs, resources, carrier,
  checkout, and focused proof;
- each dependency edge and the exact output it carries;
- each symmetric conflict over a writable path, resource, or proof gate; and
- currently proven carrier, resource, and proof-gate capacity.

Add an edge only when the successor consumes same-path content, a type or
contract absent from its base, generated or migration output, state transition,
or proof input. Shared feature labels, packages, broad final gates, and possible
merge risk create no edge. Shared writes or exclusive resources create a
conflict unless one side consumes the other's output. Collapse cycles and any
dependency chain whose successor base the harness cannot materialize into one
serial slice before dispatch.

For a dirty working-tree base, derive one synthetic Git tree with a temporary
index covering tracked and untracked non-ignored bytes. Validate that identity
before each first edit. External or ignored inputs cross as read-only locators
with exact identities; never copy them into a Worker checkout.

## Scheduling And Intake

A slice is ready when every predecessor is integrated; its exact base and
inputs match; a harness-valid Worker carrier can materialize its checkout; and
no active slice conflicts with its writes, resources, or proof. Dispatch every
ready slice that fits current proven capacity. When capacity is unknown,
dispatch conservatively and adapt to the native result rather than inventing a
limit.

Reserve a slice's writes, resources, and proof gates at dispatch. Before its
first write, record the actual native identity and checkout and apply the Agent
Harness [Write-Carrier
Gate](../../agent-harness/shared/write-carrier.md#write-carrier-gate). A
carrier mismatch invalidates the lane; post-write bytes are diagnostic only.

Treat a Worker checkout as mutable until `DONE`. Observe completion and stable
status, not provisional bytes. Consume returned candidates serially: validate
scope, ownership, mergeability, base and input identities, and proof provenance;
integrate an intake-valid delta; release reservations; then recompute readiness.
If integration changes an active sibling's declared input, stop that sibling
before another edit and remap it.

When a corrected slice changes a downstream input, invalidate only the affected
dependency closure, rebuild from the initial base plus preserved independent
deltas, emit the updated map, and resume affected original Workers on their new
bases when the harness supports it.
