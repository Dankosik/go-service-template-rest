#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${ROOT_DIR}"

failed=0
fail() {
	echo "project structure: $*"
	failed=1
}

for forbidden_dir in pkg internal/app internal/api internal/requestmeta; do
	if [[ -e "${forbidden_dir}" ]]; then
		fail "${forbidden_dir}/ is not an approved package owner"
	fi
done

while IFS= read -r file; do
	[[ -n "${file}" ]] && fail "${file} uses the forbidden *_additional_test.go suffix"
done < <(find cmd internal test -type f -name '*_additional_test.go' -print 2>/dev/null)

while IFS= read -r file; do
	[[ -n "${file}" ]] && fail "${file} uses an ordinal *_partN_test.go suffix"
done < <(find cmd internal test -type f -name '*_part[0-9]*_test.go' -print 2>/dev/null)

while IFS= read -r file; do
	[[ -n "${file}" ]] && fail "${file} must use <package>_test.go for shared package test helpers"
done < <(find cmd internal test -type f -name 'test_helpers_test.go' -print 2>/dev/null)

while IFS= read -r file; do
	[[ -n "${file}" ]] && fail "${file} uses a generic *_helpers.go production name"
done < <(find cmd internal -type f -name '*_helpers.go' ! -name '*_test.go' -print 2>/dev/null)

for generic_name in util.go common.go misc.go; do
	while IFS= read -r file; do
		[[ -n "${file}" ]] && fail "${file} uses the forbidden generic production name ${generic_name}"
	done < <(find cmd internal -type f -name "${generic_name}" -print 2>/dev/null)
done

while IFS= read -r file; do
	base="${file##*/}"
	if [[ "${base}" != "openapi.gen.go" && ! "${base}" =~ ^[a-z0-9]+(_[a-z0-9]+)*(_test)?\.go$ ]]; then
		fail "${file} must use lowercase snake_case Go file naming"
	fi
done < <(find cmd internal test -type f -name '*.go' -print 2>/dev/null)

