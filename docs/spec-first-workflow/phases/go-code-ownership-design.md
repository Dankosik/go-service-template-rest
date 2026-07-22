# Go Code / Ownership Design

Place the accepted mechanism in Go packages and files without changing behavior. Give each changed responsibility one clear, evidence-backed owner and make its affected execution paths reviewer-traceable.

## Read When

- Planning cannot name the owning package/file, dependency direction, generated/manual boundary, cleanup owner, or test owner.
- A substantial change could enlarge a mixed-responsibility file or introduce an interface/helper/dependency.
- Technical review found ownership or placement ambiguity.

## Inputs

- Ready spec and system/integration design when present.
- `docs/repo-architecture.md` and current package/file responsibilities.
- Relevant callers, siblings, composition root, generated sources, tests, and any replaced or compatibility paths.

## Method

Reconstruct the complete changed-responsibility set from current files and symbols, callers, siblings, composition roots, generated sources, tests, and replaced or compatibility paths. For each responsibility, select one clear implementation/source owner with current evidence and state why that owner stays or changes. Separately disposition its package/file placement, dependency/composition, generated authority, cleanup owner, and test/proof owner. Validate every selected owner against the actual Go import graph, `internal` visibility, generated/manual boundary, and acyclicity. Do not add a producer-owned interface used only to reverse an invalid dependency; repair the owner or dependency direction instead.

## Outputs

A compact ownership section in `design/overview.md` or `design/go-code-ownership.md`, grouped by affected responsibility:

- source: implementation/source owner, current file/symbol evidence, why that owner stays or changes, selected package/file placement, and what stays, moves, is added, or is removed; only when exact file selection depends on implementation-local facts, give the owning surface, deterministic placement rule, and inspection bounds instead;
- dependency/composition: dependency direction, composition boundary, and the owner and minimum required shape of each introduced or changed cross-package type, error, mapping, constructor, or exported symbol that planning would otherwise choose;
- authority: generated source of truth and its hand-written change or regeneration point;
- concrete types by default; when a present consumer must substitute implementations or direct coupling would violate dependency direction, use the smallest interface in the consumer package and name its composition-root wiring;
- cleanup: keep/split rationale plus the disposition of each replaced or compatibility path and every now-obsolete caller, wiring/registration, test, config, generated input/artifact, and doc; if retained, name the present need, owner, and removal condition;
- test and proof: test owner and proof entrypoint.

Add a file, package, or seam only when the selected responsibility owner cannot preserve the required responsibility, dependency direction, or generated/manual boundary in the current surface.

Keep symbols unexported and code with its current owner unless the selected responsibility or dependency direction requires a move. Prefer explicit control flow, the Go standard library, and established repository patterns. Expected future reuse, line count alone, or test convenience alone do not justify a new interface, package, helper, factory, or seam. Keep owner-specific behavior out of generic helper buckets, one-product factories, and speculative extension points.

## Review

Apply focused root self-review with system design before planning. Use independent [Technical Design Review](technical-design-review.md) only when the shared review trigger applies.

## Stop Rule

This phase is complete when accepted behavior is preserved, every changed responsibility has an evidence-backed owner and file placement or deterministic implementation-local rule, planning has no material ownership, dependency, generated/manual, or exported-surface decision left to make, and any triggered technical-design review has returned `PASS` or dispositioned `CONCERNS`. Reopen system design if the proposed ownership changes runtime behavior or source of truth; reopen specification if it changes scope or contract.
