#!/usr/bin/env bash
# Copyright 2021 The Kubernetes Authors.
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
TMP_DIR_TEST_INFRA=$(mktemp -d)

# cleanup
exitHandler() (
  echo "Cleaning up..."
  rm -rf "${TMP_DIR}"
  rm -rf "${TMP_DIR_TEST_INFRA}"
)
trap exitHandler EXIT

# copy all workflows trakced in git in a temp location
# add them to git tracking there
echo "Coping workflows to $TMP_DIR"
FILES=$(git ls-files | grep ci/workflows | tr "\n" " ")
 # shellcheck disable=SC2086
cp ${FILES} "${TMP_DIR}"
pushd "${TMP_DIR}" || exit 1
git init > /dev/null
git config user.email "test@test-email.com"
git config user.name "Test Name"
git add . > /dev/null
git commit -m "..." > /dev/null
popd || exit 1

# call the update-workflows.sh script to update the workflwos in the temp location
echo "Running ./hack/update-workflows.sh"
PATH_WORKFLOWS="${TMP_DIR}" PATH_TEST_INFRA="${TMP_DIR}" ./hack/update-workflows.sh > "${TMP_DIR}/output.log" 2>&1
RES=$?
if [[ $RES -ne 0 ]]; then
    echo "error: Running ./hack/update-workflows.sh failed"
    cat "${TMP_DIR}/output.log"
    exit 1
fi

# use git status to ensure there is no diff between the copied workflows
# and the output of update-workflows.sh
pushd "${TMP_DIR}" || exit 1
OUT=$(git status --porcelain --untracked-files=no)
if [[ -n "${OUT}" ]]; then
  echo "${OUT}"
  echo "error: Found difference in the output; please run ./hack/update-workflows.sh"
  exit 1
fi

echo "Done!"
