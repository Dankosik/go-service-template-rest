# Component Map

| Surface | Change |
| --- | --- |
| `.codex/config.toml` | Retain the repository fan-out ceiling, retain depth, and add registry compatibility comment. |
| `.codex/agents/*.toml` | Add shared-contract reference and model/reasoning fields; update review routing for delivery/distributed/observability. |
| `.claude/agents/*.md` | Regenerate from Codex agent files using the new sync script. |
| `.agents/skills` | Add `go-devops-review`, `go-distributed-review`, and `go-observability-review`. |
| Runtime skill mirrors | Receive the three new skills through existing skill sync. |
| `scripts/dev/sync-agents.sh` | New mirror sync/check script for `.codex/agents` -> `.claude/agents`. |
| `Makefile` and `scripts/dev/docker-tooling.sh` | Add native and Docker agent mirror check targets. |
| `.github/workflows/ci.yml` | Run `make agents-check` in repo integrity. |
| `README.md` and docs | Document new agent hygiene commands, review-skill coverage, and brief template. |

No Go service packages are changed.
