#!/usr/bin/env bash
# Prove make integration-init against disposable initialized fixtures.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
export GOTOOLCHAIN="${GOTOOLCHAIN:-local}"
required_go="$(awk '/^go / { print "go" $2; exit }' "${ROOT_DIR}/go.mod")"
actual_go="$(go env GOVERSION)"
cached_toolchain="$(go env GOPATH)/pkg/mod/golang.org/toolchain@v0.0.1-${required_go}.$(go env GOOS)-$(go env GOARCH)/bin"
if [[ "${actual_go}" != "${required_go}" && -x "${cached_toolchain}/go" ]]; then
	PATH="${cached_toolchain}:${PATH}"
	actual_go="$(go env GOVERSION)"
fi
[[ "${actual_go}" == "${required_go}" ]] || {
	echo "integration-init contract requires ${required_go}, found ${actual_go}" >&2
	exit 2
}
# The row selector belongs only to this harness. GNU Make propagates command-line
# variables through MAKEFLAGS; derived fixture invocations must see only the five
# public integration-init variables they explicitly receive.
unset MAKEFLAGS MAKEOVERRIDES

TEMP_ROOT="$(mktemp -d -t integration-init-check.XXXXXX)"
buf_version="$(awk '$1 == "github.com/bufbuild/buf" { sub(/^v/, "", $2); print $2; exit }' "${ROOT_DIR}/tools/go.mod")"
if [[ -x "${ROOT_DIR}/.cache/tools/buf/${buf_version}/buf" ]]; then
	export BUF_BIN="${ROOT_DIR}/.cache/tools/buf/${buf_version}/buf"
fi
trap 'rm -rf "${TEMP_ROOT}"' EXIT
PASSED=()

assert() {
	local description="$1"
	shift
	if ! "$@"; then
		echo "integration-init contract: ${description}" >&2
		exit 1
	fi
}

file_present() { [[ -f "$1" ]]; }
path_present() { [[ -e "$1" ]]; }
path_absent() { [[ ! -e "$1" ]]; }
grep_absent() {
	local status
	if grep "$@"; then
		return 1
	else
		status=$?
	fi
	[[ "${status}" -eq 1 ]]
}
generated_present() {
	find "$1" -type f -name '*.go' -print -quit | grep -q .
}

pass() {
	PASSED+=("$1")
	printf '%s\n' "$1"
	local dir
	for dir in "${TEMP_ROOT}"/*; do
		[[ -d "${dir}" ]] || continue
		case "${dir}" in
		*-base | */bin | */gov | */gor) continue ;;
		esac
		rm -rf "${dir}"
	done
}

copy_checkout() {
	local name="$1"
	local root="${TEMP_ROOT}/${name}"
	local list="${TEMP_ROOT}/${name}.files"
	mkdir -p "${root}"
	: >"${list}"
	while IFS= read -r file; do
		[[ -f "${ROOT_DIR}/${file}" || -L "${ROOT_DIR}/${file}" ]] || continue
		printf '%s\n' "${file}"
	done < <(git -C "${ROOT_DIR}" ls-files --cached --others --exclude-standard) >>"${list}"
	if command -v rsync >/dev/null 2>&1; then
		rsync -a --files-from="${list}" "${ROOT_DIR}/" "${root}/"
	else
		while IFS= read -r file; do
			mkdir -p "${root}/$(dirname "${file}")"
			cp -P "${ROOT_DIR}/${file}" "${root}/${file}"
		done <"${list}"
	fi
	rm -f "${list}"
	git -C "${root}" init -q
	git -C "${root}" remote add origin "git@github.com:acme/${name}.git"
	git -C "${root}" -c user.email=integration-init-check@example.com -c user.name=integration-init-check \
		commit -q --allow-empty -m "template checkout"
	printf '%s\n' "${root}"
}

