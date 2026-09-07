# Container Validation

Use only for a matching explicit verification requirement or bounded diagnostic
under the [Evidence Contract](../spec-first-workflow/shared/evidence-contract.md#required-and-optional-proof).
A changed file or available Docker daemon does not create a local image gate.

Build with `make runtime-image-build RUNTIME_IMAGE=service:ci`. A successful
image build is not a runtime observation; use `make runtime-image-check` with
the same tag when lifecycle behavior is explicitly required. Reuse that image
for `ALLOW_HEAVY=1 make migration-validate` and
`ALLOW_HEAVY=1 make container-security CONTAINER_IMAGE=service:ci` only when those
claims are required. Actual CI retains its existing gates and heavy authority.

Run only matching leaves. Preserve caches and owner cleanup; do not use no-cache
builds or broad pruning as iteration. An unavailable optional container check is
a disclosed gap, not a reason to provision or repair an environment before
local completion.
