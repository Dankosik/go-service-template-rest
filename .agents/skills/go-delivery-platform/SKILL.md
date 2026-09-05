---
name: go-delivery-platform
description: "Release gates. Use when CI/CD, artifact provenance, drift, containers, migrations, rollout, or control-plane evidence determines whether a candidate may ship."
metadata:
  invocation: model
  kind: method
---

# Go Delivery Platform

Delivery trust is a chain of **gates** with named artifacts, pass conditions,
exceptions, and recovery consequences.

`accepted policy -> gate inventory -> artifact and command -> pass condition -> exception owner -> rollout and recovery -> proof`

Treat each status, parity check, provenance rule, container, migration, or
rollout control as a gate. A waiver has an owner and expiry; drift between
declared and actual controls is a finding.

Load the [shared specialist contract](../../contracts/specialist-contract.md).
From accepted delivery policy to terminal rollout or rollback, build
`DeliveryGate{control, artifact, command, pass_condition, exception_owner,
expiry, rollout, recovery, proof}` for every required status, parity check,
provenance rule, container, migration, or runtime control.

## Choose The Branch

- **Decision** — load one matching [decision reference](references/decision/index.md)
  and disposition every gate, forced consequence, proof, and exception owner.
- **Review** — load one matching [review reference](references/review/index.md)
  and account for every gate; a waived required gate remains a finding.

Complete only when every required gate has a current artifact, fail-closed pass
condition, exception disposition, and recovery consequence. A green job that
did not compare or exercise its intended surface is not a passing gate.
