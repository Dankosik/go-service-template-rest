# Go Code / Ownership Design

Work as the future implementer without writing the implementation: mentally
rehearse the concrete Go edit far enough to give each changed responsibility
one clear, evidence-backed owner and exact repository-relative directory,
package, and file placement. Produce an implementation-ready placement
blueprint that preserves accepted behavior and system decisions. Implementation
may choose ordinary local syntax, but it must not have to invent a location,
file purpose, declaration owner, dependency, composition seam, lifecycle owner,
or proof location. Make every affected execution path reviewer-traceable.

## Read When

- Planning cannot name the owning package/file, declaration split and
  visibility, dependency direction, generated/manual boundary, cleanup owner,
  or test owner.
- A change adds or materially changes code in a package with multiple lifecycle stages, audiences, authorities, or operator flows, unless current file reasons make placement mechanical.
- One policy gains a second live representation, or a new interface, helper, or dependency has more than one evidence-backed present owner.
- Technical review found ownership or placement ambiguity.

## Inputs

- Ready spec and system/integration design when present.
- `docs/repo-architecture.md` and current package/file responsibilities.

## Method

Start from every selected system decision that enters Go. Decompose its Go manifestations — component responsibility, boundary adapter, transformation, lifecycle or sequence edge, failure/recovery path, operator path, composition point, authority change, and proof carrier — into the complete changed-responsibility set. Add each required responsibility exactly once and give it one owner; one system decision may produce several responsibilities. Reconcile that set against current files and symbols, callers, siblings, composition roots, generated sources, tests, and replaced or compatibility paths.

