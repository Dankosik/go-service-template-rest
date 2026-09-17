# Isolated And Remote Execution Inputs

Load before executing a check in an isolated or remote environment.
[Implementation](../implementation.md) owns timing and acceptance.

For isolated or remote execution, use the existing runner and one current
candidate record in task state or its manifest. Resolve source/patch or image
identities, stages, checksums, and host-specific paths there. Verify the actual
execution inputs before consuming a result; do not reconstruct ad hoc transfer
commands or treat a workstation path as a remote path. After repair, update
the replacement record and invalidate only affected evidence. This bookkeeping
needs no new scheduler or registry.
