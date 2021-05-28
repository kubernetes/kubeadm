# Test HA

This document assumes vX the Kubernetes release to be tested.

See [Prepare for tests](prepare-for-tests.md) for how to create a node-image for Kubernetes vX.

```bash
# create a cluster with at least two control plane nodes (and eventually some worker node)
kinder create cluster --image kindest/node:vX --control-plane-nodes 2 --workers-nodes 0

# initialize the bootstrap control plane
kinder do kubeadm-init

# join secondary control planes and nodes (if any)
kinder do kubeadm-join
```

Please note that if you need a better control of pre-defined actions with `kinder do`, you can use
the `--only-node` flag to execute actions only on a selected node.

As alternative, instead of using kinder pre-defined actions with `kinder do`, it is possible to
use `docker exec` and `docker cp` to work on nodes invoking directly `kubeadm`, `kubectl` or
any other shell commands.

## Test variants

1. add `--external-etcd` flag to `kinder create cluster` to test usage of external etcd cluster
2. add `--use-phases` flag to `kubeadm-init` and/or `kubeadm-join` to test phases
3. add `--copy-certs=auto` flag both to `kubeadm-init` and `kubeadm-join` to test the automatic copy certs feature
4. any combination of the above

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

Also in this case:

- you can use the `--only-node` flag to execute actions only on a selected node.
- as alternative to `kinder do`, it is possible to use `docker exec` and `docker cp`

See [run e2e tests](e2e-test.md) for validating your cluster with Kubernetes/Kubeadm e2e test suites.

## Cleanup

```bash
kinder do kubeadm-reset

kinder delete cluster
```
