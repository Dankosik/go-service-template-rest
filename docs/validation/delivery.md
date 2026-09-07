# Delivery Validation

Select these procedures for an explicit verification requirement or a concrete
bounded diagnostic under the [Evidence Contract](../spec-first-workflow/shared/evidence-contract.md#required-and-optional-proof).
The domain label alone does not add local gates or authorize test infrastructure.
Existing CI and release requirements remain unchanged.

The canonical local `make actionlint` and `make shellcheck` leaves download the
pinned native release once into the Git-common tool cache and verify its SHA-256
and reported version. CI keeps the pinned read-only Docker fallback.
`actionlint-fast` and `shellcheck-fast` are standalone diagnostic commands,
outside ledger implementation, and refuse CI or version drift. Final ShellCheck receives only changed
shell files on diff-routed CI events.

Changes under `test/performance/` or to its evidence script use
`make performance-harness-check`. It runs metadata self-tests and pinned k6
inspection without starting a service or generating load.

A release or merge-readiness claim also needs the exact CI/release evidence
named by the delivery owner; local analyzers do not prove platform state.
