SERVICE_NAME := service
SERVICE_CMD := ./cmd/service
BINARY := bin/$(SERVICE_NAME)
# profile:messaging-nats-jetstream:start
WORKER_CMD := ./cmd/worker
WORKER_BINARY := bin/$(SERVICE_NAME)-worker
# profile:messaging-nats-jetstream:end
# profile:outbox-postgres:start
OUTBOX_RELAY_CMD := ./cmd/outbox-relay
OUTBOX_RELAY_BINARY := bin/$(SERVICE_NAME)-outbox-relay
# profile:outbox-postgres:end
# profile:jobs-postgres:start
JOBS_WORKER_CMD := ./cmd/jobs-worker
JOBS_WORKER_BINARY := bin/$(SERVICE_NAME)-jobs-worker
# profile:jobs-postgres:end
GO ?= go
PGO_PROFILE ?= off
OPENAPI_FILE := api/openapi/service.yaml
REFERENCE_OPENAPI_FILE := $(wildcard examples/reference-service/api/openapi.yaml)
REFERENCE_OPENAPI_PACKAGE := $(if $(REFERENCE_OPENAPI_FILE),./examples/reference-service/internal/openapi)
OPENAPI_FILES := $(OPENAPI_FILE) $(REFERENCE_OPENAPI_FILE)
OPENAPI_PACKAGES := ./internal/openapi $(REFERENCE_OPENAPI_PACKAGE)
OPENAPI_GENERATED_FILES := internal/openapi/openapi.gen.go $(if $(REFERENCE_OPENAPI_FILE),examples/reference-service/internal/openapi/openapi.gen.go)
GO_FILES := $(shell git ls-files --cached --others --exclude-standard -- '*.go' 2>/dev/null | awk '!/^(\.agents|\.cache|vendor)\//' | while IFS= read -r file; do [ -f "$$file" ] && printf '%s\n' "$$file"; done)
PROTO_GENERATED_GO_FILES := internal/gen/proto/% examples/grpc-reference-service/internal/gen/proto/%
GOIMPORTS_FILES := $(filter-out $(PROTO_GENERATED_GO_FILES),$(GO_FILES))
GOFUMPT_FILES := $(filter-out internal/openapi/openapi.gen.go internal/infra/postgres/sqlcgen/% $(PROTO_GENERATED_GO_FILES),$(GO_FILES))
SHELL_FILES := $(shell git ls-files --cached --others --exclude-standard -- '*.sh' 2>/dev/null | awk '!/^(\.agents|\.cache|vendor)\//' | while IFS= read -r file; do [ -f "$$file" ] && printf '%s\n' "$$file"; done)
REDOCLY_CLI_VERSION := 2.40.0
REDOCLY_CLI ?= npx --yes @redocly/cli@$(REDOCLY_CLI_VERSION)
GO_TOOL := go tool -modfile=tools/go.mod
GOLANGCI_LINT ?= $(GO_TOOL) golangci-lint
GO_REQUIRED_VERSION := $(shell awk '/^go / {print $$2; exit}' go.mod)
INTEGRATION_PACKAGES := ./test/...
# profile:http-idempotency-postgres:start
INTEGRATION_PACKAGES += ./internal/infra/postgresidempotency
# profile:http-idempotency-postgres:end
# profile:messaging-nats-jetstream:start
INTEGRATION_PACKAGES += ./internal/infra/natsjs ./cmd/worker/internal/bootstrap
MESSAGING_RACE_PACKAGES := ./internal/infra/natsjs ./cmd/worker/internal/bootstrap ./test/...
# profile:outbox-postgres:start
MESSAGING_RACE_PACKAGES += ./cmd/outbox-relay/internal/bootstrap
# profile:outbox-postgres:end
# profile:messaging-nats-jetstream:end
# profile:outbox-postgres:start
OUTBOX_RACE_PACKAGES := ./internal/domainevent ./internal/infra/postgresoutbox ./internal/infra/natsjs ./cmd/outbox-relay/internal/bootstrap ./test/...
# profile:outbox-postgres:end
# profile:webhooks-durable:start
WEBHOOK_RACE_PACKAGES := ./internal/infra/postgreswebhook ./test/...
# profile:webhooks-durable:end
LINT_BASE_REF ?= origin/main
LINT_CONCURRENCY ?= 4
SECRET_SCAN_BASE_REF ?= $(if $(strip $(BASE_REF)),$(BASE_REF),origin/main)

