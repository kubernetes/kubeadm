# Testing upgrades

## Preparation

This document assumes vX the Kubernetes release to be used when creating the cluster, and vY
the target Kubernetes version for the upgrades.

In order to test upgrades, the recommended approach in kinder is to bundle in one
image both vX and vY:

1. take a node-image for Kubernetes vX, e.g. `kindest/node:vX`
2. prepare locally the Kubernetes binaries and the docker images for Kubernetes vY
3. create an variant of `kindest/node:vX`, e.g. `kindest/node:vX.to.Y`, that contains vY artifacts as well

e.g. assuming vY artifacts stored in $artifacts

```bash
kinder build node-image-variant --base-image kindest/node:vX --image kindest/node:vX.to.Y \
    --with-upgrade-artifacts $artifacts/binaries
```

> `kinder build node-image-variant` accepts in input a version, a release or ci build label,
> a remote repository or a local folder. see [Kinder reference](reference.md) for more info.

> vY artifacts will be saved in the `/kinder/upgrades/vy` folder; those binaries will be used
> by the kinder `kubeadm-upgrade` action.

## Creating and initializing the cluster

See [getting started (test single control-plane)](getting-started.md) or [testing HA](test-HA.md);
in summary:

```bash
# create a cluster (choose the desired number of control-plane/worker nodes)
kinder create cluster --image kindest/node:vX.to.Y --control-plane-nodes 1 --worker-nodes 0

# initialize the bootstrap control plane
kinder do kubeadm-init

# join secondary control planes and nodes (if any)
kinder do kubeadm-join
```

Also in this case:

- test variants can be achieved adding `--kube-dns`, `--external-etcd`, `--copy-certs` flags to `kinder create cluster` or `--use-phases` to `kubeadm-init` and/or `kubeadm-join`

## Testing upgrades

```bash
# upgrade the cluster form vX to vY
# - upgrade kubeadm
# - run kubeadm upgrade (or upgrade node)
# - upgrade kubelet
kinder do kubeadm-upgrade --upgrade-version vY
```

As usual:

- you can use the `--only-node` flag to execute actions only on a selected node.
- as alternative to `kinder do`, it is possible to use `docker exec` and `docker cp`

## Validation

```bash
# verify kubeadm commands outputs

# get an overview of the resulting cluster
kinder do cluster-info
# > check for nodes, Kubernetes version x, ready
# > check all the components running, Kubernetes version x + related dependencies
# > check for etcd member

# run a smoke test
kinder do smoke-test
```

See [run e2e tests](e2e-test.md) for validating your cluster with Kubernetes/Kubeadm e2e test suites.

## Cleanup

```bash
kinder do kubeadm-reset

kinder delete cluster
```
