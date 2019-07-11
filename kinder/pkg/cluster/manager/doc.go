/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

/*
Package manager implements the highest level abstraction implemented of kinder, that is the ClusterManager.

ClusterManager allows to operate on an existing K8s cluster created with kind(er) by exposing
high level actions like cp, exec or do actions.

This package implement also support for creating a new Kinder cluster, with the support for both
containerd and docker as a container runtime running inside kind(er).
*/
package manager
