SERVICE_NAME := service
BINARY := bin/$(SERVICE_NAME)
OPENAPI_FILE := api/openapi/service.yaml
REDOCLY_CLI_VERSION := 2.20.0
KIN_OPENAPI_VALIDATE_VERSION := v0.133.0
OASDIFF_VERSION := v1.11.10
MIGRATE_VERSION := v4.19.0
GOLANGCI_LINT_VERSION := v2.10.1
GOVULNCHECK_VERSION := v1.1.4
GOSEC_VERSION := v2.24.6
GOIMPORTS_VERSION := v0.32.0
GITLEAKS_VERSION := v8.30.0
DOCS_DRIFT_SCRIPT := bash ./scripts/ci/docs-drift-check.sh
GUARDRAILS_CHECK_SCRIPT := bash ./scripts/ci/required-guardrails-check.sh
BRANCH_PROTECTION_SCRIPT := bash ./scripts/dev/configure-branch-protection.sh
DOCKER_TOOLING_SCRIPT := bash ./scripts/dev/docker-tooling.sh
SKILLS_SYNC_SCRIPT := bash ./scripts/dev/sync-skills.sh

.DEFAULT_GOAL := help

.PHONY: help bootstrap bootstrap-native bootstrap-docker check \
	setup setup-strict setup-native setup-native-strict setup-docker doctor init-module tidy fmt test test-race test-cover test-cover-local test-integration lint go-security secrets-scan ci-local run build docker-build docker-run compose-up compose-down vendor \
	openapi-generate openapi-drift-check openapi-runtime-contract-check openapi-lint openapi-validate openapi-breaking openapi-check \
	mod-check fmt-check docs-drift-check guardrails-check migration-validate gh-protect skills-sync skills-check \
	doctor-native doctor-docker docker-pull-tools docker-init-module docker-mod-check docker-fmt docker-fmt-check \
	docker-test docker-test-race docker-test-cover docker-test-integration docker-lint docker-openapi-check docker-go-security docker-secrets-scan docker-ci \
	docker-guardrails-check docker-skills-check docker-docs-drift-check docker-migration-validate docker-container-security

help:
	@echo "Quick onboarding commands:"
	@echo "  make bootstrap      # prepare environment (auto mode, prefers Docker)"
	@echo "  make check          # run full checks (Docker if available, else native)"
	@echo "  make run            # run service locally"
	@echo ""
	@echo "Most used commands:"
	@echo "  make setup-native   # force native setup (Go + Node on host)"
	@echo "  make setup-docker   # force zero-setup Docker mode"
	@echo "  make ci-local       # native CI-like checks"
	@echo "  make docker-ci      # Docker CI-like checks"
	@echo "  make gh-protect BRANCH=main"
	@echo ""
	@echo "Reference: docs/build-test-and-development-commands.md"

bootstrap: setup

bootstrap-native: setup-native

bootstrap-docker: setup-docker

check:
	@if command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then \
		echo "docker daemon detected: running zero-setup CI checks"; \
		$(MAKE) docker-ci; \
	else \
		echo "docker daemon unavailable: running native CI checks"; \
		$(MAKE) ci-local; \
	fi

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
	GOCOVERDIR= go test -covermode=atomic -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

test-cover-local:
	@coverage_log="$$(mktemp)"; \
	if GOCOVERDIR= go test -covermode=atomic -coverprofile=coverage.out ./... >"$$coverage_log" 2>&1; then \
		cat "$$coverage_log"; \
		go tool cover -func=coverage.out; \
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
	@test -n "$(MIGRATION_DSN)" || (echo "MIGRATION_DSN is required"; exit 1)
	go run github.com/golang-migrate/migrate/v4/cmd/migrate@$(MIGRATE_VERSION) -path env/migrations -database "$(MIGRATION_DSN)" up
	go run github.com/golang-migrate/migrate/v4/cmd/migrate@$(MIGRATE_VERSION) -path env/migrations -database "$(MIGRATION_DSN)" down 1
	go run github.com/golang-migrate/migrate/v4/cmd/migrate@$(MIGRATE_VERSION) -path env/migrations -database "$(MIGRATION_DSN)" up 1

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
