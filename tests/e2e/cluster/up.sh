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

source "${script_dir}/../lib.sh"

# Run kubeadm init and KUBECONFIG is exported on success.
#
init_kubeadm() {
	local kubeadm_config_file="/etc/kubeadm/kubeadm.conf"
	# Bootstrap the control-plane node.
	kubeadm init --config "${kubeadm_config_file}"

	export KUBECONFIG=/etc/kubernetes/admin.conf

	# TODO: if we want to run as a regular user.
	#mkdir -p $HOME/.kube
	#sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
	#sudo chown $(id -u):$(id -g) $HOME/.kube/config

	# TODO: wait node to show up
	#kubectl get nodes
}

# Configure the cluster network with flannel.
#
configure_flannel() {
	local flannel_ns="kube-flannel"

	kubectl apply -f /opt/flannel/kube-flannel.yaml

	if ! wait_pods "$flannel_ns"; then
		echo "ERROR: pods didn't show up in $flannel_ns"
		return 1
	fi

	if ! check_pods_are_ready "$flannel_ns" 120; then
		echo "ERROR: flannel pods are not ready"
		return 1
	fi
	# TODO: check coredns is up and running too
}

main() {
	init_kubeadm
	configure_flannel
	check_node_is_ready

	# Untaint the node so that pods can be scheduled on it.
	for role in master control-plane; do
		kubectl taint nodes "$SAFE_HOST_NAME" \
			"node-role.kubernetes.io/$role:NoSchedule-" || true
	done

	kubectl label node "$SAFE_HOST_NAME" node.kubernetes.io/worker=
}

main "$@"
