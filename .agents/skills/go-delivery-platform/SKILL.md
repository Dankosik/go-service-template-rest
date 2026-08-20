---
name: go-delivery-platform
description: "Delivery: Use for CI/CD, release trust, drift, containers, migrations, rollout, or control review. Own delivery acceptance; Skip implementation."
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

Load the [shared specialist contract](../../contracts/specialist-contract.md). Reconstruct required gates from accepted delivery policy, repository workflows, build and deploy surfaces, migrations, generated/docs controls, and rollout dependencies.

## Choose The Branch

- **Decision** — load one matching [decision reference](references/decision/index.md)
  and cover every gate, forced consequence, proof, and exception owner.
- **Review** — load one matching [review reference](references/review/index.md)
  and account for every gate; a waived required gate remains a finding.

Hand resilience to `go-reliability`, trust to `go-security`, and a release schema
migration to [PostgreSQL schema design](../../../docs/universal-disciplines/postgres-schema-design/SKILL.md).