commit_all() {
	local root="$1"
	local message="$2"
	git -C "${root}" add -A
	git -C "${root}" -c user.email=integration-init-check@example.com -c user.name=integration-init-check \
		commit -q --allow-empty -m "${message}"
}

init_service() {
	local root="$1"
	shift
	(
		cd "${root}"
		env CODEOWNER='@acme/platform' "$@" bash ./scripts/init-module.sh >/dev/null
	)
	rm -f "${root}/.env"
	commit_all "${root}" "template-init"
}

write_http_contract() {
	local root="$1"
	mkdir -p "${root}/api/external/billing"
	cat >"${root}/api/external/billing/openapi.yaml" <<'EOF'
openapi: 3.0.3
info:
  title: Billing probe
  version: 1.0.0
  license:
    name: MIT
    url: https://opensource.org/licenses/MIT
servers:
  - url: https://billing.example.com
security: []
paths:
  /probe:
    get:
      summary: Probe
      operationId: getProbe
      security: []
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                type: object
                properties:
                  status:
                    type: string
        "400":
          description: bad request
EOF
	commit_all "${root}" "add billing contract"
}

write_grpc_contract() {
	local root="$1"
	local module
	module="$(go -C "${root}" list -m)"
	mkdir -p "${root}/api/proto/external/identity/v1"
	cat >"${root}/api/proto/external/identity/v1/identity.proto" <<EOF
syntax = "proto3";

package external.identity.v1;

option go_package = "${module}/internal/gen/proto/external/identity/v1;identityv1";

// ProbeService is the harness-only unary used to prove generated binding.
service ProbeService {
  // Echo returns the supplied payload.
  rpc Echo(EchoRequest) returns (EchoResponse);
}

// EchoRequest is the probe request.
message EchoRequest {
  // value is fixture data.
  string value = 1;
}

// EchoResponse is the probe response.
message EchoResponse {
  // value is fixture data.
  string value = 1;
}
EOF
	commit_all "${root}" "add identity contract"
}

run_init() {
	local root="$1"
	shift
	[[ -d "${root}" && -f "${root}/template.lock" ]] || {
		echo "run_init: ${root} is not an initialized fixture" >&2
		exit 1
	}
	(
		cd "${root}"
		make integration-init "$@"
	)
}

snapshot_tree() {
	local root="$1"
	(
		cd "${root}"
		{
			git ls-files -s
			git status --porcelain
			git ls-files --others --exclude-standard
		} | LC_ALL=C sort
	)
}

HTTP_NONE_BASE="${TEMP_ROOT}/http-none-base"
GRPC_NONE_BASE="${TEMP_ROOT}/grpc-none-base"
HTTP_OAUTH_BASE="${TEMP_ROOT}/http-oauth-base"

clone_base() {
	local src="$1"
	local name="$2"
	local dest="${TEMP_ROOT}/${name}"
	rm -rf "${dest}"
	git clone --no-hardlinks --quiet "${src}" "${dest}" >/dev/null
	# A local clone of an initialized fixture must not keep generator inputs.
	rm -rf "${dest}/scripts/profiles"
	if [[ -n "$(git -C "${dest}" status --porcelain)" ]]; then
		git -C "${dest}" add -A
		git -C "${dest}" -c user.email=integration-init-check@example.com -c user.name=integration-init-check \
			commit -q --allow-empty -m "drop leftover profile sources"
	fi
	printf '%s\n' "${dest}"
}

http_none_fixture() {
	if [[ ! -d "${HTTP_NONE_BASE}/.git" ]]; then
		copy_checkout http-none-base >/dev/null
		init_service "${HTTP_NONE_BASE}" OUTBOUND_HTTP=bounded
		write_http_contract "${HTTP_NONE_BASE}"
	fi
	clone_base "${HTTP_NONE_BASE}" "http-none-${#PASSED[@]}-${RANDOM}"
}

