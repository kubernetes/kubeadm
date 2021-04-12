# kind & kinder

[kind](https://github.com/kubernetes-sigs/kind) is awesome!

kinder shares the same core ideas of kind, but in agreement with the kind team, was developed
as a separated tool with the goal to explore new kubeadm-related use cases, share issues and solutions, and
ultimately contribute back new features.

## Target use cases

At high level you should use [kind](https://github.com/kubernetes-sigs/kind) whenever you want
a working Kubernetes cluster and _just_ work on it (for DEV, test, CI/CD).

As for kinder, you should use it when you want to have more control over the process of creating a
new Kubernetes cluster from "bare machines" running as containers.

## Differences

Kinder provides a UX designed for helping kubeadm contributors and for kubeadm E2E tests.
Only a subset of the [kind](https://github.com/kubernetes-sigs/kind) commands which are useful for the targeted use
cases is available in kinder.

_Building images:_
- kinder supports both `containerd` and `docker` as container runtime inside the images
- kinder provides support for altering base/node images by:
     - Adding a Kubernetes version to be used for `kubeadm init` or `kubeadm upgrade` (from release, CI/CD or locally build artifacts)
     - Pre-loading tar image files into the base/node image
     - Replacing the kubectl, kubelet or kubeadm binary to be used for `kubeadm init` (from release, CI/CD or locally
       build artifacts)
- kinder can build images only on top of linux/amd64 base images (currently ubuntu:18.04)

_Creating the cluster:_
- kinder support both `containerd` and `docker` as container runtime inside the images
- kinder allows to break down the `create` operation into several atomic actions:
    - Creating machines running as containers
    - Generating kubeadm config
    - Generating load balancer config

  NB. kind itself also has a `kind create` operation that includes creating nodes, generating kubeadm/load balancer
  config init, join (eventually, only the last two operations can be skipped)
- kinder allows adding an external etcd to the cluster
- kinder allows adding an external load balancer to the cluster even if there are less than two control-plane nodes
- kinder still uses kustomize for bulding the kubeadm config file, while kind dropped this dependency for a lighther solution

_Actions on a running cluster:_
- kinder supports running actions on a running cluster (after `kinder create`)
- kinder supports running actions selectively on nodes
- kinder supports running actions in a dry-run mode
- kinder actions provide output for everything happening on the nodes (for debugging purposes)
- kinder actions supports the following variations of corresponding kind actions
    - generate kubeadm config and generate load balancer config can be invoked also after `kinder create`
    - generate kubeadm config supports:
        - automatic configuration for external etcd/external lb, if present
        - shortcut for using kube-dns instead of CoreDNS
        - shortcut for setting certificateKey field (for testing the automatic copy certs feature of kubeadm)
        - shortcut for testing different kubeadm join discovery mechanics
    - `kubeadm init` can be executed as a unique workflow or using phases
    - `kubeadm join` can be executed as a unique workflow or using phases
    - the init action installs kindnet CNI plugin in a custom workflow defined by kinder. kubeadm should be CNI agnostic and that validation of other CNI providers is out of the scope of our current E2E testing.
    - the init/join actions can use the automatic copy certs feature of kubeadm (or mimic the manual copy process)
- kinder support additional actions
    - upgrade
    - reset
    - cluster-info
    - smoke test

_Utils:_
- kinder provides a utility for downloading release or CI/CD artifacts (kinder get)
- kinder provides a utility for running E2E tests or E2E-kubeadm tests on a running cluster (kinder test)
- kinder provides a utility for automating complex workflows (used for implementing E2E test workflows)
- kinder supports topology aware wrapper on cp / exec

## Credits

kinder is an example of [kind](https://github.com/kubernetes-sigs/kind) used as a library.

This is a curated list of what kinder is using from kind; please note that kinder is using
kind packages that are not intended for public usage, but this was agreed with the
[kind](https://github.com/kubernetes-sigs/kind) team as part of the process of exploring
new use cases, share lessons learned, issues and solutions, and ultimately contribute
back new features.

- "sigs.k8s.io/kind/pkg/cmd"
- "sigs.k8s.io/kind/pkg/cmd/kind/delete"
- "sigs.k8s.io/kind/pkg/cmd/kind/export"
    - providing access to few kind commands useful for the use cases targeted by kinder
- "sigs.k8s.io/kind/pkg/fs" (*) for
    - `TempDir`
    - `Copy`

(*) to be evaluated if removing dependency in future kinder versions
