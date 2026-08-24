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
EXTERNAL_OPENAPI_FILES := $(wildcard api/external/*/openapi.yaml)
EXTERNAL_OPENAPI_PACKAGES := $(patsubst api/external/%/openapi.yaml,./internal/infra/%/internal/openapi,$(EXTERNAL_OPENAPI_FILES))
EXTERNAL_OPENAPI_GENERATED_FILES := $(patsubst api/external/%/openapi.yaml,internal/infra/%/internal/openapi/client.gen.go,$(EXTERNAL_OPENAPI_FILES))
OPENAPI_FILES := $(OPENAPI_FILE) $(REFERENCE_OPENAPI_FILE) $(EXTERNAL_OPENAPI_FILES)
OPENAPI_PACKAGES := ./internal/openapi $(REFERENCE_OPENAPI_PACKAGE) $(EXTERNAL_OPENAPI_PACKAGES)
OPENAPI_GENERATED_FILES := internal/openapi/openapi.gen.go $(if $(REFERENCE_OPENAPI_FILE),examples/reference-service/internal/openapi/openapi.gen.go) $(EXTERNAL_OPENAPI_GENERATED_FILES)
INTEGRATION_INIT_VARS := NAME TRANSPORT CONTRACT TARGET AUTH
SHELL_FILES = $(shell git ls-files --cached --others --exclude-standard -- '*.sh' 2>/dev/null | awk '!/^(\.agents|\.cache|vendor)\//' | while IFS= read -r file; do [ -f "$$file" ] && printf '%s\n' "$$file"; done)
REDOCLY_CLI_VERSION := 2.40.0
REDOCLY_CLI ?= npx --yes @redocly/cli@$(REDOCLY_CLI_VERSION)
GO_TOOL := go tool -modfile=tools/go.mod
GOLANGCI_LINT ?= $(GO_TOOL) golangci-lint
GO_REQUIRED_VERSION = $(shell awk '/^go / {print $$2; exit}' go.mod)
INTEGRATION_PACKAGES := ./test/...
# profile:http-idempotency-postgres:start
INTEGRATION_PACKAGES += ./internal/infra/postgresidempotency
# profile:http-idempotency-postgres:end
# profile:messaging-nats-jetstream:start
INTEGRATION_PACKAGES += ./internal/infra/natsjs ./cmd/worker/internal/bootstrap
MESSAGING_RACE_PACKAGES := ./internal/infra/natsjs ./cmd/worker/internal/bootstrap ./test
# profile:messaging-nats-jetstream:end
# profile:outbox-postgres:start
OUTBOX_RACE_PACKAGES := ./test
# profile:outbox-postgres:end
# profile:webhooks-durable:start
WEBHOOK_RACE_PACKAGES := ./test
# profile:webhooks-durable:end
INTEGRATION_RACE_PACKAGES := $(sort $(MESSAGING_RACE_PACKAGES) $(OUTBOX_RACE_PACKAGES) $(WEBHOOK_RACE_PACKAGES))
LINT_BASE_REF ?= origin/main
LINT_CONCURRENCY ?= 2
PKG ?=
FILES ?=
ALLOW_HEAVY ?=
export ALLOW_HEAVY
LINT_PACKAGE_LINTERS := govet,errcheck,staticcheck,ineffassign,unused,bodyclose,nilerr,errorlint,forcetypeassert,noctx
LINT_PR_LINTERS := $(LINT_PACKAGE_LINTERS),depguard,sqlclosecheck,exhaustive,containedctx,contextcheck,iface,interfacebloat,ireturn,rowserrcheck,wrapcheck
PR_LINT_TARGET ?= lint-pr
SECRET_SCAN_BASE_REF ?= $(if $(strip $(BASE_REF)),$(BASE_REF),origin/main)
GITLEAKS_FLAGS := --no-banner --redact --verbose --exit-code 1 --config .gitleaks.toml --baseline-path .gitleaks.baseline.json
GITLEAKS ?= $(GO_TOOL) gitleaks
AUDIT_RUNTIME_IMAGE ?= $(SERVICE_NAME):audit-full-manual
VERIFY_RUNTIME_IMAGE ?= $(SERVICE_NAME):verify

# Not a security boundary: stop an agent from launching a costly matrix by accident.
# GitHub Actions and other CI systems set CI=true, which is enough.
HEAVY_GUARD = @if [ "$(ALLOW_HEAVY)" != "1" ] && [ "$(CI)" != "true" ]; then printf 'refusing %s: set ALLOW_HEAVY=1 (CI sets CI=true)\n' "$@"; exit 2; fi
REQUIRE_PKG = @if [ -z "$(strip $(PKG))" ]; then printf '%s requires PKG=./path/to/package (use %s for the full module)\n' "$@" "test-all/lint-all/check"; exit 2; fi
REQUIRE_FILES = @if [ -z "$(strip $(FILES))" ]; then printf '%s requires FILES="file.go ..."\n' "$@"; exit 2; fi

TRIVY_IMAGE ?= aquasec/trivy:0.72.0@sha256:cffe3f5161a47a6823fbd23d985795b3ed72a4c806da4c4df16266c02accdd6f
TRIVY_CACHE_VOLUME ?= trivy-cache
ACTIONLINT_VERSION := 1.7.12
ACTIONLINT_IMAGE ?= rhysd/actionlint:$(ACTIONLINT_VERSION)@sha256:b1934ee5f1c509618f2508e6eb47ee0d3520686341fec936f3b79331f9315667
SHELLCHECK_VERSION := 0.11.0
SHELLCHECK_IMAGE ?= koalaman/shellcheck:v$(SHELLCHECK_VERSION)@sha256:61862eba1fcf09a484ebcc6feea46f1782532571a34ed51fedf90dd25f925a8d
HARNESS_SKILLS_SYNC_SCRIPT := bash ./scripts/harness-skills-sync.sh
AGENT_ROLES_SYNC_SCRIPT := bash ./scripts/agent-roles-sync.sh
CODEX_AGENTS_SYNC_SCRIPT := bash ./scripts/codex-agents-sync.sh
INTEGRATION_INIT_SCRIPT := bash ./scripts/integration-init.sh
INTEGRATION_INIT_CHECK_SCRIPT := bash ./scripts/ci/integration-init-check.sh
VERIFY_SCRIPT := bash ./scripts/ci/verify.sh
RUNTIME_IMAGE_CHECK_SCRIPT := bash ./scripts/ci/runtime-image-check.sh
# profile:object-storage:start
S3_CONFORMANCE_TEST := go test -mod=readonly -vet=off -tags=integration ./test/s3conformance -run '^TestS3ObjectStorageConformanceRequiresProviderCertification$$' -count=1
# profile:object-storage:end
# profile:database-postgres:start
MIGRATION_HISTORY_CHECK_SCRIPT := bash ./scripts/ci/migration-history-check.sh
MIGRATION_VALIDATE_SCRIPT := bash ./scripts/ci/migration-validate.sh
# profile:database-postgres:end
TEMPLATE_OWNED_PURITY_CHECK_SCRIPT := bash ./scripts/ci/template-owned-purity-check.sh
TEMPLATE ?= ../go-service-template-rest

.DEFAULT_GOAL := help

# The broad-aggregate reference-host A/B recorded by 5d8d9652e rejected -j4.
# Only targets with prerequisites belong here; re-measure after their membership,
# host, or toolchain changes.
.NOTPARALLEL: check check-go check-go-pr unit-check tools-dependencies-check mod-check openapi-check proto-check

.PHONY: help template-init template-init-check integration-init integration-init-check \
	tidy fmt mod-check root-mod-check tools-mod-check tools-smoke tools-dependencies-check mod-tidy-check mod-verify fmt-check fmt-files-check unit-check plan verify verify-check check check-go check-go-pr check-openapi check-sqlc check-instructions check-delivery check-security-go audit-full-manual changed-surfaces-check \
	test test-package test-all test-watch test-race test-integration test-integration-db test-integration-messaging test-integration-process test-integration-race \
	lint lint-package lint-changed lint-pr lint-all lint-deep lint-fast deadcode nilaway modernize-check test-parallelism-check \
	govulncheck gosec secret-scan secret-scan-history \
	actionlint actionlint-fast shellcheck shellcheck-fast dockerfile-check \
	openapi-generate openapi-drift-check openapi-runtime-contract-check openapi-lint openapi-validate openapi-breaking openapi-check \
	proto-format proto-format-check proto-lint proto-generate proto-drift-check proto-breaking proto-check check-proto \
	sqlc-check runtime-image-build runtime-image-check container-security run build build-pgo docker-build docker-run vendor claude-skills-sync claude-skills-check qwen-skills-sync qwen-skills-check agent-roles-sync agent-roles-check codex-agents-sync codex-agents-check \
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
	@echo "  make template-init MODULE=github.com/acme/service CODEOWNER=@acme/team AGENT_HARNESS=core"
	@echo "  go test -vet=off ./internal/<package>     # edit loop"
	@echo "  make unit-check PKG=./internal/<package> FILES='...'"
	@echo "  make plan or make verify                 # explain or run the surface-aware final route"
	@echo "  make check                               # explicit full-repository aggregate"
	@echo "  ALLOW_HEAVY=1 make audit-full-manual     # rare template/release audit; refused otherwise"
	@echo "  make lint-fast PKG=./internal/config      # local changed-code signal; not a lint claim"
	@echo "  make integration-init NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=none"
	@echo "  make agent-roles-check, make codex-agents-check, make claude-skills-check, or make qwen-skills-check"
	@echo "  make template-sync-check TEMPLATE=<path>   # drift against the template instructions"
	@echo "  make template-sync TEMPLATE=<path>         # adopt committed template instructions"
	@echo "  make run"
	@echo "  make build or make build-pgo PGO_PROFILE=<cpu.pprof>"
# profile:messaging-nats-jetstream:start
	@echo "  make run-worker or make build-worker"
	@echo "  make test-messaging-race"
# profile:messaging-nats-jetstream:end

# profile:outbox-postgres:start
	@echo "  make run-outbox-relay or make build-outbox-relay"
	@echo "  make test-outbox-race"
# profile:outbox-postgres:end
# profile:jobs-postgres:start
	@echo "  make run-jobs-worker or make build-jobs-worker"
# profile:jobs-postgres:end
# profile:webhooks-durable:start
	@echo "  make test-webhook-race"
# profile:webhooks-durable:end
	@echo ""
	@echo "Focused validation:"
	@echo "  make test-package PKG=./pkg or make test-all"
	@echo "  make lint-changed PKG=./pkg, make lint-pr, make lint-all, or make lint-deep"
	@echo "  make test-race, make test-integration, or a focused test-integration-{db,messaging,process,race} target"
	@echo "  make root-mod-check, make tools-mod-check, or make mod-check"
	@echo "  make govulncheck, make gosec, make secret-scan, or make secret-scan-history"
	@echo "  local agents: make actionlint-fast or make shellcheck-fast"
	@echo "  make openapi-check"
	@echo "  make proto-check"
# profile:database-postgres:start
	@echo "  make sqlc-check or make migration-validate"
# profile:database-postgres:end
	@echo "  make docker-build or make container-security"
	@echo ""
	@echo "Reference: docs/build-test-and-development-commands.md"

template-init:
	@if [ -n "$(MODULE)" ]; then \
		CODEOWNER="$(CODEOWNER)" bash ./scripts/init-module.sh "$(MODULE)"; \
	else \
		CODEOWNER="$(CODEOWNER)" bash ./scripts/init-module.sh; \
	fi

template-init-check:
	$(HEAVY_GUARD)
	bash ./scripts/ci/init-module-contract-check.sh

integration-init:
	@extra="$(strip $(foreach v,$(.VARIABLES),$(and $(filter command line,$(origin $(v))),$(if $(filter $(INTEGRATION_INIT_VARS),$(v)),,$(v)))))"; \
	if [ -n "$$extra" ]; then echo "unknown initializer variable(s): $$extra" >&2; exit 2; fi
	$(INTEGRATION_INIT_SCRIPT) "$(NAME)" "$(TRANSPORT)" "$(CONTRACT)" "$(TARGET)" "$(AUTH)"

integration-init-check:
	@if [ -z "$(strip $(INTEGRATION_INIT_ROWS))" ]; then \
		if [ "$(ALLOW_HEAVY)" != "1" ] && [ "$(CI)" != "true" ]; then printf 'refusing %s: set ALLOW_HEAVY=1 (CI sets CI=true)\n' "$@"; exit 2; fi; \
	fi
	MAKEFLAGS= MAKEOVERRIDES= INTEGRATION_INIT_ROWS= $(INTEGRATION_INIT_CHECK_SCRIPT) $(INTEGRATION_INIT_ROWS)

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
	@set -e; \
	files="$$(git ls-files --cached --others --exclude-standard -- '*.go' 2>/dev/null | awk '!/^(\.agents|\.cache|vendor)\//' | while IFS= read -r file; do [ -f "$$file" ] && printf '%s\n' "$$file"; done)"; \
	goimports_files="$$(printf '%s\n' "$$files" | grep -vE '^(internal/gen/proto/|examples/grpc-reference-service/internal/gen/proto/)')"; \
	gofumpt_files="$$(printf '%s\n' "$$goimports_files" | grep -vE '^(internal/openapi/openapi\.gen\.go$$|examples/reference-service/internal/openapi/openapi\.gen\.go$$|internal/infra/.*/internal/openapi/client\.gen\.go$$|internal/infra/postgres/sqlcgen/)')"; \
	$(GO_TOOL) goimports -w $$goimports_files; \
	$(GO_TOOL) gofumpt -w $$gofumpt_files

