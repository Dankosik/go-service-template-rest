#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"
cd "${ROOT_DIR}"

shopt -s nullglob
records=(integrations/*.toml)
status=0

require_literal() {
	local record="$1" file="$2" literal="$3" description="$4"
	if [[ ! -f "${file}" ]] || ! grep -Fq -- "${literal}" "${file}"; then
		echo "${record}: ${description} is missing from ${file}" >&2
		status=1
	fi
}

require_regex() {
	local record="$1" file="$2" pattern="$3" description="$4"
	if [[ ! -f "${file}" ]] || ! grep -Eq -- "${pattern}" "${file}"; then
		echo "${record}: ${description} is missing from ${file}" >&2
		status=1
	fi
}

forbid_literal() {
	local record="$1" file="$2" literal="$3" description="$4"
	if [[ -f "${file}" ]] && grep -Fq -- "${literal}" "${file}"; then
		echo "${record}: ${description} is forbidden in ${file}" >&2
		status=1
	fi
}

owned_integration_names() {
	local path name
	for path in api/external/*/openapi.yaml; do
		name="${path#api/external/}"
		printf '%s\n' "${name%%/*}"
	done
	for path in internal/config/*_integration_config.go; do
		name="${path##*/}"
		printf '%s\n' "${name%_integration_config.go}"
	done
	for path in docs/integrations/*.md; do
		name="${path##*/}"
		printf '%s\n' "${name%.md}"
	done
	for path in internal/infra/*/internal/openapi/client.gen.go; do
		name="${path#internal/infra/}"
		printf '%s\n' "${name%%/*}"
	done
	for path in internal/gen/proto/external/*; do
		[[ -d "${path}" ]] || continue
		printf '%s\n' "${path##*/}"
	done
	for path in internal/infra/*/doc.go; do
		grep -Fq 'concrete outbound adapter for the' "${path}" || continue
		name="${path#internal/infra/}"
		printf '%s\n' "${name%%/*}"
	done
	for path in cmd/service/internal/bootstrap/startup_*.go; do
		name="${path##*/startup_}"
		name="${name%.go}"
		grep -Fq 'IntegrationConfig' "${path}" || continue
		grep -Fq "/internal/infra/${name}\"" "${path}" || continue
		printf '%s\n' "${name}"
	done
	if [[ -f env/.env.example ]]; then
		awk -F'__' '/^APP__INTEGRATIONS__/ { print tolower($3) }' env/.env.example
	fi
}

while IFS= read -r owned_name; do
	[[ -n "${owned_name}" ]] || continue
	if [[ ! -f "integrations/${owned_name}.toml" || -L "integrations/${owned_name}.toml" ]]; then
		echo "integrations/${owned_name}.toml: missing record for integration-owned artifacts" >&2
		status=1
	fi
done < <(owned_integration_names | LC_ALL=C sort -u)

