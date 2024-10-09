#!/bin/bash
#
# Copyright Confidential Containers Contributors
#
# SPDX-License-Identifier: Apache-2.0
#
set -o errexit
set -o nounset
set -o pipefail

# Source the list of yaml files that will be updated
source release-helper-yaml-list.sh

#
# This script automates the changes across the operator repo that have to be
# done during a confidential-containers release. These should correspond
# to several key steps in the release checklist:
#   https://github.com/confidential-containers/confidential-containers/blob/main/.github/ISSUE_TEMPLATE/release-check-list.md
# The script does not automate git commits or pushes, and it does not open PRs.
#
# Note: This script intentionally uses sed. yq (or even python libraries like
# "yaml" or "ruamel.yaml") do not preserve yaml formatting.
#




# Assumption: This script lives in operator's top-level hack/ folder
script_dir=$(dirname "$(readlink -f "$0")")
proj_root=$(readlink -f "${script_dir}"/..)




function usage_and_exit() {
    echo
    echo "Usage:"
    echo "  $0  <ACTION> [ARGS]"
    echo
    echo "  ACTION"
    echo "  One of { check, update }"
    echo "  check   Check that it's OK to proceed with the release. May modify"
    echo "          local files if needed but does not commit or push them."
    echo "          Note: Takes no args."
    echo "  update  Update the operator repo with new tags. Modifies local"
    echo "          files but does not commit or push them."
    echo "          Note: All args are required"
    echo
    echo "  ARGS"
    echo "  -o  The operator tag for your release"
    echo "      Example: v0.9.0"
    echo "  -e  The enclave-cc tag for your release"
    echo "      Example: v0.9.1"
    echo "  -k  The kata tag for your release (n.b. no 'v' prefix)"
    echo "      Example: 3.6.0"
    echo
    echo "Example usage:"
    echo "    $0 check"
    echo "    $0 update -o v0.9.0 -e v0.9.1 -k 3.6.0"
    echo
    exit 1
}


function bail_if_dirty_repo() {
    git diff --quiet ||
          (echo
           echo "Error: Must use a clean repo to use this script."
           echo "       Stash, commit, branch, or use a fresh clone, etc."
           echo
           exit 1)
}


function parse_args() {
    # pull the ACTION from the args list
    if [ $# = 0 ]; then
        usage_and_exit
    fi
    action=$1
    shift

    # parse the o, e, and k args
    while getopts ":o:e:k:" opt; do
        case "${opt}" in
            o)
                operator_tag=${OPTARG} ;;
            e)
                enclave_cc_tag=${OPTARG} ;;
            k)
                kata_tag=${OPTARG} ;;
            *)
                usage_and_exit ;;
        esac
    done

    # minimal checking: ensure all args are provided when using 'update'
    if [ "${action}" = "update" ]; then
        if [[ ! -v operator_tag ]] || [[ ! -v enclave_cc_tag ]] || [[ ! -v kata_tag ]]; then
            echo "Error: Missing ARGS"
            usage_and_exit
        fi
    fi
}


