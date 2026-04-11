# Generated Source Of Truth And Drift

## Behavior Change Thesis
When loaded for generated-code or drift pressure, this file makes the model change the owning source and run the matching generator/check instead of hand-editing generated output or leaving source and artifacts half-updated.

## When To Load
Load this when work touches generated files, generation configs, OpenAPI, sqlc, mockgen, stringer, generated enum strings, generated mocks, or drift-check failures.

## Decision Rubric
- Identify the source of truth before editing: OpenAPI spec/config, SQL migrations or queries, Go interfaces with `//go:generate`, enum source, or generator config.
- Edit generated files only as the output of the generator, not as the primary source.
- Keep source and generated artifacts in the same diff when the repository tracks generated output.
- Remove stale generated files when the source no longer owns them.
- Run the narrow generator/check for the touched surface before broader tests.
- If generated drift appears outside the approved task, stop and report it instead of mixing unrelated regeneration into the implementation.

## Imitate
For OpenAPI server bindings, change the contract or generator config, then regenerate and check.

```text
Source: api/openapi/service.yaml or internal/api/oapi-codegen.yaml
Generated: internal/api/openapi.gen.go
Proof: make openapi-check
```

For sqlc, change query/schema sources, then regenerate and check the generated package.

```text
Source: env/migrations/*.up.sql or internal/infra/postgres/queries/*.sql
Generated: internal/infra/postgres/sqlcgen/*
Proof: make sqlc-check
```

For mocks or enum stringers, update the owning Go source or directive, then run the matching drift check.

```text
Mock proof: make mocks-drift-check
Enum proof: make stringer-drift-check
```

## Reject
Reject direct generated edits as the primary fix.

```go
// internal/api/openapi.gen.go
func (c *Client) NewMethod(...) { // hand-written patch
	// ...
}
```

Reject source-only changes that leave tracked generated output stale.

```text
Changed: internal/infra/postgres/queries/ping_history.sql
Missing: internal/infra/postgres/sqlcgen/ping_history.sql.go regeneration
```

Reject broad regeneration that hides unrelated drift.

```text
Task: update one generated mock
Action: run all generators and commit unrelated OpenAPI/sqlc changes
```

## Agent Traps
- Treating "DO NOT EDIT" as a comment that only applies to humans.
- Fixing a compile error in generated code instead of finding the schema, query, directive, or generator config that produced it.
- Running `go test ./...` and forgetting the drift check that proves generated artifacts are in sync.
- Keeping a stale generated file after removing or renaming the source query or enum.
- Regenerating everything and accepting unrelated generated churn without a source explanation.
- Editing runtime adapter code when the real contract change belongs in OpenAPI or sqlc source.

## Validation Shape
- OpenAPI contract or generated API changes: `make openapi-check`.
- SQL query, schema, or sqlc output changes: `make sqlc-check`.
- Mock changes: `make mocks-drift-check`.
- Stringer enum output changes: `make stringer-drift-check`.
- After generation, inspect `git diff` and keep only generated changes that trace back to an approved source change.
