#!/usr/bin/env bash
# make integration-init: add one named outbound HTTP or gRPC integration.
set -euo pipefail

export GOTOOLCHAIN="${GOTOOLCHAIN:-local}"
export GOPROXY="${GOPROXY:-off}"
export GOSUMDB="${GOSUMDB:-off}"

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
NAME="${1-}"
TRANSPORT="${2-}"
CONTRACT="${3-}"
TARGET="${4-}"
AUTH="${5-}"

reason_input="invalid initializer input"
reason_precondition="initializer precondition failed"
reason_contract="invalid integration contract"
reason_env="move or preserve .env before the first OAuth integration; the initializer does not take custody of .env"

die() {
	local class="$1"
	shift
	printf '%s: %s\n' "${class}" "$*" >&2
	exit 1
}

usage() {
	die "${reason_input}" "usage: $0 NAME TRANSPORT CONTRACT TARGET AUTH"
}

# env_entry_present observes only whether repository-root .env exists.
# It never opens, reads, follows, or mutates the entry.
env_entry_present() {
	local root="$1"
	[[ -n "$(find "${root}" -maxdepth 1 -name .env -print)" ]]
}

lock_path=""
stage_dir=""
cleanup() {
	if [[ -n "${stage_dir}" && -d "${stage_dir}" ]]; then
		git -C "${ROOT_DIR}" worktree remove --force "${stage_dir}" >/dev/null 2>&1 || rm -rf "${stage_dir}"
	fi
	if [[ -n "${lock_path}" && -d "${lock_path}" ]]; then
		rmdir "${lock_path}" >/dev/null 2>&1 || true
	fi
}

module_path() {
	go -C "${ROOT_DIR}" list -m
}

field_name() {
	local name="$1"
	local first
	first="$(printf '%s' "${name}" | cut -c1 | tr '[:lower:]' '[:upper:]')"
	printf '%s%s' "${first}" "${name:1}"
}

admit_name() {
	local name="$1"
	[[ "${name}" =~ ^[a-z][a-z0-9_]*$ ]] || return 1
	local tmp
	tmp="$(mktemp -d)"
	cat >"${tmp}/main.go" <<'EOF'
package main

import (
	"go/token"
	"os"
)

func main() {
	name := os.Args[1]
	if !token.IsIdentifier(name) || token.IsKeyword(name) {
		os.Exit(1)
	}
}
EOF
	(cd "${tmp}" && go mod init integration-init-name >/dev/null 2>&1 && go run . "${name}")
	local status=$?
	rm -rf "${tmp}"
	return "${status}"
}

lock_value() {
	local key="$1"
	local file="${ROOT_DIR}/template.lock"
	local line
	line="$(grep -E "^${key} = \"" "${file}" | head -n1 || true)"
	[[ -n "${line}" ]] || return 1
	line="${line#*\"}"
	printf '%s' "${line%\"}"
}

record_path() {
	printf '%s/integrations/%s.toml' "${ROOT_DIR}" "${NAME}"
}

parse_record() {
	local file="$1"
	local key="$2"
	local line
	line="$(grep -E "^${key} = \"" "${file}" 2>/dev/null | head -n1 || true)"
	[[ -n "${line}" ]] || return 1
	line="${line#*\"}"
	printf '%s' "${line%\"}"
}

write_record() {
	local dest="$1"
	local generator_source="$2"
	mkdir -p "$(dirname "${dest}")"
	{
		echo "schema = 1"
		echo "name = \"${NAME}\""
		echo "transport = \"${TRANSPORT}\""
		echo "contract = \"${CONTRACT}\""
		if [[ "${TRANSPORT}" == "http" ]]; then
			echo "target = \"${TARGET}\""
		fi
		echo "auth = \"${AUTH}\""
		echo "generator_source = \"${generator_source}\""
	} >"${dest}"
}

clean_worktree() {
	[[ -z "$(git -C "${ROOT_DIR}" status --porcelain)" ]]
}

same_head() {
	local expected="$1"
	[[ "$(git -C "${ROOT_DIR}" rev-parse HEAD)" == "${expected}" ]]
}

