SERVICE_NAME := service
BINARY := bin/$(SERVICE_NAME)
OPENAPI_FILE := api/openapi/service.yaml
OPENAPI_GENERATED_FILES := internal/api/openapi.gen.go
GO_FILES := $(shell find . -type f -name '*.go' -not -path './vendor/*' -not -path './.cache/*')
GOFUMPT_FILES := $(shell find . -type f -name '*.go' -not -path './vendor/*' -not -path './.cache/*' -not -path './internal/api/openapi.gen.go' -not -path './internal/infra/postgres/sqlcgen/*' -not -name '*_mock_test.go' -not -name '*_string.go')
REDOCLY_CLI_VERSION := 2.20.3
GO_REQUIRED_VERSION := $(shell awk '/^go / {print $$2; exit}' go.mod)
TEST_REPORT_DIR := .artifacts/test
TEST_JUNIT_FILE := $(TEST_REPORT_DIR)/junit.xml
TEST_JSON_FILE := $(TEST_REPORT_DIR)/test2json.json
COVERAGE_MIN ?= 65.0
COVERAGE_GOTOOLCHAIN ?= go$(GO_REQUIRED_VERSION)
COVERAGE_EXCLUDE_REGEX ?= (^|/)internal/api/openapi\.gen\.go:|(^|/)cmd/service/main\.go:
FUZZ_TIME ?= 45s
DOCS_DRIFT_SCRIPT := bash ./scripts/ci/docs-drift-check.sh
GUARDRAILS_CHECK_SCRIPT := bash ./scripts/ci/required-guardrails-check.sh
BRANCH_PROTECTION_SCRIPT := bash ./scripts/dev/configure-branch-protection.sh
DOCKER_TOOLING_SCRIPT := bash ./scripts/dev/docker-tooling.sh
SKILLS_SYNC_SCRIPT := bash ./scripts/dev/sync-skills.sh
AGENTS_SYNC_SCRIPT := bash ./scripts/dev/sync-agents.sh

.DEFAULT_GOAL := help

.PHONY: help bootstrap bootstrap-native bootstrap-docker check docker-check check-full \
	template-init template-init-strict template-init-native template-init-native-strict template-init-docker \
	setup setup-strict setup-native setup-native-strict setup-docker doctor init-module tidy fmt vet test test-race test-cover test-cover-local test-report coverage-check test-fuzz-smoke test-integration lint go-security secret-scan secrets-scan ci-local run build docker-build docker-run compose-up compose-down vendor \
	openapi-generate openapi-drift-check openapi-runtime-contract-check openapi-lint openapi-validate openapi-breaking openapi-check \
	mod-check fmt-check docs-drift-check guardrails-check migration-validate gh-protect skills-sync skills-check agents-sync agents-check \
	doctor-native doctor-docker docker-pull-tools docker-init-module docker-mod-check docker-fmt docker-fmt-check \
	docker-test docker-vet docker-test-race docker-test-cover docker-test-report docker-test-integration docker-lint docker-openapi-check docker-sqlc-check docker-go-security docker-secret-scan docker-secrets-scan docker-ci \
	docker-guardrails-check docker-skills-check docker-agents-check docker-docs-drift-check docker-migration-validate docker-container-security \
	mocks-generate mocks-drift-check stringer-generate stringer-drift-check sqlc-generate sqlc-check

