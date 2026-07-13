---
name: workflow-status
description: "Read-only status and next-action summary for one identified task path."
---

# Workflow Status

Use for one explicit task path or the current directory when the active task is unambiguous. Do not guess from recency across several task bundles.

Read in this order: current `tasks.md` for implementation/validation; otherwise `workflow-plan.md`; then only the spec/design/test/research/rollout artifacts named by the next action.

Before reporting `ready`, `done`, or `Implementation may start: yes`, inspect the current workspace and relevant Git diff/status for drift from the owning artifact. Treat unreviewed implementation changes, stale generated or mirrored outputs, and missing fresh proof as evidence gaps rather than inheriting an artifact's older status.

Return:

```text
Status: draft | ready | blocked | done
Current outcome/phase:
Owning artifact:
Evidence or blocker:
Next action:
Implementation may start: yes | no, with reason
```

Do not edit, approve, or repair state. If no task can be identified, ask for one path.

If the next action is an internal review, repair, fresh re-review, validation, or closeout checkpoint, label it as same-request work and do not propose a next-session prompt. A handoff is ready only at an intentional next macro phase or an honest blocker the current root cannot resolve.