TRIVY_IMAGE ?= aquasec/trivy:0.72.0@sha256:cffe3f5161a47a6823fbd23d985795b3ed72a4c806da4c4df16266c02accdd6f
TRIVY_CACHE_VOLUME ?= trivy-cache
ACTIONLINT_IMAGE ?= rhysd/actionlint:1.7.12@sha256:b1934ee5f1c509618f2508e6eb47ee0d3520686341fec936f3b79331f9315667
SHELLCHECK_IMAGE ?= koalaman/shellcheck:v0.11.0@sha256:61862eba1fcf09a484ebcc6feea46f1782532571a34ed51fedf90dd25f925a8d
HARNESS_SKILLS_SYNC_SCRIPT := bash ./scripts/harness-skills-sync.sh
AGENT_ROLES_SYNC_SCRIPT := bash ./scripts/agent-roles-sync.sh
CODEX_AGENTS_SYNC_SCRIPT := bash ./scripts/codex-agents-sync.sh
# profile:object-storage:start
S3_CONFORMANCE_TEST := go test -mod=readonly -vet=off -tags=integration ./test/s3conformance -run '^TestS3ObjectStorageConformanceRequiresProviderCertification$$' -count=1
# profile:object-storage:end
# profile:database-postgres:start
MIGRATION_HISTORY_CHECK_SCRIPT := bash ./scripts/ci/migration-history-check.sh
# profile:database-postgres:end
TEMPLATE_OWNED_PURITY_CHECK_SCRIPT := bash ./scripts/ci/template-owned-purity-check.sh
TEMPLATE ?= ../go-service-template-rest

.DEFAULT_GOAL := help

# One same-target A/B on the 10-core/16-GiB reference Mac measured 138.7s
# serial versus 294.6s with make -j4. Re-measure after host, toolchain, or
# aggregate membership changes before enabling parallel prerequisites.
.NOTPARALLEL: mod-check lint-deep openapi-check proto-check

.PHONY: help template-init template-init-check \
	tidy fmt mod-check mod-tidy-check mod-verify fmt-check test test-watch test-race test-integration \
	lint lint-deep lint-fast deadcode nilaway modernize-check test-parallelism-check govulncheck gosec secret-scan secret-scan-history \
	actionlint shellcheck dockerfile-check \
	openapi-generate openapi-drift-check openapi-reference-compile openapi-runtime-contract-check openapi-lint openapi-validate openapi-breaking openapi-check \
	proto-format proto-format-check proto-lint proto-generate proto-drift-check proto-breaking proto-check \
	sqlc-check runtime-image-build container-security run build build-pgo docker-build docker-run vendor claude-skills-sync claude-skills-check qwen-skills-sync qwen-skills-check agent-roles-sync agent-roles-check codex-agents-sync codex-agents-check \
	template-sync template-sync-check template-owned-purity-check
# profile:object-storage:start
.PHONY: test-s3-conformance-amazon test-s3-conformance-r2
# profile:object-storage:end
# profile:messaging-nats-jetstream:start
.PHONY: run-worker build-worker test-messaging-race
# profile:messaging-nats-jetstream:end
# profile:outbox-postgres:start
.PHONY: run-outbox-relay build-outbox-relay test-outbox-race
# profile:outbox-postgres:end
# profile:jobs-postgres:start
.PHONY: run-jobs-worker build-jobs-worker
# profile:jobs-postgres:end
# profile:webhooks-durable:start
.PHONY: test-webhook-race
# profile:webhooks-durable:end
# profile:database-postgres:start
.PHONY: sqlc-generate migration-history-check migration-check migration-validate compose-up compose-down
# profile:database-postgres:end

