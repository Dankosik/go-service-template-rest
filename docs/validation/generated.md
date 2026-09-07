# Generated Contract Validation

Edit the canonical source and regenerate as implementation work. For an
explicit generated-drift or compatibility verification requirement, select an
existing command under [Validation Routing](../validation-routing.md) and the
[Evidence Contract](../spec-first-workflow/shared/evidence-contract.md#required-and-optional-proof).
The table is a method reference, not an extra local gate or a reason to rerun a
check already covered by valid evidence:

| Authority | Generate | Prove |
| --- | --- | --- |
| OpenAPI | `make openapi-generate` | `make openapi-check` |
| Protobuf | `make proto-generate` | `make proto-check` |
| SQLC query/schema sources | `make sqlc-generate` | `make sqlc-check` |

Explicit compatibility verification also uses the matching breaking-change
target against a readable base. Generated files are evidence, never the edit owner.
