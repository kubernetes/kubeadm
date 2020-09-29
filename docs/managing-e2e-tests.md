## Managing kubeadm end-to-end tests

### Overview

Kubernetes has a rich end-to-end (e2e) testing infrastructure, which allows detailed testing of clusters and their assets. Most settings and tools for that can be found in the [test-infra](https://github.com/kubernetes/test-infra) GitHub repository.

Kubernetes uses applications such as the web-based [testgrid](https://k8s-testgrid.appspot.com/) for monitoring the status of e2e tests. test-infra also hosts the configuration on individual test jobs and Docker images that contain tools to invoke the jobs.

### Prow job configuration

The following folder contains all the SIG Cluster Lifecycle (the SIG that maintains kubeadm) originated test jobs:
[sig-cluster-lifecycle](https://github.com/kubernetes/test-infra/tree/master/config/jobs/kubernetes/sig-cluster-lifecycle)

Please note that this document will only cover details on the `kubeadm*.yaml` files and only on some of the parameters
these files contain.

For example, let's have a look at this file:
[kubeadm-kinder.yaml](https://github.com/kubernetes/test-infra/blob/master/config/jobs/kubernetes/sig-cluster-lifecycle/kubeadm-kinder.yaml)

It contains a list of jobs such as:
```
- name: ci-kubernetes-e2e-kubeadm-kind-master
  interval: 2h
  ...
```

In this case, `ci-kubernetes-e2e-kubeadm-kind-master` is a test job that runs every 2 hours and it also
has a set of other parameters defined for it. Such test jobs use an image that will run as a container inside
a Pod of the Kubernetes [Prow](https://github.com/kubernetes/test-infra/tree/master/prow) cluster.

For this example job the deployment tool is called [kind](https://github.com/kubernetes-sigs/kind).
As a very high level summary, the way this works is when a job is invoked all the job parameters
are passed to a CLI tool called kubetest, which then instantiates the job container and then the deployment tool
inside it. Please note that kubetest is not required and test authors can decide to call any bash script instead.

The SIG also uses another deployment tool called [kinder](https://github.com/kubernetes/kubeadm/tree/master/kinder).
kinder is based on kind and it's used for upgrades and version skew tests, but it does not require kubetest integration.

Kinder uses test workflow files that run sequences of tasks, such as "upgrade", "run e2e conformance tests", "run e2e kubeadm tests".
An example of such a workflow file can be seen [here](https://github.com/kubernetes/kubeadm/blob/master/kinder/ci/workflows/presubmit-upgrade-latest.yaml).

### Testgrid configuration

Testgrid contains elements like dashboard groups, dashboards and tabs.

As an overview:
- SIG Cluster Lifecycle dashboards reside in a dashboard group that can be found [here](https://k8s-testgrid.appspot.com).
- Inside this dashboard group there is a [dashboard for kubeadm](https://k8s-testgrid.appspot.com/sig-cluster-lifecycle-kubeadm).
- Inside this dashboard there are individual tabs such as `kubeadm-kind-master` (which is the tab name for the
job `ci-kubernetes-e2e-kubeadm-kind-master`).

In the YAML structure of jobs such as `ci-kubernetes-e2e-kubeadm-kind-master` the following list
of annotations can be seen:

```
annotations:
  testgrid-dashboards: sig-cluster-lifecycle-kubeadm,sig-release-master-informing
  testgrid-tab-name: kubeadm-kind-master
  testgrid-alert-email: kubernetes-sig-cluster-lifecycle@googlegroups.com
  description: "OWNER: sig-cluster-lifecycle (kind); Uses kubeadm/kind to create a cluster and run the conformance suite"
  testgrid-num-columns-recent: "20"
  testgrid-num-failures-to-alert: "4"
  testgrid-alert-stale-results-hours: "8"
```

These annotations configure the testgrid entries for this job from the job configuration itself.
They contain information such as dashboards where this job will appear, tab name, where to send email alerts,
description and other.

For more information about configuring testgrid see [this page](https://github.com/kubernetes/test-infra/blob/master/testgrid/config.md).

### Updates to kubeadm tests

Before each new Kubernetes release and during the second month of the [release-cycle](https://github.com/kubernetes/kubeadm/blob/master/docs/release-cycle.md), a set of manual actions have to be performed, so that the kubeadm e2e tests are up to date with the new release.

The operation can be broken down into:
- Sending a PR for updating kinder workflows.
- Sending a PR for updating test jobs in kubernetes/test-infra.

Ideally both PRs should be merged around the same time, so that there are no failed runs.

The idea is to remove old tests and add new ones. For example. If Kubernetes 1.15 is about to be released soon,
this means that the `15 - 3 = 12` MINOR version would go outside of the support skew of the project. `1.12` jobs
have to be removed and `1.15` jobs have to be added. The project would support version `15 - 2 = 13` as the minimum.

The summary of the actions is the following:
- If there are `1.12 -> 1.13` upgrade jobs leave them, because upgrades to `1.13` still have to be supported.
- Remove plain `1.12` jobs (only after 1.15 is released).
- Remove `1.13 on 1.12` version skew jobs (only after 1.15 is released).
- Remove `1.11 -> 1.12` upgrade jobs (only after 1.15 is released).
- Add upgrade jobs for `1.14 -> 1.15` and `1.15 on 1.14` skew jobs.
- Add plain `1.15` jobs.

#### Updating kinder workflows

Kinder workflows need to be updated each cycle in this location:
https://github.com/kubernetes/kubeadm/blob/master/kinder/ci/workflows/

Additional actions to be performed:
- Make sure that workflows include the correct `ci/latest-x` labels.
- Make sure that workflows have the correct testgrid URLs.

#### Updating test jobs in kubernetes/test-infra

This document will cover information on how to perform updates on these files:
- [kubeadm-kinder.yaml](https://github.com/kubernetes/test-infra/blob/master/config/jobs/kubernetes/sig-cluster-lifecycle/kubeadm-kinder.yaml)

Holds test jobs where the kubeadm version matches the Kubernetes control-plane and the kubelet versions.

- [kubeadm-kinder-upgrade.yaml](https://github.com/kubernetes/test-infra/blob/master/config/jobs/kubernetes/sig-cluster-lifecycle/kubeadm-kinder-upgrade.yaml)

Holds test jobs that perform a upgrade from version X to version Y (usually `Y = X + 1`).

- [kubeadm-kinder-x-on-y.yaml](https://github.com/kubernetes/test-infra/blob/master/config/jobs/kubernetes/sig-cluster-lifecycle/kubeadm-kinder-x-on-y.yaml)

Holds test jobs that run kubeadm version X, on a control plane and kubelet version Y,
as kubeadm does support `Y = X - 1`

- [kubeadm-kinder-external-etcd.yaml](https://github.com/kubernetes/test-infra/blob/master/config/jobs/kubernetes/sig-cluster-lifecycle/kubeadm-kinder-external-etcd.yaml)

Holds tests that deploy a kubeadm / kinder cluster using external etcd, instead of the default stacked etcd.

The files can be found in the [sig-cluster-lifecycle](https://github.com/kubernetes/test-infra/tree/master/config/jobs/kubernetes/sig-cluster-lifecycle) folder.


Additional actions to be performed:
- Make sure the correct image tag for kubekins is used:
  - `kubekins-e2e:<TAG>-1.15` should be the kubekins image:tag for a `1.15` job.
  - `kubekins-e2e:<TAG>-master` should be the kubekins image:tag for `master` job.
- Include the correct annotations:

```
annotations:
  testgrid-dashboards: sig-cluster-lifecycle-kubeadm,sig-release-1.15-informing
  testgrid-tab-name: kubeadm-kind-1-15
```

The above indicates that a tab called `kubeadm-kind-1-15` will appear in the dashboards
`sig-cluster-lifecycle-kubeadm` and `sig-release-1.15-informing`.

- Jobs should clone the correct kubernetes/kubernetes branch:

```
extra_refs:
- org: kubernetes
  repo: kubernetes
  base_ref: <branch-name>
```
For example, a `1.15` job would need to clone the branch `release-1.15` and `master` job would
need to clone `master`.

- kinder jobs would also need to execute the correct workflow - e.g. `upgrade-1.14-1.15` or `upgrade-1.15-master`.

Once you have made the above test-infra changes, you need to run a certain script to reflect the changes in generated configuration. For that you can call the following command:

```
./hack/update-all.sh
```

You can also run verification for your changes by calling the verification scripts:
```
./hack/verify-*.sh
```

Or building test-infra:
```
make
```
Note that test-infra requires [Bazel](https://bazel.build/).

At this point you can commit your changes and send a GitHub PR against the test-infra repository.

### Job run frequency

Different kubeadm test jobs run on a different frequency of time.

Jobs against the `master` kubernetes/kubernetes branch
run more often, e.g `ci-kubernetes-e2e-kubeadm-kind-master`, `ci-kubernetes-e2e-kubeadm-kinder-master-on-stable` and `ci-kubernetes-e2e-kubeadm-kinder-upgrade-stable-master` run every 2 hours. Given the `master` branch receives a lot
of updates it's important to catch problems more often.

Test jobs for the older branches, e.g. `ci-kubernetes-e2e-kubeadm-kind-(master-1)`,
`ci-kubernetes-e2e-kubeadm-kinder-(master-1)-on-(master-2)` and `ci-kubernetes-e2e-kubeadm-kinder-upgrade-(master-2)-(master-1)`
run every 12 hours. The older maintained kubernetes/kubernetes branches receive less updates, so less frequent job runs there
are sufficient.

### Email alerts

Test job failures can trigger email alerts. This can be configured using annotations:

```
  testgrid-alert-email: kubernetes-sig-cluster-lifecycle@googlegroups.com
  testgrid-num-failures-to-alert: "4"
  testgrid-alert-stale-results-hours: "8"
```

- `testgrid-alert-email` should be set to `kubernetes-sig-cluster-lifecycle@googlegroups.com`.
in the case of SIG Cluster Lifecycle. This is a mailing list (Google Group) that will receive the alert.
- `testgrid-alert-stale-results-hours` means an alert will be sent in case the job is in a stale state
and is not reporting new status after N hours. Usually a restart by a test-infra "on-call" operator
is required in such cases.
- `testgrid-num-failures-to-alert` sets the N failed runs ("red runs") after which an alert will be sent.

Jobs that runs against the `master` kubernetes/kubernetes branch should send email alerts more often (e.g. 8 hours),
while jobs for the older branches should report less often (e.g. 24 hours).

### Release informing and blocking jobs

Certain test jobs maintained by SIG Cluster Lifecycle can be present in release blocking or informing dashboards, such as:
https://k8s-testgrid.appspot.com/sig-release-master-informing

These test jobs are of higher importance as they can block a Kubernetes release in case they are failing.
Such dashboards are managed by SIG Release, but SIG Cluster Lifecycle can propose changes by adding or removing
test jobs, preferably near the beginning of a release cycle.

SIG Cluster Lifecycle must be responsive in case the release team tries to contact them about failing jobs.

Please note that if a job is consistently failing or flaky it will be removed from release dashboards,
by SIG Release or SIG Testing.

### Resources

- Example PRs can be found at the following links:
  - https://github.com/kubernetes/kubeadm/pull/1744
  - https://github.com/kubernetes/test-infra/pull/14037
