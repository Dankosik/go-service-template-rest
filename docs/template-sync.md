# Template Sync

This template is the single source of truth for the workflow instruction set. A repository derived from it adopts instruction changes by running a script, not by hand-merging documents and not by asking an agent to copy files.

## Read When

- Changing any instruction file and deciding whether derived repositories must receive that change.
- Recording something that is true of one service rather than of every service built from this template.
- Adopting the current instruction set in a derived repository, or explaining why one drifted.

## The Ownership Split

`template-owned.paths` is the manifest. Every path it lists is mirrored verbatim into a derived repository, including deletions inside listed directories. Two generated views are also owned by the sync: `.claude/skills` exposes the canonical skill set, and the marked role-registry block in `.codex/config.toml` exposes template-owned Codex agents while preserving every repository-specific table outside that block.

A path may appear in the manifest only while it carries no repository-specific content: no service name, no module path, no deployment target, no owner, no service-specific invariant, and no initialization-profile marker. A profile marker means different derived repositories retain different content, which a verbatim mirror cannot preserve. That restriction is what makes mirroring safe — there is nothing in an owned file for a derived repository to lose.

These documents stay repository-owned and the sync never touches them:

| Repository-owned | Records |
| --- | --- |
| `README.md` | What this service is and who consumes it |
| `docs/repo-architecture.md` | This service's boundaries, ownership, interaction style, system neighbors, and stable domain vocabulary |
| `docs/project-structure-and-module-organization.md` | This service's package layout |
| `docs/build-test-and-development-commands.md` | The commands this service actually has |
| `docs/ci-cd-production-ready.md` | This repository's setup, module path, and owners |
| `docs/railway-deployment-profile.md` | This service's deployment target |
| `docs/first-production-feature.md` | Template-only onboarding; not shipped to services |

Record a service-specific decision in one of those, or in a task-local artifact.
Never in an owned path. `make template-owned-purity-check` validates safe
manifest paths, existing non-empty owners, non-overlap, repository-owned
exclusions, absence of profile markers, and propagation of the sync mechanism.
Broader content portability remains a review responsibility; the actual sync
independently refuses an owned path that names the target module.

## Adopting Instruction Changes

Run the script from the template checkout, never a potentially stale target
copy. From the derived repository:

```
bash ../go-service-template-rest/scripts/template-sync.sh \
  --check --from ../go-service-template-rest --repo .
bash ../go-service-template-rest/scripts/template-sync.sh \
  --apply --from ../go-service-template-rest --repo .
```

`--check` compares owned content, generated Claude skill links, and the Codex
role registry directly, prints drift, and exits non-zero. `--apply` mirrors the
exact committed `HEAD`, rebuilds both generated views through template-owned
helpers without depending on the target `Makefile`, and removes the retired
`.template-sync` file. Ignored and untracked empty content does not enter that
snapshot. Generated paths stay out of the manifest, but the sync commits the
ones it changed instead of leaving the target dirty behind it.

To fan out from this template to several local checkouts in one run:

```
make template-sync-all TARGETS="../service-a ../service-b"
```

## Uncommitted Work

The sync commits only what it produced. It never commits work it found, because a commit is what reaches a release: unfinished work swept into a sync commit becomes deployable without anyone deciding that it should be.

**Work outside the manifest and its two generated views never blocks a sync and
is never touched.** A repository can carry any other work in progress and still
sync. The script stages explicit pathspecs and never stashes, resets, cleans, or
checks out over a change.

**Work inside the manifest, generated `.claude/skills`, or
`.codex/config.toml` refuses that target.** The sync cannot safely distinguish a
concurrent config edit from its managed-block update, so it stops before writes:

```
   uncommitted changes inside the manifest:
      D .agents/skills/go-coder/SKILL.md
   refused: the sync would overwrite them. Commit or discard these paths yourself, then sync again
```

Deciding between them is a judgement about your own work, so the script leaves it to you. Commit the change if it is wanted, discard it if it was scratch, then sync again. Other targets in the same run continue; the run exits non-zero and reports how many were refused.

Because the manifest is clean before the mirror runs, everything the sync stages was produced by the sync. That is what makes the resulting commit reviewable, and `git revert` undoes it exactly.

## What The Sync Refuses

Every refusal happens before the first write, so a refused target is left exactly as it was found.

| Refusal | Why it matters |
| --- | --- |
| Manifest paths hold uncommitted changes | The mirror would destroy them |
| The template's own manifest is dirty | Targets must receive one committed, reviewable template revision |
| A manifest path contains a symlink | The copy could read or write outside the repository |
| `.claude/skills` is a symlink or contains a real directory | Generated skill links could overwrite content without a canonical owner |
| Ignored content exists inside the template or target manifest | It could leak into a target or be silently deleted |
| Detached HEAD with commits enabled | A sync commit would belong to no branch; `--no-commit` remains safe |
| A manifest path is gitignored in the target | The mirror writes it, git refuses to track it, and drift never clears |
| Target has a directory where the template has a file, or the reverse | The copy cannot land |
| An owned path names the target's own module | The manifest is wrong, not the target |

Two environment assumptions are not checked. A target with `core.autocrlf` enabled rewrites line endings on checkout, which reads as permanent drift; keep derived repositories on LF. And `TARGETS` is word-split by make, so a checkout path containing spaces needs `--targets` on the script directly.

## Changing The Manifest

Adding a path is a claim that the path is free of repository-specific content. Prove it before adding: a file that is byte-identical across the repositories derived from this template carries none, and a file that diverges between them carries some. Divergence caused by different template generations is not repository-specific content; divergence caused by a service naming itself is.

Removing a path stops propagation but leaves whatever each derived repository currently has. Delete the file through a sync first if it must disappear everywhere.
