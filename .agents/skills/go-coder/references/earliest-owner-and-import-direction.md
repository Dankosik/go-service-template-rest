# Earliest Owner And Import Direction

## Load When

Load this when the change adds a file, moves code between packages, introduces a
helper or interface, or when the obvious edit site would need a new import.

## Decide

Read the target's [Repository Architecture](../../../../docs/repo-architecture.md)
and dependency rules in `.golangci.yml` for the accepted import direction.
Resolve applicable selectors and exceptions for the changed packages, tests,
and examples. Missing lint enforcement does not remove an architectural
boundary; a portable reference does not define the target's package map.

- The earliest valid owner is the package that can already see what the change
  needs. When the symptom surfaces in a package that would need a denied import
  to fix it, the defect is upstream: its caller is passing too little, and the
  change belongs where the value is already known. Adding the import is not the
  smaller diff — it is a larger change to the architecture wearing a one-line
  disguise.
- Inspect the current caller and composition root for an existing seam before
  adding one. Extend it when the accepted responsibility belongs there.
- Extract a helper when repeated behavior is stable policy that one package
  owns; keep one-use logic inline where the state change stays visible.
- Define an interface at the consumer that needs substitution. Check the target's
  enabled interface linters and exceptions without treating them as a substitute
  for that ownership decision.

## Prove

- Select scoped import and interface checks through [Validation
  Routing](../../../../docs/validation-routing.md).
- Check the imports of the package you moved into: a correct boundary adds no
  cycle and no unrelated dependency.
- Run the tests of the old call sites and of the package that now owns the code.