help:
	@echo "Setup and everyday development:"
	@echo "  make template-init MODULE=github.com/acme/service CODEOWNER=@acme/team"
	@echo "  make fmt-check | lint | test"
	@echo "  make agent-roles-check | codex-agents-check | claude-skills-check | qwen-skills-check"
	@echo "  make template-sync-check TEMPLATE=<path>   # drift against the template instructions"
	@echo "  make template-sync TEMPLATE=<path>         # adopt committed template instructions"
	@echo "  make run"
	@echo "  make build | build-pgo PGO_PROFILE=<cpu.pprof>"
# profile:messaging-nats-jetstream:start
	@echo "  make run-worker | build-worker"
	@echo "  make test-messaging-race"
# profile:messaging-nats-jetstream:end

# profile:outbox-postgres:start
	@echo "  make run-outbox-relay | build-outbox-relay"
	@echo "  make test-outbox-race"
# profile:outbox-postgres:end
# profile:webhooks-durable:start
	@echo "  make test-webhook-race"
# profile:webhooks-durable:end
	@echo ""
	@echo "Focused validation:"
	@echo "  make test | test-race | test-integration"
	@echo "  make mod-check | mod-tidy-check | mod-verify"
	@echo "  make lint | lint-deep | lint-fast | govulncheck | gosec | secret-scan | secret-scan-history"
	@echo "  make openapi-check"
	@echo "  make proto-check"
# profile:database-postgres:start
	@echo "  make sqlc-check | migration-validate"
# profile:database-postgres:end
	@echo "  make docker-build | container-security"
	@echo ""
	@echo "Reference: docs/build-test-and-development-commands.md"

template-init:
	@if [ -n "$(MODULE)" ]; then \
		CODEOWNER="$(CODEOWNER)" bash ./scripts/init-module.sh "$(MODULE)"; \
	else \
		CODEOWNER="$(CODEOWNER)" bash ./scripts/init-module.sh; \
	fi

template-init-check:
	bash ./scripts/ci/init-module-contract-check.sh

template-owned-purity-check:
	$(TEMPLATE_OWNED_PURITY_CHECK_SCRIPT)

# TEMPLATE points at a checkout of the template that owns the instructions.
# Run the source script from that checkout so a stale target copy never controls
# its own upgrade.
template-sync-check:
	bash "$(TEMPLATE)/scripts/template-sync.sh" --check --from "$(TEMPLATE)" --repo .

template-sync:
	bash "$(TEMPLATE)/scripts/template-sync.sh" --apply --from "$(TEMPLATE)" --repo .

claude-skills-sync:
	$(HARNESS_SKILLS_SYNC_SCRIPT) claude --apply --repo .

claude-skills-check:
	$(HARNESS_SKILLS_SYNC_SCRIPT) claude --check --repo .

qwen-skills-sync:
	$(HARNESS_SKILLS_SYNC_SCRIPT) qwen --apply --repo .

qwen-skills-check:
	$(HARNESS_SKILLS_SYNC_SCRIPT) qwen --check --repo .

agent-roles-sync:
	$(AGENT_ROLES_SYNC_SCRIPT) --apply --repo .

agent-roles-check:
	$(AGENT_ROLES_SYNC_SCRIPT) --check --repo .

codex-agents-sync:
	$(CODEX_AGENTS_SYNC_SCRIPT) --apply --repo .

codex-agents-check:
	$(CODEX_AGENTS_SYNC_SCRIPT) --check --repo .

tidy:
	go mod tidy

fmt:
	$(GO_TOOL) goimports -w $(GOIMPORTS_FILES)
	$(GO_TOOL) gofumpt -w $(GOFUMPT_FILES)

mod-check: mod-tidy-check mod-verify

mod-tidy-check:
	GOFLAGS= go mod tidy -diff
	GOFLAGS= go -C tools mod tidy -diff
	@test "$$(awk '/^go / {print $$2; exit}' go.mod)" = "$$(awk '/^go / {print $$2; exit}' tools/go.mod)" || { \
		echo "go.mod and tools/go.mod must use the same Go version"; \
		exit 1; \
	}

mod-verify:
	go mod verify
	go -C tools mod verify

