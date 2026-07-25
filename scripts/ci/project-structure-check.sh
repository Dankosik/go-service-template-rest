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
		printf '%s\n' "${make_targets}" | grep -Fxq "${target}" ||
			fail "${file} references 'make ${target}', which is not a Makefile target"
	done <"${documented_targets}"
	rm -f "${documented_targets}"
fi

if ((failed != 0)); then
	exit 1
fi

echo "project structure is current"