for command_dir in cmd/*; do
	[[ -d "${command_dir}" ]] || continue
	[[ -f "${command_dir}/main.go" ]] || fail "${command_dir}/ must contain main.go"
done

for integration_test in test/*_test.go; do
	[[ -e "${integration_test}" ]] || continue
	[[ "${integration_test}" == *_integration_test.go ]] ||
		fail "${integration_test} must use the *_integration_test.go suffix"
	[[ "$(head -n 1 "${integration_test}")" == "//go:build integration" ]] ||
		fail "${integration_test} must start with //go:build integration"
	grep -Eq '^package integration_test$' "${integration_test}" ||
		fail "${integration_test} must declare package integration_test"
done

for skill_dir in .agents/skills/*; do
	[[ -d "${skill_dir}" ]] || continue
	[[ -f "${skill_dir}/SKILL.md" ]] || fail "${skill_dir}/ must contain SKILL.md"
done

if [[ -d api/proto ]] && ! find api/proto -type f -name '*.proto' -print -quit | grep -q .; then
	fail "api/proto/ must not exist before the first owned .proto contract"
fi
if [[ -d migrations ]] && ! find migrations -type f -name '*.up.sql' -print -quit | grep -q .; then
	fail "migrations/ must not exist before the first owned migration"
fi
if [[ -d internal/infra/postgres/queries ]] &&
	! find internal/infra/postgres/queries -type f -name '*.sql' -print -quit | grep -q .; then
	fail "internal/infra/postgres/queries/ must not exist before the first owned query"
fi
if [[ -d internal/infra/postgres/sqlcgen ]] &&
	! find internal/infra/postgres/sqlcgen -type f -name '*.go' -print -quit | grep -q .; then
	fail "internal/infra/postgres/sqlcgen/ must not exist without generated Go output"
fi

# Template fixtures must share the cache selected by Go and restored by CI.
# A repository-local override silently creates a second multi-gigabyte cache.
if grep -Eq '(^|[[:space:]])(export[[:space:]]+)?(GOCACHE|GOLANGCI_LINT_CACHE)=' \
	scripts/ci/template-init-check.sh; then
	fail "template-init-check must not override Go or golangci-lint cache paths"
fi

if grep -Fq -- '--allow-parallel-runners' Makefile scripts/ci/template-init-check.sh; then
	fail "golangci-lint callers must wait on the native serial-runner lock"
fi
grep -Fq -- '--allow-serial-runners' Makefile ||
	fail "Make lint targets must retain native runner serialization"
grep -Fq -- '--allow-serial-runners' scripts/ci/template-init-check.sh ||
	fail "template-init-check must retain native runner serialization"
grep -Fq 'GOLANGCI_LINT_BIN' scripts/ci/template-init-check.sh ||
	fail "template-init-check must accept the CI-installed golangci-lint binary"
grep -Fq 'TEMPLATE_INIT_PROFILE' scripts/ci/template-init-check.sh ||
	fail "template-init-check must retain independently runnable profile proof"

grep -Fq '.NOTPARALLEL: check mod-check lint-deep go-security openapi-check ci-local' Makefile ||
	fail "heavy Make aggregates must remain serial under inherited MAKEFLAGS"
grep -Fq 'ci-local: mod-tidy-check project-structure-check' Makefile ||
	fail "ci-local must keep manifest drift without repeating full module and generator integrity"
grep -A5 '^pr-check:' Makefile | grep -Fq $'\t$(MAKE) mod-verify' ||
	fail "pr-check must retain downloaded-module verification"
grep -A5 '^pr-check:' Makefile | grep -Fq $'\t$(MAKE) template-init-check' ||
	fail "pr-check must retain the complete template initialization contract"
grep -Fqx 'LINT_CONCURRENCY ?= 4' Makefile ||
	fail "golangci-lint must retain the measured four-worker default"
grep -Fq -- '-covermode=set' Makefile ||
	fail "coverage must use statement-presence counters"
if grep -Fq -- '-covermode=atomic' Makefile; then
	fail "coverage must not pay for unused atomic execution counts"
fi
grep -Fq 'GOSECGOVERSION=go$(GO_REQUIRED_VERSION)' Makefile ||
	fail "gosec must receive the repository Go version without a fallback go list"
if grep -Fq -- '-tags=timetzdata' build/docker/Dockerfile; then
	fail "the Distroless runtime already owns timezone data"
fi

for workflow in .github/workflows/ci.yml .github/workflows/cd.yml; do
	grep -Fq 'golangci/golangci-lint-action@e7fa5ac41e1cf5b7d48e45e42232ce7ada589601' "${workflow}" ||
		fail "${workflow} must use the pinned prebuilt golangci-lint installer"
done
if grep -Fq 'skip-cache: true' \
	.github/workflows/ci.yml .github/workflows/nightly.yml .github/workflows/cd.yml; then
	fail "workflows must retain the action-owned golangci-lint analysis cache"
fi

for duplicate in \
	'actions/setup-node@' \
	'golangci/golangci-lint-action@' \
	'make mod-check' \
	'make project-structure-check' \
	'make fmt-check' \
	'make lint' \
	'make modernize-check' \
	'make test-parallelism-check' \
	'make test-race' \
	'make openapi-' \
	'make secret-scan-history' \
	'repository-security:'; do
	if grep -Fq "${duplicate}" .github/workflows/nightly.yml; then
		fail "nightly must not repeat deterministic merge proof: ${duplicate}"
	fi
done
for nightly_proof in \
	'make test-flake-smoke' \
	'make test-fuzz-smoke' \
	'make test-integration' \
	'make benchmark-infra-check' \
	'make go-security' \
	'docker build -f build/docker/Dockerfile' \
	'aquasecurity/trivy-action@'; do
	grep -Fq "${nightly_proof}" .github/workflows/nightly.yml ||
		fail "nightly must retain time-varying proof: ${nightly_proof}"
done

[[ "$(grep -Fc '"${LINTER}" run' scripts/ci/template-init-check.sh)" == 1 ]] ||
	fail "template-init-check must prove generated lint and depguard in one linter load"
grep -Fq 'generated minimal profile must have exactly the intentional depguard issue' \
	scripts/ci/template-init-check.sh ||
	fail "template-init-check must reject any generated lint issue beyond the depguard probe"

if grep -Rn 'postgresTestImage' Makefile scripts/dev test/README.md; then
	fail "tooling must read pgtest.DefaultImage instead of a removed test-local image constant"
fi

if [[ -f internal/infra/postgres/pgtest/pgtest.go ]]; then
	grep -Fq 'tcpostgres.BasicWaitStrategies()' internal/infra/postgres/pgtest/pgtest.go ||
		fail "pgtest must retain the Mac-safe PostgreSQL readiness strategy"
	grep -Fq '"POSTGRES_INITDB_ARGS": "--no-sync"' internal/infra/postgres/pgtest/pgtest.go ||
		fail "the disposable PostgreSQL harness must retain measured initdb tuning"
	if grep -Eq 'WithReuseByName|TESTCONTAINERS_RYUK_DISABLED' \
		internal/infra/postgres/pgtest/pgtest.go; then
		fail "local Testcontainers must retain fresh containers and Ryuk cleanup"
	fi
fi
if grep -Fq 'TESTCONTAINERS_RYUK_DISABLED' Makefile; then
	fail "local Testcontainers must retain Ryuk cleanup"
fi

if grep -Fq 'TESTCONTAINERS_RYUK_DISABLED' \
	.github/workflows/ci.yml .github/workflows/nightly.yml .github/workflows/cd.yml; then
	fail "workflow Testcontainers must retain Ryuk cleanup"
fi

for ignored in Makefile railway.toml template-owned.paths .golangci.yml; do
	grep -Fqx "${ignored}" .dockerignore ||
		fail ".dockerignore must exclude compiler-irrelevant ${ignored}"
done

if [[ -f env/docker-compose.yml ]]; then
	grep -Fq 'POSTGRES_INITDB_ARGS: --no-sync' env/docker-compose.yml ||
		fail "disposable Compose PostgreSQL must retain measured initdb tuning"
	grep -Fq 'interval: 30s' env/docker-compose.yml ||
		fail "Compose must retain its low-wakeup steady healthcheck"
	grep -Fq 'start_interval: 1s' env/docker-compose.yml ||
		fail "Compose must retain fast startup healthchecks"
fi

if [[ -f .github/workflows/ci.yml ]]; then
	ci_image_builds="$(grep -Ec '^[[:space:]]+docker build([[:space:]]|$)' .github/workflows/ci.yml || true)"
	[[ "${ci_image_builds}" == "1" ]] ||
		fail "CI must build the production runtime image once, found ${ci_image_builds} builds"
	grep -Fq 'scripts/ci/ci-change-scope.sh classify' .github/workflows/ci.yml ||
		fail "CI must use the fail-closed change-scope classifier"
	grep -Fq 'scripts/ci/ci-change-scope.sh template-required' .github/workflows/ci.yml ||
		fail "CI must use the fail-closed template-scope classifier"
	grep -Fq "needs.repo-integrity.outputs.template_required == 'true'" .github/workflows/ci.yml ||
		fail "template jobs must be gated by their owned change scope"
	grep -Fqx '  template-minimal-feature:' .github/workflows/ci.yml ||
		fail "CI must retain the independently scheduled minimal template proof"
	grep -Fq 'TEMPLATE_INIT_PROFILE=minimal make template-init-check' .github/workflows/ci.yml ||
		fail "minimal template CI must select only its owned profile"
	grep -Fq 'TEMPLATE_INIT_PROFILE=postgres TEMPLATE_POSTGRES_PROOF=1 make template-init-check' .github/workflows/ci.yml ||
		fail "PostgreSQL template CI must select only its owned profile"
	grep -A20 '^  ci-required:$' .github/workflows/ci.yml |
		grep -Fq -- '- template-minimal-feature' ||
		fail "ci-required must include the minimal template proof"
	grep -Fq 'TEMPLATE_REQUIRED: ${{ needs.repo-integrity.outputs.template_required }}' .github/workflows/ci.yml ||
		fail "ci-required must distinguish an intentional template-job skip"
	grep -Fq 'go_cache_enabled:' .github/workflows/ci.yml ||
		fail "manual CI must retain the setup-go cache A/B input"
	if grep -Fq 'cache: true' .github/workflows/ci.yml; then
		fail "CI setup-go cache selection must remain measurable through GO_CACHE_ENABLED"
	fi
	setup_go_steps="$(grep -Fc 'uses: actions/setup-go@' .github/workflows/ci.yml)"
	cache_selected_steps="$(grep -Fc 'cache: ${{ env.GO_CACHE_ENABLED }}' .github/workflows/ci.yml)"
	[[ "${cache_selected_steps}" == "${setup_go_steps}" ]] ||
		fail "every CI setup-go step must use the cache A/B input"
	grep -Fq 'gotestsum tool slowest' .github/workflows/ci.yml ||
		fail "coverage summary must retain the no-rerun slow-test report"
	grep -A1 '^  dependency-review:$' .github/workflows/ci.yml |
		grep -Fq "github.event_name == 'pull_request' && needs.repo-integrity.outputs.expensive_required == 'true'" ||
		fail "Dependency Review must skip classifier-proven Markdown-only changes"
	if grep -Eq '^  migration-validate:' .github/workflows/ci.yml; then
		fail "migration validation must reuse the image from container-security, not build in a separate job"
	fi
fi

while IFS= read -r profile_path; do
	[[ -n "${profile_path}" ]] || continue
	template_scope="$(
		printf '%s\0' "${profile_path}" |
			bash ./scripts/ci/ci-change-scope.sh template-required
	)"
	[[ "${template_scope}" == "true" ]] ||
		fail "${profile_path} has generator profile markers but is classified template-independent"
done < <(
	grep -rl 'profile:[^:][^:]*:\(start\|end\)' \
		Makefile railway.toml build cmd env internal test .github 2>/dev/null || true
)

grep -Fq -- '--cache-dir /root/.cache/trivy' Makefile ||
	fail "container-security must retain its persistent Trivy cache"
grep -Fq $'\tmake mod-tidy-check' scripts/ci/template-init-check.sh ||
	fail "template-init-check must use the manifest-only module gate"

if [[ -d migrations ]]; then
	for up in migrations/*.up.sql; do
		[[ -e "${up}" ]] || continue
		down="${up%.up.sql}.down.sql"
		[[ -f "${down}" ]] || fail "${up} is missing paired ${down}"
	done
	for down in migrations/*.down.sql; do
		[[ -e "${down}" ]] || continue
		up="${down%.down.sql}.up.sql"
		[[ -f "${up}" ]] || fail "${down} is missing paired ${up}"
	done
fi

# Documentation and skills tell agents and humans which commands to run. A
# reference to a target that no longer exists sends them down a failing path,
# so every documented `make <target>` must resolve in this Makefile.
if [[ -f Makefile ]]; then
	# Kept portable to bash 3.2, which macOS still ships and which has no
	# associative arrays. The backtick lives in a variable so it never has to
	# survive quoting inside a process substitution.
	backtick="$(printf '\140')"
	# Inline-code spans that are entirely a make invocation, so prose such as
	# "do not make auth global" is ignored.
	make_reference_pattern="${backtick}make [a-z][a-z0-9-]*( [A-Z_][A-Z0-9_]*=[^${backtick}]*)?${backtick}"
	make_targets="$(grep -oE '^[a-z][a-z0-9-]*:' Makefile | tr -d ':' | sort -u)"

	# Prose is shared across database profiles; the Makefile is not. DATABASE=none
	# removes the targets below along with the PostgreSQL profile, so a generated
	# service still reads instructions that name them. That is not a dangling
	# reference — the command exists in the profile that owns it.
	#
	# The tolerance is conditional on the owning profile being absent, so a rename
	# or deletion upstream still fails here, where the profile is present.
	database_profile_targets="$(printf '%s\n' \
		bench-db \
		bench-db-baseline \
		bench-db-compare \
		compose-down \
		compose-up \
		migration-validate \
		sqlc-generate)"
	database_profile_present=true
	[[ -d internal/infra/postgres ]] || database_profile_present=false

	documented_targets="$(mktemp)"
	while IFS= read -r doc; do
		[[ -f "${doc}" ]] || continue
		# A file without any make reference is the common case, so a non-zero
		# grep exit must not abort the scan under `set -e -o pipefail`.
		matches="$(grep -oE "${make_reference_pattern}" "${doc}" 2>/dev/null || true)"
		[[ -n "${matches}" ]] || continue
		printf '%s\n' "${matches}" |
			sed -E "s|^${backtick}make ([a-z][a-z0-9-]*).*|${doc}:\1|" >>"${documented_targets}"
		# specs/ records decisions as they were accepted at the time; it is a
		# historical archive, not live instruction, so renaming a target must
		# not require rewriting it.
	done < <(git ls-files -- '*.md' ':!:specs/**' | sort -u)

	while IFS= read -r documented; do
		[[ -n "${documented}" ]] || continue
		file="${documented%%:*}"
		target="${documented#*:}"
		if printf '%s\n' "${make_targets}" | grep -Fxq "${target}"; then
			continue
		fi
		if [[ "${database_profile_present}" == false ]] &&
			printf '%s\n' "${database_profile_targets}" | grep -Fxq "${target}"; then
			continue
		fi
		fail "${file} references 'make ${target}', which is not a Makefile target"
	done <"${documented_targets}"
	rm -f "${documented_targets}"
fi

if ((failed != 0)); then
	exit 1
fi

echo "project structure is current"
