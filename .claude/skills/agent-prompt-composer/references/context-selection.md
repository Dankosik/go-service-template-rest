# Context Selection Map

Rule: load the smallest useful set of repo context.
Do not bulk-read directories by default.

## Step 1: Classify The Task Mode

Common signals in rough user input:
- feature/change
  - `add`, `build`, `implement`, `change`, `new`, `добавить`, `сделать`, `hacer`, `ajouter`
- bugfix
  - `breaks`, `bug`, `not working`, `fails`, `ошибка`, `падает`, `no funciona`, `cassé`
- investigation
  - `investigate`, `why`, `where does it break`, `разобраться`, `понять`, `por que`, `pourquoi`
- refactor/simplify
  - `cleanup`, `refactor`, `simplify`, `tidy`, `упростить`, `почистить`, `refactorizar`
- plan/spec/design
  - `spec`, `plan`, `design`, `architecture`, `спека`, `план`, `diseño`
- prompt/tooling/skills
  - `skill`, `prompt`, `agent`, `workflow`, `skills-sync`, `codex`, `claude`

If the input mixes several modes, choose the one that best matches the requested outcome.
If the same mode appears several times through repeated wording, treat that as emphasis rather than separate tasks.

## Step 2: Map The Smallest Relevant Repo Surface

### HTTP / API / Router / OpenAPI
Signals:
- `handler`, `хендлер`, `route`, `роут`, `endpoint`, `chi`, `openapi`, `swagger`, `cors`, `405`, `404`, `problem json`

Inspect first:
- `internal/infra/http/`
- `api/openapi/service.yaml`
- `internal/api/README.md`

Validation to mention:
- focused `go test` in `./internal/infra/http`
- `make openapi-check` only if contract or generated API code changes

### Postgres / SQL / sqlc / Migrations / Cache
Signals:
- `postgres`, `pgx`, `sql`, `sqlc`, `query`, `transaction`, `migration`, `cache`, `кэш`

Inspect first:
- `internal/infra/postgres/`
- `env/migrations/`

Validation to mention:
- focused package tests
- `make sqlc-check`
- `make migration-validate` when migrations are involved

### Bootstrap / Startup / Shutdown / Config / Readiness
Signals:
- `bootstrap`, `startup`, `shutdown`, `drain`, `readiness`, `health`, `probe`, `config`, `конфиг`, `env`

Inspect first:
- `cmd/service/internal/bootstrap/`
- `internal/config/`
- `internal/app/health/`

Validation to mention:
- focused package tests
- `make test-race` when shutdown or concurrency is implicated
- `make test-integration` when lifecycle behavior crosses integration boundaries

### Telemetry / Metrics / Tracing
Signals:
- `metric`, `metrics`, `trace`, `tracing`, `otel`, `prometheus`, `/metrics`, `span`

Inspect first:
- `internal/infra/telemetry/`
- `internal/infra/http/` when `/metrics` or request instrumentation is involved

Validation to mention:
- focused package tests
- `make test`

### Skills / Prompts / Agents / Workflow Tooling
Signals:
- `skill`, `prompt`, `agent`, `workflow`, `skills-sync`, `agents-sync`, `mirror`, `codex`, `claude`, `cursor`

Inspect first:
- `.agents/skills/`
- `.codex/agents/`
- `.claude/agents/`
- `scripts/dev/sync-skills.sh`
- `scripts/dev/sync-agents.sh`
- `AGENTS.md`
- `docs/spec-first-workflow.md`
- `docs/subagent-contract.md`
- `docs/subagent-brief-template.md`

Validation to mention:
- `make agents-sync`
- `make agents-check`
- `make skills-sync`
- `make skills-check`
- `git diff --check` when files are created or edited

### Tests / Flakes / Race / CI
Signals:
- `test`, `flake`, `flaky`, `race`, `hang`, `timeout`, `CI`, `integration`

Inspect first:
- nearby package tests
- `test/`
- `.github/workflows/ci.yml`

Validation to mention:
- focused `go test`
- `make test`
- `make test-race`
- `make test-integration` when integration behavior matters

### Planning / Architecture / Specs
Signals:
- `spec`, `plan`, `architecture`, `boundary`, `ownership`, `workflow`

Inspect first:
- `AGENTS.md`
- `docs/spec-first-workflow.md`
- `specs/`
- relevant `.agents/skills/*-spec/`

Validation to mention:
- artifact consistency
- the smallest repo checks that match the touched surface

## Step 3: Safe Live Lookup Rules
- If the user names an exact file or directory, inspect that exact surface first.
- If the user names only a concept, inspect at most one mapped directory and one nearby test or source-of-truth file.
- If the mapping still leaves multiple plausible surfaces, stop and record an assumption instead of widening the search.

## Step 4: Source-Of-Truth Reminders
- OpenAPI changes belong in `api/openapi/service.yaml`, not hand-edited generated code.
- sqlc changes belong in SQL/query or config sources, not generated files alone.
- Mock or enum generation changes should mention the owning source and drift-check path.
