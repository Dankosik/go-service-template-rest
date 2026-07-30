#!/usr/bin/env bash
set -euo pipefail

BUF_VERSION="1.72.0"

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tool_dir="${root_dir}/.cache/tools/buf/${BUF_VERSION}"
buf_bin="${BUF_BIN:-${tool_dir}/buf}"

if [[ ! -x "${buf_bin}" ]]; then
	mkdir -p "${tool_dir}"
	GOBIN="${tool_dir}" go install "github.com/bufbuild/buf/cmd/buf@v${BUF_VERSION}"
fi

actual_version="$("${buf_bin}" --version)"
if [[ "${actual_version}" != "${BUF_VERSION}" ]]; then
	echo "buf version mismatch: got ${actual_version}, want ${BUF_VERSION}" >&2
	exit 1
fi

exec "${buf_bin}" "$@"
