## Managing kubeadm end-to-end tests

### Overview

Kubernetes has a rich end-to-end (e2e) testing infrastructure, which allows detailed testing of clusters and their assets. Most settings and tools for that can be found in the [test-infra](https://github.com/kubernetes/test-infra) GitHub repository.

Kubernetes uses applications such as the web-based [testgrid](https://k8s-testgrid.appspot.com/) for monitoring the status of e2e tests. test-infra also hosts the configuration on individual test jobs and Docker images that contain tools to invoke the jobs.

### Details on kubeadm test jobs

The following folder contains all the SIG Cluster Lifecycle (the SIG that maintains kubeadm) originated test jobs:
[sig-cluster-lifecycle](https://github.com/kubernetes/test-infra/tree/master/config/jobs/kubernetes/sig-cluster-lifecycle)

Please note that this document will only cover details on the `kubeadm*.yaml` files.

For example, let's have a look at this file:
[kubeadm.yaml](https://github.com/kubernetes/test-infra/blob/master/config/jobs/kubernetes/sig-cluster-lifecycle/kubeadm-kind.yaml)

It contains a list of jobs such as:
```
- name: ci-kubernetes-e2e-kubeadm-kind-master
  interval: 2h
  ...
```

In this case, `ci-kubernetes-e2e-kubeadm-kind-master` is a test job that runs every 2 hours and it also has a set of parameters defined for it. This document will only cover some of them.

Important to note here is that testgrid jobs have a docker `image` that contains all the tools needed to run tests. Among them, one of the most relevant ones, is the tool that will be used for the deployment of the cluster.

For this example job the `deployment` tool is called [kind](https://github.com/kubernetes-sigs/kind) (with integration in [this file](https://github.com/kubernetes/test-infra/blob/master/kubetest/kind/kind.go)) and the `image` is called `*kubekins:...`. As a very high level summary, the way this works is when a job is invoked all the job parameters are passed to a CLI tool called `kubetest`, which then instantiates the job container and then the deployment tool inside it.

The SIG also uses another deployment tool called [kinder](https://github.com/kubernetes/kubeadm/kinder). kinder is based on kind and it's used for upgrades and version skew tests, but it does not require kubetest integration.

Kinder uses test workflow files that run sequences of tasks, such as "upgrade", "run e2e conformance tests", "run e2e kubeadm tests".
An example of such a workflow file can be seen [here](https://github.com/kubernetes/kubeadm/blob/master/kinder/ci/workflows/upgrade-stable-master.yaml).

### The testgrid configuration

testgrid uses a rather large configuration file that hosts the dashboard of all the tests that can be found [here](https://k8s-testgrid.appspot.com). The initial list of menu items mostly holds Kubernetes SIGs, while the SIG Cluster Lifecycle specific dashboard can be found [here](https://k8s-testgrid.appspot.com/sig-cluster-lifecycle-all).

The configuration file itself is located [here](https://github.com/kubernetes/test-infra/blob/master/testgrid/config.yaml). The file is broken into multiple sections. If you wish to find the SIG Cluster Lifecycle and kubeadm tests you can search for `sig-cluster-lifecycle`. For instance, `sig-cluster-lifecycle-all` has all the relevant kubeadm tests.

A dashboard is defined like so:
```
- name: sig-cluster-lifecycle-kubeadm
  dashboard_tab:
# kubeadm-kind tests
  - name: kubeadm-kind-...
    test_group_name: ci-kubernetes-e2e-kubeadm-kind-...
    ...
  - name: kubeadm-kind-...
    test_group_name: ci-kubernetes-e2e-kubeadm-kind-...
```

What the above does is defining a dashboard called `sig-cluster-lifecycle-all` and placing inside a couple of tests defined by `name` and linking to a `test_group_name`. Some items may also have a `description`.

Note that `test_group_name` represents the actual test name (as in the [kubeadm-kind.yaml](https://github.com/kubernetes/test-infra/blob/master/config/jobs/kubernetes/sig-cluster-lifecycle/kubeadm-kind.yaml) example earlier), while `name` is only what is shown on the website as an alias.

Tests can exist in multiple dashboards at once. For example a test defined like:
```
- name: kubeadm-kind-1-12
  test_group_name: ci-kubernetes-e2e-kubeadm-kind-1-12
```
can exists in both the `sig-cluster-lifecycle-all` and `sig-release-master-blocking` dashboards. Essentially both tests would show details about the same test job called `ci-kubernetes-e2e-kubeadm-kind-1-12`.

### Updates to kubeadm tests

Before each new Kubernetes release and during the second month of the [release-cycle](https://github.com/kubernetes/kubeadm/blob/master/docs/release-cycle.md), a set of manual actions have to be performed, so that the kubeadm e2e tests are up to date with the new release.

The operation can be broken down in 3 steps:
1) Updating the kubeadm test jobs and parameters.
2) Updating testgrid to reflect the changes.
3) Finalizing the changes.

There is work in progress in test-infra on tools that can hopefully automate this process.

#### Updating test jobs

This document will cover information on how to perform updates on these 3 files:
- [kubeadm-kind.yaml](https://github.com/kubernetes/test-infra/blob/master/config/jobs/kubernetes/sig-cluster-lifecycle/kubeadm-kind.yaml "kubeadm-kind.yaml")

Holds test jobs where the kubeadm version matches the Kubernetes control-plane and the kubelet versions.

- [kubeadm-kinder-upgrade.yaml](https://github.com/kubernetes/test-infra/blob/master/config/jobs/kubernetes/sig-cluster-lifecycle/kubeadm-kinder-upgrade.yaml "kubeadm-kinder-upgrade.yaml")

Holds test jobs that perform a upgrade from version X to version Y (usually `Y = X + 1`).

- [kubeadm-kinder-x-on-y.yaml](https://github.com/kubernetes/test-infra/blob/master/config/jobs/kubernetes/sig-cluster-lifecycle/kubeadm-kinder-x-on-y.yaml "kubeadm-kinder-x-on-y.yaml")

Holds test jobs that run kubeadm version X, on a control plane and kubelet version Y,
as kubeadm does support `Y = X - 1`

The files can be found in the [sig-cluster-lifecycle](https://github.com/kubernetes/test-infra/tree/master/config/jobs/kubernetes/sig-cluster-lifecycle) folder.

The idea is to remove old test jobs and add new ones. If Kubernetes 1.15 is about to be released soon, this means that the `15 - 3 = 12` MINOR version would go outside of the support skew of the project. `1.12` jobs have to be removed and `1.15` jobs have to be added. The project would support version `15 - 2 = 13` as the minimum.

- If there are `1.12 -> 1.13` upgrade jobs leave them, because upgrades to `1.13` still have to be supported.
- Remove `1.13 on 1.12` jobs.
- Remove plain `1.12` jobs.
- Add upgrade jobs for `1.14 -> 1.15` and `1.14 on 1.15` jobs.
- Add plain `1.15` jobs.

#### Updating testgrid to reflect the changes

Once the changes to the jobs have been made it is time to open the testgrid [configuration file](https://github.com/kubernetes/test-infra/blob/master/testgrid/config.yaml) and modify it as well .

The same procedure applies here - old jobs are being removed and new jobs are being added.

Search and replace is your friend. Remember the job names that you just changed in the `kubeadm*.yaml` files. Don't forget that jobs can exist under multiple dashboards in testgrid, so make sure you find all occurrences and modify them.

Following the `1.15` example from earlier, a job called `ci-kubernetes-e2e-kubeadm-gce-1-12` would have to be removed from all occurrences and a new job `ci-kubernetes-e2e-kubeadm-gce-1-15` has to be added in it's place.

Removed:
```
- name: kubeadm-kind-1-12
  test_group_name: ci-kubernetes-e2e-kubeadm-kind-1-12
```

Added:
```
- name: kubeadm-kind-1-15
  test_group_name: ci-kubernetes-e2e-kubeadm-kind-1-15
```

#### Finalizing the changes

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

#### Job run frequency

Different kubeadm test jobs run on a different frequency of time.

Jobs against the `master` kubernetes/kubernetes branch
run more often, e.g `ci-kubernetes-e2e-kubeadm-kind-master`, `ci-kubernetes-e2e-kubeadm-kinder-master-on-stable` and `ci-kubernetes-e2e-kubeadm-kinder-upgrade-stable-master` run every 2 hours. Given the `master` branch receives a lot
of updates it's important to catch problems more often.

Test jobs for the older branches, e.g. `ci-kubernetes-e2e-kubeadm-kind-(master-1)`,
`ci-kubernetes-e2e-kubeadm-kinder-(master-1)-on-(master-2)` and `ci-kubernetes-e2e-kubeadm-kinder-upgrade-(master-2)-(master-1)`
run every 12 hours. The older maintained kubernetes/kubernetes branches receive less updates, so less frequent job runs there
are sufficient.

#### Email alerts

Test job failures can trigger email alerts. This can be configured when a job is added in a testgrid dashboard.
For example:

```
- name: kubeadm-kind-master
    test_group_name: ci-kubernetes-e2e-kubeadm-kind-master
    alert_options:
      alert_mail_to_addresses: kubernetes-sig-cluster-lifecycle@googlegroups.com
    alert_stale_results_hours: 8
    num_failures_to_alert: 4 # Runs every 2h. Alert when it's been failing for 8 hours
```

- `alert_mail_to_addresses` should be set to `kubernetes-sig-cluster-lifecycle@googlegroups.com`
in the case of SIG Cluster Lifecycle. This is a mailing list (Google Group) that will receive the alert.
- `alert_stale_results_hours` means an alert will be sent in case the job is in a stale state and is not reporting new status
after N hours. Usually a restart by a test-infra "on-call" operator is required in such cases.
- `num_failures_to_alert` sets the N failed runs ("red runs") after which an alert will be sent.

Jobs that runs against the `master` kubernetes/kubernetes branch should send email alerts more often (e.g. 8 hours), while
jobs for the older branches should report less often (e.g. 24 hours).

#### Release blocking jobs

Certain test jobs maintained by SIG Cluster Lifecycle can be present in release blocking or informing dashboards, such as:
https://k8s-testgrid.appspot.com/sig-release-master-informing

These test jobs are of higher importance as they can block a Kubernetes release in case they are failing.
Such dashboards are managed by SIG Release, but SIG Cluster Lifecycle can propose changes by adding or removing test jobs,
preferably near the beginning of a release cycle.

#### Resources

- Here is a pair-programming [video session](https://www.youtube.com/watch?v=aTGbdPU0fE8) between Lucas Käldström and Hippie Hacker covering the same process for the Kubernetes 1.11 release (outdated).
- Example PRs can be found at the following links (outdated):
  - https://github.com/kubernetes/test-infra/pull/9219
  - https://github.com/kubernetes/test-infra/pull/8142