mod-check: root-mod-check tools-mod-check

root-mod-check:
	GOFLAGS= go mod tidy -diff
	go mod verify

tools-mod-check:
	GOFLAGS= go -C tools mod tidy -diff
	go -C tools mod verify
	@test "$$(awk '/^go / {print $$2; exit}' go.mod)" = "$$(awk '/^go / {print $$2; exit}' tools/go.mod)" || { \
		echo "go.mod and tools/go.mod must use the same Go version"; \
		exit 1; \
	}

tools-smoke:
	bash ./scripts/ci/tools-smoke.sh

tools-dependencies-check: tools-mod-check tools-smoke

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
	@set -e; \
	files="$$(git ls-files --cached --others --exclude-standard -- '*.go' 2>/dev/null | awk '!/^(\.agents|\.cache|vendor)\//' | while IFS= read -r file; do [ -f "$$file" ] && printf '%s\n' "$$file"; done)"; \
	goimports_files="$$(printf '%s\n' "$$files" | grep -vE '^(internal/gen/proto/|examples/grpc-reference-service/internal/gen/proto/)')"; \
	gofumpt_files="$$(printf '%s\n' "$$goimports_files" | grep -vE '^(internal/openapi/openapi\.gen\.go$$|examples/reference-service/internal/openapi/openapi\.gen\.go$$|internal/infra/.*/internal/openapi/client\.gen\.go$$|internal/infra/postgres/sqlcgen/)')"; \
	unformatted="$$( $(GO_TOOL) goimports -l $$goimports_files )"; \
	if [ -n "$$unformatted" ]; then echo "goimports required for:"; echo "$$unformatted"; echo "run 'make fmt'"; exit 1; fi; \
	gofumpt_unformatted="$$( $(GO_TOOL) gofumpt -l $$gofumpt_files )"; \
	if [ -n "$$gofumpt_unformatted" ]; then echo "gofumpt required for:"; echo "$$gofumpt_unformatted"; echo "run 'make fmt'"; exit 1; fi

