# Spec Handoff To Technical Design

Load this file when a non-trivial `spec.md` is about to hand off to `technical design`, or when a draft starts to absorb component maps, sequences, or ownership details.

The goal is a stable `spec.md -> design/` boundary: design can derive the task-local bundle without reopening core problem framing.

## Good: Handoff-Ready Spec Snippet

```markdown
## Decisions
- Runtime token reload preserves the existing authentication contract.
- The configuration source remains the source of truth.
- Failed reloads keep the last known-good token set active and surface degraded operator state.

## Plan Summary / Link
- Technical design must derive component ownership, reload sequence, and observability placement from these decisions before planning starts.
```

Why this works: the spec fixes the behavior and boundary, then points the next phase at design-owned work.

## Bad: Design Smuggled Into Spec

```markdown
## Decisions
- Add `ReloadCoordinator` in `internal/auth/reload.go`.
- The handler calls config, then auth, then metrics, then logger.
- Implementation tasks:
  - Create coordinator.
  - Add tests.
  - Update docs.
```

Why this fails: package ownership, runtime sequence, and task breakdown belong in `design/`, `plan.md`, and `tasks.md`.

## Good: Explicit Reopen Condition

```markdown
## Open Questions / Assumptions
- [reopen_spec_if_false] If technical design finds that config is not the sole source of truth for admin tokens, return to specification before planning.
```

Why this works: design has a clear stop rule instead of silently changing the spec's core decision.

## Foreign-Template Translation Examples

| Foreign section | Repo-native translation |
|---|---|
| "Architecture" | Move component and sequence detail to `design/overview.md`, `design/component-map.md`, and `design/sequence.md`. |
| "System ownership" | Move ownership boundaries to `design/ownership-map.md`; keep only source-of-truth decisions in `spec.md`. |
| "Implementation milestones" | Move execution order to `plan.md` and `tasks.md`; keep only `Plan Summary / Link` in `spec.md`. |
| "Detailed API contract" | If the API shape is canonical elsewhere, link it; if task-local design context is needed, use `design/contracts/`. |

## Exa / External Source Links

Exa MCP was attempted before authoring (`web_search_exa` and `web_fetch_exa`) but returned a 402 credits-limit error. The links below were gathered with browser fallback and are calibration only; this repository's artifact model decides the handoff.

- IEEE SA, "IEEE 830-1984: IEEE Guide for Software Requirements Specifications": https://standards.ieee.org/ieee/830/1220/
- NASA, "4.2 Technical Requirements Definition": https://www.nasa.gov/reference/4-2-technical-requirements-definition/
- IREB CPRE Online Glossary: https://cpre.ireb.org/en/downloads-and-resources/glossary
