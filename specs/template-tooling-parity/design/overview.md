# Template tooling parity design

status: ready

## Owners

- `template-owned.paths` remains the sole propagation manifest.
- `scripts/template-sync.sh` remains the sole check/apply engine.
- `make/template.mk` owns the standard Make implementation. Repository root
  `Makefile` selects `SERVICE_NAME`, includes the standard file, and may include
  repository-owned `make/service.mk` extensions.
- Portable scripts keep their current paths and are listed individually in the
  manifest so sibling service scripts are never deleted.
- `tools/go.mod` and `tools/go.sum` own the standard pinned tool set. Their module
  identity is repository-neutral and the initialized service retains the full
  tool set; optional runtime profiles do not prune development tools.
- Portable lint and generator configuration is profile-tolerant. Module identity
  is derived at invocation rather than stored as target-specific bytes.
- Shared Make and script gates are mirrored. Executable CI workflows remain
  repository-owned because a wrapper or job-topology change can rename required
  status contexts; workflow and Ruleset migration is an explicit per-repository
  delivery gate. CD and external activation remain repository-owned.

## Migration

The source template changes first. Existing services are separate acceptance
units: move only their service-specific Make extensions into `make/service.mk`,
record a complete `template.lock` when current evidence proves every field, then
run the committed source template's check/apply path. A checkout with dirty
portable paths or ambiguous profile evidence is blocked without writes.

Two local checkout copies of one origin are not two migration authorities. One
canonical branch is migrated and committed; other checkouts consume that Git
result after their unrelated work is reconciled.

## Rejected alternatives

- A second tooling-sync script duplicates preflight, ownership, and drift logic.
- Mirroring all of `scripts/` deletes service scripts.
- Overwriting legacy root Makefiles loses service-specific gates.
- Keeping profile-pruned standard commands prevents one command contract.
- A compatibility Makefile that silently runs old standard recipes is not parity.
