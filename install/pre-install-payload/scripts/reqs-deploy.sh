#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset

containerd_config="/etc/containerd/config.toml"

die() {
	msg="$*"
	echo "ERROR: $msg" >&2
	exit 1
}

function host_systemctl() {
	nsenter --target 1 --mount systemctl "${@}"
}

function get_container_engine() {
	local container_engine=$(kubectl get node "$NODE_NAME" -o jsonpath='{.status.nodeInfo.containerRuntimeVersion}' | awk -F '[:]' '{print $1}')
	if [ "${container_engine}" != "containerd" ]; then
		die "${container_engine} is not yet supported"
	fi

	echo "$container_engine"
}

function set_container_engine() {
	# Those are intentionally not set as local

	container_engine=$(get_container_engine)
}

function install_containerd_artefacts() {
	flavour=${1}

	echo "Copying ${flavour} containerd-for-cc artifacts onto host"

	local artifacts_dir="/opt/confidential-containers-pre-install-artifacts"

	install -D -m 755 ${artifacts_dir}/opt/confidential-containers/bin/${flavour}-containerd /opt/confidential-containers/bin/containerd
	install -D -m 644 ${artifacts_dir}/etc/systemd/system/containerd.service.d/containerd-for-cc-override.conf /etc/systemd/system/containerd.service.d/containerd-for-cc-override.conf
}

function install_coco_containerd_artefacts() {
	install_containerd_artefacts "coco"
}

function install_official_containerd_artefacts() {
	install_containerd_artefacts "official"
}

function install_vfio_gpu_containerd_artefacts() {
	install_containerd_artefacts "vfio-gpu"
}

function install_nydus_snapshotter_artefacts() {
	echo "Copying nydus-snapshotter artifacts onto host"
	install -D -m 755 ${artifacts_dir}/opt/confidential-containers/bin/containerd-nydus-grpc /opt/confidential-containers/bin/containerd-nydus-grpc
	install -D -m 755 ${artifacts_dir}/opt/confidential-containers/bin/nydus-overlayfs /opt/confidential-containers/bin/nydus-overlayfs
	#NOTE: symlink nydus-overlayfs to /usr/local/bin or /usr/bin
	ln -s /opt/confidential-containers/bin/nydus-overlayfs /usr/bin/nydus-overlayfs
	install -D -m 644 ${artifacts_dir}/opt/confidential-containers/share/nydus-snapshotter/config-coco-guest-pulling.toml /opt/confidential-containers/share/nydus-snapshotter/config-coco-guest-pulling.toml
	install -D -m 644 ${artifacts_dir}/etc/systemd/system/nydus-snapshotter.service /etc/systemd/system/nydus-snapshotter.service

	host_systemctl daemon-reload
	host_systemctl enable nydus-snapshotter.service

	configure_nydus_snapshotter_for_containerd

	restart_systemd_service
}

function install_artifacts() {
	if [ "${INSTALL_COCO_CONTAINERD}" = "true" ]; then
		install_coco_containerd_artefacts
	fi

	if [ "${INSTALL_OFFICIAL_CONTAINERD}" = "true" ]; then
		install_coco_containerd_artefacts
	fi

	if [ "${INSTALL_VFIO_GPU_CONTAINERD}" = "true" ]; then
		install_vfio_gpu_containerd_artefacts
	fi

	if [ "${INSTALL_NYDUS_SNAPSHOTTER}" = "true" ]; then
		install_nydus_snapshotter_artefacts
	fi
}

function uninstall_containerd_artefacts() {
	echo "Removing containerd-for-cc artifacts from host"

	echo "Removing the systemd drop-in file"
	rm -f /etc/systemd/system/${container_engine}.service.d/${container_engine}-for-cc-override.conf
	echo "Removing the systemd drop-in file's directory, if empty"
	if [ -d /etc/systemd/system/${container_engine}.service.d ]; then
		rmdir --ignore-fail-on-non-empty /etc/systemd/system/${container_engine}.service.d
	fi

	restart_systemd_service

	echo "Removing the containerd binary"
	rm -f /opt/confidential-containers/bin/containerd
	echo "Removing the /opt/confidential-containers/bin directory"
	if [ -d /opt/confidential-containers/bin ]; then
		rmdir --ignore-fail-on-non-empty -p /opt/confidential-containers/bin
	fi
}

