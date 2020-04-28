# Kubeadm CI / E2E periodic tests

## Overview

Kubeadm has a set of CI tests, that you can access at:

<https://testgrid.k8s.io/sig-cluster-lifecycle-all>

TODO: create a sig-cluster-lifecycle-kubeadm dashboard

## Version in scope

Kubeadm test spans across 5 Kubernetes versions:

| Version          | e.g.   |                                                              |
| ---------------- | ------ | ------------------------------------------------------------ |
| master           | v1.17  | The release under current development                        |
| current          | v1.16  | Current GA release                                           |
| current -1/minor | V1.15  | Former GA release, still officially supported                |
| current -2/minor | V1.14  | Former GA release, still officially supported for one more cycle |
| current -3/minor | V1.13  | Former GA release that is no longer supported but still tested for upgrade and skew |

## Type of tests

Kubeadm tests can be grouped in different families of tests, each one covering a different type of test workflow. Each test workflow
might be eventually repeated across all/a subset of the Kubernetes versions in scope.

### Regular tests

Kubeadm regular test are meant to create a cluster with `kubeadm init`, `kubeadm join` and then verify cluster
conformance.

| Version          | e.g.   |                                                              |
| ---------------- | ------ | ------------------------------------------------------------ |
| master<br />(master branch) | v1.17.0-alpha...  | The release under current development  |
| current<br />(release-1.16 branch) | v1.16.2-alpha...  | Current GA release              |
| current -1/minor<br />(release-1.15 branch)  | V1.15.6-alpha...   | Former GA release, still officially supported |
| current -2/minor<br />(release-1.14 branch)  | V1.14.10-alpha...  | Former GA release, still officially supported for one more cycle |

### Upgrade tests

Upgrade tests are meant to verify the proper functioning of the `kubeadm upgrade` workflow. Following upgrade tests are verified:

| from                                    | e.g.              | to                                     | e.g.             |
| --------------------------------------- | ----------------- | -------------------------------------- | ---------------- |
| current<br />(ci/latest-1.16)           | v1.16.1-alpha...  | master<br />(ci/latest)                | v1.17.0-alpha... |
| current -1/minor<br />(ci/latest-1.15)  | V1.15.6-alpha...  | current<br />(ci/latest-1.16)          | v1.16.1-alpha... |
| current -2/minor<br />(ci/latest-1.14)  | V1.14.9-alpha...  | current -1/minor<br />(ci/latest-1.15) | V1.15.5-alpha... |
| current -3/minor<br />(ci/latest-1.13)  | V1.13.10-alpha... | current -2/minor<br />(ci/latest-1.14) | V1.14.8-alpha... |

NB. currently we are testing `ci/latest` and not (e.g.) `ci/latest-1.16`. That is because the 1.16 branch
is aligned only periodically until the release is cut. At a certain point master will become 1.17 even
if 1.16 is not actually out. To make this work, `kubeadm upgrade` is required to use the `-f` flag to ignore
the error for skipping one version, but this is an acceptable trade off.

### X on Y tests

X on Y tests are meant to verify the proper functioning of kubeadm version X with Kubernetes Y = X-1/minor. Following X on Y tests are implemented:

| kubeadm (X)                            | e.g.             | Kubernetes (Y)                         | e.g.              |
| -------------------------------------- | ---------------- | -------------------------------------- | ----------------- |
| master<br />(ci/latest)                | v1.17.0-alpha... | current<br />(ci/latest-1.16)          | v1.16.1-alpha...  |
| current<br />(ci/latest-1.16)          | v1.16.1-alpha... | current -1/minor<br />(ci/latest-1.15) | V1.15.4-alpha...  |
| current -1/minor<br />(ci/latest-1.15) | V1.15.6-alpha... | current -2/minor<br />(ci/latest-1.14) | V1.14.7-alpha...  |
| current -2/minor<br />(ci/latest-1.14) | V1.14.9-alpha... | current -3/minor<br />(ci/latest-1.13) | V1.13.8-alpha...  |

### External etcd with secret copy tests

Kubeadm external etcd tests are meant to create a cluster with `kubeadm init`, `kubeadm join` using an external etcd cluster,
using kubeadm secret copy feature among control planes and then verify the cluster conformance. Currently, 1.14 is
the minimal supported version that is tested for external etcd.

| Version                                | e.g.              |
| -------------------------------------- | ------            |
| master<br />(ci/latest)                | v1.17.0-alpha...  |
| current<br />(ci/latest-1.16)          | v1.16.2-alpha...  |
| current -1/minor<br />(ci/latest-1.15) | v1.15.4-alpha...  |

### Discovery tests

Kubeadm discovery tests are meant for testing alternative discovery methods for kubeadm join. Kubernetes 1.16 is
the minimal supported version that is tested for join discovery variants.

| Version                                | e.g.              |
| -------------------------------------- | ------            |
| master<br />(ci/latest)                | v1.17.0-alpha...  |
| master<br />(ci/latest-1.16)           | v1.16.2-alpha...  |

### Kustomize tests

Kubeadm kustomize tests are meant for testing usage of kustomize patches with kubeadm init, join and kubeadm upgrade.
Kubernetes 1.16 is the minimal supported version that is tested for kustomize.

| Version                                | e.g.              |
| -------------------------------------- | ------            |
| master<br />(ci/latest)                | v1.17.0-alpha...  |
| master<br />(ci/latest-1.16)           | v1.16.2-alpha...  |

### Tests without addon ConfigMaps

Kubeadm join and upgrade tests that ensure that kubeadm tolerates missing addon "kube-proxy" and "coredns" ConfigMaps.

| from                            | e.g.              | to                       | e.g.             |
| ------------------------------- | ----------------- | -------------------------| ---------------- |
| current<br />(ci/latest-1.18)   | v1.18.1-alpha...  | master<br />(ci/latest)  | v1.19.0-alpha... |
