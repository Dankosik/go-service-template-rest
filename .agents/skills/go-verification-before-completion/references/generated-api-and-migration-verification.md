# Generated API And Migration Verification

## Load When

Load this when the claim depends on an OpenAPI spec, generated API or sqlc
output, a SQL query, or a migration.

## Decide

Use `go-coder`'s [generated-source reference](../../go-coder/references/generated-source-of-truth-and-drift.md)
to identify the target's canonical source, generator, and command side effects.
Select commands through [Validation Routing](../../../../docs/validation-routing.md).
Inspect their current implementation and actual exercised surfaces before
assigning these proof boundaries:

- Generated drift, runtime contract tests, lint, and spec validation do not
  establish compatibility with the previous API. That claim needs the matching
  breaking-change check against the accepted base contract.
- `make sqlc-check` catches generated output that no longer matches its queries,
  including output left behind after its sources were removed. Data-access tests
  pass over stale generated files, so they cannot stand in for it.
- A source migration rehearsal can prove apply, reapply, reversal, and reapply
  only when the target owns and actually executes those assertions against its
  current migration corpus. A production-image rehearsal adds evidence about
  the shipped migration entrypoint and runtime. Reuse existing rehearsal proof
  for the matching claim; do not infer either level from the target name.
- Neither proves that repository and domain code behave correctly over the new
  schema. That is a separate result from the tests covering the affected surface.
- A no-migrations, disabled-profile, or history-check exemption is not migration
  evidence. Confirm what the target actually examined before citing success.

## Reject

Reject a compile or a passing unit test as evidence about generated artifacts:
stale generated code can compile and pass tests regardless of lint exclusions.
Reject a drift check that
ended by modifying files as a clean result; the artifacts are reconciled only
once the check reruns without changing anything.

## Prove

Name the surface, the drift or rehearsal command and whether it altered files or
skipped, and — separately — the behavior tests that prove code works over that
contract or schema. A claim covering both needs both, quoted as two results.
