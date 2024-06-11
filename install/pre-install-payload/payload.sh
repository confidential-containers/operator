#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset

script_dir=$(dirname "$(readlink -f "$0")")

coco_containerd_repo=${coco_containerd_repo:-"https://github.com/confidential-containers/containerd"}
official_containerd_repo=${official_containerd_repo:-"https://github.com/containerd/containerd"}
vfio_gpu_containerd_repo=${vfio_gpu_containerd_repo:-"https://github.com/confidential-containers/containerd"}
nydus_snapshotter_repo=${nydus_snapshotter_repo:-"https://github.com/containerd/nydus-snapshotter"}
containerd_dir="$(mktemp -d -t containerd-XXXXXXXXXX)/containerd"
extra_docker_manifest_flags="${extra_docker_manifest_flags:-}"
http_proxy="${http_proxy:-}"
https_proxy="${https_proxy:-}"

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
		*) echo "$1 is not supported" >/dev/stderr && exit 1 ;;
	esac
}

function purge_previous_manifests() {
	local manifest
	local sanitised_manifest
	manifest="${1}"
	# We need to sanitise the name by:
	# * Replacing:
	#   * '/' by '_'
	#   * ':' by '-'

	sanitised_manifest="$(echo ${manifest} | sed 's|/|_|g' | sed 's|:|-|g')"
	rm -rf "${HOME}/.docker/manifests/${sanitised_manifest}" || true
}

function build_payload() {
	pushd "${script_dir}"
	local tag

	tag=$(git rev-parse HEAD)

	manifest_args=()
	for arch in "${supported_arches[@]}"; do
		setup_env_for_arch "${arch}"

		echo "Building containerd payload image for ${arch}"
		docker buildx build \
			--build-arg HTTP_PROXY="${http_proxy}" \
			--build-arg HTTPS_PROXY="${https_proxy}" \
			--build-arg ARCH="${golang_arch}" \
			--build-arg COCO_CONTAINERD_VERSION="${coco_containerd_version}" \
			--build-arg COCO_CONTAINERD_REPO="${coco_containerd_repo}" \
			--build-arg OFFICIAL_CONTAINERD_VERSION="${official_containerd_version}" \
			--build-arg OFFICIAL_CONTAINERD_REPO="${official_containerd_repo}" \
			--build-arg VFIO_GPU_CONTAINERD_VERSION="${vfio_gpu_containerd_version}" \
			--build-arg VFIO_GPU_CONTAINERD_REPO="${vfio_gpu_containerd_repo}" \
			--build-arg NYDUS_SNAPSHOTTER_VERSION="${nydus_snapshotter_version}" \
			--build-arg NYDUS_SNAPSHOTTER_REPO="${nydus_snapshotter_repo}" \
			-t "${registry}:${kernel_arch}-${tag}" \
			--platform="${arch}" \
			--load \
			.
		docker push "${registry}:${kernel_arch}-${tag}"
		manifest_args+=(--amend "${registry}:${kernel_arch##*/}-${tag}")
	done

	purge_previous_manifests "${registry}:${tag}"
	purge_previous_manifests "${registry}:latest"

	docker manifest create ${extra_docker_manifest_flags} \
		"${registry}:${tag}" \
		"${manifest_args[@]}"

	docker manifest create ${extra_docker_manifest_flags} \
		"${registry}:latest" \
		"${manifest_args[@]}"

	docker manifest push ${extra_docker_manifest_flags} "${registry}:${tag}"
	docker manifest push ${extra_docker_manifest_flags} "${registry}:latest"

	popd
}

function main() {
	build_payload
}

main "$@"
