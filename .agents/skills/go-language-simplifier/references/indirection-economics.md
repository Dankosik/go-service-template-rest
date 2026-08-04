# Indirection Economics

Behavior Change Thesis: this file makes the model price a helper by what its
interface costs against what it hides, so it stops recommending the inline of a
single-use helper that is the only place a non-obvious constraint is written
down — the mistake `SKILL.md`'s own "deletion beats abstraction" invites and no
configured linter catches.

## When To Load
Load this when a diff extracts, inlines, generalizes, or flag-parameterizes a
helper, wrapper, interface, callback, or option shape.

## Decision Rubric
- Price the interface, not the line count. A helper earns its place when what a
  caller must learn to use it — name, parameters, results, and the contract it
  implies — is smaller than what it removes from the call site. Shallow
  indirection is the reverse: an interface nearly as large as its body.
- A single-use helper still earns its place when its name or doc comment is the
  only place a constraint, safety argument, or ownership rule is recorded.
  Inlining deletes the reasoning while leaving the behavior, and nothing in the
  build reports it. Ask what a reader loses, not how many callers it has.
- Prefer the parameter shape that makes an invalid call unrepresentable. Boolean
  and same-typed positional arguments push the decision to the call site and are
  not linted here: `unparam` does not report a flag that is constant across
  every caller.
- Repeated stable policy wants one named owner in the package that owns the
  error identities and the contract, not a new shared bucket. Placement of a new
  owner is `go-implementation-ownership`; duplication judged across the whole
  diff is `go-structural-quality`.

Spend no finding where a gate already fails the build: `iface` reports unused,
identical, and single-implementation interfaces; `gocritic` reports if-else
chains and single-case switches; `nestif` reports nesting at complexity 5;
`goconst` reports a literal repeated 4 times; `nakedret` reports naked returns
past 30 lines; `revive` reports unused parameters.

## Reject
`func write(w, v, cached bool, private bool)` — the call site no longer says
which behavior was selected and invalid combinations stay callable. Split on the
stable outcomes instead, or keep the branches local.

## Validation Shape
Call-site clarity needs a targeted build and reviewer inspection of the corrected
callers. When the change moves branch selection into or out of a helper, the
proof is one case per behavior class, not the happy path.
