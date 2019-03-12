# Roadmap 🗺️

This document outlines some goals, non-goals, and future aspirations for kinder.

kinder is an example of [kind](https://github.com/kubernetes-sigs/kind) used as a library.

All the kind commands will be available in kinder, side by side with additional commands designed
for helping kubeadm contributors.

High level goals for kinder v0.1 include:

- [ ] Provide a local test environment for kubeadm development
   - [x] Allow creation of nodes "ready for installing Kubernetes"
   - [x] Provide pre built “developer” workflows for kubedam init, join, reset
      - [x] init and init with phases
      - [x] join and join with phases
      - [x] init and join with automatic copy certs
      - [x] Provide pre built “developer” workflow for kubeadm upgrades
      - [x] reset
   - [x] Allow build of node-image variants
      - [x] add pre-loaded images to a node-image
      - [x] replace the kubeadm binary into a node-image
      - [x] add kubernetes binaries for a second kubernetes versions (target for upgrades)
   - [x] Allow test of kubeadm cluster variations
      - [x] external etcd
      - [x] kube-dns
   - [x] Provide "topology aware" wrappers for `docker exec` and `docker cp`
   - [x] Provide a way to add nodes to an existing cluster
      - [x] Add worker node
      - [x] Add control plane node (and reconfigure load balancer)
   - [x] Provide smoke test action
   - [ ] Provide E2E run action(s)

**Non**-Goals include:

- Replace or fork kind. kind is awesome and we are committed to help to improve it (see long term goals)
- Supporting every possible use cases that can be build on top of kind as a library

Longer Term goals include:

- Simplify kubeadm development/local testing
- Help new contributors on kubeadm development
- Contribute to improving and testing "kind as a library"
- Contribute back idea/code for new features in kind
- Provide a home for use cases that are difficult to support in the main kind CLI
