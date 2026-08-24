#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
version=$(awk '$1 == "github.com/zricethezav/gitleaks/v8" { sub(/^v/, "", $2); print $2; exit }' "${ROOT_DIR}/tools/go.mod")
[[ -n ${version} ]] || {
	echo "tools/go.mod does not pin Gitleaks" >&2
	exit 1
}
destination=${1:?destination directory is required}

[[ $(uname -s) == Linux ]] || {
	echo "install-gitleaks.sh supports Linux CI runners only" >&2
	exit 2
}

case "${version}:$(uname -m)" in
	8.30.1:x86_64|8.30.1:amd64)
		arch=x64
		checksum=551f6fc83ea457d62a0d98237cbad105af8d557003051f41f3e7ca7b3f2470eb
		;;
	8.30.1:aarch64|8.30.1:arm64)
		arch=arm64
		checksum=e4a487ee7ccd7d3a7f7ec08657610aa3606637dab924210b3aee62570fb4b080
		;;
	*)
		echo "no Gitleaks checksum registered for ${version} on $(uname -m)" >&2
		exit 2
		;;
esac

archive="gitleaks_${version}_linux_${arch}.tar.gz"
tmp=$(mktemp -d)
trap 'rm -rf -- "${tmp}"' EXIT

curl -fsSL "https://github.com/gitleaks/gitleaks/releases/download/v${version}/${archive}" -o "${tmp}/${archive}"
printf '%s  %s\n' "${checksum}" "${tmp}/${archive}" | sha256sum -c -
mkdir -p "${destination}"
tar -xzf "${tmp}/${archive}" -C "${destination}" gitleaks
"${destination}/gitleaks" version