# One focused aggregate. The leaves also let verify collapse package tests into
# test-all when a root dependency change already requires the wider oracle.
fmt-files-check:
	$(REQUIRE_FILES)
	@unformatted="$$( $(GO_TOOL) goimports -l $(FILES) )"; \
	if [ -n "$$unformatted" ]; then echo "goimports required for:"; echo "$$unformatted"; echo "run 'make fmt'"; exit 1; fi
	@gofumpt_unformatted="$$( $(GO_TOOL) gofumpt -l $(FILES) )"; \
	if [ -n "$$gofumpt_unformatted" ]; then echo "gofumpt required for:"; echo "$$gofumpt_unformatted"; echo "run 'make fmt'"; exit 1; fi

lint-changed:
	$(REQUIRE_PKG)
	@new_from=""; \
	if git rev-parse --verify "$(LINT_BASE_REF)" >/dev/null 2>&1; then \
		new_from="--new-from-merge-base=$(LINT_BASE_REF)"; \
	fi; \
	$(GOLANGCI_LINT) run --allow-serial-runners --enable-only=$(LINT_PACKAGE_LINTERS) $$new_from --concurrency=$(LINT_CONCURRENCY) --timeout=3m $(PKG)

unit-check: fmt-files-check test-package lint-changed

# Atomic CI owners; check remains the one local aggregate for an integrated tree.
check-go: fmt-check lint-all test-all mod-tidy-check

