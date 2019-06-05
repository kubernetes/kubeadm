# Kubeadm CI / E2E periodic tests

## Overview

Kubeadm has a set of CI tests, that you can access at:

<https://testgrid.k8s.io/sig-cluster-lifecycle-all>

TODO: create a sig-cluster-lifecycle-kubeadm dashboard

## Version in scope

Kubeadm test spans across 5 Kubernetes versions:

| Version          | e.g.   |                                                              |
| ---------------- | ------ | ------------------------------------------------------------ |
| master           | v1.15  | The release under current development                        |
| current          | v1.14  | Current GA release                                           |
| current -1/minor | V1.13  | Former GA release, still officially supported                |
| current -2/minor | V1.12  | First release out of support, but test are preserved for one additional release cycle (one cycle after going out of support) |
| current -3/minor | V1.11  | Second release out of support, but _upgrade tests only_ are preserved for one additional release additional cycle (two cycles after going out of support) |

## Type of tests

Kubeadm tests can be grouped in different families of tests, each one covering a different type of test workflow. Each test workflow
might be eventually repeated across all/a subset of the Kubernetes versions in scope.

### Regular tests

Kubeadm regular test are meant to create a cluster with `kubeadm init`, `kubeadm join` and then verify cluster
conformance. Following regular tests are verified using kind (no need of kinder customization so far):

| Version          | e.g.   |                                                              |
| ---------------- | ------ | ------------------------------------------------------------ |
| master<br />(master branch) | v1.15.0-alpha...  | The release under current development                        |
| current<br />(release-1.14 branch) | v1.14.2-alpha...  | Current GA release                                           |
| current -1/minor<br />(release-1.13 branch)  | V1.13.6-alpha...  | Former GA release, still officially supported                |
| current -2/minor<br />(release-1.12 branch)  | V1.12.10-alpha...  | First release out of support, but test are preserved for one additional release cycle (one cycle after going out of support) |

NB. currently kind tests are build Kubernetes from the selected branch. This is slightly different from what
kinder is doing, that is to use an existing CI/Release build.

### Upgrade tests

Upgrade tests are meant to verify the proper functioning of the `kubeadm upgrade` workflow. Following upgrade tests are verified:

| from                                         | e.g.              | to                                     | e.g.             |
| -------------------------------------------- | ----------------- | -------------------------------------- | ---------------- |
| current<br />(release/stable)                | v1.14.1           | master<br />(ci/latest)                | v1.15.0-alpha... |
| current -1/minor<br />(ci/latest-1.13) | V1.13.6-alpha...  | current<br />(release/latest-1.14)     | v1.14.1          |
| current -2/minor<br />(ci/latest-1.12)       | V1.12.9-alpha...  | current -1/minor<br />(ci/latest-1.13) | V1.13.5          |
| current -3/minor<br />(ci/latest-1.11)       | V1.11.10-alpha... | current -2/minor<br />(ci/latest-1.12) | V1.12.8          |

TODO: currently tests are not consistent with regards to the selection of from/to versions (some stable-->ci,
others ci-->latest). Define if/how to rationalize

### X on Y tests

X on Y tests are meant to verify the proper functioning of kubeadm version X with Kubernetes Y = X-1/minor. Following X on Y tests are implemented:

| kubeadm (X)                            | e.g.             | Kubernetes (Y)                         | e.g.              |
| -------------------------------------- | ---------------- | -------------------------------------- | ----------------- |
| master<br />(ci/latest)                | v1.15.0-alpha... | current<br />(release/stable)          | v1.14.1           |
| current<br />(ci/latest-1.14)          | v1.14.1-alpha... | current -1/minor<br />(ci/latest-1.13) | V1.13.6-alpha...  |
| current -1/minor<br />(ci/latest-1.13) | V1.13.6-alpha... | current -2/minor<br />(ci/latest-1.12) | V1.12.9-alpha...  |
| current -2/minor<br />(ci/latest-1.12) | V1.12.9-alpha... | current -3/minor<br />(ci/latest-1.11) | V1.11.10-alpha... |

TODO: currently tests are not consistent with regards to the selection of from/to versions (some ci on stable, others ci on ci). Define if/how to rationalize

### External etcd with secret copy tests

Kubeadm external etcd tests are meant to create a cluster with `kubeadm init`, `kubeadm join` using an external etcd cluster, using
kubeadm secret copy feature among control planes and then verify the cluster conformance.

| Version          | e.g.   |                                                              |
| ---------------- | ------ | ------------------------------------------------------------ |
| master<br />(ci/latest) | v1.15.0-alpha...  | The release under current development |
| current<br />(ci/latest-1.14) | v1.14.2-alpha...  | Current GA release |
