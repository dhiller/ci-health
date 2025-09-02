#!/usr/bin/env bash
#
# This file is part of the KubeVirt project
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# Copyright the KubeVirt Authors.
#
#

set -euo pipefail

function usage() {
    cat <<EOF
usage: $0 {file_with_job_urls}

    fetches failures from the build logs of build urls given through the file - each line correlates to a build url.
EOF
}

if [ "$#" -gt 0 ]; then
    if [[ "$1" == -h ]] || [[ "$1" == --help ]]; then
        usage
        exit 0
    fi

    if [ ! -f "$1" ]; then
        usage
        exit 1
    fi
else
    usage
    exit 1
fi

urls_file="$1"

work_dir=$(mktemp -d)

function cleanup() {
    rm -rf "${work_dir}"
}

current_commit="$(git rev-parse HEAD)"

#trap 'cleanup' SIGINT SIGTERM EXIT

declare -a groups=( 'sig-compute|sig-operator|sev|vgpu|windows' 'sig-network|sriov' 'sig-storage' 'sig-monitoring')
for group in "${groups[@]}"; do
    output_file="/tmp/errors-${group//|/-}-${current_commit:0:7}.txt"
    echo '' > "${output_file}"

    while IFS= read -r job_url; do
        build_log_file="${work_dir}/build-log.txt"
        job_name=$(echo "${job_url}" | sed -re 's#.*(pull-kubevirt-[^/]+).*#\1#g')
        curl --silent --fail \
            "$(echo "${job_url}" | sed -re 's#^https://prow.ci.kubevirt.io//view/gs/(.*)#https://storage.googleapis.com/\1#g')/build-log.txt" \
            -o "${build_log_file}"
        echo "job ${job_name} has the following errors:" >>"${output_file}"
        echo "-----------------------------------------" >>"${output_file}"
        if ! rg -i -B 3 -A 1 'ERROR|Could not resolve host|panic|failure' "${build_log_file}" >>"${output_file}"; then
            echo "no matches found for ERROR in ${job_url}"
        fi
        echo "-----------------------------------------" >>"${output_file}"
        echo "" >>"${output_file}"
    done< <(grep -E "$group" "${urls_file}")

    echo "${output_file}"
done
