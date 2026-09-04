#!/usr/bin/env bash
# make integration-init: add one named outbound HTTP or gRPC integration.
set -euo pipefail

export GOTOOLCHAIN=local
export GOPROXY=off
export GOSUMDB=off

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
NAME="${1-}"
TRANSPORT="${2-}"
CONTRACT="${3-}"
TARGET="${4-}"
AUTH="${5-}"

reason_input="invalid initializer input"
reason_precondition="initializer precondition failed"
reason_contract="invalid integration contract"

die() {
	local class="$1"
	shift
	printf '%s: %s\n' "${class}" "$*" >&2
	exit 1
}

usage() {
	die "${reason_input}" "usage: $0 NAME TRANSPORT CONTRACT TARGET AUTH"
}

lock_path=""
stage_dir=""
patch_path=""
cleanup() {
	if [[ -n "${stage_dir}" && -d "${stage_dir}" ]]; then
		git -C "${ROOT_DIR}" worktree remove --force "${stage_dir}" >/dev/null 2>&1 || rm -rf "${stage_dir}"
	fi
	if [[ -n "${lock_path}" && -d "${lock_path}" ]]; then
		rmdir "${lock_path}" >/dev/null 2>&1 || true
	fi
	if [[ -n "${patch_path}" ]]; then
		rm -f "${patch_path}"
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
	[[ "${name}" != *"__"* ]] || return 1
	# The shape above leaves only Go's reserved keywords to exclude.
	case "${name}" in
	break | case | chan | const | continue | default | defer | else | fallthrough | for | func | go | goto | if | import | interface | map | package | range | return | select | struct | switch | type | var) return 1 ;;
	esac
	return 0
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

generator_source() {
	if [[ "${TRANSPORT}" == "http" ]]; then
		printf 'internal/infra/%s/internal/openapi/doc.go' "${NAME}"
	else
		printf 'buf.gen.yaml'
	fi
}

render_record() {
	local source="$1"
	{
		echo "schema = 1"
		echo "name = \"${NAME}\""
		echo "transport = \"${TRANSPORT}\""
		echo "contract = \"${CONTRACT}\""
		if [[ "${TRANSPORT}" == "http" ]]; then
			echo "target = \"${TARGET}\""
		fi
		echo "auth = \"${AUTH}\""
		echo "generator_source = \"${source}\""
	}
}

