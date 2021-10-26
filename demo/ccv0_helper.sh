#!/bin/bash -e
# Adapted from 
# https://raw.githubusercontent.com/kata-containers/kata-containers/CCv0/docs/how-to/ccv0.sh
# to use with kata-deploy kind of installs

export BUNDLE_DIR_PREFIX="/opt/kata/oci-bundle"
export BUNDLE_DIR="/opt/kata/oci-bundle/bundle"

export CONTAINER_ID="${CONTAINER_ID:-0123456789}"

get_ids() {

	guest_cid=$(ps -ef | grep qemu-system-x86_64 | egrep -o "guest-cid=[0-9]*" | cut -d= -f2) && sandbox_id=$(ps -ef | grep qemu | egrep -o "sandbox-[^,][^,]*" | sed 's/sandbox-//g' | awk '{print $1}')

}

open_kata_console() { 
	# Get Ids
	get_ids

	# Connect to VM console
        cd /var/run/vc/vm/${sandbox_id} && socat "stdin,raw,echo=0,escape=0x11" "unix-connect:console.sock"

}


open_kata_shell() {
	# Get Ids
	get_ids
        #Get VM shell
        /opt/kata/bin/kata-runtime exec ${sandbox_id}

}

build_bundle_dir_if_necessary() {

    if [ ! -d "${BUNDLE_DIR}" ]; then
	 mkdir -p ${BUNDLE_DIR_PREFIX}
	 wget https://github.com/confidential-containers/operator/raw/ccv0-demo/demo/oci-bundle.tar.gz
         tar xvf oci-bundle.tar.gz -C ${BUNDLE_DIR_PREFIX}
    fi

}
agent_pull_image() {
	 get_ids
	 build_bundle_dir_if_necessary

	# Pull image
	/opt/kata/bin/kata-agent-ctl -l debug connect --bundle-dir "${BUNDLE_DIR}" --server-address "vsock://${guest_cid}:1024" -c  "PullImage image=${PULL_IMAGE} cid=${CONTAINER_ID} source_creds=${SOURCE_CREDS}"

}

agent_create_container() {

	 get_ids
	 build_bundle_dir_if_necessary
	# Create container
	/opt/kata/bin/kata-agent-ctl -l debug connect --bundle-dir "${BUNDLE_DIR}" --server-address "vsock://${guest_cid}:1024" -c "CreateContainer cid=${CONTAINER_ID}"

}

usage() {
    exit_code="$1"
    cat <<EOT
Usage:
    ${script_name} [options] <command>
Commands:
- help:                         Display this help
- open_kata_console:            Stream the kata VM console
- open_kata_shell:              Open a shell into the kata VM
- agent_pull_image:             Run PullImage command against the agent with kata-agent-ctl
- agent_create_container:       Run CreateContainer command against the agent with kata-agent-ctl

Options:
    -h: Display this help
EOT
    # if script sourced don't exit as this will exit the main shell, just return instead
          [[ $_ != $0 ]] && return "$exit_code" || exit "$exit_code"
}

main() {
    while getopts "dh" opt; do
        case "$opt" in
            h)
                usage 0
                ;;
            \?)
                echo "Invalid option: -$OPTARG" >&2
                usage 1
                ;;
        esac
    done

    shift $((OPTIND - 1))

    subcmd="${1:-}"

    [ -z "${subcmd}" ] && usage 1

    case "${subcmd}" in
        open_kata_console)
            open_kata_console
            ;;
        open_kata_shell)
            open_kata_shell
            ;;
        agent_pull_image)
            agent_pull_image
            ;;
        agent_create_container)
            agent_create_container
            ;;
        *)
            usage 1
            ;;
    esac
}

main $@       
