#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

usage() {
	echo "usage: $0 [--auto|--native|--docker]"
	echo "examples:"
	echo "  $0"
	echo "  $0 --native"
	echo "  $0 --docker"
}

mode="auto"

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

ensure_env_file() {
	if [[ ! -f ".env" ]]; then
		cp env/.env.example .env
		echo "Created .env from env/.env.example"
	fi
}

sync_skills_mirrors() {
	echo "Syncing agent skills mirrors..."
	"${ROOT_DIR}/scripts/dev/sync-skills.sh" --sync
}

setup_native() {
	if ! command -v go >/dev/null 2>&1; then
		echo "go is required for native setup. install Go from https://go.dev/dl/"
		exit 1
	fi

	echo "Running native setup..."

	ensure_env_file

	echo "Downloading Go modules..."
	go mod download || return 1

	echo "Running environment doctor (native mode)..."
	"${ROOT_DIR}/scripts/dev/doctor.sh" --mode native || return 1

	sync_skills_mirrors || return 1

	echo "Setup complete (native mode)."
	echo "Next steps:"
	echo "  1) make init-module CODEOWNER=@<your-org>/<your-team>   # MODULE auto-detects from origin when omitted"
	echo "  2) make ci-local"
	echo "  3) make gh-protect BRANCH=main  # requires GitHub admin permissions"
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
	"${ROOT_DIR}/scripts/dev/docker-tooling.sh" pull-images || return 1
	"${ROOT_DIR}/scripts/dev/doctor.sh" --mode docker || return 1

	sync_skills_mirrors || return 1

	echo "Setup complete (docker mode)."
	echo "Next steps:"
	echo "  1) make docker-init-module CODEOWNER=@<your-org>/<your-team>   # MODULE auto-detects from origin when omitted"
	echo "  2) make docker-ci"
	echo "  3) make gh-protect BRANCH=main  # requires GitHub admin permissions"
}

case "${mode}" in
auto)
	if command -v go >/dev/null 2>&1; then
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
		setup_docker
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