help:
	@echo "Quick onboarding commands:"
	@echo "  make bootstrap      # prepare local environment (.env + dependencies)"
	@echo "  make check          # quick checks (fmt/lint/test)"
	@echo "  make docker-check   # quick checks through pinned Docker tooling"
	@echo "  make check-full     # full local baseline (prefers docker-ci)"
	@echo "  make run            # run service locally"
	@echo ""
	@echo "Template/admin commands:"
	@echo "  make template-init          # module/CODEOWNERS/skills initialization"
	@echo "  make template-init-native   # force native template initialization"
	@echo "  make template-init-docker   # force docker template initialization"
	@echo ""
	@echo "Advanced validation commands:"
	@echo "  make ci-local       # native CI-like checks"
	@echo "  make docker-ci      # closest zero-setup CI parity"
	@echo "  make openapi-check  # OpenAPI generation, lint, validation, and runtime contract"
	@echo "  make sqlc-check     # SQLC generation and drift checks"
	@echo "  make test-integration        # integration tests"
	@echo "  make test-report             # coverage report and threshold"
	@echo "  make migration-validate      # migration rehearsal"
	@echo "  make go-security             # govulncheck + gosec"
	@echo "  make secret-scan             # gitleaks secret scan"
	@echo "  make mocks-drift-check       # mockgen drift checks"
	@echo "  make stringer-drift-check    # stringer drift checks"
	@echo "  make agents-check            # Codex/Claude agent mirror drift check"
	@echo "  make skills-check            # skill mirror drift check"
	@echo "  make docker-openapi-check    # Docker OpenAPI validation"
	@echo "  make docker-sqlc-check       # Docker SQLC validation"
	@echo "  make docker-test-integration # Docker integration tests"
	@echo "  make docker-migration-validate  # Docker migration rehearsal"
	@echo "  make docker-go-security      # Docker govulncheck + gosec"
	@echo "  make docker-secret-scan      # Docker gitleaks secret scan"
	@echo "  make docker-container-security  # Docker image scan"
	@echo "  make gh-protect BRANCH=main"
	@echo ""
	@echo "Reference: docs/build-test-and-development-commands.md"

bootstrap:
	@if [ ! -f .env ]; then \
		cp env/.env.example .env; \
		echo "Created .env from env/.env.example"; \
	fi
	@if command -v go >/dev/null 2>&1; then \
		echo "go toolchain detected: downloading Go modules"; \
		go mod download; \
	elif command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then \
		echo "go toolchain missing: preparing Docker tooling images"; \
		$(MAKE) docker-pull-tools; \
	else \
		echo "bootstrap requires either a local Go toolchain or Docker daemon"; \
		exit 1; \
	fi
	@echo "Bootstrap complete. Next steps: make check && make run"

bootstrap-native: bootstrap

bootstrap-docker: bootstrap

check:
	@if command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then \
		if command -v go >/dev/null 2>&1; then \
			echo "go toolchain detected: running quick local checks"; \
			$(MAKE) fmt-check lint test; \
		else \
			echo "go toolchain missing: running quick checks in docker mode"; \
			$(MAKE) docker-fmt-check docker-lint docker-test; \
		fi; \
	elif command -v go >/dev/null 2>&1; then \
		echo "go toolchain detected: running quick local checks"; \
		$(MAKE) fmt-check lint test; \
	else \
		echo "quick checks require either local Go toolchain or Docker daemon"; \
		exit 1; \
	fi

docker-check: docker-fmt-check docker-lint docker-test

check-full:
	@if command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then \
		echo "docker daemon detected: running zero-setup CI checks"; \
		$(MAKE) docker-ci BASE_REF="$(BASE_REF)" HEAD_REF="$(HEAD_REF)"; \
	else \
		echo "docker daemon unavailable: running native partial CI-like checks"; \
		echo "Docker-only integration, migration, and container checks may be skipped; start Docker and run 'make docker-ci' for closest parity"; \
		$(MAKE) ci-local BASE_REF="$(BASE_REF)" HEAD_REF="$(HEAD_REF)"; \
	fi

template-init: setup

template-init-strict: setup-strict

template-init-native: setup-native

template-init-native-strict: setup-native-strict

template-init-docker: setup-docker

setup:
	bash ./scripts/dev/setup.sh

setup-strict:
	bash ./scripts/dev/setup.sh --strict

setup-native:
	bash ./scripts/dev/setup.sh --native

