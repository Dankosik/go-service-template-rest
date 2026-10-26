# Ownership Map V1

Return reconciled Responsibilities and Files sections, plus Reused
Capabilities and Consumer Changes when they have rows.

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

## Reused Capabilities

One row per capability that a new package, executable, composition root, or
test fixture needs and an existing owner already provides, such as
configuration loading, logging, telemetry, lifecycle, or fixtures:

```text
capability | new consumer | existing owner | reuse or shared owner | dependency rule forbidding direct reuse | extraction dependency
```

Reuse the existing owner. When a dependency rule forbids direct reuse, name
the shared owner both consumers import; its extraction is a Planning
dependency. A copy is not a placement.

## Consumer Changes

One row per existing shared package whose importing packages the change
reduces:

```text
package | importing packages before -> after | keep with current constraint, or collapse into remaining owner
```

One remaining production owner makes the package a collapse candidate.

Every responsibility appears exactly once and every file maps back to a present
responsibility. Use an evidence-bounded deterministic placement rule only when
implementation-local facts prevent an exact path now.
