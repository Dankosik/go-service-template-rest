# Validation Routing

Use this router for mixed surfaces or specialized verification. [AGENTS.md](../AGENTS.md#validation-budget)
owns ordinary local completion; load the [Evidence Contract](spec-first-workflow/shared/evidence-contract.md)
for claim scope, reuse, additional proof, or infrastructure gaps.
This router selects existing commands; a file
path, domain label, or available command does not create a local acceptance gate.
The Makefile owns command composition. The full [command
reference](build-test-and-development-commands.md) remains the human-facing
explanation.

## Ordinary Local Completion

| Changed surface | Load | Local validation |
| --- | --- | --- |
| Ordinary Go behavior or unit tests | [Go](validation/go.md) | matching build and relevant unit tests after assembly |
| Agent instructions, roles, skills, mirrors, or template propagation | [Instructions](validation/instructions.md) | static consistency review and matching existing structural check |

[Implementation](spec-first-workflow/phases/implementation.md#feedback-during-coding)
owns the narrower coding-feedback allowance. At final validation, run the local
checks and only genuinely required additions. Stop when that boundary passes;
unrun optional integration, image, provider, or performance checks do not block
local completion. Missing required build or unit-test execution is not a pass.

## Explicit Verification And External Gates

Load a branch below only for a matching verification requirement or diagnostic
question, within the Evidence Contract's scope and infrastructure limits.
Regenerating changed contracts remains implementation work; this table does not
make every available drift or runtime check mandatory locally.

| Required claim | Load | Existing command |
| --- | --- | --- |
| OpenAPI, protobuf, SQLC, or generated drift | [Generated Contracts](validation/generated.md) | matching `*-check` |
| PostgreSQL transactions, migrations, or integration semantics | [PostgreSQL](validation/postgres.md) | `REQUIRE_DOCKER=1 ALLOW_HEAVY=1 make test-integration-db` |
| Runtime image, container behavior, or migration rehearsal | [Containers](validation/containers.md) | `make runtime-image-build` and the required scenario |
| CI/CD, workflows, Dockerfile, or shell-script validation | [Delivery](validation/delivery.md) | matching delivery leaf |
| Secrets, dependencies, Go or image vulnerability verification | [Security](validation/security.md) | matching security target |
| Measured latency, throughput, allocation, contention, or capacity | [Benchmarking](benchmarking.md) | workload-matched benchmark |

Existing CI/release gates keep their own admission scope. Do not change their
classifier or reproduce them locally merely to finish development. An explicit
request to obtain green CI, verify a runtime scenario, or perform a release
still requires that result; local completion alone does not complete it.

`make verify` is an explicitly selected expanded, surface-aware verification
route, not the ordinary local completion command. Its plan may include Docker,
integration, migration, and image checks. Heavy steps are CI-owned: CI runs them
on every surface that selects them, so the plan lists them under `ci-owned` and
a local run leaves them to CI; `ALLOW_HEAVY=1` keeps them local. `make plan`
diagnoses that selection; it is not a gate and does not authorize the plan. Prefer a matching canonical
leaf when an aggregate would add irrelevant work.

For an expanded run, `make verify` reuses an exact Git-common passing receipt
while resolved base, merge base, candidate, plan, execution inputs, and
environment remain unchanged. Docker and binary checks for the local steps
happen before execution; selected integration leaves force `REQUIRE_DOCKER=1`.
While CI-owned steps remain the receipt records `status: partially_verified`,
lists them under `ci_owned`, and names CI as the next owner; a route with only
CI-owned steps runs nothing locally.
A changed candidate cannot produce a receipt. `ALLOW_FULL=1 make check` remains
the explicit deterministic full-repository gate, never a default follow-up.

Each actual expanded run prints a persistent attempt record with its plan,
candidate, environment, step states, and durations. Failed or interrupted
attempts are not passing receipts. The [Evidence
Contract](spec-first-workflow/shared/evidence-contract.md#execution-evidence)
owns continuation and scoped reuse; `make verify` does not infer cross-candidate
equivalence. Use Implementation's [Progress](spec-first-workflow/phases/implementation/progress.md)
method for genuinely required long-running work.

`*-fast` targets remain standalone or focused repair diagnostics; they refuse
CI and local tool version drift. They do not establish an explicitly required
canonical check. Unrelated or pre-existing defects remain observations unless
the accepted task actually spans them.
