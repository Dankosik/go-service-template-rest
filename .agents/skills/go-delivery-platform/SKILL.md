---
name: go-delivery-platform
description: "Delivery platform: Use when CI/CD gates, release trust, drift controls, containers, migrations, or rollout policy needs a decision, or when changed delivery controls need conformance review. Own delivery acceptance and review; Skip when API, data, security, reliability, or pipeline implementation is primary."
---

# Go Delivery Platform

Load the [shared specialist contract](../specialist-contract.md). Reconstruct required gates from accepted delivery policy, repository workflows, build and deploy surfaces, migrations, generated/docs controls, and rollout dependencies. Treat each status, parity check, provenance rule, container, rollout, or exception as a gate with an artifact, command, observable prerequisite, pass condition, exception owner, and recovery consequence.

## Choose The Branch

- **Decision** — select when delivery policy is absent or changing. Load the [decision selector](references/decision/index.md) for one result-changing pressure. Complete when shared Decision dispositions cover every gate, forced consequence, proof, and exception owner.
- **Review** — select when changed delivery artifacts must conform to accepted policy. Load the [review selector](references/review/index.md) for the changed control. Account for every gate through the shared finding envelope, naming any outside boundary or proof blocker; a waived required gate remains a finding with focused proof. Missing policy ends this run with a named Delivery Decision handoff; conformance Review begins separately after acceptance.

Hand resilience policy to `go-reliability` and trust controls to `go-security`.