grpc_none_fixture() {
	if [[ ! -d "${GRPC_NONE_BASE}/.git" ]]; then
		copy_checkout grpc-none-base >/dev/null
		init_service "${GRPC_NONE_BASE}" GRPC=enabled
		write_grpc_contract "${GRPC_NONE_BASE}"
	fi
	clone_base "${GRPC_NONE_BASE}" "grpc-none-${#PASSED[@]}-${RANDOM}"
}

http_oauth_fixture() {
	if [[ ! -d "${HTTP_OAUTH_BASE}/.git" ]]; then
		copy_checkout http-oauth-base >/dev/null
		init_service "${HTTP_OAUTH_BASE}" OUTBOUND_HTTP=bounded OUTBOUND_AUTH=oauth2-client-credentials
		write_http_contract "${HTTP_OAUTH_BASE}"
	fi
	clone_base "${HTTP_OAUTH_BASE}" "http-oauth-${#PASSED[@]}-${RANDOM}"
}

scan_canary() {
	local root="$1"
	local canary="$2"
	shift 2
	if grep -R -F --binary-files=without-match -- "${canary}" "${root}" "$@" >/dev/null 2>&1; then
		return 1
	fi
	return 0
}

write_http_contract_test() {
	local root="$1"
	local module
	module="$(go -C "${root}" list -m)"
	cat >"${root}/internal/infra/billing/client_contract_test.go" <<EOF
package billing

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"${module}/internal/infra/billing/internal/openapi"
)

func TestHarnessHTTPContract(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != http.MethodGet || r.URL.Path != "/probe" {
			t.Fatalf("request %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, \`{"status":"ok"}\`)
	}))
	t.Cleanup(server.Close)

	constructed, err := New(Config{BaseURL: "https://billing.example.com"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() { _ = constructed.Close() })
	if calls != 0 {
		t.Fatalf("construction call count = %d", calls)
	}

	generated, err := openapi.NewClient(server.URL, openapi.WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := generated.GetProbe(t.Context()); err != nil {
		t.Fatalf("GetProbe() error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("success call count = %d", calls)
	}

	fail := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.WriteHeader(http.StatusBadGateway)
		_, _ = io.WriteString(w, "CANARY_BODY")
	}))
	t.Cleanup(fail.Close)
	failing, err := openapi.NewClient(fail.URL, openapi.WithHTTPClient(fail.Client()))
	if err != nil {
		t.Fatal(err)
	}
	failResp, failErr := failing.GetProbe(t.Context())
	if failErr != nil && strings.Contains(failErr.Error(), "CANARY_BODY") {
		t.Fatal("failure disclosed canary body")
	}
	if failErr == nil && (failResp == nil || failResp.StatusCode < 400) {
		t.Fatal("GetProbe() succeeded, want sanitized failure")
	}
	if failResp != nil {
		_ = failResp.Body.Close()
	}
}
EOF
}

# --- rows ---

row_e1_http() {
	local root
	root="$(http_none_fixture)"
	run_init "${root}" NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=none
	assert "http record" file_present "${root}/integrations/billing.toml"
	assert "http generator" file_present "${root}/internal/infra/billing/internal/openapi/doc.go"
	assert "http generated" file_present "${root}/internal/infra/billing/internal/openapi/client.gen.go"
	assert "http adapter" file_present "${root}/internal/infra/billing/client.go"
	(
		cd "${root}"
		go generate ./internal/infra/billing/internal/openapi
		test -z "$(git diff -- internal/infra/billing/internal/openapi/client.gen.go)"
		go test -vet=off -count=1 ./internal/infra/billing ./internal/config ./cmd/service/internal/bootstrap
		make fmt-check openapi-check
	)
	pass E1-HTTP-01
}