contract_ok() {
	local path="$1"
	local abs
	abs="$(cd "${ROOT_DIR}" && python3 -c 'import os,sys; print(os.path.realpath(sys.argv[1]))' "${path}")"
	root_real="$(python3 -c 'import os,sys; print(os.path.realpath(sys.argv[1]))' "${ROOT_DIR}")"
	case "${abs}" in
	"${root_real}"/*) ;;
	*) return 1 ;;
	esac
	[[ -f "${ROOT_DIR}/${path}" ]] || return 1
	[[ ! -L "${ROOT_DIR}/${path}" ]] || return 1
	git -C "${ROOT_DIR}" cat-file -e "HEAD:${path}" 2>/dev/null || return 1
	[[ -z "$(git -C "${ROOT_DIR}" status --porcelain -- "${path}")" ]] || return 1
	return 0
}

reserved_outputs() {
	printf '%s\n' \
		"integrations/${NAME}.toml" \
		"internal/infra/${NAME}" \
		"internal/config/${NAME}_integration_config.go" \
		"internal/config/${NAME}_integration_config_test.go" \
		"cmd/service/internal/bootstrap/startup_${NAME}.go" \
		"cmd/service/internal/bootstrap/startup_${NAME}_test.go" \
		"docs/integrations/${NAME}.md"
	if [[ "${TRANSPORT}" == "http" ]]; then
		printf '%s\n' "internal/infra/${NAME}/internal/openapi"
	else
		printf '%s\n' "internal/gen/proto/external/${NAME}"
	fi
}

path_exists() {
	[[ -e "$1" || -L "$1" ]]
}

singleton_oauth_present() {
	[[ -f "${ROOT_DIR}/internal/config/outbound_auth_config.go" ]]
}

classify_mode() {
	local rec
	rec="$(record_path)"
	if [[ ! -f "${rec}" ]]; then
		printf 'initial'
		return
	fi
	local got_name got_transport got_contract got_target got_auth
	got_name="$(parse_record "${rec}" name || true)"
	got_transport="$(parse_record "${rec}" transport || true)"
	got_contract="$(parse_record "${rec}" contract || true)"
	got_auth="$(parse_record "${rec}" auth || true)"
	if [[ "${got_name}" != "${NAME}" || "${got_transport}" != "${TRANSPORT}" || "${got_contract}" != "${CONTRACT}" || "${got_auth}" != "${AUTH}" ]]; then
		die "${reason_precondition}" "locked integration identity does not match this invocation"
	fi
	if [[ "${TRANSPORT}" == "http" ]]; then
		got_target="$(parse_record "${rec}" target || true)"
		if [[ "${got_target}" != "${TARGET}" ]]; then
			die "${reason_precondition}" "locked integration identity does not match this invocation"
		fi
	elif parse_record "${rec}" target >/dev/null; then
		die "${reason_precondition}" "gRPC record must not declare target"
	fi
	printf 'refresh'
}

retires_singleton_oauth() {
	local mode="$1"
	[[ "${mode}" == "initial" && "${AUTH}" == "oauth2-client-credentials" && "$(singleton_oauth_present && echo yes || echo no)" == "yes" ]]
}

require_tools() {
	if [[ "${TRANSPORT}" == "http" ]]; then
		bash "${ROOT_DIR}/scripts/run-go-tool.sh" oapi-codegen -version >/dev/null
	else
		bash "${ROOT_DIR}/scripts/run-buf.sh" --version >/dev/null
	fi
}

validate_contract_schema() {
	if [[ "${TRANSPORT}" == "http" ]]; then
		bash "${ROOT_DIR}/scripts/run-go-tool.sh" validate -- "${ROOT_DIR}/${CONTRACT}"
	else
		(
			cd "${ROOT_DIR}"
			bash ./scripts/run-buf.sh lint "${CONTRACT}"
		)
	fi
}

insert_integrations_field() {
	local types="${ROOT_DIR}/internal/config/types.go"
	if grep -q 'Integrations IntegrationsConfig' "${types}"; then
		return
	fi
	python3 - "${types}" <<'PY'
from pathlib import Path
import sys
path = Path(sys.argv[1])
text = path.read_text()
needle = "\tRuntime       RuntimeConfig       `koanf:\"runtime\"`\n"
insert = "\tIntegrations  IntegrationsConfig  `koanf:\"integrations\"`\n"
if insert.strip() in text:
    raise SystemExit(0)
if needle not in text:
    raise SystemExit("types.go is missing the Runtime field anchor")
path.write_text(text.replace(needle, insert + needle, 1))
PY
}

ensure_integrations_config() {
	local field="$1"
	local dest="${ROOT_DIR}/internal/config/integrations_config.go"
	if [[ ! -f "${dest}" ]]; then
		cat >"${dest}" <<EOF
package config

// IntegrationsConfig holds one concrete field per named outbound integration.
type IntegrationsConfig struct {
	${field} ${field}IntegrationConfig \`koanf:"${NAME}"\`
}
EOF
		return
	fi
	if grep -q "${field} ${field}IntegrationConfig" "${dest}"; then
		return
	fi
	python3 - "${dest}" "${field}" "${NAME}" <<'PY'
from pathlib import Path
import sys
path = Path(sys.argv[1])
field, name = sys.argv[2], sys.argv[3]
text = path.read_text()
needle = "type IntegrationsConfig struct {\n"
insert = f"type IntegrationsConfig struct {{\n\t{field} {field}IntegrationConfig `koanf:\"{name}\"`\n"
if f"{field} {field}IntegrationConfig" in text:
    raise SystemExit(0)
if needle not in text:
    raise SystemExit("integrations_config.go is missing IntegrationsConfig")
path.write_text(text.replace(needle, insert, 1))
PY
}

add_defaults_call() {
	local fn="$1"
	local dest="${ROOT_DIR}/internal/config/defaults.go"
	if grep -q "${fn}()" "${dest}"; then
		return
	fi
	python3 - "${dest}" "${fn}" <<'PY'
from pathlib import Path
import sys
path = Path(sys.argv[1])
fn = sys.argv[2]
text = path.read_text()
needle = "\tmaps.Copy(values, observabilityDefaults())\n"
insert = f"\tmaps.Copy(values, observabilityDefaults())\n\tmaps.Copy(values, {fn}())\n"
if f"maps.Copy(values, {fn}())" in text:
    raise SystemExit(0)
if needle not in text:
    raise SystemExit("defaults.go is missing the observabilityDefaults anchor")
path.write_text(text.replace(needle, insert, 1))
PY
}

add_validate_call() {
	local fn="$1"
	local dest="${ROOT_DIR}/internal/config/validate.go"
	if grep -q "${fn}(" "${dest}"; then
		return
	fi
	python3 - "${dest}" "${fn}" <<'PY'
from pathlib import Path
import sys
path = Path(sys.argv[1])
fn = sys.argv[2]
text = path.read_text()
needle = "\tif err := validateHTTPConfig(&cfg.HTTP); err != nil {\n\t\treturn err\n\t}\n"
insert = needle + f"\tif err := {fn}(&cfg.Integrations); err != nil {{\n\t\treturn err\n\t}}\n"
if f"{fn}(" in text:
    raise SystemExit(0)
if needle not in text:
    raise SystemExit("validate.go is missing the validateHTTPConfig anchor")
path.write_text(text.replace(needle, insert, 1))
PY
}

add_snapshot_sentinels() {
	local dest="${ROOT_DIR}/internal/config/snapshot_contract_test.go"
	local keys_src="$1"
	local keys_exp="$2"
	python3 - "${dest}" "${keys_src}" "${keys_exp}" <<'PY'
from pathlib import Path
import sys
path = Path(sys.argv[1])
src, exp = sys.argv[2], sys.argv[3]
text = path.read_text()
for fn, block in (
    ("func sentinelConfigSourceValues() map[string]any {", src),
    ("func expectedSentinelSnapshotValues() map[string]any {", exp),
):
    start = text.find(fn)
    if start < 0:
        raise SystemExit(f"missing {fn}")
    marker = "return map[string]any{"
    brace = text.find(marker, start)
    if brace < 0:
        raise SystemExit(f"missing map literal in {fn}")
    insert_at = text.find("\n", brace) + 1
    if block.strip() and block.strip() not in text:
        text = text[:insert_at] + block + text[insert_at:]
path.write_text(text)
PY
}

remove_singleton_oauth() {
	rm -f \
		"${ROOT_DIR}/internal/config/outbound_auth_config.go" \
		"${ROOT_DIR}/internal/config/outbound_auth_config_test.go"
	python3 - "${ROOT_DIR}" <<'PY'
from pathlib import Path
import re, sys
root = Path(sys.argv[1])

def strip_profile(path, start_mark, end_mark):
    text = path.read_text()
    pattern = re.compile(re.escape(start_mark) + r".*?" + re.escape(end_mark) + r"\n?", re.S)
    path.write_text(pattern.sub("", text))

start = "// profile:outbound-auth-oauth2-client-credentials:start"
end = "// profile:outbound-auth-oauth2-client-credentials:end"
for rel in [
    "internal/config/types.go",
    "internal/config/defaults.go",
    "internal/config/validate.go",
    "internal/config/snapshot_contract_test.go",
    "internal/config/configtest/configtest.go",
]:
    path = root / rel
    if path.exists():
        strip_profile(path, start, end)

types = root / "internal/config/types.go"
if types.exists():
    text = types.read_text()
    text = re.sub(r"\tOutboundAuth OutboundAuthConfig `koanf:\"outbound_auth\"`\n", "", text)
    types.write_text(text)
defaults = root / "internal/config/defaults.go"
if defaults.exists():
    text = defaults.read_text()
    text = text.replace("\tmaps.Copy(values, outboundAuthDefaults())\n", "")
    defaults.write_text(text)
validate = root / "internal/config/validate.go"
if validate.exists():
    text = validate.read_text()
    text = re.sub(
        r"\tif err := validateOutboundAuthConfig\(&cfg\.OutboundAuth\); err != nil \{\n\t\treturn err\n\t\}\n",
        "",
        text,
    )
    validate.write_text(text)
PY
}

write_http_config() {
	local field="$1"
	local dest="${ROOT_DIR}/internal/config/${NAME}_integration_config.go"
	local suffix_field=""
	local suffix_validate=""
	local suffix_default=""
	if [[ "${TARGET}" == "private-https" ]]; then
		suffix_field=$'\n\tPrivateDNSSuffix string `koanf:"private_dns_suffix"`'
		suffix_default=$'\n\t\t"integrations.'"${NAME}"'.private_dns_suffix": "",'
		suffix_validate=$(
			cat <<EOF

	cfg.${field}.PrivateDNSSuffix = strings.TrimSpace(cfg.${field}.PrivateDNSSuffix)
	if cfg.${field}.PrivateDNSSuffix == "" {
		return fmt.Errorf("%w: integrations.${NAME}.private_dns_suffix is required", ErrValidate)
	}
EOF
		)
	fi
	local oauth_field=""
	local oauth_default=""
	local oauth_validate=""
	if [[ "${AUTH}" == "oauth2-client-credentials" ]]; then
		oauth_field=$'\n\tOAuth OutboundAuthConfig `koanf:"oauth"`'
		oauth_default=$(
			cat <<EOF

		"integrations.${NAME}.oauth.token_url":     "",
		"integrations.${NAME}.oauth.client_id":     "",
		"integrations.${NAME}.oauth.client_secret": "",
		"integrations.${NAME}.oauth.scopes":        "",
		"integrations.${NAME}.oauth.audience":      "",
		"integrations.${NAME}.oauth.resource":      "",
EOF
		)
		oauth_validate=$(
			cat <<EOF

	if err := validateNamedOAuthConfig(&cfg.${field}.OAuth, "integrations.${NAME}.oauth"); err != nil {
		return err
	}
EOF
		)
	fi
	cat >"${dest}" <<EOF
package config

import (
	"fmt"
	"net/url"
	"strings"
)

type ${field}IntegrationConfig struct {
	BaseURL string \`koanf:"base_url"\`${suffix_field}${oauth_field}
}

func ${NAME}IntegrationDefaults() map[string]any {
	return map[string]any{
		"integrations.${NAME}.base_url": "",${suffix_default}${oauth_default}
	}
}

func validate${field}Integration(cfg *IntegrationsConfig) error {
	cfg.${field}.BaseURL = strings.TrimSpace(cfg.${field}.BaseURL)
	endpoint, err := url.Parse(cfg.${field}.BaseURL)
	if err != nil || endpoint == nil || !endpoint.IsAbs() || endpoint.Opaque != "" ||
		!strings.EqualFold(endpoint.Scheme, "https") || endpoint.Host == "" || endpoint.Hostname() == "" ||
		endpoint.User != nil || endpoint.RawQuery != "" || endpoint.ForceQuery || endpoint.Fragment != "" {
		return fmt.Errorf("%w: integrations.${NAME}.base_url must be an absolute HTTPS URL", ErrValidate)
	}
	endpoint.Scheme = "https"
	endpoint.Host = strings.ToLower(endpoint.Host)
	cfg.${field}.BaseURL = endpoint.String()
${suffix_validate}${oauth_validate}
	return nil
}
EOF
	if [[ "${AUTH}" == "oauth2-client-credentials" ]]; then
		write_named_oauth_validator
	fi
	write_config_test "${field}" "http"
}

write_grpc_config() {
	local field="$1"
	local dest="${ROOT_DIR}/internal/config/${NAME}_integration_config.go"
	local oauth_field=""
	local oauth_default=""
	local oauth_validate=""
	if [[ "${AUTH}" == "oauth2-client-credentials" ]]; then
		oauth_field=$'\n\tOAuth OutboundAuthConfig `koanf:"oauth"`'
		oauth_default=$(
			cat <<EOF

		"integrations.${NAME}.oauth.token_url":     "",
		"integrations.${NAME}.oauth.client_id":     "",
		"integrations.${NAME}.oauth.client_secret": "",
		"integrations.${NAME}.oauth.scopes":        "",
		"integrations.${NAME}.oauth.audience":      "",
		"integrations.${NAME}.oauth.resource":      "",
EOF
		)
		oauth_validate=$(
			cat <<EOF

	if err := validateNamedOAuthConfig(&cfg.${field}.OAuth, "integrations.${NAME}.oauth"); err != nil {
		return err
	}
EOF
		)
	fi
	cat >"${dest}" <<EOF
package config

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

type ${field}IntegrationConfig struct {
	Target string \`koanf:"target"\`${oauth_field}
}

func ${NAME}IntegrationDefaults() map[string]any {
	return map[string]any{
		"integrations.${NAME}.target": "",${oauth_default}
	}
}

func validate${field}Integration(cfg *IntegrationsConfig) error {
	cfg.${field}.Target = strings.TrimSpace(cfg.${field}.Target)
	if err := validateGRPCIntegrationTarget(cfg.${field}.Target, "integrations.${NAME}.target"); err != nil {
		return err
	}
${oauth_validate}
	return nil
}

func validateGRPCIntegrationTarget(raw, key string) error {
	endpoint, err := url.Parse(raw)
	if err != nil || endpoint == nil || endpoint.Scheme != "dns" || endpoint.Opaque != "" ||
		endpoint.User != nil || endpoint.RawQuery != "" || endpoint.ForceQuery || endpoint.Fragment != "" {
		return fmt.Errorf("%w: %s must be dns:///hostname:443", ErrValidate, key)
	}
	host := strings.TrimPrefix(endpoint.Path, "/")
	if host == "" || host != endpoint.Path[1:] && endpoint.Path != "/"+host {
		return fmt.Errorf("%w: %s must be dns:///hostname:443", ErrValidate, key)
	}
	hostname, port, err := net.SplitHostPort(host)
	if err != nil || port != "443" || hostname == "" || net.ParseIP(hostname) != nil ||
		strings.Contains(hostname, "/") {
		return fmt.Errorf("%w: %s must be dns:///hostname:443", ErrValidate, key)
	}
	return nil
}
EOF
	if [[ "${AUTH}" == "oauth2-client-credentials" ]]; then
		write_named_oauth_validator
	fi
	write_config_test "${field}" "grpc"
}

write_named_oauth_validator() {
	local dest="${ROOT_DIR}/internal/config/${NAME}_integration_config.go"
	if grep -q 'func validateNamedOAuthConfig' "${ROOT_DIR}/internal/config/"*_integration_config.go 2>/dev/null; then
		return
	fi
	if ! grep -q 'type OutboundAuthConfig struct' "${dest}"; then
		cat >>"${dest}" <<'EOF'

type OutboundAuthConfig struct {
	TokenURL     string `koanf:"token_url"`
	ClientID     string `koanf:"client_id"`
	ClientSecret string `koanf:"client_secret"`
	Scopes       string `koanf:"scopes"`
	Audience     string `koanf:"audience"`
	Resource     string `koanf:"resource"`
}
EOF
	fi
	cat >>"${dest}" <<'EOF'

func validateNamedOAuthConfig(cfg *OutboundAuthConfig, prefix string) error {
	cfg.TokenURL = strings.TrimSpace(cfg.TokenURL)
	cfg.Scopes = strings.Join(strings.Fields(cfg.Scopes), " ")
	if cfg.ClientID == "" {
		return fmt.Errorf("%w: %s.client_id is required", ErrValidate, prefix)
	}
	if strings.TrimSpace(cfg.ClientSecret) == "" {
		return fmt.Errorf("%w: %s.client_secret is required", ErrValidate, prefix)
	}
	endpoint, err := url.Parse(cfg.TokenURL)
	if err != nil || endpoint == nil || !endpoint.IsAbs() || endpoint.Opaque != "" ||
		!strings.EqualFold(endpoint.Scheme, "https") || endpoint.Host == "" || endpoint.Hostname() == "" ||
		endpoint.User != nil || endpoint.RawQuery != "" || endpoint.ForceQuery || endpoint.Fragment != "" {
		return fmt.Errorf("%w: %s.token_url must be an absolute HTTPS URL", ErrValidate, prefix)
	}
	endpoint.Scheme = "https"
	endpoint.Host = strings.ToLower(endpoint.Host)
	cfg.TokenURL = endpoint.String()
	if cfg.Audience != "" && cfg.Resource != "" {
		return fmt.Errorf("%w: %s.audience and %s.resource are mutually exclusive", ErrValidate, prefix, prefix)
	}
	return nil
}
EOF
}

write_config_test() {
	local field="$1"
	local kind="$2"
	local dest="${ROOT_DIR}/internal/config/${NAME}_integration_config_test.go"
	cat >"${dest}" <<EOF
package config

import (
	"errors"
	"strings"
	"testing"
)

func Test${field}IntegrationRejectsInvalidTarget(t *testing.T) {
	t.Parallel()
	cfg := IntegrationsConfig{}
	if err := validate${field}Integration(&cfg); err == nil {
		t.Fatal("validate${field}Integration() error = nil, want rejection")
	} else if !errors.Is(err, ErrValidate) {
		t.Fatalf("validate${field}Integration() error = %v, want ErrValidate", err)
	} else if strings.Contains(err.Error(), "canary") {
		t.Fatal("validation error disclosed a canary")
	}
}
EOF
	if [[ "${kind}" == "http" ]]; then
		local extra=""
		if [[ "${TARGET}" == "private-https" ]]; then
			extra+=$'\n\t\tPrivateDNSSuffix: "svc.cluster.local",'
		fi
		if [[ "${AUTH}" == "oauth2-client-credentials" ]]; then
			extra+=$'\n\t\tOAuth: OutboundAuthConfig{TokenURL: "https://auth.example.com/oauth/token", ClientID: "client", ClientSecret: "secret"},'
		fi
		cat >>"${dest}" <<EOF

func Test${field}IntegrationAcceptsHTTPS(t *testing.T) {
	t.Parallel()
	cfg := IntegrationsConfig{${field}: ${field}IntegrationConfig{
		BaseURL: "https://billing.example.com",${extra}
	}}
	if err := validate${field}Integration(&cfg); err != nil {
		t.Fatalf("validate${field}Integration() error = %v", err)
	}
}
EOF
	else
		local extra=""
		if [[ "${AUTH}" == "oauth2-client-credentials" ]]; then
			extra=$'\n\t\tOAuth: OutboundAuthConfig{TokenURL: "https://auth.example.com/oauth/token", ClientID: "client", ClientSecret: "secret"},'
		fi
		cat >>"${dest}" <<EOF

func Test${field}IntegrationAcceptsDNSTarget(t *testing.T) {
	t.Parallel()
	cfg := IntegrationsConfig{${field}: ${field}IntegrationConfig{
		Target: "dns:///identity.example.com:443",${extra}
	}}
	if err := validate${field}Integration(&cfg); err != nil {
		t.Fatalf("validate${field}Integration() error = %v", err)
	}
}
EOF
	fi
}

write_http_adapter() {
	local field="$1"
	local module="$2"
	local dir="${ROOT_DIR}/internal/infra/${NAME}"
	mkdir -p "${dir}/internal/openapi"
	cat >"${dir}/doc.go" <<EOF
// Package ${NAME} is the concrete outbound adapter for the ${NAME} integration.
//
// The OpenAPI contract at ${CONTRACT} is the source of truth. Generated client
// types stay in the internal/openapi subpackage. This package owns target
// construction, optional retained OAuth binding, and Close. It exposes no
// provider operation.
package ${NAME}
EOF
	cat >"${dir}/internal/openapi/doc.go" <<EOF
package openapi

//go:generate bash ../../../../../scripts/run-go-tool.sh oapi-codegen -config oapi-codegen.yaml ../../../../../${CONTRACT}
EOF
	cat >"${dir}/internal/openapi/oapi-codegen.yaml" <<EOF
package: openapi
output: client.gen.go
generate:
  models: true
  client: true
EOF
	local ctor="httpclient.NewExternalHTTPS(cfg.BaseURL)"
	local extra_field=""
	if [[ "${TARGET}" == "private-https" ]]; then
		ctor="httpclient.NewPrivateHTTPS(cfg.BaseURL, cfg.PrivateDNSSuffix)"
		extra_field=$'\n\tPrivateDNSSuffix string'
	fi
	local auth_import=""
	local auth_field=""
	local auth_bind=""
	local auth_close=""
	local doer="transport"
	if [[ "${AUTH}" == "oauth2-client-credentials" ]]; then
		auth_import=$'\n\t"'"${module}"'/internal/infra/oauth2clientcredentials"'
		auth_field=$'\n\tauth *oauth2clientcredentials.Client'
		auth_bind=$(
			cat <<EOF
	auth, err := oauth2clientcredentials.New(oauth2clientcredentials.Config{
		TokenURL:     cfg.OAuth.TokenURL,
		ClientID:     cfg.OAuth.ClientID,
		ClientSecret: cfg.OAuth.ClientSecret,
		Scopes:       strings.Fields(cfg.OAuth.Scopes),
		Audience:     cfg.OAuth.Audience,
		Resource:     cfg.OAuth.Resource,
	})
	if err != nil {
		transport.CloseIdleConnections()
		return nil, fmt.Errorf("build ${NAME} client: %w", err)
	}
	authenticated, err := auth.HTTP(transport)
	if err != nil {
		_ = auth.Close(context.Background())
		transport.CloseIdleConnections()
		return nil, fmt.Errorf("build ${NAME} client: %w", err)
	}
	doer := authenticated
EOF
		)
		doer="doer"
		auth_close=$(
			cat <<'EOF'
	if c.auth != nil {
		if err := c.auth.Close(ctx); err != nil {
			c.transport.CloseIdleConnections()
			return err
		}
	}
EOF
		)
	fi
	cat >"${dir}/client.go" <<EOF
package ${NAME}

import (
	"context"
	"fmt"$(if [[ "${AUTH}" == "oauth2-client-credentials" ]]; then printf '\n\t"strings"'; fi)

	"${module}/internal/infra/${NAME}/internal/openapi"
	"${module}/internal/infra/httpclient"${auth_import}
)

// Config is the validated runtime tuple for this integration.
type Config struct {
	BaseURL string${extra_field}
$(if [[ "${AUTH}" == "oauth2-client-credentials" ]]; then
		cat <<'INNER'
	OAuth struct {
		TokenURL     string
		ClientID     string
		ClientSecret string
		Scopes       string
		Audience     string
		Resource     string
	}
INNER
	fi)
}

// Client owns the retained transport and generated binding. It has no operation method.
type Client struct {
	generated *openapi.Client
	transport *httpclient.Client${auth_field}
}

// New constructs the adapter without provider I/O.
func New(cfg Config) (*Client, error) {
	transport, err := ${ctor}
	if err != nil {
		return nil, fmt.Errorf("build ${NAME} client: %w", err)
	}
	${auth_bind}
	generated, err := openapi.NewClient(transport.BaseURL(), openapi.WithHTTPClient(${doer}))
	if err != nil {
		$(if [[ "${AUTH}" == "oauth2-client-credentials" ]]; then echo '_ = auth.Close(context.Background())'; fi)
		transport.CloseIdleConnections()
		return nil, fmt.Errorf("build ${NAME} client: %w", err)
	}
	return &Client{
		generated: generated,
		transport: transport,
		$(if [[ "${AUTH}" == "oauth2-client-credentials" ]]; then echo 'auth: auth,'; fi)
	}, nil
}

// Close retires authentication first, then idle HTTP connections.
func (c *Client) Close(ctx context.Context) error {
	if c == nil {
		return nil
	}
${auth_close}
	if c.transport != nil {
		c.transport.CloseIdleConnections()
	}
	return nil
}
EOF
	write_adapter_test "${dir}" "http"
}

write_grpc_adapter() {
	local field="$1"
	local module="$2"
	local dir="${ROOT_DIR}/internal/infra/${NAME}"
	mkdir -p "${dir}"
	cat >"${dir}/doc.go" <<EOF
// Package ${NAME} is the concrete outbound adapter for the ${NAME} integration.
//
// The Protobuf contract below ${CONTRACT%/*}/ is the source of truth. Generated
// types stay under internal/gen/proto/external/${NAME}. This package owns the
// bounded connection and optional retained OAuth binding. It selects no service
// stub and exposes no provider operation.
package ${NAME}
EOF
	if [[ "${AUTH}" == "oauth2-client-credentials" ]]; then
		cat >"${dir}/client.go" <<EOF
package ${NAME}

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"strings"

	"${module}/internal/infra/grpcclient"
	"${module}/internal/infra/oauth2clientcredentials"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type Config struct {
	Target string
	OAuth  struct {
		TokenURL     string
		ClientID     string
		ClientSecret string
		Scopes       string
		Audience     string
		Resource     string
	}
}

type Client struct {
	conn   grpc.ClientConnInterface
	closer interface{ Close() error }
	auth   *oauth2clientcredentials.Client
}

func New(cfg Config) (*Client, error) {
	hostname, err := grpcHostname(cfg.Target)
	if err != nil {
		return nil, fmt.Errorf("build ${NAME} client: %w", err)
	}
	creds := credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS12, ServerName: hostname})
	auth, err := oauth2clientcredentials.New(oauth2clientcredentials.Config{
		TokenURL:     cfg.OAuth.TokenURL,
		ClientID:     cfg.OAuth.ClientID,
		ClientSecret: cfg.OAuth.ClientSecret,
		Scopes:       strings.Fields(cfg.OAuth.Scopes),
		Audience:     cfg.OAuth.Audience,
		Resource:     cfg.OAuth.Resource,
	})
	if err != nil {
		return nil, fmt.Errorf("build ${NAME} client: %w", err)
	}
	authenticated, err := auth.GRPC(grpcclient.Config{Target: cfg.Target}, grpcclient.Options{TransportCredentials: creds})
	if err != nil {
		_ = auth.Close(context.Background())
		return nil, fmt.Errorf("build ${NAME} client: %w", err)
	}
	return &Client{conn: authenticated, closer: authenticated, auth: auth}, nil
}

func grpcHostname(raw string) (string, error) {
	endpoint, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	host := strings.TrimPrefix(endpoint.Path, "/")
	hostname, port, splitErr := net.SplitHostPort(host)
	if splitErr != nil || port != "443" || hostname == "" {
		return "", fmt.Errorf("target must be dns:///hostname:443")
	}
	return hostname, nil
}

func (c *Client) Close(ctx context.Context) error {
	if c == nil {
		return nil
	}
	if c.auth != nil {
		if err := c.auth.Close(ctx); err != nil {
			if c.closer != nil {
				_ = c.closer.Close()
			}
			return err
		}
	}
	if c.closer != nil {
		return c.closer.Close()
	}
	return nil
}
EOF
	else
		cat >"${dir}/client.go" <<EOF
package ${NAME}

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"strings"

	"${module}/internal/infra/grpcclient"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type Config struct {
	Target string
}

type Client struct {
	conn   *grpc.ClientConn
	closer interface{ Close() error }
}

func New(cfg Config) (*Client, error) {
	hostname, err := grpcHostname(cfg.Target)
	if err != nil {
		return nil, fmt.Errorf("build ${NAME} client: %w", err)
	}
	creds := credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS12, ServerName: hostname})
	conn, err := grpcclient.New(grpcclient.Config{Target: cfg.Target}, grpcclient.Options{TransportCredentials: creds})
	if err != nil {
		return nil, fmt.Errorf("build ${NAME} client: %w", err)
	}
	return &Client{conn: conn, closer: conn}, nil
}

func grpcHostname(raw string) (string, error) {
	endpoint, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	host := strings.TrimPrefix(endpoint.Path, "/")
	hostname, port, splitErr := net.SplitHostPort(host)
	if splitErr != nil || port != "443" || hostname == "" {
		return "", fmt.Errorf("target must be dns:///hostname:443")
	}
	return hostname, nil
}

func (c *Client) Close(ctx context.Context) error {
	if c == nil {
		return nil
	}
	if c.closer != nil {
		return c.closer.Close()
	}
	return nil
}
EOF
	fi
	write_adapter_test "${dir}" "grpc"
}

write_adapter_test() {
	local dir="$1"
	local kind="$2"
	cat >"${dir}/client_test.go" <<EOF
package ${NAME}

import (
	"context"
	"testing"
)

func TestNewRejectsEmptyTarget(t *testing.T) {
	t.Parallel()
	if _, err := New(Config{}); err == nil {
		t.Fatal("New() error = nil, want rejection")
	}
}

func TestCloseIdempotent(t *testing.T) {
	t.Parallel()
	var client *Client
	if err := client.Close(context.Background()); err != nil {
		t.Fatalf("nil Close() error = %v", err)
	}
}
EOF
}

write_bootstrap() {
	local field="$1"
	local module="$2"
	local dest="${ROOT_DIR}/cmd/service/internal/bootstrap/startup_${NAME}.go"
	if [[ "${TRANSPORT}" == "http" ]]; then
		cat >"${dest}" <<EOF
package bootstrap

import (
	"fmt"

	"github.com/example/go-service-template-rest/internal/config"
	"${module}/internal/infra/${NAME}"
)

func init${field}(cfg config.${field}IntegrationConfig) (*${NAME}.Client, error) {
	client, err := ${NAME}.New(${NAME}.Config{
		BaseURL: cfg.BaseURL,
$(if [[ "${TARGET}" == "private-https" ]]; then echo '		PrivateDNSSuffix: cfg.PrivateDNSSuffix,'; fi)
$(if [[ "${AUTH}" == "oauth2-client-credentials" ]]; then
			cat <<'INNER'
		OAuth: struct {
			TokenURL     string
			ClientID     string
			ClientSecret string
			Scopes       string
			Audience     string
			Resource     string
		}{
			TokenURL:     cfg.OAuth.TokenURL,
			ClientID:     cfg.OAuth.ClientID,
			ClientSecret: cfg.OAuth.ClientSecret,
			Scopes:       cfg.OAuth.Scopes,
			Audience:     cfg.OAuth.Audience,
			Resource:     cfg.OAuth.Resource,
		},
INNER
		fi)
	})
	if err != nil {
		return nil, fmt.Errorf("init ${NAME}: %w", err)
	}
	return client, nil
}
EOF
	else
		cat >"${dest}" <<EOF
package bootstrap

import (
	"fmt"

	"github.com/example/go-service-template-rest/internal/config"
	"${module}/internal/infra/${NAME}"
)

func init${field}(cfg config.${field}IntegrationConfig) (*${NAME}.Client, error) {
	client, err := ${NAME}.New(${NAME}.Config{
		Target: cfg.Target,
$(if [[ "${AUTH}" == "oauth2-client-credentials" ]]; then
			cat <<'INNER'
		OAuth: struct {
			TokenURL     string
			ClientID     string
			ClientSecret string
			Scopes       string
			Audience     string
			Resource     string
		}{
			TokenURL:     cfg.OAuth.TokenURL,
			ClientID:     cfg.OAuth.ClientID,
			ClientSecret: cfg.OAuth.ClientSecret,
			Scopes:       cfg.OAuth.Scopes,
			Audience:     cfg.OAuth.Audience,
			Resource:     cfg.OAuth.Resource,
		},
INNER
		fi)
	})
	if err != nil {
		return nil, fmt.Errorf("init ${NAME}: %w", err)
	}
	return client, nil
}
EOF
	fi
	# Fix module path in bootstrap import of config - use actual module
	python3 - "${dest}" "${module}" <<'PY'
from pathlib import Path
import sys
path = Path(sys.argv[1])
module = sys.argv[2]
text = path.read_text().replace("github.com/example/go-service-template-rest/internal/config", f"{module}/internal/config")
path.write_text(text)
PY
	cat >"${ROOT_DIR}/cmd/service/internal/bootstrap/startup_${NAME}_test.go" <<EOF
package bootstrap

import (
	"testing"

	"${module}/internal/config"
)

func TestInit${field}RejectsEmptyConfig(t *testing.T) {
	t.Parallel()
	if _, err := init${field}(config.${field}IntegrationConfig{}); err == nil {
		t.Fatal("init${field}() error = nil, want rejection")
	}
}
EOF
	wire_run_go "${field}"
}

wire_run_go() {
	local field="$1"
	local dest="${ROOT_DIR}/cmd/service/internal/bootstrap/run.go"
	python3 - "${dest}" "${NAME}" "${field}" <<'PY'
from pathlib import Path
import sys
path = Path(sys.argv[1])
name, field = sys.argv[2], sys.argv[3]
text = path.read_text()
marker = f"{name}Client"
if marker not in text:
    construct = f'''
	{name}Client, err := init{field}(bootstrap.cfg.Integrations.{field})
	if err != nil {{
		return err
	}}
	{name}Closed := false
	defer func() {{
		if !{name}Closed {{
			_ = {name}Client.Close(shutdown.stage(signalCtx, dependencyCloseTimeout))
		}}
	}}()
'''
    anchor = "\tdependencies, err := wiring.dependencies(startupCtx, bootstrap)\n"
    if anchor not in text:
        raise SystemExit("run.go missing dependencies construction anchor")
    text = text.replace(anchor, construct + "\n" + anchor, 1)
    close_anchor = "\tbackgroundErr := supervisor.Shutdown(backgroundCtx)\n\twiring.lifecycle(runtimeLifecycleBackgroundJoined)\n"
    close_insert = close_anchor + f"\tif !{name}Closed {{\n\t\t_ = {name}Client.Close(shutdown.stage(signalCtx, dependencyCloseTimeout))\n\t\t{name}Closed = true\n\t}}\n"
    if close_anchor not in text:
        raise SystemExit("run.go missing background join anchor")
    text = text.replace(close_anchor, close_insert, 1)
path.write_text(text)
PY
}

write_docs() {
	mkdir -p "${ROOT_DIR}/docs/integrations"
	cat >"${ROOT_DIR}/docs/integrations/${NAME}.md" <<EOF
# ${NAME} integration

Contract: \`${CONTRACT}\`
Transport: \`${TRANSPORT}\`
$(if [[ "${TRANSPORT}" == "http" ]]; then echo "Target class: \`${TARGET}\`"; fi)
Authentication: \`${AUTH}\`

## Runtime inputs

$(if [[ "${TRANSPORT}" == "http" ]]; then
		echo "- \`integrations.${NAME}.base_url\`"
		if [[ "${TARGET}" == "private-https" ]]; then
			echo "- \`integrations.${NAME}.private_dns_suffix\`"
		fi
	else
		echo "- \`integrations.${NAME}.target\` (\`dns:///hostname:443\`)"
	fi)
$(if [[ "${AUTH}" == "oauth2-client-credentials" ]]; then
		cat <<INNER
- \`integrations.${NAME}.oauth.token_url\`
- \`integrations.${NAME}.oauth.client_id\`
- \`integrations.${NAME}.oauth.client_secret\` (environment-only)
- \`integrations.${NAME}.oauth.scopes\`
- \`integrations.${NAME}.oauth.audience\`
- \`integrations.${NAME}.oauth.resource\`
INNER
	fi)

The scaffold constructs transport and generated bindings only. A later manual
operation in this adapter must own request/response mapping, errors, budget,
and retry eligibility. The initializer does not claim provider compatibility.
EOF
}

render_examples() {
	local yaml="${ROOT_DIR}/env/config/local.yaml"
	local envf="${ROOT_DIR}/env/.env.example"
	if [[ "${AUTH}" == "oauth2-client-credentials" ]] && grep -q 'outbound_auth:' "${yaml}"; then
		python3 - "${yaml}" "${NAME}" <<'PY'
from pathlib import Path
import re, sys
path = Path(sys.argv[1])
name = sys.argv[2]
text = path.read_text()
block = f"""integrations:
  {name}:
    base_url: ""
    oauth:
      token_url: ""
      client_id: ""
      client_secret: ""
      scopes: ""
      audience: ""
      resource: ""
"""
if "transport" == "skip":
    pass
text = re.sub(r"# profile:outbound-auth-oauth2-client-credentials:start.*?outbound_auth:\n(?:  .*\n)+# profile:outbound-auth-oauth2-client-credentials:end\n", block, text, count=1, flags=re.S)
if f"integrations:" not in text:
    # fallback: append
    text += "\n" + block
path.write_text(text)
PY
		# The python above is too fragile because TRANSPORT isn't in that script.
		# Do a simpler replacement below.
	fi
	python3 - "${yaml}" "${envf}" "${NAME}" "${TRANSPORT}" "${TARGET}" "${AUTH}" <<'PY'
from pathlib import Path
import re, sys
yaml_path, env_path, name, transport, target, auth = sys.argv[1:7]
yaml = Path(yaml_path)
env = Path(env_path)
ytext = yaml.read_text()
etext = env.read_text()

if transport == "http":
    fields = [f"    base_url: \"\""]
    if target == "private-https":
        fields.append(f"    private_dns_suffix: \"\"")
    env_keys = [f"APP__INTEGRATIONS__{name.upper()}__BASE_URL="]
    if target == "private-https":
        env_keys.append(f"APP__INTEGRATIONS__{name.upper()}__PRIVATE_DNS_SUFFIX=")
else:
    fields = [f"    target: \"\""]
    env_keys = [f"APP__INTEGRATIONS__{name.upper()}__TARGET="]

if auth == "oauth2-client-credentials":
    fields += [
        "    oauth:",
        "      token_url: \"\"",
        "      client_id: \"\"",
        "      client_secret: \"\"",
        "      scopes: \"\"",
        "      audience: \"\"",
        "      resource: \"\"",
    ]
    prefix = f"APP__INTEGRATIONS__{name.upper()}__OAUTH__"
    env_keys += [
        f"{prefix}TOKEN_URL=",
        f"{prefix}CLIENT_ID=",
        f"{prefix}CLIENT_SECRET=",
        f"{prefix}SCOPES=",
        f"{prefix}AUDIENCE=",
        f"{prefix}RESOURCE=",
    ]

section = "integrations:\n  " + name + ":\n" + "\n".join(fields) + "\n"
env_block = "\n".join(env_keys) + "\n"

singleton_yaml = re.compile(
    r"# profile:outbound-auth-oauth2-client-credentials:start.*?outbound_auth:\n(?:  .*\n)+# profile:outbound-auth-oauth2-client-credentials:end\n",
    re.S,
)
singleton_env = re.compile(
    r"# profile:outbound-auth-oauth2-client-credentials:start.*?APP__OUTBOUND_AUTH__RESOURCE=\n# profile:outbound-auth-oauth2-client-credentials:end\n",
    re.S,
)

if auth == "oauth2-client-credentials" and "outbound_auth:" in ytext:
    ytext = singleton_yaml.sub(section, ytext, count=1)
    if f"  {name}:" not in ytext:
        ytext += "\n" + section
    etext = singleton_env.sub(env_block, etext, count=1)
    if f"APP__INTEGRATIONS__{name.upper()}__" not in etext:
        etext += "\n" + env_block
else:
    if f"  {name}:" not in ytext:
        if "\nintegrations:\n" in ytext:
            ytext = ytext.replace("\nintegrations:\n", "\nintegrations:\n  " + name + ":\n" + "\n".join(fields) + "\n", 1)
        else:
            ytext += "\n" + section
    if f"APP__INTEGRATIONS__{name.upper()}__" not in etext:
        etext += "\n" + env_block

yaml.write_text(ytext)
env.write_text(etext)
PY
}

add_golangci_grpc_rule() {
	local dest="${ROOT_DIR}/.golangci.yml"
	local module
	module="$(module_path)"
	python3 - "${dest}" "${module}" "${NAME}" <<'PY'
from pathlib import Path
import sys
path = Path(sys.argv[1])
module, name = sys.argv[2], sys.argv[3]
text = path.read_text()
rule = f"""        {name}_grpc_generated_adapter_only:
          list-mode: lax
          files:
            - "**/*.go"
            - "!**/internal/infra/{name}/**/*.go"
            - "!**/internal/gen/proto/external/{name}/**/*.go"
          deny:
            - pkg: {module}/internal/gen/proto/external/{name}
              desc: generated {name} gRPC types stay inside the named adapter.
"""
if f"{name}_grpc_generated_adapter_only:" in text:
    raise SystemExit(0)
anchor = "    depguard:\n      rules:\n"
if anchor not in text:
    raise SystemExit(".golangci.yml missing depguard rules")
path.write_text(text.replace(anchor, anchor + rule, 1))
PY
}

add_snapshot_keys() {
	local src exp
	src="$(mktemp)"
	exp="$(mktemp)"
	if [[ "${TRANSPORT}" == "http" ]]; then
		printf '\t\t"integrations.%s.base_url": "https://%s.snapshot.example",\n' "${NAME}" "${NAME}" >"${src}"
		printf '\t\t"integrations.%s.base_url": "https://%s.snapshot.example",\n' "${NAME}" "${NAME}" >"${exp}"
		if [[ "${TARGET}" == "private-https" ]]; then
			printf '\t\t"integrations.%s.private_dns_suffix": "svc.cluster.local",\n' "${NAME}" >>"${src}"
			printf '\t\t"integrations.%s.private_dns_suffix": "svc.cluster.local",\n' "${NAME}" >>"${exp}"
		fi
	else
		printf '\t\t"integrations.%s.target": "dns:///%s.snapshot.example:443",\n' "${NAME}" "${NAME}" >"${src}"
		printf '\t\t"integrations.%s.target": "dns:///%s.snapshot.example:443",\n' "${NAME}" "${NAME}" >"${exp}"
	fi
	if [[ "${AUTH}" == "oauth2-client-credentials" ]]; then
		cat >>"${src}" <<EOF
		"integrations.${NAME}.oauth.token_url":     "https://auth.snapshot.example/oauth/token",
		"integrations.${NAME}.oauth.client_id":     " snapshot-client:id ",
		"integrations.${NAME}.oauth.client_secret": " snapshot-client-secret ",
		"integrations.${NAME}.oauth.scopes":        "snapshot.read snapshot.write",
		"integrations.${NAME}.oauth.resource":      "https://resource.snapshot.example",
		"integrations.${NAME}.oauth.audience":      "",
EOF
		cat >>"${exp}" <<EOF
		"integrations.${NAME}.oauth.token_url":     "https://auth.snapshot.example/oauth/token",
		"integrations.${NAME}.oauth.client_id":     " snapshot-client:id ",
		"integrations.${NAME}.oauth.client_secret": " snapshot-client-secret ",
		"integrations.${NAME}.oauth.scopes":        "snapshot.read snapshot.write",
		"integrations.${NAME}.oauth.resource":      "https://resource.snapshot.example",
		"integrations.${NAME}.oauth.audience":      "",
EOF
	fi
	python3 - "${ROOT_DIR}/internal/config/snapshot_contract_test.go" "${src}" "${exp}" <<'PY'
from pathlib import Path
import sys
path = Path(sys.argv[1])
src = Path(sys.argv[2]).read_text()
exp = Path(sys.argv[3]).read_text()
text = path.read_text()
for fn, block in (
    ("func sentinelConfigSourceValues() map[string]any {", src),
    ("func expectedSentinelSnapshotValues() map[string]any {", exp),
):
    start = text.find(fn)
    if start < 0:
        raise SystemExit(f"missing {fn}")
    marker = "return map[string]any{"
    brace = text.find(marker, start)
    if brace < 0:
        raise SystemExit(f"missing map literal in {fn}")
    insert_at = text.find("\n", brace) + 1
    end = text.find("\n}", insert_at)
    region = text[insert_at:end]
    if block.strip() and block.strip() not in region:
        text = text[:insert_at] + block + text[insert_at:]
path.write_text(text)
PY
	rm -f "${src}" "${exp}"
}

install_test_env_tuple() {
	local dest="${ROOT_DIR}/internal/config/testhelpers_test.go"
	local env_key env_val
	if [[ "${TRANSPORT}" == "http" ]]; then
		env_key="APP__INTEGRATIONS__$(printf '%s' "${NAME}" | tr '[:lower:]' '[:upper:]')__BASE_URL"
		env_val="https://${NAME}.example.com"
	else
		env_key="APP__INTEGRATIONS__$(printf '%s' "${NAME}" | tr '[:lower:]' '[:upper:]')__TARGET"
		env_val="dns:///${NAME}.example.com:443"
	fi
	if [[ -f "${dest}" ]] && ! grep -q "${env_key}" "${dest}"; then
		python3 - "${dest}" "${env_key}" "${env_val}" <<'PY'
from pathlib import Path
import sys
path = Path(sys.argv[1])
key, value = sys.argv[2], sys.argv[3]
text = path.read_text()
needle = '\tt.Setenv("APP__APP__ENV", "local")\n'
insert = needle + f'\tt.Setenv("{key}", "{value}")\n'
if key in text:
    raise SystemExit(0)
if needle not in text:
    raise SystemExit("testhelpers_test.go missing APP__APP__ENV anchor")
path.write_text(text.replace(needle, insert, 1))
PY
	fi
	dest="${ROOT_DIR}/internal/config/configtest/configtest.go"
	if [[ -f "${dest}" ]] && ! grep -q "${env_key}" "${dest}"; then
		python3 - "${dest}" "${env_key}" "${env_val}" <<'PY'
from pathlib import Path
import sys
path = Path(sys.argv[1])
key, value = sys.argv[2], sys.argv[3]
text = path.read_text()
needle = '\ttb.Setenv("APP__APP__ENV", "local")\n'
insert = needle + f'\ttb.Setenv("{key}", "{value}")\n'
if key in text:
    raise SystemExit(0)
if needle not in text:
    raise SystemExit("configtest.go missing APP__APP__ENV anchor")
path.write_text(text.replace(needle, insert, 1))
PY
	fi
	dest="${ROOT_DIR}/cmd/service/internal/bootstrap/run_test.go"
	if [[ -f "${dest}" ]] && ! grep -q "${env_key}" "${dest}"; then
		python3 - "${dest}" "${env_key}" "${env_val}" <<'PY'
from pathlib import Path
import sys
path = Path(sys.argv[1])
key, value = sys.argv[2], sys.argv[3]
text = path.read_text()
if key in text:
    raise SystemExit(0)
needles = [
    "\t}\n}\n\nfunc testRuntimeWiring()",
    "\t}\n}\n",
]
for needle in needles:
    if needle in text and "func resetShutdownConfigEnv" in text:
        insert = needle.replace("\t}\n}\n", f'\t}}\n\tt.Setenv("{key}", "{value}")\n}}\n', 1)
        path.write_text(text.replace(needle, insert, 1))
        raise SystemExit(0)
raise SystemExit(0)
PY
	fi
	if [[ "${AUTH}" == "oauth2-client-credentials" ]]; then
		migrate_oauth_helpers "$(field_name "${NAME}")"
	fi
}

migrate_oauth_helpers() {
	local field="$1"
	# Replace remaining singleton helper bodies with named keys when the helper file still exists.
	local dest="${ROOT_DIR}/internal/config/testhelpers_test.go"
	if [[ -f "${dest}" ]] && grep -q 'APP__OUTBOUND_AUTH__' "${dest}"; then
		python3 - "${dest}" "${NAME}" <<'PY'
from pathlib import Path
import sys
path = Path(sys.argv[1])
name = sys.argv[2].upper()
text = path.read_text()
text = text.replace("APP__OUTBOUND_AUTH__", f"APP__INTEGRATIONS__{name}__OAUTH__")
path.write_text(text)
PY
	fi
	dest="${ROOT_DIR}/internal/config/configtest/configtest.go"
	if [[ -f "${dest}" ]] && grep -q 'APP__OUTBOUND_AUTH__' "${dest}"; then
		python3 - "${dest}" "${NAME}" <<'PY'
from pathlib import Path
import sys
path = Path(sys.argv[1])
name = sys.argv[2].upper()
text = path.read_text()
text = text.replace("APP__OUTBOUND_AUTH__", f"APP__INTEGRATIONS__{name}__OAUTH__")
path.write_text(text)
PY
	fi
	for dest in \
		"${ROOT_DIR}/cmd/service/internal/bootstrap/run_test.go" \
		"${ROOT_DIR}/internal/config/secret_policy_test.go" \
		"${ROOT_DIR}/internal/config/load_environment_test.go"; do
		if [[ -f "${dest}" ]] && grep -qE 'APP__OUTBOUND_AUTH__|outbound_auth\.|OutboundAuth|setOutboundAuthTestEnv' "${dest}"; then
			python3 - "${dest}" "${NAME}" "$(field_name "${NAME}")" <<'PY'
from pathlib import Path
import sys
path = Path(sys.argv[1])
name, field = sys.argv[2], sys.argv[3]
text = path.read_text()
text = text.replace("APP__OUTBOUND_AUTH__", f"APP__INTEGRATIONS__{name.upper()}__OAUTH__")
text = text.replace("outbound_auth.", f"integrations.{name}.oauth.")
text = text.replace("cfg.OutboundAuth", f"cfg.Integrations.{field}.OAuth")
path.write_text(text)
PY
		fi
	done
}

generate_clients() {
	if [[ "${TRANSPORT}" == "http" ]]; then
		(
			cd "${ROOT_DIR}"
			go generate "./internal/infra/${NAME}/internal/openapi"
		)
	else
		(
			cd "${ROOT_DIR}"
			make proto-generate
		)
	fi
}

format_manual_go() {
	local files=(
		"${ROOT_DIR}/internal/config/${NAME}_integration_config.go"
		"${ROOT_DIR}/internal/config/${NAME}_integration_config_test.go"
		"${ROOT_DIR}/internal/config/integrations_config.go"
		"${ROOT_DIR}/internal/config/types.go"
		"${ROOT_DIR}/internal/config/defaults.go"
		"${ROOT_DIR}/internal/config/validate.go"
		"${ROOT_DIR}/internal/config/snapshot_contract_test.go"
		"${ROOT_DIR}/internal/config/testhelpers_test.go"
		"${ROOT_DIR}/internal/infra/${NAME}/client.go"
		"${ROOT_DIR}/internal/infra/${NAME}/client_test.go"
		"${ROOT_DIR}/internal/infra/${NAME}/doc.go"
		"${ROOT_DIR}/cmd/service/internal/bootstrap/startup_${NAME}.go"
		"${ROOT_DIR}/cmd/service/internal/bootstrap/startup_${NAME}_test.go"
		"${ROOT_DIR}/cmd/service/internal/bootstrap/run.go"
		"${ROOT_DIR}/cmd/service/internal/bootstrap/run_test.go"
	)
	local present=()
	local file
	for file in "${files[@]}"; do
		[[ -f "${file}" ]] && present+=("${file}")
	done
	if ((${#present[@]} > 0)); then
		(
			cd "${ROOT_DIR}"
			bash ./scripts/run-go-tool.sh goimports -w "${present[@]}"
			bash ./scripts/run-go-tool.sh gofumpt -w "${present[@]}"
		)
	fi
}

admit_stage() {
	(
		cd "${ROOT_DIR}"
		go test -vet=off -count=1 -run '^$' \
			"./internal/config" \
			"./internal/infra/${NAME}" \
			"./cmd/service/internal/bootstrap"
	)
}

changed_paths() {
	git -C "${ROOT_DIR}" diff --name-only --no-renames HEAD
	git -C "${ROOT_DIR}" ls-files --others --exclude-standard
}

path_allowed_initial() {
	local path="$1"
	case "${path}" in
	integrations/"${NAME}".toml | \
		internal/infra/"${NAME}"/* | \
		internal/config/"${NAME}"_integration_config.go | \
		internal/config/"${NAME}"_integration_config_test.go | \
		internal/config/integrations_config.go | \
		internal/config/types.go | \
		internal/config/defaults.go | \
		internal/config/validate.go | \
		internal/config/snapshot_contract_test.go | \
		internal/config/testhelpers_test.go | \
		internal/config/load_environment_test.go | \
		internal/config/secret_policy_test.go | \
		internal/config/configtest/configtest.go | \
		internal/config/outbound_auth_config.go | \
		internal/config/outbound_auth_config_test.go | \
		cmd/service/internal/bootstrap/startup_"${NAME}".go | \
		cmd/service/internal/bootstrap/startup_"${NAME}"_test.go | \
		cmd/service/internal/bootstrap/run.go | \
		cmd/service/internal/bootstrap/run_test.go | \
		cmd/service/internal/bootstrap/startup_dependencies_test.go | \
		docs/integrations/"${NAME}".md | \
		docs/outbound-machine-authentication.md | \
		env/config/local.yaml | \
		env/.env.example | \
		.golangci.yml | \
		internal/gen/proto/external/"${NAME}"/*)
		return 0
		;;
	esac
	return 1
}

path_allowed_refresh() {
	local path="$1"
	case "${path}" in
	internal/infra/"${NAME}"/internal/openapi/client.gen.go | \
		internal/gen/proto/external/"${NAME}"/*)
		return 0
		;;
	esac
	return 1
}

containment() {
	local mode="$1"
	local path
	while IFS= read -r path; do
		[[ -n "${path}" ]] || continue
		if [[ "${mode}" == "refresh" ]]; then
			path_allowed_refresh "${path}" || die "${reason_precondition}" "refresh changed non-generated path ${path}"
		else
			path_allowed_initial "${path}" || die "${reason_precondition}" "initial changed out-of-allowlist path ${path}"
		fi
	done < <(changed_paths)
}

stage_work() {
	local mode="$1"
	local field
	field="$(field_name "${NAME}")"
	local module
	module="$(module_path)"
	if [[ "${mode}" == "initial" ]]; then
		insert_integrations_field
		ensure_integrations_config "${field}"
		add_defaults_call "${NAME}IntegrationDefaults"
		add_validate_call "validate${field}Integration"
		if [[ "${TRANSPORT}" == "http" ]]; then
			write_http_config "${field}"
			write_http_adapter "${field}" "${module}"
		else
			write_grpc_config "${field}"
			write_grpc_adapter "${field}" "${module}"
			add_golangci_grpc_rule
		fi
		write_bootstrap "${field}" "${module}"
		write_docs
		render_examples
		if retires_singleton_oauth initial; then
			remove_singleton_oauth
			migrate_oauth_helpers "${field}"
		fi
		add_snapshot_keys
		local generator_source
		if [[ "${TRANSPORT}" == "http" ]]; then
			generator_source="internal/infra/${NAME}/internal/openapi/doc.go"
		else
			generator_source="buf.gen.yaml"
		fi
		write_record "$(record_path)" "${generator_source}"
		install_test_env_tuple
	fi
	generate_clients
	if [[ "${mode}" == "initial" ]]; then
		format_manual_go
	fi
	admit_stage
	containment "${mode}"
}

parent_main() {
	[[ "${#}" -eq 5 ]] || usage
	[[ -n "${NAME}" && -n "${TRANSPORT}" && -n "${CONTRACT}" && -n "${AUTH}" ]] || die "${reason_input}" "NAME, TRANSPORT, CONTRACT, and AUTH are required"
	admit_name "${NAME}" || die "${reason_input}" "NAME must be a lower-case Go package identifier"
	case "${TRANSPORT}" in
	http | grpc) ;;
	*) die "${reason_input}" "TRANSPORT must be http or grpc" ;;
	esac
	case "${AUTH}" in
	none | oauth2-client-credentials) ;;
	*) die "${reason_input}" "AUTH must be none or oauth2-client-credentials" ;;
	esac
	if [[ "${TRANSPORT}" == "http" ]]; then
		case "${TARGET}" in
		external-https | private-https) ;;
		*) die "${reason_input}" "TARGET must be external-https or private-https" ;;
		esac
		[[ "${CONTRACT}" == "api/external/${NAME}/openapi.yaml" ]] || die "${reason_contract}" "HTTP CONTRACT must be api/external/${NAME}/openapi.yaml"
	else
		[[ -z "${TARGET}" ]] || die "${reason_input}" "TARGET is not accepted for gRPC"
		case "${CONTRACT}" in
		api/proto/external/"${NAME}"/*.proto | api/proto/external/"${NAME}"/*/*.proto) ;;
		*)
			case "${CONTRACT}" in
			api/proto/external/"${NAME}"/*) ;;
			*) die "${reason_contract}" "gRPC CONTRACT must be a .proto under api/proto/external/${NAME}/" ;;
			esac
			[[ "${CONTRACT}" == *.proto ]] || die "${reason_contract}" "gRPC CONTRACT must be a .proto under api/proto/external/${NAME}/"
			;;
		esac
	fi

	cd "${ROOT_DIR}"
	[[ -f template.lock && ! -L template.lock ]] || die "${reason_precondition}" "template.lock is required"
	clean_worktree || die "${reason_input}" "worktree must be clean"
	if [[ -d scripts/profiles ]]; then
		die "${reason_precondition}" "unresolved template profile sources remain"
	fi
	if git grep -q 'profile:.*:start' HEAD -- . ':(exclude)scripts/init-module.sh' ':(exclude)scripts/ci/*' >/dev/null 2>&1; then
		# Remaining markers are expected in the source template; initialized services have none.
		if [[ -f scripts/profiles ]]; then
			die "${reason_precondition}" "unresolved template profile markers remain"
		fi
	fi

	case "${TRANSPORT}" in
	http)
		[[ "$(lock_value outbound_http || true)" == "bounded" ]] || die "${reason_precondition}" "TRANSPORT=http requires outbound_http = \"bounded\""
		;;
	grpc)
		[[ "$(lock_value grpc || true)" == "enabled" ]] || die "${reason_precondition}" "TRANSPORT=grpc requires grpc = \"enabled\""
		;;
	esac
	if [[ "${AUTH}" == "oauth2-client-credentials" ]]; then
		[[ "$(lock_value outbound_auth || true)" == "oauth2-client-credentials" ]] || die "${reason_precondition}" "AUTH=oauth2-client-credentials requires outbound_auth = \"oauth2-client-credentials\""
	fi

	contract_ok "${CONTRACT}" || die "${reason_contract}" "CONTRACT must be a tracked committed regular non-symlink file"

	local mode
	mode="$(classify_mode)"
	if [[ "${mode}" == "initial" ]]; then
		local reserved
		while IFS= read -r reserved; do
			[[ "${reserved}" == "${CONTRACT}" ]] && continue
			if path_exists "${ROOT_DIR}/${reserved}"; then
				die "${reason_precondition}" "reserved output already exists: ${reserved}"
			fi
		done < <(reserved_outputs)
	fi

	local start_head
	start_head="$(git rev-parse HEAD)"
	local retire=false
	if retires_singleton_oauth "${mode}"; then
		retire=true
	fi
	if [[ "${retire}" == true ]] && env_entry_present "${ROOT_DIR}"; then
		die "${reason_env}" ".env"
	fi

	require_tools
	validate_contract_schema

	local common
	common="$(git rev-parse --git-common-dir)"
	lock_path="${common}/integration-init.lock"
	mkdir "${lock_path}" || die "${reason_precondition}" "another initializer is running"
	trap cleanup EXIT

	stage_dir="$(mktemp -d "${TMPDIR:-/tmp}/integration-init.XXXXXX")"
	rmdir "${stage_dir}"
	git worktree add --detach "${stage_dir}" "${start_head}" >/dev/null
	(
		cd "${stage_dir}"
		INTEGRATION_INIT_STAGE=1 INTEGRATION_INIT_MODE="${mode}" ROOT_OVERRIDE="${stage_dir}" \
			bash "${ROOT_DIR}/scripts/integration-init.sh" "${NAME}" "${TRANSPORT}" "${CONTRACT}" "${TARGET}" "${AUTH}"
	)

	local patch
	patch="$(mktemp)"
	git -C "${stage_dir}" add -A
	git -C "${stage_dir}" diff --binary --cached >"${patch}"
	git worktree remove --force "${stage_dir}" >/dev/null
	stage_dir=""

	same_head "${start_head}" || die "${reason_precondition}" "HEAD changed during staging"
	clean_worktree || die "${reason_precondition}" "worktree changed during staging"
	if [[ ! -s "${patch}" ]]; then
		rm -f "${patch}"
		return 0
	fi
	git apply --check "${patch}"
	if [[ "${retire}" == true ]] && env_entry_present "${ROOT_DIR}"; then
		rm -f "${patch}"
		die "${reason_env}" ".env"
	fi
	git apply "${patch}"
	rm -f "${patch}"
}

stage_main() {
	ROOT_DIR="${ROOT_OVERRIDE:-${ROOT_DIR}}"
	cd "${ROOT_DIR}"
	stage_work "${INTEGRATION_INIT_MODE}"
}

if [[ "${INTEGRATION_INIT_STAGE:-}" == "1" ]]; then
	stage_main
else
	parent_main "$@"
fi
