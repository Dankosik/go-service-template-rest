SERVICE_NAME := service
SERVICE_CMD := ./cmd/service
BINARY := bin/$(SERVICE_NAME)
OPENAPI_FILE := api/openapi/service.yaml
REFERENCE_OPENAPI_FILE := $(wildcard examples/reference-service/api/openapi.yaml)
REFERENCE_OPENAPI_PACKAGE := $(if $(REFERENCE_OPENAPI_FILE),./examples/reference-service/internal/openapi)
OPENAPI_FILES := $(OPENAPI_FILE) $(REFERENCE_OPENAPI_FILE)
OPENAPI_PACKAGES := ./internal/openapi $(REFERENCE_OPENAPI_PACKAGE)
GO_FILES := $(shell git ls-files --cached --others --exclude-standard -- '*.go' 2>/dev/null | awk '!/^(\.agents|\.cache|vendor)\//' | while IFS= read -r file; do [ -f "$$file" ] && printf '%s\n' "$$file"; done)
GOFUMPT_FILES := $(filter-out internal/openapi/openapi.gen.go internal/infra/postgres/sqlcgen/%,$(GO_FILES))
REDOCLY_CLI_VERSION := 2.40.0
GO_TOOL := bash ./scripts/run-go-tool.sh
GOLANGCI_LINT ?= $(GO_TOOL) golangci-lint
GO_REQUIRED_VERSION := $(shell awk '/^go / {print $$2; exit}' go.mod)
TEST_REPORT_DIR := .artifacts/test
TEST_JUNIT_FILE := $(TEST_REPORT_DIR)/junit.xml
TEST_JSON_FILE := $(TEST_REPORT_DIR)/test2json.json
# Effective coverage is measured across the whole module, so a freshly generated
# service already sits near this floor on template tests alone. Initialization
# lowers it to 70.0 so early feature work has runway; raise it as your own tests
# land. See rebase_coverage_floor in scripts/init-module.sh.
COVERAGE_MIN ?= 80.0
COVERAGE_GOTOOLCHAIN ?= go$(GO_REQUIRED_VERSION)
COVERAGE_EXCLUDE_REGEX ?= (^|/)internal/openapi/openapi\.gen\.go:|(^|/)internal/infra/postgres/sqlcgen/|(^|/)internal/infra/postgres/pgtest/|(^|/)internal/infra/telemetry/telemetrytest/|(^|/)cmd/service/main\.go:|(^|/)cmd/migrate/main\.go:
FUZZ_TIME ?= 45s
LINT_BASE_REF ?= origin/main
LINT_CONCURRENCY ?= 4
GENTLE_GOMAXPROCS ?= 6
GENTLE_NICE ?= 10
SECRET_SCAN_BASE_REF ?= $(if $(strip $(BASE_REF)),$(BASE_REF),origin/main)

