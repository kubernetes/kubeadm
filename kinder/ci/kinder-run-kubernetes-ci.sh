#!/usr/bin/env bash
# Copyright 2026 The Kubernetes Authors.
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

echo "Starting the kinder workflow runner for Kubernetes CI..."

if [ "$#" -ne 1 ]; then
    echo "error: expected exactly one argument for the workflow name"
    echo "usage:"
    echo "# build kubeadm from KUBE_ROOT and inject it in the node image of a workflow from /ci/workflows/"
    echo "KUBE_ROOT=/some-path/kubernetes kinder-run-kubernetes-ci.sh some-workflow"
    echo "# KUBE_ROOT defaults to \$GOPATH/src/k8s.io/kubernetes"
    exit 1
fi

WORKFLOW_NAME="${1%.yaml}"
ANCHOR_TASK="add-kubernetes-versions"
INJECT_TASK="inject-kubeadm"

CURRENT_PATH="$(pwd)"
SCRIPT_PATH="$(dirname "$(readlink -f "$0")")"
WORKFLOW_BUCKET_PATH="${SCRIPT_PATH}/workflows"
KUBE_ROOT="$(readlink -f "${KUBE_ROOT:-$(go env GOPATH)/src/k8s.io/kubernetes}")"

if [ ! -f "${WORKFLOW_BUCKET_PATH}/${WORKFLOW_NAME}.yaml" ]; then
    echo "error: workflow ${WORKFLOW_NAME}.yaml not found in ${WORKFLOW_BUCKET_PATH}"
    exit 1
fi

if [ ! -f "${KUBE_ROOT}/cmd/kubeadm/kubeadm.go" ]; then
    echo "error: ${KUBE_ROOT} is not a valid Kubernetes source directory"
    exit 1
fi

if [ -z "${ARTIFACTS:-}" ]; then
    ARTIFACTS="$(mktemp -d "${CURRENT_PATH}/kinder-test-workflow.XXXXXX")"
    export ARTIFACTS
fi

WORK_DIR="$(mktemp -d)"
trap 'rm -rf "${WORK_DIR}"' EXIT

# Build kubeadm from the k/k tree.
echo "Building kubeadm from ${KUBE_ROOT}..."
make -C "${KUBE_ROOT}" WHAT=cmd/kubeadm
KUBEADM_BINARY="${KUBE_ROOT}/_output/bin/kubeadm"
"${KUBEADM_BINARY}" version -o short

# Prepare a Docker image that can replace the kubeadm binary in a node image.
BUILD_CONTEXT="${WORK_DIR}/build"
mkdir -p "${BUILD_CONTEXT}"
cp "${KUBEADM_BINARY}" "${BUILD_CONTEXT}/kubeadm"
cat > "${BUILD_CONTEXT}/Dockerfile" <<EOF
ARG IMAGE
FROM \${IMAGE}
COPY kubeadm /usr/bin/kubeadm
RUN set -e; for d in /kinder/upgrade/*/; do if [ -d "\${d}" ]; then cp /usr/bin/kubeadm "\${d}kubeadm"; fi; done
EOF

# Create a kinder workflow task that can be injected in the workflow.
cat > "${WORK_DIR}/task.yaml" <<EOF
name: ${INJECT_TASK}
description: |
  injects the kubeadm binary built from ${KUBE_ROOT} in the node image
cmd: docker
args:
  - build
  - --build-arg=IMAGE={{ .vars.image }}
  - --tag={{ .vars.image }}
  - ${BUILD_CONTEXT}
timeout: 5m
EOF

# Copy the workflow and imported tasks file.
mkdir -p "${WORK_DIR}/workflows"
WORKFLOW_PATH="${WORK_DIR}/workflows/${WORKFLOW_NAME}.yaml"
cp "${WORKFLOW_BUCKET_PATH}/${WORKFLOW_NAME}.yaml" "${WORKFLOW_PATH}"

TASKS_FILE="$(yq '.tasks[0].import // ""' "${WORKFLOW_PATH}")"
if [ -z "${TASKS_FILE}" ]; then
    echo "error: ${WORKFLOW_NAME}.yaml does not import a tasks file"
    exit 1
fi
TASKS_PATH="${WORK_DIR}/workflows/${TASKS_FILE}"
cp "${WORKFLOW_BUCKET_PATH}/${TASKS_FILE}" "${TASKS_PATH}"

# Inject the custom task after the anchor task.
INDEX="$(yq ".tasks | to_entries | .[] | select(.value.name == \"${ANCHOR_TASK}\") | .key" "${TASKS_PATH}")"
if [ -z "${INDEX}" ]; then
    echo "error: task ${ANCHOR_TASK} not found in ${TASKS_FILE}"
    exit 1
fi
NEXT=$((INDEX + 1))
TASK_PATH="${WORK_DIR}/task.yaml" yq -i ".tasks |= .[0:${NEXT}] + [load(env(TASK_PATH))] + .[${NEXT}:]" "${TASKS_PATH}"

# Run the new workflow with the injected task.
pushd "${WORK_DIR}/workflows"
"${SCRIPT_PATH}/kinder-run.sh" "${WORKFLOW_NAME}.yaml"
popd
