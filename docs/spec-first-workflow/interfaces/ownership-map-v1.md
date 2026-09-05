# Ownership Map V1

Return two reconciled sections.

Describe each responsibility, its constraints, and evidence once in
Responsibilities. Files references those entries by unambiguous name or anchor
and supplies the file-specific facts. A reference may fill a repeated field only
when it resolves that field completely; retain every required field and check
both responsibility-to-file and file-to-responsibility coverage.

## Responsibilities

One row per changed responsibility:

```text
responsibility | affected path | current evidence | semantic owner | exact package/file action | dependency/composition/generated boundary | cleanup | proof owner | reopen condition
```

For a non-mechanical implementation source, also record its selected reuse rung,
authority or version, strongest viable rejected source, parity proof, and
upgrade condition.

## Files

One row per added or materially changed Go file:

```text
path | responsibilities | one present reason to exist | declarations/visibility | call-path role | lifecycle/error ownership | allowed dependencies | forbidden responsibilities
```

Every responsibility appears exactly once and every file maps back to a present
responsibility. Use an evidence-bounded deterministic placement rule only when
implementation-local facts prevent an exact path now.
