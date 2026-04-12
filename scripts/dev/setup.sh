#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

usage() {
	echo "usage: $0 [--auto|--native|--docker] [--strict]"
	echo "default auto mode prefers docker when daemon is reachable"
	echo "examples:"
	echo "  $0"
	echo "  $0 --native"
	echo "  $0 --docker"
	echo "  $0 --strict"
	echo "  $0 --native --strict"
}

mode="auto"
strict_mode=0
template_module="github.com/example/go-service-template-rest"
template_source_origin="github.com/Dankosik/go-service-template-rest"
codeowner_placeholder="@your-org/your-team"

while [[ $# -gt 0 ]]; do
	case "$1" in
	--auto)
		mode="auto"
		;;
	--native)
		mode="native"
		;;
	--docker)
		mode="docker"
		;;
	--strict)
		strict_mode=1
		;;
	-h | --help)
		usage
		exit 0
		;;
	-*)
		echo "unknown flag: $1"
		usage
		exit 1
		;;
	*)
		echo "unknown argument: $1"
		usage
		exit 1
		;;
	esac
	shift
done

detect_module_from_origin() {
	local remote_url host path without_scheme

	remote_url="$(git config --get remote.origin.url 2>/dev/null || true)"
	if [[ -z "${remote_url}" ]]; then
		return 1
	fi

	case "${remote_url}" in
	git@*:* )
		host="${remote_url#git@}"
		host="${host%%:*}"
		path="${remote_url#*:}"
		;;
	ssh://git@*/*)
		without_scheme="${remote_url#ssh://git@}"
		host="${without_scheme%%/*}"
		path="${without_scheme#*/}"
		;;
	http://*|https://*)
		without_scheme="${remote_url#*://}"
		host="${without_scheme%%/*}"
		path="${without_scheme#*/}"
		;;
	*)
		return 1
		;;
	esac

	path="${path%.git}"
	path="${path#/}"
	path="${path%/}"
	host="${host%/}"
	host="${host%%:*}"

	if [[ -z "${host}" || -z "${path}" || "${path}" == "${remote_url}" ]]; then
		return 1
	fi

	printf '%s/%s\n' "${host}" "${path}"
}

detect_codeowner_from_origin() {
	local module_from_origin path owner

	module_from_origin="$(detect_module_from_origin || true)"
	if [[ -z "${module_from_origin}" ]]; then
		return 1
	fi

	path="${module_from_origin#*/}"
	owner="${path%%/*}"
	if [[ -z "${owner}" || "${owner}" == "${path}" ]]; then
		return 1
	fi

	printf '@%s\n' "${owner}"
}

current_module_from_go_mod() {
	awk '/^module /{print $2; exit}' go.mod
}

codeowner_placeholder_exists() {
	[[ -f ".github/CODEOWNERS" ]] && grep -Fq "${codeowner_placeholder}" ".github/CODEOWNERS"
}

ensure_env_file() {
	if [[ ! -f ".env" ]]; then
		cp env/.env.example .env
		echo "Created .env from env/.env.example"
	fi
}

sync_skills_mirrors() {
	echo "Syncing agent skills mirrors..."
	bash "${ROOT_DIR}/scripts/dev/sync-skills.sh" --sync
}

sync_agent_mirrors() {
	echo "Syncing subagent mirrors..."
	bash "${ROOT_DIR}/scripts/dev/sync-agents.sh" --sync
}

maybe_infer_codeowner_from_origin() {
	local detected_module inferred_codeowner

	if [[ -n "${CODEOWNER:-}" ]] || ! codeowner_placeholder_exists; then
		return 0
	fi

	detected_module="$(detect_module_from_origin || true)"
	if [[ "${detected_module}" == "${template_source_origin}" ]]; then
		return 0
	fi

	inferred_codeowner="$(detect_codeowner_from_origin || true)"
	if [[ -z "${inferred_codeowner}" ]]; then
		echo "CODEOWNER is not set and could not be inferred from git remote origin."
		echo "Set CODEOWNER=@your-org/your-team and rerun setup to enable 'make gh-protect' without manual CODEOWNERS edits."
		return 0
	fi

	CODEOWNER="${inferred_codeowner}"
	export CODEOWNER
	echo "CODEOWNER was not provided. Inferred CODEOWNER=${CODEOWNER} from git remote origin."
}

strict_native_coverage_check() {
	local coverage_check_log
	coverage_check_log="$(mktemp)"
	if env -u GOCOVERDIR go test -covermode=atomic -run '^$' ./internal/api >"${coverage_check_log}" 2>&1; then
		rm -f "${coverage_check_log}"
		return 0
	fi

	if grep -Eq 'does not match go tool version' "${coverage_check_log}"; then
		echo "strict mode: local Go coverage toolchain mismatch detected"
		echo "native setup is rejected in strict mode; rerun with Docker available for automatic fallback"
	else
		echo "strict mode: native Go coverage sanity check failed"
	fi
	cat "${coverage_check_log}"
	rm -f "${coverage_check_log}"
	return 1
}