setup-native-strict:
	bash ./scripts/dev/setup.sh --native --strict

setup-docker:
	bash ./scripts/dev/setup.sh --docker

doctor:
	bash ./scripts/dev/doctor.sh --mode auto

doctor-native:
	bash ./scripts/dev/doctor.sh --mode native

doctor-docker:
	bash ./scripts/dev/doctor.sh --mode docker

init-module:
	@if [ -n "$(MODULE)" ]; then \
		bash ./scripts/init-module.sh "$(MODULE)"; \
	else \
		bash ./scripts/init-module.sh; \
	fi

docker-pull-tools:
	$(DOCKER_TOOLING_SCRIPT) pull-images

docker-init-module:
	@if [ -n "$(MODULE)" ]; then \
		CODEOWNER="$(CODEOWNER)" $(DOCKER_TOOLING_SCRIPT) init-module "$(MODULE)"; \
	else \
		CODEOWNER="$(CODEOWNER)" $(DOCKER_TOOLING_SCRIPT) init-module; \
	fi

tidy:
	go mod tidy

fmt:
	go tool goimports -w $(GO_FILES)
	go tool gofumpt -w $(GOFUMPT_FILES)

mod-check:
	GOFLAGS= go mod tidy -diff
	go mod verify
	git diff --exit-code -- go.mod go.sum

docker-mod-check:
	$(DOCKER_TOOLING_SCRIPT) mod-check

fmt-check:
	@unformatted="$$(go tool goimports -l $(GO_FILES))"; \
	if [ -n "$$unformatted" ]; then \
		echo "goimports required for:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi
	@gofumpt_unformatted="$$(go tool gofumpt -l $(GOFUMPT_FILES))"; \
	if [ -n "$$gofumpt_unformatted" ]; then \
		echo "gofumpt required for:"; \
		echo "$$gofumpt_unformatted"; \
		exit 1; \
	fi

docker-fmt:
	$(DOCKER_TOOLING_SCRIPT) fmt

docker-fmt-check:
	$(DOCKER_TOOLING_SCRIPT) fmt-check

test:
	go test ./...

vet:
	go vet ./...

test-race:
	go test -race ./...

test-cover:
	GOTOOLCHAIN=$(COVERAGE_GOTOOLCHAIN) GOCOVERDIR= go test -covermode=atomic -coverprofile=coverage.out ./...
	GOTOOLCHAIN=$(COVERAGE_GOTOOLCHAIN) go tool cover -func=coverage.out

test-report:
	@mkdir -p $(TEST_REPORT_DIR)
	GOTOOLCHAIN=$(COVERAGE_GOTOOLCHAIN) go tool gotestsum --format=standard-verbose --junitfile=$(TEST_JUNIT_FILE) --jsonfile=$(TEST_JSON_FILE) -- -covermode=atomic -coverprofile=coverage.out ./...
	GOTOOLCHAIN=$(COVERAGE_GOTOOLCHAIN) go tool cover -func=coverage.out
	$(MAKE) coverage-check COVERAGE_MIN=$(COVERAGE_MIN)

coverage-check:
	@test -f coverage.out || (echo "coverage.out not found; run 'make test-cover' or 'make test-report'"; exit 1)
	@filtered_cov="$$(mktemp)"; \
	grep -Ev '$(COVERAGE_EXCLUDE_REGEX)' coverage.out > "$$filtered_cov"; \
	total="$$(GOTOOLCHAIN=$(COVERAGE_GOTOOLCHAIN) go tool cover -func="$$filtered_cov" | awk '/^total:/ {gsub(/%/, "", $$3); print $$3}')"; \
	rm -f "$$filtered_cov"; \
	if [ -z "$$total" ]; then \
		echo "failed to parse total coverage from coverage.out"; \
		exit 1; \
	fi; \
	awk -v total="$$total" -v minimum="$(COVERAGE_MIN)" 'BEGIN { \
		if ((total + 0) < (minimum + 0)) { \
			printf "coverage %.2f%% is below threshold %.2f%%\n", total, minimum; \
			exit 1; \
		} \
		printf "coverage %.2f%% meets threshold %.2f%%\n", total, minimum; \
	}'

