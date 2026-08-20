# Security Validation

Select the command that matches the claim:

| Claim | Command |
| --- | --- |
| Go dependency vulnerability | `make govulncheck` |
| Go source security | `make gosec` or `make go-security` |
| Current reviewable secret exposure | `make secret-scan` |
| Full history secret exposure | `make secret-scan-history` |
| Runtime image vulnerabilities | `make container-security CONTAINER_IMAGE=<tag>` |

Security validation supplements the negative-path behavior proof at the trust
boundary; it does not replace it.