write_record() {
	local dest="$1"
	local source="$2"
	mkdir -p "$(dirname "${dest}")"
	render_record "${source}" >"${dest}"
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
	local abs root_real
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

classify_mode() {
	local rec
	rec="$(record_path)"
	if ! path_exists "${rec}"; then
		printf 'initial'
		return
	fi
	[[ -f "${rec}" && ! -L "${rec}" ]] || die "${reason_precondition}" "integration record must be a regular non-symlink file"
	cmp -s "${rec}" <(render_record "$(generator_source)") ||
		die "${reason_precondition}" "locked integration identity does not match the canonical record"
	printf 'refresh'
}

require_tools() {
	if [[ "${TRANSPORT}" == "http" ]]; then
		go -C "${ROOT_DIR}" tool -modfile=tools/go.mod oapi-codegen -version >/dev/null
	else
		go -C "${ROOT_DIR}" tool -modfile=tools/go.mod buf --version >/dev/null
	fi
}

validate_contract_schema() {
	if [[ "${TRANSPORT}" == "http" ]]; then
		go -C "${ROOT_DIR}" tool -modfile=tools/go.mod validate -- "${ROOT_DIR}/${CONTRACT}"
	else
		(
			cd "${ROOT_DIR}"
			go tool -modfile=tools/go.mod buf lint "${CONTRACT}"
		)
	fi
}

validate_http_contract_refs() {
	go -C "${ROOT_DIR}" run -modfile=tools/go.mod ./scripts/openapi-ref-check.go -- "${CONTRACT}"
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
EOF
		)
		oauth_validate=$(
			cat <<EOF

		if err := validateOutboundAuthConfig(&cfg.${field}.OAuth, "integrations.${NAME}.oauth"); err != nil {
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
	"time"
)

type ${field}IntegrationConfig struct {
	BaseURL                string        \`koanf:"base_url"\`
	ResponseHeaderTimeout  time.Duration \`koanf:"response_header_timeout"\`
	MaxResponseHeaderBytes int64         \`koanf:"max_response_header_bytes"\`
	MaxInFlight            int           \`koanf:"max_in_flight"\`
	MaxResponseBodyBytes   int64         \`koanf:"max_response_body_bytes"\`${suffix_field}${oauth_field}
}

func ${NAME}IntegrationDefaults() map[string]any {
	return map[string]any{
		"integrations.${NAME}.base_url":                  "",
		"integrations.${NAME}.response_header_timeout":    0,
		"integrations.${NAME}.max_response_header_bytes": 0,
		"integrations.${NAME}.max_in_flight":              0,
		"integrations.${NAME}.max_response_body_bytes":   0,${suffix_default}${oauth_default}
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
	if cfg.${field}.ResponseHeaderTimeout <= 0 || cfg.${field}.MaxResponseHeaderBytes <= 0 ||
		cfg.${field}.MaxInFlight <= 0 || cfg.${field}.MaxResponseBodyBytes <= 0 {
		return fmt.Errorf("%w: integrations.${NAME} HTTP limits must be positive", ErrValidate)
	}
${suffix_validate}${oauth_validate}
	return nil
}
EOF
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
EOF
		)
		oauth_validate=$(
			cat <<EOF

		if err := validateOutboundAuthConfig(&cfg.${field}.OAuth, "integrations.${NAME}.oauth"); err != nil {
		return err
	}
EOF
		)
	fi
	cat >"${dest}" <<EOF
package config

import (
	"fmt"
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
	if cfg.${field}.Target == "" {
		return fmt.Errorf("%w: integrations.${NAME}.target is required", ErrValidate)
	}
	${oauth_validate}
	return nil
}
EOF
	write_config_test "${field}" "grpc"
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
$(if [[ "${kind}" == "http" ]]; then echo '	"time"'; fi)
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
		local extra=$'\n\t\tResponseHeaderTimeout: 5 * time.Second,\n\t\tMaxResponseHeaderBytes: 32 << 10,\n\t\tMaxInFlight: 16,\n\t\tMaxResponseBodyBytes: 1 << 20,'
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
	local module="$1"
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

//go:generate go tool -modfile=../../../../../tools/go.mod oapi-codegen -config oapi-codegen.yaml ../../../../../${CONTRACT}
EOF
	cat >"${dir}/internal/openapi/oapi-codegen.yaml" <<EOF
package: openapi
output: client.gen.go
generate:
  models: true
  client: true
EOF
	local ctor="httpclient.NewExternalHTTPS(cfg.BaseURL, cfg.Limits)"
	local extra_field=""
	if [[ "${TARGET}" == "private-https" ]]; then
		ctor="httpclient.NewPrivateHTTPS(cfg.BaseURL, cfg.PrivateDNSSuffix, cfg.Limits)"
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
	})
	if err != nil {
		transport.CloseIdleConnections()
		return nil, fmt.Errorf("build ${NAME} client: %w", err)
	}
	authenticated, err := auth.HTTP(transport)
	if err != nil {
		auth.Close()
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
		c.auth.Close()
	}
EOF
		)
	fi
	cat >"${dir}/client.go" <<EOF
package ${NAME}

import (
	"fmt"$(if [[ "${AUTH}" == "oauth2-client-credentials" ]]; then printf '\n\t"strings"'; fi)

	"${module}/internal/infra/${NAME}/internal/openapi"
	"${module}/internal/infra/httpclient"${auth_import}
)

// Config is the validated runtime tuple for this integration.
type Config struct {
	BaseURL string${extra_field}
	Limits  httpclient.TransportLimits
$(if [[ "${AUTH}" == "oauth2-client-credentials" ]]; then
		cat <<'INNER'
	OAuth struct {
		TokenURL     string
		ClientID     string
		ClientSecret string
		Scopes       string
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
		$(if [[ "${AUTH}" == "oauth2-client-credentials" ]]; then echo 'auth.Close()'; fi)
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
func (c *Client) Close() error {
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
	local module="$1"
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
	local auth_import=""
	local auth_config=""
	local auth_build=""
	local auth_field=""
	local auth_close=""
	local auth_return=""
	local connection_build=""
	if [[ "${AUTH}" == "oauth2-client-credentials" ]]; then
		auth_import=$'\n\t"'"${module}"'/internal/infra/oauth2clientcredentials"'
		auth_config=$(
			cat <<'EOF'
	OAuth struct {
		TokenURL     string
		ClientID     string
		ClientSecret string
		Scopes       string
	}
EOF
		)
		auth_field=$'\n\tauth *oauth2clientcredentials.Client'
		auth_build=$(
			cat <<EOF
	auth, err := oauth2clientcredentials.New(oauth2clientcredentials.Config{
		TokenURL:     cfg.OAuth.TokenURL,
		ClientID:     cfg.OAuth.ClientID,
		ClientSecret: cfg.OAuth.ClientSecret,
		Scopes:       strings.Fields(cfg.OAuth.Scopes),
	})
	if err != nil {
		return nil, fmt.Errorf("build ${NAME} client: %w", err)
	}
	conn, err := auth.GRPC(target, grpcclient.Options{TransportCredentials: creds})
	if err != nil {
		auth.Close()
		return nil, fmt.Errorf("build ${NAME} client: %w", err)
	}
EOF
		)
		auth_close=$'\n\t\tif c.auth != nil {\n\t\t\tc.auth.Close()\n\t\t}'
		auth_return=$'\n\t\tauth: auth,'
	else
		connection_build=$(
			cat <<EOF
	conn, err := grpcclient.New(target, grpcclient.Options{TransportCredentials: creds})
	if err != nil {
		return nil, fmt.Errorf("build ${NAME} client: %w", err)
	}
EOF
		)
	fi
	cat >"${dir}/client.go" <<EOF
package ${NAME}

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync"

	"${module}/internal/infra/grpcclient"${auth_import}
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type Config struct {
	Target string
${auth_config}
}

type grpcConnection interface {
	grpc.ClientConnInterface
	Close() error
}

type Client struct {
	conn grpcConnection${auth_field}
	closeOnce sync.Once
	closeErr  error
}

func New(cfg Config) (*Client, error) {
	target, hostname, err := parseGRPCTarget(cfg.Target)
	if err != nil {
		return nil, fmt.Errorf("build ${NAME} client: %w", err)
	}
	creds := credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS12, ServerName: hostname})
${auth_build}${connection_build}
	return &Client{
		conn: conn,${auth_return}
	}, nil
}

func parseGRPCTarget(raw string) (string, string, error) {
	endpoint, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || endpoint == nil || endpoint.Scheme != "dns" || endpoint.Opaque != "" ||
		endpoint.User != nil || endpoint.Host != "" || endpoint.RawPath != "" ||
		endpoint.RawQuery != "" || endpoint.ForceQuery || endpoint.Fragment != "" {
		return "", "", fmt.Errorf("target must be dns:///hostname:443")
	}
	hostPort := strings.TrimPrefix(endpoint.Path, "/")
	if hostPort == "" || endpoint.Path != "/"+hostPort || strings.Contains(hostPort, "/") {
		return "", "", fmt.Errorf("target must be dns:///hostname:443")
	}
	hostname, port, splitErr := net.SplitHostPort(hostPort)
	if splitErr != nil || port != "443" || net.ParseIP(hostname) != nil || !validDNSHostname(hostname) {
		return "", "", fmt.Errorf("target must be dns:///hostname:443")
	}
	hostname = strings.ToLower(hostname)
	return "dns:///" + net.JoinHostPort(hostname, "443"), hostname, nil
}

func validDNSHostname(hostname string) bool {
	if hostname == "" || len(hostname) > 253 || strings.HasPrefix(hostname, ".") || strings.HasSuffix(hostname, ".") {
		return false
	}
	for _, label := range strings.Split(hostname, ".") {
		if label == "" || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for i := range len(label) {
			char := label[i]
			if char != '-' && (char < '0' || char > '9') && (char < 'A' || char > 'Z') && (char < 'a' || char > 'z') {
				return false
			}
		}
	}
	return true
}

func (c *Client) Close() error {
	if c == nil {
		return nil
	}
	c.closeOnce.Do(func() {${auth_close}
		if c.conn != nil {
			c.closeErr = c.conn.Close()
		}
	})
	return c.closeErr
}
EOF
	write_adapter_test "${dir}" "grpc"
}

write_adapter_test() {
	local dir="$1"
	local kind="$2"
	local valid_target auth_setup test_imports=""
	if [[ "${kind}" == "http" ]]; then
		test_imports=$'\t"time"\n\n\t"'"${module}"$'/internal/infra/httpclient"'
		if [[ "${TARGET}" == "private-https" ]]; then
			printf -v valid_target '\t\tBaseURL: "https://%s.svc.cluster.local",\n\t\tPrivateDNSSuffix: "svc.cluster.local",' "${NAME}"
		else
			printf -v valid_target '\t\tBaseURL: "https://%s.example.com",' "${NAME}"
		fi
		valid_target+=$'\n\t\tLimits: httpclient.TransportLimits{ResponseHeaderTimeout: 5 * time.Second, MaxResponseHeaderBytes: 32 << 10, MaxInFlight: 16, AbsoluteBodyBytes: 1 << 20},'
	else
		printf -v valid_target '\t\tTarget: "dns:///%s.example.com:443",' "${NAME}"
	fi
	if [[ "${AUTH}" == "oauth2-client-credentials" ]]; then
		auth_setup=$(cat <<'EOF'
	cfg.OAuth.TokenURL = "https://auth.example.com/oauth/token"
	cfg.OAuth.ClientID = "client"
	cfg.OAuth.ClientSecret = "secret"
EOF
		)
	else
		auth_setup=""
	fi
	cat >"${dir}/client_test.go" <<EOF
package ${NAME}

import (
	"testing"
${test_imports}
)

func TestNewRejectsEmptyTarget(t *testing.T) {
	t.Parallel()
	if _, err := New(Config{}); err == nil {
		t.Fatal("New() error = nil, want rejection")
	}
}

func TestCloseIdempotent(t *testing.T) {
	t.Parallel()
	cfg := Config{
${valid_target}
	}
${auth_setup}
	client, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := client.Close(); err != nil {
		t.Fatalf("first Close() error = %v", err)
	}
	if err := client.Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
	var nilClient *Client
	if err := nilClient.Close(); err != nil {
		t.Fatalf("nil Close() error = %v", err)
	}
}
EOF
	if [[ "${kind}" == "grpc" ]]; then
		cat >>"${dir}/client_test.go" <<EOF

func TestNewRejectsNonCanonicalGRPCTargets(t *testing.T) {
	t.Parallel()
	for _, target := range []string{
		"dns://authority/${NAME}.example.com:443",
		"https:///${NAME}.example.com:443",
		"dns:///127.0.0.1:443",
		"dns:///${NAME}.example.com:8443",
		"dns:///bad host:443",
		"dns:///${NAME}.example.com:443?route=other",
	} {
		if _, err := New(Config{Target: target}); err == nil {
			t.Errorf("New(%q) error = nil, want rejection", target)
		}
	}
}
EOF
	fi
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
	"${module}/internal/infra/httpclient"
)

func init${field}(cfg config.${field}IntegrationConfig) (*${NAME}.Client, error) {
	client, err := ${NAME}.New(${NAME}.Config{
		BaseURL: cfg.BaseURL,
		Limits: httpclient.TransportLimits{
			ResponseHeaderTimeout:  cfg.ResponseHeaderTimeout,
			MaxResponseHeaderBytes: cfg.MaxResponseHeaderBytes,
			MaxInFlight:            cfg.MaxInFlight,
			AbsoluteBodyBytes:      cfg.MaxResponseBodyBytes,
		},
$(if [[ "${TARGET}" == "private-https" ]]; then echo '		PrivateDNSSuffix: cfg.PrivateDNSSuffix,'; fi)
$(if [[ "${AUTH}" == "oauth2-client-credentials" ]]; then
			cat <<'INNER'
			OAuth: struct {
				TokenURL     string
				ClientID     string
				ClientSecret string
				Scopes       string
			}{
				TokenURL:     cfg.OAuth.TokenURL,
				ClientID:     cfg.OAuth.ClientID,
				ClientSecret: cfg.OAuth.ClientSecret,
				Scopes:       cfg.OAuth.Scopes,
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
			}{
				TokenURL:     cfg.OAuth.TokenURL,
				ClientID:     cfg.OAuth.ClientID,
				ClientSecret: cfg.OAuth.ClientSecret,
				Scopes:       cfg.OAuth.Scopes,
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
			_ = {name}Client.Close()
		}}
	}}()
'''
    anchor = "\tdependencies, err := wiring.dependencies(startupCtx, bootstrap)\n"
    if anchor not in text:
        raise SystemExit("run.go missing dependencies construction anchor")
    text = text.replace(anchor, construct + "\n" + anchor, 1)
    close_anchor = "\tbackgroundErr := supervisor.Shutdown(backgroundCtx)\n"
    close_insert = close_anchor + f"\tif !{name}Closed {{\n\t\t_ = {name}Client.Close()\n\t\t{name}Closed = true\n\t}}\n"
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
		echo "- \`integrations.${NAME}.response_header_timeout\`"
		echo "- \`integrations.${NAME}.max_response_header_bytes\`"
		echo "- \`integrations.${NAME}.max_in_flight\`"
		echo "- \`integrations.${NAME}.max_response_body_bytes\`"
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
INNER
	fi)

The scaffold constructs transport and generated bindings only. HTTP limits are
required and remain service-owned production-contract values. A later manual
operation in this adapter must own request/response mapping, errors, any smaller
deadline/body policy, and retry eligibility. The initializer does not claim
provider compatibility.
EOF
}

render_examples() {
	local yaml="${ROOT_DIR}/env/config/local.yaml"
	local envf="${ROOT_DIR}/env/.env.example"
	python3 - "${yaml}" "${envf}" "${NAME}" "${TRANSPORT}" "${TARGET}" "${AUTH}" <<'PY'
from pathlib import Path
import sys
yaml_path, env_path, name, transport, target, auth = sys.argv[1:7]
yaml = Path(yaml_path)
env = Path(env_path)
ytext = yaml.read_text()
etext = env.read_text()

if transport == "http":
    fields = [
        f"    base_url: \"\"",
        "    response_header_timeout: 0s",
        "    max_response_header_bytes: 0",
        "    max_in_flight: 0",
        "    max_response_body_bytes: 0",
    ]
    if target == "private-https":
        fields.append(f"    private_dns_suffix: \"\"")
    prefix = f"APP__INTEGRATIONS__{name.upper()}__"
    env_keys = [
        f"{prefix}BASE_URL=",
        f"{prefix}RESPONSE_HEADER_TIMEOUT=",
        f"{prefix}MAX_RESPONSE_HEADER_BYTES=",
        f"{prefix}MAX_IN_FLIGHT=",
        f"{prefix}MAX_RESPONSE_BODY_BYTES=",
    ]
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
    ]
    prefix = f"APP__INTEGRATIONS__{name.upper()}__OAUTH__"
    env_keys += [
        f"{prefix}TOKEN_URL=",
        f"{prefix}CLIENT_ID=",
        f"{prefix}CLIENT_SECRET=",
        f"{prefix}SCOPES=",
    ]

section = "integrations:\n  " + name + ":\n" + "\n".join(fields) + "\n"
env_block = "\n".join(env_keys) + "\n"

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
	local entries
	entries="$(mktemp)"
	if [[ "${TRANSPORT}" == "http" ]]; then
		printf '\t\t"integrations.%s.base_url":                  sameSnapshotValue("https://%s.snapshot.example"),\n' \
			"${NAME}" "${NAME}" >"${entries}"
		cat >>"${entries}" <<EOF
		"integrations.${NAME}.response_header_timeout":    {source: "5s", want: 5 * time.Second},
		"integrations.${NAME}.max_response_header_bytes": {source: "32768", want: int64(32 << 10)},
		"integrations.${NAME}.max_in_flight":              {source: "16", want: 16},
		"integrations.${NAME}.max_response_body_bytes":   {source: "1048576", want: int64(1 << 20)},
EOF
		if [[ "${TARGET}" == "private-https" ]]; then
			printf '\t\t"integrations.%s.private_dns_suffix": sameSnapshotValue("svc.cluster.local"),\n' \
				"${NAME}" >>"${entries}"
		fi
	else
		printf '\t\t"integrations.%s.target": sameSnapshotValue("dns:///%s.snapshot.example:443"),\n' \
			"${NAME}" "${NAME}" >"${entries}"
	fi
	if [[ "${AUTH}" == "oauth2-client-credentials" ]]; then
		cat >>"${entries}" <<EOF
		"integrations.${NAME}.oauth.token_url":     sameSnapshotValue("https://auth.snapshot.example/oauth/token"),
		"integrations.${NAME}.oauth.client_id":     sameSnapshotValue(" snapshot-client:id "),
		"integrations.${NAME}.oauth.client_secret": sameSnapshotValue(" snapshot-client-secret "),
		"integrations.${NAME}.oauth.scopes":        sameSnapshotValue("snapshot.read snapshot.write"),
EOF
	fi
	python3 - "${ROOT_DIR}/internal/config/snapshot_contract_test.go" "${entries}" <<'PY'
from pathlib import Path
import sys
path = Path(sys.argv[1])
block = Path(sys.argv[2]).read_text()
text = path.read_text()
fn = "func snapshotContractValues() map[string]snapshotContractValue {"
start = text.find(fn)
if start < 0:
    raise SystemExit(f"missing {fn}")
marker = "return map[string]snapshotContractValue{"
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
	rm -f "${entries}"
}

install_test_env_tuple() {
	local dest
	local env_key env_val
	if [[ "${TRANSPORT}" == "http" ]]; then
		env_key="APP__INTEGRATIONS__$(printf '%s' "${NAME}" | tr '[:lower:]' '[:upper:]')__BASE_URL"
		if [[ "${TARGET}" == "private-https" ]]; then
			env_val="https://${NAME}.svc.cluster.local"
		else
			env_val="https://${NAME}.example.com"
		fi
	else
		env_key="APP__INTEGRATIONS__$(printf '%s' "${NAME}" | tr '[:lower:]' '[:upper:]')__TARGET"
		env_val="dns:///${NAME}.example.com:443"
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
raise SystemExit("run_test.go is missing resetShutdownConfigEnv insertion anchor")
PY
	fi
	if [[ "${TRANSPORT}" == "http" && "${TARGET}" == "private-https" ]]; then
		install_named_test_env_key \
			"${env_key}" \
			"APP__INTEGRATIONS__$(printf '%s' "${NAME}" | tr '[:lower:]' '[:upper:]')__PRIVATE_DNS_SUFFIX" \
			"svc.cluster.local"
	fi
	if [[ "${TRANSPORT}" == "http" ]]; then
		local limit_prefix="${env_key%BASE_URL}"
		install_named_test_env_key "${env_key}" "${limit_prefix}RESPONSE_HEADER_TIMEOUT" "5s"
		install_named_test_env_key "${env_key}" "${limit_prefix}MAX_RESPONSE_HEADER_BYTES" "32768"
		install_named_test_env_key "${env_key}" "${limit_prefix}MAX_IN_FLIGHT" "16"
		install_named_test_env_key "${env_key}" "${limit_prefix}MAX_RESPONSE_BODY_BYTES" "1048576"
	fi
	if [[ "${AUTH}" == "oauth2-client-credentials" ]]; then
		install_named_oauth_test_env "${env_key}"
	fi
}

install_named_test_env_key() {
	local anchor_key="$1"
	local key="$2"
	local value="$3"
	local item dest receiver
	for item in \
		"${ROOT_DIR}/internal/config/configtest/configtest.go|tb" \
		"${ROOT_DIR}/cmd/service/internal/bootstrap/run_test.go|t"; do
		dest="${item%|*}"
		receiver="${item##*|}"
		[[ -f "${dest}" ]] || continue
		python3 - "${dest}" "${receiver}" "${anchor_key}" "${key}" "${value}" <<'PY'
from pathlib import Path
import sys
path = Path(sys.argv[1])
receiver, anchor_key, key, value = sys.argv[2:6]
text = path.read_text()
if key in text:
    raise SystemExit(0)
anchor = f'{receiver}.Setenv("{anchor_key}",'
start = text.find(anchor)
if start < 0:
    raise SystemExit(f"missing generated integration env anchor {anchor_key}")
end = text.find("\n", start) + 1
path.write_text(text[:end] + f'\t{receiver}.Setenv("{key}", "{value}")\n' + text[end:])
PY
	done
}

install_named_oauth_test_env() {
	local anchor_key="$1"
	local item dest receiver
	for item in \
		"${ROOT_DIR}/internal/config/configtest/configtest.go|tb" \
		"${ROOT_DIR}/cmd/service/internal/bootstrap/run_test.go|t"; do
		dest="${item%|*}"
		receiver="${item##*|}"
		[[ -f "${dest}" ]] || continue
		python3 - "${dest}" "${receiver}" "${anchor_key}" "${NAME}" <<'PY'
from pathlib import Path
import sys
path = Path(sys.argv[1])
receiver, anchor_key, name = sys.argv[2], sys.argv[3], sys.argv[4].upper()
text = path.read_text()
prefix = f"APP__INTEGRATIONS__{name}__OAUTH__"
if prefix + "TOKEN_URL" in text:
    raise SystemExit(0)
anchor = f'{receiver}.Setenv("{anchor_key}",'
start = text.find(anchor)
if start < 0:
    raise SystemExit(f"missing generated integration env anchor {anchor_key}")
end = text.find("\n", start) + 1
values = [
    ("TOKEN_URL", "https://auth.example.com/oauth/token"),
    ("CLIENT_ID", "test-client"),
    ("CLIENT_SECRET", "test-client-secret"),
    ("SCOPES", "payments.read payments.write"),
]
insert = "".join(f'\t{receiver}.Setenv("{prefix}{key}", "{value}")\n' for key, value in values)
path.write_text(text[:end] + insert + text[end:])
PY
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
			go tool -modfile=tools/go.mod buf generate
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
		"${ROOT_DIR}/internal/config/configtest/configtest.go"
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
			go tool -modfile=tools/go.mod goimports -w "${present[@]}"
			go tool -modfile=tools/go.mod gofumpt -w "${present[@]}"
		)
	fi
}

admit_stage() {
	local packages=(
		"./internal/config"
		"./internal/infra/${NAME}"
		"./cmd/service/internal/bootstrap"
	)
	if [[ "${TRANSPORT}" == "grpc" ]]; then
		packages+=("./internal/gen/proto/external/${NAME}/...")
	fi
	(cd "${ROOT_DIR}" && go test -vet=off -run '^$' "${packages[@]}")
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
		internal/config/configtest/configtest.go | \
		cmd/service/internal/bootstrap/startup_"${NAME}".go | \
		cmd/service/internal/bootstrap/startup_"${NAME}"_test.go | \
		cmd/service/internal/bootstrap/run.go | \
		cmd/service/internal/bootstrap/run_test.go | \
		cmd/service/internal/bootstrap/startup_dependencies_test.go | \
		docs/integrations/"${NAME}".md | \
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
			write_http_adapter "${module}"
		else
			write_grpc_config "${field}"
			write_grpc_adapter "${module}"
			add_golangci_grpc_rule
		fi
		write_bootstrap "${field}" "${module}"
		write_docs
		render_examples
		add_snapshot_keys
		write_record "$(record_path)" "$(generator_source)"
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
	admit_name "${NAME}" || die "${reason_input}" "NAME must be a lower-case Go package identifier without __"
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
	[[ "$(lock_value state || true)" == "complete" ]] || die "${reason_precondition}" "template.lock state must be complete"
	clean_worktree || die "${reason_input}" "worktree must be clean"
	if [[ -d scripts/profiles ]]; then
		die "${reason_precondition}" "unresolved template profile sources remain"
	fi
	local marker_roots=(
		README.md Makefile railway.toml .gitleaks.toml .golangci.yml
		api build cmd docs env internal migrations test .github scripts/dev scripts/ci
	)
	local present_marker_roots=()
	local marker_root
	for marker_root in "${marker_roots[@]}"; do
		[[ -e "${marker_root}" ]] && present_marker_roots+=("${marker_root}")
	done
	if grep -R -E 'profile:[a-z0-9-]+:(start|end)' "${present_marker_roots[@]}"; then
		die "${reason_precondition}" "unresolved template profile markers remain"
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
	if [[ "${TRANSPORT}" == "http" ]]; then
		validate_http_contract_refs || die "${reason_contract}" "HTTP CONTRACT permits only document-local \$ref values"
	fi

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

	patch_path="$(mktemp)"
	git -C "${stage_dir}" add -A
	git -C "${stage_dir}" diff --binary --cached >"${patch_path}"
	git worktree remove --force "${stage_dir}" >/dev/null
	stage_dir=""

	same_head "${start_head}" || die "${reason_precondition}" "HEAD changed during staging"
	clean_worktree || die "${reason_precondition}" "worktree changed during staging"
	if [[ ! -s "${patch_path}" ]]; then
		rm -f "${patch_path}"
		patch_path=""
		return 0
	fi
	git apply --check "${patch_path}"
	git apply "${patch_path}"
	rm -f "${patch_path}"
	patch_path=""
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
