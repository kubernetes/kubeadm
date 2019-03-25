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

package actions

// APIServerPort is the expected default APIServerPort on the control plane node(s)
// https://kubernetes.io/docs/reference/access-authn-authz/controlling-access/#api-server-ports-and-ips
const APIServerPort = 6443

// Token defines a dummy, well known token for automating TLS bootstrap process
const Token = "abcdef.0123456789abcdef"

// CertificateKey defines a dummy, well known CertificateKey for automating automatic copy certs process
// const CertificateKey = "d02db674b27811f4508bf8a5fa19fbe060921340552f13c15c9feb05aaa96824"
const CertificateKey = "0123456789012345678901234567890123456789012345678901234567890123"

// ControlPlanePort defines the port where the control plane is listening on the load balancer node
const ControlPlanePort = 6443
