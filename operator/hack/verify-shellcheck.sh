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

# set -o errexit
set -o nounset
set -o pipefail

# shellcheck source=/dev/null
source "$(dirname "$0")/utils.sh"
ROOT_PATH=$(get_root_path)
os=$(go env GOOS)

# create a temporary directory
TMP_DIR=$(mktemp -d)

# cleanup on exit
cleanup() {
  echo "Cleaning up..."
  rm -rf "${TMP_DIR}"
}
trap cleanup EXIT

# install shellcheck
cd "${TMP_DIR}" || exit
VERSION="shellcheck-stable"
DOWNLOAD_FILE="${VERSION}.${os}.x86_64.tar.xz"
wget https://github.com/koalaman/shellcheck/releases/download/stable/"${DOWNLOAD_FILE}"
tar xf "${DOWNLOAD_FILE}"
cd "${VERSION}" || exit

echo "Running shellcheck..."
cd "${ROOT_PATH}" || exit
OUT="${TMP_DIR}/out.log"
FILES=$(find . -name "*.sh")
while read -r file; do
    "${TMP_DIR}/${VERSION}/shellcheck" "$file" >> "${OUT}" 2>&1
done <<< "$FILES"

if [[ -s "${OUT}" ]]; then
  echo "Found errors:"
  cat "${OUT}"
  exit 1
fi
