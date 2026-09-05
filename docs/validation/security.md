# Security Validation

Select the command that matches the claim:

| Claim | Command |
| --- | --- |
| Go dependency vulnerability | `make govulncheck` |
| Go source security | `make gosec` and, when dependencies matter, `make govulncheck` |
| Current reviewable secret exposure | `make secret-scan BASE_REF=origin/main` |
| Full history secret exposure | `ALLOW_HEAVY=1 make secret-scan-history` |
| Runtime image vulnerabilities | `make container-security CONTAINER_IMAGE=<tag>` |

Security validation supplements the negative-path behavior proof at the trust
boundary; it does not replace it. Full-history proof applies only when the
intended claim explicitly spans repository history, such as main or release
validation.

PR lint includes `gosec` in the shared golangci-lint package load. The standalone
PR scan remains non-blocking parity telemetry until enough runs establish equal
findings; main, schedule, and release keep the standalone blocking scan.