#
# Make sure that the pre-reqs container ships the same version of the nydus
# snapshotter that we use in the kata ci. Update the Makefile if needed.
#
function check_nydus() {
    echo "Checking that the operator's nydus version matches the one used in kata"

    kata_versions="https://raw.githubusercontent.com/kata-containers/kata-containers/refs/tags/${kata_tag}/versions.yaml"
    kata_versions_dest="versions.yaml.$$"

    # pull the versions.yaml that kata uses
    wget --quiet "${kata_versions}" -O "${kata_versions_dest}"

    # multi-line grep to find the nydus-snapshotter version in kata's version.yaml
    # then awk/tail/tr and string manipulation to sand it down.
    nydus_version_kata="$(grep -zoP "nydus-snapshotter.*\n.*description.*\n.*url.*\n.*version.*" ${kata_versions_dest} | awk '{print $2}' | tail -n1 | tr -d '\0')"
    nydus_version_kata=${nydus_version_kata//\"} # strip double quotes

    # delete temporary kata versions.yaml that we downloaded
    rm "${kata_versions_dest}"

    # grab the nydus version in the operator
    prereqs_makefile="install/pre-install-payload/Makefile"
    nydus_version_prereqs="$(grep '^NYDUS_SNAPSHOTTER_VERSION = ' $prereqs_makefile | awk '{print $3}')"

    # check for match and update if needed
    if [ "${nydus_version_kata}" == "${nydus_version_prereqs}" ]; then
        echo "-> OK, no changes made: nydus-snapshotter already up-to-date" \
             "(version: $nydus_version_prereqs)"
    else
        echo "-> Found mismatch: updating nydus-snapshotter from" \
             "$nydus_version_prereqs to $nydus_version_kata"
        sed -i "s/^NYDUS_SNAPSHOTTER_VERSION = .*/NYDUS_SNAPSHOTTER_VERSION = $nydus_version_kata/" "${prereqs_makefile}"
        echo "-> Changes made to ${prereqs_makefile}."
        echo "   Please verify and open PR as needed."
    fi
}


#
# Update the operator tag for the new release.
#
function update_operator() {
    operator_yaml="config/release/kustomization.yaml"
    echo "Updating operator version to ${operator_tag}"
    echo "-> Updating ${operator_yaml}"
    sed -i "s/newTag:.*/newTag: ${operator_tag}/" "${operator_yaml}"
}


#
# Update the CRDs to point to the most recent version of the pre-reqs payload.
#
# To determine the tag of the reqs-payload on quay that needs to be pinned, we
# have to dig deeper than simply looking at the latest commit associated with
# the install/pre-install-payload/ folder. At the time of this writing, the
# reqs-payload is updated automatically by the CI workflow in
# pre-install-image-publish-on-merge.yaml. It is triggered on merge, and it
# tags the reqs-payload in quay with the commit sha of that merge.
# Thus, we use `git rev-list` to find the commit sha of the merge that is
# associated with the most recent commit to install/pre-install-payload/.
#
function update_prereqs() {
    commit="$(git log -1 --pretty=format:"%H" install/pre-install-payload/)"
    prereqs_latest_hash=$( (git rev-list $commit..HEAD --ancestry-path | cat -n; git rev-list $commit..HEAD --first-parent | cat -n) | sort -k2 -s | uniq -f1 -d | sort -n | tail -1 | cut -f2)

    echo "Updating pre-reqs payload to $prereqs_latest_hash"

    for f in "${RHYL_PREREQS[@]}"; do
        echo "-> Updating $f"
        # match the reqs-payload line; then replace the newTag line after it
        sed -i '/quay.io\/confidential-containers\/reqs-payload/{n;s/.*/  newTag: '"${prereqs_latest_hash}"'/}' "$f"
    done

}


#
# Update enclave-cc and ccruntime bundles
#
function update_bundles() {

    echo "Updating enclave-cc yamls to $enclave_cc_tag"

    echo "-> Updating config/samples/enclave-cc/sim/kustomization.yaml"
    # this is a release; use runtime-payload instead of runtime-payload-ci
    sed -i 's/quay.io\/confidential-containers\/runtime-payload-ci/quay.io\/confidential-containers\/runtime-payload/' config/samples/enclave-cc/sim/kustomization.yaml
    # match the runtime-payload line; then replace the newTag line after it
    sed -i '/quay.io\/confidential-containers\/runtime-payload/{n;s/.*/  newTag: enclave-cc-SIM-sample-kbc-'"${enclave_cc_tag}"'/}' config/samples/enclave-cc/sim/kustomization.yaml

    echo "-> Updating config/samples/enclave-cc/base/ccruntime-enclave-cc.yaml"
    # this is a release; use runtime-payload instead of runtime-payload-ci
    # and change the tag from latest to the provided version tag
    sed -i "s/runtime-payload-ci:enclave-cc-HW-cc-kbc-latest/runtime-payload:enclave-cc-HW-cc-kbc-${enclave_cc_tag}/" config/samples/enclave-cc/base/ccruntime-enclave-cc.yaml

    echo "Updating ccruntime yamls to $kata_tag"

    for f in "${RHYL_KATA[@]}"; do
        echo "-> Updating $f"
        # this is a release; delete the newName kata-deploy-ci line
        sed -i '/newName: quay.io\/kata-containers\/kata-deploy-ci/d' "$f"
        # match the kata-deploy line; then replace the newTag line after it
        sed -i '/quay.io\/kata-containers\/kata-deploy/{n;s/.*/  newTag: '"${kata_tag}"'/}' "$f"
    done

}


function check_release() {
    check_nydus
}


function update_payloads() {
    update_operator
    update_prereqs
    update_bundles
}


function main() {
    bail_if_dirty_repo
    parse_args "$@"

    echo
    echo "** Running script for action \"${action}\" **"
    echo "**    Working directory: ${proj_root} **"
    echo
    pushd "${proj_root}" &> /dev/null

    case "${action}" in
    check) check_release ;;
    update) update_payloads ;;
    *) echo "Unexpected ACTION provided: ${action}"; usage_and_exit;; # unexpected
    esac

    echo
    echo "** Done **"
    echo
    popd &> /dev/null
}




main "$@"
