# Reference Selector

Load at most one reference by default; use more only for independent proof pressures.

| Symptom | Load | Decision it sharpens |
| --- | --- | --- |
| “Fixed”, “green”, “ready”, test, lint, build, race, package, or repo claim is ambiguous. | [claim-to-proof-mapping.md](claim-to-proof-mapping.md) | Select the narrowest sufficient proof without overclaiming. |
| OpenAPI, generated API, sqlc, query, or migration changed. | [generated-api-and-migration-verification.md](generated-api-and-migration-verification.md) | Add authoritative drift or migration proof. |
| An agent, tool, CI snippet, or prior session says work is done. | [delegated-work-verification.md](delegated-work-verification.md) | Rebind the claim to current workspace evidence. |
| Proof failed, skipped, is absent, or is weaker than the claim. | [failure-and-gap-reporting.md](failure-and-gap-reporting.md) | Report the gap and next proving action without optimistic wording. |
