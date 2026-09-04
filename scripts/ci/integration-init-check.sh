#!/usr/bin/env bash
# Prove make integration-init against disposable initialized fixtures.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

if [[ ${1:-} == --select-from-files ]]; then
	engine=false
	http=false
	grpc=false
	oauth=false
	while IFS= read -r file; do
		[[ -n ${file} ]] || continue
		case "${file}" in
		scripts/integration-init.sh | scripts/ci/integration-init-check.sh | scripts/openapi-ref-check.go)
			engine=true
			;;
		*.proto | buf.yaml | buf.gen.yaml | buf.lock | api/proto/* | internal/gen/proto/*)
			grpc=true
			;;
		.redocly.yaml | api/openapi/* | api/external/*/openapi.yaml | examples/reference-service/api/openapi.yaml | internal/openapi/* | */internal/openapi/*)
			http=true
			;;
		*oauth* | *outbound-auth* | *clientcredentials*)
			oauth=true
			;;
		esac
	done
	if [[ ${engine} == true ]] || [[ ${http} != true && ${grpc} != true && ${oauth} != true ]]; then
		printf '%s\n' row_e1_http row_e1_grpc row_e2_oauth row_e2_none row_e5_openapi row_e5_proto row_e6_http row_e6_grpc row_e7_refresh row_e8_disclosure
		exit 0
	fi
	rows=()
	if [[ ${http} == true ]]; then
		rows+=(row_e1_http row_e5_openapi row_e6_http)
	fi
	if [[ ${grpc} == true ]]; then
		rows+=(row_e1_grpc row_e5_proto row_e6_grpc)
	fi
	if [[ ${oauth} == true ]]; then
		rows+=(row_e2_oauth row_e8_disclosure)
	fi
	if [[ ${http} == true || ${grpc} == true ]]; then
		rows+=(row_e2_none)
	fi
	printf '%s\n' "${rows[@]}" | LC_ALL=C sort -u
	exit 0
fi

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

replace_text() {
	python3 - "$1" "$2" "$3" <<'PY'
from pathlib import Path
import sys
path = Path(sys.argv[1])
old, new = sys.argv[2], sys.argv[3]
text = path.read_text()
if old not in text:
    raise SystemExit(f"missing replacement anchor {old!r}")
path.write_text(text.replace(old, new, 1))
PY
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
	local name="${2:-billing}"
	mkdir -p "${root}/api/external/${name}"
	cat >"${root}/api/external/${name}/openapi.yaml" <<EOF
openapi: 3.0.3
info:
  title: ${name} probe
  version: 1.0.0
  license:
    name: MIT
    url: https://opensource.org/licenses/MIT
servers:
  - url: https://${name}.example.com
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
	commit_all "${root}" "add ${name} contract"
}

mutate_http_contract() {
	local root="$1"
	replace_text "${root}/api/external/billing/openapi.yaml" \
		$'                  status:\n                    type: string\n' \
		$'                  status:\n                    type: string\n                  refreshed:\n                    type: boolean\n'
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

assert_init_rejected() {
	local root="$1"
	local description="$2"
	shift 2
	local before
	before="$(snapshot_tree "${root}")"
	if run_init "${root}" "$@"; then
		echo "${description} was accepted" >&2
		exit 1
	fi
	assert "${description} unchanged" same_text "${before}" "$(snapshot_tree "${root}")"
}

write_stage_guard() {
	local guard="$1"
	local marker="$2"
	local real_git
	real_git="$(command -v git)"
	mkdir -p "${guard}"
	cat >"${guard}/git" <<EOF
#!/usr/bin/env bash
case " \$* " in
*" worktree add "* | *" apply "*)
	: >"${marker}"
	exit 96
	;;
esac
exec "${real_git}" "\$@"
EOF
	chmod +x "${guard}/git"
}

assert_rejected_before_stage() {
	local root="$1"
	local description="$2"
	local guard="$3"
	local marker="$4"
	shift 4
	rm -f "${marker}"
	PATH="${guard}:${PATH}" assert_init_rejected "${root}" "${description}" "$@"
	assert "${description} stopped before staging" path_absent "${marker}"
}

assert_record_check_rejected() {
	local root="$1"
	local description="$2"
	if (cd "${root}" && make integration-record-check); then
		echo "${description} passed integration record parity" >&2
		exit 1
	fi
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

snapshot_ignored_path() {
	local path="$1"
	if [[ -L "${path}" ]]; then
		printf 'symlink %s\n' "$(readlink "${path}" | git hash-object --stdin)"
		return
	fi
	if [[ ! -e "${path}" ]]; then
		printf 'absent\n'
		return
	fi
	local mode
	if ! mode="$(stat -f '%Lp' "${path}" 2>/dev/null)"; then
		mode="$(stat -c '%a' "${path}")"
	fi
	printf 'regular %s %s\n' "${mode}" "$(git hash-object --no-filters "${path}")"
}

assert_only_path_changed() {
	local root="$1"
	local expected="$2"
	local line path count=0
	while IFS= read -r line; do
		[[ -n "${line}" ]] || continue
		path="${line:3}"
		if [[ "${path}" != "${expected}" ]]; then
			echo "unexpected changed path ${path}, want only ${expected}" >&2
			return 1
		fi
		((count += 1))
	done < <(git -C "${root}" status --porcelain)
	[[ "${count}" -eq 1 ]]
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
	if grep -R -F --exclude=.env --binary-files=without-match -- "${canary}" "${root}" "$@" >/dev/null 2>&1; then
		return 1
	fi
	return 0
}

initializer_avoids_root_env() {
	python3 - "$1" <<'PY'
from pathlib import Path
import re
import sys
text = Path(sys.argv[1]).read_text().replace("env/.env.example", "")
raise SystemExit(1 if re.search(r"(?<![A-Za-z0-9_])\.env(?![A-Za-z0-9_])", text) else 0)
PY
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
	"time"

	"${module}/internal/infra/billing/internal/openapi"
	"${module}/internal/infra/httpclient"
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

	constructed, err := New(Config{
		BaseURL: "https://billing.example.com",
		Limits: httpclient.TransportLimits{
			ResponseHeaderTimeout: 5 * time.Second,
			MaxResponseHeaderBytes: 32 << 10,
			MaxInFlight:            2,
			AbsoluteBodyBytes:      1 << 20,
		},
	})
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
	local root private_root
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
	private_root="$(http_none_fixture)"
	run_init "${private_root}" NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=private-https AUTH=none
	assert "private suffix config" grep -q 'PrivateDNSSuffix' "${private_root}/internal/config/billing_integration_config.go"
	assert "private suffix test env" grep -q 'APP__INTEGRATIONS__BILLING__PRIVATE_DNS_SUFFIX' "${private_root}/internal/config/configtest/configtest.go"
	(
		cd "${private_root}"
		go test -vet=off ./internal/config ./internal/infra/billing ./cmd/service/internal/bootstrap \
			-run 'Billing|Integration|CloseIdempotent|NewRejects'
		make integration-record-check
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
		make fmt-check proto-check integration-record-check
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
	(
		cd "${later}"
		go test -vet=off ./internal/infra/billing ./internal/infra/identity ./internal/infra/oauth2clientcredentials \
			-run 'CloseIdempotent|GRPC|HTTP|Competing|Cache|Target'
		make integration-record-check
	)
	pass E2-OAUTH-01
}

row_e2_none() {
	local root
	root="$(http_none_fixture)"
	run_init "${root}" NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=none
	assert "no oauth import" grep_absent -R -F oauth2clientcredentials "${root}/internal/infra/billing"
	assert "no oauth adapter/config/wiring" grep_absent -R -F 'OAuth' \
		"${root}/internal/infra/billing/client.go" \
		"${root}/internal/config/billing_integration_config.go" \
		"${root}/cmd/service/internal/bootstrap/startup_billing.go" \
		"${root}/docs/integrations/billing.md"
	assert "no oauth env" grep_absent -F 'APP__INTEGRATIONS__BILLING__OAUTH__' "${root}/env/.env.example"
	pass E2-NONE-01
}

row_e3_input() {
	local root underscore hostile wrap real_go
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
	local keyword output
	for keyword in break case chan const continue default defer else fallthrough for func go goto if import interface map package range return select struct switch type var; do
		if output=$(cd "${root}" && bash scripts/integration-init.sh "${keyword}" http "api/external/${keyword}/openapi.yaml" external-https none 2>&1); then
			echo "Go keyword ${keyword} was accepted as NAME" >&2
			exit 1
		fi
		grep -Fq 'NAME must be a lower-case Go package identifier' <<<"${output}"
	done
	assert "keyword rejections unchanged" same_text "${before}" "$(snapshot_tree "${root}")"
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
	root="$(http_none_fixture)"
	local before_delimiter
	before_delimiter="$(snapshot_tree "${root}")"
	if (
		cd "${root}"
		make integration-init NAME=billing__api TRANSPORT=http CONTRACT=api/external/billing__api/openapi.yaml TARGET=external-https AUTH=none
	); then
		echo "reserved NAME delimiter was accepted" >&2
		exit 1
	fi
	assert "reserved delimiter unchanged" same_text "${before_delimiter}" "$(snapshot_tree "${root}")"
	underscore="$(copy_checkout underscore-name)"
	init_service "${underscore}" OUTBOUND_HTTP=bounded
	write_http_contract "${underscore}" billing_api
	run_init "${underscore}" NAME=billing_api TRANSPORT=http CONTRACT=api/external/billing_api/openapi.yaml TARGET=external-https AUTH=none
	assert "single underscore accepted" file_present "${underscore}/integrations/billing_api.toml"

	hostile="$(http_none_fixture)"
	wrap="${TEMP_ROOT}/go-env-guard"
	real_go="$(command -v go)"
	mkdir -p "${wrap}"
	cat >"${wrap}/go" <<EOF
#!/usr/bin/env bash
[[ "\${GOTOOLCHAIN-}" == local && "\${GOPROXY-}" == off && "\${GOSUMDB-}" == off ]] || exit 91
exec "${real_go}" "\$@"
EOF
	chmod +x "${wrap}/go"
	(
		cd "${hostile}"
		PATH="${wrap}:${PATH}" GOTOOLCHAIN=auto GOPROXY=https://network.invalid GOSUMDB=sum.golang.org \
			make integration-init NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=none
	)
	assert "hostile Go environment was pinned offline" file_present "${hostile}/integrations/billing.toml"
	pass E3-INPUT-01
}

row_e3_precondition() {
	local root base mutant guard marker
	guard="${TEMP_ROOT}/precondition-git"
	marker="${TEMP_ROOT}/precondition-stage.marker"
	write_stage_guard "${guard}" "${marker}"
	root="$(copy_checkout precond)"
	init_service "${root}" # outbound_http=none
	write_http_contract "${root}"
	assert_rejected_before_stage "${root}" "missing outbound_http" "${guard}" "${marker}" \
		NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=none

	mutant="$(http_none_fixture)"
	replace_text "${mutant}/template.lock" 'state = "complete"' 'state = "initializing"'
	commit_all "${mutant}" "leave template initialization incomplete"
	assert_rejected_before_stage "${mutant}" "incomplete template journal" "${guard}" "${marker}" \
		NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=none

	root="$(http_none_fixture)"
	run_init "${root}" NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=none
	commit_all "${root}" "initialized billing"
	base="${root}"
	assert_rejected_before_stage "${base}" "changed target" "${guard}" "${marker}" \
		NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=private-https AUTH=none

	mutant="$(clone_base "${base}" record-schema)"
	replace_text "${mutant}/integrations/billing.toml" "schema = 1" "schema = 2"
	commit_all "${mutant}" "mutate integration schema"
	assert_rejected_before_stage "${mutant}" "changed record schema" "${guard}" "${marker}" \
		NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=none

	mutant="$(clone_base "${base}" record-source)"
	replace_text "${mutant}/integrations/billing.toml" \
		'generator_source = "internal/infra/billing/internal/openapi/doc.go"' \
		'generator_source = "other.go"'
	commit_all "${mutant}" "mutate integration generator source"
	assert_rejected_before_stage "${mutant}" "changed generator source" "${guard}" "${marker}" \
		NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=none

	mutant="$(clone_base "${base}" record-extra)"
	echo 'extra = "rejected"' >>"${mutant}/integrations/billing.toml"
	commit_all "${mutant}" "add integration record key"
	assert_rejected_before_stage "${mutant}" "extra record key" "${guard}" "${marker}" \
		NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=none

	mutant="$(clone_base "${base}" record-duplicate)"
	echo 'name = "billing"' >>"${mutant}/integrations/billing.toml"
	commit_all "${mutant}" "duplicate integration record key"
	assert_rejected_before_stage "${mutant}" "duplicate record key" "${guard}" "${marker}" \
		NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=none

	mutant="$(clone_base "${base}" record-symlink)"
	cp "${mutant}/integrations/billing.toml" "${mutant}/billing-record.toml"
	rm "${mutant}/integrations/billing.toml"
	ln -s ../billing-record.toml "${mutant}/integrations/billing.toml"
	commit_all "${mutant}" "replace integration record with symlink"
	assert_rejected_before_stage "${mutant}" "symlink integration record" "${guard}" "${marker}" \
		NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=none

	mutant="$(clone_base "${base}" residual-marker)"
	printf '%s%s\n' '// profile:residual:' 'start' >"${mutant}/internal/residual_marker.go"
	commit_all "${mutant}" "add residual profile marker"
	assert_rejected_before_stage "${mutant}" "residual profile marker" "${guard}" "${marker}" \
		NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=none
	pass E3-PRECONDITION-01
}

row_e3_contract() {
	local root remote alias_remote invalid wrap real_go marker stage_guard stage_marker
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

	stage_guard="${TEMP_ROOT}/contract-git"
	stage_marker="${TEMP_ROOT}/contract-stage.marker"
	write_stage_guard "${stage_guard}" "${stage_marker}"
	invalid="$(http_none_fixture)"
	replace_text "${invalid}/api/external/billing/openapi.yaml" 'openapi: 3.0.3' 'openapi: invalid'
	commit_all "${invalid}" "add invalid OpenAPI schema"
	assert_rejected_before_stage "${invalid}" "invalid OpenAPI schema" "${stage_guard}" "${stage_marker}" \
		NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=none

	remote="$(http_none_fixture)"
	replace_text "${remote}/api/external/billing/openapi.yaml" \
		'                type: object' \
		"                \$ref: \"http://169.254.169.254/latest/meta-data/iam/security-credentials/\""
	commit_all "${remote}" "add remote OpenAPI ref"
	wrap="${TEMP_ROOT}/ref-go"
	marker="${TEMP_ROOT}/remote-ref-reached-tool.marker"
	real_go="$(command -v go)"
	mkdir -p "${wrap}"
	cat >"${wrap}/go" <<EOF
#!/usr/bin/env bash
case " \$* " in
*" mod init integration-init-name "* | *" run . billing "* | *" ./scripts/openapi-ref-check.go "*)
	exec "${real_go}" "\$@"
	;;
esac
: >"${marker}"
exit 97
EOF
	chmod +x "${wrap}/go"
	PATH="${wrap}:${PATH}" assert_init_rejected "${remote}" "remote OpenAPI ref" \
		NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=none
	assert "remote ref rejected before another Go tool" path_absent "${marker}"

	alias_remote="$(http_none_fixture)"
	replace_text "${alias_remote}/api/external/billing/openapi.yaml" \
		'                type: object' \
		"                refKey: &refKey \"\$ref\"
                *refKey: \"http://169.254.169.254/latest/meta-data/iam/security-credentials/\""
	commit_all "${alias_remote}" "add aliased remote OpenAPI ref"
	PATH="${wrap}:${PATH}" assert_init_rejected "${alias_remote}" "aliased remote OpenAPI ref" \
		NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=none
	assert "aliased remote ref rejected before another Go tool" path_absent "${marker}"

	root="$(http_none_fixture)"
	cat >>"${root}/api/external/billing/openapi.yaml" <<'EOF'
components:
  schemas:
    Probe:
      type: object
    ProbeAlias:
      $ref: "#/components/schemas/Probe"
EOF
	commit_all "${root}" "add document-local OpenAPI ref"
	run_init "${root}" NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=none
	pass E3-CONTRACT-01
}

row_e4_initial() {
	local root
	root="$(http_oauth_fixture)"
	local wrap="${TEMP_ROOT}/gov"
	local real_go marker
	real_go="$(command -v go)"
	marker="${TEMP_ROOT}/initial-stage-test.marker"
	mkdir -p "${wrap}"
	cat >"${wrap}/go" <<EOF
#!/usr/bin/env bash
if [[ "\${1-}" == "test" ]]; then
	echo WRAPPER_GO_TEST=1
	: >"${marker}"
	exit 1
fi
exec "${real_go}" "\$@"
EOF
	chmod +x "${wrap}/go"
	local before env_before
	before="$(snapshot_tree "${root}")"
	env_before="$(snapshot_ignored_path "${root}/.env")"
	if (
		cd "${root}"
		PATH="${wrap}:${PATH}" make integration-init NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=oauth2-client-credentials
	); then
		echo "induced staging failure succeeded" >&2
		exit 1
	fi
	assert "initial failure reached detached stage tests" file_present "${marker}"
	assert "initial failure restored" same_text "${before}" "$(snapshot_tree "${root}")"
	assert "initial ignored env restored" same_text "${env_before}" "$(snapshot_ignored_path "${root}/.env")"
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
	mutate_http_contract "${root}"
	echo sentinel >"${root}/internal/infra/billing/manual.sentinel"
	commit_all "${root}" "contract + sentinel"
	run_init "${root}" NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=none
	assert "sentinel kept" grep -q sentinel "${root}/internal/infra/billing/manual.sentinel"
	assert "refresh generated-only" assert_only_path_changed "${root}" "internal/infra/billing/internal/openapi/client.gen.go"
	pass E4-REPEAT-01
}

row_e4_refresh() {
	local root
	root="$(http_oauth_fixture)"
	run_init "${root}" NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=oauth2-client-credentials
	commit_all "${root}" "oauth initial"
	mutate_http_contract "${root}"
	commit_all "${root}" "contract change"
	printf 'CANARY_REFRESH=1\n' >"${root}/.env"
	local wrap="${TEMP_ROOT}/gor"
	local real_go marker
	real_go="$(command -v go)"
	marker="${TEMP_ROOT}/refresh-stage-test.marker"
	mkdir -p "${wrap}"
	cat >"${wrap}/go" <<EOF
#!/usr/bin/env bash
if [[ "\${1-}" == "test" ]]; then
	: >"${marker}"
	exit 1
fi
exec "${real_go}" "\$@"
EOF
	chmod +x "${wrap}/go"
	local before env_before
	before="$(snapshot_tree "${root}")"
	env_before="$(snapshot_ignored_path "${root}/.env")"
	if (
		cd "${root}"
		PATH="${wrap}:${PATH}" make integration-init NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=oauth2-client-credentials
	); then
		echo "refresh failure succeeded" >&2
		exit 1
	fi
	assert "refresh failure reached detached stage tests" file_present "${marker}"
	assert "refresh failure restored" same_text "${before}" "$(snapshot_tree "${root}")"
	assert "refresh ignored env restored" same_text "${env_before}" "$(snapshot_ignored_path "${root}/.env")"
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
	local root mutant
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
		make integration-record-check
	)
	mutant="$(clone_base "${root}" proto-record-source)"
	replace_text "${mutant}/integrations/identity.toml" \
		'generator_source = "buf.gen.yaml"' \
		'generator_source = "other.yaml"'
	commit_all "${mutant}" "mutate gRPC generator source"
	assert_record_check_rejected "${mutant}" "gRPC generator source mismatch"
	pass E5-PROTO-01
}

row_e5_routing() {
	local root output
	root="$(http_none_fixture)"
	run_init "${root}" NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=none
	(
		cd "${root}"
		make integration-record-check changed-surfaces-check
	)
	output="$(printf '%s\n' integrations/billing.toml | bash "${root}/scripts/ci/changed-surfaces.sh")"
	assert "record routes to parity" grep -qx 'integration_records=true' <<<"${output}"
	output="$(printf '%s\n' api/external/billing/openapi.yaml | bash "${root}/scripts/ci/changed-surfaces.sh")"
	assert "external contract routes to OpenAPI" grep -qx 'openapi=true' <<<"${output}"
	pass E5-ROUTING-01
}

row_e5_boundary() {
	local http_root grpc_root
	http_root="$(http_none_fixture)"
	grpc_root="$(grpc_none_fixture)"
	run_init "${http_root}" NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=private-https AUTH=none
	run_init "${grpc_root}" NAME=identity TRANSPORT=grpc CONTRACT=api/proto/external/identity/v1/identity.proto TARGET= AUTH=none
	(
		cd "${http_root}"
		go test -vet=off -count=1 ./internal/config ./internal/infra/billing ./internal/infra/httpclient \
			-run 'Billing|Integration|NewRejects|CloseIdempotent|Target|Denied'
	)
	(
		cd "${grpc_root}"
		go test -vet=off -count=1 ./internal/config ./internal/infra/identity ./internal/infra/grpcclient \
			-run 'Identity|Integration|NewRejects|CloseIdempotent|Target|Denied'
	)
	pass E5-BOUNDARY-01
}

row_e5_gates() {
	local root base mutant grpc_root private_root oauth_root grpc_oauth
	root="$(http_none_fixture)"
	run_init "${root}" NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=none
	(
		cd "${root}"
		make fmt-check integration-record-check changed-surfaces-check
		report="$(mktemp)"
		go test -vet=off -count=1 -json ./internal/infra/billing ./internal/config \
			-run 'Billing|Integration|NewRejects|CloseIdempotent' >"${report}"
		grep -q '"Action":"run"' "${report}"
		rm -f "${report}"
	)
	commit_all "${root}" "http integration ready"
	base="${root}"

	mutant="$(clone_base "${base}" gate-record)"
	replace_text "${mutant}/integrations/billing.toml" \
		'generator_source = "internal/infra/billing/internal/openapi/doc.go"' \
		'generator_source = "wrong.go"'
	assert_record_check_rejected "${mutant}" "mutated record"

	mutant="$(clone_base "${base}" gate-orphan)"
	rm "${mutant}/integrations/billing.toml"
	assert_record_check_rejected "${mutant}" "missing integration record"

	mutant="$(clone_base "${base}" gate-config)"
	replace_text "${mutant}/internal/config/billing_integration_config.go" \
		'BaseURL                string' \
		'RemovedBaseURL         string'
	assert_record_check_rejected "${mutant}" "mutated config"

	mutant="$(clone_base "${base}" gate-adapter)"
	replace_text "${mutant}/internal/infra/billing/client.go" '/internal/infra/httpclient"' '/internal/infra/httpclient_removed"'
	assert_record_check_rejected "${mutant}" "mutated adapter"

	mutant="$(clone_base "${base}" gate-generated-binding)"
	replace_text "${mutant}/internal/infra/billing/client.go" \
		'openapi.NewClient(transport.BaseURL(), openapi.WithHTTPClient(transport))' \
		'openapi.NewClient(cfg.BaseURL)'
	assert_record_check_rejected "${mutant}" "generated client without bounded transport"

	mutant="$(clone_base "${base}" gate-client-return)"
	replace_text "${mutant}/internal/infra/billing/client.go" 'generated: generated,' 'generated: nil,'
	assert_record_check_rejected "${mutant}" "adapter returns discarded generated client"

	mutant="$(clone_base "${base}" gate-bootstrap)"
	replace_text "${mutant}/cmd/service/internal/bootstrap/startup_billing.go" 'BaseURL: cfg.BaseURL' 'BaseURL: "", // BaseURL: cfg.BaseURL'
	assert_record_check_rejected "${mutant}" "mutated bootstrap"

	mutant="$(clone_base "${base}" gate-bootstrap-discarded)"
	replace_text "${mutant}/cmd/service/internal/bootstrap/startup_billing.go" 'BaseURL: cfg.BaseURL' 'BaseURL: ""'
	replace_text "${mutant}/cmd/service/internal/bootstrap/startup_billing.go" \
		'if err != nil {' \
		$'_ = billing.Config{BaseURL: cfg.BaseURL}\n\tif err != nil {'
	assert_record_check_rejected "${mutant}" "discarded correct bootstrap config"

	mutant="$(clone_base "${base}" gate-bootstrap-return)"
	replace_text "${mutant}/cmd/service/internal/bootstrap/startup_billing.go" 'return client, nil' 'return nil, nil'
	assert_record_check_rejected "${mutant}" "bootstrap returns a different client"

	mutant="$(clone_base "${base}" gate-run-reassign)"
	replace_text "${mutant}/cmd/service/internal/bootstrap/run.go" \
		'billingClosed := false' \
		$'billingClosed := false\n\tbillingClient = nil'
	assert_record_check_rejected "${mutant}" "run wiring reassigns the integration client"

	mutant="$(clone_base "${base}" gate-docs)"
	replace_text "${mutant}/docs/integrations/billing.md" "Authentication: \`none\`" "Authentication: \`other\`"
	assert_record_check_rejected "${mutant}" "mutated documentation"

	mutant="$(clone_base "${base}" gate-generator)"
	replace_text "${mutant}/internal/infra/billing/internal/openapi/doc.go" \
		'../../../../../api/external/billing/openapi.yaml' \
		'../../../../../api/external/other/openapi.yaml'
	assert_record_check_rejected "${mutant}" "mutated generator source"

	private_root="$(http_none_fixture)"
	run_init "${private_root}" NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=private-https AUTH=none
	replace_text "${private_root}/internal/infra/billing/client.go" \
		'httpclient.NewPrivateHTTPS(cfg.BaseURL, cfg.PrivateDNSSuffix, cfg.Limits)' \
		'httpclient.NewExternalHTTPS(cfg.BaseURL, cfg.Limits)'
	assert_record_check_rejected "${private_root}" "mutated private HTTP constructor"

	private_root="$(http_none_fixture)"
	run_init "${private_root}" NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=private-https AUTH=none
	replace_text "${private_root}/internal/infra/billing/client.go" \
		'transport, err := httpclient.NewPrivateHTTPS(cfg.BaseURL, cfg.PrivateDNSSuffix, cfg.Limits)' \
		$'_, _ = httpclient.NewPrivateHTTPS(cfg.BaseURL, cfg.PrivateDNSSuffix, cfg.Limits)\n\ttransport, err := httpclient.NewExternalHTTPS(cfg.BaseURL, cfg.Limits)'
	assert_record_check_rejected "${private_root}" "dead private constructor literal"

	private_root="$(http_none_fixture)"
	run_init "${private_root}" NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=private-https AUTH=none
	replace_text "${private_root}/internal/infra/billing/client.go" \
		$'\t"github.com/acme/http-none-base/internal/infra/httpclient"' \
		$'\thttpc "github.com/acme/http-none-base/internal/infra/httpclient"'
	replace_text "${private_root}/internal/infra/billing/client.go" \
		'transport, err := httpclient.NewPrivateHTTPS(cfg.BaseURL, cfg.PrivateDNSSuffix, cfg.Limits)' \
		$'// httpclient.NewPrivateHTTPS(cfg.BaseURL, cfg.PrivateDNSSuffix, cfg.Limits)\n\ttransport, err := httpc.NewExternalHTTPS(cfg.BaseURL, cfg.Limits)'
	assert_record_check_rejected "${private_root}" "aliased external constructor with private comment"

	private_root="$(http_none_fixture)"
	run_init "${private_root}" NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=private-https AUTH=none
	replace_text "${private_root}/internal/infra/billing/client.go" \
		'transport, err := httpclient.NewPrivateHTTPS(cfg.BaseURL, cfg.PrivateDNSSuffix, cfg.Limits)' \
		'transport, err := buildTransport(cfg)'
	cat >>"${private_root}/internal/infra/billing/client.go" <<'EOF'

func buildTransport(cfg Config) (*httpclient.Client, error) {
	return httpclient.NewExternalHTTPS(cfg.BaseURL, cfg.Limits)
}

func (Client) New(cfg Config) {
	transport, _ := httpclient.NewPrivateHTTPS(cfg.BaseURL, cfg.PrivateDNSSuffix, cfg.Limits)
	_ = transport
}
EOF
	assert_record_check_rejected "${private_root}" "helper constructor with dummy New method"

	oauth_root="$(http_oauth_fixture)"
	run_init "${oauth_root}" NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=oauth2-client-credentials
	replace_text "${oauth_root}/internal/infra/billing/client.go" \
		'openapi.WithHTTPClient(doer)' \
		'openapi.WithHTTPClient(transport)'
	assert_record_check_rejected "${oauth_root}" "OAuth generated client without authenticated doer"

	oauth_root="$(http_oauth_fixture)"
	run_init "${oauth_root}" NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=oauth2-client-credentials
	replace_text "${oauth_root}/internal/infra/billing/client.go" 'TokenURL:     cfg.OAuth.TokenURL' 'TokenURL:     ""'
	assert_record_check_rejected "${oauth_root}" "HTTP OAuth client uses the wrong token tuple"

	grpc_root="$(grpc_none_fixture)"
	run_init "${grpc_root}" NAME=identity TRANSPORT=grpc CONTRACT=api/proto/external/identity/v1/identity.proto TARGET= AUTH=none
	replace_text "${grpc_root}/.golangci.yml" 'identity_grpc_generated_adapter_only:' 'removed_grpc_rule:'
	assert_record_check_rejected "${grpc_root}" "mutated gRPC depguard"

	grpc_oauth="$(copy_checkout gate-grpc-oauth)"
	init_service "${grpc_oauth}" GRPC=enabled OUTBOUND_AUTH=oauth2-client-credentials
	write_grpc_contract "${grpc_oauth}"
	run_init "${grpc_oauth}" NAME=identity TRANSPORT=grpc CONTRACT=api/proto/external/identity/v1/identity.proto TARGET= AUTH=oauth2-client-credentials
	commit_all "${grpc_oauth}" "gRPC OAuth integration ready"

	mutant="$(clone_base "${grpc_oauth}" gate-grpc-oauth-config)"
	replace_text "${mutant}/internal/infra/identity/client.go" 'TokenURL:     cfg.OAuth.TokenURL' 'TokenURL:     ""'
	assert_record_check_rejected "${mutant}" "gRPC OAuth client uses the wrong token tuple"

	mutant="$(clone_base "${grpc_oauth}" gate-grpc-oauth-bypass)"
	replace_text "${mutant}/internal/infra/identity/client.go" \
		'conn, err := auth.GRPC(target, grpcclient.Options{TransportCredentials: creds})' \
		'conn, err := grpcclient.New(target, grpcclient.Options{TransportCredentials: creds})'
	assert_record_check_rejected "${mutant}" "gRPC connection bypasses OAuth"

	mutant="$(clone_base "${grpc_oauth}" gate-grpc-oauth-return)"
	replace_text "${mutant}/internal/infra/identity/client.go" 'auth: auth,' 'auth: nil,'
	assert_record_check_rejected "${mutant}" "gRPC client discards OAuth owner"

	mutant="$(clone_base "${grpc_oauth}" gate-grpc-oauth-close)"
	replace_text "${mutant}/internal/infra/identity/client.go" 'c.auth.Close()' '// c.auth.Close()'
	assert_record_check_rejected "${mutant}" "gRPC client does not retire OAuth owner"

	mutant="$(clone_base "${grpc_oauth}" gate-grpc-oauth-close-outside-once)"
	replace_text "${mutant}/internal/infra/identity/client.go" \
		$'c.closeOnce.Do(func() {\n\t\tif c.auth != nil {\n\t\t\tc.auth.Close()\n\t\t}\n\t\tif c.conn != nil {\n\t\t\tc.closeErr = c.conn.Close()\n\t\t}\n\t})' \
		$'c.closeOnce.Do(func() {})\n\tif c.auth != nil {\n\t\tc.auth.Close()\n\t}\n\tif c.conn != nil {\n\t\tc.closeErr = c.conn.Close()\n\t}'
	assert_record_check_rejected "${mutant}" "gRPC resources close outside sync.Once"

	mutant="$(clone_base "${grpc_oauth}" gate-grpc-oauth-inverted-close-guards)"
	replace_text "${mutant}/internal/infra/identity/client.go" 'if c.auth != nil {' 'if c.auth == nil {'
	replace_text "${mutant}/internal/infra/identity/client.go" 'if c.conn != nil {' 'if c.conn == nil {'
	assert_record_check_rejected "${mutant}" "gRPC resources use inverted close guards"

	mutant="$(clone_base "${grpc_oauth}" gate-grpc-oauth-dead-close)"
	replace_text "${mutant}/internal/infra/identity/client.go" \
		$'c.closeOnce.Do(func() {\n\t\tif c.auth != nil {\n\t\t\tc.auth.Close()\n\t\t}\n\t\tif c.conn != nil {\n\t\t\tc.closeErr = c.conn.Close()\n\t\t}\n\t})' \
		$'if false {\n\t\tc.closeOnce.Do(func() {\n\t\t\tif c.auth != nil {\n\t\t\t\tc.auth.Close()\n\t\t\t}\n\t\t\tif c.conn != nil {\n\t\t\t\tc.closeErr = c.conn.Close()\n\t\t\t}\n\t\t})\n\t}'
	assert_record_check_rejected "${mutant}" "gRPC close is hidden in a dead branch"
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
	local root module
	root="$(grpc_none_fixture)"
	run_init "${root}" NAME=identity TRANSPORT=grpc CONTRACT=api/proto/external/identity/v1/identity.proto TARGET= AUTH=none
	module="$(go -C "${root}" list -m)"
	cat >"${root}/internal/infra/identity/client_contract_test.go" <<EOF
package identity

import (
	"context"
	"net"
	"testing"

	identityv1 "${module}/internal/gen/proto/external/identity/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

type harnessProbeServer struct {
	identityv1.UnimplementedProbeServiceServer
	calls int
}

func (s *harnessProbeServer) Echo(_ context.Context, request *identityv1.EchoRequest) (*identityv1.EchoResponse, error) {
	s.calls++
	return &identityv1.EchoResponse{Value: request.GetValue()}, nil
}

func TestHarnessGRPCContract(t *testing.T) {
	constructed, err := New(Config{Target: "dns:///identity.example.com:443"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() { _ = constructed.Close() })

	listener := bufconn.Listen(1 << 20)
	server := grpc.NewServer()
	probe := new(harnessProbeServer)
	identityv1.RegisterProbeServiceServer(server, probe)
	serveDone := make(chan error, 1)
	go func() { serveDone <- server.Serve(listener) }()
	connection, err := grpc.NewClient(
		"passthrough:///bufconn",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		server.Stop()
		_ = listener.Close()
		<-serveDone
		t.Fatalf("grpc.NewClient() error = %v", err)
	}
	t.Cleanup(func() {
		_ = connection.Close()
		server.Stop()
		_ = listener.Close()
		<-serveDone
	})

	response, err := identityv1.NewProbeServiceClient(connection).Echo(t.Context(), &identityv1.EchoRequest{Value: "probe"})
	if err != nil {
		t.Fatalf("Echo() error = %v", err)
	}
	if response.GetValue() != "probe" || probe.calls != 1 {
		t.Fatalf("Echo() = %q, calls=%d", response.GetValue(), probe.calls)
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

http_initial_path_allowed() {
	case "$1" in
	integrations/ | integrations/billing.toml | \
		internal/infra/billing/* | \
		internal/config/billing_integration_config.go | \
		internal/config/billing_integration_config_test.go | \
		internal/config/integrations_config.go | \
		internal/config/types.go | \
		internal/config/defaults.go | \
		internal/config/validate.go | \
		internal/config/snapshot_contract_test.go | \
		internal/config/testhelpers_test.go | \
		internal/config/configtest/configtest.go | \
		cmd/service/internal/bootstrap/startup_billing.go | \
		cmd/service/internal/bootstrap/startup_billing_test.go | \
		cmd/service/internal/bootstrap/run.go | \
		cmd/service/internal/bootstrap/run_test.go | \
		docs/integrations/ | docs/integrations/billing.md | \
		env/config/local.yaml | \
		env/.env.example)
		return 0
		;;
	esac
	return 1
}

assert_only_http_initial_paths_changed() {
	local root="$1"
	local line path
	while IFS= read -r line; do
		[[ -n "${line}" ]] || continue
		path="${line:3}"
		if ! http_initial_path_allowed "${path}"; then
			echo "unexpected initial path change: ${path}" >&2
			return 1
		fi
	done < <(git -C "${root}" status --porcelain)
}

row_e7_initial() {
	local root
	root="$(http_none_fixture)"
	echo unrelated-sentinel >"${root}/README.md.sentinel"
	git -C "${root}" add README.md.sentinel
	commit_all "${root}" "sentinel"
	run_init "${root}" NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=none
	assert "initial allowlist" assert_only_http_initial_paths_changed "${root}"
	assert "unrelated sentinel" grep -qx unrelated-sentinel "${root}/README.md.sentinel"
	pass E7-INITIAL-01
}

row_e7_refresh() {
	local root
	root="$(http_none_fixture)"
	run_init "${root}" NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=none
	echo manual >"${root}/docs/integrations/billing.md"
	commit_all "${root}" "manual doc"
	mutate_http_contract "${root}"
	commit_all "${root}" "contract"
	run_init "${root}" NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=none
	assert "manual doc kept" grep -qx manual "${root}/docs/integrations/billing.md"
	assert "manual refresh generated-only" assert_only_path_changed "${root}" "internal/infra/billing/internal/openapi/client.gen.go"
	pass E7-REFRESH-01
}

row_e8_disclosure() {
	local root
	root="$(http_oauth_fixture)"
	local disclosure_canary
	disclosure_canary="$(printf '%s%s' 'CANARY_DISCLOSE_' 'runtime_secret')"
	printf 'CANARY_DISCLOSE=%s\n' "${disclosure_canary}" >"${root}/.env"
	local out="${TEMP_ROOT}/disclose.out"
	assert "initializer source excludes root env" initializer_avoids_root_env "${root}/scripts/integration-init.sh"
	local mutant="${TEMP_ROOT}/integration-init-env-mutant.sh"
	cp "${root}/scripts/integration-init.sh" "${mutant}"
	printf '%s\n' 'head -n1 .env >/dev/null' >>"${mutant}"
	if initializer_avoids_root_env "${mutant}"; then
		echo "root env structural mutant passed" >&2
		exit 1
	fi

	local trace="${TEMP_ROOT}/disclose.trace"
	: >"${trace}"
	local command=(make integration-init NAME=billing TRANSPORT=http CONTRACT=api/external/billing/openapi.yaml TARGET=external-https AUTH=oauth2-client-credentials)
	if [[ "$(uname -s)" == "Linux" ]] && command -v strace >/dev/null 2>&1; then
		if ! (cd "${root}" && strace -f -qq -e trace=%file -o "${trace}" "${command[@]}") >"${out}" 2>&1; then
			echo "disclosure fixture failed under strace" >&2
			exit 1
		fi
		assert "root env was not opened" grep_absent -E '"(\./)?\.env"' "${trace}"
	elif ! (cd "${root}" && "${command[@]}") >"${out}" 2>&1; then
		echo "disclosure fixture failed" >&2
		exit 1
	fi
	assert "env bytes unchanged" grep -q "CANARY_DISCLOSE=${disclosure_canary}" "${root}/.env"
	assert "no canary in output" grep_absent -F "${disclosure_canary}" "${out}"
	assert "no canary outside env" scan_canary "${root}" "${disclosure_canary}" "${out}" "${trace}"
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
	cat >"${root}/internal/config/named_integration_contract_test.go" <<'EOF'
package config

import (
	"errors"
	"strings"
	"testing"
)

//nolint:paralleltest // resetConfigEnv mutates process-wide configuration environment.
func TestHarnessNamedIntegrationEnvironment(t *testing.T) {
	resetConfigEnv(t)
	cfg, _, err := LoadDetailed(LoadOptions{})
	if err != nil {
		t.Fatalf("LoadDetailed() error = %v", err)
	}
	if cfg.Integrations.Billing.BaseURL != "https://billing.example.com" ||
		cfg.Integrations.Billing.OAuth.ClientID != "test-client" ||
		cfg.Integrations.Billing.OAuth.ClientSecret != "test-client-secret" {
		t.Fatalf("named billing tuple = %+v", cfg.Integrations.Billing)
	}

	t.Setenv("APP__INTEGRATIONS__BILLING__OAUTH__CLIENT_SECRET", "")
	if _, _, err := LoadDetailed(LoadOptions{}); !errors.Is(err, ErrValidate) {
		t.Fatalf("missing secret error = %v, want ErrValidate", err)
	}

	const canary = "NAMED_FILE_SECRET_CANARY"
	path := writeTempConfig(t, `integrations:
  billing:
    oauth:
      client_secret: "`+canary+`"`)
	if _, _, err := LoadDetailed(LoadOptions{ConfigPath: path}); err == nil {
		t.Fatal("file-sourced named secret was accepted")
	} else if strings.Contains(err.Error(), canary) {
		t.Fatal("file-secret error disclosed canary")
	}
}
EOF
	(
		cd "${root}"
		go test -vet=off -count=1 ./internal/config -run '^TestHarnessNamedIntegrationEnvironment$'
	)
	rm -f "${root}/internal/config/named_integration_contract_test.go"
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
