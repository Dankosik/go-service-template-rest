---
name: go-delivery-platform
description: "Delivery platform: Use for CI/CD gates, release trust, drift, containers, migrations, rollout, or control review. Own delivery acceptance; Skip API, data, security, reliability, or pipeline implementation."
---

# Go Delivery Platform

Load the [shared specialist contract](../specialist-contract.md). Reconstruct required gates from accepted delivery policy, repository workflows, build and deploy surfaces, migrations, generated/docs controls, and rollout dependencies. Treat each status, parity check, provenance rule, container, rollout, or exception as a gate with an artifact, command, observable prerequisite, pass condition, exception owner, and recovery consequence.

## Choose The Branch

- **Decision** — select when delivery policy is absent or changing. Load the [decision selector](references/decision/index.md) for one result-changing pressure. Complete when shared Decision dispositions cover every gate, forced consequence, proof, and exception owner.
- **Review** — select when changed delivery artifacts must conform to accepted policy. Load the [review selector](references/review/index.md) for the changed control. Account for every gate through the shared finding envelope, naming any outside boundary or proof blocker; a waived required gate remains a finding with focused proof. Missing policy returns to the named Delivery Decision owner.

Hand resilience policy to `go-reliability` and trust controls to `go-security`.