After that set is concrete, identify each non-trivial capability whose
implementation source is not mechanically fixed by accepted System Design or
current repository authority. When repository reuse, the standard library, a
native capability, an existing or external dependency, generated source, and a
custom implementation remain materially different ways to realize the same
accepted mechanism, suspend ownership synthesis and apply Research's [Solution
Discovery Evidence](research-branches.md#solution-discovery-evidence) as a
supporting step. Resume here with its candidate evidence and select the
implementation source only when behavior and the system mechanism remain
unchanged; otherwise reopen System Design. A retained branch, copied algorithm,
or preference for no new dependency is evidence to inspect, not closure. Admit
custom code only after current evidence eliminates the relevant higher reuse
rungs.

Reconcile the design in both directions. The responsibility map proves that every changed responsibility has one owner and placement; the inverse file map proves that every added or materially changed Go file has one present reason to exist. Follow each in-scope normal, failure/recovery, and operator path from composition through owner to proof so file boundaries keep owner transitions traceable. Replay each accepted policy change against the proposed owners: it has one semantic edit site plus only boundary-forced representations with executable parity proof. Validate both maps against the actual Go import graph, `internal` visibility, generated/manual boundary, and acyclicity.

Rehearse the edit from composition root through caller, owner, wiring, and proof.
For each responsibility, identify the exact existing file to keep, change,
move, or remove, or the exact new directory/package and filename to add; state
why the current owner remains valid or why a new location is required; and name
the owner-bearing declarations only far enough to prove the file split,
visibility, dependency direction, and material state, resource, cancellation,
or lifecycle ownership. Reuse current declarations and repository patterns
before adding a surface. Do not write pseudocode, function bodies, local
variables, statement-level control flow, or mechanically implied private
helpers. Record a private helper only when it owns policy, state, a lifecycle
edge, or another responsibility that could plausibly land elsewhere. If the
rehearsed edit still permits two materially different locations or file shapes,
the design is not closed.

When the accepted change must edit a file that fails the inverse file map, make
the smallest behavior-preserving split, move, or deletion needed to leave that
touched surface coherent part of the ownership result. Include only the named
responsibilities and the callers, tests, generated or manual companions, and
documentation required to move and prove them; unrelated cleanup remains an
observation. Record the preserved behavior and proof so Planning can keep the
restructure with its obligation task or route a valid enabling change.

## Outputs

A compact ownership section in `design/overview.md` or `design/go-code-ownership.md` containing two reconciled maps:

- responsibility map — for each changed responsibility, including its affected execution paths: current file/symbol evidence; selected owner, why it stays or changes, and exact repository-relative directory/package/file action (`keep`, `add`, `move`, or `remove`); dependency direction, composition boundary, and the owner and minimum required shape of each cross-package type, error, mapping, constructor, or exported symbol; for each non-mechanical implementation-source choice, its selected reuse rung and evidence locator, exact dependency/version or upstream generation identity, each viable rejected source and why it loses, behavior/parity proof owner, and upgrade or reopen condition; generated source of truth and its hand-written change or regeneration point; replacement cleanup; and test/proof owner, entrypoint, and fixture or corpus placement. When a real placement fork exists, name each viable alternative and why it loses. Only when exact file selection depends on implementation-local facts, give the owning surface, deterministic placement rule, and inspection bounds instead;
- file map — for every added or materially changed Go file: its repository-relative path, changed responsibilities, and one present reason to exist under [File naming and granularity](../../project-structure-and-module-organization.md#4-file-naming-and-granularity); the placement-relevant declaration actions and visibility; its role in each affected call path; material state, resource, cancellation, lifecycle, and error ownership; allowed dependencies and forbidden responsibilities; and any required co-location rationale or `doc.go` decision.

The maps describe the smallest coherent target ownership and placement, not a
draft implementation. Planning may turn that placement into ordered work, and
Implementation may write the code, without either phase choosing a different
directory, package, file, owner, surface, or file responsibility.

Fix one placement from current evidence instead of deferring alternatives, but
do not present it as infallible. Name the concrete code-shape evidence that
would invalidate the file map, such as independently changing responsibilities
or an ownership transition that becomes hard to trace. [Implementation](implementation.md#execute)
owns adapting the placement or reopening this decision when that evidence
appears in the real code.

Cleanup covers every replaced or compatibility path and now-obsolete caller,
wiring or registration, test, configuration, generated input or artifact, and
document. A retained path names its present need, owner, and removal condition.

Keep owner-specific behavior with its current owner and symbols unexported. Use concrete types unless a present consumer must substitute implementations or direct coupling would violate dependency direction; then use the smallest interface in the consumer package and name its composition-root wiring. When a second present path would otherwise duplicate the same owned policy, consolidate that policy at the smallest shared owner. If required dependency direction or a generated/manual boundary forces two live representations, record the semantic owner, every copy, the allowed containment direction, and the lowest package permitted to import both; that package owns one shared-corpus parity test. A prose statement that the copies match does not close the decision. Add the smallest new surface or seam only when consolidation or another present responsibility cannot remain at its current owner without violating one of those boundaries. Prefer explicit control flow, the Go standard library, and established repository patterns. Expected future reuse, line count, test convenience, generic helper naming, and one-product factories do not meet that admission rule.

## Review

After both maps are fixed, run one required complementary panel over that exact
candidate. Dispatch the three fresh-context, read-only lanes concurrently when
capacity permits and sequentially otherwise:

1. **Responsibility and execution-path ownership** — an `architecture-agent`
   verifies that every selected system decision and normal, failure/recovery,
   operator, lifecycle, cleanup, and proof path contributes each necessary
   responsibility exactly once, with one semantic owner and source of truth. It
   accepts package and file placement as out of scope and returns those issues
   to the root.
2. **Package and dependency architecture** — a separate `architecture-agent`
   accepts the responsibility map as fixed and verifies exact directory/package
   placement, import direction and acyclicity, `internal` visibility,
   composition-root wiring, consumer-owned interfaces and exported surfaces,
   repository extension seams, the evidence-backed implementation-source
   disposition and exact dependency or generation pin, and generated/manual
   containment. It treats filenames only as locators and does not judge file
   cohesion or naming.
3. **File cohesion and naming** — a `quality-agent` accepts the fixed
   responsibility ownership and package-placement decisions and verifies each
   exact production and test file's
   one present reason to exist, filename, declaration grouping and visibility,
   split/co-location rationale, `doc.go` decision, fixture placement, and
   file-map implementation readiness. File length is only a signal to inspect
   these pressures. It does not re-inventory responsibilities or decide
   package/import architecture; contradictory semantic or package evidence
   returns to the root without a second lane verdict.

Each lane uses the shared [Lane Brief](../shared/subagents-and-handoff.md#lane-brief)
and [Finding Envelope](../shared/review-findings-and-convergence.md#finding-envelope)
and returns a lane-specific `PASS`, `CONCERNS`, or `FAIL` recommendation with
current evidence anchors. The root synthesizes cross-lane compatibility and
accepts the Go Ownership review only when all three lanes recommend `PASS` on
the same fixed candidate; it may not convert mixed verdicts into approval. A
finding or concern is repaired or reopened, then only each materially affected
lane receives the fixed candidate in a fresh focused review; unaffected `PASS`
receipts remain valid. If a required lane cannot run, review is incomplete.

This panel is the one phase-owned review branch for the Go Ownership boundary
and replaces a broad reviewer over the same questions. Apply independent
[Technical Design Review](technical-design-review.md) only when the shared
trigger still applies to a broader cross-domain technical-design boundary; that
review consumes the panel receipts instead of repeating their lenses.

During synthesis, reconcile both maps and reject an abstract layer or package name without an exact repository-relative path; a missing or duplicated responsibility; a file named without enough declaration evidence to justify its placement; a required edit leaving its touched file with independent reasons to change; a restructure extending beyond the responsibilities and companion surfaces it must move; an accepted policy change requiring multiple unconstrained semantic edits; a forced representation without parity proof; hidden state, resource, cancellation, or lifecycle ownership; test support outside its narrowest current owner; `doc.go` compensating for unclear names or placement; a new abstraction or exported symbol without a present consumer; or a stale second path. The design remains open when the concrete edit rehearsal still leaves Planning or Implementation to choose among plausible directories, packages, files, owners, visibility, or seams. Passing behavior proof does not dispose of these findings.

## Stop Rule

This phase completes Technical Design when the fixed System / Integration Design still satisfies its Stop Rule; accepted behavior is preserved; the responsibility and inverse file maps are complete, consistent, and supported by current evidence; every non-mechanical implementation-source choice has an evidence-backed disposition and exact authority or pin; no Review falsifier survives; and all three required panel lanes recommend `PASS` on the same fixed candidate with compatible current evidence. Every downstream mechanism, boundary, authority, failure policy, and Go owner is fixed. The resulting import graph is acyclic; Planning can preserve the exact repository-relative directory/package/file plan or its recorded deterministic implementation-local rule without choosing placement, dependency/composition, generated/manual authority, cleanup, proof ownership, lifecycle ownership, or exported surface; the plan names the implementation evidence that would invalidate it rather than using uncertainty to defer a current decision; Implementation can start from those fixed locations and is left only the actual code plus behaviorally equivalent local Go choices under repository [Engineering](../../../AGENTS.md#engineering); and any broader triggered technical-design review has returned `PASS` or dispositioned `CONCERNS`. Reopen system design only when placement cannot preserve the selected mechanism, runtime behavior, or source of truth; reopen specification only when placement cannot preserve scope or contract.