test-fuzz-smoke:
	@found=0; \
	for pkg in $$(go list ./...); do \
		if go test "$$pkg" -list '^Fuzz' | grep -q '^Fuzz'; then \
			found=1; \
			echo "running fuzz smoke for $$pkg"; \
			go test "$$pkg" -run '^$$' -fuzz=Fuzz -fuzztime=$(FUZZ_TIME); \
		fi; \
	done; \
	if [ "$$found" -eq 0 ]; then \
		echo "no fuzz targets found; skipping fuzz smoke run"; \
	fi

test-cover-local:
	@coverage_log="$$(mktemp)"; \
	if GOTOOLCHAIN=$(COVERAGE_GOTOOLCHAIN) GOCOVERDIR= go test -covermode=atomic -coverprofile=coverage.out ./... >"$$coverage_log" 2>&1; then \
		cat "$$coverage_log"; \
		GOTOOLCHAIN=$(COVERAGE_GOTOOLCHAIN) go tool cover -func=coverage.out; \
	else \
		cat "$$coverage_log"; \
		if grep -Eq 'does not match go tool version' "$$coverage_log"; then \
			echo "coverage check skipped: local coverage tooling is unhealthy"; \
			echo "run 'make doctor-native' for diagnostics or use 'make docker-test-cover'"; \
		else \
			rm -f "$$coverage_log"; \
			exit 1; \
		fi; \
	fi; \
	rm -f "$$coverage_log"

docker-test:
	$(DOCKER_TOOLING_SCRIPT) test

docker-vet:
	$(DOCKER_TOOLING_SCRIPT) vet

docker-test-race:
	$(DOCKER_TOOLING_SCRIPT) test-race

docker-test-cover:
	$(DOCKER_TOOLING_SCRIPT) test-cover

docker-test-report:
	$(DOCKER_TOOLING_SCRIPT) test-report

test-integration:
	go test -tags=integration ./test/...

docker-test-integration:
	$(DOCKER_TOOLING_SCRIPT) test-integration

lint:
	go tool golangci-lint config verify
	go tool golangci-lint run --timeout=3m

go-security:
	go tool govulncheck ./...
	go tool gosec -exclude-generated -exclude-dir=.cache ./...

secret-scan:
	go tool gitleaks git --no-banner --redact --exit-code 1 .

secrets-scan: secret-scan

ci-local:
	$(MAKE) mod-check guardrails-check agents-check skills-check fmt-check lint test vet test-race test-report mocks-drift-check stringer-drift-check sqlc-check openapi-check go-security secret-scan
	@if [ -n "$(BASE_REF)" ] && [ -n "$(HEAD_REF)" ]; then \
		$(MAKE) docs-drift-check BASE_REF="$(BASE_REF)" HEAD_REF="$(HEAD_REF)"; \
	else \
		echo "BASE_REF/HEAD_REF are not set, skipping docs drift check in ci-local"; \
	fi
	@if command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then \
		echo "docker daemon detected: running integration, migration rehearsal, and container scan"; \
		REQUIRE_DOCKER=1 $(MAKE) test-integration; \
		$(MAKE) docker-migration-validate; \
		$(MAKE) docker-container-security; \
	else \
		echo "docker daemon is unavailable: skipping integration, migration rehearsal, and container scan"; \
		echo "start Docker and run 'make docker-ci' for full CI parity"; \
	fi

docker-lint:
	$(DOCKER_TOOLING_SCRIPT) lint

stringer-generate:
	go generate -run "stringer" ./...