check-go-pr: fmt-check $(PR_LINT_TARGET) test-all

check-openapi: openapi-check

check-proto:
	@if ! find api/proto -type f -name '*.proto' -print -quit 2>/dev/null | grep -q . && [ ! -f examples/grpc-reference-service/buf.yaml ]; then \
		echo "not applicable: no protobuf sources"; \
	fi

check-sqlc: sqlc-check

check-instructions: template-owned-purity-check

check-delivery: actionlint shellcheck dockerfile-check

check-security-go: govulncheck gosec

changed-surfaces-check:
	bash ./scripts/ci/changed-surfaces.sh --self-test

plan:
	$(VERIFY_SCRIPT) --plan

verify:
	$(VERIFY_SCRIPT)

verify-check:
	$(VERIFY_SCRIPT) --self-test

check: check-go check-openapi check-proto check-sqlc

audit-full-manual:
	$(HEAVY_GUARD)
	$(MAKE) check
	$(MAKE) lint-deep
	$(MAKE) test-race
	$(MAKE) test-integration
	$(MAKE) template-init-check
	$(MAKE) integration-init-check
	$(MAKE) govulncheck
	$(MAKE) gosec
	$(MAKE) secret-scan-history
	$(MAKE) runtime-image-build RUNTIME_IMAGE=$(AUDIT_RUNTIME_IMAGE)
	$(MAKE) migration-validate RUNTIME_IMAGE=$(AUDIT_RUNTIME_IMAGE)
	$(MAKE) container-security CONTAINER_IMAGE=$(AUDIT_RUNTIME_IMAGE)

