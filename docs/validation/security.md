# Security Validation

Select the command that matches the claim:

| Claim | Command |
| --- | --- |
| Go dependency vulnerability | `make govulncheck` |
| Go source security | `make gosec` or `make go-security` |
| Current reviewable secret exposure | `bash scripts/ci/secret-scan.sh change origin/main` |
| Full history secret exposure | `bash scripts/ci/secret-scan.sh history` |
| Runtime image vulnerabilities | `make container-security CONTAINER_IMAGE=<tag>` |

Security validation supplements the negative-path behavior proof at the trust
boundary; it does not replace it. Full-history proof applies only when the
intended claim explicitly spans repository history, such as main, nightly, or
release validation.
