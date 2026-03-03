SERVICE_NAME := service
BINARY := bin/$(SERVICE_NAME)
OPENAPI_FILE := api/openapi/service.yaml
REDOCLY_CLI_VERSION := 2.20.3
KIN_OPENAPI_VALIDATE_VERSION := v0.133.0
OASDIFF_VERSION := v1.11.10
MIGRATE_VERSION := v4.19.1
GOLANGCI_LINT_VERSION := v2.10.1
GOVULNCHECK_VERSION := v1.1.4
GOSEC_VERSION := v2.24.7
GOIMPORTS_VERSION := v0.42.0
GITLEAKS_VERSION := v8.30.0
GO_REQUIRED_VERSION := $(shell awk '/^go / {print $$2; exit}' go.mod)
TEST_REPORT_DIR := .artifacts/test
TEST_JUNIT_FILE := $(TEST_REPORT_DIR)/junit.xml
TEST_JSON_FILE := $(TEST_REPORT_DIR)/test2json.json
COVERAGE_MIN ?= 70.0
COVERAGE_GOTOOLCHAIN ?= go$(GO_REQUIRED_VERSION)
COVERAGE_EXCLUDE_REGEX ?= (^|/)internal/api/openapi\.gen\.go:|(^|/)cmd/service/main\.go:
FUZZ_TIME ?= 45s
DOCS_DRIFT_SCRIPT := bash ./scripts/ci/docs-drift-check.sh
GUARDRAILS_CHECK_SCRIPT := bash ./scripts/ci/required-guardrails-check.sh
BRANCH_PROTECTION_SCRIPT := bash ./scripts/dev/configure-branch-protection.sh
DOCKER_TOOLING_SCRIPT := bash ./scripts/dev/docker-tooling.sh
SKILLS_SYNC_SCRIPT := bash ./scripts/dev/sync-skills.sh

.DEFAULT_GOAL := help

.PHONY: help bootstrap bootstrap-native bootstrap-docker check check-full \
	template-init template-init-strict template-init-native template-init-native-strict template-init-docker \
	setup setup-strict setup-native setup-native-strict setup-docker doctor init-module tidy fmt test test-race test-cover test-cover-local test-report coverage-check test-fuzz-smoke test-integration lint go-security secrets-scan ci-local run build docker-build docker-run compose-up compose-down vendor \
	openapi-generate openapi-drift-check openapi-runtime-contract-check openapi-lint openapi-validate openapi-breaking openapi-check \
	mod-check fmt-check docs-drift-check guardrails-check migration-validate gh-protect skills-sync skills-check \
	doctor-native doctor-docker docker-pull-tools docker-init-module docker-mod-check docker-fmt docker-fmt-check \
	docker-test docker-test-race docker-test-cover docker-test-integration docker-lint docker-openapi-check docker-go-security docker-secrets-scan docker-ci \
	docker-guardrails-check docker-skills-check docker-docs-drift-check docker-migration-validate docker-container-security

help:
	@echo "Quick onboarding commands:"
	@echo "  make bootstrap      # prepare local environment (.env + dependencies)"
	@echo "  make check          # quick checks (fmt/lint/test)"
	@echo "  make check-full     # full CI-like checks"
	@echo "  make run            # run service locally"
	@echo ""
	@echo "Template/admin commands:"
	@echo "  make template-init          # module/CODEOWNERS/skills initialization"
	@echo "  make template-init-native   # force native template initialization"
	@echo "  make template-init-docker   # force docker template initialization"
	@echo ""
	@echo "Advanced validation commands:"
	@echo "  make ci-local       # native CI-like checks"
	@echo "  make docker-ci      # Docker CI-like checks"
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

check-full:
	@if command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then \
		echo "docker daemon detected: running zero-setup CI checks"; \
		$(MAKE) docker-ci; \
	else \
		echo "docker daemon unavailable: running native CI checks"; \
		$(MAKE) ci-local; \
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
	go run golang.org/x/tools/cmd/goimports@$(GOIMPORTS_VERSION) -w $(shell find . -type f -name '*.go' -not -path './vendor/*')

mod-check:
	GOFLAGS= go mod tidy -diff
	go mod verify
	git diff --exit-code -- go.mod go.sum

docker-mod-check:
	$(DOCKER_TOOLING_SCRIPT) mod-check

fmt-check:
	@unformatted="$$(go run golang.org/x/tools/cmd/goimports@$(GOIMPORTS_VERSION) -l $$(find . -type f -name '*.go' -not -path './vendor/*'))"; \
	if [ -n "$$unformatted" ]; then \
		echo "goimports required for:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

docker-fmt:
	$(DOCKER_TOOLING_SCRIPT) fmt

docker-fmt-check:
	$(DOCKER_TOOLING_SCRIPT) fmt-check

test:
	go test ./...

