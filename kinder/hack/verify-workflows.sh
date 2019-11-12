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

set -o nounset
set -o pipefail

# shellcheck source=/dev/null
source "$(dirname "$0")/utils.sh"
# cd to the root path
cd_root_path

# create a temporary directory
TMP_DIR=$(mktemp -d)

# cleanup
exitHandler() (
  echo "Cleaning up..."
  rm -rf "${TMP_DIR}"
)
trap exitHandler EXIT

# build the verify-workflow binary
echo "Building verify-workflow..."
export GO111MODULE=on
BIN="${TMP_DIR}/verify-workflow"
go build -o "${BIN}" ./ci/tools/verify-workflow.go

# verify files
echo "Verifying workflow files..."
ERR="0"
FILES="$(git ls-files | grep ci/workflows)"
while read -r file; do
    "${BIN}" "${file}" || ERR="1"
done <<< "$FILES"

if [[ "${ERR}" == "1" ]]; then
    echo ""
    echo "Found errors in workflow files! See output above..."
    exit 1
fi
