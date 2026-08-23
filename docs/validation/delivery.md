# Delivery Validation

During local iteration, `make actionlint-fast` and `make shellcheck-fast` use
native binaries only when their versions exactly match the repository pins.
They refuse CI and version drift. Use the affected final leaf: `make actionlint`
for workflows, `make shellcheck` for tracked shell, or `make dockerfile-check`
for the Dockerfile.

A release or merge-readiness claim also needs the exact CI/release evidence
named by the delivery owner; local analyzers do not prove platform state.
