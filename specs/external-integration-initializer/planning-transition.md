# Planning transition

status: superseded
superseded_by: 2026-08-23 named-only OAuth maintainability repair
owner: Planning
result: `specs/external-integration-initializer/tasks.md` SHA-256 `70090d5907c3260c940e2420c2a627e331edc7c04dd7a5eda1762fb3a0d09824`; packets `T001` SHA-256 `f84d97fc92aab9de6cffad5053f28256f89037d7cb947b094f43a49479fd701c` and `T002` SHA-256 `c4bf2b1b05d7a8f1e9fddc6b8d89923b296cbba9245ecbb08c8507f5fbc46b91`
review: `specs/external-integration-initializer/planning-review.md` SHA-256 `168cc4c7013a4fee78f3cf39fcf67f99a8f0612b33a5727263452021bfe23236` — fresh independent Task Review / Readiness `PASS`
movement_evidence: Every accepted implementation-changing obligation has one disposition in the two-unit dependency ledger. T001 is the only ready unit and can reach its `make template-init-check` acceptance from the fixed inputs. T002 consumes accepted T001 and keeps all 25 Test Design rows, the nine-command ladder, the actual-`.env` exclusion, and mandatory pinned-tool, readable-`origin/main`, Linux-`strace`, and Docker gates in one independently acceptable final outcome. Mutable owners and exclusive locks make the frontier deterministic, and the fresh reviewer found no hidden decision, invalid split, missing next-unit input, or non-dispositive check.
reopen_owner: Planning
next_owner: Implementation

Implementation may open only T001 first. T002 remains unopened until T001 is
accepted and every packet gate required at dispatch is available. Pre-existing
dirty checkout bytes remain evidence only and outside every candidate; actual
`.env` contents and every provider, network, credential, deployment, or other
external effect remain unauthorized.

Reopen Planning only if a unit is not independently acceptable, a dependency
or lock is incomplete, or a named check cannot falsify its postcondition.
Reopen the smallest row-specific Test Design or Technical Design owner when its
fixed proof or mechanism is invalidated. Reopen Specification only if the fixed
fail-closed behavior, developer/operator custody boundary, byte-restoration
rules, or initial/refresh consequences cannot be preserved without changing
accepted behavior or authority.

This receipt names Implementation only as the next owner. It does not enter
Implementation or authorize any implementation or external action.
