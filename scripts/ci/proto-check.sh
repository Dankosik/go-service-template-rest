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
		tar -cf - -T -
) | tar -xf - -C "${fixture}"

git -C "${fixture}" init -q
git -C "${fixture}" add .
git -C "${fixture}" \
	-c user.email=proto-check@example.com \
	-c user.name=proto-check \
	commit -qm "current fixture"

# Reuse the exact pinned Buf binary after the repository runner has installed
# and verified it. The fixture still resolves both Go generators from its own
# copied tools module.
bash "${ROOT_DIR}/scripts/run-buf.sh" --version >/dev/null
BUF_BIN="${ROOT_DIR}/.cache/tools/buf/1.72.0/buf"

run_proto() {
	(
		cd "${fixture}"
		PATH="${PATH}" BUF_BIN="${BUF_BIN}" bash ./scripts/proto.sh "$@"
	)
}

expect_failure() {
	local name="$1"
	shift
	if "$@" >"${TEMP_ROOT}/${name}.log" 2>&1; then
		echo "protobuf self-test: ${name} unexpectedly succeeded" >&2
		exit 1
	fi
}

# The copied current tree is the valid baseline for this isolated harness.
# Dedicated cases below prove that new legacy syntax is rejected relative to
# an earlier commit.
BASE_REF=HEAD run_proto check >/dev/null

if grep -R -Fq 'default_api_level' \
	"${fixture}/buf.gen.yaml" \
	"${fixture}/examples/grpc-reference-service/buf.gen.yaml"; then
	echo "protobuf self-test: generator config overrides schema-owned Go API level" >&2
	exit 1
fi

generated="$(
	find "${fixture}/examples/grpc-reference-service/internal/gen/proto" \
		-type f -name '*.pb.go' -print -quit
)"
printf '\n// deliberate drift\n' >>"${generated}"
expect_failure generated-drift run_proto drift
git -C "${fixture}" checkout -q -- "${generated#"${fixture}"/}"

proto="${fixture}/examples/grpc-reference-service/api/proto/reference/v1/echo.proto"
awk '
	/^service EchoService \{$/ {
		print "service   EchoService{"
		next
	}
	{ print }
' "${proto}" >"${proto}.new"
mv "${proto}.new" "${proto}"
cp "${proto}" "${TEMP_ROOT}/unformatted.proto"
expect_failure unformatted-source run_proto format-check
if ! cmp -s "${TEMP_ROOT}/unformatted.proto" "${proto}"; then
	echo "protobuf self-test: format check mutated source" >&2
	exit 1
fi
git -C "${fixture}" checkout -q -- "${proto#"${fixture}"/}"

grep -Fv '// EchoService demonstrates unary and every streaming RPC cardinality.' \
	"${proto}" >"${proto}.new"
mv "${proto}.new" "${proto}"
expect_failure undocumented-contract run_proto lint
git -C "${fixture}" checkout -q -- "${proto#"${fixture}"/}"

printf '\nthis is not protobuf;\n' >>"${proto}"
expect_failure malformed-schema run_proto lint
git -C "${fixture}" checkout -q -- "${proto#"${fixture}"/}"

grep -Fv 'features.(pb.go).api_level = API_OPAQUE' "${proto}" >"${proto}.new"
mv "${proto}.new" "${proto}"
expect_failure missing-opaque-policy run_proto lint
git -C "${fixture}" checkout -q -- "${proto#"${fixture}"/}"

new_proto3="${fixture}/examples/grpc-reference-service/api/proto/legacy/v1/legacy.proto"
mkdir -p "$(dirname "${new_proto3}")"
cat >"${new_proto3}" <<'EOF'
syntax = "proto3";

package legacy.v1;

option go_package = "github.com/example/go-service-template-rest/examples/grpc-reference-service/internal/gen/proto/legacy/v1;legacyv1";

// LegacyMessage represents a retained proto3 contract.
message LegacyMessage {
  // value is the retained payload.
  string value = 1;
}
EOF
expect_failure legacy-missing-base \
	env -u BASE_REF BUF_BIN="${BUF_BIN}" bash -c \
	"cd '${fixture}' && bash ./scripts/proto.sh lint"
expect_failure legacy-invalid-base \
	env BASE_REF=definitely-not-a-ref BUF_BIN="${BUF_BIN}" bash -c \
	"cd '${fixture}' && bash ./scripts/proto.sh lint"
expect_failure new-proto3 \
	env BASE_REF=HEAD BUF_BIN="${BUF_BIN}" bash -c \
	"cd '${fixture}' && bash ./scripts/proto.sh lint"
git -C "${fixture}" add "${new_proto3#"${fixture}"/}"
git -C "${fixture}" \
	-c user.email=proto-check@example.com \
	-c user.name=proto-check \
	commit -qm "retained legacy protobuf contract"
BASE_REF=HEAD run_proto lint >/dev/null

edition_2024_proto="${fixture}/examples/grpc-reference-service/api/proto/future/v1/future.proto"
mkdir -p "$(dirname "${edition_2024_proto}")"
cat >"${edition_2024_proto}" <<'EOF'
edition = "2024";

package future.v1;

import "google/protobuf/go_features.proto";

option features.(pb.go).api_level = API_OPAQUE;
option go_package = "github.com/example/go-service-template-rest/examples/grpc-reference-service/internal/gen/proto/future/v1;futurev1";

// FutureMessage represents an explicitly accepted Edition 2024 contract.
message FutureMessage {
  // value is the future payload.
  string value = 1;
}
EOF
expect_failure unaccepted-edition-2024 \
	env BASE_REF=HEAD BUF_BIN="${BUF_BIN}" bash -c \
	"cd '${fixture}' && bash ./scripts/proto.sh lint"
rm -rf -- "${fixture}/examples/grpc-reference-service/api/proto/future"

absent_output="$(
	cd "${fixture}"
	BASE_REF=HEAD BUF_BIN="${BUF_BIN}" bash ./scripts/proto.sh breaking
)"
if [[ "${absent_output}" != *"breaking check not applicable"* ]]; then
	echo "protobuf self-test: absent base contract was not reported as not applicable" >&2
	exit 1
fi

expect_failure invalid-base \
	env BASE_REF=definitely-not-a-ref BUF_BIN="${BUF_BIN}" bash -c \
	"cd '${fixture}' && bash ./scripts/proto.sh breaking"

production_proto="${fixture}/api/proto/acme/example/v1/example.proto"
mkdir -p "$(dirname "${production_proto}")"
cat >"${production_proto}" <<'EOF'
edition = "2023";

package acme.example.v1;

import "google/protobuf/go_features.proto";

option go_package = "github.com/example/go-service-template-rest/internal/gen/proto/acme/example/v1;examplev1";
option features.(pb.go).api_level = API_OPAQUE;

message Example {
  string value = 1;
}
EOF
git -C "${fixture}" add api/proto
git -C "${fixture}" \
	-c user.email=proto-check@example.com \
	-c user.name=proto-check \
	commit -qm "base protobuf contract"

python_replacement="${TEMP_ROOT}/replace.awk"
cat >"${python_replacement}" <<'EOF'
{ if ($0 !~ /string value = 1;/) print }
EOF
awk -f "${python_replacement}" "${production_proto}" >"${production_proto}.new"
mv "${production_proto}.new" "${production_proto}"
expect_failure breaking-field-removal \
	env BASE_REF=HEAD BUF_BIN="${BUF_BIN}" bash -c \
	"cd '${fixture}' && bash ./scripts/proto.sh breaking"

echo "protobuf contract self-test passed"
