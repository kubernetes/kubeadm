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

# script for running conformance tests on a kind cluster
# based on code in sigs.k8s.io/kind by @bentheelder

set -o errexit
set -o nounset
set -o pipefail

# export the KUBECONFIG
KUBECONFIG="$(kind get kubeconfig-path)"
export KUBECONFIG

# base kubetest args
KUBETEST_ARGS="--provider=skeleton --test --check-version-skew=false"

# get the number of worker nodes
NUM_NODES="$(kind get nodes | grep worker | wc -l)"

# ginkgo regexes
SKIP="${SKIP:-"Alpha|Kubectl|\\[(Disruptive|Feature:[^\\]]+|Flaky)\\]"}"
FOCUS="${FOCUS:-"\\[Conformance\\]"}"
# if we set PARALLEL=true, skip serial tests set --ginkgo-parallel
PARALLEL="${PARALLEL:-false}"
if [[ "${PARALLEL}" == "true" ]]; then
    SKIP="\\[Serial\\]|${SKIP}"
    KUBETEST_ARGS="${KUBETEST_ARGS} --ginkgo-parallel"
fi

# add ginkgo args
KUBETEST_ARGS="${KUBETEST_ARGS} --test_args=\"--ginkgo.focus=${FOCUS} --ginkgo.skip=${SKIP} --report-dir=${ARTIFACTS} --disable-log-dump=true --num-nodes=${NUM_NODES}\""

# setting this env prevents ginkg e2e from trying to run provider setup
export KUBERNETES_CONFORMANCE_TEST="y"

# run kubetest, if it fails clean up and exit failure
eval "kubetest ${KUBETEST_ARGS}"
