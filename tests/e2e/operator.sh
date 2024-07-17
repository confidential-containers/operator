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
project_dir="$(readlink -f "${script_dir}"/../..)"

source "${script_dir}/lib.sh"

readonly ccruntime_overlay_basedir="${project_dir}/config/samples/ccruntime"
# The operator namespace.
readonly op_ns="confidential-containers-system"
# There should be a registry running locally on port 5000.
export IMG=localhost:5000/cc-operator:latest
export PRE_INSTALL_IMG=localhost:5000/reqs-payload

# Build the operator and push images to a local registry.
#
build_operator () {
	start_local_registry

	# Note: git config --global --add safe.directory will always
	# append the target to .gitconfig without checking the
	# existence of the target,
	# so it's better to check it before adding the target repo.
	local sd
	sd="$(git config --global --get safe.directory "${project_dir}" || true)"
	if [ "${sd}" == "" ]; then
		echo "::debug:: Add repo ${project_dir} to git's safe.directory"
		git config --global --add safe.directory "${project_dir}"
	else
		echo "::debug:: Repo ${project_dir} already in git's safe.directory"
	fi

	pushd "$project_dir" >/dev/null
	make docker-build
	make docker-push
	popd >/dev/null
}

# Build the reqs-payload and push images to a local registry.
#
build_pre_install_img() {
	start_local_registry

	pushd "${project_dir}/install/pre-install-payload" >/dev/null
	make reqs-image registry="${PRE_INSTALL_IMG}" \
		extra_docker_manifest_flags="--insecure"
	popd >/dev/null
}

# CoCo 0.8.0 onward requires containerd 1.7+ and the operator will rely on the
# system's installed version by default. Optionally, switching the
# INSTALL_OFFICIAL_CONTAINERD environment variable to "true" tells the pre-reqs
# script to install an official containerd that matches the 1.7+ requirement.
# Because some of our CI systems are running on Ubuntu 20.04 and CentOS Stream8,
# and those OSes comes with containerd 1.6, this function is used to flip the
# INSTALL_OFFICIAL_CONTAINERD's value to true, i.e., tells the operator to
# installation to deploy containerd too.
#
handle_older_containerd() {
	command -v containerd >/dev/null || return
	local version
	version=$(containerd -v | awk '{ print $3 }' | sed 's/^v//')
	echo "::debug:: system's containerd version: $version"
	if [[ "$version" =~ ^1.6 || "$version" =~ ^1.5 ]]; then
		echo "::warning:: Old system's containerd ($version). Configuring the operator to install a newer one"
		pushd "$project_dir" >/dev/null
		for kfile in $(find config/ -name "kustomization.yaml" \
			-exec grep -l INSTALL_OFFICIAL_CONTAINERD {} \;);do
			sed -i '/INSTALL_OFFICIAL_CONTAINERD/!b;n;s/false/true/;' "$kfile"
		done
		popd >/dev/null
	fi
}

# Install the operator.
#
install_operator() {
	start_local_registry

	# The node should be 'worker' labeled
	local label="node.kubernetes.io/worker"
	if ! kubectl get node "$SAFE_HOST_NAME" -o jsonpath='{.metadata.labels}' \
		| grep -q "$label"; then
		kubectl label node "$SAFE_HOST_NAME" "$label="
	fi

	handle_older_containerd

	pushd "$project_dir" >/dev/null
	# We should use a locally built image for operator.
	kustomization_set_image config/manager controller "${IMG}"
	kubectl apply -k config/default
	popd >/dev/null

	# Wait the operator controller to be running.
	local controller_pod="cc-operator-controller-manager"
	local cmd="kubectl get pods -n '$op_ns' --no-headers |"
	cmd+="egrep -q ${controller_pod}.*'\<Running\>'"
	if ! wait_for_process 120 10 "$cmd"; then
		echo "::error:: ${controller_pod} pod is not running"

		local pod_id
		pod_id="$(get_pods_regex "$controller_pod" "$op_ns")"
		echo "::debug:: Pod $pod_id"
		debug_pod "$pod_id" "$op_ns"

		return 1
	fi
}

