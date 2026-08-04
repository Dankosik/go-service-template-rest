---
name: go-delivery-platform
description: "Delivery platform: Use for CI/CD gates, release trust, drift, containers, migrations, rollout, or control review. Own delivery acceptance; Skip API, data, security, reliability, or pipeline implementation."
---

# Go Delivery Platform

Delivery trust is a chain of **gates**: an artifact reaches production only through named checks whose pass conditions, exceptions, and recovery consequences are explicit — a waived gate is a decision with an owner and an expiry, not a shortcut.

`accepted policy -> gate inventory -> artifact and command -> pass condition -> exception owner -> rollout and recovery -> proof`

Treat each status, parity check, provenance rule, container, migration step, or rollout control as a gate with an artifact, command, observable prerequisite, pass condition, exception owner, and recovery consequence. Drift between declared and actual controls is itself a finding: a gate that exists only in documentation protects nothing.

Load the [shared specialist contract](../specialist-contract.md). Reconstruct required gates from accepted delivery policy, repository workflows, build and deploy surfaces, migrations, generated/docs controls, and rollout dependencies.

## Choose The Branch

- **Decision** — select when delivery policy is absent or changing. Load the [decision selector](references/decision/index.md) for one result-changing pressure. Complete when shared Decision dispositions cover every gate, forced consequence, proof, and exception owner.
- **Review** — select when changed delivery artifacts must conform to accepted policy. Load the [review selector](references/review/index.md) for the changed control. Account for every gate through the shared finding envelope, naming any outside boundary or proof blocker; a waived required gate remains a finding with focused proof. Missing policy returns to the named Delivery Decision owner.

Hand resilience policy to `go-reliability` and trust controls to `go-security`. Load [`postgres-schema-design`](../../../docs/universal-disciplines/postgres-schema-design/SKILL.md) when a release carries a schema migration: it forces an expand-then-contract sequence that the running and the deploying version can both read and write, instead of a gate that only checks the migration executed.
