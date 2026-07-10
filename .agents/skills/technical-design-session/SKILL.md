---
name: technical-design-session
description: "Create or repair only the system and Go-ownership design needed to make implementation executable, then review it proportionately."
---

# Technical Design Session

Use when a ready spec still leaves runtime mechanism or Go ownership decisions open.

Read [system/integration design](../../../docs/spec-first-workflow/phases/system-integration-design.md), then [Go ownership design](../../../docs/spec-first-workflow/phases/go-code-ownership-design.md). Keep both in one compact `design/overview.md` unless a focused split materially improves ownership or review.

Use specialist lanes only for independent live forks that can change the design. Invoke [technical design review](../../../docs/spec-first-workflow/phases/technical-design-review.md) when independent review is user-required or justified by impact, reversibility, ambiguity, or cross-owner risk.

Treat review as an internal checkpoint: the owning root repairs findings, obtains any needed fresh re-review, and continues in the same authorized request. Stop after findings only when the user explicitly requested a standalone read-only design review.

Success means planning can name behavior, source of truth, files/packages, cleanup, tests, and proof without choosing design. Reopen the narrowest upstream owner when it cannot.
