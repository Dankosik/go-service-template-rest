---
name: auth-access-control
description: "Principal, credential, permission, tenant-isolation, and revocation depth reference reached from go-security."
---

# Auth And Access Control

Use only after the active phase identifies an auth-specific decision. Inherit
its authority, artifact, review, proof, output, and completion contract; do not
select a work mode or production action here.

Core invariant: every protected effect is bound to a verified principal and an
explicit permission decision; missing or ambiguous identity, tenant, scope, or
context denies access.

Load one branch:

- principal types, entry points, delegation, or cross-service identity ->
  [principal-boundaries.md](references/principal-boundaries.md);
- credential, session/token, rotation, recovery, or revocation window ->
  [credential-lifecycle.md](references/credential-lifecycle.md);
- permission model, enforcement, object access, or tenant isolation ->
  [authorization-enforcement.md](references/authorization-enforcement.md).
