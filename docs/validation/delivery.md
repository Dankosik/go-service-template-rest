# Delivery Validation

Use `make delivery-quality` when workflows, tracked shell scripts, the
Dockerfile, or delivery policy changes. It owns actionlint, workflow-security
analysis, ShellCheck, and native Dockerfile checks.

Use `make ci-change-scope-check` when path classification changes. A release or
merge-readiness claim also needs the exact CI/release evidence named by the
delivery owner; local analyzers do not prove platform state.
