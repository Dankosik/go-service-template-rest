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
	[[ -n "${file}" ]] && fail "${file} must use harness_test.go for shared package test helpers"
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
	if [[ "${base}" != "openapi.gen.go" &&
		"${base}" != "client.gen.go" &&
		"${file}" != internal/infra/postgres/sqlcgen/*.sql.go &&
		"${file}" != internal/gen/proto/* &&
		! "${base}" =~ ^[a-z0-9]+(_[a-z0-9]+)*(_test)?\.go$ ]]; then
		fail "${file} must use lowercase snake_case Go file naming"
	fi
done < <(find cmd internal test -type f -name '*.go' -print 2>/dev/null)

# Every directory under cmd/ is a binary, except cmd/internal/, which holds the
# composition support more than one of them shares. Go's own internal rule is
# what keeps that support out of reach of feature packages.
for command_dir in cmd/*; do
	[[ -d "${command_dir}" ]] || continue
	[[ "${command_dir}" == "cmd/internal" ]] && continue
	[[ -f "${command_dir}/main.go" ]] || fail "${command_dir}/ must contain main.go"
done

integration_test_count=0
for integration_test in test/*_test.go; do
	[[ -e "${integration_test}" ]] || continue
	integration_test_count=$((integration_test_count + 1))
	[[ "${integration_test}" == *_integration_test.go ]] ||
		fail "${integration_test} must use the *_integration_test.go suffix"
done

if ((integration_test_count > 0)); then
	if go list ./test >/dev/null 2>&1; then
		fail "test/ must expose no package without the integration build tag"
	fi

	integration_package="$(
		go list -tags=integration \
			-f '{{.Name}}|{{len .TestGoFiles}}|{{len .XTestGoFiles}}' \
			./test 2>/dev/null
	)" || {
		fail "test/ must be loadable with the integration build tag"
		integration_package=""
	}
	[[ "${integration_package}" == "integration|0|${integration_test_count}" ]] ||
		fail "test/ must contain only external integration tests; got ${integration_package:-no package}"
fi

bootstrap_integration_test="cmd/service/internal/bootstrap/startup_idempotency_integration_test.go"
while IFS= read -r integration_test; do
	[[ -n "${integration_test}" ]] || continue
	[[ "${integration_test}" == "${bootstrap_integration_test}" ]] ||
		fail "${integration_test} is not the sole approved package-local integration test"
done < <(find cmd -type f -name '*_integration_test.go' -print 2>/dev/null)
if [[ -e "${bootstrap_integration_test}" ]]; then
	grep -qx '//go:build integration' "${bootstrap_integration_test}" ||
		fail "${bootstrap_integration_test} must declare the integration build tag"
	grep -qx 'package bootstrap' "${bootstrap_integration_test}" ||
		fail "${bootstrap_integration_test} must remain in package bootstrap"
fi

for skill_dir in .agents/skills/*; do
	[[ -d "${skill_dir}" ]] || continue
	[[ -f "${skill_dir}/SKILL.md" ]] || fail "${skill_dir}/ must contain SKILL.md"
done

if [[ -d api/proto && -z "$(find api/proto -type f -name '*.proto' -print -quit)" ]]; then
	fail "api/proto/ must not exist before the first owned .proto contract"
fi
if [[ -d migrations && -z "$(find migrations -mindepth 1 -maxdepth 1 -type f -name '*.sql' -print -quit)" ]]; then
	fail "migrations/ must not exist before the first owned migration"
fi
if [[ -d internal/infra/postgres/queries &&
	-z "$(find internal/infra/postgres/queries -type f -name '*.sql' -print -quit)" ]]; then
	fail "internal/infra/postgres/queries/ must not exist before the first owned query"
fi
if [[ -d internal/infra/postgres/sqlcgen &&
	-z "$(find internal/infra/postgres/sqlcgen -type f -name '*.go' -print -quit)" ]]; then
	fail "internal/infra/postgres/sqlcgen/ must not exist without generated Go output"
fi

if [[ -d migrations ]]; then
	while IFS= read -r nested; do
		[[ -n "${nested}" ]] && fail "${nested} is nested; migrations must be flat"
	done < <(find migrations -mindepth 1 -type d -print)
	while IFS= read -r migration; do
		[[ -n "${migration}" ]] || continue
		name="${migration##*/}"
		[[ "${name}" =~ ^[0-9]{6}_[a-z][a-z0-9]*(_[a-z0-9]+)*\.sql$ ]] ||
			fail "${migration} must match NNNNNN_lower_snake.sql"
	done < <(find migrations -mindepth 1 -maxdepth 1 -type f -print)
fi

if [[ -d integrations ]]; then
	while IFS= read -r rec; do
		[[ -n "${rec}" ]] || continue
		name="$(sed -n 's/^name = "\(.*\)"/\1/p' "${rec}" | head -n1)"
		transport="$(sed -n 's/^transport = "\(.*\)"/\1/p' "${rec}" | head -n1)"
		contract="$(sed -n 's/^contract = "\(.*\)"/\1/p' "${rec}" | head -n1)"
		[[ "${rec}" == "integrations/${name}.toml" ]] || fail "${rec} name does not match its path"
		[[ -f "${contract}" ]] || fail "${rec} contract ${contract} is missing"
		[[ -d "internal/infra/${name}" ]] || fail "missing adapter internal/infra/${name}"
		[[ -f "internal/config/${name}_integration_config.go" ]] || fail "missing config for ${name}"
		[[ -f "docs/integrations/${name}.md" ]] || fail "missing docs/integrations/${name}.md"
		if [[ "${transport}" == "http" ]]; then
			[[ "${contract}" == "api/external/${name}/openapi.yaml" ]] || fail "${rec} HTTP contract path is not canonical"
			[[ -f "internal/infra/${name}/internal/openapi/doc.go" ]] || fail "missing HTTP generator for ${name}"
		elif [[ "${transport}" == "grpc" ]]; then
			case "${contract}" in
			api/proto/external/"${name}"/*.proto) ;;
			*) fail "${rec} gRPC contract path is not canonical" ;;
			esac
		else
			fail "${rec} transport must be http or grpc"
		fi
	done < <(find integrations -maxdepth 1 -type f -name '*.toml' -print)
fi

if [[ -d api/external ]]; then
	while IFS= read -r contract; do
		[[ -n "${contract}" ]] || continue
		name="$(basename "$(dirname "${contract}")")"
		[[ -f "integrations/${name}.toml" ]] || fail "orphan external OpenAPI contract ${contract}"
	done < <(find api/external -mindepth 2 -maxdepth 2 -type f -name openapi.yaml -print)
fi

if [[ -d api/proto/external ]]; then
	while IFS= read -r contract; do
		[[ -n "${contract}" ]] || continue
		name="$(basename "$(dirname "${contract}")")"
		[[ -f "integrations/${name}.toml" ]] || fail "orphan external Protobuf contract ${contract}"
	done < <(find api/proto/external -type f -name '*.proto' -print)
fi

env_helper_ok=0
if [[ -f scripts/integration-init.sh ]]; then
	if grep -q '^env_entry_present()' scripts/integration-init.sh; then
		calls="$(grep -c 'env_entry_present "' scripts/integration-init.sh || true)"
		[[ "${calls}" == "2" ]] || fail "scripts/integration-init.sh must call env_entry_present exactly twice"
		other="$(grep -nE '(^|[^[:alnum:]_.])\.env($|[^[:alnum:]_.])' scripts/integration-init.sh | grep -v env_entry_present | grep -v reason_env | grep -v 'move or preserve' | grep -v '\.env.example' || true)"
		if [[ -n "${other}" ]]; then
			while IFS= read -r line; do
				[[ "${line}" == *'"${reason_env}" ".env"'* ]] && continue
				[[ "${line}" == *'name .env'* ]] && continue
				fail "scripts/integration-init.sh has an extra .env consumer: ${line}"
			done <<<"${other}"
		fi
		env_helper_ok=1
	else
		fail "scripts/integration-init.sh must define env_entry_present"
	fi
	[[ "${env_helper_ok}" -eq 1 ]]
fi

if ((failed != 0)); then
	exit 1
fi

echo "project structure is current"
