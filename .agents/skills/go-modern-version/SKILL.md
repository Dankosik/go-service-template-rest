---
name: go-modern-version
description: "Go version idioms. Use when the module's Go version can change a language or standard-library choice in the planned diff, or modernization is requested."
metadata:
  invocation: model
  kind: method
---

# Go Modern Version

Modern Go is a **version surface**: the module's Go version determines which
language and standard-library forms are available.

Read the affected module's `go.mod`. When version availability affects the
planned choice or modernization is requested, query the pinned JetBrains
inventory from the repository root against that module's Go file or `go.mod`:

```bash
go tool -modfile=tools/go.mod go-modern-guidelines list --file-path <path>
```

Reuse the inventory for the same resolved version. Read and explain only
guidelines that may affect the diff. Text-only edits or edits with no
version-dependent choice need no inventory. For a modernization spanning
multiple files, record applicable guideline IDs and a disposition per affected
file; a single local choice needs only its rationale and focused proof.

The wrong default is copying nearby legacy syntax after the target version
supports a clearer form. Apply the modern form unless it changes behavior,
error identity, nil or empty meaning, aliasing, ordering, performance contracts,
or source ownership; those semantic owners take precedence.

Complete when the chosen forms are available at the module's version and the
affected behavior is preserved with focused proof. A modernization sweep also
accounts for each affected file as applied, not applicable, or blocked with an
exact reason. Use the active workflow's validation plan.