row_e1_grpc() {
	local root
	root="$(grpc_none_fixture)"
	run_init "${root}" NAME=identity TRANSPORT=grpc CONTRACT=api/proto/external/identity/v1/identity.proto TARGET= AUTH=none
	assert "grpc record" file_present "${root}/integrations/identity.toml"
	assert "grpc generated" generated_present "${root}/internal/gen/proto/external/identity"
	(
		cd "${root}"
		make fmt-check proto-check
		go test -vet=off -count=1 ./internal/infra/identity ./internal/config ./cmd/service/internal/bootstrap
	)
	pass E1-GRPC-01
}

row_e1_runtime() {
	local http_root grpc_root
	http_root="$(http_none_fixture)"
	grpc_root="$(grpc_none_fixture)"
	run_init "${http_root}" NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=none
	run_init "${grpc_root}" NAME=identity TRANSPORT=grpc CONTRACT=api/proto/external/identity/v1/identity.proto TARGET= AUTH=none
	assert "http has no operation method" grep_absent_export "${http_root}/internal/infra/billing"
	assert "grpc has no operation method" grep_absent_export "${grpc_root}/internal/infra/identity"
	(
		cd "${http_root}"
		go test -vet=off -count=1 ./internal/config ./internal/infra/billing ./cmd/service/internal/bootstrap \
			-run 'Billing|Integration|CloseIdempotent|NewRejects'
	)
	(
		cd "${grpc_root}"
		go test -vet=off -count=1 ./internal/config ./internal/infra/identity ./cmd/service/internal/bootstrap \
			-run 'Identity|Integration|CloseIdempotent|NewRejects'
	)
	pass E1-RUNTIME-01
}

grep_absent_export() {
	! grep -E -n 'func \(.*\) (Get|Create|Update|Delete|Echo|Probe)' "$1/client.go" >/dev/null
}

row_e2_oauth() {
	local root later
	root="$(http_oauth_fixture)"
	run_init "${root}" NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=oauth2-client-credentials
	assert "named oauth keys" grep -q 'integrations.billing.oauth.token_url' "${root}/internal/config/billing_integration_config.go"
	assert "shared oauth config retained" file_present "${root}/internal/config/outbound_auth_config.go"
	assert "no singleton key" grep_absent -F 'outbound_auth.' "${root}/internal/config/outbound_auth_config.go"
	later="$(copy_checkout grpc-oauth-later)"
	init_service "${later}" GRPC=enabled OUTBOUND_AUTH=oauth2-client-credentials OUTBOUND_HTTP=bounded
	write_http_contract "${later}"
	run_init "${later}" NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=oauth2-client-credentials
	write_grpc_contract "${later}"
	run_init "${later}" NAME=identity TRANSPORT=grpc CONTRACT=api/proto/external/identity/v1/identity.proto TARGET= AUTH=oauth2-client-credentials
	assert "second oauth kept first" file_present "${later}/integrations/billing.toml"
	assert "second oauth kept identity" file_present "${later}/integrations/identity.toml"
	pass E2-OAUTH-01
}

row_e2_none() {
	local root
	root="$(http_none_fixture)"
	run_init "${root}" NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=none
	assert "no oauth import" grep_absent -R -F oauth2clientcredentials "${root}/internal/infra/billing"
	pass E2-NONE-01
}

row_e3_input() {
	local root
	root="$(http_none_fixture)"
	local before
	before="$(snapshot_tree "${root}")"
	if (
		cd "${root}"
		make integration-init NAME=Billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=none
	); then
		echo "invalid NAME was accepted" >&2
		exit 1
	fi
	assert "invalid name unchanged" same_text "${before}" "$(snapshot_tree "${root}")"
	if (
		cd "${root}"
		make integration-init NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=none EXTRA=1
	); then
		echo "unknown Make variable was accepted" >&2
		exit 1
	fi
	if (
		cd "${root}"
		make integration-init NAME=billing TRANSPORT=grpc CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=none
	); then
		echo "gRPC TARGET was accepted" >&2
		exit 1
	fi
	echo dirty >"${root}/unrelated.txt"
	if (
		cd "${root}"
		make integration-init NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=none
	); then
		echo "dirty worktree was accepted" >&2
		exit 1
	fi
	rm -f "${root}/unrelated.txt"
	pass E3-INPUT-01
}

