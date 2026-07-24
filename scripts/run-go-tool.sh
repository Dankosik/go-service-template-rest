#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 1 ]]; then
	echo "usage: $0 <tool> [args...]"
	exit 2
fi

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tool_name="$1"
shift

tool_path="$(go -C "${root_dir}/tools" tool -n "${tool_name}")"
exec "${tool_path}" "$@"
