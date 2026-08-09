#!/usr/bin/env bash
# require-docker.sh fails a target that needs Docker before it does any work.
#
# Seven Makefile recipes and two benchmark helpers asked the same two questions
# with the same two messages. One copy is what lets the check grow — a rootless
# hint, a disk-space probe, a clearer remediation — without growing nine times,
# and what keeps a target from being given only half of it.
#
# It is run rather than sourced: it needs nothing from the caller's shell, and a
# Makefile recipe cannot source anything anyway.
#
# purpose names the target in the failure, so an operator is told which one needs
# Docker rather than only that something did. DOCKER_BIN covers the benchmark
# caller that runs a client under a configured binary rather than `docker`.
set -euo pipefail

purpose="${1:-this target}"
docker_bin="${DOCKER_BIN:-docker}"

command -v "${docker_bin}" >/dev/null 2>&1 || {
	echo "Docker is required for ${purpose}"
	exit 1
}
"${docker_bin}" info >/dev/null 2>&1 || {
	echo "Docker daemon is not reachable"
	exit 1
}