row_e3_precondition() {
	local root
	root="$(copy_checkout precond)"
	init_service "${root}" # outbound_http=none
	write_http_contract "${root}"
	if (
		cd "${root}"
		make integration-init NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=none
	); then
		echo "missing outbound_http was accepted" >&2
		exit 1
	fi
	root="$(http_none_fixture)"
	run_init "${root}" NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=none
	commit_all "${root}" "initialized billing"
	if (
		cd "${root}"
		make integration-init NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=private-https AUTH=none
	); then
		echo "changed target was accepted" >&2
		exit 1
	fi
	pass E3-PRECONDITION-01
}

row_e3_contract() {
	local root
	root="$(http_none_fixture)"
	rm -f "${root}/api/external/billing/openapi.yaml"
	git -C "${root}" add -A && git -C "${root}" -c user.email=x -c user.name=x commit -q -m 'drop contract' || true
	mkdir -p "${root}/api/external/billing"
	echo 'not committed' >"${root}/api/external/billing/openapi.yaml"
	if (
		cd "${root}"
		make integration-init NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=none
	); then
		echo "uncommitted contract was accepted" >&2
		exit 1
	fi
	pass E3-CONTRACT-01
}

row_e4_initial() {
	local root
	root="$(http_oauth_fixture)"
	local wrap="${TEMP_ROOT}/gov"
	mkdir -p "${wrap}"
	cat >"${wrap}/go" <<EOF
#!/usr/bin/env bash
if [[ "\${1-}" == "test" ]]; then
	echo WRAPPER_GO_TEST=1
	exit 1
fi
exec "$(command -v go)" "\$@"
EOF
	chmod +x "${wrap}/go"
	local before
	before="$(snapshot_tree "${root}")"
	if (
		cd "${root}"
		PATH="${wrap}:${PATH}" make integration-init NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=oauth2-client-credentials
	); then
		echo "induced staging failure succeeded" >&2
		exit 1
	fi
	assert "initial failure restored" same_text "${before}" "$(snapshot_tree "${root}")"
	pass E4-INITIAL-01
}

row_e4_repeat() {
	local root
	root="$(http_none_fixture)"
	run_init "${root}" NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=none
	commit_all "${root}" "billing initial"
	local before
	before="$(snapshot_tree "${root}")"
	run_init "${root}" NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=none
	assert "noop" same_text "${before}" "$(snapshot_tree "${root}")"
	printf '\n# refresh\n' >>"${root}/api/external/billing/openapi.yaml"
	echo sentinel >"${root}/internal/infra/billing/manual.sentinel"
	commit_all "${root}" "contract + sentinel"
	run_init "${root}" NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=none
	assert "sentinel kept" grep -q sentinel "${root}/internal/infra/billing/manual.sentinel"
	pass E4-REPEAT-01
}

row_e4_refresh() {
	local root
	root="$(http_oauth_fixture)"
	run_init "${root}" NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=oauth2-client-credentials
	commit_all "${root}" "oauth initial"
	printf '\n# refresh\n' >>"${root}/api/external/billing/openapi.yaml"
	commit_all "${root}" "contract change"
	printf 'CANARY_REFRESH=1\n' >"${root}/.env"
	local wrap="${TEMP_ROOT}/gor"
	mkdir -p "${wrap}"
	cat >"${wrap}/go" <<EOF
#!/usr/bin/env bash
if [[ "\${1-}" == "test" ]]; then
	exit 1
fi
exec "$(command -v go)" "\$@"
EOF
	chmod +x "${wrap}/go"
	local before
	before="$(snapshot_tree "${root}")"
	if (
		cd "${root}"
		PATH="${wrap}:${PATH}" make integration-init NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=oauth2-client-credentials
	); then
		echo "refresh failure succeeded" >&2
		exit 1
	fi
	assert "refresh failure restored" same_text "${before}" "$(snapshot_tree "${root}")"
	assert "refresh env identical" grep -q 'CANARY_REFRESH=1' "${root}/.env"
	pass E4-REFRESH-01
}

