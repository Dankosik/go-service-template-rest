#!/usr/bin/env bash
set -euo pipefail

route=${1:-}
[[ -n ${route} && ${2:-} == -- && $# -ge 3 ]] || {
	echo "usage: $0 route -- command [args...]" >&2
	exit 2
}
shift 2

candidate=$(git rev-parse HEAD 2>/dev/null || echo unavailable)
command_text=$(printf '%q ' "$@")
started=$(date +%s)
metrics=$(mktemp)
trap 'rm -f "${metrics}"' EXIT

cache_kb() {
	local total=0 path size seen=''
	for path in "${GOMODCACHE:-}" "${TOOL_GOMODCACHE:-}" "${HOME}/.cache/go-build"; do
		[[ -n ${path} && -d ${path} ]] || continue
		case ":${seen}:" in *":${path}:"*) continue ;; esac
		seen=${seen}:${path}
		size=$(du -sk "${path}" 2>/dev/null | awk '{print $1}')
		total=$((total + ${size:-0}))
	done
	printf '%s\n' "${total}"
}

cache_before=$(cache_kb)
set +e
if [[ $(uname -s) == Linux && -x /usr/bin/time ]]; then
	/usr/bin/time -f 'user=%U\nsystem=%S\nmax_rss_kb=%M' -o "${metrics}" "$@"
	status=$?
else
	"$@"
	status=$?
	printf 'user=unknown\nsystem=unknown\nmax_rss_kb=unknown\n' >"${metrics}"
fi
set -e
cache_after=$(cache_kb)
wall_seconds=$(($(date +%s) - started))
user_seconds=$(awk -F= '$1 == "user" { print $2 }' "${metrics}")
system_seconds=$(awk -F= '$1 == "system" { print $2 }' "${metrics}")
max_rss_kb=$(awk -F= '$1 == "max_rss_kb" { print $2 }' "${metrics}")
if [[ ${user_seconds} == unknown ]]; then
	cpu_seconds=unknown
	max_rss_mb=unknown
else
	cpu_seconds=$(awk -v user="${user_seconds}" -v system="${system_seconds}" 'BEGIN { printf "%.2f", user + system }')
	max_rss_mb=$(awk -v kb="${max_rss_kb}" 'BEGIN { printf "%.1f", kb / 1024 }')
fi
cache_restore_mb=$(awk -v kb="${cache_before}" 'BEGIN { printf "%.1f", kb / 1024 }')
cache_save_mb=$(awk -v before="${cache_before}" -v after="${cache_after}" 'BEGIN { delta=after-before; if (delta < 0) delta=0; printf "%.1f", delta / 1024 }')
result=pass
((status == 0)) || result=fail

summary=${GITHUB_STEP_SUMMARY:-/dev/stdout}
{
	printf '### validation metric: %s\n\n' "${route}"
	printf -- '- route: %s\n' "${route}"
	printf -- '- candidate: %s\n' "${candidate}"
	printf -- '- command: %s\n' "${command_text% }"
	printf -- '- wall_seconds: %s\n' "${wall_seconds}"
	printf -- '- cpu_seconds: %s\n' "${cpu_seconds}"
	printf -- '- max_rss_mb: %s\n' "${max_rss_mb}"
	printf -- '- cache_restore_mb: %s\n' "${cache_restore_mb}"
	printf -- '- cache_save_mb: %s\n' "${cache_save_mb}"
	printf -- '- cache_hit: %s\n' "${CACHE_HIT:-unknown}"
	printf -- '- tool_bootstrap_seconds: %s\n' "${TOOL_BOOTSTRAP_SECONDS:-unknown}"
	printf -- '- result: %s\n\n' "${result}"
} >>"${summary}"

exit "${status}"
