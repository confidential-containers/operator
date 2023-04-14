#!/bin/bash
#
# Copyright Confidential Containers Contributors
#
# SPDX-License-Identifier: Apache-2.0
#
set -o errexit
set -o nounset
set -o pipefail

script_dir="$(dirname "$(readlink -f "$0")")"
project_dir="$(readlink -f ${script_dir}/../)"
ccruntime_overlays="${project_dir}/config/samples/ccruntime"

readonly registry="quay.io/confidential-containers"
readonly runtime_img="${registry}/runtime-payload"
readonly runtime_img_ci="${registry}/runtime-payload-ci"
readonly preinstall_img="quay.io/confidential-containers/container-engine-for-cc-payload"

runtime_payload_commit=""
preinstall_payload_commit=""
ci_images=0

usage() {
	cat <<-EOF
	This script updates the runtime-payload and pre-install images on all the
	ccruntime deployment files.

	Usage: $0 [-h] [-c] [-p commit] [-k commit]
	Where
	-c: use CI images
	-h: print this help message
	-p: preinstall image commit
	-k: runtime-payload image commit
	EOF
}

parse_opts() {
	while getopts "chp:k:" OPT; do
		case "$OPT" in
			k) runtime_payload_commit="$OPTARG" ;;
			c) ci_images=1 ;;
			p) preinstall_payload_commit="$OPTARG" ;;
			h) usage && exit 0 ;;
			*) usage && exit 1 ;;
		esac
	done
}

update_img() {
	local overlay_dir="$1"
	local old_img="$2"
	local img="$3"
	local tag="$4"
	local new_img="${img}:${tag}"

	pushd "$overlay_dir" >/dev/null
	if [ -n "$img" ];then
		replace_str="${old_img}=${img}:${tag}"
	else
		# In case of replacing only the tag.
		replace_str="${old_img}:${tag}"
	fi

	kustomize edit set image "$replace_str"

	# kustomize is going to silently keep the kustomization.toml intact if
	# the to be replaced image does not match. We expect the image to
	# always be replaced otherwise something is wrong.
	if ! grep -q "${img}" kustomization.yaml ||
		! grep -q "${tag}" kustomization.yaml; then
		echo "[ERROR] New image not set in $overlay_dir/kustomization.yaml"
		exit 1
	fi

	popd >/dev/null
}

main() {
	parse_opts $@

	if ! command -v kustomize >/dev/null; then
		echo "[ERROR] You must install 'kustomize' to run this script."
		exit 1
	fi

	if [ $ci_images -eq 0 ];then
		local new_img_name="$runtime_img"
	else
		local new_img_name="$runtime_img_ci"
	fi

	local base_tag="kata-containers-${runtime_payload_commit}"

	for overlay_dir in ${ccruntime_overlays}/*; do
		case $(basename "$overlay_dir") in
			base)
				echo "Skipping overlay: $overlay_dir"
				continue ;;
			default|peer-pods) new_img_tag="${base_tag}-x86_64" ;;
			s390x) new_img_tag="${base_tag}-s390x" ;;
			*)
				echo "[Warning] Do not know how to handle: $overlay_dir"
				continue ;;
		esac

		echo "Updating overlay: $overlay_dir"

		# Update the runtime-payload image
		if [ -n "$runtime_payload_commit" ];then
			update_img "$overlay_dir" "$runtime_img" "$new_img_name" \
				"$new_img_tag"
		fi

		# Update the preinstall image
		if [ -n "$preinstall_payload_commit" ];then
			update_img "$overlay_dir" "$preinstall_img" "" \
				"$preinstall_payload_commit"
		fi
	done
}

main $@
