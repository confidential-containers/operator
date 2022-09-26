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

# Build the operator and push images to a local registry.
#
build_operator () {
	start_local_registry

	pushd "$project_dir" >/dev/null
	make docker-build
	make docker-push
	popd >/dev/null
}

# Install the operator.
#
install_operator() {
	start_local_registry

	# The node should be 'worker' labeled
	local label="node-role.kubernetes.io/worker"
	if ! kubectl get node "$(hostname)" -o jsonpath='{.metadata.labels}' \
		| grep -q "$label"; then
		kubectl label node "$(hostname)" "$label="
	fi

	pushd "$project_dir" >/dev/null
	kubectl apply -f deploy/deploy.yaml
	popd >/dev/null

	# Wait the operator controller to be running.
	local cmd="kubectl get pods -n "$op_ns" --no-headers |"
	cmd+="egrep -q cc-operator-controller-manager.*'\<Running\>'"
	if ! wait_for_process 120 10 "$cmd"; then
		echo "ERROR: operator-controller-manager pod is not running"
		return 1
	fi
}

# Install the CC runtime.
#
install_ccruntime() {
	pushd "$project_dir" >/dev/null
	kubectl create -f config/samples/ccruntime.yaml
	popd >/dev/null

	local pod=""
	local cmd=""
	for pod in cc-operator-daemon-install cc-operator-pre-install-daemon; do
		cmd="kubectl get pods -n "$op_ns" --no-headers |"
		cmd+="egrep -q ${pod}.*'\<Running\>'"
		if ! wait_for_process 600 30 "$cmd"; then
			echo "ERROR: $pod pod is not running"
			return 1
		fi
	done

	# TODO: check the runtime is up.
	# kubectl get runtimeclass
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
	kubectl delete -f config/samples/ccruntime.yaml
	kubectl delete -f deploy/deploy.yaml
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
	if [ $# -eq 0 ]; then
		build_operator
		install_operator
		install_ccruntime
	else
		case $1 in
			-h|--help) usage && exit 0;;
			build) build_operator;;
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
