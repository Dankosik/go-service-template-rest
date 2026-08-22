# Security Validation

Select the command that matches the claim:

| Claim | Command |
| --- | --- |
| Go dependency vulnerability | `make govulncheck` |
| Go source security | `make gosec` and, when dependencies matter, `make govulncheck` |
| Current reviewable secret exposure | `make secret-scan BASE_REF=origin/main` |
| Full history secret exposure | `make secret-scan-history` |
| Runtime image vulnerabilities | `make container-security CONTAINER_IMAGE=<tag>` |

Security validation supplements the negative-path behavior proof at the trust
boundary; it does not replace it. Full-history proof applies only when the
intended claim explicitly spans repository history, such as main or release
validation.