test-race:
	go test -race ./...

test-cover:
	GOTOOLCHAIN=$(COVERAGE_GOTOOLCHAIN) GOCOVERDIR= go test -covermode=atomic -coverprofile=coverage.out ./...
	GOTOOLCHAIN=$(COVERAGE_GOTOOLCHAIN) go tool cover -func=coverage.out

test-report:
	@mkdir -p $(TEST_REPORT_DIR)
	GOTOOLCHAIN=$(COVERAGE_GOTOOLCHAIN) go tool gotestsum --format=standard-verbose --junitfile=$(TEST_JUNIT_FILE) --jsonfile=$(TEST_JSON_FILE) -- -race -covermode=atomic -coverprofile=coverage.out ./...
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

docker-test-race:
	$(DOCKER_TOOLING_SCRIPT) test-race

docker-test-cover:
	$(DOCKER_TOOLING_SCRIPT) test-cover

test-integration:
	go test -tags=integration ./test/...

docker-test-integration:
	$(DOCKER_TOOLING_SCRIPT) test-integration

lint:
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION) run --timeout=3m

go-security:
	go run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) ./...
	go run github.com/securego/gosec/v2/cmd/gosec@$(GOSEC_VERSION) -exclude-generated ./...

secrets-scan:
	go run github.com/zricethezav/gitleaks/v8@$(GITLEAKS_VERSION) git --no-banner --redact --exit-code 1 .

ci-local:
	$(MAKE) mod-check guardrails-check skills-check fmt-check lint test test-race test-cover-local openapi-check go-security secrets-scan
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

openapi-generate:
	go generate ./internal/api

openapi-drift-check:
	@git diff --quiet -- internal/api || (echo "tracked openapi codegen drift detected in internal/api"; git diff -- internal/api; exit 1)
	@untracked="$$(git ls-files --others --exclude-standard -- internal/api)"; \
	if [ -n "$$untracked" ]; then \
		echo "untracked openapi artifacts detected in internal/api"; \
		echo "$$untracked"; \
		echo "run 'make openapi-generate' and commit updated generated files"; \
		exit 1; \
	fi

openapi-runtime-contract-check:
	go test ./internal/infra/http -run '^TestOpenAPIRuntimeContract' -count=1

openapi-lint:
	npx @redocly/cli@$(REDOCLY_CLI_VERSION) lint --config .redocly.yaml $(OPENAPI_FILE)

openapi-validate:
	go run github.com/getkin/kin-openapi/cmd/validate@$(KIN_OPENAPI_VALIDATE_VERSION) -- $(OPENAPI_FILE)

openapi-breaking:
	@test -n "$(BASE_OPENAPI)" || (echo "BASE_OPENAPI is required"; exit 1)
	go run github.com/oasdiff/oasdiff@$(OASDIFF_VERSION) breaking --fail-on ERR $(BASE_OPENAPI) $(OPENAPI_FILE)

openapi-check: openapi-generate openapi-drift-check
	go test ./internal/api
	$(MAKE) openapi-runtime-contract-check openapi-lint openapi-validate

docker-openapi-check:
	$(DOCKER_TOOLING_SCRIPT) openapi-check

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

docker-go-security:
	$(DOCKER_TOOLING_SCRIPT) go-security

docker-secrets-scan:
	$(DOCKER_TOOLING_SCRIPT) secrets-scan

docker-guardrails-check:
	$(DOCKER_TOOLING_SCRIPT) guardrails-check

docker-skills-check:
	$(DOCKER_TOOLING_SCRIPT) skills-check

docker-docs-drift-check:
	@test -n "$(BASE_REF)" || (echo "BASE_REF is required"; exit 1)
	@test -n "$(HEAD_REF)" || (echo "HEAD_REF is required"; exit 1)
	$(DOCKER_TOOLING_SCRIPT) docs-drift-check "$(BASE_REF)" "$(HEAD_REF)"

docker-migration-validate:
	$(DOCKER_TOOLING_SCRIPT) migration-validate

docker-container-security:
	$(DOCKER_TOOLING_SCRIPT) container-security

docker-ci:
	$(DOCKER_TOOLING_SCRIPT) ci

migration-validate:
	@if [ -n "$(MIGRATION_DSN)" ]; then \
		go run github.com/golang-migrate/migrate/v4/cmd/migrate@$(MIGRATE_VERSION) -path env/migrations -database "$(MIGRATION_DSN)" up; \
		go run github.com/golang-migrate/migrate/v4/cmd/migrate@$(MIGRATE_VERSION) -path env/migrations -database "$(MIGRATION_DSN)" down 1; \
		go run github.com/golang-migrate/migrate/v4/cmd/migrate@$(MIGRATE_VERSION) -path env/migrations -database "$(MIGRATION_DSN)" up 1; \
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
