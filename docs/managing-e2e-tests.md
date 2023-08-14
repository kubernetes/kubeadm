## Managing kubeadm end-to-end tests

### Overview

Kubernetes has a rich end-to-end (e2e) testing infrastructure, which allows detailed testing of clusters and their assets. Most settings and tools for that can be found in the [test-infra](https://git.k8s.io/test-infra) GitHub repository.

Kubernetes uses applications such as the web-based [testgrid](https://testgrid.k8s.io/) for monitoring the status of e2e tests. test-infra also hosts the configuration on individual test jobs and Docker images that contain tools to invoke the jobs.

### Prow job configuration

The following folder contains all the SIG Cluster Lifecycle (the SIG that maintains kubeadm) originated test jobs:
[sig-cluster-lifecycle](https://git.k8s.io/test-infra/config/jobs/kubernetes/sig-cluster-lifecycle)

Please note that this document will only cover details on the `kubeadm*.yaml` files and only on some parameters
these files contain.

For example, let's have a look at this file:
[kubeadm-kinder.yaml](https://git.k8s.io/test-infra/config/jobs/kubernetes/sig-cluster-lifecycle/kubeadm-kinder.yaml)

It contains a list of jobs such as:
```
- name: ci-kubernetes-e2e-kubeadm-foo
  interval: 2h
  ...
```

In this case, `ci-kubernetes-e2e-kubeadm-foo` is a test job that runs every 2 hours and it also
has a set of other parameters defined for it. Such test jobs use an image that will run as a container inside
a Pod of the Kubernetes [Prow](https://git.k8s.io/test-infra/prow) cluster.

Prow jobs execute a tool called `kinder`.

### Kinder workflows

kubeadm uses a deployment tool called [kinder](https://git.k8s.io/kubeadm/kinder).
kinder is based on kind and it's used for a variety of different tests, such as upgrades and skew tests.

Kinder uses test workflow files that run sequences of tasks, such as "upgrade", "run e2e conformance tests", "run e2e kubeadm tests".

More information on kinder workflows can be found [here](kinder/ci/kubeadm-periodic.tests.md).
The same document must be updated every time kinder workflows are added/deleted.

### Testgrid configuration

Testgrid contains elements like dashboard groups, dashboards and tabs.

As an overview:
- SIG Cluster Lifecycle dashboards reside in a dashboard group that can be found [here](https://testgrid.k8s.io).
- Inside this dashboard group there is a [dashboard for kubeadm](https://testgrid.k8s.io/sig-cluster-lifecycle-kubeadm).
- Inside this dashboard there are individual tabs such as `kubeadm-foo` (which is the tab name for the
job `ci-kubernetes-e2e-kubeadm-foo`).

In the YAML structure of jobs such as `ci-kubernetes-e2e-kubeadm-foo` the following list
of annotations can be seen:

```
annotations:
  testgrid-dashboards: sig-cluster-lifecycle-kubeadm,sig-release-master-informing
  testgrid-tab-name: kubeadm-foo
  testgrid-alert-email: sig-cluster-lifecycle-kubeadm-alerts@kubernetes.io
  description: "OWNER: sig-cluster-lifecycle: some description here"
  testgrid-num-columns-recent: "20"
  testgrid-num-failures-to-alert: "4"
  testgrid-alert-stale-results-hours: "8"
```

These annotations configure the testgrid entries for this job from the job configuration itself.
They contain information such as dashboards where this job will appear, tab name, where to send email alerts,
description and other.

For more information about configuring testgrid see [this page](https://git.k8s.io/test-infra/testgrid/config.md).

### Updating kubeadm e2e test jobs

The script `kinder/hack/update-workflows.sh` can be used to update
the workflows in the current clone of the kubeadm repository and also
update the test-infra jobs in a local `kubernetes/test-infra` clone.

It requires the following environment variable to be set:
- `TEST_INFRA_KUBEADM_PATH`: passes `--path-test-infra ` to the
[`kinder/ci/tools/update-workflows`](kinder/ci/tools/update-workflows/README.md) tool.

The `update-workflows` tool uses templates to generate YAML files.
The templates are located in `kinder/ci/tools/update-workflows/templates`.

Let's define the size of the Kubernetes support skew as `N`.
For `N=3`, 3 releases will be in support.

When a new release is about to happen, the kubeadm test workflows and jobs have
to be updated two times:
1. When the new `release-x.yy` branch of `kubernetes/kubernetes` is created and the
  `release-x.yy-*` test-grid dashboards are created.
  - Edit `kinder/hack/update-workflows.sh`:
    - The MINOR value of `KUBERNETES_VERSION` must be incremented
    - `SKEW_SIZE` must be set to `N+1` to also include the new version
  - Run `./hack/update-workflows.sh` from the kinder directory

2. When the oldest release (i.e. `x.yy-N`) goes out support:
  - Edit `kinder/hack/update-workflows.sh`:
    - `SKEW_SIZE` must be set back to `N`, to exclude the oldest release
  - Run `./hack/update-workflows.sh`  from the kinder directory

### Job run frequency

Different kubeadm test jobs run on a different frequency of time.
Jobs against the latest development branch run more often.

### Email alerts

Test job failures can trigger email alerts. This can be configured using annotations:

```
  testgrid-alert-email: sig-cluster-lifecycle-kubeadm-alerts@kubernetes.io
  testgrid-num-failures-to-alert: "4"
  testgrid-alert-stale-results-hours: "8"
```

- `testgrid-alert-email` should be set to `sig-cluster-lifecycle-kubeadm-alerts@kubernetes.io`.
in the case of SIG Cluster Lifecycle. This is a mailing list (Google Group) that will receive the alert.
- `testgrid-alert-stale-results-hours` means an alert will be sent in case the job is in a stale state
and is not reporting new status after N hours. Usually a restart by a test-infra "on-call" operator
is required in such cases.
- `testgrid-num-failures-to-alert` sets the N failed runs ("red runs") after which an alert will be sent.

Jobs that runs against the development kubernetes/kubernetes branch should send email alerts more often
(e.g. 8 hours), while jobs for the older branches should report less often (e.g. 24 hours).

### Release informing and blocking jobs

Certain test jobs maintained by SIG Cluster Lifecycle can be present in release blocking or informing dashboards, such as:
https://testgrid.k8s.io/sig-release-master-informing

These test jobs are of higher importance as they can block a Kubernetes release in case they are failing.
Such dashboards are managed by SIG Release, but SIG Cluster Lifecycle can propose changes by adding or removing
test jobs, preferably near the beginning of a release cycle.

SIG Cluster Lifecycle must be responsive in case the release team tries to contact them about failing jobs.

Please note that if a job is consistently failing or flaky it will be removed from release dashboards,
by SIG Release or SIG Testing.