stringer-drift-check: stringer-generate
	@git diff --quiet -- ':(glob)**/*_string.go' || (echo "tracked stringer drift detected in *_string.go files"; git diff -- ':(glob)**/*_string.go'; exit 1)
	@untracked="$$(git ls-files --others --exclude-standard -- ':(glob)**/*_string.go')"; \
	if [ -n "$$untracked" ]; then \
		echo "untracked stringer artifacts detected"; \
		echo "$$untracked"; \
		echo "run 'make stringer-generate' and commit updated enum string files"; \
		exit 1; \
	fi

sqlc-generate:
	go tool sqlc generate -f internal/infra/postgres/sqlc.yaml

sqlc-check: sqlc-generate
	@git diff --quiet -- internal/infra/postgres/sqlcgen || (echo "tracked sqlc drift detected in internal/infra/postgres/sqlcgen"; git diff -- internal/infra/postgres/sqlcgen; exit 1)
	@untracked="$$(git ls-files --others --exclude-standard -- internal/infra/postgres/sqlcgen)"; \
	if [ -n "$$untracked" ]; then \
		echo "untracked sqlc artifacts detected in internal/infra/postgres/sqlcgen"; \
		echo "$$untracked"; \
		echo "run 'make sqlc-generate' and commit updated sqlc generated files"; \
		exit 1; \
	fi
	@expected="$$(for f in internal/infra/postgres/queries/*.sql; do [ -e "$$f" ] || continue; basename "$$f" .sql; done | sort)"; \
	actual="$$(for f in internal/infra/postgres/sqlcgen/*.sql.go; do [ -e "$$f" ] || continue; basename "$$f" .sql.go; done | sort)"; \
	if [ "$$expected" != "$$actual" ]; then \
		echo "sqlc query/source mismatch detected"; \
		echo "expected generated query stems:"; \
		printf '%s\n' "$$expected"; \
		echo "actual generated query stems:"; \
		printf '%s\n' "$$actual"; \
		echo "remove stale generated files and run 'make sqlc-generate'"; \
		exit 1; \
	fi

mocks-generate:
	go generate -run "mockgen" ./...

mocks-drift-check: mocks-generate
	@git diff --quiet -- ':(glob)**/*_mock_test.go' || (echo "tracked mockgen drift detected in *_mock_test.go files"; git diff -- ':(glob)**/*_mock_test.go'; exit 1)
	@untracked="$$(git ls-files --others --exclude-standard -- ':(glob)**/*_mock_test.go')"; \
	if [ -n "$$untracked" ]; then \
		echo "untracked mockgen artifacts detected"; \
		echo "$$untracked"; \
		echo "run 'make mocks-generate' and commit updated mock files"; \
		exit 1; \
	fi

openapi-generate:
	go generate ./internal/api

openapi-drift-check:
	@git diff --quiet -- $(OPENAPI_GENERATED_FILES) || (echo "tracked openapi codegen drift detected in $(OPENAPI_GENERATED_FILES)"; git diff -- $(OPENAPI_GENERATED_FILES); exit 1)
	@untracked="$$(git ls-files --others --exclude-standard -- $(OPENAPI_GENERATED_FILES))"; \
	if [ -n "$$untracked" ]; then \
		echo "untracked openapi artifacts detected in $(OPENAPI_GENERATED_FILES)"; \
		echo "$$untracked"; \
		echo "run 'make openapi-generate' and commit updated generated files"; \
		exit 1; \
	fi

openapi-runtime-contract-check:
	go test ./internal/infra/http -run '^TestOpenAPIRuntimeContract' -count=1

openapi-lint:
	npx @redocly/cli@$(REDOCLY_CLI_VERSION) lint --config .redocly.yaml $(OPENAPI_FILE)

openapi-validate:
	go tool validate -- $(OPENAPI_FILE)

openapi-breaking:
	@test -n "$(BASE_OPENAPI)" || (echo "BASE_OPENAPI is required"; exit 1)
	go tool oasdiff breaking --fail-on ERR $(BASE_OPENAPI) $(OPENAPI_FILE)

