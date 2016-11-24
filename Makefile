# Copyright 2016 The Kubernetes Authors.
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

OS?=linux
ARCH?=amd64

build:
	docker run -it -v $(shell pwd):/go/src/k8s.io/kubeadm -w /go/src/k8s.io/kubeadm -u $(shell id -u):$(shell id -g) \
		gcr.io/google_containers/kube-cross:v1.7.1-3 /bin/bash -c "go build -o bin/$(OS)/$(ARCH)/kubeadm k8s.io/kubeadm/cmd/kubeadm"

clean:
	rm -rf bin
