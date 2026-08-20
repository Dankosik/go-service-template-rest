# Artifact Lifecycle V1

Use one status when durable state changes an action:

| Status | Meaning |
| --- | --- |
| `draft` | The owning decision or execution record is still being authored or repaired. |
| `ready` | The artifact has closed its owned decisions and its next consumer can act without semantic invention. |
| `blocked` | A required missing decision or input and its reopen owner prevent useful progress. |
| `done` | Execution and global completion are proved; this status is not evidence by itself. |

A material invalidation moves `ready` back to `draft` or `blocked`. Only the
artifact owner changes status. Triggered review follows the shared [Review](../shared/review.md)
lifecycle. Task receipts and executable-ledger transitions use [Task Ledger
V1](task-ledger-v1.md).