test test-package:
	$(REQUIRE_PKG)
	$(GO_TOOL) gotestsum --format=pkgname-and-test-fails -- -vet=off $(PKG)

test-all:
	$(GO_TOOL) gotestsum --format=pkgname-and-test-fails -- -vet=off ./...

test-watch:
	$(GO_TOOL) gotestsum --watch --format=pkgname-and-test-fails -- -vet=off

test-race:
	$(HEAVY_GUARD)
	go test -vet=off -race ./...

# profile:messaging-nats-jetstream:start
test-messaging-race:
	$(HEAVY_GUARD)
	go test -vet=off -p=1 -count=1 -race -tags=integration $(MESSAGING_RACE_PACKAGES) -run '^(TestOutboxWorkerPublishesStableWireIdentityAndTrace|TestNATSWorkerRegistrationIsSingleton|TestNATSNativeConsumeSurvivesBrokerRestart|TestWorkerUsesNativeBoundedConsumeContextsAndJoinsDrain|TestTypedPublisherAndHandlerHideBrokerFields|TestNATSPublishDispatchCancellationAndNoRetry|TestNATSWorkerComposition|TestNATSWorkerForcedShutdownDoesNotRaceHandlerCleanup|TestNATSConsumerSaturation|TestNATSForcedShutdownRedelivers|TestNATSGracefulDrain)$$'