openapi-check: openapi-generate openapi-drift-check
	go test ./internal/api
	$(MAKE) openapi-runtime-contract-check openapi-lint openapi-validate

docker-openapi-check:
	$(DOCKER_TOOLING_SCRIPT) openapi-check

docker-sqlc-check:
	$(DOCKER_TOOLING_SCRIPT) sqlc-check

docs-drift-check:
	@test -n "$(BASE_REF)" || (echo "BASE_REF is required"; exit 1)
	@test -n "$(HEAD_REF)" || (echo "HEAD_REF is required"; exit 1)
	$(DOCS_DRIFT_SCRIPT) "$(BASE_REF)" "$(HEAD_REF)"

guardrails-check:
	$(GUARDRAILS_CHECK_SCRIPT)

skills-sync:
	$(SKILLS_SYNC_SCRIPT)

skills-check:
	$(SKILLS_SYNC_SCRIPT) --check

agents-sync:
	$(AGENTS_SYNC_SCRIPT)

agents-check:
	$(AGENTS_SYNC_SCRIPT) --check

docker-go-security:
	$(DOCKER_TOOLING_SCRIPT) go-security

docker-secret-scan:
	$(DOCKER_TOOLING_SCRIPT) secret-scan

docker-secrets-scan: docker-secret-scan

docker-guardrails-check:
	$(DOCKER_TOOLING_SCRIPT) guardrails-check

docker-skills-check:
	$(DOCKER_TOOLING_SCRIPT) skills-check

docker-agents-check:
	$(DOCKER_TOOLING_SCRIPT) agents-check

docker-docs-drift-check:
	@test -n "$(BASE_REF)" || (echo "BASE_REF is required"; exit 1)
	@test -n "$(HEAD_REF)" || (echo "HEAD_REF is required"; exit 1)
	$(DOCKER_TOOLING_SCRIPT) docs-drift-check "$(BASE_REF)" "$(HEAD_REF)"

docker-migration-validate:
	$(DOCKER_TOOLING_SCRIPT) migration-validate

docker-container-security:
	$(DOCKER_TOOLING_SCRIPT) container-security

docker-ci:
	BASE_REF="$(BASE_REF)" HEAD_REF="$(HEAD_REF)" $(DOCKER_TOOLING_SCRIPT) ci

migration-validate:
	@if [ -n "$(MIGRATION_DSN)" ]; then \
		go tool migrate -path env/migrations -database "$(MIGRATION_DSN)" up; \
		go tool migrate -path env/migrations -database "$(MIGRATION_DSN)" down 1; \
		go tool migrate -path env/migrations -database "$(MIGRATION_DSN)" up 1; \
	elif command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then \
		echo "MIGRATION_DSN is empty: running docker-migration-validate"; \
		$(MAKE) docker-migration-validate; \
	else \
		echo "MIGRATION_DSN is empty and docker daemon is unavailable: skipping migration validation"; \
		echo "set MIGRATION_DSN or start Docker to run migration rehearsal"; \
	fi

gh-protect:
	$(BRANCH_PROTECTION_SCRIPT) "$${BRANCH:-main}"

run:
	@set -a; \
	if [ -f .env ]; then . ./.env; fi; \
	set +a; \
	go run ./cmd/$(SERVICE_NAME)

build:
	mkdir -p bin
	CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o $(BINARY) ./cmd/$(SERVICE_NAME)

docker-build:
	docker build -f build/docker/Dockerfile -t $(SERVICE_NAME):local .

docker-run:
	docker run --rm -p 8080:8080 --env-file .env $(SERVICE_NAME):local

compose-up:
	docker compose -f env/docker-compose.yml up -d

compose-down:
	docker compose -f env/docker-compose.yml down -v

vendor:
	go mod vendor
