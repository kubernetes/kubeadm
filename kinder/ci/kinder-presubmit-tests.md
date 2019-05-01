# Kinder presubmit tests

## Overview

Kinder has a set of presubmit tests, designed in order to verify that:

- Any PR sent to `kubernetes/kubeadm` repository with impact to the `kinder` codebase satisfies
  all the expected code checks
- Any PR sent to `kubernetes/kubeadm` repository with impact to the `kinder` codebase does not
  compromise `kinder` features

Kinder presubmit tests configuration can be found at <https://github.com/kubernetes/test-infra/blob/master/config/jobs/kubernetes/kubeadm/kubeadm-presubmits.yaml>

## Code Checks

`pull-kubeadm-kinder-verify` is presubmits job that runs a set of code checks:

- verify-whitespace
- verify-spelling
- verify-gofmt
- verify-golint
- verify-govet
- verify-workflows
- verify-deps
- verify-gotest
- verify-build

All checks are implemented under the `hack` folder and the entry point is `hack/verify-all.sh`

## Kinder feature checks

`pull-kubeadm-kinder-init-versions` job checks that kinder can create cluster for the supported init versions:

- current -3/minor (e.g. v1.11)
- current -2/minor (e.g. v1.12)
- current -1/minor (e.g. v1.13)
- current          (e.g. v1.14)
- master           (e.g. v1.15)

`pull-kubeadm-kinder-variants` job checks that kinder can manage supported kubeadm config variants and
workflow variants:

- kube-dns
- external-etcd
- HA
- HA with automatic copy certs

- TODO: add-node

`pull-kubeadm-kinder-utils` job checks that kinder utility command works properly:

- TODO: `kinder get artifacts`
- TODO: `kinder build node-image-variants`
- TODO: `kinder cp`
- TODO: `kinedr exec`
- TODO: `kinder do cluster-info`
- TODO: `kinder do smoke-test`
- TODO: `kinder test e2e-kubeadm`
- TODO: `kinder test e2e`
