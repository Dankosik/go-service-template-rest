# Validation Routing

Load only the branch selected by the changed surface or intended claim. The
Makefile owns command composition; these files own agent-facing selection. The
full [command reference](build-test-and-development-commands.md) remains the
human-facing explanation.

| Changed surface or claim | Load | Primary proof |
| --- | --- | --- |
| Agent instructions, roles, skills, mirrors, or template propagation | [Instructions](validation/instructions.md) | `make template-owned-purity-check` |
| Ordinary Go behavior, formatting, analysis, or unit tests | [Go](validation/go.md) | focused `go test` while iterating; `make unit-check` once per unit; `make check` once on the integrated tree |
| OpenAPI, protobuf, SQLC, or generated drift | [Generated Contracts](validation/generated.md) | matching `*-check` |
| PostgreSQL transactions, migrations, or integration semantics | [PostgreSQL](validation/postgres.md) | `REQUIRE_DOCKER=1 ALLOW_HEAVY=1 make test-integration` |
| Runtime image, container behavior, or migration rehearsal | [Containers](validation/containers.md) | `make runtime-image-build` |
| CI/CD, workflows, Dockerfile, or shell scripts | [Delivery](validation/delivery.md) | matching delivery leaf |
| Secrets, dependencies, Go or image vulnerability claims | [Security](validation/security.md) | matching security target |
| Latency, throughput, allocation, contention, or capacity | [Benchmarking](benchmarking.md) | workload-matched benchmark |

Start with the focused target that can falsify the change. The table names each
leaf's aggregate command; run that aggregate only when the completion claim
spans it. Missing Docker or an external provider narrows the claim; it does not
become a passing skip.

`*-fast` targets are local iteration signals. They refuse CI and local tool
version drift; run the matching canonical leaf for final proof.

Validation and blockers stay within the fixed accepted unit and its required
evidence. Unrelated or pre-existing defects are observations, not blockers,
unless the intended claim explicitly spans that broader surface.
