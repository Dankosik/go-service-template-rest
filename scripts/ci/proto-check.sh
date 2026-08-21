#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TEMP_ROOT="$(mktemp -d -t proto-check.XXXXXX)"
trap 'rm -rf "${TEMP_ROOT}"' EXIT
fixture="${TEMP_ROOT}/repo"
mkdir -p "${fixture}"
(
	cd "${ROOT_DIR}"
	git ls-files --cached --others --exclude-standard |
		while IFS= read -r file; do
			[[ -f "${file}" ]] && printf '%s\n' "${file}"
		done |
		tar -cf - -T -
) | tar -xf - -C "${fixture}"
git -C "${fixture}" init -q
git -C "${fixture}" add .
git -C "${fixture}" -c user.email=proto-check@example.com -c user.name=proto-check commit -qm baseline

bash "${ROOT_DIR}/scripts/run-buf.sh" --version >/dev/null
export BUF_BIN="${ROOT_DIR}/.cache/tools/buf/1.72.0/buf"

run_proto() (
	cd "${fixture}"
	BASE_REF=HEAD bash ./scripts/proto.sh "$@"
)

expect_failure() {
	local name="$1"
	shift
	if "$@" >"${TEMP_ROOT}/${name}.log" 2>&1; then
		echo "protobuf self-test: ${name} unexpectedly succeeded" >&2
		exit 1
	fi
}

run_proto check >/dev/null

generated="$(find "${fixture}/examples/grpc-reference-service/internal/gen/proto" -name '*.pb.go' -print -quit)"
printf '\n// deliberate drift\n' >>"${generated}"
expect_failure generated-drift run_proto drift
git -C "${fixture}" checkout -q -- "${generated#"${fixture}"/}"

proto="${fixture}/examples/grpc-reference-service/api/proto/reference/v1/echo.proto"
sed 's/service EchoService {/service   EchoService{/' "${proto}" >"${proto}.new"
mv "${proto}.new" "${proto}"
expect_failure unformatted-source run_proto format-check
git -C "${fixture}" checkout -q -- "${proto#"${fixture}"/}"

printf '\nthis is not protobuf;\n' >>"${proto}"
expect_failure malformed-schema run_proto lint

echo "protobuf contract self-test passed"
