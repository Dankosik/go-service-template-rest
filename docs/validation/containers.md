# Container Validation

Use `make runtime-image-build RUNTIME_IMAGE=service:ci` for runtime-image
behavior. Reuse that exact tag for `ALLOW_HEAVY=1 make migration-validate` and
`ALLOW_HEAVY=1 make container-security CONTAINER_IMAGE=service:ci` when those
claims apply. CI sets `CI=true`, which satisfies the heavy-target guard.

Run only the matching image, migration, or vulnerability leaf. Keep caches and
owner cleanup; do not use no-cache builds or broad pruning as an iteration
strategy.