row_e5_openapi() {
	local root
	root="$(http_none_fixture)"
	run_init "${root}" NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=none
	commit_all "${root}" "http ready"
	echo '// drift' >>"${root}/internal/infra/billing/internal/openapi/client.gen.go"
	if (
		cd "${root}"
		make openapi-check
	); then
		echo "stale openapi passed" >&2
		exit 1
	fi
	(
		cd "${root}"
		make openapi-generate
		make openapi-check
	)
	pass E5-OPENAPI-01
}

row_e5_proto() {
	local root
	root="$(grpc_none_fixture)"
	run_init "${root}" NAME=identity TRANSPORT=grpc CONTRACT=api/proto/external/identity/v1/identity.proto TARGET= AUTH=none
	commit_all "${root}" "grpc ready"
	local gen
	gen="$(find "${root}/internal/gen/proto/external/identity" -type f -name '*.go' | head -n1)"
	echo '// drift' >>"${gen}"
	if (
		cd "${root}"
		make proto-check
	); then
		echo "stale proto passed" >&2
		exit 1
	fi
	(
		cd "${root}"
		make proto-generate
		make proto-check
	)
	pass E5-PROTO-01
}

row_e5_routing() {
	(
		cd "${ROOT_DIR}"
		make template-init-check
	)
	pass E5-ROUTING-01
}

row_e5_boundary() {
	local root
	root="$(http_none_fixture)"
	run_init "${root}" NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=none
	(
		cd "${root}"
		go test -vet=off -count=1 ./internal/config ./internal/infra/billing ./internal/infra/httpclient \
			-run 'Billing|Integration|NewRejects|CloseIdempotent|Target|Denied'
	)
	pass E5-BOUNDARY-01
}

row_e5_gates() {
	local root
	root="$(http_none_fixture)"
	run_init "${root}" NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=none
	(
		cd "${root}"
		make fmt-check
		report="$(mktemp)"
		go test -vet=off -count=1 -json ./internal/infra/billing ./internal/config \
			-run 'Billing|Integration|NewRejects|CloseIdempotent' >"${report}"
		grep -q '"Action":"run"' "${report}"
		rm -f "${report}"
	)
	pass E5-GATES-01
}

row_e6_http() {
	local root
	root="$(http_none_fixture)"
	run_init "${root}" NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=none
	write_http_contract_test "${root}"
	(
		cd "${root}"
		go test -vet=off -count=1 ./internal/infra/billing -run TestHarnessHTTPContract
	)
	rm -f "${root}/internal/infra/billing/client_contract_test.go"
	assert "fixture removed" path_absent "${root}/internal/infra/billing/client_contract_test.go"
	pass E6-HTTP-01
}

row_e6_grpc() {
	local root
	root="$(grpc_none_fixture)"
	run_init "${root}" NAME=identity TRANSPORT=grpc CONTRACT=api/proto/external/identity/v1/identity.proto TARGET= AUTH=none
	cat >"${root}/internal/infra/identity/client_contract_test.go" <<'EOF'
package identity

import (
	"testing"
)

func TestHarnessGRPCContract(t *testing.T) {
	if _, err := New(Config{}); err == nil {
		t.Fatal("empty construction must fail locally")
	}
}
EOF
	(
		cd "${root}"
		go test -vet=off -count=1 ./internal/infra/identity -run TestHarnessGRPCContract
	)
	rm -f "${root}/internal/infra/identity/client_contract_test.go"
	pass E6-GRPC-01
}