# profile:messaging-nats-jetstream:end

# profile:outbox-postgres:start
test-outbox-race:
	$(HEAVY_GUARD)
	go test -vet=off -p=1 -count=1 -race -tags=integration $(OUTBOX_RACE_PACKAGES) -run '^TestPostgresOutbox'
# profile:outbox-postgres:end

# profile:webhooks-durable:start
test-webhook-race:
	$(HEAVY_GUARD)
	go test -vet=off -p=1 -count=1 -race -tags=integration $(WEBHOOK_RACE_PACKAGES) -run '^Test(PostgresWebhookAcceptance|WebhookNetwork)'
# profile:webhooks-durable:end

test-integration-race:
	$(HEAVY_GUARD)
	@if [ -z "$(strip $(INTEGRATION_RACE_PACKAGES))" ]; then \
		echo "not applicable: no focused integration race packages"; \
	else \
		go test -vet=off -p=1 -count=1 -race -tags=integration $(INTEGRATION_RACE_PACKAGES) -run '^(TestOutboxWorkerPublishesStableWireIdentityAndTrace|TestNATSWorkerRegistrationIsSingleton|TestNATSNativeConsumeSurvivesBrokerRestart|TestWorkerUsesNativeBoundedConsumeContextsAndJoinsDrain|TestTypedPublisherAndHandlerHideBrokerFields|TestNATSPublishDispatchCancellationAndNoRetry|TestNATSWorkerComposition|TestNATSWorkerForcedShutdownDoesNotRaceHandlerCleanup|TestNATSConsumerSaturation|TestNATSForcedShutdownRedelivers|TestNATSGracefulDrain|TestPostgresOutbox.*|TestPostgresWebhookAcceptance.*|TestWebhookNetwork.*)$$'; \
	fi

test-integration:
	$(HEAVY_GUARD)
	go test -p=1 -count=1 -tags=integration $(INTEGRATION_PACKAGES)

test-integration-db:
	$(HEAVY_GUARD)
	@if [ ! -d internal/infra/postgresidempotency ] && ! find test -maxdepth 1 -type f -name 'postgres*integration_test.go' -print -quit 2>/dev/null | grep -q .; then \
		echo "not applicable: no database integration surface"; \
	else \
		if [ -d internal/infra/postgresidempotency ]; then go test -vet=off -p=1 -count=1 -tags=integration ./internal/infra/postgresidempotency; fi; \
		go test -vet=off -p=1 -count=1 -tags=integration ./test -run '^Test(Postgres|InboundWebhook|Webhook)'; \
	fi

test-integration-messaging:
	$(HEAVY_GUARD)
	@if [ ! -d internal/infra/natsjs ]; then \
		echo "not applicable: no messaging integration surface"; \
	else \
		go test -vet=off -p=1 -count=1 -tags=integration ./internal/infra/natsjs ./cmd/worker/internal/bootstrap; \
		go test -vet=off -p=1 -count=1 -tags=integration ./test -run '^Test(NATS|PostgresOutbox)'; \
	fi

