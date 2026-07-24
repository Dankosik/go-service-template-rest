# Adoption v1 closeout test plan

TD-1 | workflow preservation | complete and minimal derived checkouts are initialized with no workflow option | current test deletes workflow by default | repository contract | snapshot workflow paths and SHA-256 content before initialization | identical path/content snapshot after initialization; database/outbound pruning still occurs | fails current default | `make template-init-check` | reopen initialization owner on any workflow mutation

TD-2 | optional reference example | delete `examples/reference-service` in a temporary initialized checkout | current Make/OpenAPI drift commands use unconditional paths | generated-contract gate | delete example, generate and run service OpenAPI gate | service generation, drift, generated package test, lint, validation, and runtime contract all still execute | fails current Makefile and drift script | `make template-init-check` | example-specific quality is proved only while example exists

TD-3 | migration budget input | zero, negative, or inconsistent values | no statement budget is configured today | focused unit/config tests | load invalid typed config and call migration options validation | fail before database access with named key/option | current code accepts zero driver timeout | `go test ./internal/config ./internal/infra/postgresmigrate ./cmd/migrate` | none

TD-4 | cancellation result and cleanup | context expires while a stub migration returns nil | graceful stop can currently look successful | focused unit test | cancel during `Up`, then allow it to return | context error returned, changed false, runner closed once | fails current execute signature | `go test ./internal/infra/postgresmigrate` | driver interruption is covered by TD-5

TD-5 | in-flight SQL bound | migration executes `pg_sleep` beyond statement budget | no database statement timeout | PostgreSQL integration | 100ms statement budget against long SQL | returns well before SQL duration and schema is dirty | fails current driver config | `REQUIRE_DOCKER=1 go test -tags=integration ./test -run 'TestPostgresMigrate'` | timing ceiling is generous for CI scheduling

TD-6 | dirty recovery diagnostic | rerun after a deliberately failed migration | upstream error is not locally actionable | PostgreSQL integration | fail version 1, invoke migrator again | exact dirty version, no automatic force claim, runbook pointer | current error lacks local recovery policy | same as TD-5 | actual repair remains an operator decision

TD-7 | generated POST happy/negative paths | temporary PostgreSQL-derived service adds fixture contract and feature | no executable DB-backed endpoint workflow | generated-service PostgreSQL integration | generate OpenAPI/sqlc, migrate, POST valid/malformed/invalid/unknown fields | 201 plus durable row only for valid input; all rejected inputs leave count unchanged | fixture absent today | `REQUIRE_DOCKER=1 TEMPLATE_POSTGRES_PROOF=1 make template-init-check` | fixture is intentionally not base runtime code

TD-8 | transaction finality | temporary fixture repository writes under real pgx transactions | documentation only | PostgreSQL integration | write then rollback; write then commit | rollback row absent, committed row present | fixture absent today | same as TD-7 | no fake public transaction requirement