fmt-check:
	@unformatted="$$( $(GO_TOOL) goimports -l $(GOIMPORTS_FILES) )"; \
	if [ -n "$$unformatted" ]; then \
		echo "goimports required for:"; \
		echo "$$unformatted"; \
		echo "run 'make fmt'"; \
		exit 1; \
	fi
	@gofumpt_unformatted="$$( $(GO_TOOL) gofumpt -l $(GOFUMPT_FILES) )"; \
	if [ -n "$$gofumpt_unformatted" ]; then \
		echo "gofumpt required for:"; \
		echo "$$gofumpt_unformatted"; \
		echo "run 'make fmt'"; \
		exit 1; \
	fi

test:
	$(GO_TOOL) gotestsum --format=pkgname-and-test-fails -- -vet=off ./...

test-watch:
	$(GO_TOOL) gotestsum --watch --format=pkgname-and-test-fails -- -vet=off ./...

test-race:
	go test -vet=off -race ./...

# profile:messaging-nats-jetstream:start
test-messaging-race:
	go test -vet=off -p=1 -count=1 -race -tags=integration $(MESSAGING_RACE_PACKAGES) -run '^(TestOutboxWorkerPublishesStableWireIdentityAndTrace|TestNATSWorkerRegistrationIsSingleton|TestNATSNativeConsumeSurvivesBrokerRestart|TestWorkerUsesNativeBoundedConsumeContextsAndJoinsDrain|TestTypedPublisherAndHandlerHideBrokerFields|TestNATSPublishDispatchCancellationAndNoRetry|TestNATSWorkerComposition|TestNATSWorkerForcedShutdownDoesNotRaceHandlerCleanup|TestNATSConsumerSaturation|TestNATSForcedShutdownRedelivers|TestNATSGracefulDrain)$$'
# profile:messaging-nats-jetstream:end

# profile:outbox-postgres:start
test-outbox-race:
	go test -vet=off -p=1 -count=1 -race -tags=integration $(OUTBOX_RACE_PACKAGES) -run '^TestPostgresOutbox'
# profile:outbox-postgres:end

# profile:webhooks-durable:start
test-webhook-race:
	go test -vet=off -p=1 -count=1 -race -tags=integration $(WEBHOOK_RACE_PACKAGES) -run '^Test(PostgresWebhookAcceptance|WebhookNetwork)'
# profile:webhooks-durable:end

test-integration:
	go test -p=1 -count=1 -tags=integration $(INTEGRATION_PACKAGES)
# profile:messaging-nats-jetstream:start
	$(MAKE) test-messaging-race
# profile:messaging-nats-jetstream:end
# profile:outbox-postgres:start
	$(MAKE) test-outbox-race
# profile:outbox-postgres:end
# profile:webhooks-durable:start
	$(MAKE) test-webhook-race
# profile:webhooks-durable:end

# profile:jobs-postgres:start
run-jobs-worker:
	@set -a; \
	if [ -f .env ]; then . ./.env; fi; \
	set +a; \
	go run $(JOBS_WORKER_CMD)

build-jobs-worker:
	mkdir -p bin
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags='-s -w' -o $(JOBS_WORKER_BINARY) $(JOBS_WORKER_CMD)
# profile:jobs-postgres:end

# profile:outbox-postgres:start
run-outbox-relay:
	@set -a; \
	if [ -f .env ]; then . ./.env; fi; \
	set +a; \
	go run $(OUTBOX_RELAY_CMD)
# profile:outbox-postgres:end

lint:
	$(GOLANGCI_LINT) run --allow-serial-runners --concurrency=$(LINT_CONCURRENCY) --timeout=3m

lint-deep: deadcode nilaway

lint-fast:
	$(GOLANGCI_LINT) run --allow-serial-runners --fast-only --new-from-rev=$(LINT_BASE_REF) --concurrency=$(LINT_CONCURRENCY) --timeout=3m

deadcode:
	$(GO_TOOL) deadcode -test -tags=integration ./...

nilaway:
	@module_path="$$(go list -m)"; \
	printf '$(GO_TOOL) nilaway -include-pkgs=%s -test ./...\n' "$$module_path"; \
	$(GO_TOOL) nilaway -include-pkgs="$$module_path" -test ./...

modernize-check:
	go fix -diff ./...

