#!/usr/bin/env bash
# Copyright 2019 The Kubernetes Authors.
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

set -o errexit
set -o nounset
set -o pipefail

echo "Starting the kinder workflow runner..."

# this script should be used as a bridge between CI jobs and the kinder binary
if [ "$#" -eq 0 ]; then
    echo "error: missing argument for workflow config"
    echo "usage:"
    echo "# to pick a workflow .yaml file by name from /ci/workflows/"
    echo "kinder-run.sh some-workflow"
    echo "# to use a file from an absolute path"
    echo "kinder-run.sh /some-path/some-workflow.yaml"
    exit 1
fi

# set paths
CURRENT_PATH="$(pwd)"
SCRIPT_PATH="$(dirname "$(readlink -f "$0")")"
ROOT_PATH="$(readlink -f "${SCRIPT_PATH}/..")"
WORKFLOW_BUCKET_PATH="${SCRIPT_PATH}/workflows"

# build kinder
pushd "${ROOT_PATH}"
echo "Building kinder..."
GO111MODULE=on go build
popd

# add the kinder ROOT directory to PATH
# this is needed so that the workflow runner can easy call "kinder ..." commands.
export PATH="${PATH}:${ROOT_PATH}"

# determine if the passed config is by name or file path
if [[ "$1" != *".yaml" ]] && [[ "$1" != *"/"* ]]; then
  echo "Using $1.yaml from the default workflow path..."
  WORKFLOW_PATH="${WORKFLOW_BUCKET_PATH}/$1.yaml"
else
  WORKFLOW_PATH="$(readlink -f "${CURRENT_PATH}/$1")"
fi

# run the workflow
KINDER_FLAGS=""
kinder test workflow "${WORKFLOW_PATH}" "${KINDER_FLAGS}"
