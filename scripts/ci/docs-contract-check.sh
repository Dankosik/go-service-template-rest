#!/usr/bin/env bash
set -euo pipefail

require_literal() {
	local file=$1 literal=$2
	grep -Fq -- "${literal}" "${file}" || {
		printf '%s: missing contract literal: %s\n' "${file}" "${literal}" >&2
		exit 1
	}
}

forbid_literal() {
	local file=$1 literal=$2
	if grep -Fq -- "${literal}" "${file}"; then
		printf '%s: stale contract literal: %s\n' "${file}" "${literal}" >&2
		exit 1
	fi
}

metrics_default=$(awk -F'"' '$2 == "observability.metrics.addr" { print $4; exit }' internal/config/observability_config.go)
[[ -n ${metrics_default} ]] || { echo 'metrics address default is missing' >&2; exit 1; }
require_literal docs/configuration-source-policy.md "defaults to \`${metrics_default}\`"
forbid_literal docs/architecture/http.md 'Diagnostics default to'
forbid_literal docs/first-production-feature.md 'loopback by default'
forbid_literal docs/railway-deployment-profile.md 'default loopback bind'

request_budget=.agents/skills/go-reliability/references/request-budget.md
require_literal "${request_budget}" 'template provides no separate acquire timeout or saturation error'
forbid_literal "${request_budget}" 'ErrSaturated'
forbid_literal "${request_budget}" 'Config validates PostgreSQL acquire'

client=internal/infra/httpclient/client.go
if [[ -f ${client} ]]; then
	for literal in \
		'type TransportLimits struct {' \
		'ResponseHeaderTimeout  time.Duration' \
		'MaxResponseHeaderBytes int64' \
		'MaxInFlight            int' \
		'AbsoluteBodyBytes      int64' \
		'func NewExternalHTTPS(baseURL string, limits TransportLimits)' \
		'func NewPrivateHTTPS(baseURL, privateSuffix string, limits TransportLimits)' \
		'func (c *Client) DoWithPolicy(request *http.Request, policy OperationPolicy)' \
		'ErrResponseTooLarge' \
		'ErrSaturated'; do
		require_literal "${client}" "${literal}"
	done
	forbid_literal "${client}" 'type ResponseLimits struct {'
	forbid_literal "${client}" 'NewExternalHTTPSWithLimits'
	forbid_literal "${client}" 'NewPrivateHTTPSWithLimits'
	require_literal README.md 'A fixed-authority HTTPS client with mandatory header, decoded-body, and request-concurrency ceilings'
fi

if [[ -f scripts/init-module.sh ]]; then
	require_literal scripts/init-module.sh 'OUTBOUND_HTTP must be one of: none, bounded'
fi
if [[ -f docs/production-contract.md ]]; then
	require_literal docs/production-contract.md '## Dependency contract'
	require_literal docs/production-contract.md '## Capacity envelope'
	require_literal docs/production-contract.md '## Operation and recovery'
fi

echo 'docs contract check passed'
