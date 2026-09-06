# Delivery Validation

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