test-parallelism-check:
	$(GOLANGCI_LINT) run --allow-serial-runners --concurrency=$(LINT_CONCURRENCY) --enable-only=paralleltest,tparallel --timeout=3m --max-issues-per-linter=0 --max-same-issues=0

actionlint:
	docker run --rm --read-only --network none \
		-v "$(CURDIR):/src:ro" \
		-w /src \
		"$(ACTIONLINT_IMAGE)"

shellcheck:
	@test -n "$(SHELL_FILES)" || { echo "no shell scripts found; skipping ShellCheck"; exit 0; }
	docker run --rm --read-only --network none \
		-v "$(CURDIR):/src:ro" \
		-w /src \
		"$(SHELLCHECK_IMAGE)" \
		-x \
		-- $(SHELL_FILES)

dockerfile-check:
	docker buildx build --check -f build/docker/Dockerfile .

govulncheck:
	$(GO_TOOL) govulncheck ./...

gosec:
	GOSECGOVERSION=go$(GO_REQUIRED_VERSION) $(GO_TOOL) gosec $(if $(strip $(GOMAXPROCS)),-concurrency=$(GOMAXPROCS)) -quiet -exclude-generated -exclude-dir=.agents -exclude-dir=.cache -exclude-dir=.artifacts ./...

secret-scan:
	$(GO_TOOL) gitleaks dir --no-banner --redact --verbose --exit-code 1 --config .gitleaks.toml .
	@base="$$(git merge-base "$(SECRET_SCAN_BASE_REF)" HEAD)"; \
	$(GO_TOOL) gitleaks git --no-banner --redact --verbose --exit-code 1 --config .gitleaks.toml --log-opts="$$base..HEAD" .

secret-scan-history:
	$(GO_TOOL) gitleaks git --no-banner --redact --verbose --exit-code 1 --config .gitleaks.toml .

# profile:database-postgres:start
sqlc-generate:
	@if ! find internal/infra/postgres/queries -type f -name '*.sql' -print -quit 2>/dev/null | grep -q .; then \
		echo "no sqlc query sources; skipping sqlc generation"; \
	else \
		$(GO_TOOL) sqlc generate -f internal/infra/postgres/sqlc.yaml; \
	fi
# profile:database-postgres:end

sqlc-check:
	@set -e; if find internal/infra/postgres/queries -type f -name '*.sql' -print -quit 2>/dev/null | grep -q .; then \
		before="$$(find internal/infra/postgres/sqlcgen -type f -exec shasum -a 256 {} + | LC_ALL=C sort)"; \
		$(MAKE) sqlc-generate; \
		after="$$(find internal/infra/postgres/sqlcgen -type f -exec shasum -a 256 {} + | LC_ALL=C sort)"; \
		test "$$before" = "$$after" || { git diff -- internal/infra/postgres/sqlcgen; exit 1; }; \
		expected="$$(mktemp)"; actual="$$(mktemp)"; trap 'rm -f "$$expected" "$$actual"' EXIT; \
		find internal/infra/postgres/queries -type f -name '*.sql' -exec basename {} .sql \; | sort >"$$expected"; \
		find internal/infra/postgres/sqlcgen -type f -name '*.sql.go' -exec basename {} .sql.go \; | sort >"$$actual"; \
		diff -u "$$expected" "$$actual"; \
	elif find internal/infra/postgres/sqlcgen -type f -name '*.go' -print -quit 2>/dev/null | grep -q .; then \
		echo "sqlc output exists without query sources" >&2; exit 1; \
	fi

# profile:database-postgres:start
migration-history-check:
	BASE_REF="$(BASE_REF)" HEAD_REF="$(HEAD_REF)" MIGRATION_HISTORY_MODE="$(if $(MIGRATION_HISTORY_MODE),$(MIGRATION_HISTORY_MODE),worktree)" $(MIGRATION_HISTORY_CHECK_SCRIPT)

migration-check: migration-history-check
	$(GO_TOOL) goose -dir migrations validate
# profile:database-postgres:end

openapi-generate:
	go generate $(OPENAPI_PACKAGES)

