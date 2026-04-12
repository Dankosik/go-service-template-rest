I might be naming this wrong: maybe `internal/db/ping_repo.go`? I don't think that file is real.

The actual thing is ping history persistence. Duplicate writes started showing up after a retry path, and I want the agent to inspect the repository layer and sqlc query shape before changing anything.
Signals: `ping_history`, `sqlc`, migrations if needed, postgres adapter, maybe repository test.
Do not hand-edit generated sqlc output as the only fix.
If the guessed path is bogus, say so instead of pretending it exists.
