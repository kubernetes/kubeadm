# Kubeadm

The purpose of this repo is to aggregate issues filed against the [kubeadm component](https://git.k8s.io/kubernetes/cmd/kubeadm).

## What is Kubeadm ?
Kubeadm is a tool built to provide best-practice "fast paths" for creating Kubernetes clusters.
It performs the actions necessary to get a minimum viable, secure cluster up and running in a user friendly way.
Kubeadm's scope is limited to the local node filesystem and the Kubernetes API, and it is intended to be a composable building block of higher level tools.


## Common Kubeadm cmdlets
1. **kubeadm init** to bootstrap the initial Kubernetes control-plane node.
1. **kubeadm join** to bootstrap a Kubernetes worker node or an additional control plane node, and join it to the cluster.
1. **kubeadm upgrade** to upgrade a Kubernetes cluster to a newer version.
1. **kubeadm reset** to revert any changes made to this host by kubeadm init or kubeadm join.

## Documentation

- [Installing kubeadm](https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/install-kubeadm/)
- [Creating a cluster](https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/create-cluster-kubeadm/)
- [Command-line reference](https://kubernetes.io/docs/reference/setup-tools/kubeadm/kubeadm/)
- [Configuration API reference](https://godoc.org/k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm) (pick an API version from the [list of packages](https://godoc.org/k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm#section-directories))

## Community, discussion, contribution, and support

Learn how to engage with the Kubernetes community on the [community page](https://kubernetes.io/community/).

You can reach the maintainers of this project at the [Cluster Lifecycle SIG](https://git.k8s.io/community/sig-cluster-lifecycle#cluster-lifecycle-sig).

## Roadmap

The full picture of which direction we're taking is described in [this blog post](https://kubernetes.io/blog/2017/01/stronger-foundation-for-creating-and-managing-kubernetes-clusters/).

Please also refer to the latest [milestones in this repo](https://github.com/kubernetes/kubeadm/milestones).

### Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct](code-of-conduct.md).