openapi-drift-check:
	@before="$$(shasum -a 256 $(OPENAPI_GENERATED_FILES))"; \
	$(MAKE) openapi-generate; \
	after="$$(shasum -a 256 $(OPENAPI_GENERATED_FILES))"; \
	test "$$before" = "$$after" || { git diff -- $(OPENAPI_GENERATED_FILES); exit 1; }

openapi-reference-compile:
	@if [ -n "$(REFERENCE_OPENAPI_PACKAGE)" ]; then \
		go test -vet=off $(REFERENCE_OPENAPI_PACKAGE); \
	fi

openapi-runtime-contract-check:
	@report="$$(mktemp)"; \
	trap 'rm -f "$$report"' EXIT; \
	if ! go test -vet=off -json ./internal/infra/http -run '^TestOpenAPIRuntimeContract' -count=1 >"$$report" 2>&1; then \
		cat "$$report"; \
		exit 1; \
	fi; \
	if ! grep -Eq '"Action":"run".*"Test":"TestOpenAPIRuntimeContract[^"]*"' "$$report"; then \
		cat "$$report"; \
		echo "no OpenAPI runtime contract tests matched"; \
		exit 1; \
	fi

openapi-lint:
	REDOCLY_SUPPRESS_UPDATE_NOTICE=true REDOCLY_TELEMETRY=off npm_config_prefer_offline=true $(REDOCLY_CLI) lint --config .redocly.yaml $(OPENAPI_FILES)

openapi-validate:
	@set -e; for file in $(OPENAPI_FILES); do $(GO_TOOL) validate -- "$$file"; done

OPENAPI_BREAKING_APPROVALS ?= api/openapi/breaking-changes-approvals.txt

openapi-breaking:
	@test -n "$(BASE_OPENAPI)" || { echo "BASE_OPENAPI is required"; exit 1; }
	@if [ -s "$(OPENAPI_BREAKING_APPROVALS)" ]; then \
		$(GO_TOOL) oasdiff breaking --fail-on ERR --err-ignore "$(OPENAPI_BREAKING_APPROVALS)" $(BASE_OPENAPI) $(OPENAPI_FILE); \
	else \
		$(GO_TOOL) oasdiff breaking --fail-on ERR $(BASE_OPENAPI) $(OPENAPI_FILE); \
	fi

openapi-check: openapi-drift-check openapi-reference-compile openapi-runtime-contract-check openapi-lint openapi-validate

# profile:grpc:start
proto-format:
	@if find api/proto -type f -name '*.proto' -print -quit 2>/dev/null | grep -q .; then $(GO_TOOL) buf format -w; fi
	cd examples/grpc-reference-service && go tool -modfile=../../tools/go.mod buf format -w

proto-format-check:
	@if find api/proto -type f -name '*.proto' -print -quit 2>/dev/null | grep -q .; then $(GO_TOOL) buf format --diff --exit-code; fi
	cd examples/grpc-reference-service && go tool -modfile=../../tools/go.mod buf format --diff --exit-code

proto-lint:
	@if find api/proto -type f -name '*.proto' -print -quit 2>/dev/null | grep -q .; then $(GO_TOOL) buf lint; fi
	cd examples/grpc-reference-service && go tool -modfile=../../tools/go.mod buf lint

proto-generate:
	@if find api/proto -type f -name '*.proto' -print -quit 2>/dev/null | grep -q .; then $(GO_TOOL) buf generate; fi
	cd examples/grpc-reference-service && go tool -modfile=../../tools/go.mod buf generate

proto-drift-check:
	@before="$$(find internal/gen/proto examples/grpc-reference-service/internal/gen/proto -type f -exec shasum -a 256 {} + 2>/dev/null | LC_ALL=C sort)"; \
	$(MAKE) proto-generate; \
	after="$$(find internal/gen/proto examples/grpc-reference-service/internal/gen/proto -type f -exec shasum -a 256 {} + 2>/dev/null | LC_ALL=C sort)"; \
	test "$$before" = "$$after" || { git diff -- internal/gen/proto examples/grpc-reference-service/internal/gen/proto; exit 1; }

