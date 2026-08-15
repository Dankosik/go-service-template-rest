#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
platform="${S3_RECEIPT_PLATFORM:-}"

case "${platform}" in
	linux/amd64)
		expected_go_manifest="sha256:433f9dc4f8ea3a1ce4e28f9f15d0f7c056b10475307f886d6f1ac1ccc4abd976"
		expected_final_manifest="sha256:b7ebc675e6df3e26840de28a9d119969806b5542902111bfd111964d0930c08a"
		;;
	linux/arm64)
		expected_go_manifest="sha256:7939e2c75db3d059fc944bb6464a916d0fa64bd5a3bd7b3528f2a1ac7673a0eb"
		expected_final_manifest="sha256:ff3d007f159ebbbdf9130affa1125b10c50a909821c7130419df4e979a197c79"
		;;
	*)
		echo "S3_RECEIPT_PLATFORM must be linux/amd64 or linux/arm64" >&2
		exit 2
		;;
esac

command -v docker >/dev/null
command -v jq >/dev/null
command -v tar >/dev/null

fail() {
	echo "T9A source receipt: $*" >&2
	exit 1
}

platform_manifest() {
	docker buildx imagetools inspect --raw "$1" |
		jq -er --arg os "${platform%/*}" --arg arch "${platform#*/}" \
			'[.manifests[] | select(.platform.os == $os and .platform.architecture == $arch)][0].digest'
}

source_file_sha256() {
	docker cp "${go_container}:$1" - | tar -xO | shasum -a 256 | awk '{print $1}'
}

image_tag="s3-source-receipt-$$"
temp_dir="$(mktemp -d -t s3-source-receipt.XXXXXX)"
final_container=""
go_container=""

cleanup() {
	local status=$?
	if [[ -n "${final_container}" ]]; then
		docker rm -f "${final_container}" >/dev/null 2>&1 || true
	fi
	if [[ -n "${go_container}" ]]; then
		docker rm -f "${go_container}" >/dev/null 2>&1 || true
	fi
	docker image rm -f "${image_tag}" >/dev/null 2>&1 || true
	rm -rf -- "${temp_dir}"
	exit "${status}"
}
trap cleanup EXIT INT TERM

docker image inspect "${image_tag}" >/dev/null 2>&1 && fail "temporary image tag already exists: ${image_tag}"
cd "${root_dir}"

dockerfile="${root_dir}/build/docker/Dockerfile"
dockerfile_sha256="$(shasum -a 256 "${dockerfile}" | awk '{print $1}')"
go_ref="$(awk '$1 == "FROM" { for (i = 1; i <= NF; i++) if ($i ~ /^golang:/) { print $i; exit } }' "${dockerfile}")"
final_ref="$(awk '$1 == "FROM" && $2 ~ /^gcr\.io\/distroless\/static-debian12:/ { print $2; exit }' "${dockerfile}")"
[[ -n "${go_ref}" && -n "${final_ref}" ]] || fail "Dockerfile pins are missing"

go_manifest="$(platform_manifest "${go_ref}")"
final_manifest="$(platform_manifest "${final_ref}")"
[[ "${go_manifest}" == "${expected_go_manifest}" ]] || fail "unexpected Go ${platform} manifest: ${go_manifest}"
[[ "${final_manifest}" == "${expected_final_manifest}" ]] || fail "unexpected Distroless ${platform} manifest: ${final_manifest}"

printf 'Dockerfile SHA-256: %s\nplatform: %s\nGo index: %s\nGo manifest: %s\nDistroless index: %s\nDistroless manifest: %s\n' \
	"${dockerfile_sha256}" "${platform}" "${go_ref##*@}" "${go_manifest}" "${final_ref##*@}" "${final_manifest}"

go_platform_ref="${go_ref%@*}@${go_manifest}"
docker pull --platform "${platform}" "${go_platform_ref}" >/dev/null
go_container="$(docker create --platform "${platform}" "${go_platform_ref}")"
while IFS=' ' read -r source expected; do
	actual="$(source_file_sha256 "${source}")"
	[[ "${actual}" == "${expected}" ]] || fail "Go source identity mismatch for ${source}: ${actual}"
	printf 'Go source SHA-256 %s %s\n' "${source}" "${actual}"
