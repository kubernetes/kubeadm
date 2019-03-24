# Kubeadm

The purpose of this repo is to aggregate issues filed against the [kubeadm component](https://github.com/kubernetes/kubernetes/tree/master/cmd/kubeadm).

## What is Kubeadm ?
Kubeadm is a tool built to provide best-practice “fast paths” for creating Kubernetes clusters. It was designed with the purpose of simplifing creation, configuration, upgrade, downgrade, and teardown of Kubernetes clusters and their components. It performs the actions necessary to get a minimum viable cluster up and running with best-practice cluster for each minor version. Simplify the user experience and secure the cluster in fatest way. 


## Common Kubeadm cmdlets 
1. **kubeadm init** to bootstrap kubernetes control-plane nodes.
1. **kubeadm join** to bootstrap a Kubernetes worker node and join it to the cluster.
1. **kubeadm upgrade** to upgrade a Kubernetes cluster to a newer version.
1. **kubeadm reset** to revert any changes made to this host by kubeadm init or kubeadm join.

## Community, discussion, contribution, and support

Learn how to engage with the Kubernetes community on the [community page](https://kubernetes.io/community/).

You can reach the maintainers of this project at the [Cluster Lifecycle SIG](https://github.com/kubernetes/community/tree/master/sig-cluster-lifecycle#cluster-lifecycle-sig).

## Roadmap

The full picture of which direction we're taking is described in [this blog post](https://kubernetes.io/blog/2017/01/stronger-foundation-for-creating-and-managing-kubernetes-clusters/).

Please also refer to the latest [milestones in this repo](https://github.com/kubernetes/kubeadm/milestones).

### Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct](code-of-conduct.md).
