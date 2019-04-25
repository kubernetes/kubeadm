# Testing X on Y

X on Y test are meant to verify kubeadm vX managing Kubernetes cluster of different version vY,
with vY being 1 minor less or same minor of vX

## Preparation

In order to test X on Y, the recommended approach in kinder is to:

1. take a node-image for Kubernetes vY, e.g. `kindest/node:vY`
2. prepare locally the kubeadm binaries vX
3. create an variant of `kindest/node:vY`, e.g. `kindest/node:vX.on.Y`, that replaces kubeadm vY
with kubeadm vX

e.g. assuming vX artifacts stored in $artifacts

```bash
kinder build node-image-variant --base-image kindest/node:vY --image kindest/node:vX.on.Y \
    --with-kubeadm $artifacts/vY/kubeadm
```

> `kinder build node-image-variant` accepts in input a version, a release or ci build label,
> a remote repository or a local folder. see [Kinder reference](reference.md) for more info.

See [Prepare for tests](prepare-for-tests.md) for more detail

## Creating and initializing the cluster

See [getting started (test single control-plane)](getting-started.md) or [testing HA](test-HA.md);
in summary:

```bash
# create a cluster (choose the desired number of control-plane/worker nodes)
kinder create cluster --image kindest/node:vX.on.Y --control-plane-nodes 1 --workers-nodes 0

# initialize the bootstrap control plane
kinder do kubeadm-init

# join secondary control planes and nodes (if any)
kinder do kubeadm-join
```

Also in this case:

- test variants can be achieved adding `--kube-dns`, `--external-etcd`, `--automatic-copy-certs` flags to `kinder create cluster` or `--use-phases` to `kubeadm-init` and/or `kubeadm-join`

## Validation

```bash
# verify kubeadm commands outputs

# get an overview of the resulting cluster
kinder do cluster-info
# > check for nodes, Kubernetes version y, ready
# > check all the components running, Kubernetes version y + related dependencies
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
