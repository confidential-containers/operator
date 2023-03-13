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
project_dir="$(readlink -f ${script_dir}/../..)"

source "${script_dir}/lib.sh"

# The operator namespace.
readonly op_ns="confidential-containers-system"
# There should be a registry running locally on port 5000.
export IMG=localhost:5000/cc-operator
export PRE_INSTALL_IMG=localhost:5000/container-engine-for-cc-payload

# Build the operator and push images to a local registry.
#
build_operator () {
	start_local_registry

        # Note: git config --global --add safe.directory will always
        # append the target to .gitconfig without checking the
        # existence of the target,
        # so it's better to check it before adding the target repo.
        local sd="$(git config --global --get safe.directory ${project_dir} || true)"
        if [ "${sd}" == "" ]; then
                echo "Add repo ${project_dir} to git's safe.directory"
                git config --global --add safe.directory "${project_dir}"
        else
                echo "Repo ${project_dir} already in git's safe.directory"
        fi

	pushd "$project_dir" >/dev/null
	make docker-build
	make docker-push
	popd >/dev/null
}

# Build the container-engine-for-cc-payload and push images to a local registry.
#
build_pre_install_img() {
	start_local_registry

	pushd "${project_dir}/install/pre-install-payload" >/dev/null
	make containerd registry="${PRE_INSTALL_IMG}" \
		extra_docker_manifest_flags="--insecure"
	popd >/dev/null
}

# Install the operator.
#
install_operator() {
	start_local_registry

	# The node should be 'worker' labeled
	local label="node.kubernetes.io/worker"
	if ! kubectl get node "$(hostname)" -o jsonpath='{.metadata.labels}' \
		| grep -q "$label"; then
		kubectl label node "$(hostname)" "$label="
	fi

	pushd "$project_dir" >/dev/null
	# We should use a locally built image for operator.
	sed -i "s~\(.*newName: \).*~\1${IMG}~g" config/manager/kustomization.yaml
	kubectl apply -k config/default
	popd >/dev/null

	# Wait the operator controller to be running.
	local controller_pod="cc-operator-controller-manager"
	local cmd="kubectl get pods -n "$op_ns" --no-headers |"
	cmd+="egrep -q ${controller_pod}.*'\<Running\>'"
	if ! wait_for_process 120 10 "$cmd"; then
		echo "ERROR: ${controller_pod} pod is not running"

		local pod_id="$(get_pods_regex $controller_pod $op_ns)"
		echo "DEBUG: Pod $pod_id"
		debug_pod "$pod_id" "$op_ns"

		return 1
	fi
}

# Install the CC runtime.
#
install_ccruntime() {
	local runtimeclass="${RUNTIMECLASS:-kata-qemu}"
	local ccruntime_overlay_dir="${project_dir}/config/samples/ccruntime"
	local overlay_dir="${ccruntime_overlay_dir}/${ccruntime_overlay}"

	# Use the built pre-install image
	kustomization_set_image  "${ccruntime_overlay_dir}/default" \
		"quay.io/confidential-containers/container-engine-for-cc-payload" \
		"${PRE_INSTALL_IMG}"

	pushd "$overlay_dir" >/dev/null
	kubectl create -k .
	popd >/dev/null

	local pod=""
	local cmd=""
	for pod in cc-operator-daemon-install cc-operator-pre-install-daemon; do
		cmd="kubectl get pods -n "$op_ns" --no-headers |"
		cmd+="egrep -q ${pod}.*'\<Running\>'"
		if ! wait_for_process 600 30 "$cmd"; then
			echo "ERROR: $pod pod is not running"

			local pod_id="$(get_pods_regex $pod $op_ns)"
			echo "DEBUG: Pod $pod_id"
			debug_pod "$pod_id" "$op_ns"

			return 1
		fi
	done

	# Check if the runtime is up.
	# There could be a case where it is not even if the pods above are running.
	cmd="kubectl get runtimeclass | grep -q ${runtimeclass}"
	if ! wait_for_process 300 30 "$cmd"; then
		echo "ERROR: runtimeclass ${runtimeclass} is not up"
		return 1
	fi
	# To keep operator running, we should resume registry stopped during containerd restart.
	start_local_registry
}

# Set image on a kustomize's kustomization.yaml.
#
# Parameters:
#	$1 - path to the overlay directory
#	$2 - name of the old image
#	$3 - name of the new image
#
kustomization_set_image() {
	local overlay_dir="$1"
	local old="$2"
	local new="$3"

	pushd "$overlay_dir" >/dev/null
	# The kustomize tool will silently add a new image name if the old one does not exist,
	# and this can introduce false-positive on the tests. So let's check the old image really
	# exist.
	if ! grep -q "name: ${old}$" ./kustomization.yaml; then
		echo "ERROR: expected image ${old} in ${overlay_dir}/kustomization.yaml"
		return 1
	fi

	kustomize edit set image "${old}=${new}"
	popd >/dev/null
}

# Start a local registry where images can be stored.
# The ansible playbooks should start it however it can get stopped when,
# for example, the operator is unistalled.
#
start_local_registry() {
	# TODO: allow callers to override the container name, port..etc
	local registry_container="local-registry"

	if ! curl -s localhost:5000; then
		docker start local-registry >/dev/null
		local cnt=0
		while ! curl -s localhost:5000 -o $cnt -lt 5; do
			sleep 1
			cnt=$(($cnt+1))
		done
		[ $cnt -ne 5 ]
	fi
}

# Uninstall the operator and ccruntime.
#
uninstall_operator() {
	pushd "$project_dir" >/dev/null
	kubectl delete -k config/samples/ccruntime/${ccruntime_overlay}
	kubectl delete -k config/default
	popd >/dev/null
}

usage() {
	cat <<-EOF
	Utility to build/install/uninstall the operator.

	Use: $0 [-h|--help] [command], where:
	-h | --help : show this usage
	command : optional command (build and install by default). Can be:
	 "build": build only,
	 "install": install only,
	 "uninstall": uninstall the operator.
	EOF
}

main() {
	ccruntime_overlay="default"
	if [ "$(uname -m)" = "s390x" ]; then
		ccruntime_overlay="s390x"
	fi
	if [ $# -eq 0 ]; then
		build_operator
		install_operator
		build_pre_install_img
		install_ccruntime
	else
		case $1 in
			-h|--help) usage && exit 0;;
			build)
				build_operator
				build_pre_install_img
				;;
			install)
				install_operator
				install_ccruntime
				;;
			uninstall) uninstall_operator;;
			*)
				echo "Unknown command '$1'"
				usage && exit 1
		esac
	fi
}

main "$@"
