# Go Code / Ownership Design

Give each changed responsibility one clear, evidence-backed owner, then place the accepted mechanism in Go packages and files without changing behavior. Make every affected execution path reviewer-traceable.

## Read When

- Planning cannot name the owning package/file, dependency direction, generated/manual boundary, cleanup owner, or test owner.
- A substantial change could enlarge a mixed-responsibility file or introduce an interface/helper/dependency.
- Technical review found ownership or placement ambiguity.

## Inputs

- Ready spec and system/integration design when present.
- `docs/repo-architecture.md` and current package/file responsibilities.

## Method

Reconstruct the complete changed-responsibility set from current files and symbols, callers, siblings, composition roots, generated sources, tests, and replaced or compatibility paths. Build the per-responsibility ownership record below from that evidence. Validate every recorded decision against the actual Go import graph, `internal` visibility, generated/manual boundary, and acyclicity.

## Outputs

A compact ownership section in `design/overview.md` or `design/go-code-ownership.md`, grouped by affected responsibility:

- owner/placement: implementation owner; current file/symbol evidence; why that owner stays or changes; when a real placement fork exists, each viable alternative owner/location and why it is rejected; exact package/file placement; and what stays, moves, is added, or is removed. Only when exact file selection depends on implementation-local facts, give the owning surface, deterministic placement rule, and inspection bounds instead;
- dependency/composition: dependency direction, composition boundary, and the owner and minimum required shape of each introduced or changed cross-package type, error, mapping, constructor, or exported symbol that planning would otherwise choose;
- authority: generated source of truth and its hand-written change or regeneration point;
- concrete types by default; when a present consumer must substitute implementations or direct coupling would violate dependency direction, use the smallest interface in the consumer package and name its composition-root wiring;
- cleanup: keep/split rationale plus the disposition of each replaced or compatibility path and every now-obsolete caller, wiring/registration, test, config, generated input/artifact, and doc; if retained, name the present need, owner, and removal condition;
- test and proof: test owner and proof entrypoint.

Keep owner-specific behavior with its current owner and symbols unexported. Add the smallest new surface or seam only when a present responsibility cannot remain there without violating required dependency direction or the generated/manual boundary. Prefer explicit control flow, the Go standard library, and established repository patterns. Expected future reuse, line count, test convenience, generic helper naming, and one-product factories do not meet that admission rule.

## Review

Apply focused root self-review with system design before planning. Use independent [Technical Design Review](technical-design-review.md) only when the shared review trigger applies.

## Stop Rule

This phase is complete when accepted behavior is preserved and every changed responsibility has an evidence-backed owner, package/file placement or deterministic implementation-local rule, dependency/composition disposition, generated/manual authority and hand-written change or regeneration point, cleanup disposition, and test/proof owner and entrypoint. The resulting import graph must be validated as acyclic, planning must have no material ownership, placement, dependency, generated/manual, cleanup, test/proof, or exported-surface decision left to make, and any triggered technical-design review must have returned `PASS` or dispositioned `CONCERNS`. Reopen system design only when placement cannot preserve the accepted mechanism, runtime behavior, or source of truth; reopen specification only when placement cannot preserve scope or contract.
