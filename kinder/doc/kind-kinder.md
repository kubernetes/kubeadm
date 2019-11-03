# kind & kinder

[kind](https://github.com/kubernetes-sigs/kind) is awesome!

kinder is an example of [kind](https://github.com/kubernetes-sigs/kind) used as a library.

## Target use cases

At high level you should use [kind](https://github.com/kubernetes-sigs/kind) whenever you want
a working Kubernetes cluster and _just_ work on it (for DEV, test, CI/CD).

Instead, you should use kinder when you want to get more control on the process of creating a
new Kubernetes cluster from "bare" machines running into a container to a working Kubernetes.

## Differences

All the [kind](https://github.com/kubernetes-sigs/kind) commands will be available in kinder,
side by side with additional commands designed for helping kubeadm contributors.

_Building images:_
- kinder support both containerd and docker as container runtime inside the images
- kinder provides support for altering a base/node images and:
     - Add a Kubernetes version to be used for `kubeadm init` (from release, CI/CD or locally build artifacts)
     - Pre-load  TAR image files into the base/node image
     - Replace the kubectl, kubelet, kubeadm binary to be used for `kubeadm init` (from release, CI/CD or locally
       build artifacts)
     - Add a Kubernetes version to be used for `kubeadm upgrade` (from release, CI/CD or locally build artifacts)

_Creating the cluster:_
- kinder support both containerd and as container runtime inside the images
- kinder allows to break down the `create` operation into several atomic actions:
    - Create machines running into a container
    - Generate kubeadm config
    - Generate load balancer config

  NB. kind manages a unique `kind create` operation, that includes create nodes, generate kubeadm/load balancer
  config init, join (eventually, only the last two operations can be skipped)
- kinder allows adding an external etcd to the cluster
- kinder allows adding an external loadbalancer to the cluster even if less than two control-plane nodes

_Actions on a running cluster:_
- kinder support running actions on a running cluster (after `kinder create`)
- kinder support running actions selectively on nodes
- kinder support dry running actions
- kinder actions provide output for everything happening on the nodes (for debugging purposes)
- kinder actions supports the following variations of corresponding kind actions
    - generate kubeadm config and generate load balancer config can be invoked also after `kinder create`
    - generate kubeadm config supports:
        - automatic configuration for external etcd/external lb, if present
        - shortcut for using kube-dns instead of CoreDNS
        - shortcut for setting certificateKey field (for testing the automatic copy certs feature of kubeadm)
        - shortcut for testing different kubeadm join discovery mechanins
    - `kubeadm init` can be executed as a unique workflow or using phases
    - `kubeadm join` can be executed as a unique workflow or using phases
    - the init action installs Calico as a CNI plugin instead of kindnet
    - the init/join actions can use  theautomatic copy certs feature of kubeadm (or mimic the manual copy process)
- kinder support additional actions
    - upgrade
    - reset
    - cluster-info
    - smoke test

_Utils:_
- kinder provides a utility for downloading release or CI/CD artifacts
- kinder provides a utility for running E2E tests or E2E-kubeadm tests on a running cluster
- kinder provides a utility for automating complex workflows (used for implementing E2E test workflows)
- kinder support topology aware wrapper on cp / exec

## Credits

kinder is an example of [kind](https://github.com/kubernetes-sigs/kind) used as a library.

This is a curated list of what kinder is using from kind; please note that kinder is using
kind packages that are not intended for public usage, but this was agreed with the
[kind](https://github.com/kubernetes-sigs/kind) team as part of the process of exploring
new use cases, share lesson learned, issues and solutions, and ultimately contribute
back new features.

- "sigs.k8s.io/kind/cmd/*" for
    - providing access to kind commands from a single UX
- "sigs.k8s.io/kind/pkg/build/base" for
    - building a containerd base image
- "sigs.k8s.io/kind/pkg/cluster/constants" for
    - constants
- "sigs.k8s.io/kind/pkg/cluster/nodes" for
    - `Node` struct
    - `CreateControlPlaneNode`
    - `CreateWorkerNode`
    - `CreateExternalLoadBalancerNode`
- "sigs.k8s.io/kind/pkg/container/docker" for
    - `EditArchiveRepositories`
    - `PullIfNotPresent`
    - `Run`
    - `UsernsRemap`
    - `Kill`
- "sigs.k8s.io/kind/pkg/container/cri" for
    - `Mount` struct
    - `PortMapping` struct
- "sigs.k8s.io/kind/pkg/concurrent" for
    - `UntilError`
- "sigs.k8s.io/kind/pkg/exec" (*) for
    - `Command`
    - `InheritOutput`
    - `CombinedOutputLines`
- "sigs.k8s.io/kind/pkg/fs" (*) for
    - `TempDir`
    - `Copy`
- "sigs.k8s.io/kind/pkg/log" for the spinner (*)
- "sigs.k8s.io/kind/pkg/kustomize" for
    - `PatchJSON6902` struct
    - `Build`

(*) to be evaluated if removing dependency in future kinder versions
