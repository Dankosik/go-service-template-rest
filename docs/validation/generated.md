# Generated Contract Validation

Edit the canonical source and regenerate. Map the required proof to these
commands through Validation Routing and the Evidence Contract; the table does
not add a separate run when the selected `make verify` plan covers that proof:

| Authority | Generate | Prove |
| --- | --- | --- |
| OpenAPI | `make openapi-generate` | `make openapi-check` |
| Protobuf | `make proto-generate` | `make proto-check` |
| SQLC query/schema sources | `make sqlc-generate` | `make sqlc-check` |

Compatibility claims also run the matching breaking-change target against a
readable base. Generated files are evidence, never the edit owner.
