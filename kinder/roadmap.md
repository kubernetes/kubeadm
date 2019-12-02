# Roadmap 🗺️

This document outlines some goals, non-goals, and future aspirations for kinder.

kinder is heavily inspired by [kind](https://github.com/kubernetes-sigs/kind), kinder shares the same core ideas of kind, but in agreement with the kind team,
was developed as a separated tool with the goal to explore new kubeadm-related use cases, share issues and solutions, and
ultimately contribute back new features.

High level goals for kinder include:

- [x] Support docker and containerd as a container runtime inside nodes

- [ ] Provide a local test environment for kubeadm development
   - [x] Allow creation of nodes "ready for installing Kubernetes"
      - [ ] Support mount volumes
   - [x] Provide pre built developer-workflows for kubeadm init, join, reset
      - [x] init and init with phases
      - [x] join and join with phases
      - [x] init and join with automatic copy certs
      - [x] Provide pre built developer-workflow for kubeadm upgrades
      - [x] reset
   - [x] Allow build of node-image variants
      - [x] add pre-loaded images to a node-image
      - [x] replace the kubeadm binary into a node-image
      - [x] add Kubernetes binaries for a second Kubernetes versions (target for upgrades)
   - [x] Allow test of kubeadm cluster variations
      - [x] external etcd
      - [x] kube-dns
   - [x] Allow test of kubeadm features
      - [x] discovery types
      - [x] kustomize
      - [ ] certificate renewal
      - [ ] machine readable output
   - [x] Provide "topology aware" wrappers for `docker exec` and `docker cp`
   - [ ] Provide a way to add nodes to an existing cluster
      - [ ] Add worker node
      - [ ] Add control plane node (and reconfigure load balancer)
   - [x] Provide smoke test action
   - [ ] Support for testing concurrency on joining nodes
   - [ ] Support testing the kubeadm-operator
   - [ ] Explore synergies with CAPD

- [x] Be the kubeadm CI glue
   - [x] Provide get Kubernetes artifacts command(s)
   - [x] Allow build of node-image variants using Kubernetes artifacts from different sources
   - [x] Provide E2E run command(s)
   - [x] Provide test command that automates complex test scenarios composed by many steps/stages (Workflows)

**Non**-Goals include:

- Replace or fork kind. kind is awesome and we are committed to help to improve it (see long term goals)
- Supporting every possible use cases that can be build on top of kind as a library

Longer Term goals include:

- Improve the kubeadm CI signal
- Simplify kubeadm development/local testing
- Help new contributors on kubeadm development
- Contribute to improving and testing "kind as a library"
- Contribute back idea/code for new features in kind