maybe_init_module_native() {
	local current_module detected_module module_to_set

	current_module="$(current_module_from_go_mod)"
	if [[ -z "${current_module}" ]]; then
		echo "Skipping automatic module initialization: cannot read module path from go.mod"
		return 0
	fi

	detected_module="$(detect_module_from_origin || true)"
	module_to_set=""

	if [[ "${current_module}" == "${template_module}" ]]; then
		if [[ "${detected_module}" == "${template_source_origin}" ]]; then
			echo "Template source repository detected; skipping module path initialization."
		elif [[ -n "${detected_module}" && "${detected_module}" != "${template_module}" ]]; then
			module_to_set="${detected_module}"
			echo "Detected cloned template repository. Initializing module path: ${module_to_set}"
		elif [[ -z "${detected_module}" ]]; then
			echo "Skipping automatic module initialization: cannot detect git remote origin."
			echo "Run 'make init-module MODULE=<host/org/repo>' after clone."
		fi
	fi

	if [[ -z "${module_to_set}" && -n "${CODEOWNER:-}" ]] && codeowner_placeholder_exists; then
		module_to_set="${current_module}"
		echo "Applying CODEOWNER=${CODEOWNER} to .github/CODEOWNERS"
	fi

	if [[ -z "${module_to_set}" ]]; then
		return 0
	fi

	if [[ -n "${CODEOWNER:-}" ]]; then
		CODEOWNER="${CODEOWNER}" bash "${ROOT_DIR}/scripts/init-module.sh" "${module_to_set}" || return 1
	else
		bash "${ROOT_DIR}/scripts/init-module.sh" "${module_to_set}" || return 1
	fi
}

maybe_init_module_docker() {
	local current_module detected_module module_to_set

	current_module="$(current_module_from_go_mod)"
	if [[ -z "${current_module}" ]]; then
		echo "Skipping automatic module initialization: cannot read module path from go.mod"
		return 0
	fi

	detected_module="$(detect_module_from_origin || true)"
	module_to_set=""

	if [[ "${current_module}" == "${template_module}" ]]; then
		if [[ "${detected_module}" == "${template_source_origin}" ]]; then
			echo "Template source repository detected; skipping module path initialization."
		elif [[ -n "${detected_module}" && "${detected_module}" != "${template_module}" ]]; then
			module_to_set="${detected_module}"
			echo "Detected cloned template repository. Initializing module path: ${module_to_set}"
		elif [[ -z "${detected_module}" ]]; then
			echo "Skipping automatic module initialization: cannot detect git remote origin."
			echo "Run 'make docker-init-module MODULE=<host/org/repo>' after clone."
		fi
	fi

	if [[ -z "${module_to_set}" && -n "${CODEOWNER:-}" ]] && codeowner_placeholder_exists; then
		module_to_set="${current_module}"
		echo "Applying CODEOWNER=${CODEOWNER} to .github/CODEOWNERS"
	fi

	if [[ -z "${module_to_set}" ]]; then
		return 0
	fi

	if [[ -n "${CODEOWNER:-}" ]]; then
		CODEOWNER="${CODEOWNER}" bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" init-module "${module_to_set}" || return 1
	else
		bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" init-module "${module_to_set}" || return 1
	fi
}

setup_native() {
	if ! command -v go >/dev/null 2>&1; then
		echo "go is required for native setup. install Go from https://go.dev/dl/"
		exit 1
	fi

	echo "Running native setup..."

	ensure_env_file
	maybe_infer_codeowner_from_origin
	maybe_init_module_native || return 1

	echo "Downloading Go modules..."
	go mod download || return 1

	echo "Running environment doctor (native mode)..."
	bash "${ROOT_DIR}/scripts/dev/doctor.sh" --mode native || return 1

	if [[ "${strict_mode}" -eq 1 ]]; then
		echo "Running strict native coverage sanity check..."
		strict_native_coverage_check || return 1
	fi

	sync_skills_mirrors || return 1
	sync_agent_mirrors || return 1

	echo "Setup complete (native mode)."
	echo "Next steps:"
	echo "  1) make check"
	echo "  2) make gh-protect BRANCH=main  # requires GitHub admin permissions"
	echo "  3) if module path init was skipped, run: make init-module MODULE=<host/org/repo> CODEOWNER=@<your-org>/<your-team>"
}

setup_docker() {
	if ! command -v docker >/dev/null 2>&1; then
		echo "docker is required for zero-setup mode. install Docker Desktop/Engine."
		exit 1
	fi
	if ! docker info >/dev/null 2>&1; then
		echo "docker daemon is not reachable. start Docker Desktop/Engine and retry."
		exit 1
	fi

	echo "Running zero-setup Docker bootstrap..."
	ensure_env_file
	maybe_infer_codeowner_from_origin
	bash "${ROOT_DIR}/scripts/dev/docker-tooling.sh" pull-images || return 1
	maybe_init_module_docker || return 1
	bash "${ROOT_DIR}/scripts/dev/doctor.sh" --mode docker || return 1

	sync_skills_mirrors || return 1
	sync_agent_mirrors || return 1

	echo "Setup complete (docker mode)."
	echo "Next steps:"
	echo "  1) make check"
	echo "  2) make gh-protect BRANCH=main  # requires GitHub admin permissions"
	echo "  3) if module path init was skipped, run: make docker-init-module MODULE=<host/org/repo> CODEOWNER=@<your-org>/<your-team>"
}

case "${mode}" in
auto)
	if command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
		echo "auto mode: docker daemon detected, using zero-setup docker mode"
		setup_docker
	elif command -v go >/dev/null 2>&1; then
		echo "auto mode: docker unavailable, using native mode"
		if setup_native; then
			exit 0
		fi

		if command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
			echo "native setup failed, falling back to zero-setup docker mode..."
			setup_docker
		else
			echo "native setup failed and docker fallback is unavailable"
			echo "fix native issues with 'make doctor-native' or start Docker and rerun 'make setup-docker'"
			exit 1
		fi
	elif command -v docker >/dev/null 2>&1; then
		echo "auto setup failed: docker is installed but daemon is unreachable, and Go is not installed"
		echo "start Docker Desktop/Engine or install Go and rerun setup"
		exit 1
	else
		echo "auto setup failed: install either Go (native mode) or Docker (zero-setup mode)"
		exit 1
	fi
	;;
native)
	setup_native
	;;
docker)
	setup_docker
	;;
*)
	echo "unknown mode: ${mode}"
	usage
	exit 1
	;;
esac
