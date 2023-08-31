#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset

script_dir=$(dirname "$(readlink -f "$0")")

coco_containerd_repo=${coco_containerd_repo:-"https://github.com/confidential-containers/containerd"}
coco_containerd_version=${coco_containerd_version:-"v1.6.6.0"}
official_containerd_repo=${official_containerd_repo:-"https://github.com/containerd/containerd"}
official_containerd_version=${official_containerd_version:-"1.7.0"}
containerd_dir="$(mktemp -d -t containerd-XXXXXXXXXX)/containerd"
extra_docker_manifest_flags="${extra_docker_manifest_flags:-}"

registry="${registry:-quay.io/confidential-containers/reqs-payload}"

supported_arches=(
	"linux/amd64"
	"linux/s390x"
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
		(*) echo "$1 is not supported" > /dev/stderr && exit 1
	esac
		
}

function purge_previous_manifests() {
	manifest=${1}
	
	# We need to sanitise the name by:
	# * Replacing:
	#   * '/' by '_'
	#   * ':' by '-'
	
	sanitised_manifest="$(echo ${manifest} | sed 's|/|_|g' | sed 's|:|-|g')"
	rm -rf ${HOME}/.docker/manifests/${sanitised_manifest}
}

function build_payload() {
	pushd "${script_dir}"

	tag=$(git rev-parse HEAD)

	for arch in ${supported_arches[@]}; do
		setup_env_for_arch "${arch}"

		echo "Building containerd payload image for ${arch}"
		docker buildx build \
			--build-arg ARCH="${golang_arch}" \
			--build-arg COCO_CONTAINERD_VERSION="${coco_containerd_version}" \
			--build-arg COCO_CONTAINERD_REPO="${coco_containerd_repo}" \
			--build-arg OFFICIAL_CONTAINERD_VERSION="${official_containerd_version}" \
			--build-arg OFFICIAL_CONTAINERD_REPO="${official_containerd_repo}" \
			-t "${registry}:${kernel_arch}-${tag}" \
			--platform="${arch}" \
			--load \
			.
		docker push "${registry}:${kernel_arch}-${tag}"
	done

	purge_previous_manifests ${registry}:${tag}
	purge_previous_manifests ${registry}:latest

	docker manifest create ${extra_docker_manifest_flags} \
		${registry}:${tag} \
		--amend ${registry}:x86_64-${tag} \
		--amend ${registry}:s390x-${tag}

	docker manifest create ${extra_docker_manifest_flags} \
		${registry}:latest \
		--amend ${registry}:x86_64-${tag} \
		--amend ${registry}:s390x-${tag}

	docker manifest push ${extra_docker_manifest_flags} ${registry}:${tag}
	docker manifest push ${extra_docker_manifest_flags} ${registry}:latest

	popd
}

function main() {
	build_payload
}

main "$@"