proto-breaking:
	@test -n "$(BASE_REF)" || { echo "BASE_REF is required" >&2; exit 2; }
	@if find api/proto -type f -name '*.proto' -print -quit 2>/dev/null | grep -q .; then \
		$(GO_TOOL) buf breaking api/proto --against '.git#ref=$(BASE_REF),subdir=api/proto'; \
	fi

proto-check: proto-format-check proto-lint proto-drift-check
# profile:grpc:end

# profile:database-postgres:start
migration-validate:
	@project="service-migration-$$(date +%s)-$$$$"; \
	runtime=""; \
	compose() { POSTGRES_PORT=0 docker compose -p "$$project" -f env/docker-compose.yml "$$@"; }; \
	cleanup() { \
		if [ -n "$$runtime" ]; then docker rm -f "$$runtime" >/dev/null 2>&1 || true; fi; \
		compose down -v --remove-orphans >/dev/null 2>&1 || true; \
	}; \
	trap cleanup EXIT INT TERM; \
	compose up -d --wait postgres; \
	address="$$(compose port postgres 5432)"; \
	port="$${address##*:}"; \
	test -n "$$port" || { echo "failed to resolve rehearsal Postgres port"; exit 1; }; \
	dsn="postgres://app:app@localhost:$$port/app?sslmode=disable"; \
	$(GO_TOOL) goose -dir migrations validate; \
	PGTEST_POSTGRES_DSN="$$dsn" REQUIRE_DOCKER=1 $(GO) test -vet=off -count=1 -tags=integration ./test \
		-run '^TestPostgres(MigrateRepositorySourceRehearsal|HTTPIdempotencySchemaReplacementIsFailClosed)$$'; \
	image="$(RUNTIME_IMAGE)"; \
	if [ -z "$$image" ]; then \
		image="$(SERVICE_NAME):migration"; \
		$(MAKE) runtime-image-build RUNTIME_IMAGE="$$image" || exit 1; \
	fi; \
	docker run --rm --network "$${project}_default" \
		-e APP__POSTGRES__ENABLED=true \
		-e APP__POSTGRES__DSN="postgres://app:app@postgres:5432/app?sslmode=disable" \
		--entrypoint /migrate "$$image"; \
	command -v curl >/dev/null 2>&1 || { echo "curl is required for runtime image readiness validation"; exit 1; }; \
	runtime="$${project}-runtime"; \
	docker run -d --name "$$runtime" \
		--network "$${project}_default" \
		-p 127.0.0.1::8080 \
		--read-only \
		--cap-drop=ALL \
		--security-opt=no-new-privileges \
		-e APP__POSTGRES__ENABLED=true \
		-e APP__POSTGRES__DSN="postgres://app:app@postgres:5432/app?sslmode=disable" \
		"$$image" >/dev/null; \
	address="$$(docker port "$$runtime" 8080/tcp | head -n 1)"; \
	port="$${address##*:}"; \
	test -n "$$port" || { echo "failed to resolve runtime service port"; docker logs "$$runtime"; exit 1; }; \
	ready=false; \
	attempt=0; \
	while [ "$$attempt" -lt 45 ]; do \
		if curl -fs --max-time 2 "http://127.0.0.1:$$port/health/ready" >/dev/null; then ready=true; break; fi; \
		if [ "$$(docker inspect -f '{{.State.Running}}' "$$runtime")" != "true" ]; then break; fi; \
		attempt=$$((attempt + 1)); \
		sleep 1; \
	done; \
	if [ "$$ready" != "true" ]; then echo "runtime image did not become ready"; docker logs "$$runtime"; exit 1; fi; \
	if [ -n "$(RUNTIME_EXPECTED_VERSION)" ] && ! docker logs "$$runtime" 2>&1 | grep -Fq "\"service.version\":\"$(RUNTIME_EXPECTED_VERSION)\""; then \
		echo "runtime image did not report expected version $(RUNTIME_EXPECTED_VERSION)"; \
		docker logs "$$runtime"; \
		exit 1; \
	fi; \
	docker stop --time 45 "$$runtime" >/dev/null; \
	exit_code="$$(docker inspect -f '{{.State.ExitCode}}' "$$runtime")"; \
	test "$$exit_code" = "0" || { echo "runtime image exited with code $$exit_code after SIGTERM"; docker logs "$$runtime"; exit 1; }
