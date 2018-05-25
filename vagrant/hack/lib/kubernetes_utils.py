
# -*- coding: utf-8 -*-

# Copyright 2018 The Kubernetes Authors.
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

import os
import re

RFC1123_LABEL_PATTERN = "^[a-z0-9]([-a-z0-9]*[a-z0-9])?$"

def validate_rfc123_label(arg):
    if re.match(RFC1123_LABEL_PATTERN, arg) == None:
        raise ValueError("invalid RFC1123 label '%s'" % (arg)) 

RFC1123_SUBDOMAIN_PATTERN = "^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$"

def validate_rfc123_subdomain(arg):
    if re.match(RFC1123_SUBDOMAIN_PATTERN, arg) == None:
        raise ValueError("invalid RFC1123 subdomain '%s'" % (arg)) 

KUBERNETES_BAZEL_BUILD  = 'bazel'     
KUBERNETES_DOCKER_BUILD = 'docker'
KUBERNETES_LOCAL_BUILD = 'local'

KUBERNETES_BUILDERS = [KUBERNETES_BAZEL_BUILD, KUBERNETES_DOCKER_BUILD, KUBERNETES_LOCAL_BUILD]

def build_output_path(builder):
    """ returns the Kubernetes build output path for
        specified in ENV.builder of for bazel the env var is missing. 
        the path of the kubernetes project is computed according go standard conventions, 
        but it can be eventually overwritten with ENV.kube_root """

    goPath = os.path.join(os.environ['HOME'], 'go')
    if 'KUBEADM_BUILD_ROOT' in os.environ: 
        goPath = os.environ['KUBEADM_BUILD_ROOT']
    elif 'GOPATH' in os.environ:
        goPath = os.environ['GOPATH']

    kubernetes_project_path = os.path.join(goPath, 'src/k8s.io/kubernetes')

    if builder == KUBERNETES_BAZEL_BUILD:
        return os.path.join(kubernetes_project_path, 'bazel-bin/cmd/kubeadm/linux_amd64_pure_stripped')
    elif builder == KUBERNETES_DOCKER_BUILD:
        return os.path.join(kubernetes_project_path, '_output/dockerized/bin/linux/amd64/')
    elif builder == KUBERNETES_LOCAL_BUILD:
        return os.path.join(kubernetes_project_path, '_output/local/bin/linux/amd64/')
    else:
        raise ValueError("invalid builder type '%s' value. Valid types are %s" % (builder, ', '.join(KUBERNETES_BUILDERS))) 