test-integration-process:
	$(HEAVY_GUARD)
	@if ! find test -maxdepth 1 -type f -name '*process_integration_test.go' -print -quit 2>/dev/null | grep -q .; then \
		echo "not applicable: no process integration surface"; \
	else \
		go test -vet=off -p=1 -count=1 -tags=integration ./test -run '^Test(GRPCProcessLifecycle|NATS.*Process|NATSWorkerMain|PostgresJobsWorkerProcess|InboundWebhookProcess)'; \
	fi

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

lint lint-package: lint-changed

lint-all:
	$(GOLANGCI_LINT) run --allow-serial-runners --concurrency=$(LINT_CONCURRENCY) --timeout=3m

lint-pr:
	$(GOLANGCI_LINT) run --allow-serial-runners --enable-only=$(LINT_PR_LINTERS) --concurrency=$(LINT_CONCURRENCY) --timeout=3m $(PKG)

lint-deep:
	$(HEAVY_GUARD)
	$(MAKE) deadcode
	$(MAKE) nilaway

lint-fast:
	$(REQUIRE_PKG)
	$(GOLANGCI_LINT) run --allow-serial-runners --fast-only --new-from-rev=$(LINT_BASE_REF) --concurrency=$(LINT_CONCURRENCY) --timeout=3m $(PKG)

deadcode:
	$(HEAVY_GUARD)
	$(GO_TOOL) deadcode -test -tags=integration ./...

nilaway:
	$(HEAVY_GUARD)
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

actionlint-fast:
	@test -z "$${CI:-}" || { echo "actionlint-fast is local-only; run make actionlint in CI"; exit 2; }
	@command -v actionlint >/dev/null 2>&1 || { echo "actionlint $(ACTIONLINT_VERSION) is required; run make actionlint instead"; exit 2; }
	@version="$$(actionlint -version | sed -n '1p')"; \
	test "$$version" = "$(ACTIONLINT_VERSION)" || { echo "actionlint $$version found, expected $(ACTIONLINT_VERSION); run make actionlint instead"; exit 2; }
	actionlint

shellcheck:
	@test -n "$(SHELL_FILES)" || { echo "no shell scripts found; skipping ShellCheck"; exit 0; }
	docker run --rm --read-only --network none \
		-v "$(CURDIR):/src:ro" \
		-w /src \
		"$(SHELLCHECK_IMAGE)" \
		-x \
		-- $(SHELL_FILES)

shellcheck-fast:
	@test -z "$${CI:-}" || { echo "shellcheck-fast is local-only; run make shellcheck in CI"; exit 2; }
	@command -v shellcheck >/dev/null 2>&1 || { echo "ShellCheck $(SHELLCHECK_VERSION) is required; run make shellcheck instead"; exit 2; }
	@version="$$(shellcheck --version | awk '$$1 == "version:" { print $$2; exit }')"; \
	test "$$version" = "$(SHELLCHECK_VERSION)" || { echo "ShellCheck $$version found, expected $(SHELLCHECK_VERSION); run make shellcheck instead"; exit 2; }
	shellcheck -x -- $(SHELL_FILES)

dockerfile-check:
	docker buildx build --check -f build/docker/Dockerfile .

govulncheck:
	$(HEAVY_GUARD)
	$(GO_TOOL) govulncheck ./...

gosec:
	$(HEAVY_GUARD)
	GOSECGOVERSION=go$(GO_REQUIRED_VERSION) $(GO_TOOL) gosec $(if $(strip $(GOMAXPROCS)),-concurrency=$(GOMAXPROCS)) -quiet -exclude-generated -exclude-dir=.agents -exclude-dir=.cache -exclude-dir=.artifacts ./...

