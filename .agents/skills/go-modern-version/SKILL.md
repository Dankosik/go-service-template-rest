---
name: go-modern-version
description: "Go version idioms. Use before creating or editing handwritten Go when the target module version determines which current language and standard-library forms are available."
metadata:
  invocation: model
  kind: method
---

# Go Modern Version

Modern Go is a **version surface**: the module's Go version determines which
language and standard-library forms are available.

Before the first handwritten Go edit in each affected module, run the pinned
JetBrains inventory from the repository root against an existing Go file or its
`go.mod`:

```bash
go tool -modfile=tools/go.mod go-modern-guidelines list --file-path <path>
```

Read the complete output once per resolved version without filtering or
truncation. Build one
`Modernization{file, go_version, applicable_ids, disposition}` record for every
changed handwritten Go file. Call `explain <id...>` only for returned guidelines
that may affect the planned diff.

The wrong default is copying nearby legacy syntax after the target version
supports a clearer form. Apply the modern form unless it changes behavior,
error identity, nil or empty meaning, aliasing, ordering, performance contracts,
or source ownership; those semantic owners take precedence.

Complete when every record is applied, not applicable, or blocked with an exact
reason, and the changed package's focused lint accepts the result.
