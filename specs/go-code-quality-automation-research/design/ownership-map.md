# Ownership Map

- `Makefile` owns public local command names and composite targets.
- `scripts/dev/docker-tooling.sh` owns zero-setup Docker implementation details.
- `scripts/ci/required-guardrails-check.sh` owns repository guardrail policy and must remain runnable in CI/native and Docker-wrapper modes.
- `.golangci.yml` owns default lint policy.
- `.github/workflows/ci.yml` and `.github/workflows/nightly.yml` own GitHub gate placement.
- `.github/pull_request_template.md`, `CONTRIBUTING.md`, and `docs/build-test-and-development-commands.md` own user-facing workflow instructions.
- Package-local Go tests own fuzz/goleak coverage; no cross-package helper or generated artifact is expected.
