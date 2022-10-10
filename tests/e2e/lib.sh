#!/bin/bash
#
# Copyright Confidential Containers Contributors
#
# SPDX-License-Identifier: Apache-2.0
#

# Wait until the node is ready. It is set a timeout of 180 seconds.
#
check_node_is_ready() {
	wait_for_process 180 10 "kubectl get nodes | grep -q '\<Ready\>'"
}

# Check that all pods running on a given namespace are ready.
#
# Parameters:
#	$1 - the namespace
#	$2 - timeout in seconds to check each pod (default to 30)
#
check_pods_are_ready() {
	local ns="$1"
	# Some pods take longer than the default 30s.
	local timeout="${2:-30}s"

	local pods=($(kubectl get -n "$ns" pods \
		-o jsonpath='{.items[*].metadata.name}'))

	# At least one pod is expected.
	if [ ${#pods[@]} -eq 0 ]; then
		echo "ERROR: no pods found in $ns"
		return 1
	fi

	for p in ${pods[@]}; do
		kubectl wait "--timeout=$timeout" -n "$ns" \
			--for=condition=Ready "pod/${p}" >/dev/null
	done
}

# Print information about the pod in namespace.
#
# Parameters
#	$1 - the pod id.
#	$2 - (optional) the namespace.
#
debug_pod() {
	local pod="$1"
	local ns="$2"

	set -x
	kubectl describe "pods/$1" ${ns:+"-n $ns"} || true
	kubectl logs "pods/$1" ${ns:+"-n $ns"} || true
	set +x
}

# Return a list of pod ids (pod1 pod2 ... podn) that match the regex.
#
# Parameters:
#	$1 - the regex as accepted by grep.
#	$2 - (optional) the namespace. Otherwise search on default namespace.
#
get_pods_regex() {
	local regex="$1"
	local ns="$2"
	echo $(kubectl get pods ${ns:+-n "$ns"} --no-headers 2>/dev/null \
		| grep "$regex" | cut -d" " -f1)
}

# Wait for at least one pod to show up on the namespace.
#
# Parameters:
#	$1 - the namespace
#	$2 - (optional) wait time in seconds between each checking
#	     (default to 10)
#	$3 - (optional) checking counter (default to 6)
#
wait_pods() {
	local ns="$1"
	local wait_time="${2:-10}"
	local cnt="${3:-6}"

	local cmd="kubectl get pods -n "$ns" -o jsonpath='{.items[*].metadata.name}' || true"
	local pods=($(eval $cmd))
	while [ "${#pods[@]}" -eq 0 -a "$cnt" -gt 0 ]; do
		sleep "$wait_time"
		cnt=$(($cnt-1))
		pods=($(eval $cmd))
	done
	[ "${#pods[@]}" -gt 0 ]
}

# Wait until the command exit with success.
#
# Parameters:
#	$1 - the total wait time
#	$2 - the sleep time between runs
#	$3 - the command to run
#
# Note: copied from kata-containers/tests/lib/common.bash
#
wait_for_process() {
	wait_time="$1"
	sleep_time="$2"
	cmd="$3"
	while [ "$wait_time" -gt 0 ]; do
		if eval "$cmd"; then
			return 0
		else
			sleep "$sleep_time"
			wait_time=$((wait_time-sleep_time))
		fi
	done
	return 1
}