BENCH_PACKAGE ?= ./...
BENCH_PATTERN ?= .
BENCH_COUNT ?= 10
BENCH_TIME ?= 1s
BENCH_TAGS ?=
BENCH_OUTPUT ?= .artifacts/bench/current.txt
BENCH_BASELINE ?= .artifacts/bench/baseline.txt
BENCH_CURRENT ?= .artifacts/bench/current.txt
BENCH_COMPARE_OUTPUT ?= .artifacts/bench/comparison.txt
BENCH_PROFILE ?= cpu
BENCH_PROFILE_DIR ?= .artifacts/bench/profiles
BENCH_WORKLOAD_ID ?= go-benchmark
# profile:database-postgres:start
BENCH_DB_PACKAGE ?= ./test/...
BENCH_DB_PATTERN ?= .
BENCH_DB_OUTPUT ?= .artifacts/bench/db/current.txt
BENCH_DB_BASELINE ?= .artifacts/bench/db/baseline.txt
BENCH_DB_CURRENT ?= .artifacts/bench/db/current.txt
BENCH_DB_COMPARE_OUTPUT ?= .artifacts/bench/db/comparison.txt
BENCH_DB_WORKLOAD_ID ?=
BENCH_DB_SCHEMA_PATH := $(if $(wildcard migrations/*.up.sql),migrations,)
POSTGRES_TEST_IMAGE := $(shell sed -n 's/^const DefaultImage = "\(.*\)"$$/\1/p' internal/infra/postgres/pgtest/pgtest.go)
# profile:database-postgres:end
HTTP_BENCH_SCRIPT ?= test/performance/http/single-flow.js
HTTP_BENCH_ARTIFACT_DIR ?= .artifacts/bench/http
HTTP_BENCH_ENV_FILE ?= .env.bench
HTTP_BENCH_DOCKER_NETWORK ?=
HTTP_BENCH_RAW_SAMPLES ?= 0

TRIVY_IMAGE ?= aquasec/trivy:0.72.0@sha256:cffe3f5161a47a6823fbd23d985795b3ed72a4c806da4c4df16266c02accdd6f
TRIVY_CACHE_VOLUME ?= trivy-cache
CLAUDE_SKILLS_CHECK_SCRIPT := bash ./scripts/ci/claude-skills-check.sh
CI_CHANGE_SCOPE_SCRIPT := bash ./scripts/ci/ci-change-scope.sh
GENERATED_DRIFT_CHECK_SCRIPT := bash ./scripts/ci/generated-drift-check.sh
PROJECT_STRUCTURE_CHECK_SCRIPT := bash ./scripts/ci/project-structure-check.sh
SECRET_SCAN_SCRIPT := bash ./scripts/ci/secret-scan.sh
TEMPLATE_OWNED_PURITY_CHECK_SCRIPT := bash ./scripts/ci/template-owned-purity-check.sh
TEMPLATE_SYNC_SCRIPT := bash ./scripts/template-sync.sh
TEMPLATE ?= ../go-service-template-rest
BENCHMARK_SCRIPT := bash ./scripts/dev/benchmark.sh
BENCHMARK_REMOTE_SCRIPT := bash ./scripts/dev/benchmark-remote.sh

.DEFAULT_GOAL := help

# One same-target A/B on the 10-core/16-GiB reference Mac measured 138.7s
# serial versus 294.6s with make -j4. Re-measure after host, toolchain, or
# aggregate membership changes before enabling parallel prerequisites.
.NOTPARALLEL: check mod-check lint-deep go-security openapi-check ci-local

.PHONY: help template-init template-init-check project-structure-check ci-change-scope-check check check-gentle check-full check-full-gentle pr-check \
	tidy fmt mod-check mod-tidy-check mod-verify fmt-check test test-watch test-race test-cover test-report coverage-effective-total coverage-summary coverage-check test-fuzz-smoke test-flake-smoke test-integration \
	bench bench-baseline bench-compare bench-profile bench-http bench-http-inspect benchmark-infra-check benchmark-remote-check benchmark-remote-image \
	lint lint-deep lint-fast deadcode nilaway modernize-check test-parallelism-check govulncheck gosec go-security secret-scan secret-scan-history secret-scan-check ci-local ci-local-gentle \
	openapi-generate openapi-drift-check openapi-reference-compile openapi-runtime-contract-check openapi-lint openapi-validate openapi-breaking openapi-check \
	sqlc-check container-security run build docker-build docker-run vendor claude-skills-sync claude-skills-check \
	template-sync template-sync-check template-sync-all template-owned-purity-check
# profile:database-postgres:start
.PHONY: bench-db bench-db-baseline bench-db-compare sqlc-generate migration-validate compose-up compose-down
# profile:database-postgres:end

help:
	@echo "Setup and everyday development:"
	@echo "  make template-init MODULE=github.com/acme/service CODEOWNER=@acme/team"
	@echo "  make check              # formatting, lint, and unit tests"
	@echo "  make check-gentle       # same checks with bounded Go concurrency"
	@echo "  make project-structure-check"
	@echo "  make template-sync-check TEMPLATE=<path>   # drift against the template instructions"
	@echo "  make template-sync TEMPLATE=<path>         # adopt them as its own commit"
	@echo "  make ci-local           # deterministic native CI aggregate"
	@echo "  make ci-local-gentle    # same aggregate with bounded Go concurrency"
	@echo "  make check-full         # native aggregate plus Docker-backed gates"
	@echo "  make check-full-gentle  # same full gate with bounded host Go concurrency"
	@echo "  make pr-check BASE_REF=origin/main"
	@echo "  make run"
	@echo ""
	@echo "Focused validation:"
	@echo "  make test | test-race | test-report | test-integration"
	@echo "  make mod-check | mod-tidy-check | mod-verify"
	@echo "  make lint | lint-deep | lint-fast | go-security | secret-scan | secret-scan-history"
	@echo "  make openapi-check"
# profile:database-postgres:start
	@echo "  make sqlc-check | migration-validate"
# profile:database-postgres:end
	@echo "  make docker-build | container-security"
	@echo ""
	@echo "Benchmarking:"
	@echo "  make bench | bench-baseline | bench-compare | bench-profile"
# profile:database-postgres:start
	@echo "  make bench-db BENCH_DB_WORKLOAD_ID=<fixture-state> | bench-db-baseline | bench-db-compare"
# profile:database-postgres:end
	@echo "  make bench-http | bench-http-inspect | benchmark-infra-check | benchmark-remote-check | benchmark-remote-image"
	@echo ""
	@echo "Reference: docs/build-test-and-development-commands.md"

template-init:
	@if [ -n "$(MODULE)" ]; then \
		CODEOWNER="$(CODEOWNER)" bash ./scripts/init-module.sh "$(MODULE)"; \
	else \
		CODEOWNER="$(CODEOWNER)" bash ./scripts/init-module.sh; \
	fi

template-init-check:
	bash ./scripts/ci/template-init-check.sh

project-structure-check:
	$(PROJECT_STRUCTURE_CHECK_SCRIPT)

ci-change-scope-check:
	$(CI_CHANGE_SCOPE_SCRIPT) self-test

template-owned-purity-check:
	$(TEMPLATE_OWNED_PURITY_CHECK_SCRIPT)

# TEMPLATE points at a checkout of the template that owns the instructions.
# Run these from the derived repository; the template is the source of truth.
template-sync-check:
	$(TEMPLATE_SYNC_SCRIPT) --check --from "$(TEMPLATE)" --repo .

template-sync:
	$(TEMPLATE_SYNC_SCRIPT) --apply --from "$(TEMPLATE)" --repo .

# Fan out from this template to several local checkouts in one run.
template-sync-all:
	@if [ -z "$(TARGETS)" ]; then echo "TARGETS is required: make template-sync-all TARGETS=\"../a ../b\"" >&2; exit 2; fi
	$(TEMPLATE_SYNC_SCRIPT) --apply --from . --targets $(TARGETS)

# .claude/skills holds nothing but generated links, so every entry is cleared
# before the rebuild. Deleting only symlinks would leave behind the regular
# files a checkout without symlink support materializes, and `ln -s` would then
# either fail on a file or silently create the link *inside* a directory.
# A real directory is the one entry that may hold the only copy of something,
# so it stops the rebuild instead of being removed.
claude-skills-sync:
	@mkdir -p .claude/skills
	@set -e; for entry in .claude/skills/*; do \
		{ [ -e "$$entry" ] || [ -L "$$entry" ]; } || continue; \
		if [ -d "$$entry" ] && [ ! -L "$$entry" ]; then \
			echo "$$entry is a real directory, not a generated link; move or remove it first" >&2; \
			exit 1; \
		fi; \
		rm -f "$$entry"; \
	done
	@set -e; for d in .agents/skills/*/; do n=$$(basename "$$d"); ln -s "../../.agents/skills/$$n" ".claude/skills/$$n"; done
	@echo ".claude/skills: $$(ls .claude/skills | wc -l | tr -d ' ') skill links"

claude-skills-check:
	$(CLAUDE_SKILLS_CHECK_SCRIPT)

check: project-structure-check ci-change-scope-check template-owned-purity-check claude-skills-check fmt-check lint test

check-gentle:
	nice -n $(GENTLE_NICE) env GOMAXPROCS=$(GENTLE_GOMAXPROCS) $(MAKE) check

ci-local: mod-tidy-check project-structure-check ci-change-scope-check template-owned-purity-check claude-skills-check fmt-check lint lint-deep test-race test-report sqlc-check openapi-check go-security secret-scan

ci-local-gentle:
	nice -n $(GENTLE_NICE) env GOMAXPROCS=$(GENTLE_GOMAXPROCS) $(MAKE) ci-local

check-full:
	@command -v docker >/dev/null 2>&1 || { echo "Docker is required for make check-full"; exit 1; }
	@docker info >/dev/null 2>&1 || { echo "Docker daemon is not reachable"; exit 1; }
	$(MAKE) ci-local
	REQUIRE_DOCKER=1 $(MAKE) test-integration
	docker build -f build/docker/Dockerfile -t $(SERVICE_NAME):ci .
# profile:database-postgres:start
	$(MAKE) migration-validate RUNTIME_IMAGE=$(SERVICE_NAME):ci
# profile:database-postgres:end
	$(MAKE) container-security CONTAINER_IMAGE=$(SERVICE_NAME):ci

check-full-gentle:
	nice -n $(GENTLE_NICE) env GOMAXPROCS=$(GENTLE_GOMAXPROCS) $(MAKE) check-full

pr-check:
	@test -n "$(BASE_REF)" || { echo "BASE_REF is required, for example BASE_REF=origin/main"; exit 1; }
	$(MAKE) check-full
	$(MAKE) template-init-check
	$(MAKE) mod-verify
	@mkdir -p .cache
	@base_openapi="$$(mktemp .cache/openapi-base.XXXXXX)"; \
	trap 'rm -f "$$base_openapi"' EXIT; \
	if git show "$(BASE_REF):$(OPENAPI_FILE)" >"$$base_openapi" 2>/dev/null; then \
		$(MAKE) openapi-breaking BASE_OPENAPI="$$base_openapi"; \
	else \
		echo "No base OpenAPI spec at $(BASE_REF):$(OPENAPI_FILE); breaking check not applicable"; \
	fi

tidy:
	go mod tidy

fmt:
	$(GO_TOOL) goimports -w $(GO_FILES)
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
	@unformatted="$$( $(GO_TOOL) goimports -l $(GO_FILES) )"; \
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

test-cover:
	GOTOOLCHAIN=$(COVERAGE_GOTOOLCHAIN) GOCOVERDIR= go test -vet=off -covermode=set -coverprofile=coverage.out ./...
	$(MAKE) coverage-summary

test-report:
	@mkdir -p $(TEST_REPORT_DIR)
	GOTOOLCHAIN=$(COVERAGE_GOTOOLCHAIN) GOCOVERDIR= $(GO_TOOL) gotestsum --format=pkgname-and-test-fails --junitfile=$(TEST_JUNIT_FILE) --jsonfile=$(TEST_JSON_FILE) -- -vet=off -covermode=set -coverprofile=coverage.out ./...
	$(MAKE) coverage-summary
	$(MAKE) coverage-check COVERAGE_MIN=$(COVERAGE_MIN)

coverage-effective-total:
	@test -f coverage.out || { echo "coverage.out not found; run 'make test-cover' or 'make test-report'"; exit 1; }
	@filtered_cov="$$(mktemp)"; \
	trap 'rm -f "$$filtered_cov"' EXIT; \
	grep -Ev '$(COVERAGE_EXCLUDE_REGEX)' coverage.out >"$$filtered_cov"; \
	total="$$(GOTOOLCHAIN=$(COVERAGE_GOTOOLCHAIN) go tool cover -func="$$filtered_cov" | awk '/^total:/ {gsub(/%/, "", $$3); print $$3}')"; \
	test -n "$$total" || { echo "failed to parse total coverage"; exit 1; }; \
	printf '%s\n' "$$total"

coverage-summary:
	@test -f coverage.out || { echo "coverage.out not found; run 'make test-cover' or 'make test-report'"; exit 1; }
	@raw="$$(GOTOOLCHAIN=$(COVERAGE_GOTOOLCHAIN) go tool cover -func=coverage.out | awk '/^total:/ {print $$3}')"; \
	effective="$$( $(MAKE) --no-print-directory coverage-effective-total )"; \
	test -n "$$raw" && test -n "$$effective" || { echo "failed to parse coverage totals"; exit 1; }; \
	printf 'Raw coverage: %s\nEffective coverage (filtered): %s%%\n' "$$raw" "$$effective"

coverage-check:
	@total="$$( $(MAKE) --no-print-directory coverage-effective-total )"; \
	awk -v total="$$total" -v minimum="$(COVERAGE_MIN)" 'BEGIN { \
		if ((total + 0) < (minimum + 0)) { \
			printf "coverage %.2f%% is below threshold %.2f%%\n", total, minimum; \
			exit 1; \
		} \
		printf "coverage %.2f%% meets threshold %.2f%%\n", total, minimum; \
	}'

test-fuzz-smoke:
	@found=0; \
	pkgs="$$(go list ./...)" || exit $$?; \
	for pkg in $$pkgs; do \
		fuzz_targets="$$(go test -vet=off "$$pkg" -list '^Fuzz' 2>&1)" || { status=$$?; printf '%s\n' "$$fuzz_targets"; exit $$status; }; \
		if printf '%s\n' "$$fuzz_targets" | grep -q '^Fuzz'; then \
			found=1; \
			go test -vet=off "$$pkg" -run '^$$' -fuzz=Fuzz -fuzztime=$(FUZZ_TIME) || exit $$?; \
		fi; \
	done; \
	if [ "$$found" -eq 0 ]; then echo "no fuzz targets found; skipping fuzz smoke run"; fi

test-flake-smoke:
	go test -vet=off -count=5 -shuffle=on ./...

test-integration:
	go test -count=1 -tags=integration ./test/...

bench:
	BENCH_PACKAGE="$(BENCH_PACKAGE)" BENCH_PATTERN="$(BENCH_PATTERN)" BENCH_COUNT="$(BENCH_COUNT)" BENCH_TIME="$(BENCH_TIME)" BENCH_TAGS="$(BENCH_TAGS)" BENCH_OUTPUT="$(BENCH_OUTPUT)" BENCH_WORKLOAD_ID="$(BENCH_WORKLOAD_ID)" $(BENCHMARK_SCRIPT) run

bench-baseline:
	$(MAKE) bench BENCH_OUTPUT="$(BENCH_BASELINE)"

bench-compare:
	BENCH_BASELINE="$(BENCH_BASELINE)" BENCH_CURRENT="$(BENCH_CURRENT)" BENCH_COMPARE_OUTPUT="$(BENCH_COMPARE_OUTPUT)" $(BENCHMARK_SCRIPT) compare

bench-profile:
	BENCH_PACKAGE="$(BENCH_PACKAGE)" BENCH_PATTERN="$(BENCH_PATTERN)" BENCH_TIME="$(BENCH_TIME)" BENCH_TAGS="$(BENCH_TAGS)" BENCH_PROFILE="$(BENCH_PROFILE)" BENCH_PROFILE_DIR="$(BENCH_PROFILE_DIR)" BENCH_WORKLOAD_ID="$(BENCH_WORKLOAD_ID)" $(BENCHMARK_SCRIPT) profile

# profile:database-postgres:start
bench-db:
	@test -n "$(BENCH_DB_WORKLOAD_ID)" || { echo "BENCH_DB_WORKLOAD_ID is required, for example fixture-10k-warm"; exit 1; }
	REQUIRE_DOCKER=1 BENCH_PACKAGE="$(BENCH_DB_PACKAGE)" BENCH_PATTERN="$(BENCH_DB_PATTERN)" BENCH_COUNT="$(BENCH_COUNT)" BENCH_TIME="$(BENCH_TIME)" BENCH_TAGS=integration BENCH_OUTPUT="$(BENCH_DB_OUTPUT)" BENCH_WORKLOAD_ID="$(BENCH_DB_WORKLOAD_ID)" BENCH_DEPENDENCY_IMAGE="$(POSTGRES_TEST_IMAGE)" BENCH_SCHEMA_PATH="$(BENCH_DB_SCHEMA_PATH)" $(BENCHMARK_SCRIPT) run

bench-db-baseline:
	$(MAKE) bench-db BENCH_DB_OUTPUT="$(BENCH_DB_BASELINE)"

bench-db-compare:
	BENCH_BASELINE="$(BENCH_DB_BASELINE)" BENCH_CURRENT="$(BENCH_DB_CURRENT)" BENCH_COMPARE_OUTPUT="$(BENCH_DB_COMPARE_OUTPUT)" $(BENCHMARK_SCRIPT) compare
# profile:database-postgres:end

bench-http:
	HTTP_BENCH_SCRIPT="$(HTTP_BENCH_SCRIPT)" HTTP_BENCH_ARTIFACT_DIR="$(HTTP_BENCH_ARTIFACT_DIR)" HTTP_BENCH_ENV_FILE="$(HTTP_BENCH_ENV_FILE)" HTTP_BENCH_DOCKER_NETWORK="$(HTTP_BENCH_DOCKER_NETWORK)" HTTP_BENCH_RAW_SAMPLES="$(HTTP_BENCH_RAW_SAMPLES)" $(BENCHMARK_SCRIPT) http

bench-http-inspect:
	HTTP_BENCH_SCRIPT="$(HTTP_BENCH_SCRIPT)" HTTP_BENCH_ARTIFACT_DIR="$(HTTP_BENCH_ARTIFACT_DIR)" HTTP_BENCH_ENV_FILE="$(HTTP_BENCH_ENV_FILE)" HTTP_BENCH_DOCKER_NETWORK="$(HTTP_BENCH_DOCKER_NETWORK)" HTTP_BENCH_RAW_SAMPLES=0 $(BENCHMARK_SCRIPT) http-inspect

benchmark-infra-check:
	$(BENCHMARK_SCRIPT) check

benchmark-remote-check:
	$(BENCHMARK_REMOTE_SCRIPT) check

benchmark-remote-image:
	$(BENCHMARK_REMOTE_SCRIPT) image-build

# lint is the gate `make check` runs, so it stays fast enough to run before every
# commit. deadcode and nilaway are whole-program analyses that dominate its wall
# clock; they live in lint-deep, which ci-local and the CI lint job run.
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

govulncheck:
	$(GO_TOOL) govulncheck ./...

gosec:
	GOSECGOVERSION=go$(GO_REQUIRED_VERSION) $(GO_TOOL) gosec $(if $(strip $(GOMAXPROCS)),-concurrency=$(GOMAXPROCS)) -quiet -exclude-generated -exclude-dir=.agents -exclude-dir=.cache -exclude-dir=.artifacts ./...

go-security: govulncheck gosec

secret-scan:
	$(SECRET_SCAN_SCRIPT) change "$(SECRET_SCAN_BASE_REF)"

secret-scan-history:
	$(SECRET_SCAN_SCRIPT) history

secret-scan-check:
	$(SECRET_SCAN_SCRIPT) self-test

# profile:database-postgres:start
sqlc-generate:
	@if ! find internal/infra/postgres/queries -type f -name '*.sql' -print -quit 2>/dev/null | grep -q .; then \
		echo "no sqlc query sources; skipping sqlc generation"; \
	else \
		$(GO_TOOL) sqlc generate -f internal/infra/postgres/sqlc.yaml; \
	fi
# profile:database-postgres:end

# sqlc-check stays outside the database profile because CI runs it unconditionally.
# Without query sources it reports that there is nothing to generate and also
# fails on generated output left behind without them, which is the drift a
# profile-less service can still have.
sqlc-check:
	$(GENERATED_DRIFT_CHECK_SCRIPT) sqlc

openapi-generate:
	go generate $(OPENAPI_PACKAGES)

openapi-drift-check:
	$(GENERATED_DRIFT_CHECK_SCRIPT) openapi

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
	REDOCLY_SUPPRESS_UPDATE_NOTICE=true REDOCLY_TELEMETRY=off npm_config_prefer_offline=true npx --yes @redocly/cli@$(REDOCLY_CLI_VERSION) lint --config .redocly.yaml $(OPENAPI_FILES)

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

# profile:database-postgres:start
migration-validate:
	@command -v docker >/dev/null 2>&1 || { echo "Docker is required"; exit 1; }; \
	docker info >/dev/null 2>&1 || { echo "Docker daemon is not reachable"; exit 1; }; \
	project="service-migration-$$(date +%s)-$$$$"; \
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
	image="$(RUNTIME_IMAGE)"; \
	if [ -z "$$image" ]; then image="$(SERVICE_NAME):migration"; docker build -f build/docker/Dockerfile -t "$$image" .; fi; \
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
	@command -v docker >/dev/null 2>&1 || { echo "Docker is required"; exit 1; }
	@docker info >/dev/null 2>&1 || { echo "Docker daemon is not reachable"; exit 1; }
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

run:
	@set -a; \
	if [ -f .env ]; then . ./.env; fi; \
	set +a; \
	go run $(SERVICE_CMD)

build:
	mkdir -p bin
	CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o $(BINARY) $(SERVICE_CMD)

docker-build:
	docker build -f build/docker/Dockerfile -t $(SERVICE_NAME):local .

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
