#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset

script_dir=$(dirname "$(readlink -f "$0")")

containerd_repo=${containerd_repo:-"https://github.com/confidential-containers/containerd"}
containerd_version=${containerd_version:-"v1.6.6.0"}
containerd_dir="$(mktemp -d -t containerd-XXXXXXXXXX)/containerd"

registry="${registry:-quay.io/confidential-containers/container-engine-for-cc-payload}"

supported_arches=(
	"linux/amd64"
	"linux/s390x"
	"linux/ppc64le"
)

function setup_env_for_arch() {
	case "$1" in
		"linux/amd64") 
			kernel_arch="x86_64"
			golang_arch="amd64"
			;;
		"linux/s390x")
			kernel_arch="s390x"
			golang_arch="s390x"
			;;
		"linux/ppc64le")
			kernel_arch="ppc64le"
			golang_arch="ppc64le"
			;;
		(*) echo "$1 is not supported" > /dev/stderr && exit 1
	esac
		
}

function build_containerd_payload() {
	pushd "${script_dir}/.."

	tag=$(git rev-parse HEAD)

	for arch in ${supported_arches[@]}; do
		setup_env_for_arch "${arch}"

		echo "Building containerd payload image for ${arch}"
		docker buildx build \
			--build-arg ARCH="${golang_arch}" \
			--build-arg VERSION="${containerd_version}" \
			-f "containerd/Dockerfile" \
			-t "${registry}:${kernel_arch}-${tag}" \
			--platform="${arch}" \
			--load \
			.
		docker push "${registry}:${kernel_arch}-${tag}"
	done

	docker manifest create \
		${registry}:${tag} \
		--amend ${registry}:x86_64-${tag} \
		--amend ${registry}:s390x-${tag} \
		--amend ${registry}:ppc64le-${tag}

	docker manifest create \
		${registry}:latest \
		--amend ${registry}:x86_64-${tag} \
		--amend ${registry}:s390x-${tag} \
		--amend ${registry}:ppc64le-${tag}

	docker manifest push ${registry}:${tag}
	docker manifest push ${registry}:latest

	popd
}

function main() {
	build_containerd_payload
}

main "$@"
