#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset

script_dir=$(dirname "$(readlink -f "$0")")

containerd_repo=${containerd_repo:-"https://github.com/confidential-containers/containerd"}
containerd_version=${containerd_version:-"v1.6.6.0"}
containerd_dir="$(mktemp -d -t containerd-XXXXXXXXXX)/containerd"

supported_arches=(
	"linux/amd64"
#	"linux/s390x"
)

function clone_repo() {
	echo "Cloning the ${containerd_version} branch of the ${containerd_repo} repo"
	git clone -b "${containerd_version}" "${containerd_repo}" "${containerd_dir}"
}

function go_to_kernel_arch() {
	case "$1" in
		"linux/amd64") echo "x86_64";;
		"linux/s390x") echo "s390x";;
		(*) echo "$1 is not supporte" && exit 1
	esac
		
}

function build_containerd() {
	cc_in_containerd_dir="${containerd_dir}/confidential-containers"
	cc_in_containerd_dockerfile="${cc_in_containerd_dir}/Dockerfile"
	cc_in_containerd_bin_dir="${cc_in_containerd_dir}/bin"

	mkdir -p "${cc_in_containerd_dir}"
	cp ${script_dir}/build_Dockerfile "${cc_in_containerd_dockerfile}"

	pushd "${containerd_dir}"
	for arch in ${supported_arches[@]}; do
		kernel_arch=$(go_to_kernel_arch "${arch}")
		output_dir="${script_dir}/../output/opt/containerd-engine-for-cc-artifacts/${kernel_arch}"

		echo "Building containerd for ${arch}"
		docker buildx build \
			--build-arg RELEASE_VER="${containerd_version}" \
			--build-arg GO_VERSION="1.17.8" \
			-f "${cc_in_containerd_dockerfile}" \
			--platform="${arch}" \
			-o "${output_dir}" \
			.
	done
	popd
}

function main() {
	clone_repo
	build_containerd
}

main "$@"