if ((${#records[@]} == 0)); then
	if [[ "${status}" -eq 0 ]]; then
		echo "not applicable: no external integration records"
	fi
	exit "${status}"
fi

for record in "${records[@]}"; do
	name="${record#integrations/}"
	name="${name%.toml}"
	if [[ ! -f "${record}" || -L "${record}" || ! "${name}" =~ ^[a-z][a-z0-9_]*$ || "${name}" == *"__"* ]]; then
		echo "${record}: invalid integration record path" >&2
		status=1
		continue
	fi

	transport="$(sed -n '3s/^transport = "\([^"]*\)"$/\1/p' "${record}")"
	contract="$(sed -n '4s/^contract = "\([^"]*\)"$/\1/p' "${record}")"
	first="$(printf '%s' "${name}" | cut -c1 | tr '[:lower:]' '[:upper:]')"
	field="${first}${name:1}"
	env_name="$(printf '%s' "${name}" | tr '[:lower:]' '[:upper:]')"
	config="internal/config/${name}_integration_config.go"
	adapter="internal/infra/${name}/client.go"
	bootstrap="cmd/service/internal/bootstrap/startup_${name}.go"
	documentation="docs/integrations/${name}.md"
	bootstrap_mappings=()
	case "${transport}" in
	http)
		target="$(sed -n '5s/^target = "\([^"]*\)"$/\1/p' "${record}")"
		auth="$(sed -n '6s/^auth = "\([^"]*\)"$/\1/p' "${record}")"
		if [[ "${contract}" != "api/external/${name}/openapi.yaml" ||
			( "${target}" != "external-https" && "${target}" != "private-https" ) ||
			( "${auth}" != "none" && "${auth}" != "oauth2-client-credentials" ) ]] ||
			! cmp -s "${record}" <(printf '%s\n' \
				'schema = 1' \
				"name = \"${name}\"" \
				'transport = "http"' \
				"contract = \"${contract}\"" \
				"target = \"${target}\"" \
				"auth = \"${auth}\"" \
				"generator_source = \"internal/infra/${name}/internal/openapi/doc.go\""); then
			echo "${record}: non-canonical HTTP integration record" >&2
			status=1
			continue
		fi
		for required in \
			"${contract}" \
			"internal/infra/${name}/internal/openapi/doc.go" \
			"internal/infra/${name}/internal/openapi/client.gen.go"; do
			if [[ ! -f "${required}" || -L "${required}" ]]; then
				echo "${record}: missing regular source/output ${required}" >&2
				status=1
			fi
		done
		require_regex "${record}" "${config}" "^[[:space:]]+BaseURL[[:space:]]+string[[:space:]]+\`koanf:\"base_url\"\`" "HTTP base URL field"
		require_literal "${record}" "${adapter}" '/internal/infra/httpclient"' "bounded HTTP owner"
		bootstrap_mappings+=("BaseURL=cfg.BaseURL")
		require_literal "${record}" "${documentation}" "Target class: \`${target}\`" "HTTP target class"
		require_literal "${record}" "internal/infra/${name}/internal/openapi/doc.go" "../../../../../${contract}" "exact OpenAPI source"
		require_literal "${record}" "internal/infra/${name}/internal/openapi/oapi-codegen.yaml" 'client: true' "client-only OpenAPI generator"
		if [[ "${target}" == "private-https" ]]; then
			require_regex "${record}" "${config}" "^[[:space:]]+PrivateDNSSuffix[[:space:]]+string[[:space:]]+\`koanf:\"private_dns_suffix\"\`" "private DNS suffix field"
			bootstrap_mappings+=("PrivateDNSSuffix=cfg.PrivateDNSSuffix")
			if ! go run ./scripts/ci/integration-record-constructor-check.go -- "${adapter}" /internal/infra/httpclient NewPrivateHTTPS NewExternalHTTPS "${auth}"; then
				echo "${record}: private HTTP constructor ownership is invalid" >&2
				status=1
			fi
			require_literal "${record}" env/.env.example "APP__INTEGRATIONS__${env_name}__PRIVATE_DNS_SUFFIX=" "private DNS suffix environment key"
		else
			forbid_literal "${record}" "${config}" 'PrivateDNSSuffix' "private DNS suffix field"
			if ! go run ./scripts/ci/integration-record-constructor-check.go -- "${adapter}" /internal/infra/httpclient NewExternalHTTPS NewPrivateHTTPS "${auth}"; then
				echo "${record}: external HTTP constructor ownership is invalid" >&2
				status=1
			fi
		fi
		require_literal "${record}" env/.env.example "APP__INTEGRATIONS__${env_name}__BASE_URL=" "HTTP base URL environment key"
		;;
	grpc)
		auth="$(sed -n '5s/^auth = "\([^"]*\)"$/\1/p' "${record}")"
		if [[ "${contract}" != api/proto/external/"${name}"/*.proto ||
			( "${auth}" != "none" && "${auth}" != "oauth2-client-credentials" ) ]] ||
			! cmp -s "${record}" <(printf '%s\n' \
				'schema = 1' \
				"name = \"${name}\"" \
				'transport = "grpc"' \
				"contract = \"${contract}\"" \
				"auth = \"${auth}\"" \
				'generator_source = "buf.gen.yaml"'); then
			echo "${record}: non-canonical gRPC integration record" >&2
			status=1
			continue
		fi
		if [[ ! -f "${contract}" || -L "${contract}" ]] ||
			! find "internal/gen/proto/external/${name}" -type f -name '*.go' -print -quit 2>/dev/null | grep -q .; then
			echo "${record}: missing regular gRPC source or generated output" >&2
			status=1
		fi
		require_regex "${record}" "${config}" "^[[:space:]]+Target[[:space:]]+string[[:space:]]+\`koanf:\"target\"\`" "gRPC target field"
		require_literal "${record}" "${adapter}" '/internal/infra/grpcclient"' "bounded gRPC owner"
		bootstrap_mappings+=("Target=cfg.Target")
		require_literal "${record}" .golangci.yml "${name}_grpc_generated_adapter_only:" "gRPC generated import rule"
		require_literal "${record}" .golangci.yml "internal/gen/proto/external/${name}" "gRPC generated import path"
		require_literal "${record}" env/.env.example "APP__INTEGRATIONS__${env_name}__TARGET=" "gRPC target environment key"
		if ! go run ./scripts/ci/integration-record-grpc-check.go -- "${adapter}" "${auth}"; then
			echo "${record}: gRPC transport/auth/lifecycle ownership is invalid" >&2
			status=1
		fi
		;;
	*)
		echo "${record}: unknown integration transport" >&2
		status=1
		continue
		;;
	esac

	for required in \
		"internal/config/${name}_integration_config.go" \
		"internal/infra/${name}/client.go" \
		"cmd/service/internal/bootstrap/startup_${name}.go" \
		"docs/integrations/${name}.md"; do
		if [[ ! -f "${required}" || -L "${required}" ]]; then
			echo "${record}: missing regular owner ${required}" >&2
			status=1
		fi
	done

	require_literal "${record}" internal/config/integrations_config.go "${field}IntegrationConfig" "named config aggregate type"
	require_literal "${record}" internal/config/integrations_config.go "koanf:\"${name}\"" "named config aggregate tag"
	require_literal "${record}" internal/config/types.go 'koanf:"integrations"' "root integrations field"
	require_literal "${record}" internal/config/defaults.go "${name}IntegrationDefaults()" "named defaults composition"
	require_literal "${record}" internal/config/validate.go "validate${field}Integration(&cfg.Integrations)" "named validation composition"
	require_literal "${record}" "${documentation}" "Contract: \`${contract}\`" "documented contract"
	require_literal "${record}" "${documentation}" "Transport: \`${transport}\`" "documented transport"
	require_literal "${record}" "${documentation}" "Authentication: \`${auth}\`" "documented authentication"
	require_literal "${record}" env/config/local.yaml "  ${name}:" "named YAML section"

	if [[ "${auth}" == "oauth2-client-credentials" ]]; then
		require_regex "${record}" "${config}" "^[[:space:]]+OAuth[[:space:]]+OutboundAuthConfig[[:space:]]+\`koanf:\"oauth\"\`" "named OAuth config"
		require_literal "${record}" "${adapter}" '/internal/infra/oauth2clientcredentials"' "OAuth credential owner"
		bootstrap_mappings+=(
			"OAuth.TokenURL=cfg.OAuth.TokenURL"
			"OAuth.ClientID=cfg.OAuth.ClientID"
			"OAuth.ClientSecret=cfg.OAuth.ClientSecret"
			"OAuth.Scopes=cfg.OAuth.Scopes"
		)
		require_literal "${record}" "${documentation}" ".oauth.client_secret\` (environment-only)" "OAuth secret documentation"
		require_literal "${record}" env/.env.example "APP__INTEGRATIONS__${env_name}__OAUTH__CLIENT_SECRET=" "OAuth secret environment key"
	else
		forbid_literal "${record}" "${config}" 'OAuth OutboundAuthConfig' "OAuth config"
		forbid_literal "${record}" "${adapter}" 'oauth2clientcredentials' "OAuth credential owner"
		forbid_literal "${record}" "${bootstrap}" 'cfg.OAuth' "OAuth bootstrap mapping"
		forbid_literal "${record}" "${documentation}" '.oauth.' "OAuth documentation"
	fi
	if ! go run ./scripts/ci/integration-record-bootstrap-check.go -- \
		"${bootstrap}" cmd/service/internal/bootstrap/run.go "/internal/infra/${name}" \
		"init${field}" "${name}Client" "bootstrap.cfg.Integrations.${field}" "${bootstrap_mappings[@]}"; then
		echo "${record}: bootstrap mapping or run wiring is invalid" >&2
		status=1
	fi
done

exit "${status}"
