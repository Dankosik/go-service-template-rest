# Template Sync

This template is the single source of truth for the workflow instruction set. A repository derived from it adopts instruction changes by running a script, not by hand-merging documents and not by asking an agent to copy files.

## Read When

- Changing any instruction file and deciding whether derived repositories must receive that change.
- Recording something that is true of one service rather than of every service built from this template.
- Adopting the current instruction set in a derived repository, or explaining why one drifted.

## The Ownership Split

`template-owned.paths` is the manifest. Every path it lists is mirrored verbatim into a derived repository, including deletions inside listed directories. `.agents/roles` owns role semantics; `scripts/agent-roles-sync.sh` generates and validates the checked-in Codex, Claude, Qwen, Grok, Cursor, and OpenCode carriers before propagation. `.grok/agents` also keeps the handwritten `orchestrator` and `acceptance-unit-lead` primary-session agents. `.cursor/agents` keeps the handwritten `acceptance-unit-lead` Task agent. `.opencode/agents` keeps the handwritten `orchestrator` and `acceptance-unit-lead` agents. `.agents/codex-project.toml` owns portable Codex runtime defaults. Three target-local generated views are rebuilt after mirroring: `.claude/skills` and `.qwen/skills` expose the same canonical skill set, while `.codex/config.toml` is generated exactly from the portable runtime and role registry. Cursor, Grok, and OpenCode read `.agents/skills` directly. Machine-specific Codex settings belong only in user or system config.

A path may appear in the manifest only while it carries no repository-specific content: no service name, no module path, no deployment target, no owner, no service-specific invariant, and no initialization-profile marker. A profile marker means different derived repositories retain different content, which a verbatim mirror cannot preserve. That restriction is what makes mirroring safe — there is nothing in an owned file for a derived repository to lose.

The one target-local exception is a service capability skill under
`.agents/skills/<name>/` with a `.service-owned` marker. The sync
preserves that directory, requires a real `SKILL.md`, refuses an ownership
collision with a template skill of the same name, excludes it from the sync
write regardless of Git status, and regenerates its Claude and Qwen discovery
links locally. An unmarked target-only skill remains drift and is deleted by
apply.

These documents stay repository-owned and the sync never touches them:

| Repository-owned | Records |
| --- | --- |
| `README.md` | What this service is and who consumes it |
| `docs/repo-architecture.md` and `docs/architecture/` | This service's architecture selector, boundaries, runtime flows, system neighbors, and stable domain vocabulary |
| `docs/project-structure-and-module-organization.md` | This service's package layout |
| `docs/build-test-and-development-commands.md` | The commands this service actually has |
| `docs/ci-cd-production-ready.md` | This repository's setup, module path, and owners |
| `docs/railway-deployment-profile.md` | This service's deployment target |
| `test/README.md` | This service's integration-test topology and commands |
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

`--check` and `--apply` use the same committed `HEAD` snapshot and refuse dirty
template-owned source. Before inspecting a target, both validate the snapshot's
canonical skills, roles, role carriers, and Codex project view. `--check`
compares owned content, generated role carriers, Claude/Qwen skill links, and
the Codex project runtime and role registry directly, prints drift, and exits non-zero. It also requires every
repository-owned authority listed above except `README.md` and the template-only
onboarding document; it verifies their presence but never copies their
service-specific content. `--apply` mirrors the validated role carriers and
rebuilds only target-local generated views through template-owned helpers without depending on the
target `Makefile`, and removes the retired `.template-sync` file. Ignored and
untracked empty content does not enter that snapshot. Generated paths stay out
of the manifest. The result stays in the target working tree for ordinary
review and commit. A service-owned skill and its Claude/Qwen links retain their
existing Git status.

Portable validation invokes the synced helper scripts directly. A derived
repository's `Makefile` is repository-owned, so template Make targets are
convenience aliases for the template checkout, not propagated interfaces.

## Uncommitted Work

The sync never commits. Its result remains a normal reviewable working-tree
change in one target.

**Work outside the manifest and its generated views never blocks a sync and
is never touched.** A repository can carry any other work in progress and still
sync. The script never stashes, resets, cleans, stages, or commits.

**Work inside the manifest, generated `.claude/skills`, `.qwen/skills`, or
`.codex/config.toml` refuses that target, except a marked service-owned skill
and its matching harness links.** The sync neither mirrors nor stages those local
paths. For every other owned path, it cannot safely distinguish a concurrent
edit from its update, so it stops before writes:

```
   uncommitted changes inside the manifest:
      D .agents/skills/go-coder/SKILL.md
   refused: the sync would overwrite them. Commit or discard these paths yourself, then sync again
```

Deciding between them is a judgement about your own work, so the script leaves
it to you. Commit the change if it is wanted, discard it if it was scratch,
then sync again.

## What The Sync Refuses

Every deterministic refusal below is checked before the first write. An
unexpected filesystem or helper failure can leave sync-produced uncommitted
changes.

| Refusal | Why it matters |
| --- | --- |
| Manifest paths hold uncommitted changes | The mirror would destroy them |
| The template's own manifest is dirty | Targets must receive one committed, reviewable template revision |
| A manifest path contains a symlink | The copy could read or write outside the repository |
| `.claude/skills` or `.qwen/skills` is a symlink or contains a real file or directory | Generated skill links could overwrite content without a canonical owner |
| `.claude`, `.qwen`, or `.codex` has an unsafe path shape, or the Codex managed markers are malformed | A generated view could fail after manifest writes |
| Ignored content exists inside the template manifest or a target owned/generated path | It could leak into a target or be silently deleted |
| The committed canonical inputs and generated projections disagree | Apply would fail or create drift after target writes |
| A required repository-owned instruction is missing | Mirrored instructions would point at an absent local authority |
| A marked service-owned skill is malformed or collides with a template skill | The sync cannot preserve one unambiguous local owner |
| A manifest path is gitignored in the target | The mirror writes it, git refuses to track it, and drift never clears |
| Target has a directory where the template has a file, or the reverse | The copy cannot land |
| An owned path names the target's own module | The manifest is wrong, not the target |

One environment assumption is not checked. A target with `core.autocrlf`
enabled rewrites line endings on checkout, which reads as permanent drift; keep
derived repositories on LF.

## Changing The Manifest

Adding a path is a claim that the path is free of repository-specific content. Prove it before adding: a file that is byte-identical across the repositories derived from this template carries none, and a file that diverges between them carries some. Divergence caused by different template generations is not repository-specific content; divergence caused by a service naming itself is.

Removing a path stops propagation but leaves whatever each derived repository currently has. Delete the file through a sync first if it must disappear everywhere.
