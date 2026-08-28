# Template tooling parity test plan

status: ready

| ID | Wrong behavior | Deterministic falsifier and oracle | Gate |
| --- | --- | --- | --- |
| TP-01 | A portable file changes upstream but remains stale downstream. | Commit fixture template v2, apply from that exact Git snapshot, compare every applicable manifest path byte-for-byte. | focused template-sync behavior check |
| TP-02 | Adoption deletes a service Make target or sibling script. | Add fixed local owners before apply; execute/list them and compare their hashes after apply. | focused template-sync behavior check |
| TP-03 | Standard command availability depends on initialized profile. | Initialize minimal and full fixtures; compare the standard target inventory from `make/template.mk`; invoke one absent-capability target and require `not applicable`. | initializer/tooling fixture |
| TP-04 | A dirty owned file is overwritten. | Modify one portable target path, run check and apply, require refusal and an unchanged whole-target hash. | focused template-sync behavior check |
| TP-05 | Pinned tooling differs or cannot resolve downstream. | Compare `tools/go.mod`/`go.sum`, then run the existing tools-resolution owner in the initialized fixture. | tools resolution check |
| TP-06 | Module-specific lint bytes reappear. | Initialize two module paths, require identical portable config bytes and successful rendered-config validation for each module. | lint config fixture |
| TP-07 | Legacy migration guesses profile state. | Remove or contradict one profile signal and require pre-write refusal naming the unresolved field. | migration fixture |
| TP-08 | CI reports green without exercising applicable tooling. | Change a portable script/config in the fixture and require the classifier and aggregate required gate to select/fail the matching leaf. | changed-surfaces self-test and CI review |

Reopen Test Design if the portable set gains a network/control-plane dependency
or if a target-local merge replaces byte identity as the oracle.