secret-scan:
	@if [ "$(CI)" != "true" ]; then $(GITLEAKS) dir $(GITLEAKS_FLAGS) .; fi
	@git cat-file -e "$(SECRET_SCAN_BASE_REF)^{commit}" 2>/dev/null || { echo "secret scan base is unavailable: $(SECRET_SCAN_BASE_REF)" >&2; exit 2; }
	$(GITLEAKS) git $(GITLEAKS_FLAGS) --log-opts="$(SECRET_SCAN_BASE_REF)..HEAD" .

secret-scan-history:
	$(HEAVY_GUARD)
	$(GITLEAKS) git $(GITLEAKS_FLAGS) --log-opts=--all .

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
	else \
		echo "not applicable: no SQLC query sources"; \
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

openapi-runtime-contract-check:
	go test -vet=off ./internal/infra/http $(OPENAPI_PACKAGES)
	go test -vet=off ./cmd/service/internal/bootstrap -run '^TestInboundWebhookHeaderOverflowUsesListener431$$'

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

openapi-check: openapi-drift-check openapi-runtime-contract-check openapi-lint openapi-validate

# profile:grpc:start
proto-format:
	@if find api/proto -type f -name '*.proto' -print -quit 2>/dev/null | grep -q .; then $(GO_TOOL) buf format -w; fi
	@if [ -f examples/grpc-reference-service/buf.yaml ]; then cd examples/grpc-reference-service && go tool -modfile=../../tools/go.mod buf format -w; fi

proto-format-check:
	@if find api/proto -type f -name '*.proto' -print -quit 2>/dev/null | grep -q .; then $(GO_TOOL) buf format --diff --exit-code; fi
	@if [ -f examples/grpc-reference-service/buf.yaml ]; then cd examples/grpc-reference-service && go tool -modfile=../../tools/go.mod buf format --diff --exit-code; fi

proto-lint:
	@if find api/proto -type f -name '*.proto' -print -quit 2>/dev/null | grep -q .; then $(GO_TOOL) buf lint; fi
	@if [ -f examples/grpc-reference-service/buf.yaml ]; then cd examples/grpc-reference-service && go tool -modfile=../../tools/go.mod buf lint; fi

proto-generate:
	@if find api/proto -type f -name '*.proto' -print -quit 2>/dev/null | grep -q .; then $(GO_TOOL) buf generate; fi
	@if [ -f examples/grpc-reference-service/buf.yaml ]; then cd examples/grpc-reference-service && go tool -modfile=../../tools/go.mod buf generate; fi

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
check-proto: proto-check
# profile:grpc:end

# profile:database-postgres:start
migration-validate:
	$(HEAVY_GUARD)
	GO="$(GO)" $(MIGRATION_VALIDATE_SCRIPT) "$(RUNTIME_IMAGE)" "$(RUNTIME_EXPECTED_VERSION)" "$(SERVICE_NAME)"
# profile:database-postgres:end

container-security:
	$(HEAVY_GUARD)
	@image="$(CONTAINER_IMAGE)"; \
	if [ -z "$$image" ]; then image="$(SERVICE_NAME):ci"; docker build -f build/docker/Dockerfile -t "$$image" .; fi; \
	docker run --rm \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v "$(TRIVY_CACHE_VOLUME):/root/.cache/trivy" \
		-e DOCKER_HOST=unix:///var/run/docker.sock \
		-e TRIVY_DB_REPOSITORY \
		"$(TRIVY_IMAGE)" image \
		--cache-dir /root/.cache/trivy \
		--quiet \
		--severity HIGH,CRITICAL \
		--scanners vuln \
		--ignore-unfixed \
		--exit-code 1 \
		--format table \
		"$$image"

runtime-image-build:
	bash ./scripts/ci/runtime-image-build.sh "$(RUNTIME_IMAGE)"

runtime-image-check:
	$(HEAVY_GUARD)
	$(RUNTIME_IMAGE_CHECK_SCRIPT) "$(if $(RUNTIME_IMAGE),$(RUNTIME_IMAGE),$(VERIFY_RUNTIME_IMAGE))" "$(RUNTIME_EXPECTED_VERSION)"

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
