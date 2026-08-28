# Template tooling parity

status: ready

- [ ] T001: Make the source template's portable tooling one committed,
  profile-independent surface propagated by the existing sync engine.
  - Accept when TP-01 through TP-06 and TP-08 pass on one fixed template
    candidate and independent review finds no surviving ownership loss.
- [ ] T002: Migrate the 29 discovered repository identities without overwriting
  local tooling or ambiguous profile state.
  - Depends on: accepted and committed T001.
  - Accept when every canonical repository is either zero-drift with focused
    verification or has one explicit blocked receipt; duplicate local checkouts
    consume the canonical Git result rather than an independent mutation.

No push, pull request, deployment, required-check mutation, or credential action
is authorized by this ledger.
