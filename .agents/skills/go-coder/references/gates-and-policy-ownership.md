# Gates And Policy Ownership

## Load When

Load this when adjacent code looks modernizable, when a local helper looks
replaceable by `slices`, `maps`, or `cmp`, or when deciding how much of a
mechanical concern still needs proof in this change.

## Decide

Resolve available idioms through [Go Modern Version](../../go-modern-version/SKILL.md)
for the module containing the changed file. Read the target repository's
`.golangci.yml` for enabled checks and accepted exceptions; a portable reference
cannot establish that local policy. Apply mechanical suggestions within the
accepted change, without modernizing unrelated files.

- A gate suggestion is a default, not a decision. Accept a swap only when the
  replaced code carried no policy the stdlib call drops: `cmp.Or(input.Limit,
  defaultLimit)` is wrong wherever `0` means "no limit" rather than "unset", and
  `maps.Clone` and `slices.Clone` are shallow, so nested slices, maps, and
  pointers still alias every caller after the swap.
- Cleanup is scoped to what this change replaced: the superseded function, its
  wiring, its tests, its config keys and env defaults, its generated output, and
  the docs and comments that named it. Tidying anything else is a separate change.

## Prove

Select scoped lint and any modernization check through [Validation
Routing](../../../../docs/validation-routing.md). When a swap replaces a
policy-carrying helper, require proof of nil versus empty, zero-value fallback,
order, or shallow versus deep copy as applicable. Reuse or add that proof through
the [Evidence Contract](../../../../docs/spec-first-workflow/shared/evidence-contract.md).
