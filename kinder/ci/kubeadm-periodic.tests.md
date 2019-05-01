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

### Upgrade tests

Upgrade tests are meant to verify the proper functioning of the `kubeadm upgrade` workflow. More
specifically the following upgrade tests are verified:

| from                                         | e.g.              | to                                     | e.g.             |
| -------------------------------------------- | ----------------- | -------------------------------------- | ---------------- |
| current<br />(release/stable)                | v1.14.1           | master<br />(ci/latest)                | v1.15.0-alpha... |
| current -1/minor<br />(ci/latest-1.13) | V1.13.6-alpha...  | current<br />(release/latest-1.14)     | v1.14.1          |
| current -2/minor<br />(ci/latest-1.12)       | V1.12.9-alpha...  | current -1/minor<br />(ci/latest-1.13) | V1.13.5          |
| current -3/minor<br />(ci/latest-1.11)       | V1.11.10-alpha... | current -2/minor<br />(ci/latest-1.12) | V1.12.8          |

TODO: currently tests are not consistent with regards to the selection of from/to versions (some stable-->ci,
others ci-->latest). Define if/how to rationalize

### X on Y tests

X on Y tests are meant to verify the proper functioning of kubeadm version X with Kubernetes Y = X-1/minor.
More specifically following X on Y tests are verified:

| kubeadm (X) | e.g. | Kubernetes (Y) | e.g. |
| ----------- | ---- | -------------- | ---- |
|             |      |                |      |
|             |      |                |      |
|             |      |                |      |
|             |      |                |      |
|             |      |                |      |
