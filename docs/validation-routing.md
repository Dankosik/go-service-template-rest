# Validation Routing

Load only the branch selected by the changed surface or intended claim. The
Makefile owns command composition; these files own agent-facing selection. The
full [command reference](build-test-and-development-commands.md) remains the
human-facing explanation.

| Changed surface or claim | Load | Primary proof |
| --- | --- | --- |
| Agent instructions, roles, skills, mirrors, or template propagation | [Instructions](validation/instructions.md) | `make template-owned-purity-check` |
| Ordinary Go behavior, formatting, analysis, or unit tests | [Go](validation/go.md) | start `make verify` after assembly; continue failed plans under the Evidence Contract |
| OpenAPI, protobuf, SQLC, or generated drift | [Generated Contracts](validation/generated.md) | matching `*-check` |
| PostgreSQL transactions, migrations, or integration semantics | [PostgreSQL](validation/postgres.md) | `REQUIRE_DOCKER=1 ALLOW_HEAVY=1 make test-integration-db` |
| Runtime image, container behavior, or migration rehearsal | [Containers](validation/containers.md) | `make runtime-image-build` |
| CI/CD, workflows, Dockerfile, or shell scripts | [Delivery](validation/delivery.md) | matching delivery leaf |
| Secrets, dependencies, Go or image vulnerability claims | [Security](validation/security.md) | matching security target |
| Latency, throughput, allocation, contention, or capacity | [Benchmarking](benchmarking.md) | workload-matched benchmark |

This router selects proof commands for final validation or a separate verification
request. [Implementation](spec-first-workflow/phases/implementation.md#feedback-during-coding)
owns the narrower coding-feedback allowance; a fast target is not
automatically eligible. Run the smallest
aggregate matching the final claim after all planned code is assembled. Missing
Docker or an external provider narrows the claim; it is not a passing skip.

`make plan` diagnoses the current worktree's selected surfaces, commands, and
not-applicable gates; it is not a required completion gate. `make verify` prints
and runs that non-overlapping plan and reuses an exact Git-common passing
receipt while resolved base, merge base, candidate, plan, execution inputs, and
environment remain unchanged. Heavy authorization, Docker, and binary checks
happen before execution; selected integration leaves force `REQUIRE_DOCKER=1`,
and a changed candidate cannot produce a receipt. `ALLOW_FULL=1 make check`
remains the explicit deterministic full-repository gate.

Each actual run also prints a persistent attempt record with the complete plan,
original candidate, execution environment, and each step's pending/running or
terminal state and duration. Failed or interrupted attempts are not passing
receipts. The [Evidence Contract](spec-first-workflow/shared/evidence-contract.md#execution-evidence)
owns continuation and scoped reuse after repair; `make verify` does not infer
cross-candidate dependency equivalence or automatically skip partial results.
Use Implementation's [Progress](spec-first-workflow/phases/implementation.md#progress)
method to supervise long-running steps and waits.

`*-fast` targets are available for standalone debugging and eligible repair
diagnostics during final validation. They refuse CI and local tool version drift. Final proof
uses the matching canonical leaf.

Final validation and blockers stay within the accepted delivery scope and its
required evidence. Unrelated or pre-existing defects are observations, not blockers,
unless the intended claim explicitly spans that broader surface.
