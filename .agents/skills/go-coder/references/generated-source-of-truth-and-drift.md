# Generated Source Of Truth And Drift

## Load When

Load this when the change touches an OpenAPI operation or schema, a SQL query or
migration, a `.proto` service, a generator config, or a drift-check failure.

## Decide

Reconstruct `canonical source -> generator configuration/directive -> command
-> generated output` from the target's contracts, configuration, generation
directives, and Makefile. Its [Repository
Architecture](../../../../docs/repo-architecture.md) owns generated/manual
boundaries. Change the canonical source, regenerate, and remove superseded
output.

- Inspect the selected command before execution: a drift check may regenerate
  in place and leave changed output even when it fails. Such a command is not a
  read-only review probe. Review any resulting hunks against the source delta
  before retaining them; an unsuccessful check is not consistency proof.
- Generator flags and output paths belong to the configuration or directive the
  target actually invokes. A portable path example cannot establish that owner.
- A successful no-source or disabled-capability branch proves no generated
  consistency. Check both source presence and orphaned output when a profile or
  contract is removed. An available Make target does not establish an active
  capability.
- Resolve decoding and validation behavior from the target's contract,
  generator options, and validator wiring before changing handwritten handlers.
- Keep regeneration scoped to the source you changed. Generated churn with no
  source change is a separate finding to report, not diff to absorb.

## Reject

Lint cannot establish source-to-output consistency, whether or not the target's
lint configuration includes generated files. A compiling generated file can
still be stale.

## Prove

- Select generated-contract proof through [Validation
  Routing](../../../../docs/validation-routing.md) and the [Evidence
  Contract](../../../../docs/spec-first-workflow/shared/evidence-contract.md).
  Run a separate check only for required final proof absent from the selected
  aggregate. During implementation, generate sources under the active workflow's
  coding-feedback boundary; generation does not require a drift-check gate.
- Inspect `git diff` over the generated path and keep only hunks that trace back
  to the source change you made.
- When a generated symbol disappears, prove in the same diff that its
  hand-written callers were migrated or removed.