# Install the CC runtime.
#
install_ccruntime() {
	local runtimeclass="${RUNTIMECLASS:-kata-qemu}"
	local overlay_dir="${ccruntime_overlay_basedir}/${ccruntime_overlay}"

	# Use the built pre-install image
	kustomization_set_image  "${ccruntime_overlay_basedir}/default" \
		"quay.io/confidential-containers/reqs-payload" \
		"${PRE_INSTALL_IMG}"

	pushd "$overlay_dir" >/dev/null
	kubectl create -k .
	popd >/dev/null

	local pod=""
	local cmd=""
	for pod in cc-operator-daemon-install cc-operator-pre-install-daemon; do
		cmd="kubectl get pods -n '$op_ns' --no-headers |"
		cmd+="egrep -q ${pod}.*'\<Running\>'"
		if ! wait_for_process 600 30 "$cmd"; then
			echo "::error:: $pod pod is not running"
			local pod_id
			pod_id="$(get_pods_regex "$pod" "$op_ns")"
			echo "::debug:: Pod $pod_id"
			debug_pod "$pod_id" "$op_ns"

			return 1
		fi
	done

	# Check if the runtime is up.
	# There could be a case where it is not even if the pods above are running.
	local cmd="kubectl get runtimeclass | grep -q ${runtimeclass}"
	if ! wait_for_process 300 30 "$cmd"; then
		echo "::error:: runtimeclass ${runtimeclass} is not up"
		return 1
	fi
	# To keep operator running, we should resume registry stopped during containerd restart.
	start_local_registry
}

# Uninstall the CC runtime.
#
uninstall_ccruntime() {
	pushd "${ccruntime_overlay_basedir}/${ccruntime_overlay}" >/dev/null
	kubectl delete -k .
	popd >/dev/null

	# Wait and ensure ccruntime pods are gone
	# (ensure failing kubectl keeps iterating)
	local cmd="_OUT=\$(sudo -E kubectl get pods -n '$op_ns')"
	cmd+=" && ! echo \$_OUT | grep -q -e cc-operator-daemon-install"
	cmd+=" -e cc-operator-pre-install-daemon"
	if ! wait_for_process 720 30 "$cmd"; then
		echo "::error:: there are ccruntime pods still running"
		echo "::group::Describe pods from $op_ns namespace"
		kubectl -n "$op_ns" describe pods || true
		echo "::endgroup::"

		return 1
	fi

	# Runtime classes should be gone
	! kubectl get --no-headers runtimeclass 2>/dev/null | grep -q kata

	# Labels should be gone
	if kubectl get nodes "$SAFE_HOST_NAME" -o jsonpath='{.metadata.labels}' | \
		grep -q -e cc-preinstall -e katacontainers.io; then
		echo "::error:: there are labels left behind"
		kubectl get nodes "$SAFE_HOST_NAME" -o jsonpath='{.metadata.labels}'

		return 1
	fi
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
		echo "::error:: expected image ${old} in ${overlay_dir}/kustomization.yaml"
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
		docker start "$registry_container" >/dev/null
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
	kubectl delete -k config/default
	popd >/dev/null

	# Wait and ensure the controller pod is gone
	# (ensure failing kubectl keeps iterating)
	local cmd="_OUT=\$(sudo -E kubectl get pods -n '$op_ns')"
	cmd+="&& ! echo \$_OUT | grep -q -e cc-operator-controller-manager"
	if ! wait_for_process 180 30 "$cmd"; then
		echo "::error:: the controller manager is still running"
		local pod_id
		pod_id="$(get_pods_regex "$pod" "$op_ns")"
		echo "::debug:: Pod $pod_id"
		debug_pod "$pod_id" "$op_ns"

		return 1
	fi
}

# Wait for no new/restarted pod in 3x21s (20s is the liveness probe)
#
wait_for_stabilization() {
	declare -A restart_counts
	local iteration=0
	local count=0
	while true; do
		local change=0
		local pod_info
		pod_info=$(kubectl get pods -n "$op_ns" -o=jsonpath='{range .items[*]}{.metadata.name}{" "}{range .status.containerStatuses[*]}{.name}{" "}{.restartCount}{"\n"}{end}{end}')

		while read -r pod container restart_count; do
			if [ "${restart_counts[$pod-$container]--1}" != "$restart_count" ]; then
				echo "::debug:: Pod: $pod, Container: $container, Restart count: $restart_count"
				restart_counts["$pod-$container"]=$restart_count
				change=1
			fi
		done <<< "$pod_info"

		[ $change -eq 0 ] && ((iteration+=1))

		if [ $iteration -gt 3 ]; then
			echo "::info:: No new restarts in 3x21s, proceeding..."
			break
		elif [ $count -gt 20 ]; then
			echo "::error:: Pods are still restarting after 20x21s, bailing out!"
			return 1
		fi

		((count+=1))
		sleep 21
	done
}


usage() {
	cat <<-EOF
	Utility to build/install/uninstall the operator.

	Use: $0 [-h|--help] [command], where:
	-h | --help : show this usage
	command : optional command (build and install by default). Can be:
	 "build": build only,
	 "install": install only,
	 "wait_for_stabilization": wait for CoCo pods to be stable
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
		wait_for_stabilization
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
			uninstall)
				uninstall_ccruntime
				uninstall_operator
				;;
			wait_for_stabilization)
				wait_for_stabilization
				;;
			*)
				echo "::error:: Unknown command '$1'"
				usage && exit 1
		esac
	fi
}

main "$@"
