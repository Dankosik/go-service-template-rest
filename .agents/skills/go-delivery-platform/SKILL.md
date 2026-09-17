---
name: go-delivery-platform
description: "Use when CI/CD, artifact provenance, drift, containers, migrations, rollout, or control-plane evidence determines whether a candidate may ship."
metadata:
  invocation: model
  kind: method
---

# Go Delivery Platform

Delivery trust is a chain of **gates** with named artifacts, pass conditions,
exceptions, and recovery consequences.

`accepted policy -> gate inventory -> artifact and command -> pass condition -> exception owner -> rollout and recovery -> proof`

Inventory the gates required by the accepted delivery policy. The presence of
a status, parity check, container, migration, or available control does not
create an additional acceptance gate. For a required gate, a waiver has an
owner and expiry; drift between declared and actual controls is a finding.

For a delegated Decision or Review, or when the active artifact requires its
result interface, load the
[shared specialist contract](../../contracts/specialist-contract.md).
When meaningful ordering, comparison, exhaustive accounting, or a required
decision/review handoff needs structured representation, trace accepted delivery policy to terminal rollout or rollback in
`DeliveryGate{control, artifact, command, pass_condition, exception_owner,
expiry, rollout, recovery, proof}` for every required status, parity check,
provenance rule, container, migration, or runtime control.
Otherwise, a single local control may retain its grounded gate judgment and
proof, including its exception and recovery disposition.

## Choose The Branch

- **Decision** — load one matching [decision reference](references/decision/index.md)
  and disposition every gate, forced consequence, proof, and exception owner.
- **Review** — load one matching [review reference](references/review/index.md)
  and account for every gate; a waived required gate remains a finding.

Complete only when every required gate has a current artifact, fail-closed pass
condition, exception disposition, and recovery consequence. A green job that
did not compare or exercise its intended surface is not a passing gate.
