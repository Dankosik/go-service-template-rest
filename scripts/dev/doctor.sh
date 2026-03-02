#!/usr/bin/env bash
set -euo pipefail

usage() {
	echo "usage: $0 [--strict] [--mode auto|native|docker]"
}

strict_mode=0
mode="auto"

while [[ $# -gt 0 ]]; do
	case "$1" in
	--strict)
		strict_mode=1
		;;
	--mode)
		shift
		if [[ $# -eq 0 ]]; then
			echo "--mode requires a value: auto|native|docker"
			exit 1
		fi
		mode="$1"
		;;
	--mode=*)
		mode="${1#*=}"
		;;
	-h | --help)
		usage
		exit 0
		;;
	*)
		echo "unknown argument: $1"
		usage
		exit 1
		;;
	esac
	shift
done

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

if [[ "${mode}" == "auto" ]]; then
	if command -v go >/dev/null 2>&1; then
		mode="native"
	elif command -v docker >/dev/null 2>&1; then
		mode="docker"
	else
		mode="native"
	fi
fi

if [[ "${mode}" != "native" && "${mode}" != "docker" ]]; then
	echo "unsupported mode: ${mode}"
	usage
	exit 1
fi

required_failures=0
optional_failures=0

ok() {
	echo "[OK] $1"
}

fail_required() {
	echo "[MISSING][required] $1"
	required_failures=$((required_failures + 1))
}

fail_optional() {
	echo "[MISSING][optional] $1"
	optional_failures=$((optional_failures + 1))
}

version_ge() {
	local current="$1"
	local minimum="$2"
	local c_major c_minor c_patch
	local m_major m_minor m_patch

	IFS=. read -r c_major c_minor c_patch <<<"${current}"
	IFS=. read -r m_major m_minor m_patch <<<"${minimum}"

	c_major="${c_major:-0}"
	c_minor="${c_minor:-0}"
	c_patch="${c_patch:-0}"
	m_major="${m_major:-0}"
	m_minor="${m_minor:-0}"
	m_patch="${m_patch:-0}"

	if ((c_major > m_major)); then
		return 0
	fi
	if ((c_major < m_major)); then
		return 1
	fi

	if ((c_minor > m_minor)); then
		return 0
	fi
	if ((c_minor < m_minor)); then
		return 1
	fi

	((c_patch >= m_patch))
}

normalize_go_version() {
	local raw="$1"
	raw="${raw#go}"
	raw="${raw%%[^0-9.]*}"
	printf '%s' "$raw"
}

check_cmd_required() {
	local cmd="$1"
	local hint="$2"
	if command -v "$cmd" >/dev/null 2>&1; then
		ok "$cmd found: $(command -v "$cmd")"
	else
		fail_required "$cmd not found. $hint"
	fi
}

check_cmd_optional() {
	local cmd="$1"
	local hint="$2"
	if command -v "$cmd" >/dev/null 2>&1; then
		ok "$cmd found: $(command -v "$cmd")"
	else
		fail_optional "$cmd not found. $hint"
	fi
}

check_template_placeholders() {
	template_module="github.com/example/go-service-template-rest"
	current_module="$(awk '/^module /{print $2; exit}' go.mod)"
	if [[ "${current_module}" == "${template_module}" ]]; then
		fail_optional "go.mod still uses template module path. Run 'make init-module MODULE=github.com/<your-org>/<your-service>' (or 'make docker-init-module ...')."
	fi

	codeowner_placeholder="@your-org/your-team"
	if [[ -f ".github/CODEOWNERS" ]] && grep -Fq "${codeowner_placeholder}" ".github/CODEOWNERS"; then
		fail_optional ".github/CODEOWNERS still has placeholder owner '${codeowner_placeholder}'. Update it before enabling required code owner reviews."
	fi
}

check_native_go() {
	required_go_raw="$(awk '/^go /{print $2; exit}' go.mod)"
	required_go="$(normalize_go_version "$required_go_raw")"

	current_go_raw="$(go env GOVERSION 2>/dev/null || true)"
	if [[ -z "$current_go_raw" ]]; then
		current_go_raw="$(go version | awk '{print $3}')"
	fi
	current_go="$(normalize_go_version "$current_go_raw")"

	if [[ -z "$current_go" ]]; then
		fail_required "cannot parse local Go version"
	elif version_ge "$current_go" "$required_go"; then
		ok "Go version $current_go satisfies go.mod requirement >= $required_go"
	else
		fail_required "Go version $current_go is lower than required $required_go"
	fi

	if go test -covermode=atomic -run '^$' ./internal/api >/dev/null 2>&1; then
		ok "Go coverage compile sanity check passed"
	else
		fail_required "Go coverage compile sanity check failed. Reinstall Go toolchain and remove conflicting custom GOROOT/GOTOOLDIR settings."
	fi
}

echo "Running local environment checks from $ROOT_DIR (mode=${mode})"

if [[ "${mode}" == "native" ]]; then
	check_cmd_required "make" "Install GNU Make from your package manager."
	check_cmd_required "git" "Install Git from https://git-scm.com/downloads."
	check_cmd_required "go" "Install Go from https://go.dev/dl/."
	check_cmd_required "node" "Install Node.js LTS from https://nodejs.org/."
	check_cmd_required "npx" "npx should be bundled with Node.js/npm installation."

	if command -v go >/dev/null 2>&1; then
		check_native_go
	fi

	check_cmd_optional "docker" "Install Docker Desktop/Engine to run compose, integration tests, and container build."
	if command -v docker >/dev/null 2>&1; then
		if docker info >/dev/null 2>&1; then
			ok "docker daemon is reachable"
		else
			fail_optional "docker daemon is not reachable. Start Docker Desktop/Engine."
		fi
	fi

	check_cmd_optional "gh" "Install GitHub CLI for 'make gh-protect'."
else
	check_cmd_required "git" "Install Git from https://git-scm.com/downloads."
	check_cmd_required "docker" "Install Docker Desktop/Engine."

	if command -v docker >/dev/null 2>&1; then
		if docker info >/dev/null 2>&1; then
			ok "docker daemon is reachable"
		else
			fail_required "docker daemon is not reachable. Start Docker Desktop/Engine."
		fi
	fi

	check_cmd_optional "make" "Install GNU Make for make-based shortcuts."
	check_cmd_optional "go" "Optional in docker mode; host Go is not required."
	check_cmd_optional "node" "Optional in docker mode; host Node.js is not required."
	check_cmd_optional "npx" "Optional in docker mode; host npx is not required."
	check_cmd_optional "gh" "Install GitHub CLI for 'make gh-protect'."
fi

check_template_placeholders

echo "----"
echo "doctor summary: required_failures=$required_failures optional_failures=$optional_failures"

if ((required_failures > 0)); then
	exit 1
fi

if ((strict_mode == 1 && optional_failures > 0)); then
	exit 1
fi

if ((optional_failures > 0)); then
	echo "doctor completed with optional warnings"
fi
