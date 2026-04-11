# Design Overview

## Chosen Approach

This hardening change should make the template's extension rules executable and visible without inventing a fake business product.

The implementation should fix real drift first, then remove misleading examples and split ownership, then update docs/tests so future teams can follow one path:

```text
OpenAPI contract -> generated API -> app behavior/ports -> infra adapter -> bootstrap wiring -> layered tests -> targeted Make proof
```

## Artifact Index

- `component-map.md`: affected packages and docs.
- `sequence.md`: implementation/runtime sequences to preserve.
- `ownership-map.md`: source-of-truth and dependency direction decisions.
- `data-model.md`: persistence/sqlc/migration treatment.
- `contracts/http-security-and-generated-routes.md`: HTTP, auth, metrics, and generated-route policy.

## Design Principles

- Prefer source-of-truth fixes over explanatory comments.
- Keep composition in bootstrap, not in HTTP handlers.
- Keep interfaces near consumers unless they are genuinely shared domain contracts.
- Keep generated artifacts derived from their sources.
- Do not ship sample persistence as hidden business state.
- Do not imply auth/security behavior that does not exist.
- Keep docs terse but exact enough for clone-time decisions.

## Readiness Summary

The implementation plan is ready. The main caveat is SQLC proof: native `make sqlc-check` failed during review due local tool compilation, so the coding session should use Docker SQLC validation if native remains blocked.

