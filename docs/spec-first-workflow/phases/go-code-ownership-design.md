# Go Code / Ownership Design

Place the accepted mechanism into clear Go package and file responsibilities without changing behavior.

## Read When

- Planning cannot name the owning package/file, dependency direction, generated/manual boundary, cleanup owner, or test owner.
- A substantial change could enlarge a mixed-responsibility file or introduce an interface/helper/dependency.
- Technical review found ownership or placement ambiguity.

## Inputs

- Ready spec and system/integration design when present.
- `docs/repo-architecture.md` and current package/file responsibilities.
- Relevant callers, siblings, composition root, generated sources, tests, and legacy path.

## Outputs

A compact ownership section in `design/overview.md` or `design/go-code-ownership.md` covering:

- owner package/file for each responsibility;
- dependency direction and composition boundary;
- generated versus hand-written authority;
- concrete type or narrow interface choice;
- new file/seam only when it creates a stable responsibility;
- replaced code and adjacent test/config/doc cleanup;
- test owner and proof entrypoint.

Inspect the current owner before naming a new one. Prefer concrete types, explicit control flow, same-package focused seams, current Go stdlib, and established repository patterns. Reject one-implementation interfaces, generic helper buckets, factories with one product, and speculative extension points unless a present boundary requires them.

## Review

For structured or orchestrated work, use [Technical Design Review](technical-design-review.md) with system design before planning. It is an internal checkpoint: the owning root repairs and re-reviews in the same root session. Direct work uses it only when the user or risk requires it. The smoke test is simple: can planning name files, owner, cleanup, tests, and proof without making another design choice?

## Stop Rule

Continue when placement preserves the accepted behavior and planning can task it directly. Reopen system design if the proposed ownership changes runtime behavior or source of truth; reopen specification if it changes scope or contract.
