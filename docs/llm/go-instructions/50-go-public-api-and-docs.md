# Go public API and documentation instructions for LLMs

## Load policy
- Load: Optional
- Use when:
  - Designing reusable libraries or exported packages
  - Writing exported types, functions, methods, constants, or variables
  - Writing doc comments, package documentation, or examples
  - Making compatibility decisions for public APIs
- Do not load when: The code is entirely internal and no public-facing API or documentation is involved

## Public API principles

- Treat every exported identifier as part of a contract.
- Keep public APIs small, coherent, and hard to misuse.
- Prefer stable, simple surfaces over exposing internal flexibility.
- Return concrete types when that keeps the API simpler.
- Define interfaces at the point of use, not preemptively in the producer package.

## Naming rules for public APIs

- Package names should be short, lowercase, and descriptive.
- Avoid stutter in client code.
- Use consistent initialisms such as `ID`, `URL`, `HTTP`, and `JSON`.
- Choose names that read naturally at the call site.
- Do not export names just because a type or helper exists.

## Documentation rules

- Every exported package and identifier should have a doc comment.
- Start the comment with the identifier name.
- Write complete sentences.
- Make the first sentence useful as summary documentation.
- Keep comments factual and usage-oriented rather than repeating type information mechanically.
- Use package docs to explain the package purpose, major concepts, and usage constraints.

## Examples

- Provide example functions for public APIs that are easy to misuse or non-obvious.
- Keep examples short, realistic, and executable.
- Prefer examples that teach the intended happy path first.
- Make sure examples reflect idiomatic imports and naming.

## Compatibility rules

- Avoid unnecessary breaking changes to exported names, signatures, error semantics, or behavior.
- If an error category is public, keep its detection stable through `errors.Is` or typed errors.
- Avoid exposing implementation details that would block refactoring later.
- Use `internal/` and unexported identifiers to keep freedom to change internals.

## API design guidance

- Keep zero values useful when practical.
- Make the common path easy and the unusual path possible.
- Keep constructors and option patterns simple. Do not introduce configuration complexity without need.
- Use `context.Context` consistently in public operations that may block or be canceled.
- Document concurrency guarantees, ownership expectations, and nil/empty semantics when they matter to callers.
- If the API returns wrapped or typed errors, document the stable parts the caller may depend on.

## Generics guidance

- Use generics only when they clearly reduce duplication and improve clarity for a family of types or algorithms.
- Do not add type parameters where an interface or a concrete type is simpler.
- Avoid turning an API generic just to appear modern.
- Keep generic APIs readable at the call site.

## Common anti-patterns to avoid

- Large exported surfaces with many incidental helpers
- Comments that do not start with the identifier name
- Public interfaces defined only for mocking
- Exporting types or fields that should remain private
- Forcing callers to understand internal package structure
- Generic APIs that add complexity without removing real duplication

## What good output looks like

- The API feels small, consistent, and idiomatic.
- Call sites read naturally.
- Documentation is useful both in source and in generated docs.
- Public contracts are intentional and stable.
- Internal implementation choices remain free to evolve.

## Checklist

Before finalizing, verify that:
- Every exported identifier has a proper doc comment.
- Public names read well at the call site.
- The exported surface is minimal.
- Stable error behavior, nil semantics, and concurrency expectations are documented where necessary.
- Examples exist for the most important or non-obvious entry points.
- Generics are used only where they materially improve the API.
