# Go project layout and module instructions for LLMs

## Load policy
- Load: Optional
- Use when:
  - Creating a new repository or service
  - Restructuring packages
  - Designing module boundaries
  - Working with `go.mod`, `go.sum`, toolchain configuration, internal packages, commands, vendoring, or dev tools
- Do not load when: The task is only about a small local code change inside an already-settled project structure

## Layout principles

- Start with the simplest layout that matches the current project.
- Let package boundaries follow responsibilities, not arbitrary folders.
- Keep packages small, focused, and easy to import.
- Public API should be intentional. Implementation details should stay private.
- Avoid speculative structure for problems the project does not yet have.

## Module and import rules

- Remember that package import paths are the module path plus the package directory.
- Keep import paths stable and readable.
- Minimize the number of modules unless there is a real release or dependency reason to split them.
- Avoid circular dependencies by designing package responsibilities clearly.

## Recommended directory patterns

- Use `cmd/<program>/main.go` for executable entry points.
- Use `internal/` for packages that should not be imported outside the parent tree.
- Place public reusable packages in normal package directories. A `pkg/` directory is optional, not required.
- Do not create `pkg/` or other top-level directories just because a template used them.
- Keep generated code separate and clearly labeled.

## Package design rules

- Do not create generic junk drawers such as `util`, `utils`, `common`, `helpers`, or `misc`.
- Prefer domain-specific package names that tell the reader what lives there.
- Keep package names lowercase and concise.
- Do not force unrelated responsibilities into one package just to reduce file count.
- Avoid over-layering. Small Go programs rarely need Java-style architecture directories.
- Do not let one file become a mixed-responsibility "god file". When a file accumulates distinct concerns, split by responsibility into multiple files within the same package first.

## Public versus private code

- Export only what callers truly need.
- Put volatile implementation details behind unexported identifiers or `internal/` packages.
- Treat anything exported as a contract you may need to support long term.
- If you are not ready to support it publicly, do not export it.

## Dependency rules

- Prefer fewer dependencies and smaller dependency surfaces.
- Use the standard library when it is sufficient.
- Add third-party dependencies only when they clearly improve correctness, maintainability, or capability.
- Keep `go.mod` and `go.sum` checked in.
- Treat `go.sum` as part of build integrity, not as disposable noise.

## Module maintenance

- Use `go mod tidy` to keep module metadata clean.
- Use `go mod verify` when integrity checks are important.
- Understand that version selection is module-driven; avoid manually pinning transitive details without a reason.
- Vendor dependencies only when policy, offline builds, or supply-chain controls justify it.

## Tooling as dependencies

- If the project targets Go 1.24 or newer, prefer tool directives in `go.mod` for developer tools.
- If an older toolchain is required, use the team's established fallback approach consistently.
- Pin tool versions when CI reproducibility matters.
- Keep tool choices intentional. Do not add linters or generators without a clear value.

## Toolchain guidance

- Set the `go` version in `go.mod` intentionally.
- Use the `toolchain` directive when reproducibility across developers or CI matters.
- Keep local development and CI on compatible toolchain versions.

## Common anti-patterns to avoid

- Copying a large repository template without need
- Creating `util` or `common` packages as storage for unrelated code
- Exporting internal implementation details too early
- Splitting into multiple modules prematurely
- Treating `go.sum` as a file to delete casually
- Depending on vendoring by default without a supply-chain or build reason
- Hiding poor package design behind long import paths

## What good output looks like

- The directory structure matches the actual shape of the project.
- Package boundaries are easy to explain.
- Import paths stay short and readable.
- Public APIs are intentional and minimal.
- The project works naturally with the Go module system and tooling.

## Checklist

Before finalizing, verify that:
- `cmd/` is used only for binaries.
- `internal/` protects non-public implementation where appropriate.
- Package names are specific and non-stuttering.
- No junk-drawer packages were introduced.
- `go.mod` and `go.sum` are treated as first-class project files.
- Tooling and dependency decisions are justified, not cargo-culted.
