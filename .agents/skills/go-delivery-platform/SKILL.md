---
name: go-delivery-platform
description: "Delivery platform: Use when CI/CD gates, release trust, drift controls, containers, migrations, or rollout policy needs a decision, or when changed delivery controls need conformance review. Own delivery acceptance and review; Skip when API, data, security, reliability, or pipeline implementation is primary."
---

# Go Delivery Platform

Load the [shared specialist contract](../specialist-contract.md). Keep required statuses, local/CI parity, provenance, generated/docs drift, migration, container, rollout, and exception controls enforceable.

## Choose The Branch

- **Decision** — select when delivery policy is absent or changing. Load the [decision selector](references/decision/index.md) for one result-changing pressure. Complete when exact acceptance, forced consequences, proof, exceptions, and blockers are explicit.
- **Review** — select when changed delivery artifacts must conform to accepted policy. Load the [review selector](references/review/index.md) for the changed control. Complete when every required control is dispositioned, waived controls are findings, and each finding has the smallest safe correction and focused proof; missing policy stays in the decision branch.

Hand resilience policy to `go-reliability` and trust controls to `go-security`.
