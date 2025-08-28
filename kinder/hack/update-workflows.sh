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

# script to update kinder workflows and test-infra Prow jobs
# shellcheck source=/dev/null
source "$(dirname "$0")/utils.sh"
# cd to the root path
cd_root_path

# set kubernetes version
KUBERNETES_VERSION="${KUBERNETES_VERSION:-v1.34.0}"

# set skew
KUBERNETES_SKEW="${KUBERNETES_SKEW:-3}"

# path to the config
PATH_CONFIG="${PATH_CONFIG:-./ci/tools/update-workflows/config.yaml}"

# path to the kinder workflows
PATH_WORKFLOWS="${PATH_WORKFLOWS:-./ci/workflows/}"

# set test-infra path
TEST_INFRA_SIG_DIR="config/jobs/kubernetes/sig-cluster-lifecycle"
PATH_TEST_INFRA="${PATH_TEST_INFRA:-"${GOPATH}/src/k8s.io/test-infra/${TEST_INFRA_SIG_DIR}"}"

# try to get the image from the provided test-infra path
if [[ -z "${TEST_INFRA_IMAGE}" ]]; then
  TEST_INFRA_IMAGE=$(grep image "${PATH_TEST_INFRA}" -r | head -1 | cut -d ':' -f 4 | cut -d '-' -f-2)
  echo "${TEST_INFRA_IMAGE}"
  if [[ -z "${TEST_INFRA_IMAGE}" ]]; then
    echo "Error: Could not detect TEST_INFRA_IMAGE"
    exit 1
  fi
fi

# cleanup tracked files in workflows and test-infra
echo "Cleaning up workflows..."
rm "${PATH_WORKFLOWS}"/*.yaml

echo "Cleaning up test-infra jobs..."
rm "${PATH_TEST_INFRA}"/kubeadm-*.yaml

set -o xtrace

# run the tool
go run ./ci/tools/update-workflows/cmd/main.go \
  --config "${PATH_CONFIG}" \
  --kubernetes-version="${KUBERNETES_VERSION}" \
  --path-test-infra="${PATH_TEST_INFRA}" \
  --path-workflows="${PATH_WORKFLOWS}" \
  --skew-size="${KUBERNETES_SKEW}" \
  --image-test-infra="${TEST_INFRA_IMAGE}"
