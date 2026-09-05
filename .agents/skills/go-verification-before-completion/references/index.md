# Reference Selector

Load at most one reference by default; use both only for independent proof pressures.

| Symptom | Load | Decision it sharpens |
| --- | --- | --- |
| A claim names a gate target, a scope, or readiness, and where that evidence stops is not obvious. | [claim-to-proof-mapping.md](claim-to-proof-mapping.md) | Bound the conclusion by what the gate actually ran, including runs that exit 0 having executed nothing. |
| The claim depends on an OpenAPI spec, generated API or sqlc output, a SQL query, or a migration. | [generated-api-and-migration-verification.md](generated-api-and-migration-verification.md) | Separate drift and rehearsal proof from behavior over the contract or schema. |