done <<'EOF'
/usr/local/go/src/encoding/pem/pem.go 536954f803f79d8972e8f86b792b25d4ac83167fbe4c3117954a4378e60521da
/usr/local/go/src/crypto/x509/parser.go 70ae7c65f68d17a59e7bacb0d1cc520e8e96e8eafee2b7224465180437ab3a35
/usr/local/go/src/crypto/x509/cert_pool.go d995d6e88af70f36a345185420bac88c58e86ebed1b3eea8e087228eaa7da03b
/usr/local/go/src/crypto/x509/verify.go 3fbc65e9ba1a710f1276d6b9e36483bbc9dd98f48817ed0991ca0894505783f9
/usr/local/go/src/crypto/tls/common.go d2837fbe55a398c7362b3ff8ffe43c06d0832df2305f45f32aeaebadc784486d
/usr/local/go/src/crypto/tls/handshake_client.go 7d6210c69ff9bf0d8506a8f2f59bf33a6132d0a3574487bcc356f858e50c6fab
/usr/local/go/src/net/http/transport.go 1c170ec3581321bd19ccd1b15863f46b16dc749ac6d54d00f5e97f7ffa2ccb5f
/usr/local/go/src/crypto/x509/root.go 813faea4e4990c9760b5fe99d8ae91c8f26c4d3b5dd3c4a6d7bef7b0f11dbae7
/usr/local/go/src/crypto/x509/root_unix.go 421c066f193250dc1adaf13fce4779fc3dc576e20faa86e43bc22af6c9536a14
EOF

for module in \
	'github.com/aws/aws-sdk-go-v2 v1.43.5 h1:yKT5GYnFWhuDo+DqKvE5ZPwVn3RjC4MAeBtZGlh6AVM=' \
	'github.com/aws/aws-sdk-go-v2/credentials v1.19.5 h1:xMo63RlqP3ZZydpJDMBsH9uJ10hgHYfQFIk1cHDXrR4=' \
	'github.com/aws/aws-sdk-go-v2/service/s3 v1.107.1 h1:VUTtUJMuRNMkb/7NIKmd8NQaeQLPGCMoTJxkYKre4qM=' \
	'github.com/aws/smithy-go v1.27.7 h1:Zgj5z4LfcDYoQIVk+n/yGdTkP/2y6ZT5vYxe0fp7bqE='; do
	read -r path version sum <<<"${module}"
	actual="$(go list -mod=readonly -m -f '{{.Version}} {{.Sum}}' "${path}")"
	[[ "${actual}" == "${version} ${sum}" ]] || fail "module identity mismatch for ${path}: ${actual}"
	printf 'module %s %s\n' "${path}" "${actual}"
done

docker buildx build --platform "${platform}" --load --build-arg PGO_PROFILE=off -f "${dockerfile}" -t "${image_tag}" "${root_dir}"
[[ "$(docker image inspect --format '{{.Os}}/{{.Architecture}}' "${image_tag}")" == "${platform}" ]] || fail "final image platform drift"
printf 'final image config: %s\nfinal image rootfs: %s\n' \
	"$(docker image inspect --format '{{.Id}}' "${image_tag}")" \
	"$(docker image inspect --format '{{join .RootFS.Layers ","}}' "${image_tag}")"

final_stage="$(awk '$1 == "FROM" && $2 ~ /^gcr\.io\/distroless\/static-debian12:/ { final = 1; next } final { print }' "${dockerfile}")"
if grep -Eq '^COPY .*(/etc(/|$)|ca-certificates\.crt)' <<<"${final_stage}"; then
	fail "final Dockerfile stage replaces the image root bundle path"
fi

final_container="$(docker create --platform "${platform}" "${image_tag}")"
bundle_tar="${temp_dir}/bundle.tar"
bundle="${temp_dir}/ca-certificates.crt"
docker cp "${final_container}:/etc/ssl/certs/ca-certificates.crt" - >"${bundle_tar}"
entry="$(tar -tvf "${bundle_tar}")"
[[ "$(awk 'NR == 1 { print $1 }' <<<"${entry}")" == '-r-xr-xr-x' ]] || fail "bundle is not regular 0555: ${entry}"
tar -xOf "${bundle_tar}" >"${bundle}"
bytes="$(wc -c <"${bundle}" | tr -d ' ')"
sha256="$(shasum -a 256 "${bundle}" | awk '{print $1}')"
[[ "${bytes}" == '224449' ]] || fail "bundle byte identity drift: ${bytes}"
[[ "${sha256}" == '714d457d580922dbf1d0be8bd35ba236a842b50b0072ae791582a19adef772a5' ]] || fail "bundle hash identity drift: ${sha256}"