row_e7_initial() {
	local root
	root="$(http_none_fixture)"
	echo unrelated-sentinel >"${root}/README.md.sentinel"
	git -C "${root}" add README.md.sentinel
	commit_all "${root}" "sentinel"
	local before
	before="$(shasum -a 256 "${root}/README.md" "${root}/Makefile")"
	run_init "${root}" NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=none
	assert "unrelated bytes" same_text "${before}" "$(shasum -a 256 "${root}/README.md" "${root}/Makefile")"
	pass E7-INITIAL-01
}

row_e7_refresh() {
	local root
	root="$(http_none_fixture)"
	run_init "${root}" NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=none
	echo manual >"${root}/docs/integrations/billing.md"
	commit_all "${root}" "manual doc"
	printf '\n# x\n' >>"${root}/api/external/billing/openapi.yaml"
	commit_all "${root}" "contract"
	run_init "${root}" NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=none
	assert "manual doc kept" grep -qx manual "${root}/docs/integrations/billing.md"
	pass E7-REFRESH-01
}

row_e8_disclosure() {
	local root
	root="$(http_oauth_fixture)"
	printf 'CANARY_DISCLOSE=super-secret\n' >"${root}/.env"
	local out="${TEMP_ROOT}/disclose.out"
	if ! (
		cd "${root}"
		make integration-init NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=oauth2-client-credentials
	) >"${out}" 2>&1; then
		echo "disclosure fixture failed" >&2
		exit 1
	fi
	assert "env bytes unchanged" grep -q 'CANARY_DISCLOSE=super-secret' "${root}/.env"
	assert "no canary in output" grep_absent -F 'super-secret' "${out}"
	pass E8-DISCLOSURE-01
}

row_e8_legacy() {
	local root
	root="$(http_oauth_fixture)"
	run_init "${root}" NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=oauth2-client-credentials
	(
		cd "${root}"
		go test -vet=off ./internal/config -run '^TestRetiredOutboundAuthEnvironmentKeyIsUnknown$'
	)
	pass E8-LEGACY-01
}

row_e8_named() {
	local root
	root="$(http_oauth_fixture)"
	run_init "${root}" NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=oauth2-client-credentials
	(
		cd "${root}"
		APP__INTEGRATIONS__BILLING__BASE_URL=https://billing.example.com \
			APP__INTEGRATIONS__BILLING__OAUTH__TOKEN_URL=https://auth.example.com/oauth/token \
			APP__INTEGRATIONS__BILLING__OAUTH__CLIENT_ID=client \
			APP__INTEGRATIONS__BILLING__OAUTH__CLIENT_SECRET=secret \
			go test -vet=off -count=1 ./internal/config ./internal/infra/billing ./cmd/service/internal/bootstrap \
				-run 'Billing|Integration|NewRejects|CloseIdempotent|Named'
	)
	pass E8-NAMED-01
}

same_text() { [[ "$1" == "$2" ]]; }

rows=(
	row_e1_http row_e1_grpc row_e1_runtime row_e2_oauth row_e2_none
	row_e3_input row_e3_precondition row_e3_contract
	row_e4_initial row_e4_repeat row_e4_refresh row_e5_openapi row_e5_proto
	row_e5_routing row_e5_boundary row_e5_gates row_e6_http row_e6_grpc
	row_e7_initial row_e7_refresh row_e8_disclosure row_e8_legacy row_e8_named
)
if [[ "${1:-}" == "--list" ]]; then
	printf '%s\n' "${rows[@]}"
	exit 0
fi
if (($# > 0)); then
	rows=("$@")
fi
for row in "${rows[@]}"; do
	if [[ "${row}" != row_* ]] || ! declare -F "${row}" >/dev/null; then
		echo "unknown integration-init row: ${row}" >&2
		exit 2
	fi
	"${row}"
done

if [[ "${#PASSED[@]}" -ne "${#rows[@]}" ]]; then
	echo "expected ${#rows[@]} matrix IDs, got ${#PASSED[@]}: ${PASSED[*]}" >&2
	exit 1
fi
echo "case count: ${#PASSED[@]}"