function uninstall_nydus_snapshotter_artefacts() {
	remove_nydus_snapshotter_from_containerd

	host_systemctl disable nydus-snapshotter.service
	host_systemctl daemon-reload

	rm -rf /etc/systemd/system/nydus-snapshotter.service

	restart_systemd_service

	echo "Removing nydus-snapshotter artifacts from host"
	rm -f /opt/confidential-containers/bin/containerd-nydus-grpc
	rm -f /opt/confidential-containers/bin/nydus-overlayfs
	#NOTE: remove the link of nydus-overlayfs in /usr/local/bin or /usr/bin
	rm /usr/bin/nydus-overlayfs
	rm -f /opt/confidential-containers/share/remote-snapshotter/config-coco-guest-pulling.toml
}	


function uninstall_artifacts() {
	if [ "${INSTALL_COCO_CONTAINERD}" = "true" ] || [ "${INSTALL_OFFICIAL_CONTAINERD}" = "true" ] || [ "${INSTALL_VFIO_GPU_CONTAINERD}" = "true" ]; then
		uninstall_containerd_artefacts
	fi
}

function restart_systemd_service() {
	host_systemctl daemon-reload
	echo "Restarting ${container_engine}"
	host_systemctl restart "${container_engine}"
}

function configure_nydus_snapshotter_for_containerd() {
	echo "configure nydus snapshotter for containerd"

	if [ ! -f "$containerd_config" ]; then
		die "failed to find containerd config"
	fi

	if [ "${INSTALL_NYDUS_SNAPSHOTTER}" = "true" ]; then
		echo "Plug nydus snapshotter into containerd"
		snapshotter_socket="/run/containerd-nydus/containerd-nydus-grpc.sock"
	fi
	proxy_config="  [proxy_plugins.$SNAPSHOTTER]\n    type = \"snapshot\"\n    address = \"${snapshotter_socket}\""

	if grep -q "\[proxy_plugins\]" "$containerd_config"; then
		sed -i '/\[proxy_plugins\]/a\'"$proxy_config" "$containerd_config"
	else
		echo -e "[proxy_plugins]" >>"$containerd_config"
		echo -e "$proxy_config" >>"$containerd_config"
	fi
}

function remove_nydus_snapshotter_from_containerd() {
	echo "Remove nydus snapshotter from containerd"

	sed -i '/\[proxy_plugins.nydus\]/,/address = "\/run\/containerd-nydus\/containerd-nydus-grpc\.sock"/d' "$containerd_config"
}

label_node() {
	case "${1}" in
	install)
		kubectl label node "${NODE_NAME}" cc-preinstall/done=true
		;;
	uninstall)
		kubectl label node "${NODE_NAME}" cc-postuninstall/done=true
		;;
	*)
		;;
	esac
}

function print_help() {
	echo "Help: ${0} [install/uninstall]"
}

function main() {
	echo "INSTALL_COCO_CONTAINERD: ${INSTALL_COCO_CONTAINERD}"
	echo "INSTALL_OFFICIAL_CONTAINERD: ${INSTALL_OFFICIAL_CONTAINERD}"
	echo "INSTALL_VFIO_GPU_CONTAINERD: ${INSTALL_VFIO_GPU_CONTAINERD}"
	echo "INSTALL_NYDUS_SNAPSHOTTER: ${INSTALL_NYDUS_SNAPSHOTTER}"

	# script requires that user is root
	local euid=$(id -u)
	if [ ${euid} -ne 0 ]; then
		die "This script must be run as root"
	fi

	local action=${1:-}
	if [ -z "${action}" ]; then
		print_help && die ""
	fi

	set_container_engine

	case "${action}" in
	install)
		install_artifacts
		restart_systemd_service
		;;
	uninstall)
		# Adjustment for s390x (clefos:7)
		# It is identified that a node is not labeled during post-uninstall,
		# if the function is called after container engine is restarted by systemctl.
		# This results in having the uninstallation not triggered.
		if [ "$(uname -m)" = "s390x" ]; then
			label_node "${action}"
		fi
		uninstall_artifacts
		;;
	*)
		print_help
		;;
	esac

	label_node "${action}"


	# It is assumed this script will be called as a daemonset. As a result, do
	# not return, otherwise the daemon will restart and reexecute the script.
	sleep infinity
}

main "$@"