docker run --rm --platform "${platform}" --network none \
		-v "${root_dir}:/src:ro" \
		-v "${temp_dir}:/receipt:ro" \
		-v "$(go env GOMODCACHE):/go/pkg/mod:ro" \
		-w /src \
		-e S3_IMAGE_ROOT_BUNDLE_RECEIPT_PATH=/receipt/ca-certificates.crt \
		-e S3_IMAGE_ROOT_BUNDLE_RECEIPT_BYTES="${bytes}" \
		-e S3_IMAGE_ROOT_BUNDLE_RECEIPT_SHA256="${sha256}" \
		-e S3_IMAGE_ROOT_BUNDLE_RECEIPT_ROOTS=150 \
		"${go_platform_ref}" \
		go test -mod=readonly -vet=off ./internal/infra/s3 -run '^TestFinalImageRootBundleReceipt$' -count=1

byte_headroom="$(awk 'BEGIN { printf "%.0f", (458752 - 224449) * 100 / 224449 }')"
root_headroom="$(awk 'BEGIN { printf "%.0f", (288 - 150) * 100 / 150 }')"
printf 'bundle entry: %s\nbundle bytes/hash/roots: %s %s %s\nbundle headroom: bytes-unused=%s (%s%%) roots-unused=%s (%s%%)\n' \
	"${entry}" "${bytes}" "${sha256}" 150 "$((458752 - bytes))" "${byte_headroom}" "$((288 - 150))" "${root_headroom}"

printf '%s\n' \
	'D4 class shared_fixed: driver=S; ceiling=2097152; used-max=1048576; unused-min=1048576; headroom-min=100%' \
	'D4 class shared_variable: driver=U(Q), trust_shared, trust_startup; checked by TestWorkingMemoryAccounting' \
	'D4 class operation_fixed: driver=F, trust_verify; checked by TestWorkingMemoryAccounting' \
	'D4 class request_variable: driver=U(Q+K) or U(Q+K+T); checked by TestWorkingMemoryAccounting' \
	'D4 class response_header: driver=U(H); checked by TestWorkingMemoryAccounting' \
	'D4 class control_response: driver=U(E); checked by TestWorkingMemoryAccounting' \
	'D4 class multipart_retained: driver=retained_parts(P,H), upload_session(E), complete_xml(P,H); checked by TestWorkingMemoryAccounting' \
	'D4 headroom S: ceiling=65536; used-max=32768; unused-min=32768; headroom-min=100%' \
	'D4 headroom U(x): used-max=32*heap(x)+32768; unused-min=32*heap(x)+32768; headroom-min=100%' \
	'D4 headroom trust_shared: ceiling=32047104; used-max=16023552; unused-min=16023552; headroom-min=100%' \
	'D4 headroom trust_startup: ceiling=61997056; used-max=30998528; unused-min=30998528; headroom-min=100%' \
	'D4 headroom trust_verify: ceiling=9309776; used-max=4654888; unused-min=4654888; headroom-min=100%' \
	'D4 shallow: completed_part=96 part_number_pointee=8 string_pointees=32 crc64_text=heap(12) complete_root=101 complete_part=107 escape_factor=5 complete_buffer_slack=8192' \
	'D4 simultaneous: A=2 P<=10000 B=458752 N=288 K=1024 T=1024'

compiler_output="$(
	docker run --rm --platform "${platform}" --network none \
		-v "${root_dir}:/src:ro" \
		-v "$(go env GOMODCACHE):/go/pkg/mod:ro" \
		-w /src "${go_platform_ref}" \
		go test -mod=readonly -vet=off \
			-gcflags='github.com/example/go-service-template-rest/internal/infra/s3=-m=1' \
			./internal/infra/s3 -run '^TestWorkingMemoryAccounting$' -count=1 2>&1
)"

require_compiler_evidence() {
	local description="$1"
	local pattern="$2"
	grep -Eq "${pattern}" <<<"${compiler_output}" || fail "compiler evidence misses ${description}"
}

require_compiler_evidence 'parsed root pool escape' 'image_root_bundle\.go:.*&x509\.CertPool\{\.\.\.\} escapes to heap'
require_compiler_evidence 'retained client escape' 'client\.go:.*&Client\{\.\.\.\} escapes to heap'
require_compiler_evidence 'multipart descriptor backing escape' 'upload\.go:.*make\(\[\]types\.CompletedPart, 0, ~r0\) escapes to heap'
require_compiler_evidence 'multipart request escape' 'upload\.go:.*&types\.CompletedMultipartUpload\{\.\.\.\} escapes to heap'
require_compiler_evidence 'download body escape' 'download\.go:.*&downloadBody\{\.\.\.\} escapes to heap'
require_compiler_evidence 'bounded transport body escape' 'transport\.go:.*&limitedBody\{\.\.\.\} escapes to heap'
printf 'compiler evidence: parsed roots, retained client, multipart descriptors/request, download body, bounded transport body\n'
printf 'T9A source receipt PASS\n'
