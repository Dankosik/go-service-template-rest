# Generated API And Migration Verification

## Load When

Load this when the claim depends on an OpenAPI spec, generated API or sqlc
output, a SQL query, or a migration.

## Decide

`go-coder`'s generated-source reference owns which file is canonical and which
command regenerates it, including the fact that a `*-check` target regenerates in
place and leaves the rewritten files in your tree. What matters for the claim is
the boundary of each proof:

- `make openapi-check` is drift, reference compile, runtime contract, lint, and
  validation. It says nothing about **compatibility with the previous spec** —
  that is `make openapi-breaking BASE_OPENAPI=…`, which errors out without the
  base. Pull-request CI extracts the event's exact base spec. Add it only when
  the claim is about not breaking existing clients.
- `make sqlc-check` catches generated output that no longer matches its queries,
  including output left behind after its sources were removed. Data-access tests
  pass over stale generated files, so they cannot stand in for it.
- The reversal rehearsal is not exclusive to `migration-validate`.
  `TestPostgresMigrateRepositorySourceRehearsal` in `./test` runs up, up again
  (asserting the second is a no-op), down to zero, and up again over the live
  `migrations/` corpus — and `test-integration` runs `./test/...` unfiltered, so a
  green integration run already covers reversal. What `migration-validate` adds is
  the production image: it runs that image's `/migrate` entrypoint against a
  disposable Compose Postgres and requires the container to reach `/health/ready`,
  report the expected version, and exit 0 on SIGTERM. Cite it for a
  *deployable-migration* claim, not merely a reversible-schema one.
- Neither proves that repository and domain code behave correctly over the new
  schema. That is a separate result from the tests covering the affected surface.
- The rehearsal skips itself when the repository owns no `migrations/` directory,
  and `migration-history-check` returns success without comparing anything while
  `scripts/profiles/` exists (it is authored in place here, so append-only history
  is a generated-service check). Confirm each one had something to examine before
  citing it.

## Reject

Reject a compile or a passing unit test as evidence about generated artifacts:
stale generated code compiles and its tests pass, and the linters never see it —
`.golangci.yml` excludes the generated paths outright. Reject a drift check that
ended by modifying files as a clean result; the artifacts are reconciled only
once the check reruns without changing anything.

## Prove

Name the surface, the drift or rehearsal command and whether it altered files or
skipped, and — separately — the behavior tests that prove code works over that
contract or schema. A claim covering both needs both, quoted as two results.