# profile:database-postgres:end

container-security:
	@image="$(CONTAINER_IMAGE)"; \
	if [ -z "$$image" ]; then image="$(SERVICE_NAME):ci"; docker build -f build/docker/Dockerfile -t "$$image" .; fi; \
	docker run --rm \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v "$(TRIVY_CACHE_VOLUME):/root/.cache/trivy" \
		-e DOCKER_HOST=unix:///var/run/docker.sock \
		"$(TRIVY_IMAGE)" image \
		--cache-dir /root/.cache/trivy \
		--quiet \
		--severity HIGH,CRITICAL \
		--ignore-unfixed \
		--exit-code 1 \
		--format table \
		"$$image"

runtime-image-build:
	bash ./scripts/ci/runtime-image-build.sh "$(RUNTIME_IMAGE)"

# profile:object-storage:start
test-s3-conformance-amazon:
	@REQUIRE_S3_CONFORMANCE=1 S3_CONFORMANCE_PROVIDER=amazon_s3 $(S3_CONFORMANCE_TEST)

test-s3-conformance-r2:
	@REQUIRE_S3_CONFORMANCE=1 S3_CONFORMANCE_PROVIDER=cloudflare_r2 $(S3_CONFORMANCE_TEST)
# profile:object-storage:end

run:
	@set -a; \
	if [ -f .env ]; then . ./.env; fi; \
	set +a; \
	go run $(SERVICE_CMD)

# profile:messaging-nats-jetstream:start
run-worker:
	@set -a; \
	if [ -f .env ]; then . ./.env; fi; \
	set +a; \
	go run $(WORKER_CMD)
# profile:messaging-nats-jetstream:end

# profile:outbox-postgres:start
build-outbox-relay:
	mkdir -p bin
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags='-s -w' -o $(OUTBOX_RELAY_BINARY) $(OUTBOX_RELAY_CMD)
# profile:outbox-postgres:end

build:
	@if [ "$(PGO_PROFILE)" != "off" ]; then \
		if [ -z "$(PGO_PROFILE)" ] || [ "$(PGO_PROFILE)" = "auto" ]; then \
			echo "PGO_PROFILE must be off or name an explicit representative CPU profile" >&2; \
			exit 2; \
		fi; \
		if [ ! -f "$(PGO_PROFILE)" ]; then \
			echo "PGO profile does not exist: $(PGO_PROFILE)" >&2; \
			exit 2; \
		fi; \
		if ! $(GO) tool pprof -raw "$(PGO_PROFILE)" >/dev/null; then \
			echo "PGO profile is not a valid CPU profile: $(PGO_PROFILE)" >&2; \
			exit 2; \
		fi; \
	fi
	mkdir -p bin
	CGO_ENABLED=0 $(GO) build -pgo="$(PGO_PROFILE)" -trimpath -ldflags='-s -w' -o $(BINARY) $(SERVICE_CMD)

# profile:messaging-nats-jetstream:start
build-worker:
	mkdir -p bin
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags='-s -w' -o $(WORKER_BINARY) $(WORKER_CMD)
# profile:messaging-nats-jetstream:end

build-pgo:
	@if [ -z "$(PGO_PROFILE)" ] || [ "$(PGO_PROFILE)" = "off" ] || [ "$(PGO_PROFILE)" = "auto" ]; then \
		echo "PGO_PROFILE must name an explicit representative CPU profile" >&2; \
		exit 2; \
	fi
	$(MAKE) build GO="$(GO)" PGO_PROFILE="$(PGO_PROFILE)"

docker-build:
	docker build --build-arg PGO_PROFILE="$(PGO_PROFILE)" -f build/docker/Dockerfile -t $(SERVICE_NAME):local .

docker-run:
	docker run --rm --stop-timeout 45 -p 8080:8080 --env-file .env $(SERVICE_NAME):local

# profile:database-postgres:start
compose-up:
	docker compose -f env/docker-compose.yml up -d --wait

compose-down:
	docker compose -f env/docker-compose.yml down -v
# profile:database-postgres:end

vendor:
	go mod vendor
