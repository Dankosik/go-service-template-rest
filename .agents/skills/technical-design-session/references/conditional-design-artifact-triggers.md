# Conditional Design Artifact Triggers

Use this file to avoid both missing real design artifacts and creating "just in case" placeholders. Conditional artifacts are created only when the approved spec or design pass proves the trigger is real.

## When To Load
- Load when deciding whether to add `design/data-model.md`, `design/dependency-graph.md`, `design/contracts/`, `test-plan.md`, or `rollout.md`.
- Load when an existing conditional artifact looks like filler.
- Load when planning readiness depends on making a triggered validation, rollout, dependency, data, or contract concern explicit.

## Good Design-Session Outputs
- "Triggered: `design/data-model.md`, because the approved change adds persisted export-job state, terminal statuses, and retry visibility."
- "Not expected: `design/dependency-graph.md`, because package dependency direction remains unchanged and no coupling risk is introduced."
- "Triggered: `design/contracts/`, because OpenAPI request and response shapes change. This folder is design-only; `api/openapi/service.yaml` remains canonical."
- "Triggered: `rollout.md`, because the migration needs expand, backfill/verify, and contract timing that planning must preserve."

## Bad Design-Session Outputs
- "Create all conditional artifacts for completeness."
- "Skip `design/data-model.md`; the migration can be figured out during coding."
- "Put OpenAPI snippets in `design/contracts/` and treat them as the source of truth."
- "Create `test-plan.md` with generic unit/integration/e2e headings even though validation can fit in the later `plan.md`."

## Conditional Artifact Examples
- `design/data-model.md`: persisted state, schema, migration shape, cache contract, projections, replay behavior, data retention, or correctness-sensitive backfill.
- `design/dependency-graph.md`: package/module direction changes, generated-code dependency flow, new adapter boundary, circular-coupling risk, or source-of-truth ambiguity across packages.
- `design/contracts/`: changed REST resources, event payloads, generated contracts, or material internal interfaces that planning must preserve.
- `test-plan.md`: multi-layer validation obligations across contract tests, migration tests, reliability fail-paths, and e2e smoke checks.
- `rollout.md`: mixed-version compatibility, expand/backfill/verify/contract sequencing, operational failback, or deploy ordering that affects correctness.

## Blocked Handoff Examples
- A triggered artifact is required to make planning safe but is missing or only a placeholder.
- The task changes API behavior but neither `design/contracts/` nor the canonical contract authority is named.
- Data migration order affects runtime correctness, but `rollout.md` is not triggered or explicitly waived.
- A conditional artifact would need a final business or product decision not present in `spec.md`; route back to `specification`.

## Exa Source Links
Exa MCP search and fetch were attempted before writing these examples, but the provider returned a 402 credits-limit error. Treat these fallback-verified links as calibration only; repo-local files remain authoritative.

- Google Cloud Architecture Framework: https://docs.cloud.google.com/architecture/framework
- Google Cloud Architecture decision records: https://docs.cloud.google.com/architecture/architecture-decision-records
- Azure Well-Architected Framework: https://learn.microsoft.com/en-us/azure/well-architected/what-is-well-architected-framework
- arc42 documentation: https://docs.arc42.org/home/

